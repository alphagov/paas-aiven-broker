package provider

import "context"

type ServiceProvider interface {
	Provision(context.Context, ProvisionData) (dashboardURL, operationData string, err error)
	Deprovision(context.Context, DeprovisionData) (operationData string, err error)
}
