.PHONY: test unit integration

test: unit integration

integration: export EGRESS_IP=$(shell curl --silent icanhazip.com)
integration:
	ginkgo -v --progress ci/integration

unit: export SERVICE_NAME_PREFIX=test
unit: export AIVEN_API_TOKEN=token
unit: export AIVEN_PROJECT=project
unit:
	ginkgo $(COMMAND) -r --skipPackage=ci $(PACKAGE)
