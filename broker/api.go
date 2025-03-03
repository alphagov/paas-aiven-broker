package broker

import (
	"github.com/pivotal-cf/brokerapi/domain"
	"net/http"

	"code.cloudfoundry.org/lager/v3"
	"github.com/pivotal-cf/brokerapi"
)

func NewAPI(broker domain.ServiceBroker, logger lager.Logger, config Config) http.Handler {
	credentials := brokerapi.BrokerCredentials{
		Username: config.API.BasicAuthUsername,
		Password: config.API.BasicAuthPassword,
	}

	brokerAPI := brokerapi.New(broker, logger, credentials)
	mux := http.NewServeMux()
	mux.Handle("/", brokerAPI)
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}
