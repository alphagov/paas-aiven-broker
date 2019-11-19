package provider_test

import (
	"encoding/json"

	"github.com/alphagov/paas-aiven-broker/provider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
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
										"name": "elasticsearch",
										"plans": [{
											"aiven_plan": "startup-1",
											"elasticsearch_version": "6"
										}]
									},
									{
										"name": "influxdb",
										"plans": [{
											"aiven_plan": "startup-2"
										}]
									}
								]
							}
						}
					`)

			elasticsearchPlanSpecificConfig := provider.PlanSpecificConfig{}
			elasticsearchPlanSpecificConfig.AivenPlan = "startup-1"
			elasticsearchPlanSpecificConfig.ElasticsearchVersion = "6"

			influxDBPlanSpecificConfig := provider.PlanSpecificConfig{}
			influxDBPlanSpecificConfig.AivenPlan = "startup-2"

			expectedConfig := &provider.Config{
				Cloud:             "aws-eu-west-1",
				ServiceNamePrefix: "test",
				APIToken:          "token",
				Project:           "project",
				Catalog: provider.Catalog{
					Services: []provider.Service{
						{
							Service: brokerapi.Service{
								Name: "elasticsearch",
							},
							Plans: []provider.Plan{
								{
									PlanSpecificConfig: elasticsearchPlanSpecificConfig,
								},
							},
						},
						{
							Service: brokerapi.Service{
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
									"name": "elasticsearch",
									"plans": []
								}
							]
						}
					}
				`)
			_, err := provider.DecodeConfig(rawConfig)
			Expect(err).To(MatchError("Config error: at least one plan must be configured for service elasticsearch"))
		})
	})

	Describe("Mandatory parameters", func() {
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
										"name": "elasticsearch",
										"plans": [{"name": "plan-a"}]
									}
								]
							}
						}
					`)
			_, err := provider.DecodeConfig(rawConfig)
			Expect(err).To(MatchError("Config error: every plan must specify an `aiven_plan`"))
		})

		Context("when the service is elasticsearch", func() {
			It("returns an error if a plan is missing the Elasticsearch version", func() {
				rawConfig = json.RawMessage(`
							{
								"cloud": "aws-eu-west-1",
								"catalog": {
									"services": [
										{
											"name": "elasticsearch",
											"plans": [{"aiven_plan": "plan-a"}]
										}
									]
								}
							}
						`)
				_, err := provider.DecodeConfig(rawConfig)
				Expect(err).To(MatchError("Config error: every elasticsearch plan must specify an `elasticsearch_version`"))
			})
		})

		Context("when the service is InfluxDB", func() {
			It("does not care about the Elasticsearch version", func() {
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
