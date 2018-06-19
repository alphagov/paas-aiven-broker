package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	broker "github.com/alphagov/paas-aiven-broker/broker"
	brokertesting "github.com/alphagov/paas-aiven-broker/broker/testing"
	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/pivotal-cf/brokerapi"
	uuid "github.com/satori/go.uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	ASYNC_ALLOWED                 = true
	DEFAULT_TIMEOUT time.Duration = 15 * time.Minute
)

var _ = Describe("Provider", func() {

	var (
		instanceID string
	)

	BeforeEach(func() {
		instanceID = uuid.NewV4().String()
	})

	It("should provision an Elasticsearch service", func() {
		configJSON := `{
			"basic_auth_username": "foo",
			"basic_auth_password": "bar",
			"cloud": "aws-eu-west-1",
			"catalog": {
				"services": [{
					"id": "uuid-1",
					"plans": [{
						"id": "uuid-2",
						"name": "basic",
						"aiven_plan": "startup-4",
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

		res := brokerTester.Provision(instanceID, brokertesting.RequestBody{
			ServiceID: "uuid-1",
			PlanID:    "uuid-2",
		}, ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusAccepted))

		pollForCompletion(brokerTester, instanceID, "", brokerapi.LastOperationResponse{
			State:       brokerapi.Succeeded,
			Description: "Last operation succeeded",
		})

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
