.PHONY: test unit integration

test: unit integration

integration: export EGRESS_IP=$(shell curl --silent icanhazip.com)
integration:
	go run github.com/onsi/ginkgo/v2/ginkgo -p -nodes 4 ci/integration

unit: export DEPLOY_ENV=test
unit: export BROKER_NAME=test
unit: export AIVEN_API_TOKEN=token
unit: export AIVEN_PROJECT=project
unit:
	go run github.com/onsi/ginkgo/v2/ginkgo $(COMMAND) -r --skip-package=ci $(PACKAGE)

.PHONY: generate-fakes
generate-fakes:
	cd provider && counterfeiter -o fakes/fake_service_provider.go interface.go ServiceProvider
	cd provider/aiven && counterfeiter -o fakes/fake_client.go . Client
