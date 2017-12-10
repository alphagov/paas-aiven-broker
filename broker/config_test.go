package broker_test

import (
	"strings"

	"code.cloudfoundry.org/lager"
	. "github.com/henrytk/universal-service-broker/broker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	var configSource string

	It("requires a basic auth username", func() {
		configSource = `
			{
				"basic_auth_password":"1234",
				"port": "8080",
				"log_level": "debug",
				"catalog": {}
			}
		`
		_, err := NewConfig(strings.NewReader(configSource))
		Expect(err).To(MatchError("Config error: basic auth username required"))
	})

	It("requires a basic auth password", func() {
		configSource = `
			{
				"basic_auth_username":"username",
				"port": "8080",
				"log_level": "debug",
				"catalog": {}
			}
		`
		_, err := NewConfig(strings.NewReader(configSource))
		Expect(err).To(MatchError("Config error: basic auth password required"))
	})

	It("helps convert log level string values into lager.LogLevel values", func() {
		configSource = `
			{
				"basic_auth_username":"username",
				"basic_auth_password":"1234",
				"port": "8080",
				"log_level": "debug",
				"catalog": {}
			}
		`
		config, err := NewConfig(strings.NewReader(configSource))
		Expect(err).NotTo(HaveOccurred())

		lagerLogLevel := config.API.LagerLogLevel()
		Expect(lagerLogLevel).To(Equal(lager.DEBUG))
	})

	Describe("default values", func() {
		It("sets a default port", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"log_level": "debug",
					"catalog": {}
				}
			`
			config, err := NewConfig(strings.NewReader(configSource))
			Expect(err).NotTo(HaveOccurred())
			Expect(config.API.Port).To(Equal(DefaultPort))
		})

		It("sets a default log_level", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"port": "8080",
					"catalog": {}
				}
			`
			config, err := NewConfig(strings.NewReader(configSource))
			Expect(err).NotTo(HaveOccurred())
			Expect(config.API.LogLevel).To(Equal(DefaultLogLevel))
		})
	})
})
