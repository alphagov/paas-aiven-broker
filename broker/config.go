package broker

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/henrytk/universal-service-broker/provider"
	"github.com/pivotal-cf/brokerapi"
)

type Config struct {
	API      API
	Catalog  Catalog
	Provider provider.Provider
}

func NewConfig(source io.Reader) (Config, error) {
	config := Config{}
	bytes, err := ioutil.ReadAll(source)
	if err != nil {
		return config, err
	}

	api := API{}
	if err = json.Unmarshal(bytes, &api); err != nil {
		return config, err
	}

	catalog := Catalog{}
	if err = json.Unmarshal(bytes, &catalog); err != nil {
		return config, err
	}

	provider := provider.Provider{}
	if err = json.Unmarshal(bytes, &provider); err != nil {
		return config, err
	}

	return Config{
		API:      api,
		Catalog:  catalog,
		Provider: provider,
	}, nil
}

func (c Config) Validate() error {
	if c.API.BasicAuthUsername == "" {
		return fmt.Errorf("Config error: basic auth username required")
	}
	if c.API.BasicAuthPassword == "" {
		return fmt.Errorf("Config error: basic auth password required")
	}
	return nil
}

type API struct {
	BasicAuthUsername string `json:"basic_auth_username"`
	BasicAuthPassword string `json:"basic_auth_password"`
}

type Catalog struct {
	Catalog brokerapi.CatalogResponse `json:"catalog"`
}

func findServiceByID(catalog Catalog, serviceID string) (brokerapi.Service, error) {
	for _, service := range catalog.Catalog.Services {
		if service.ID == serviceID {
			return service, nil
		}
	}
	return brokerapi.Service{}, fmt.Errorf("Error: service %s not found in the catalog", serviceID)
}

func findPlanByID(service brokerapi.Service, planID string) (brokerapi.ServicePlan, error) {
	for _, plan := range service.Plans {
		if plan.ID == planID {
			return plan, nil
		}
	}
	return brokerapi.ServicePlan{}, fmt.Errorf("Error: plan %s not found in service %s", planID, service.ID)
}
