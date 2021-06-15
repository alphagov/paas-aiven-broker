package provider

import (
	"context"
	"github.com/pivotal-cf/brokerapi/domain"
)

type ServiceProvider interface {
	Provision(context.Context, ProvisionData, domain.ProvisionDetails) (dashboardURL, operationData string, err error)
	Deprovision(context.Context, DeprovisionData) (operationData string, err error)
	Bind(context.Context, BindData) (binding domain.Binding, err error)
	Unbind(context.Context, UnbindData) (err error)
	Update(context.Context, UpdateData, domain.UpdateDetails) (operationData string, err error)
	LastOperation(context.Context, LastOperationData) (state domain.LastOperationState, description string, err error)
}
