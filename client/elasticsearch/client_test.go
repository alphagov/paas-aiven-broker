package elasticsearch

import (
	"bytes"
	"io"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

var _ = Describe("ElasticSearch Client", func() {
	var (
		client *Client
	)

	BeforeEach(func() {
		client = New("http://localhost:9200", nil)
	})

	It("should create New() client", func() {
		Expect(client.URI).To(Equal("http://localhost:9200"))
	})

	It("should readBody() successfully", func() {
		body := `{"version":{"number":"5.0.0"}}`

		resp, err := client.readBody(nopCloser{bytes.NewBufferString(body)})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Version.Number).To(Equal("5.0.0"))
	})

	It("should fail to readBody() due to json syntax error", func() {
		body := `not found`

		resp, err := client.readBody(nopCloser{bytes.NewBufferString(body)})
		Expect(err).To(HaveOccurred())
		Expect(resp).To(BeNil())
	})

	Context("making requests", func() {
		BeforeEach(func() {
			httpmock.Activate()
		})

		AfterEach(func() {
			httpmock.DeactivateAndReset()
		})

		It("should be able to Ping() the host", func() {
			httpmock.RegisterResponder("GET", "http://localhost:9200",
				httpmock.NewStringResponder(200, `{"version":{"number":"5.0.0"}}`))

			resp, err := client.Ping()
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Version.Number).To(Equal("5.0.0"))
		})

		It("should fail to Ping() the host due to 500", func() {
			httpmock.RegisterResponder("GET", "http://localhost:9200",
				httpmock.NewStringResponder(500, `{"error":"test"}`))

			resp, err := client.Ping()
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})

		It("should fail to Ping() the host due to invalid uri", func() {
			client.URI = "%gh&%ij"

			resp, err := client.Ping()
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})

		It("should be able to get Version() from ElasticSearch", func() {
			httpmock.RegisterResponder("GET", "http://localhost:9200",
				httpmock.NewStringResponder(200, `{"version":{"number":"5.0.0"}}`))

			version, err := client.Version()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal("5.0.0"))
		})

		It("should fail to get Version() due to elasticsearch not providing one", func() {
			httpmock.RegisterResponder("GET", "http://localhost:9200",
				httpmock.NewStringResponder(200, `{"version":{"number":""}}`))

			version, err := client.Version()
			Expect(err).To(HaveOccurred())
			Expect(version).NotTo(Equal("5.0.0"))
		})
	})
})
