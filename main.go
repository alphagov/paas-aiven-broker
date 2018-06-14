package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-aiven-broker/broker"
	"github.com/alphagov/paas-aiven-broker/provider"
)

var configFilePath string

func main() {
	flag.StringVar(&configFilePath, "config", "", "Location of the config file")
	flag.Parse()

	file, err := os.Open(configFilePath)
	if err != nil {
		log.Fatalf("Error opening config file %s: %s\n", configFilePath, err)
	}
	defer file.Close()

	config, err := broker.NewConfig(file)
	if err != nil {
		log.Fatalf("Error validating config file: %v\n", err)
	}

	aivenProvider, err := provider.New(config.Provider)
	if err != nil {
		log.Fatalf("Error creating Aiven provider: %v\n", err)
	}

	logger := lager.NewLogger("aiven-service-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, config.API.LagerLogLevel))

	aivenBroker := broker.New(config, aivenProvider, logger)
	brokerServer := broker.NewAPI(aivenBroker, logger, config)

	listener, err := net.Listen("tcp", ":"+config.API.Port)
	if err != nil {
		log.Fatalf("Error listening to port %s: %s", config.API.Port, err)
	}
	fmt.Println("Aiven service broker started on port " + config.API.Port + "...")
	http.Serve(listener, brokerServer)
}
