package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alphagov/paas-aiven-broker/client/elastic"
	"github.com/alphagov/paas-aiven-broker/client/influxdb"
	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/pivotal-cf/brokerapi"
)

const AIVEN_BASE_URL string = "https://api.aiven.io"

type AivenProvider struct {
	Client aiven.Client
	Config *Config
}

func New(configJSON []byte) (*AivenProvider, error) {
	config, err := DecodeConfig(configJSON)
	if err != nil {
		return nil, err
	}
	client := aiven.NewHttpClient(AIVEN_BASE_URL, config.APIToken, config.Project)
	return &AivenProvider{
		Client: client,
		Config: config,
	}, nil
}

func (ap *AivenProvider) Provision(ctx context.Context, provisionData ProvisionData) (dashboardURL, operationData string, err error) {
	plan, err := ap.Config.FindPlan(provisionData.Service.ID, provisionData.Plan.ID)
	if err != nil {
		return "", "", err
	}
	ipFilter, err := ParseIPWhitelist(os.Getenv("IP_WHITELIST"))
	if err != nil {
		return "", "", err
	}

	userConfig := aiven.UserConfig{}
	userConfig.IPFilter = ipFilter

	if provisionData.Service.Name == "elasticsearch" {
		userConfig.ElasticsearchVersion = plan.ElasticsearchVersion
	} else if provisionData.Service.Name == "influxdb" {
		// Nothing to do
	} else {
		return "", "", fmt.Errorf(
			"Cannot provision service for unknown service %s",
			provisionData.Service.Name,
		)
	}

	createServiceInput := &aiven.CreateServiceInput{
		Cloud:       ap.Config.Cloud,
		Plan:        plan.AivenPlan,
		ServiceName: buildServiceName(ap.Config.ServiceNamePrefix, provisionData.InstanceID),
		ServiceType: provisionData.Service.Name,
		UserConfig:  userConfig,
	}
	_, err = ap.Client.CreateService(createServiceInput)
	return dashboardURL, operationData, err
}

func (ap *AivenProvider) Deprovision(ctx context.Context, deprovisionData DeprovisionData) (operationData string, err error) {
	err = ap.Client.DeleteService(&aiven.DeleteServiceInput{
		ServiceName: buildServiceName(ap.Config.ServiceNamePrefix, deprovisionData.InstanceID),
	})

	if err != nil {
		if err == aiven.ErrInstanceDoesNotExist {
			return "", brokerapi.ErrInstanceDoesNotExist
		}
	}

	return "", err
}

func (ap *AivenProvider) Bind(ctx context.Context, bindData BindData) (binding brokerapi.Binding, err error) {
	serviceName := buildServiceName(ap.Config.ServiceNamePrefix, bindData.InstanceID)
	user := bindData.BindingID

	password, err := ap.Client.CreateServiceUser(&aiven.CreateServiceUserInput{
		ServiceName: serviceName,
		Username:    user,
	})
	if err != nil {
		return brokerapi.Binding{}, err
	}

	service, err := ap.Client.GetService(&aiven.GetServiceInput{
		ServiceName: serviceName,
	})
	if err != nil {
		return brokerapi.Binding{}, err
	}

	host := service.ServiceUriParams.Host
	port := service.ServiceUriParams.Port
	serviceType := service.ServiceType

	if host == "" || port == "" {
		return brokerapi.Binding{}, errors.New(
			"Error getting service connection details: no connection details found in response JSON",
		)
	}

	credentials, err := BuildCredentials(serviceType, user, password, host, port)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	if err = ensureUserAvailability(ctx, serviceType, credentials); err != nil {
		// Polling is only a best-effort attempt to work around Aiven API delays.
		// We therefore continue anyway if it times out.
		if err != context.DeadlineExceeded {
			return brokerapi.Binding{}, err
		}
	}

	return brokerapi.Binding{
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
			client := elastic.New(credentials.URI, nil)
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
		ServiceName: buildServiceName(ap.Config.ServiceNamePrefix, unbindData.InstanceID),
		Username:    unbindData.BindingID,
	})
	return err
}

func (ap *AivenProvider) Update(ctx context.Context, updateData UpdateData) (operationData string, err error) {
	plan, err := ap.Config.FindPlan(updateData.Details.ServiceID, updateData.Details.PlanID)
	if err != nil {
		return "", err
	}

	ipFilter, err := ParseIPWhitelist(os.Getenv("IP_WHITELIST"))
	if err != nil {
		return "", err
	}

	userConfig := aiven.UserConfig{}
	userConfig.IPFilter = ipFilter
	userConfig.ElasticsearchVersion = plan.ElasticsearchVersion // Pass empty version through if not InfluxDB

	_, err = ap.Client.UpdateService(&aiven.UpdateServiceInput{
		ServiceName: buildServiceName(ap.Config.ServiceNamePrefix, updateData.InstanceID),
		Plan:        plan.AivenPlan,
		UserConfig:  userConfig,
	})

	switch err := err.(type) {
	case aiven.ErrInvalidUpdate:
		return "", brokerapi.NewFailureResponseBuilder(
			err,
			http.StatusUnprocessableEntity,
			"plan-change-not-supported",
		).WithErrorKey("PlanChangeNotSupported").Build()
	default:
		return "", err
	}
}

func (ap *AivenProvider) LastOperation(
	ctx context.Context,
	lastOperationData LastOperationData,
) (state brokerapi.LastOperationState, description string, err error) {
	serviceName := buildServiceName(
		ap.Config.ServiceNamePrefix,
		lastOperationData.InstanceID,
	)

	service, err := ap.Client.GetService(&aiven.GetServiceInput{
		ServiceName: serviceName,
	})

	if err != nil {
		return "", "", err
	}

	status := service.State
	updateTime := service.UpdateTime

	if updateTime.After(time.Now().Add(-1 * 60 * time.Second)) {
		return brokerapi.InProgress, "Preparing to apply update", nil
	}

	lastOperationState, description := providerStatesMapping(status)
	return lastOperationState, description, nil
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

func buildServiceName(prefix, guid string) string {
	return strings.ToLower(prefix + "-" + guid)
}

func providerStatesMapping(status aiven.ServiceStatus) (brokerapi.LastOperationState, string) {
	switch status {
	case aiven.Running:
		return brokerapi.Succeeded, "Last operation succeeded"
	case aiven.Rebuilding:
		return brokerapi.InProgress, "Rebuilding"
	case aiven.Rebalancing:
		return brokerapi.InProgress, "Rebalancing"
	case aiven.PowerOff:
		return brokerapi.Failed, "Last operation failed: service is powered off"
	default:
		return brokerapi.InProgress, fmt.Sprintf("Unknown state: %s", status)
	}
}
