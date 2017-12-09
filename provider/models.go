package provider

import (
	"encoding/json"

	"github.com/pivotal-cf/brokerapi"
)

type Provider struct {
	Catalog ProviderCatalog `json:"catalog"`
}

type ProviderCatalog struct {
	Services []ProviderService `json:"services"`
}

type ProviderService struct {
	ID             string          `json:"id"`
	ProviderConfig json.RawMessage `json:"provider_config"`
	Plans          []ProviderPlan  `json:"plans"`
}

type ProviderPlan struct {
	ID             string          `json:"id"`
	ProviderConfig json.RawMessage `json:"provider_config"`
}

type ProvisionData struct {
	InstanceID      string
	Details         brokerapi.ProvisionDetails
	Service         brokerapi.Service
	Plan            brokerapi.ServicePlan
	ProviderCatalog ProviderCatalog
}

type DeprovisionData struct {
	InstanceID      string
	Details         brokerapi.DeprovisionDetails
	Service         brokerapi.Service
	Plan            brokerapi.ServicePlan
	ProviderCatalog ProviderCatalog
}
