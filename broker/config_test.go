package broker_test

import (
	"strings"

	"code.cloudfoundry.org/lager"
	. "github.com/henrytk/universal-service-broker/broker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var (
		err               error
		config            Config
		validConfigSource = `
			{
				"basic_auth_username":"admin",
				"basic_auth_password":"1234",
				"port": "8080",
				"log_level": "debug",
				"catalog": {
					"services": [
						{
							"id": "1",
							"provider_config": {
								"provider_specific_service_config": "blah"
							},
							"plans": [
								{
									"id": "1",
									"provider_specific_plan_config": "blah"
								}
							]
						}
					]
				}
			}
		`
	)

	BeforeEach(func() {
		config, err = NewConfig(strings.NewReader(validConfigSource))
		Expect(err).NotTo(HaveOccurred())
	})

	It("requires a basic auth username", func() {
		config.API.BasicAuthUsername = ""

		err = config.Validate()
		Expect(err).To(MatchError("Config error: basic auth username required"))
	})

	It("requires a basic auth password", func() {
		config.API.BasicAuthPassword = ""

		err = config.Validate()
		Expect(err).To(MatchError("Config error: basic auth password required"))
	})

	It("helps convert log level string values into lager.LogLevel values", func() {
		lagerLogLevel := config.API.LagerLogLevel()
		Expect(lagerLogLevel).To(Equal(lager.DEBUG))
	})

	Describe("default values", func() {
		var (
			configMissingDefaults = `
				{
					"basic_auth_username":"admin",
					"basic_auth_password":"1234",
					"catalog": {
						"services": [
							{
								"id": "1",
								"provider_config": {
									"provider_specific_service_config": "blah"
								},
								"plans": [
									{
										"id": "1",
										"provider_specific_plan_config": "blah"
									}
								]
							}
						]
					}
				}
			`
		)

		BeforeEach(func() {
			config, err = NewConfig(strings.NewReader(configMissingDefaults))
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when missing port", func() {
			It("sets a default value", func() {
				err = config.Validate()
				Expect(err).NotTo(HaveOccurred())
				Expect(config.API.Port).To(Equal(DefaultPort))
			})
		})

		Context("when missing log_level", func() {
			It("sets a default value", func() {
				err = config.Validate()
				Expect(err).NotTo(HaveOccurred())
				Expect(config.API.LogLevel).To(Equal(DefaultLogLevel))
			})
		})
	})
})
