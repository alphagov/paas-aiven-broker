package broker

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/henrytk/universal-service-broker/provider"
	"github.com/pivotal-cf/brokerapi"
)

const (
	DefaultPort     = "3000"
	DefaultLogLevel = "debug"
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
	if api.Port == "" {
		api.Port = "3000"
	}
	if api.LogLevel == "" {
		api.LogLevel = "debug"
	}

	catalog := Catalog{}
	if err = json.Unmarshal(bytes, &catalog); err != nil {
		return config, err
	}

	provider := provider.Provider{}
	if err = json.Unmarshal(bytes, &provider); err != nil {
		return config, err
	}

	config = Config{
		API:      api,
		Catalog:  catalog,
		Provider: provider,
	}

	err = config.Validate()

	return config, err
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
	Port              string `json:"port"`
	LogLevel          string `json:"log_level"`
}

func (api API) LagerLogLevel() lager.LogLevel {
	logLevels := map[string]lager.LogLevel{
		"DEBUG": lager.DEBUG,
		"INFO":  lager.INFO,
		"ERROR": lager.ERROR,
		"FATAL": lager.FATAL,
	}

	return logLevels[strings.ToUpper(api.LogLevel)]
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
