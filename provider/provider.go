package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-aiven-broker/client/elasticsearch"
	"github.com/alphagov/paas-aiven-broker/client/influxdb"
	"github.com/alphagov/paas-aiven-broker/client/opensearch"
	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
)

const AIVEN_BASE_URL string = "https://api.aiven.io"

const RestoreFromLatestBackupBeforeTimeFormat = "2006-01-02 15:04:05"
const RestoreFromPointInTimeBeforeTimeFormat = "2006-01-02 15:04:05"

type AivenProvider struct {
	Client                       aiven.Client
	Config                       *Config
	AllowUserProvisionParameters bool
	AllowUserUpdateParameters    bool
	Logger                       lager.Logger
}

func New(configJSON []byte, logger lager.Logger) (*AivenProvider, error) {
	config, err := DecodeConfig(configJSON)
	if err != nil {
		return nil, err
	}
	client := aiven.NewHttpClient(AIVEN_BASE_URL, config.APIToken, config.Project, logger)
	return &AivenProvider{
		Client:                       client,
		Config:                       config,
		AllowUserProvisionParameters: true,
		AllowUserUpdateParameters:    true,
		Logger:                       logger,
	}, nil
}

func IPAddresses(iplist string) string {
	if len(iplist) > 0 {
		_, ok := os.LookupEnv("IP_WHITELIST")
		if !ok {
			return iplist
		} else {
			filterList := os.Getenv("IP_WHITELIST") + "," + iplist
			return filterList
		}
	}
	_, ok := os.LookupEnv("IP_WHITELIST")
	if !ok {
		return ""
	} else {
		filterList := os.Getenv("IP_WHITELIST")
		return filterList
	}
}

func (ap *AivenProvider) Provision(
	ctx context.Context,
	provisionData ProvisionData,
	asyncAllowed bool,
) (result domain.ProvisionedServiceSpec, err error) {
	if !asyncAllowed {
		return result, brokerapi.ErrAsyncRequired
	}

	plan, err := ap.Config.FindPlan(provisionData.Service.ID, provisionData.Plan.ID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, fmt.Errorf("Service Plan '%s' not found", provisionData.Details.PlanID)
	}

	tags := aiven.ServiceTags{
		DeployEnv:          ap.Config.DeployEnv,
		ServiceID:          provisionData.InstanceID,
		PlanID:             provisionData.Plan.ID,
		OrganizationID:     provisionData.Details.OrganizationGUID,
		SpaceID:            provisionData.Details.SpaceGUID,
		BrokerName:         ap.Config.BrokerName,
		RestoredFromBackup: "false",
	}

	provisionParameters := ProvisionParameters{}
	if ap.AllowUserProvisionParameters && len(provisionData.Details.RawParameters) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(provisionData.Details.RawParameters))
		if err := decoder.Decode(&provisionParameters); err != nil {
			return domain.ProvisionedServiceSpec{}, err
		}
		if err := provisionParameters.Validate(); err != nil {
			return domain.ProvisionedServiceSpec{}, err
		}
	}

	userConfig := aiven.UserConfig{}

	addressList := IPAddresses(provisionParameters.UserIpFilter)
	filterlist, err := ParseIPWhitelist(addressList)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}
	userConfig.IPFilter = filterlist

	if provisionData.Service.Name == "elasticsearch" {
		userConfig.ElasticsearchVersion = plan.ElasticsearchVersion
	} else if provisionData.Service.Name == "opensearch" {
		userConfig.OpenSearchVersion = plan.OpenSearchVersion
	} else if provisionData.Service.Name == "influxdb" {
		// Nothing to do
	} else {
		return domain.ProvisionedServiceSpec{}, fmt.Errorf(
			"Cannot provision service for unknown service %s",
			provisionData.Service.Name,
		)
	}

	if provisionParameters.RestoreFromLatestBackupOf == nil && provisionParameters.RestoreFromLatestBackupBefore != nil {
		return domain.ProvisionedServiceSpec{}, fmt.Errorf(
			"Parameter restore_from_latest_backup_before should be used with restore_from_latest_backup_of",
		)
	}

	if provisionParameters.RestoreFromLatestBackupOf != nil {
		err := ap.forkFromBackup(
			ctx, provisionData, asyncAllowed,
			provisionParameters, userConfig, tags,
		)
		if err != nil {
			return domain.ProvisionedServiceSpec{}, err
		}

	} else {
		createServiceInput := &aiven.CreateServiceInput{
			Cloud:       ap.Config.Cloud,
			Plan:        plan.AivenPlan,
			ServiceName: ap.BuildServiceName(provisionData.InstanceID),
			ServiceType: provisionData.Service.Name,
			UserConfig:  userConfig,
			Tags:        tags,
		}
		if err != nil {
			return domain.ProvisionedServiceSpec{}, err
		}

		if _, err := ap.Client.CreateService(createServiceInput); err != nil {
			return domain.ProvisionedServiceSpec{}, err
		}
	}
	return domain.ProvisionedServiceSpec{IsAsync: true}, nil
}

func (ap *AivenProvider) Deprovision(ctx context.Context, deprovisionData DeprovisionData) (operationData string, err error) {
	err = ap.Client.DeleteService(&aiven.DeleteServiceInput{
		ServiceName: ap.BuildServiceName(deprovisionData.InstanceID),
	})

	if err != nil {
		if err == aiven.ErrInstanceDoesNotExist {
			return "", apiresponses.ErrInstanceDoesNotExist
		}
	}

	return "deprovisioning", err
}

func (ap *AivenProvider) Bind(ctx context.Context, bindData BindData) (binding domain.Binding, err error) {
	serviceName := ap.BuildServiceName(bindData.InstanceID)
	user := bindData.BindingID

	password, err := ap.Client.CreateServiceUser(&aiven.CreateServiceUserInput{
		ServiceName: serviceName,
		Username:    user,
	})
	if err != nil {
		return domain.Binding{}, err
	}

	service, err := ap.Client.GetService(&aiven.GetServiceInput{
		ServiceName: serviceName,
	})
	if err != nil {
		return domain.Binding{}, err
	}

	host := service.ServiceUriParams.Host
	port := service.ServiceUriParams.Port
	serviceType := service.ServiceType

	if host == "" || port == "" {
		return domain.Binding{}, errors.New(
			"Error getting service connection details: no connection details found in response JSON",
		)
	}

	credentials, err := BuildCredentials(serviceType, user, password, host, port)
	if err != nil {
		return domain.Binding{}, err
	}

	if err = ensureUserAvailability(ctx, serviceType, credentials); err != nil {
		// Polling is only a best-effort attempt to work around Aiven API delays.
		// We therefore continue anyway if it times out.
		if err != context.DeadlineExceeded {
			return domain.Binding{}, err
		}
	}

	return domain.Binding{
		Credentials: credentials,
	}, nil
}

func ensureUserAvailability(
	ctx context.Context,
	serviceType string,
	credentials Credentials,
) error {
	if serviceType == "elasticsearch" {
		return tryAvailability(ctx, func() error {
			client := elasticsearch.New(credentials.URI, nil)
			_, err := client.Version()
			return err
		})
	} else if serviceType == "opensearch" {
		return tryAvailability(ctx, func() error {
			client := opensearch.New(credentials.URI, nil)
			_, err := client.Version()
			return err
		})
	} else if serviceType == "influxdb" {
		return tryAvailability(ctx, func() error {
			client := influxdb.New(credentials.URI, nil)
			_, err := client.Ping()
			return err
		})
	} else {
		return fmt.Errorf(
			"Cannot ensure availability for unknown service %s", serviceType,
		)
	}
}

func tryAvailability(
	ctx context.Context,
	availabilityCheck func() error,
) error {
	if availabilityCheck() == nil {
		return nil
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if availabilityCheck() == nil {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ap *AivenProvider) Unbind(ctx context.Context, unbindData UnbindData) (err error) {
	_, err = ap.Client.DeleteServiceUser(&aiven.DeleteServiceUserInput{
		ServiceName: ap.BuildServiceName(unbindData.InstanceID),
		Username:    unbindData.BindingID,
	})
	return err
}

func (ap *AivenProvider) Update(
	ctx context.Context,
	updateData UpdateData,
	asyncAllowed bool,
) (result domain.UpdateServiceSpec, err error) {
	plan, err := ap.Config.FindPlan(updateData.Details.ServiceID, updateData.Details.PlanID)
	result.OperationData = "fail"
	if err != nil {
		return result, err
	}

	UpdateParameters := &UpdateParameters{}
	if ap.AllowUserProvisionParameters && len(updateData.Details.RawParameters) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(updateData.Details.RawParameters))
		if err := decoder.Decode(&UpdateParameters); err != nil {
			return result, err
		}
	}

	userConfig := aiven.UserConfig{}

	addressList := IPAddresses(UpdateParameters.UserIpFilter)
	filterlist, err := ParseIPWhitelist(addressList)
	if err != nil {
		return result, err
	}
	userConfig.IPFilter = filterlist

	userConfig.ElasticsearchVersion = plan.ElasticsearchVersion // Pass empty version through if not InfluxDB
	userConfig.OpenSearchVersion = plan.OpenSearchVersion       // Pass empty version through if not InfluxDB

	_, err = ap.Client.UpdateService(&aiven.UpdateServiceInput{
		ServiceName: ap.BuildServiceName(updateData.InstanceID),
		Plan:        plan.AivenPlan,
		UserConfig:  userConfig,
	})

	if err != nil {
		switch err := err.(type) {
		case aiven.ErrInvalidUpdate:
			return result, apiresponses.NewFailureResponseBuilder(
				err,
				http.StatusUnprocessableEntity,
				"plan-change-not-supported",
			).WithErrorKey("PlanChangeNotSupported").Build()
		default:
			return result, err
		}
	}
	serviceTags, err := ap.Client.GetServiceTags(&aiven.GetServiceTagsInput{
		ServiceName: ap.BuildServiceName(updateData.InstanceID),
	})
	if err != nil {
		return result, err
	}

	serviceTags.PlanID = plan.ID

	_, err = ap.Client.UpdateServiceTags(&aiven.UpdateServiceTagsInput{
		ServiceName: ap.BuildServiceName(updateData.InstanceID),
		Tags:        *serviceTags,
	})
	if err != nil {
		return result, fmt.Errorf("Error updating tags for service %s", ap.BuildServiceName(updateData.InstanceID))
	}
	result.OperationData = ""
	result.IsAsync = asyncAllowed
	return
}

func (ap *AivenProvider) LastOperation(
	ctx context.Context,
	lastOperationData LastOperationData,
) (state domain.LastOperationState, description string, err error) {
	serviceName := ap.BuildServiceName(lastOperationData.InstanceID)

	service, err := ap.Client.GetService(&aiven.GetServiceInput{
		ServiceName: serviceName,
	})
	if err != nil {
		if lastOperationData.OperationData == "deprovisioning" {
			if err == aiven.ErrInstanceDoesNotExist {
				return domain.Succeeded, "Service has been deleted", nil
			}
		}
		return "", "", err
	}

	status := service.State
	updateTime := service.UpdateTime

	if updateTime.After(time.Now().Add(-1 * 60 * time.Second)) {
		return domain.InProgress, "Preparing to apply update", nil
	}

	lastOperationState, description := providerStatesMapping(status)
	return lastOperationState, description, nil
}

func (ap *AivenProvider) restoreFromPointInTime(
	ctx context.Context,
	provisionData ProvisionData,
	asyncAllowed bool,
	provisionParameters ProvisionParameters,
	userConfig aiven.UserConfig, tags aiven.ServiceTags,
) error {
	return fmt.Errorf("restoreFromPointInTime not implemented yet.")
}

func (ap *AivenProvider) forkFromBackup(
	ctx context.Context,
	provisionData ProvisionData,
	asyncAllowed bool,
	provisionParameters ProvisionParameters,
	userConfig aiven.UserConfig, tags aiven.ServiceTags,
) error {
	if *provisionParameters.RestoreFromLatestBackupOf == "" {
		return fmt.Errorf("Invalid guid: '%s'", *provisionParameters.RestoreFromLatestBackupOf)
	}
	if service := provisionData.Service.Name; service != "" {
		if service != "elasticsearch" && service != "opensearch" {
			return fmt.Errorf("Restore from backup not supported for service '%s'", service)
		}
	}
	forkFromBackupInstanceName := ap.BuildServiceName(*provisionParameters.RestoreFromLatestBackupOf)

	sourceService, err := ap.Client.GetService(&aiven.GetServiceInput{
		ServiceName: forkFromBackupInstanceName,
	})
	if err != nil {
		return err
	}
	sourceServiceTags, err := ap.Client.GetServiceTags(&aiven.GetServiceTagsInput{
		ServiceName: forkFromBackupInstanceName,
	})
	if err != nil {
		return err
	}
	if provisionData.Service.Name[len(provisionData.Service.Name)-6:] != sourceService.ServiceType[len(sourceService.ServiceType)-6:] {
		return fmt.Errorf("You cannot restore an %s backup to %s", sourceService.ServiceType, provisionData.Service.Name)
	}
	if err := ap.CheckPermissionsFromTags(provisionData.Details, sourceServiceTags); err != nil {
		return err
	}
	backups := sourceService.Backups
	sort.SliceStable(backups, func(i, j int) bool {
		return backups[i].Time.After(backups[j].Time)
	})

	if provisionParameters.RestoreFromLatestBackupBefore != nil {
		if *provisionParameters.RestoreFromLatestBackupBefore == "" {
			return fmt.Errorf("Parameter restore_from_latest_snapshot_before must not be empty")
		}

		restoreFromLatestSnapshotBeforeTime, err := time.ParseInLocation(
			RestoreFromLatestBackupBeforeTimeFormat,
			*provisionParameters.RestoreFromLatestBackupBefore,
			time.UTC,
		)
		if err != nil {
			return fmt.Errorf("Parameter restore_from_latest_snapshot_before should be a date and a time: %s", err)
		}

		prunedBackups := make([]aiven.ServiceBackup, 0)
		for _, backup := range backups {
			if backup.Time.Before(restoreFromLatestSnapshotBeforeTime) {
				prunedBackups = append(prunedBackups, backup)
			}
		}

		ap.Logger.Info("pruned-backups", lager.Data{
			"instanceIDLogKey":   provisionData.InstanceID,
			"detailsLogKey":      provisionData.Details,
			"allBackupsCount":    len(sourceService.Backups),
			"prunedBackupsCount": len(prunedBackups),
		})

		backups = prunedBackups
	}

	if len(backups) == 0 {
		return fmt.Errorf("No backups found for '%s'", *provisionParameters.RestoreFromLatestBackupOf)
	}

	backup := backups[0]

	ap.Logger.Info("chose-snapshot", lager.Data{
		"instanceIDLogKey":   provisionData.InstanceID,
		"detailsLogKey":      provisionData.Details,
		"snapshotIdentifier": backup.Name,
	})
	tags.RestoredFromBackup = "true"
	tags.RestoredFromTime = backup.Time
	userConfig.ForkProject = ap.Config.Project
	userConfig.BackupServiceName = forkFromBackupInstanceName
	userConfig.BackupName = backup.Name
	forkServiceInput := aiven.ForkServiceInput{
		Cloud:       ap.Config.Cloud,
		Plan:        sourceService.Plan,
		ServiceName: ap.BuildServiceName(provisionData.InstanceID),
		ServiceType: provisionData.Service.Name,
		UserConfig:  userConfig,
		Tags:        tags,
	}
	if err != nil {
		return err
	}
	_, err = ap.Client.ForkService(&forkServiceInput)
	return err
}

func (ap *AivenProvider) BuildServiceName(guid string) string {
	return strings.ToLower(ap.Config.ServiceNamePrefix + "-" + guid)
}

func (ap *AivenProvider) CheckPermissionsFromTags(
	details domain.ProvisionDetails,
	tags *aiven.ServiceTags,
) error {
	if tags.SpaceID != details.SpaceGUID || tags.OrganizationID != details.OrganizationGUID {
		return fmt.Errorf("The service instance you are getting a backup from is not in the same org or space")
	}
	return nil
}

func ParseIPWhitelist(ips string) ([]string, error) {
	if ips == "" {
		return []string{}, nil
	}
	outIPs := []string{}
	for _, ip := range strings.Split(ips, ",") {
		if len(strings.Split(ip, ".")) != 4 {
			return []string{}, fmt.Errorf("malformed whitelist IP: %v", ip)
		}
		outIPs = append(outIPs, ip)
	}
	return outIPs, nil
}

func providerStatesMapping(status aiven.ServiceStatus) (domain.LastOperationState, string) {
	switch status {
	case aiven.Running:
		return domain.Succeeded, "Last operation succeeded"
	case aiven.Rebuilding:
		return domain.InProgress, "Rebuilding"
	case aiven.Rebalancing:
		return domain.InProgress, "Rebalancing"
	case aiven.PowerOff:
		return domain.Failed, "Last operation failed: service is powered off"
	default:
		return domain.InProgress, fmt.Sprintf("Unknown state: %s", status)
	}
}
