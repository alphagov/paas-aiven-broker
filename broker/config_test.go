package broker_test

import (
	"strings"

	"code.cloudfoundry.org/lager"
	. "github.com/alphagov/paas-aiven-broker/broker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	var configSource string

	Describe("Mandatory fields", func() {
		It("requires a basic auth username", func() {
			configSource = `
				{
					"basic_auth_password":"1234",
					"port": "8080",
					"log_level": "debug",
					"catalog": {"services": [{"name": "service1", "plans": [{"name": "plan1"}]}]}
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
					"catalog": {"services": [{"name": "service1", "plans": [{"name": "plan1"}]}]}
				}
			`
			_, err := NewConfig(strings.NewReader(configSource))
			Expect(err).To(MatchError("Config error: basic auth password required"))
		})

		It("requires a catalog", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"port": "8080",
					"log_level": "debug"
				}
			`
			_, err := NewConfig(strings.NewReader(configSource))
			Expect(err).To(MatchError("Config error: catalog required"))
		})
	})

	Describe("Log levels", func() {

		It("helps convert log level string values into lager.LogLevel values", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"port": "8080",
					"log_level": "debug",
					"catalog": {"services": [{"name": "service1", "plans": [{"name": "plan1"}]}]}
				}
			`
			config, err := NewConfig(strings.NewReader(configSource))
			Expect(err).NotTo(HaveOccurred())

			lagerLogLevel, err := config.API.ConvertLogLevel()
			Expect(err).NotTo(HaveOccurred())
			Expect(lagerLogLevel).To(Equal(lager.DEBUG))
		})

		It("errors if the log level doesn't map to a Lager log level", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"port": "8080",
					"log_level": "debuggery",
					"catalog": {"services": [{"name": "service1", "plans": [{"name": "plan1"}]}]}
				}
			`
			_, err := NewConfig(strings.NewReader(configSource))
			Expect(err).To(MatchError("Config error: log level debuggery does not map to a Lager log level"))
		})
	})

	Describe("Default values", func() {
		It("sets a default port", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"log_level": "debug",
					"catalog": {"services": [{"name": "service1", "plans": [{"name": "plan1"}]}]}
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
					"catalog": {"services": [{"name": "service1", "plans": [{"name": "plan1"}]}]}
				}
			`
			config, err := NewConfig(strings.NewReader(configSource))
			Expect(err).NotTo(HaveOccurred())
			Expect(config.API.LogLevel).To(Equal(DefaultLogLevel))
		})
	})

	Describe("Catalog", func() {
		It("requires at least one service", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"catalog": {"services": []}
				}
			`
			_, err := NewConfig(strings.NewReader(configSource))
			Expect(err).To(MatchError("Config error: at least one service is required"))
		})

		It("requires at least one plan", func() {
			configSource = `
				{
					"basic_auth_username":"username",
					"basic_auth_password":"1234",
					"catalog": {"services": [
						{"name": "service1", "plans": [{"name": "plan1"}]},
						{"name": "service2", "plans": []}
					]}
				}
			`
			_, err := NewConfig(strings.NewReader(configSource))
			Expect(err).To(MatchError("Config error: no plans found for service service2"))
		})
	})
})
