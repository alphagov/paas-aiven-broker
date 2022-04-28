package influxdb

import (
	"net/http"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("InfluxDB Client", func() {
	var (
		client *Client

		influxDBEndpoint = "http://localhost:8086"
	)

	BeforeEach(func() {
		client = New("http://localhost:8086", nil)
	})

	It("should create New() client", func() {
		Expect(client.URI).To(Equal(influxDBEndpoint))
	})

	Context("when InfluxDB has a garbage URI", func() {
		It("should fail when calling Ping", func() {
			client.URI = "%gh&%ij"

			_, err := client.Ping()

			Expect(err).To(HaveOccurred())
		})

		It("should fail when calling Version", func() {
			client.URI = "%gh&%ij"

			_, err := client.Version()

			Expect(err).To(HaveOccurred())
		})

		It("should fail when calling Build", func() {
			client.URI = "%gh&%ij"

			_, err := client.Build()

			Expect(err).To(HaveOccurred())
		})
	})

	Context("when making requests", func() {
		BeforeEach(func() {
			httpmock.Activate()
		})

		AfterEach(func() {
			httpmock.DeactivateAndReset()
		})

		Context("when InfluxDB is available", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder(
					"HEAD",
					influxDBEndpoint+"/ping",
					func(req *http.Request) (*http.Response, error) {
						res := &http.Response{
							StatusCode: 204,
							Header: map[string][]string{
								InfluxDBVersionHeader: []string{"v1.7.0"},
								InfluxDBBuildHeader:   []string{"OSS"},
							},
						}
						return res, nil
					},
				)
			})

			It("should return version and build when calling Ping()", func() {
				resp, err := client.Ping()

				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Version).To(Equal("v1.7.0"))
				Expect(resp.Build).To(Equal("OSS"))
			})

			It("should retrieve the version when calling Version()", func() {
				version, err := client.Version()

				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("v1.7.0"))
			})

			It("should retrieve the build when calling Build()", func() {
				build, err := client.Build()

				Expect(err).NotTo(HaveOccurred())
				Expect(build).To(Equal("OSS"))
			})
		})

		Context("when InfluxDB is not available", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder(
					"HEAD",
					influxDBEndpoint+"/ping",
					httpmock.NewStringResponder(200, ``),
				)
			})

			It("should fail when calling Ping()", func() {
				_, err := client.Ping()

				Expect(err).To(MatchError(And(
					ContainSubstring("Expected HTTP 204"),
					ContainSubstring("received HTTP 200"),
				)))
			})

			It("should fail when calling Version()", func() {
				_, err := client.Version()

				Expect(err).To(MatchError(And(
					ContainSubstring("Expected HTTP 204"),
					ContainSubstring("received HTTP 200"),
				)))
			})

			It("should fail when calling Build()", func() {
				_, err := client.Build()

				Expect(err).To(MatchError(And(
					ContainSubstring("Expected HTTP 204"),
					ContainSubstring("received HTTP 200"),
				)))
			})
		})
	})
})
