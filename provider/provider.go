package provider

import (
	"context"
	"fmt"
	"hash/crc32"

	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/pivotal-cf/brokerapi"
)

const AIVEN_BASE_URL string = "https://api.aiven.io"
const SERVICE_TYPE string = "elasticsearch"

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
	createServiceInput := &aiven.CreateServiceInput{
		Cloud:       ap.Config.Cloud,
		Plan:        plan.AivenPlan,
		ServiceName: BuildServiceName(ap.Config.ServiceNamePrefix, provisionData.InstanceID),
		ServiceType: SERVICE_TYPE,
		UserConfig: aiven.UserConfig{
			ElasticsearchVersion: plan.ElasticsearchVersion,
		},
	}
	_, err = ap.Client.CreateService(createServiceInput)
	return dashboardURL, operationData, err
}

func (ap *AivenProvider) Deprovision(ctx context.Context, deprovisionData DeprovisionData) (operationData string, err error) {
	_, err = ap.Client.DeleteService(&aiven.DeleteServiceInput{
		ServiceName: BuildServiceName(ap.Config.ServiceNamePrefix, deprovisionData.InstanceID),
	})
	if err != nil {
		return "", err
	}
	return "", nil
}

func (ap *AivenProvider) Bind(ctx context.Context, bindData BindData) (binding brokerapi.Binding, err error) {
	return brokerapi.Binding{}, fmt.Errorf("not implemented")
}

func (ap *AivenProvider) Unbind(ctx context.Context, unbindData UnbindData) (err error) {
	return fmt.Errorf("not implemented")
}

func (ap *AivenProvider) Update(ctx context.Context, updateData UpdateData) (operationData string, err error) {
	return "", fmt.Errorf("not implemented")
}

func (ap *AivenProvider) LastOperation(ctx context.Context, lastOperationData LastOperationData) (state brokerapi.LastOperationState, description string, err error) {
	status, err := ap.Client.GetServiceStatus(&aiven.GetServiceStatusInput{
		ServiceName: BuildServiceName(ap.Config.ServiceNamePrefix, lastOperationData.InstanceID),
	})
	if err != nil {
		return "", "", err
	}
	lastOperationState, description := ProviderStatesMapping(status)
	return lastOperationState, description, nil
}

func BuildServiceName(prefix, guid string) string {
	checksum := crc32.ChecksumIEEE([]byte(guid))
	return fmt.Sprintf("%s-%x", prefix, checksum)
}

func ProviderStatesMapping(status aiven.ServiceStatus) (brokerapi.LastOperationState, string) {
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
