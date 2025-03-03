package provider

import (
	"context"

	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_service_provider.go . ServiceProvider

type ServiceProvider interface {
	Provision(context.Context, ProvisionData, bool) (result domain.ProvisionedServiceSpec, err error)
	Deprovision(context.Context, DeprovisionData) (operationData string, err error)
	Bind(context.Context, BindData) (binding domain.Binding, err error)
	Unbind(context.Context, UnbindData) (err error)
	Update(context.Context, UpdateData, bool) (result domain.UpdateServiceSpec, err error)
	LastOperation(context.Context, LastOperationData) (state domain.LastOperationState, description string, err error)
	BuildServiceName(guid string) (serviceName string)
	CheckPermissionsFromTags(details domain.ProvisionDetails, tags *aiven.ServiceTags) (err error)
}
