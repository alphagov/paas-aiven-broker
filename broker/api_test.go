package broker_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"code.cloudfoundry.org/lager"
	. "github.com/henrytk/broker-skeleton/broker"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker API", func() {
	var (
		err               error
		config            Config
		logger            lager.Logger
		validConfigSource = `
			{
				"basic_auth_username":"admin",
				"basic_auth_password":"1234"
			}
		`
		broker       *Broker
		credentials  brokerapi.BrokerCredentials
		brokerAPI    http.Handler
		brokerClient BrokerClient
	)

	BeforeEach(func() {
		config, err = NewConfig(strings.NewReader(validConfigSource))
		Expect(err).NotTo(HaveOccurred())
		logger = lager.NewLogger("broker-api")
		broker = New(config, logger)
		brokerAPI = NewAPI(broker, logger, config)

		credentials = brokerapi.BrokerCredentials{
			Username: config.BasicAuthUsername,
			Password: config.BasicAuthPassword,
		}
		brokerClient = BrokerClient{credentials}
	})

	It("serves a healthcheck endpoint", func() {
		req := brokerClient.NewRequest("GET", "/healthcheck")
		res := httptest.NewRecorder()
		brokerAPI.ServeHTTP(res, req)
		Expect(res.Code).To(Equal(http.StatusOK))
	})
})

type BrokerClient struct {
	credentials brokerapi.BrokerCredentials
}

func (bc BrokerClient) NewRequest(method, path string) *http.Request {
	url := fmt.Sprintf("http://%s", "127.0.0.1:8080"+path)
	req := httptest.NewRequest(method, url, nil)
	req.SetBasicAuth(bc.credentials.Username, bc.credentials.Password)
	return req
}
