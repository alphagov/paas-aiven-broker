package broker

import (
	"context"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/henrytk/broker-skeleton/provider"
	"github.com/pivotal-cf/brokerapi"
)

type Broker struct {
	config   Config
	Provider provider.ServiceProvider
	logger   lager.Logger
}

func New(config Config, serviceProvider provider.ServiceProvider, logger lager.Logger) *Broker {
	return &Broker{
		config:   config,
		Provider: serviceProvider,
		logger:   logger,
	}
}

func (b *Broker) Services(ctx context.Context) []brokerapi.Service {
	return b.config.Catalog.Catalog.Services
}

func (b *Broker) Provision(
	ctx context.Context,
	instanceID string,
	details brokerapi.ProvisionDetails,
	asyncAllowed bool,
) (brokerapi.ProvisionedServiceSpec, error) {
	b.logger.Debug("provision-start", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	if !asyncAllowed {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	service, err := findServiceByID(b.config.Catalog, details.ServiceID)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	plan, err := findPlanByID(service, details.PlanID)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	provisionData := provider.ProvisionData{
		InstanceID:      instanceID,
		Details:         details,
		Service:         service,
		Plan:            plan,
		ProviderCatalog: b.config.Provider.Catalog,
	}

	dashboardURL, operationData, err := b.Provider.Provision(providerCtx, provisionData)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	b.logger.Debug("provision-success", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	return brokerapi.ProvisionedServiceSpec{
		IsAsync:       asyncAllowed,
		DashboardURL:  dashboardURL,
		OperationData: operationData,
	}, nil
}

func (b *Broker) Deprovision(
	ctx context.Context,
	instanceID string,
	details brokerapi.DeprovisionDetails,
	asyncAllowed bool,
) (brokerapi.DeprovisionServiceSpec, error) {
	b.logger.Debug("deprovision-start", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	if !asyncAllowed {
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	deprovisionData := provider.DeprovisionData{
		InstanceID:      instanceID,
		Details:         details,
		ProviderCatalog: b.config.Provider.Catalog,
	}

	operationData, err := b.Provider.Deprovision(providerCtx, deprovisionData)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	b.logger.Debug("deprovision-success", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	return brokerapi.DeprovisionServiceSpec{
		IsAsync:       asyncAllowed,
		OperationData: operationData,
	}, nil
}

func (b *Broker) Bind(
	ctx context.Context,
	instanceID,
	bindingID string,
	details brokerapi.BindDetails,
) (brokerapi.Binding, error) {
	return brokerapi.Binding{}, nil
}

func (b *Broker) Unbind(
	ctx context.Context,
	instanceID,
	bindingID string,
	details brokerapi.UnbindDetails,
) error {
	return nil
}

func (b *Broker) Update(
	ctx context.Context,
	instanceID string,
	details brokerapi.UpdateDetails,
	asyncAllowed bool,
) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, nil
}

func (b *Broker) LastOperation(
	ctx context.Context,
	instanceID,
	operationData string,
) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, nil
}
