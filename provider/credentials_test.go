package provider_test

import (
	"encoding/json"

	"github.com/alphagov/paas-aiven-broker/provider"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Credentials", func() {
	Context("Elasticsearch", func() {
		const (
			username = "hich"
			password = "rickey"

			hostname = "elasticsearch.aiven.io"
			port     = "2702"
		)

		It("should return credentials", func() {
			credentials, err := provider.BuildCredentials(
				"elasticsearch",
				username, password,
				hostname, port,
			)

			Expect(err).NotTo(HaveOccurred())

			jsonCreds, err := json.Marshal(credentials)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(jsonCreds)).To(MatchJSON(
				`{
					"uri":"https://hich:rickey@elasticsearch.aiven.io:2702",
					"hostname":"elasticsearch.aiven.io", "port":"2702",
					"username":"hich","password":"rickey"
				}`,
			))
		})
	})

	Context("OpenSearch", func() {
		const (
			username = "hich"
			password = "rickey"

			hostname = "opensearch.aiven.io"
			port     = "2702"
		)

		It("should return credentials", func() {
			credentials, err := provider.BuildCredentials(
				"opensearch",
				username, password,
				hostname, port,
			)

			Expect(err).NotTo(HaveOccurred())

			jsonCreds, err := json.Marshal(credentials)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(jsonCreds)).To(MatchJSON(
				`{
					"uri":"https://hich:rickey@opensearch.aiven.io:2702",
					"hostname":"opensearch.aiven.io", "port":"2702",
					"username":"hich","password":"rickey"
				}`,
			))
		})
	})

	Context("InfluxDB", func() {
		const (
			username = "hich"
			password = "rickey"

			hostname = "influxdb.aiven.io"
			port     = "2701"
		)

		It("should return credentials", func() {
			credentials, err := provider.BuildCredentials(
				"influxdb",
				username, password,
				hostname, port,
			)

			Expect(err).NotTo(HaveOccurred())

			jsonCreds, err := json.Marshal(credentials)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(jsonCreds)).To(MatchJSON(
				`{
					"uri": "https://hich:rickey@influxdb.aiven.io:2701",
					"hostname": "influxdb.aiven.io",
					"port": "2701",
					"username": "hich",
					"password": "rickey",
					"database": "defaultdb",
					"prometheus": {
						"remote_read": [{
							"url": "https://influxdb.aiven.io:2701/api/v1/prom/read?db=defaultdb",
							"read_recent": true,
							"basic_auth": {
								"username": "hich",
								"password": "rickey"
							}
						}],
						"remote_write": [{
							"url": "https://influxdb.aiven.io:2701/api/v1/prom/write?db=defaultdb",
							"basic_auth": {
								"username": "hich",
								"password": "rickey"
							}
						}]
					}
				}`,
			))
		})
	})

	Context("Invalid service", func() {
		It("should not return credentials", func() {
			_, err := provider.BuildCredentials("unknown-service", "", "", "", "")
			Expect(err).To(MatchError("Unknown service type unknown-service"))
		})
	})
})
