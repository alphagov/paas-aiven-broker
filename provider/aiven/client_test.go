package aiven_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/alphagov/paas-aiven-broker/provider/aiven"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client", func() {
	var (
		aivenAPI    *ghttp.Server
		aivenClient *aiven.HttpClient
		logger      lager.Logger
	)

	BeforeEach(func() {
		logger = lager.NewLogger("client")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		aivenAPI = ghttp.NewServer()
		aivenClient = aiven.NewHttpClient(aivenAPI.URL(), "token", "my-project", logger)
	})

	AfterEach(func() {
		aivenAPI.Close()
	})

	Describe("CreateService", func() {
		It("should make a valid request", func() {
			userConfig := aiven.UserConfig{}
			userConfig.OpenSearchVersion = "1"
			userConfig.IPFilter = []string{"1.2.3.4"}

			createServiceInput := &aiven.CreateServiceInput{
				Cloud:       "cloud",
				Plan:        "plan",
				ServiceName: "name",
				ServiceType: "type",
				UserConfig:  userConfig,
				Tags:        aiven.ServiceTags{},
			}
			expectedBody, _ := json.Marshal(createServiceInput)
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/v1/project/my-project/service"),
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

			Expect(err).To(MatchError("Error creating service: 404 status code returned from Aiven: '{}'"))
			Expect(actualService).To(Equal(""))
		})
	})

	Describe("ForkService", func() {
		It("should make a valid request", func() {
			userConfig := aiven.UserConfig{}
			userConfig.OpenSearchVersion = "1"
			userConfig.IPFilter = []string{"1.2.3.4"}
			userConfig.ForkProject = "my-project"
			userConfig.BackupServiceName = "some-instance"
			userConfig.BackupName = "some-backup"

			createServiceInput := &aiven.CreateServiceInput{
				Cloud:       "cloud",
				Plan:        "plan",
				ServiceName: "name",
				ServiceType: "type",
				UserConfig:  userConfig,
				Tags:        aiven.ServiceTags{},
			}
			expectedBody, _ := json.Marshal(createServiceInput)
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/v1/project/my-project/service"),
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

			Expect(err).To(MatchError("Error creating service: 404 status code returned from Aiven: '{}'"))
			Expect(actualService).To(Equal(""))
		})
	})

	Describe("GetService", func() {
		It("should return the service", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			expectedUpdateTime := "2018-06-21T10:01:05.000040+00:00"

			expectedBackupTimeS := "2021-10-24T16:11:03.171000Z"
			expectedBackupTime, err := time.Parse(time.RFC3339, expectedBackupTimeS)
			Expect(err).ToNot(HaveOccurred())

			expectedBackups := []aiven.ServiceBackup{
				{
					Name: "testbackup",
					Time: expectedBackupTime,
					Size: 123,
				},
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, fmt.Sprintf(
					`{"service": { "backups": [{"backup_name": "%s", "backup_time": "%s","data_size": %d}], "service_type": "pg", "state": "RUNNING", "update_time": "%s"}}`,
					expectedBackups[0].Name,
					expectedBackupTimeS,
					expectedBackups[0].Size,
					expectedUpdateTime,
				)),
			))

			service, err := aivenClient.GetService(getServiceInput)
			parsedTime, _ := time.Parse(time.RFC3339Nano, expectedUpdateTime)

			Expect(err).ToNot(HaveOccurred())
			Expect(service.State).To(BeEquivalentTo("RUNNING"))
			Expect(service.ServiceType).To(Equal("pg"))
			Expect(service.UpdateTime).To(Equal(parsedTime))
			Expect(service.Backups).To(Equal(expectedBackups))
		})

		It("returns an error if the state is missing", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{"service": {"service_type": "pg", "update_time": "2018-06-21T10:01:05.000040+00:00"}}`),
			))

			_, err := aivenClient.GetService(getServiceInput)

			Expect(err).To(MatchError("Error getting service: no state found in response JSON"))
		})

		It("returns an error if the service type is missing", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{"service": {"state": "RUNNING", "update_time": "2018-06-21T10:01:05.000040+00:00"}}`),
			))

			_, err := aivenClient.GetService(getServiceInput)

			Expect(err).To(MatchError("Error getting service: no service type found in response JSON"))
		})

		It("returns an error if the update time is missing", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, `{"service": {"service_type": "pg", "state": "RUNNING"}}`),
			))

			_, err := aivenClient.GetService(getServiceInput)

			Expect(err).To(MatchError("Error getting service: no update_time found in response JSON"))
		})

		It("returns an error if aiven 404s", func() {
			getServiceInput := &aiven.GetServiceInput{
				ServiceName: "my-service",
			}

			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/project/my-project/service/my-service"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			_, err := aivenClient.GetService(getServiceInput)

			Expect(err).To(MatchError(aiven.ErrInstanceDoesNotExist))
		})
	})

	Describe("DeleteService", func() {
		It("should make a valid request", func() {
			deleteServiceInput := &aiven.DeleteServiceInput{
				ServiceName: "name",
			}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("DELETE", "/v1/project/my-project/service/name"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("Authorization", "aivenv1 token"),
				ghttp.RespondWith(http.StatusOK, "{}"),
			))

			err := aivenClient.DeleteService(deleteServiceInput)

			Expect(err).ToNot(HaveOccurred())
		})

		It("returns a specific error if a 404 is returned", func() {
			deleteServiceInput := &aiven.DeleteServiceInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusNotFound, "{}"),
			))

			err := aivenClient.DeleteService(deleteServiceInput)

			Expect(err).To(MatchError(aiven.ErrInstanceDoesNotExist))
		})

		It("returns an error if the status code is unexpected", func() {
			deleteServiceInput := &aiven.DeleteServiceInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusTeapot, "{}"),
			))

			err := aivenClient.DeleteService(deleteServiceInput)

			Expect(err).To(MatchError("Error deleting service: 418 status code returned from Aiven: '{}'"))
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
				ghttp.VerifyRequest("POST", "/v1/project/my-project/service/my-service/user"),
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

			Expect(err).To(MatchError("Error creating service user: 403 status code returned from Aiven: '{}'"))
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
				ghttp.VerifyRequest("DELETE", "/v1/project/my-project/service/my-service/user/my-user"),
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

			Expect(err).To(MatchError("Error deleting service user: 403 status code returned from Aiven: '{}'"))
			Expect(actualResponse).To(Equal(""))
		})

		It("returns an error if an unexpected error message is returned", func() {
			deleteServiceUserInput := &aiven.DeleteServiceUserInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusForbidden, `{"message": "this error was not expected"}`),
			))

			actualResponse, err := aivenClient.DeleteServiceUser(deleteServiceUserInput)

			Expect(err).To(MatchError(`Error deleting service user: 403 status code returned from Aiven: '{"message": "this error was not expected"}'`))
			Expect(actualResponse).To(Equal(""))
		})

		It("returns ErrInstanceUserDoesNotExist if an error saying the user does not exist is returned", func() {
			deleteServiceUserInput := &aiven.DeleteServiceUserInput{
				ServiceName: "my-service",
				Username:    "my-deleted-user",
			}
			response := `{"errors":[{"message":"Service user does not exist","status":404}],"message":"Service user does not exist"}`
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusForbidden, response),
			))

			actualResponse, err := aivenClient.DeleteServiceUser(deleteServiceUserInput)

			Expect(err).To(Equal(aiven.ErrInstanceUserDoesNotExist))
			Expect(actualResponse).To(Equal(""))
		})
	})

	Describe("Update Service", func() {
		It("should make a valid request", func() {
			userConfig := aiven.UserConfig{}
			userConfig.OpenSearchVersion = "1"
			userConfig.IPFilter = []string{"1.2.3.4"}

			updateServiceInput := &aiven.UpdateServiceInput{
				ServiceName: "my-service",
				Plan:        "new-plan",
				UserConfig:  userConfig,
			}
			expectedBody, _ := json.Marshal(updateServiceInput)
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("PUT", "/v1/project/my-project/service/my-service"),
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

			Expect(err).To(MatchError("Error updating service: 404 status code returned from Aiven: '{}'"))
			Expect(actualResponse).To(Equal(""))
		})

		It("returns the right error type if the operation is invalid", func() {
			updateServiceInput := &aiven.UpdateServiceInput{}
			aivenAPI.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusBadRequest, `
				{
					"errors" : [
							{
								"message" : "Opensearch major version downgrade is not possible",
								"status" : 400
							}
					],
					"message" : "Opensearch major version downgrade is not possible"
				}
				`),
			))

			actualResponse, err := aivenClient.UpdateService(updateServiceInput)

			Expect(err).To(MatchError(
				aiven.ErrInvalidUpdate{"Invalid Update: Opensearch major version downgrade is not possible"},
			))
			Expect(actualResponse).To(Equal(""))
		})
	})

})
