package broker_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"code.cloudfoundry.org/lager"
	. "github.com/henrytk/broker-skeleton/broker"
	"github.com/henrytk/broker-skeleton/provider/fakes"
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
						ID:   "service1",
						Name: "service1",
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
			res := brokerTester.Get("/v2/catalog", url.Values{})
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
			res := brokerTester.Put(
				"/v2/service_instances/"+instanceID,
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"organization_guid": "%s",
					"space_guid": "%s",
					"parameters": {}
				}`, orgGUID, spaceGUID)),
				url.Values{"accepts_incomplete": []string{"true"}},
			)

			Expect(res.Code).To(Equal(http.StatusAccepted))
		})

		It("responds with an internal server error if the provider errors", func() {
			fakeProvider.ProvisionReturns("", "", errors.New("some provisioning error"))
			res := brokerTester.Put(
				"/v2/service_instances/"+instanceID,
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"organization_guid": "%s",
					"space_guid": "%s",
					"parameters": {}
				}`, orgGUID, spaceGUID)),
				url.Values{"accepts_incomplete": []string{"true"}},
			)

			Expect(res.Code).To(Equal(http.StatusInternalServerError))
		})

		It("rejects requests for synchronous provisioning", func() {
			res := brokerTester.Put(
				"/v2/service_instances/"+instanceID,
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "service1",
					"plan_id": "plan1",
					"organization_guid": "%s",
					"space_guid": "%s",
					"parameters": {}
				}`, orgGUID, spaceGUID)),
				url.Values{"accepts_incomplete": []string{"false"}},
			)

			Expect(res.Code).To(Equal(http.StatusUnprocessableEntity))
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

func (bt BrokerTester) Get(path string, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("GET", path, nil, params))
}

func (bt BrokerTester) Put(path string, body io.Reader, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("PUT", path, body, params))
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
