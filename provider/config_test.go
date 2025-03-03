package provider_test

import (
	"encoding/json"
	"os"

	"github.com/alphagov/paas-aiven-broker/provider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

var _ = Describe("Config", func() {
	var (
		rawConfig json.RawMessage
	)

	Context("when everything is configured correctly", func() {
		It("returns the correct values", func() {
			rawConfig = json.RawMessage(`
						{
							"cloud": "aws-eu-west-1",
							"catalog": {
								"services": [
									{
										"name": "opensearch",
										"plans": [{
											"aiven_plan": "startup-2",
											"opensearch_version": "1"
										}]
									},
									{
										"name": "influxdb",
										"plans": [{
											"aiven_plan": "startup-3"
										}]
									}
								]
							}
						}
					`)

			opensearchPlanSpecificConfig := provider.PlanSpecificConfig{}
			opensearchPlanSpecificConfig.AivenPlan = "startup-2"
			opensearchPlanSpecificConfig.OpenSearchVersion = "1"

			influxDBPlanSpecificConfig := provider.PlanSpecificConfig{}
			influxDBPlanSpecificConfig.AivenPlan = "startup-3"

			expectedConfig := &provider.Config{
				Cloud:             "aws-eu-west-1",
				DeployEnv:         "test",
				BrokerName:        "test",
				ServiceNamePrefix: "test",
				APIToken:          "token",
				Project:           "project",
				Catalog: provider.Catalog{
					Services: []provider.Service{
						{
							Service: domain.Service{
								Name: "opensearch",
							},
							Plans: []provider.Plan{
								{
									PlanSpecificConfig: opensearchPlanSpecificConfig,
								},
							},
						},
						{
							Service: domain.Service{
								Name: "influxdb",
							},
							Plans: []provider.Plan{
								{
									PlanSpecificConfig: influxDBPlanSpecificConfig,
								},
							},
						},
					},
				},
			}

			actualConfig, err := provider.DecodeConfig(rawConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualConfig).To(Equal(expectedConfig))
		})
	})

	Context("when there is no Catalog defined", func() {
		It("returns an error", func() {
			rawConfig = json.RawMessage(`{"cloud": "aws-eu-west-1"}`)
			_, err := provider.DecodeConfig(rawConfig)
			Expect(err).To(MatchError("Config error: no catalog found"))
		})
	})

	Context("when there are no services configured", func() {
		It("returns an error", func() {
			rawConfig = json.RawMessage(`
					{
						"cloud": "aws-eu-west-1",
						"catalog": {
							"services": []
						}
					}
				`)
			_, err := provider.DecodeConfig(rawConfig)
			Expect(err).To(MatchError("Config error: at least one service must be configured"))
		})
	})

	Context("when a service has no plans", func() {
		It("returns an error", func() {
			rawConfig = json.RawMessage(`
					{
						"cloud": "aws-eu-west-1",
						"catalog": {
							"services": [
								{
									"name": "opensearch",
									"plans": []
								}
							]
						}
					}
				`)
			_, err := provider.DecodeConfig(rawConfig)
			Expect(err).To(MatchError("Config error: at least one plan must be configured for service opensearch"))
		})
	})

	Describe("Mandatory parameters", func() {

		BeforeEach(func() {
			os.Unsetenv("AIVEN_USERNAME")
			os.Unsetenv("AIVEN_PASSWORD")
			os.Unsetenv("AIVEN_CLOUD")
		})
		It("returns an error if `cloud` is empty", func() {
			rawConfig = json.RawMessage(`
						{
							"catalog": {
								"services": []
							}
						}
					`)
			_, err := provider.DecodeConfig(rawConfig)
			Expect(err).To(MatchError("Config error: must provide cloud configuration. For example, 'aws-eu-west-1'"))
		})

		It("returns an error if a plan is missing the Aiven plan details", func() {
			rawConfig = json.RawMessage(`
						{
							"cloud": "aws-eu-west-1",
							"catalog": {
								"services": [
									{
										"name": "opensearch",
										"plans": [{"name": "plan-a"}]
									}
								]
							}
						}
					`)
			_, err := provider.DecodeConfig(rawConfig)
			Expect(err).To(MatchError("Config error: every plan must specify an `aiven_plan`"))
		})

		Context("when the service is opensearch", func() {
			It("returns an error if a plan is missing the OpenSearch version", func() {
				rawConfig = json.RawMessage(`
							{
								"cloud": "aws-eu-west-1",
								"catalog": {
									"services": [
										{
											"name": "opensearch",
											"plans": [{"aiven_plan": "plan-a"}]
										}
									]
								}
							}
						`)
				_, err := provider.DecodeConfig(rawConfig)
				Expect(err).To(MatchError("Config error: every opensearch plan must specify an `opensearch_version`"))
			})
		})

		Context("when the service is InfluxDB", func() {
			It("does not care about the OpenSearch version", func() {
				rawConfig = json.RawMessage(`
							{
								"cloud": "aws-eu-west-1",
								"catalog": {
									"services": [
										{
											"name": "influxDB",
											"plans": [{"aiven_plan": "plan-a"}]
										}
									]
								}
							}
						`)
				_, err := provider.DecodeConfig(rawConfig)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
