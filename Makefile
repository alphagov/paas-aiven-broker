.PHONY: test
test: unit integration

.PHONY: integration
integration: export EGRESS_IP=$(shell curl --silent icanhazip.com)
integration:
	go run github.com/onsi/ginkgo/v2/ginkgo -p -nodes 4 ci/integration

unit: export DEPLOY_ENV=test
unit: export BROKER_NAME=test
unit: export AIVEN_API_TOKEN=token
unit: export AIVEN_PROJECT=project
unit:
	go run github.com/onsi/ginkgo/v2/ginkgo $(COMMAND) -r --skip-package=ci $(PACKAGE)


provider/fakes/fake_service_provider.go: provider/interface.go
	go generate $<
provider/aiven/fakes/fake_client.go: provider/aiven/client.go
	go generate $<

generate-fakes: provider/fakes/fake_service_provider.go provider/aiven/fakes/fake_client.go
