package aiven_test

import (
	"encoding/json"
	"net/http"

	"github.com/alphagov/paas-aiven-broker/provider/aiven"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client", func() {
	var (
		aivenAPI    *ghttp.Server
		aivenClient *aiven.HttpClient
	)

	BeforeEach(func() {
		aivenAPI = ghttp.NewServer()
		aivenClient = aiven.NewHttpClient(aivenAPI.URL(), "token", "my-project")
	})

	AfterEach(func() {
		aivenAPI.Close()
	})

	Describe("CreateService", func() {
		It("should make a valid request", func() {
			createServiceInput := &aiven.CreateServiceInput{
				Cloud:       "cloud",
				Plan:        "plan",
				ServiceName: "name",
				ServiceType: "type",
			}
			expectedBody, _ := json.Marshal(createServiceInput)
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/v1beta/project/my-project/service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.VerifyBody(expectedBody),
				ghttp.RespondWith(http.StatusOK, "{}"),
			))

			actualService, err := aivenClient.CreateService(createServiceInput)

			Expect(err).ToNot(HaveOccurred())
			Expect(actualService).To(Equal("{}"))
		})

		It("returns an error if the http request fails", func() {
			createServiceInput := &aiven.CreateServiceInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			actualService, err := aivenClient.CreateService(createServiceInput)

			Expect(err).To(MatchError("Error creating service: 404 status code returned from Aiven"))
			Expect(actualService).To(Equal(""))
		})
	})

	Describe("GetServiceStatus", func() {
		It("should return the service state", func() {
			serviceStatusInput := &aiven.GetServiceStatusInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{
					"service": {
						"state": "RUNNING"
					}
				}`),
			))

			actualService, err := aivenClient.GetServiceStatus(serviceStatusInput)

			Expect(err).ToNot(HaveOccurred())
			Expect(actualService).To(Equal(aiven.Running))
		})

		It("returns an error if the http request fails", func() {
			serviceStatusInput := &aiven.GetServiceStatusInput{
				ServiceName: "my-service",
			}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			actualService, err := aivenClient.GetServiceStatus(serviceStatusInput)

			Expect(err).To(MatchError("Error getting service status: 404 status code returned from Aiven"))
			Expect(actualService).To(Equal(aiven.ServiceStatus("")))
		})

		It("returns an error if aiven responds with non-JSON", func() {
			serviceStatusInput := &aiven.GetServiceStatusInput{
				ServiceName: "my-service",
			}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusOK, ""),
			))

			actualService, err := aivenClient.GetServiceStatus(serviceStatusInput)

			Expect(err).To(HaveOccurred())
			Expect(actualService).To(Equal(aiven.ServiceStatus("")))
		})

		It("returns an error if aiven responds with unexpected JSON", func() {
			serviceStatusInput := &aiven.GetServiceStatusInput{
				ServiceName: "my-service",
			}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusOK, `{"foo":"bar"}`),
			))

			actualService, err := aivenClient.GetServiceStatus(serviceStatusInput)

			Expect(err).To(MatchError("Error getting service status: no state found in response JSON"))
			Expect(actualService).To(Equal(aiven.ServiceStatus("")))
		})
	})

	Describe("DeleteService", func() {
		It("should make a valid request", func() {
			deleteServiceInput := &aiven.DeleteServiceInput{
				ServiceName: "name",
			}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("DELETE", "/v1beta/project/my-project/service/name"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, "{}"),
			))

			actualResponse, err := aivenClient.DeleteService(deleteServiceInput)

			Expect(err).ToNot(HaveOccurred())
			Expect(actualResponse).To(Equal("{}"))
		})

		It("returns an error if the http request fails", func() {
			deleteServiceInput := &aiven.DeleteServiceInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			actualResponse, err := aivenClient.DeleteService(deleteServiceInput)

			Expect(err).To(MatchError("Error deleting service: 404 status code returned from Aiven"))
			Expect(actualResponse).To(Equal(""))
		})
	})
})
