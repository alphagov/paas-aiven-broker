package broker_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/lager"
	. "github.com/alphagov/paas-aiven-broker/broker"
	broker_tester "github.com/alphagov/paas-aiven-broker/broker/testing"
	"github.com/alphagov/paas-aiven-broker/provider/fakes"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker API", func() {
	var (
		instanceID   string
		orgGUID      string
		spaceGUID    string
		service1     string
		plan1        string
		validConfig  Config
		username     string
		password     string
		logger       lager.Logger
		fakeProvider *fakes.FakeServiceProvider
		broker       *Broker
		brokerAPI    http.Handler
		brokerTester broker_tester.BrokerTester
	)

	BeforeEach(func() {
		instanceID = "instanceID"
		orgGUID = "org-guid"
		spaceGUID = "space-guid"
		service1 = "service1"
		plan1 = "plan1"
		validConfig = Config{
			API: API{
				BasicAuthUsername: username,
				BasicAuthPassword: password,
			},
			Catalog: Catalog{brokerapi.CatalogResponse{
				Services: []brokerapi.Service{
					brokerapi.Service{
						ID:            service1,
						Name:          service1,
						PlanUpdatable: true,
						Plans: []brokerapi.ServicePlan{
							brokerapi.ServicePlan{
								ID:   plan1,
								Name: plan1,
							},
						},
					},
				},
			},
			},
		}
		logger = lager.NewLogger("broker-api")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		fakeProvider = &fakes.FakeServiceProvider{}
		broker = New(validConfig, fakeProvider, logger)
		brokerAPI = NewAPI(broker, logger, validConfig)

		brokerTester = broker_tester.New(brokerapi.BrokerCredentials{
			Username: validConfig.API.BasicAuthUsername,
			Password: validConfig.API.BasicAuthPassword,
		}, brokerAPI)
	})

	It("serves a healthcheck endpoint", func() {
		res := brokerTester.Get("/healthcheck", url.Values{})
		Expect(res.Code).To(Equal(http.StatusOK))
	})

	Describe("Services", func() {
		It("serves the catalog", func() {
			res := brokerTester.Services()
			Expect(res.Code).To(Equal(http.StatusOK))

			catalogResponse := brokerapi.CatalogResponse{}
			err := json.Unmarshal(res.Body.Bytes(), &catalogResponse)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(catalogResponse.Services)).To(Equal(1))
			Expect(catalogResponse.Services[0].ID).To(Equal(service1))
			Expect(len(catalogResponse.Services[0].Plans)).To(Equal(1))
			Expect(catalogResponse.Services[0].Plans[0].ID).To(Equal(plan1))
		})
	})

	Describe("Provision", func() {
		It("accepts a provision request", func() {
			fakeProvider.ProvisionReturns("dashboardURL", "operationData", nil)
			res := brokerTester.Provision(
				instanceID,
				broker_tester.RequestBody{
					ServiceID:        service1,
					PlanID:           plan1,
					OrganizationGUID: orgGUID,
					SpaceGUID:        spaceGUID,
				},
				true,
			)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			provisioningResponse := brokerapi.ProvisioningResponse{}
			err := json.Unmarshal(res.Body.Bytes(), &provisioningResponse)
			Expect(err).NotTo(HaveOccurred())

			expectedResponse := brokerapi.ProvisioningResponse{
				DashboardURL:  "dashboardURL",
				OperationData: "operationData",
			}
			Expect(provisioningResponse).To(Equal(expectedResponse))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.ProvisionReturns("", "", errors.New("some provisioning error"))
			res := brokerTester.Provision(
				instanceID,
				broker_tester.RequestBody{
					ServiceID:        service1,
					PlanID:           plan1,
					OrganizationGUID: orgGUID,
					SpaceGUID:        spaceGUID,
				},
				true,
			)
			Expect(res.Code).To(Equal(http.StatusInternalServerError))
		})

		It("rejects requests for synchronous provisioning", func() {
			res := brokerTester.Provision(
				instanceID,
				broker_tester.RequestBody{
					ServiceID:        service1,
					PlanID:           plan1,
					OrganizationGUID: orgGUID,
					SpaceGUID:        spaceGUID,
				},
				false,
			)
			Expect(res.Code).To(Equal(http.StatusUnprocessableEntity))
		})
	})

	Describe("Deprovision", func() {
		It("accepts a deprovision request", func() {
			fakeProvider.DeprovisionReturns("operationData", nil)
			res := brokerTester.Deprovision(instanceID, service1, plan1, true)
			Expect(res.Code).To(Equal(http.StatusOK))

			deprovisionResponse := brokerapi.DeprovisionResponse{}
			err := json.Unmarshal(res.Body.Bytes(), &deprovisionResponse)
			Expect(err).NotTo(HaveOccurred())

			expectedResponse := brokerapi.DeprovisionResponse{}
			Expect(deprovisionResponse).To(Equal(expectedResponse))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.DeprovisionReturns("", errors.New("some deprovisioning error"))
			res := brokerTester.Deprovision(instanceID, service1, plan1, true)
			Expect(res.Code).To(Equal(http.StatusInternalServerError))
		})
		It("responds with HTTP Status Gone if the service instance doesn't exist", func() {
			fakeProvider.DeprovisionReturns("", errors.New("some deprovisioning error"))
			res := brokerTester.Deprovision("missing-service-instance", service1, plan1, true)
			Expect(res.Code).To(Equal(http.StatusGone))
		})
	})

	Describe("Bind", func() {
		var (
			bindingID string
			appGUID   string
		)

		BeforeEach(func() {
			bindingID = "bindingID"
			appGUID = "appGUID"
		})

		It("creates a binding", func() {
			fakeProvider.BindReturns(brokerapi.Binding{Credentials: "secrets"}, nil)
			res := brokerTester.Bind(
				instanceID,
				bindingID,
				broker_tester.RequestBody{
					ServiceID: service1,
					PlanID:    plan1,
					AppGUID:   appGUID,
				},
			)
			Expect(res.Code).To(Equal(http.StatusCreated))

			binding := brokerapi.Binding{}
			err := json.Unmarshal(res.Body.Bytes(), &binding)
			Expect(err).NotTo(HaveOccurred())

			expectedBinding := brokerapi.Binding{
				Credentials: "secrets",
			}
			Expect(binding).To(Equal(expectedBinding))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.BindReturns(brokerapi.Binding{}, errors.New("some binding error"))
			res := brokerTester.Bind(
				instanceID,
				bindingID,
				broker_tester.RequestBody{
					ServiceID: service1,
					PlanID:    plan1,
					AppGUID:   appGUID,
				},
			)
			Expect(res.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("Unbind", func() {
		var bindingID string

		BeforeEach(func() {
			bindingID = "bindingID"
		})

		It("unbinds", func() {
			res := brokerTester.Unbind(
				instanceID,
				bindingID,
				broker_tester.RequestBody{
					ServiceID: service1,
					PlanID:    plan1,
				},
			)
			Expect(res.Code).To(Equal(http.StatusOK))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.UnbindReturns(errors.New("some unbinding error"))
			res := brokerTester.Unbind(
				instanceID,
				bindingID,
				broker_tester.RequestBody{
					ServiceID: service1,
					PlanID:    plan1,
				},
			)
			Expect(res.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("Update", func() {
		It("accepts an update request", func() {
			fakeProvider.UpdateReturns("operationData", nil)
			res := brokerTester.Update(
				instanceID,
				broker_tester.RequestBody{
					ServiceID: service1,
					PlanID:    plan1,
					PreviousValues: &broker_tester.RequestBody{
						PlanID: "plan2",
					},
				},
				true,
			)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			updateResponse := brokerapi.UpdateResponse{}
			err := json.Unmarshal(res.Body.Bytes(), &updateResponse)
			Expect(err).NotTo(HaveOccurred())

			expectedResponse := brokerapi.UpdateResponse{
				OperationData: "operationData",
			}
			Expect(updateResponse).To(Equal(expectedResponse))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.UpdateReturns("", errors.New("some update error"))
			res := brokerTester.Update(
				instanceID,
				broker_tester.RequestBody{
					ServiceID: service1,
					PlanID:    plan1,
					PreviousValues: &broker_tester.RequestBody{
						PlanID: "plan2",
					},
				},
				true,
			)
			Expect(res.Code).To(Equal(http.StatusInternalServerError))
		})

		It("rejects requests for synchronous updating", func() {
			res := brokerTester.Update(
				instanceID,
				broker_tester.RequestBody{
					ServiceID: service1,
					PlanID:    plan1,
					PreviousValues: &broker_tester.RequestBody{
						PlanID: "plan2",
					},
				},
				false,
			)
			Expect(res.Code).To(Equal(http.StatusUnprocessableEntity))
		})
	})

	Describe("LastOperation", func() {
		It("provides the state of the operation", func() {
			fakeProvider.LastOperationReturns(brokerapi.Succeeded, "description", nil)
			res := brokerTester.LastOperation(instanceID, "", "", "")
			Expect(res.Code).To(Equal(http.StatusOK))

			lastOperationResponse := brokerapi.LastOperationResponse{}
			err := json.Unmarshal(res.Body.Bytes(), &lastOperationResponse)
			Expect(err).NotTo(HaveOccurred())

			expectedResponse := brokerapi.LastOperationResponse{
				State:       brokerapi.Succeeded,
				Description: "description",
			}
			Expect(lastOperationResponse).To(Equal(expectedResponse))
		})

		It("responds with an internal server error if the provider errors", func() {
			lastOperationError := errors.New("some last operation error")
			fakeProvider.LastOperationReturns(brokerapi.InProgress, "", lastOperationError)
			res := brokerTester.LastOperation(instanceID, "", "", "")
			Expect(res.Code).To(Equal(http.StatusInternalServerError))

			lastOperationResponse := brokerapi.LastOperationResponse{}
			err := json.Unmarshal(res.Body.Bytes(), &lastOperationResponse)
			Expect(err).NotTo(HaveOccurred())

			expectedResponse := brokerapi.LastOperationResponse{
				State:       "",
				Description: lastOperationError.Error(),
			}
			Expect(lastOperationResponse).To(Equal(expectedResponse))
		})
	})
})
