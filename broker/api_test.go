package broker_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"

	"code.cloudfoundry.org/lager"
	. "github.com/henrytk/universal-service-broker/broker"
	"github.com/henrytk/universal-service-broker/provider/fakes"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker API", func() {
	var (
		instanceID   string
		orgGUID      string
		spaceGUID    string
		validConfig  Config
		username     string
		password     string
		logger       lager.Logger
		fakeProvider *fakes.FakeServiceProvider
		broker       *Broker
		brokerAPI    http.Handler
		brokerTester BrokerTester
	)

	BeforeEach(func() {
		instanceID = "instanceID"
		orgGUID = "org-guid"
		spaceGUID = "space-guid"
		validConfig = Config{
			API: API{
				BasicAuthUsername: username,
				BasicAuthPassword: password,
			},
			Catalog: Catalog{brokerapi.CatalogResponse{
				Services: []brokerapi.Service{
					brokerapi.Service{
						ID:            "service1",
						Name:          "service1",
						PlanUpdatable: true,
						Plans: []brokerapi.ServicePlan{
							brokerapi.ServicePlan{
								ID:   "plan1",
								Name: "plan1",
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

		brokerTester = NewBrokerTester(brokerapi.BrokerCredentials{
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
			Expect(catalogResponse.Services[0].ID).To(Equal("service1"))
			Expect(len(catalogResponse.Services[0].Plans)).To(Equal(1))
			Expect(catalogResponse.Services[0].Plans[0].ID).To(Equal("plan1"))
		})
	})

	Describe("Provision", func() {
		It("accepts a provision request", func() {
			fakeProvider.ProvisionReturns("dashboardURL", "operationData", nil)
			res := brokerTester.Provision(
				instanceID,
				RequestBody{
					ServiceID:        "service1",
					PlanID:           "plan1",
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
				RequestBody{
					ServiceID:        "service1",
					PlanID:           "plan1",
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
				RequestBody{
					ServiceID:        "service1",
					PlanID:           "plan1",
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
			res := brokerTester.Deprovision(instanceID, true)
			Expect(res.Code).To(Equal(http.StatusAccepted))

			deprovisionResponse := brokerapi.DeprovisionResponse{}
			err := json.Unmarshal(res.Body.Bytes(), &deprovisionResponse)
			Expect(err).NotTo(HaveOccurred())

			expectedResponse := brokerapi.DeprovisionResponse{
				OperationData: "operationData",
			}
			Expect(deprovisionResponse).To(Equal(expectedResponse))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.DeprovisionReturns("", errors.New("some deprovisioning error"))
			res := brokerTester.Deprovision(instanceID, true)
			Expect(res.Code).To(Equal(http.StatusInternalServerError))
		})

		It("rejects requests for synchronous deprovisioning", func() {
			res := brokerTester.Deprovision(instanceID, false)
			Expect(res.Code).To(Equal(http.StatusUnprocessableEntity))
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
				RequestBody{
					ServiceID: "service1",
					PlanID:    "plan1",
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
				RequestBody{
					ServiceID: "service1",
					PlanID:    "plan1",
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
				RequestBody{
					ServiceID: "service1",
					PlanID:    "plan1",
				},
			)
			Expect(res.Code).To(Equal(http.StatusOK))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.UnbindReturns(errors.New("some unbinding error"))
			res := brokerTester.Unbind(
				instanceID,
				bindingID,
				RequestBody{
					ServiceID: "service1",
					PlanID:    "plan1",
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
				RequestBody{
					ServiceID: "service1",
					PlanID:    "plan1",
					PreviousValues: &RequestBody{
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
				RequestBody{
					ServiceID: "service1",
					PlanID:    "plan1",
					PreviousValues: &RequestBody{
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
				RequestBody{
					ServiceID: "service1",
					PlanID:    "plan1",
					PreviousValues: &RequestBody{
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
			res := brokerTester.LastOperation(instanceID)
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
			res := brokerTester.LastOperation(instanceID)
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

type BrokerTester struct {
	credentials brokerapi.BrokerCredentials
	brokerAPI   http.Handler
}

func NewBrokerTester(credentials brokerapi.BrokerCredentials, brokerAPI http.Handler) BrokerTester {
	return BrokerTester{
		credentials: credentials,
		brokerAPI:   brokerAPI,
	}
}

type RequestBody struct {
	ServiceID        string       `json:"service_id,omitempty"`
	PlanID           string       `json:"plan_id,omitempty"`
	OrganizationGUID string       `json:"organization_guid,omitempty"`
	SpaceGUID        string       `json:"space_guid,omitempty"`
	AppGUID          string       `json:"app_guid,omitempty"`
	PreviousValues   *RequestBody `json:"previous_values,omitempty"`
}

func (bt BrokerTester) Services() *httptest.ResponseRecorder {
	return bt.Get("/v2/catalog", url.Values{})
}

func (bt BrokerTester) Provision(instanceID string, body RequestBody, async bool) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Put(
		"/v2/service_instances/"+instanceID,
		bytes.NewBuffer(bodyJSON),
		url.Values{"accepts_incomplete": []string{strconv.FormatBool(async)}},
	)
}

func (bt BrokerTester) Deprovision(instanceID string, async bool) *httptest.ResponseRecorder {
	return bt.Delete(
		"/v2/service_instances/"+instanceID,
		nil,
		url.Values{"accepts_incomplete": []string{strconv.FormatBool(async)}},
	)
}

func (bt BrokerTester) Bind(instanceID, bindingID string, body RequestBody) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Put(
		fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID),
		bytes.NewBuffer(bodyJSON),
		url.Values{},
	)
}

func (bt BrokerTester) Unbind(instanceID, bindingID string, body RequestBody) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Delete(
		fmt.Sprintf(
			"/v2/service_instances/%s/service_bindings/%s",
			instanceID,
			bindingID,
		),
		bytes.NewBuffer(bodyJSON),
		url.Values{},
	)
}

func (bt BrokerTester) Update(instanceID string, body RequestBody, async bool) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Patch(
		"/v2/service_instances/"+instanceID,
		bytes.NewBuffer(bodyJSON),
		url.Values{"accepts_incomplete": []string{strconv.FormatBool(async)}},
	)
}

func (bt BrokerTester) LastOperation(instanceID string) *httptest.ResponseRecorder {
	return bt.Get(
		fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID),
		url.Values{},
	)
}

func (bt BrokerTester) Get(path string, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("GET", path, nil, params))
}

func (bt BrokerTester) Put(path string, body io.Reader, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("PUT", path, body, params))
}

func (bt BrokerTester) Patch(path string, body io.Reader, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("PATCH", path, body, params))
}

func (bt BrokerTester) Delete(path string, body io.Reader, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("DELETE", path, body, params))
}

func (bt BrokerTester) newRequest(method, path string, body io.Reader, params url.Values) *http.Request {
	url := fmt.Sprintf("http://%s", "127.0.0.1:8080"+path)
	req := httptest.NewRequest(method, url, body)
	req.URL.RawQuery = params.Encode()
	return req
}

func (bt BrokerTester) do(req *http.Request) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req.SetBasicAuth(bt.credentials.Username, bt.credentials.Password)
	bt.brokerAPI.ServeHTTP(res, req)
	return res
}
