package broker

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-aiven-broker/provider"
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

func (b *Broker) GetBinding(ctx context.Context, first, second string) (brokerapi.GetBindingSpec, error) {
	return brokerapi.GetBindingSpec{}, fmt.Errorf("GetBinding method not implemented")
}

func (b *Broker) GetInstance(ctx context.Context, first string) (brokerapi.GetInstanceDetailsSpec, error) {
	return brokerapi.GetInstanceDetailsSpec{}, fmt.Errorf("GetInstance method not implemented")
}

func (b *Broker) LastBindingOperation(ctx context.Context, first, second string, pollDetails brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, fmt.Errorf("LastBindingOperation method not implemented")
}

func (b *Broker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return b.config.Catalog.Catalog.Services, nil
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
		InstanceID: instanceID,
		Details:    details,
		Service:    service,
		Plan:       plan,
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

	service, err := findServiceByID(b.config.Catalog, details.ServiceID)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	plan, err := findPlanByID(service, details.PlanID)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	deprovisionData := provider.DeprovisionData{
		InstanceID: instanceID,
		Service:    service,
		Plan:       plan,
		Details:    details,
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
		IsAsync:       false,
		OperationData: operationData,
	}, nil
}

func (b *Broker) Bind(
	ctx context.Context,
	instanceID, bindingID string,
	details brokerapi.BindDetails,
	asyncAllowed bool,
) (brokerapi.Binding, error) {
	b.logger.Debug("binding-start", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	bindData := provider.BindData{
		InstanceID: instanceID,
		BindingID:  bindingID,
		Details:    details,
	}

	binding, err := b.Provider.Bind(providerCtx, bindData)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	b.logger.Debug("binding-success", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	return binding, nil
}

func (b *Broker) Unbind(
	ctx context.Context,
	instanceID, bindingID string,
	details brokerapi.UnbindDetails,
	asyncAllowed bool,
) (brokerapi.UnbindSpec, error) {
	b.logger.Debug("unbinding-start", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	unbindData := provider.UnbindData{
		InstanceID: instanceID,
		BindingID:  bindingID,
		Details:    details,
	}

	err := b.Provider.Unbind(providerCtx, unbindData)
	if err != nil {
		return brokerapi.UnbindSpec{}, err
	}

	b.logger.Debug("unbinding-success", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	return brokerapi.UnbindSpec{}, nil
}

func (b *Broker) Update(
	ctx context.Context,
	instanceID string,
	details brokerapi.UpdateDetails,
	asyncAllowed bool,
) (brokerapi.UpdateServiceSpec, error) {
	b.logger.Debug("update-start", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	if !asyncAllowed {
		return brokerapi.UpdateServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	service, err := findServiceByID(b.config.Catalog, details.ServiceID)
	if err != nil {
		return brokerapi.UpdateServiceSpec{}, err
	}

	if !service.PlanUpdatable && details.PlanID != details.PreviousValues.PlanID {
		return brokerapi.UpdateServiceSpec{}, brokerapi.ErrPlanChangeNotSupported
	}

	plan, err := findPlanByID(service, details.PlanID)
	if err != nil {
		return brokerapi.UpdateServiceSpec{}, err
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	updateData := provider.UpdateData{
		InstanceID: instanceID,
		Details:    details,
		Service:    service,
		Plan:       plan,
	}

	operationData, err := b.Provider.Update(providerCtx, updateData)
	if err != nil {
		return brokerapi.UpdateServiceSpec{}, err
	}

	b.logger.Debug("update-success", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	return brokerapi.UpdateServiceSpec{
		IsAsync:       asyncAllowed,
		OperationData: operationData,
	}, nil
}

func (b *Broker) LastOperation(
	ctx context.Context,
	instanceID string,
	pollDetails brokerapi.PollDetails,
) (brokerapi.LastOperation, error) {
	b.logger.Debug("last-operation-start", lager.Data{
		"instance-id":    instanceID,
		"operation-data": pollDetails.OperationData,
	})

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	lastOperationData := provider.LastOperationData{
		InstanceID:    instanceID,
		OperationData: pollDetails.OperationData,
	}

	state, description, err := b.Provider.LastOperation(providerCtx, lastOperationData)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}

	b.logger.Debug("last-operation-success", lager.Data{
		"instance-id":    instanceID,
		"operation-data": pollDetails.OperationData,
	})

	return brokerapi.LastOperation{
		State:       state,
		Description: description,
	}, nil
}
