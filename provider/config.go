package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/pivotal-cf/brokerapi/domain"
)

type Config struct {
	DeployEnv         string
	BrokerName        string `json:"name"`
	Cloud             string `json:"cloud"`
	ServiceNamePrefix string
	APIToken          string
	Project           string
	Catalog           Catalog `json:"catalog"`
}

type Catalog struct {
	Services []Service `json:"services"`
}

type Service struct {
	domain.Service
	Plans []Plan `json:"plans"`
}

type Plan struct {
	domain.ServicePlan
	PlanSpecificConfig
}

type AivenServiceCommonConfig struct{}

type AivenServiceOpenSearchConfig struct {
	OpenSearchVersion string `json:"opensearch_version"`
}

type AivenServiceInfluxDBConfig struct{}

type PlanSpecificConfig struct {
	AivenPlan string `json:"aiven_plan"`

	AivenServiceCommonConfig
	AivenServiceOpenSearchConfig
	AivenServiceInfluxDBConfig
}

func DecodeConfig(b []byte) (*Config, error) {
	var config *Config
	err := json.Unmarshal(b, &config)
	if err != nil {
		return config, err
	}
	aivenCloud, ok := os.LookupEnv("AIVEN_CLOUD")
	if ok {
		config.Cloud = aivenCloud
	}
	if config.Cloud == "" {
		return config, errors.New("Config error: must provide cloud configuration. For example, 'aws-eu-west-1'")
	}
	if reflect.DeepEqual(config.Catalog, Catalog{}) {
		return config, errors.New("Config error: no catalog found")
	}
	if len(config.Catalog.Services) == 0 {
		return config, errors.New("Config error: at least one service must be configured")
	}

	for _, service := range config.Catalog.Services {
		if len(service.Plans) == 0 {
			return config, errors.New("Config error: at least one plan must be configured for service " + service.Name)
		}
		for _, plan := range service.Plans {
			if plan.AivenPlan == "" {
				return config, errors.New("Config error: every plan must specify an `aiven_plan`")
			}

			if service.Name == "opensearch" && plan.OpenSearchVersion == "" {
				return config, errors.New("Config error: every opensearch plan must specify an `opensearch_version`")
			}
		}
	}
	config.BrokerName = os.Getenv("BROKER_NAME")
	if config.BrokerName == "" {
		return config, errors.New("Config error: must declare a Broker name")
	}

	config.DeployEnv = os.Getenv("DEPLOY_ENV")
	if config.DeployEnv == "" {
		return config, errors.New("Config error: must declare a Deploy Environment")
	}
	config.ServiceNamePrefix = config.DeployEnv
	serviceNamePrefixFromEnv := os.Getenv("SERVICE_NAME_PREFIX")
	if serviceNamePrefixFromEnv != "" {
		config.ServiceNamePrefix = fmt.Sprintf("%s-%s", config.ServiceNamePrefix, serviceNamePrefixFromEnv)
	}

	// Aiven only allow 64 characters for the service name. The instanceID from Cloud Foundry
	// is joined with a hyphen to the service name prefix. This gives us 27 characters to use.
	if len(config.ServiceNamePrefix) > 27 {
		return config, errors.New("Config error: service name prefix cannot be longer than 8 characters")
	}

	config.APIToken = os.Getenv("AIVEN_API_TOKEN")
	if config.APIToken == "" {
		return config, errors.New("Config error: must pass an Aiven API token")
	}

	config.Project = os.Getenv("AIVEN_PROJECT")
	if config.Project == "" {
		return config, errors.New("Config error: must declare an Aiven project name")
	}

	return config, nil
}

func (c *Config) FindPlan(serviceId, planId string) (*Plan, error) {
	service, err := findServiceById(serviceId, &c.Catalog)
	if err != nil {
		return &Plan{}, err
	}
	plan, err := findPlanById(planId, service)
	if err != nil {
		return &Plan{}, err
	}
	return &plan, nil
}

func findServiceById(id string, catalog *Catalog) (Service, error) {
	for _, service := range catalog.Services {
		if service.ID == id {
			return service, nil
		}
	}
	return Service{}, errors.New("could not find service with id " + id)
}

func findPlanById(id string, service Service) (Plan, error) {
	for _, plan := range service.Plans {
		if plan.ID == id {
			return plan, nil
		}
	}
	return Plan{}, errors.New("could not find plan with id " + id)
}
