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
	"github.com/alphagov/paas-aiven-broker/client/elastic"
	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/pivotal-cf/brokerapi"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	asyncAllowed                 = true
	defaultTimeout time.Duration = 15 * time.Minute

	elasticsearchServiceGUID     = "uuid-elasticsearch-service"
	elasticsearchInitialPlanGUID = "uuid-basic-elasticsearch-6"
	elasticsearchUpgradePlanGUID = "uuid-supra-elasticsearch-6"

	influxDBServiceGUID = "uuid-influxdb-service"
	influxDBPlanGUID    = "uuid-basic-influxdb-1"
)

type BindingResponse struct {
	Credentials map[string]interface{} `json:"credentials"`
}

var _ = Describe("Broker", func() {
	configJSON := fmt.Sprintf(`{
		"catalog": {
			"services": [{
				"id": "%s",
				"name": "elasticsearch",
				"plan_updateable": true,
				"plans": [{
					"id": "%s",
					"name": "basic-6",
					"aiven_plan": "startup-4",
					"elasticsearch_version": "6"
				}, {
					"id": "%s",
					"name": "supra-6",
					"aiven_plan": "startup-8",
					"elasticsearch_version": "6"
				}]
			}, {
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
		elasticsearchServiceGUID,
		elasticsearchInitialPlanGUID, elasticsearchUpgradePlanGUID,

		influxDBServiceGUID,
		influxDBPlanGUID,
	)

	var (
		instanceID   string
		bindingID    string
		brokerTester brokertesting.BrokerTester
	)

	BeforeEach(func() {
		instanceID = uuid.NewV4().String()
		bindingID = uuid.NewV4().String()

		By("initializing")
		brokerConfig, err := broker.NewConfig(strings.NewReader(configJSON))
		Expect(err).ToNot(HaveOccurred())

		aivenProvider, err := provider.New(brokerConfig.Provider)
		Expect(err).ToNot(HaveOccurred())

		logger := lager.NewLogger("AivenServiceBroker")
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, brokerConfig.API.LagerLogLevel))
		aivenBroker := broker.New(brokerConfig, aivenProvider, logger)

		brokerServer := broker.NewAPI(aivenBroker, logger, brokerConfig)

		brokerTester = brokertesting.New(brokerapi.BrokerCredentials{
			Username: brokerConfig.API.BasicAuthUsername,
			Password: brokerConfig.API.BasicAuthPassword,
		}, brokerServer)
	})

	Context("Elasticsearch", func() {
		const (
			putData = `{"user":"kimchy","post_date":"2009-11-15T14:12:12","message":"trying out Elasticsearch"}`
		)

		AfterEach(func() {
			// Ensure the instance gets cleaned up on test failures
			_ = brokerTester.Deprovision(
				instanceID,
				elasticsearchServiceGUID,
				elasticsearchInitialPlanGUID,
				asyncAllowed,
			)
		})

		It("should manage the lifecycle of an Elasticsearch cluster", func() {
			egressIP := os.Getenv("EGRESS_IP")
			Expect(egressIP).ToNot(BeEmpty())

			os.Setenv("IP_WHITELIST", egressIP)
			defer os.Unsetenv("IP_WHITELIST")

			By("Provisioning")
			res := brokerTester.Provision(instanceID, brokertesting.RequestBody{
				ServiceID: elasticsearchServiceGUID,
				PlanID:    elasticsearchInitialPlanGUID,
			}, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, "", brokerapi.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Last operation succeeded",
			})

			By("Binding")
			res = brokerTester.Bind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID: elasticsearchServiceGUID,
				PlanID:    elasticsearchInitialPlanGUID,
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

			elasticsearchClient := elastic.New(parsedResponse.Credentials["uri"].(string), nil)

			By("ensuring credentials allow writing data")
			putURI := elasticsearchClient.URI + "/twitter/tweet/1?op_type=create"
			request, err := http.NewRequest("PUT", putURI, strings.NewReader(putData))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")
			resp, err := elasticsearchClient.Do(request)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("ensuring credentials allow reading data")
			getURI := elasticsearchClient.URI + "/twitter/tweet/1"
			get, err := elasticsearchClient.Get(getURI)
			Expect(err).NotTo(HaveOccurred())
			defer get.Body.Close()
			Expect(get.StatusCode).To(Equal(http.StatusOK))
			body, err := ioutil.ReadAll(get.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring(putData))

			By("polling for backup completion before updating")
			pollForBackupCompletion(instanceID)

			By("updating")
			res = brokerTester.Update(instanceID, brokertesting.RequestBody{
				ServiceID: elasticsearchServiceGUID,
				PlanID:    elasticsearchUpgradePlanGUID,
			}, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, "", brokerapi.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Last operation succeeded",
			})

			By("checking the version has actually been updated")
			version, err := elasticsearchClient.Version()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(HavePrefix("6."))

			By("Unbinding")
			res = brokerTester.Unbind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID: elasticsearchServiceGUID,
				PlanID:    elasticsearchUpgradePlanGUID,
			})
			Expect(res.Code).To(Equal(http.StatusOK))

			By("Deprovisioning")
			res = brokerTester.Deprovision(instanceID, elasticsearchServiceGUID, elasticsearchUpgradePlanGUID, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusOK))

			deprovisionResponse := brokerapi.DeprovisionResponse{}
			err = json.Unmarshal(res.Body.Bytes(), &deprovisionResponse)
			Expect(err).NotTo(HaveOccurred())

			By("Returning a 410 response when trying to delete a non-existent instance")
			res = brokerTester.Deprovision(instanceID, elasticsearchServiceGUID, elasticsearchUpgradePlanGUID, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusGone))
		})

		// 99% of this IP whitelisting test is stolen from the lifecycle mgmt test, below.
		// Refactor opportunity!
		It("should enforce IP whitelisting if configured to do so", func() {
			os.Setenv("IP_WHITELIST", "8.8.8.8")
			defer os.Unsetenv("IP_WHITELIST")

			By("Provisioning")

			res := brokerTester.Provision(instanceID, brokertesting.RequestBody{
				ServiceID: elasticsearchServiceGUID,
				PlanID:    elasticsearchInitialPlanGUID,
			}, asyncAllowed)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			By("Polling for success")
			pollForCompletion(brokerTester, instanceID, "", brokerapi.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "Last operation succeeded",
			})

			By("Binding")
			res = brokerTester.Bind(instanceID, bindingID, brokertesting.RequestBody{
				ServiceID: elasticsearchServiceGUID,
				PlanID:    elasticsearchInitialPlanGUID,
			})
			Expect(res.Code).To(Equal(http.StatusCreated))

			parsedResponse := BindingResponse{}
			err := json.NewDecoder(res.Body).Decode(&parsedResponse)
			Expect(err).ToNot(HaveOccurred())

			elasticsearchClient := elastic.New(parsedResponse.Credentials["uri"].(string), nil)

			By("Ensuring we can't reach the provisioned service")
			getURI := elasticsearchClient.URI + "/"
			_, err = elasticsearchClient.Get(getURI)
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
				brokerapi.LastOperationResponse{
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

			By("Deprovisioning")
			res = brokerTester.Deprovision(
				instanceID,
				influxDBServiceGUID,
				influxDBPlanGUID,
				asyncAllowed,
			)
			Expect(res.Code).To(Equal(http.StatusOK))

			deprovisionResponse := brokerapi.DeprovisionResponse{}
			err = json.Unmarshal(res.Body.Bytes(), &deprovisionResponse)
			Expect(err).NotTo(HaveOccurred())

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

func pollForCompletion(bt brokertesting.BrokerTester, instanceID, operationData string, expectedResponse brokerapi.LastOperationResponse) {
	Eventually(
		func() brokerapi.LastOperationResponse {
			lastOperationResponse := brokerapi.LastOperationResponse{}
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

func pollForBackupCompletion(instanceID string) {
	Eventually(
		func() bool {
			type getServiceResponse struct {
				Service struct {
					Backups []interface{} `json:"backups"`
				} `json:"service"`
			}

			serviceName := os.Getenv("SERVICE_NAME_PREFIX") + "-" + instanceID
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
		5*time.Minute,
		30*time.Second,
	).Should(BeTrue())
}
