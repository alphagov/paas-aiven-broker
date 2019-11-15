.PHONY: test unit integration

test: unit integration

integration: export EGRESS_IP=$(shell curl --silent icanhazip.com)
integration:
	ginkgo -p -nodes 4 ci/integration

unit: export SERVICE_NAME_PREFIX=test
unit: export AIVEN_API_TOKEN=token
unit: export AIVEN_PROJECT=project
unit:
	ginkgo $(COMMAND) -r --skipPackage=ci $(PACKAGE)

.PHONY: generate-fakes
generate-fakes:
	cd provider && counterfeiter -o fakes/fake_service_provider.go interface.go ServiceProvider
	cd provider/aiven && counterfeiter -o fakes/fake_client.go . Client
