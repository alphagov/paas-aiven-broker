package metricsconverger

import (
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/alphagov/paas-aiven-broker/provider/aiven"
)

const (
	convergeLoopInterval = 120 * time.Second
	maxServicesToUpdate  = 5
	aivenBaseURL         = "https://api.aiven.io"
)

type MetricsConverger struct {
	config provider.Config
	Client aiven.Client
	logger lager.Logger
}

func New(
	configJSON []byte,
	logger lager.Logger,
) (*MetricsConverger, error) {
	config, err := provider.DecodeConfig(configJSON)
	if err != nil {
		return nil, err
	}

	client := aiven.NewHttpClient(
		aivenBaseURL,
		config.APIToken,
		config.Project,
	)

	return &MetricsConverger{
		config: *config,
		Client: client,
		logger: logger,
	}, nil
}

func (m *MetricsConverger) Converge() {
	ticker := time.NewTicker(convergeLoopInterval)

	for {
		select {
		case <-ticker.C:
			m.RunOnce()
		}
	}
}

func (m *MetricsConverger) RunOnce() {
	lsession := m.logger.Session("run-once")

	lsession.Info("begin")
	defer lsession.Info("end")

	allServices, err := m.Client.ListServices()
	if err != nil {
		lsession.Error("failed-to-list-services", err)
		return
	}

	eligibleServices := make([]aiven.Service, 0)
	for _, service := range allServices {
		switch service.ServiceType {
		case "elasticsearch":
			eligibleServices = append(eligibleServices, service)
		default:
			continue
		}
	}

	servicesToUpdate := make([]aiven.Service, 0)
	for _, service := range eligibleServices {
		shouldEnablePrometheus := true
		for _, integration := range service.ServiceIntegrations {
			if integration.IntegrationType == "prometheus" {
				shouldEnablePrometheus = false
			}
		}

		if shouldEnablePrometheus {
			servicesToUpdate = append(servicesToUpdate, service)
		}
	}

	for serviceIndex, service := range servicesToUpdate {
		if serviceIndex >= maxServicesToUpdate {
			break
		}

		lsession.Info("updating-service", lager.Data{"service": service})

		m.Client.CreateServiceIntegration(&aiven.CreateServiceIntegrationInput{
			IntegrationType:       "prometheus",
			DestinationEndpointID: m.config.PrometheusServiceIntegrationEndpointID,
			SourceService:         service.ServiceName,
		})

		defer lsession.Info("updated-service", lager.Data{"service": service})
	}
}
