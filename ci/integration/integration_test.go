package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	ASYNC_ALLOWED                 = true
	DEFAULT_TIMEOUT time.Duration = 15 * time.Minute
	putData                       = "{\"user\" : \"kimchy\",\"post_date\" : \"2009-11-15T14:12:12\",\"message\" : \"trying out Elasticsearch\"}"
)

type BindingResponse struct {
	Credentials provider.Credentials `json:"credentials"`
}

var _ = Describe("Broker", func() {

	var (
		instanceID string
		bindingID  string
	)

	BeforeEach(func() {
		instanceID = uuid.NewV4().String()
		bindingID = uuid.NewV4().String()
	})

	It("should manage the lifecycle of an Elasticsearch cluster", func() {
		By("initializing")

		configJSON := `{
			"catalog": {
				"services": [{
					"id": "uuid-1",
					"plan_updateable": true,
					"plans": [{
						"id": "uuid-2",
						"name": "basic",
						"aiven_plan": "startup-4",
						"elasticsearch_version": "6"
					}, {
						"id": "uuid-3",
						"name": "supra",
						"aiven_plan": "startup-8",
						"elasticsearch_version": "6"
					}]
				}]
			}
		}`

		brokerConfig, err := broker.NewConfig(bytes.NewBuffer([]byte(configJSON)))
		Expect(err).ToNot(HaveOccurred())

		aivenProvider, err := provider.New(brokerConfig.Provider)
		Expect(err).ToNot(HaveOccurred())

		logger := lager.NewLogger("AivenServiceBroker")
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, brokerConfig.API.LagerLogLevel))
		aivenBroker := broker.New(brokerConfig, aivenProvider, logger)

		brokerServer := broker.NewAPI(aivenBroker, logger, brokerConfig)

		brokerTester := brokertesting.New(brokerapi.BrokerCredentials{
			Username: "foo",
			Password: "bar",
		}, brokerServer)

		By("Provisioning")

		res := brokerTester.Provision(instanceID, brokertesting.RequestBody{
			ServiceID: "uuid-1",
			PlanID:    "uuid-2",
		}, ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusAccepted))

		By("Polling for success")
		pollForCompletion(brokerTester, instanceID, "", brokerapi.LastOperationResponse{
			State:       brokerapi.Succeeded,
			Description: "Last operation succeeded",
		})

		By("Binding")
		res = brokerTester.Bind(instanceID, bindingID, brokertesting.RequestBody{
			ServiceID: "uuid-1",
			PlanID:    "uuid-2",
		})
		Expect(res.Code).To(Equal(http.StatusCreated))

		parsedResponse := BindingResponse{}

		err = json.NewDecoder(res.Body).Decode(&parsedResponse)
		Expect(err).ToNot(HaveOccurred())

		httpClient := http.Client{}
		elasticsearchClient, _ := elastic.New(parsedResponse.Credentials.Uri, &httpClient)

		By("Working around Aiven's slow DNS creation")
		pollForAvailability(elasticsearchClient)

		By("ensuring credentials allow writing data")
		putURI := parsedResponse.Credentials.Uri + "/twitter/tweet/1?op_type=create"
		request, err := http.NewRequest("PUT", putURI, strings.NewReader(putData))
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("Content-Type", "application/json")
		resp, err := elasticsearchClient.Do(request)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))

		By("ensuring credentials allow reading data")
		getURI := parsedResponse.Credentials.Uri + "/twitter/tweet/1"
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
			ServiceID: "uuid-1",
			PlanID:    "uuid-3",
		}, ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusAccepted))

		By("Polling for success")
		pollForCompletion(brokerTester, instanceID, "", brokerapi.LastOperationResponse{
			State:       brokerapi.Succeeded,
			Description: "Last operation succeeded",
		})

		By("Unbinding")
		res = brokerTester.Unbind(instanceID, bindingID, brokertesting.RequestBody{
			ServiceID: "uuid-1",
			PlanID:    "uuid-2",
		})
		Expect(res.Code).To(Equal(http.StatusOK))

		By("Deprovisioning")
		res = brokerTester.Deprovision(instanceID, "uuid-1", "uuid-2", ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusOK))

		deprovisionResponse := brokerapi.DeprovisionResponse{}
		err = json.Unmarshal(res.Body.Bytes(), &deprovisionResponse)
		Expect(err).NotTo(HaveOccurred())
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
		DEFAULT_TIMEOUT,
		30*time.Second,
	).Should(Equal(expectedResponse))
}

func pollForAvailability(esClient *elastic.Client) {
	Eventually(
		func() string {
			version, _ := esClient.Version()
			return version
		},
		5*time.Minute,
		30*time.Second,
	).ShouldNot(BeEmpty())
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
