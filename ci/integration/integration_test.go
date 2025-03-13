package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	broker "github.com/alphagov/paas-aiven-broker/broker"
	brokertesting "github.com/alphagov/paas-aiven-broker/broker/testing"
	"github.com/alphagov/paas-aiven-broker/client/opensearch"
	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	asyncAllowed                 = true
	defaultTimeout time.Duration = 15 * time.Minute

	orgGUID   = "test-org-guid"
	spaceGUID = "test-space-guid"

	openSearchServiceGUID     = "uuid-opensearch-service"
	openSearchInitialPlanGUID = "uuid-basic-opensearch-1"
	openSearchUpgradePlanGUID = "uuid-supra-opensearch-1"

	influxDBServiceGUID = "uuid-influxdb-service"
	influxDBPlanGUID    = "uuid-basic-influxdb-1"
)

type BindingResponse struct {
	Credentials map[string]interface{} `json:"credentials"`
}

var _ = Describe("Broker", func() {
	configJSON := fmt.Sprintf(`{
		"name": "aiven-broker",
		"catalog": {
			"services": [{
				"id": "%s",
				"name": "opensearch",
				"plan_updateable": true,
				"plans": [{
					"id": "%s",
					"name": "basic-7",
					"aiven_plan": "startup-4",
					"opensearch_version": "1"
				}, {
					"id": "%s",
					"name": "supra-7",
					"aiven_plan": "startup-8",
					"opensearch_version": "1"
				}]
			},
			{
				"id": "%s",
				"name": "influxdb",
				"plan_updateable": true,
				"plans": [{
					"id": "%s",
					"name": "basic-1",
					"aiven_plan": "startup-4"
				}]
			}]
		}
	}`,
		openSearchServiceGUID,
		openSearchInitialPlanGUID, openSearchUpgradePlanGUID,

		influxDBServiceGUID,
		influxDBPlanGUID,
	)

	var (
		instanceID    string
		bindingID     string
		aivenProvider *provider.AivenProvider
		brokerTester  brokertesting.BrokerTester
	)

	BeforeEach(func() {
		instanceID = uuid.NewV4().String()
		bindingID = uuid.NewV4().String()

		By("initializing")
		brokerConfig, err := broker.NewConfig(strings.NewReader(configJSON))
		Expect(err).ToNot(HaveOccurred())

		logger := lager.NewLogger("AivenServiceBroker")
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, brokerConfig.API.LagerLogLevel))

		aivenProvider, err = provider.New(brokerConfig.Provider, logger)
		Expect(err).ToNot(HaveOccurred())

		aivenBroker := broker.New(brokerConfig, aivenProvider, logger)

		brokerServer := broker.NewAPI(aivenBroker, logger, brokerConfig)

		brokerTester = brokertesting.New(brokerapi.BrokerCredentials{
			Username: brokerConfig.API.BasicAuthUsername,
			Password: brokerConfig.API.BasicAuthPassword,
		}, brokerServer)
	})

	Context("OpenSearch", func() {
		const (
			putData = `{"user":"kimchy","post_date":"2009-11-15T14:12:12","message":"trying out OpenSearch"}`
		)

		AfterEach(func() {
			// Ensure the instance gets cleaned up on test failures
			_ = brokerTester.Deprovision(
				instanceID,
				openSearchServiceGUID,
				openSearchInitialPlanGUID,
				asyncAllowed,
			)
		})

		It("should manage the lifecycle of an OpenSearch cluster", func() {
			egressIP := os.Getenv("EGRESS_IP")
			Expect(egressIP).ToNot(BeEmpty())

			os.Setenv("IP_WHITELIST", egressIP)
			defer os.Unsetenv("IP_WHITELIST")

			By("Provisioning")
			res := brokerTester.Provision(instanceID, brokertesting.RequestBody{
				ServiceID:        openSearchServiceGUID,
				PlanID:           openSearchInitialPlanGUID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			}, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, "", apiresponses.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Last operation succeeded",
			})

			By("Binding")
			res = brokerTester.Bind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID:        openSearchServiceGUID,
				PlanID:           openSearchInitialPlanGUID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			})
			Expect(res.Code).To(Equal(http.StatusCreated))

			parsedResponse := BindingResponse{}
			err := json.NewDecoder(res.Body).Decode(&parsedResponse)
			Expect(err).ToNot(HaveOccurred())

			// Ensure returned credentials follow guidelines in https://docs.cloudfoundry.org/services/binding-credentials.html
			var str string
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("uri", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("hostname", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("port", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("username", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("password", BeAssignableToTypeOf(str)))

			openSearchClient := opensearch.New(parsedResponse.Credentials["uri"].(string), nil)

			By("ensuring credentials allow writing data")
			putURI := openSearchClient.URI + "/twitter/tweet/1?op_type=create"
			request, err := http.NewRequest("PUT", putURI, strings.NewReader(putData))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")
			resp, err := openSearchClient.Do(request)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("ensuring credentials allow reading data")
			getURI := openSearchClient.URI + "/twitter/tweet/1"
			get, err := openSearchClient.Get(getURI)
			Expect(err).NotTo(HaveOccurred())
			defer get.Body.Close()
			Expect(get.StatusCode).To(Equal(http.StatusOK))
			body, err := ioutil.ReadAll(get.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring(putData))

			By("polling for backup completion before updating")
			pollForBackupCompletion(instanceID, aivenProvider)

			By("updating")
			res = brokerTester.Update(instanceID, brokertesting.RequestBody{
				ServiceID:        openSearchServiceGUID,
				PlanID:           openSearchUpgradePlanGUID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			}, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, "", apiresponses.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Last operation succeeded",
			})

			By("checking the version has actually been updated")
			version, err := openSearchClient.Version()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(HavePrefix("1."))

			By("Unbinding")
			res = brokerTester.Unbind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID:        openSearchServiceGUID,
				PlanID:           openSearchUpgradePlanGUID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			})
			Expect(res.Code).To(Equal(http.StatusOK))

			By("Returning a 410 response when trying to unbind a non-existent binding")
			res = brokerTester.Unbind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID:        openSearchServiceGUID,
				PlanID:           openSearchUpgradePlanGUID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			})
			Expect(res.Code).To(Equal(http.StatusGone))

			By("Deprovisioning")
			res = brokerTester.Deprovision(instanceID, openSearchServiceGUID, openSearchUpgradePlanGUID, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))
			deprovisionResponse := apiresponses.DeprovisionResponse{}
			err = json.Unmarshal(res.Body.Bytes(), &deprovisionResponse)
			Expect(err).NotTo(HaveOccurred())

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, deprovisionResponse.OperationData, apiresponses.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Service has been deleted",
			})

			By("Returning a 410 response when trying to delete a non-existent instance")
			res = brokerTester.Deprovision(instanceID, openSearchServiceGUID, openSearchUpgradePlanGUID, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusGone))
		})

		// 99% of this IP whitelisting test is stolen from the lifecycle mgmt test, below.
		// Refactor opportunity!
		It("should enforce IP whitelisting if configured to do so", func() {
			os.Setenv("IP_WHITELIST", "8.8.8.8")
			defer os.Unsetenv("IP_WHITELIST")

			By("Provisioning")

			res := brokerTester.Provision(instanceID, brokertesting.RequestBody{
				ServiceID:        openSearchServiceGUID,
				PlanID:           openSearchInitialPlanGUID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			}, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, "", apiresponses.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Last operation succeeded",
			})

			By("Binding")
			res = brokerTester.Bind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID:        openSearchServiceGUID,
				PlanID:           openSearchInitialPlanGUID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			})
			Expect(res.Code).To(Equal(http.StatusCreated))

			parsedResponse := BindingResponse{}
			err := json.NewDecoder(res.Body).Decode(&parsedResponse)
			Expect(err).ToNot(HaveOccurred())

			openSearchClient := opensearch.New(parsedResponse.Credentials["uri"].(string), nil)

			By("Ensuring we can't reach the provisioned service")
			getURI := openSearchClient.URI + "/"
			_, err = openSearchClient.Get(getURI)
			Expect(err).To(HaveOccurred())

			netErr, ok := err.(net.Error)
			Expect(ok).To(BeTrue())
			Expect(netErr.Timeout()).To(BeTrue())
		})
	})

	Context("InfluxDB", func() {
		AfterEach(func() {
			// Ensure the instance gets cleaned up on test failures
			_ = brokerTester.Deprovision(
				instanceID,
				influxDBServiceGUID,
				influxDBPlanGUID,
				asyncAllowed,
			)
		})

		It("should manage the lifecycle of an InfluxDB", func() {
			egressIP := os.Getenv("EGRESS_IP")
			Expect(egressIP).ToNot(BeEmpty())

			os.Setenv("IP_WHITELIST", egressIP)
			defer os.Unsetenv("IP_WHITELIST")

			By("Provisioning")
			res := brokerTester.Provision(instanceID, brokertesting.RequestBody{
				ServiceID: influxDBServiceGUID,
				PlanID:    influxDBPlanGUID,
			}, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			By("Polling for success")
			pollForCompletion(
				brokerTester,
				instanceID, "",
				apiresponses.LastOperationResponse{
					State:       brokerapi.Succeeded,
					Description: "Last operation succeeded",
				},
			)

			By("Binding")
			res = brokerTester.Bind(
				instanceID,
				bindingID,
				brokertesting.RequestBody{
					ServiceID: influxDBServiceGUID,
					PlanID:    influxDBPlanGUID,
				},
			)
			Expect(res.Code).To(Equal(http.StatusCreated))

			parsedResponse := BindingResponse{}
			err := json.NewDecoder(res.Body).Decode(&parsedResponse)
			Expect(err).ToNot(HaveOccurred())

			// Ensure returned credentials follow guidelines in https://docs.cloudfoundry.org/services/binding-credentials.html
			var str string
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("uri", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("hostname", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("port", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("username", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("password", BeAssignableToTypeOf(str)))
			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("database", BeAssignableToTypeOf(str)))

			// Ensure returned credentials conform to Prometheus config format
			// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_read
			// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write
			// remote_read:
			//   - url: https://....
			//     basic_auth:
			//       username: hello
			//       password: world
			// remote_write:
			//   - url: https://....
			//     basic_auth:
			//       username: hello
			//       password: world
			Expect(parsedResponse.Credentials).To(
				HaveKeyWithValue("prometheus", Not(BeNil())),
			)

			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("prometheus",
				HaveKeyWithValue("remote_read", ConsistOf(SatisfyAll(
					HaveKeyWithValue(
						"url", ContainSubstring("api/v1/prom/read?db=defaultdb"),
					),
					HaveKeyWithValue("basic_auth", SatisfyAll(
						HaveKeyWithValue("username", BeAssignableToTypeOf(str)),
						HaveKeyWithValue("password", BeAssignableToTypeOf(str)),
					)),
				))),
			),
				"Binding creds should have remote_read Prometheus configuration",
			)

			Expect(parsedResponse.Credentials).To(HaveKeyWithValue("prometheus",
				HaveKeyWithValue("remote_write", ConsistOf(SatisfyAll(
					HaveKeyWithValue(
						"url", ContainSubstring("api/v1/prom/write?db=defaultdb"),
					),
					HaveKeyWithValue("basic_auth", SatisfyAll(
						HaveKeyWithValue("username", BeAssignableToTypeOf(str)),
						HaveKeyWithValue("password", BeAssignableToTypeOf(str)),
					)),
				))),
			),
				"Binding creds should have remote_write Prometheus configuration",
			)

			By("Unbinding")
			res = brokerTester.Unbind(
				instanceID,
				bindingID,
				brokertesting.RequestBody{
					ServiceID: influxDBServiceGUID,
					PlanID:    influxDBPlanGUID,
				},
			)
			Expect(res.Code).To(Equal(http.StatusOK))

			By("Returning a 410 response when trying to unbind a non-existent binding")
			res = brokerTester.Unbind(
				instanceID,
				bindingID,
				brokertesting.RequestBody{
					ServiceID: influxDBServiceGUID,
					PlanID:    influxDBPlanGUID,
				},
			)
			Expect(res.Code).To(Equal(http.StatusGone))

			By("Deprovisioning")
			res = brokerTester.Deprovision(
				instanceID,
				influxDBServiceGUID,
				influxDBPlanGUID,
				asyncAllowed,
			)
			Expect(res.Code).To(Equal(http.StatusAccepted))
			deprovisionResponse := apiresponses.DeprovisionResponse{}
			err = json.Unmarshal(res.Body.Bytes(), &deprovisionResponse)
			Expect(err).NotTo(HaveOccurred())

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, deprovisionResponse.OperationData, apiresponses.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Service has been deleted",
			})

			By("Returning a 410 response when trying to delete a non-existent instance")
			res = brokerTester.Deprovision(
				instanceID,
				influxDBServiceGUID,
				influxDBPlanGUID,
				asyncAllowed,
			)
			Expect(res.Code).To(Equal(http.StatusGone))
		})
	})
})

func pollForCompletion(bt brokertesting.BrokerTester, instanceID, operationData string, expectedResponse apiresponses.LastOperationResponse) {
	Eventually(
		func() apiresponses.LastOperationResponse {
			lastOperationResponse := apiresponses.LastOperationResponse{}
			res := bt.LastOperation(instanceID, "", "", operationData)
			if res.Code != http.StatusOK {
				return lastOperationResponse
			}
			_ = json.Unmarshal(res.Body.Bytes(), &lastOperationResponse)
			return lastOperationResponse
		},
		defaultTimeout,
		30*time.Second,
	).Should(Equal(expectedResponse))
}

func pollForBackupCompletion(instanceID string, provider *provider.AivenProvider) {
	Eventually(
		func() bool {
			type getServiceResponse struct {
				Service struct {
					Backups []interface{} `json:"backups"`
				} `json:"service"`
			}
			serviceName := provider.BuildServiceName(instanceID)
			req, err := http.NewRequest("GET", fmt.Sprintf(
				"https://api.aiven.io/v1beta/project/%s/service/%s",
				os.Getenv("AIVEN_PROJECT"),
				serviceName,
			), nil)
			if err != nil {
				return false
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("aivenv1 %s", os.Getenv("AIVEN_API_TOKEN")))

			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return false
			}
			defer res.Body.Close()

			service := &getServiceResponse{}
			err = json.NewDecoder(res.Body).Decode(&service)
			if err != nil {
				return false
			}

			if len(service.Service.Backups) > 0 {
				return true
			}
			return false
		},
		10*time.Minute,
		30*time.Second,
	).Should(BeTrue())
}
