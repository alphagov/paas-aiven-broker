package broker

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
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

func (b *Broker) GetBinding(ctx context.Context, first, second string, _ domain.FetchBindingDetails) (domain.GetBindingSpec, error) {
	return domain.GetBindingSpec{}, fmt.Errorf("GetBinding method not implemented")
}

func (b *Broker) GetInstance(ctx context.Context, first string, _ domain.FetchInstanceDetails) (domain.GetInstanceDetailsSpec, error) {
	return domain.GetInstanceDetailsSpec{}, fmt.Errorf("GetInstance method not implemented")
}

func (b *Broker) LastBindingOperation(ctx context.Context, first, second string, pollDetails domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, fmt.Errorf("LastBindingOperation method not implemented")
}

func (b *Broker) Services(ctx context.Context) ([]domain.Service, error) {
	return b.config.Catalog.Catalog.Services, nil
}

func (b *Broker) Provision(
	ctx context.Context,
	instanceID string,
	details domain.ProvisionDetails,
	asyncAllowed bool,
) (domain.ProvisionedServiceSpec, error) {
	b.logger.Debug("provision-start", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	if !asyncAllowed {
		return domain.ProvisionedServiceSpec{}, apiresponses.ErrAsyncRequired
	}

	service, err := findServiceByID(b.config.Catalog, details.ServiceID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	plan, err := findPlanByID(service, details.PlanID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	provisionData := provider.ProvisionData{
		InstanceID: instanceID,
		Details:    details,
		Service:    service,
		Plan:       plan,
	}

	operationData, err := b.Provider.Provision(providerCtx, provisionData, true)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, err
	}

	b.logger.Debug("provision-success", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	return operationData, nil
}

func (b *Broker) Deprovision(
	ctx context.Context,
	instanceID string,
	details domain.DeprovisionDetails,
	asyncAllowed bool,
) (domain.DeprovisionServiceSpec, error) {
	b.logger.Debug("deprovision-start", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	if !asyncAllowed {
		return domain.DeprovisionServiceSpec{}, apiresponses.ErrAsyncRequired
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	service, err := findServiceByID(b.config.Catalog, details.ServiceID)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, err
	}

	plan, err := findPlanByID(service, details.PlanID)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, err
	}

	deprovisionData := provider.DeprovisionData{
		InstanceID: instanceID,
		Service:    service,
		Plan:       plan,
		Details:    details,
	}

	operationData, err := b.Provider.Deprovision(providerCtx, deprovisionData)
	if err != nil {
		return domain.DeprovisionServiceSpec{}, err
	}

	b.logger.Debug("deprovision-success", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	return domain.DeprovisionServiceSpec{
		IsAsync:       asyncAllowed,
		OperationData: operationData,
	}, nil
}

func (b *Broker) Bind(
	ctx context.Context,
	instanceID, bindingID string,
	details domain.BindDetails,
	asyncAllowed bool,
) (domain.Binding, error) {
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
		return domain.Binding{}, err
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
	details domain.UnbindDetails,
	asyncAllowed bool,
) (domain.UnbindSpec, error) {
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
		return domain.UnbindSpec{}, err
	}

	b.logger.Debug("unbinding-success", lager.Data{
		"instance-id": instanceID,
		"binding-id":  bindingID,
		"details":     details,
	})

	return domain.UnbindSpec{}, nil
}

func (b *Broker) Update(
	ctx context.Context,
	instanceID string,
	details domain.UpdateDetails,
	asyncAllowed bool,
) (domain.UpdateServiceSpec, error) {
	b.logger.Debug("update-start", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	if !asyncAllowed {
		return domain.UpdateServiceSpec{}, apiresponses.ErrAsyncRequired
	}

	service, err := findServiceByID(b.config.Catalog, details.ServiceID)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	if !service.PlanUpdatable && details.PlanID != details.PreviousValues.PlanID {
		return domain.UpdateServiceSpec{}, apiresponses.ErrPlanChangeNotSupported
	}

	plan, err := findPlanByID(service, details.PlanID)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	providerCtx, cancelFunc := context.WithTimeout(ctx, 30*time.Second)
	defer cancelFunc()

	updateData := provider.UpdateData{
		InstanceID: instanceID,
		Details:    details,
		Service:    service,
		Plan:       plan,
	}

	updateServiceSpec, err := b.Provider.Update(providerCtx, updateData, asyncAllowed)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	b.logger.Debug("update-success", lager.Data{
		"instance-id":   instanceID,
		"details":       details,
		"async-allowed": asyncAllowed,
	})

	return updateServiceSpec, nil
}

func (b *Broker) LastOperation(
	ctx context.Context,
	instanceID string,
	pollDetails domain.PollDetails,
) (domain.LastOperation, error) {
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
		return domain.LastOperation{}, err
	}

	b.logger.Debug("last-operation-success", lager.Data{
		"instance-id":    instanceID,
		"operation-data": pollDetails.OperationData,
	})

	return domain.LastOperation{
		State:       state,
		Description: description,
	}, nil
}
