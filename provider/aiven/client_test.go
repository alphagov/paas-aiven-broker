package aiven_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			expectedUpdateTime := "2018-06-21T10:01:05.000040+00:00"

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"service": {"state": "RUNNING", "update_time": "%s"}}`, expectedUpdateTime)),
			))

			actualState, updateTime, err := aivenClient.GetServiceStatus(getServiceInput)
			parsedTime, _ := time.Parse(time.RFC3339Nano, expectedUpdateTime)

			Expect(err).ToNot(HaveOccurred())
			Expect(actualState).To(Equal(aiven.Running))
			Expect(updateTime).To(Equal(parsedTime))
		})

		It("returns an error if the state is missing", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{"service": {"update_time": "2018-06-21T10:01:05.000040+00:00"}}`),
			))

			actualState, _, err := aivenClient.GetServiceStatus(getServiceInput)

			Expect(err).To(MatchError("Error getting service status: no state found in response JSON"))
			Expect(actualState).To(Equal(aiven.ServiceStatus("")))
		})

		It("returns an error if the update time is missing", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{"service": {"state": "RUNNING"}}`),
			))

			actualState, updateTime, err := aivenClient.GetServiceStatus(getServiceInput)

			Expect(err).To(MatchError("Error getting service status: no update_time found in response JSON"))
			Expect(actualState).To(Equal(aiven.ServiceStatus("")))
			Expect(updateTime).To(Equal(time.Time{}))
		})

		It("returns an error if aiven 404s", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			actualState, _, err := aivenClient.GetServiceStatus(getServiceInput)

			Expect(err).To(MatchError("Error getting service: 404 status code returned from Aiven"))
			Expect(actualState).To(Equal(aiven.ServiceStatus("")))
		})
	})

	Describe("GetServiceConnectionDetails", func() {
		It("should return the service state", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{"service": {"service_uri_params": {"host": "example.com", "port": "23362"}}}`),
			))

			host, port, err := aivenClient.GetServiceConnectionDetails(getServiceInput)

			Expect(err).ToNot(HaveOccurred())
			Expect(host).To(Equal("example.com"))
			Expect(port).To(Equal("23362"))
		})

		It("returns an error if the connection details are missing", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{"service": {"nonsense": "foo"}}`),
			))

			host, port, err := aivenClient.GetServiceConnectionDetails(getServiceInput)

			Expect(err).To(MatchError("Error getting service connection details: no connection details found in response JSON"))
			Expect(host).To(Equal(""))
			Expect(port).To(Equal(""))
		})

		It("returns an error if aiven 404s", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			host, port, err := aivenClient.GetServiceConnectionDetails(getServiceInput)

			Expect(err).To(MatchError("Error getting service: 404 status code returned from Aiven"))
			Expect(host).To(Equal(""))
			Expect(port).To(Equal(""))
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

	Describe("CreateServiceUser", func() {
		It("should make a valid request", func() {
			createServiceUserInput := &aiven.CreateServiceUserInput{
				ServiceName: "my-service",
				Username:    "user",
			}
			expectedBody := []byte(`{"username":"user"}`)
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/v1beta/project/my-project/service/my-service/user"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.VerifyBody(expectedBody),
				ghttp.RespondWith(http.StatusOK, `{"message":"created","user":{"password":"superdupersecret","type":"normal","username":"user"}}`),
			))

			actualPassword, err := aivenClient.CreateServiceUser(createServiceUserInput)

			Expect(err).ToNot(HaveOccurred())
			Expect(actualPassword).To(Equal("superdupersecret"))
		})

		It("returns an error if the http request fails", func() {
			createServiceUserInput := &aiven.CreateServiceUserInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusForbidden, "{}"),
			))

			actualPassword, err := aivenClient.CreateServiceUser(createServiceUserInput)

			Expect(err).To(MatchError("Error creating service user: 403 status code returned from Aiven"))
			Expect(actualPassword).To(Equal(""))
		})

		It("returns an error if the password is empty", func() {
			createServiceUserInput := &aiven.CreateServiceUserInput{
				ServiceName: "my-service",
				Username:    "user",
			}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusOK, `{"this will not":"unmarshal into the password field"}`),
			))

			actualPassword, err := aivenClient.CreateServiceUser(createServiceUserInput)

			Expect(err).To(MatchError("Error creating service user: password was empty"))
			Expect(actualPassword).To(Equal(""))
		})
	})

	Describe("DeleteServiceUser", func() {
		It("should make a valid request", func() {
			deleteServiceUserInput := &aiven.DeleteServiceUserInput{
				ServiceName: "my-service",
				Username:    "my-user",
			}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("DELETE", "/v1beta/project/my-project/service/my-service/user/my-user"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, "{}"),
			))

			actualResponse, err := aivenClient.DeleteServiceUser(deleteServiceUserInput)

			Expect(err).ToNot(HaveOccurred())
			Expect(actualResponse).To(Equal("{}"))
		})

		It("returns an error if the http request fails", func() {
			deleteServiceUserInput := &aiven.DeleteServiceUserInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusForbidden, "{}"),
			))

			actualResponse, err := aivenClient.DeleteServiceUser(deleteServiceUserInput)

			Expect(err).To(MatchError("Error deleting service user: 403 status code returned from Aiven"))
			Expect(actualResponse).To(Equal(""))
		})
	})

	Describe("Update Service", func() {
		It("should make a valid request", func() {
			updateServiceInput := &aiven.UpdateServiceInput{
				ServiceName: "my-service",
				Plan:        "new-plan",
			}
			expectedBody := []byte(`{"plan":"new-plan"}`)
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("PUT", "/v1beta/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.VerifyBody(expectedBody),
				ghttp.RespondWith(http.StatusOK, `{}`),
			))

			actualResponse, err := aivenClient.UpdateService(updateServiceInput)

			Expect(err).ToNot(HaveOccurred())
			Expect(actualResponse).To(Equal(`{}`))
		})

		It("returns an error if the http request fails", func() {
			updateServiceInput := &aiven.UpdateServiceInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			actualResponse, err := aivenClient.UpdateService(updateServiceInput)

			Expect(err).To(MatchError("Error updating service: 404 status code returned from Aiven"))
			Expect(actualResponse).To(Equal(""))
		})
	})

})
