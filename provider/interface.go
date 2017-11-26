package provider

import "context"

type ServiceProvider interface {
	Provision(context.Context, ProvisionData) (dashboardURL, operationData string, err error)
}
