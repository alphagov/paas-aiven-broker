.PHONY: test
test: unit integration

.PHONY: integration
integration: export EGRESS_IP=$(shell curl --silent icanhazip.com)
integration:
	go run github.com/onsi/ginkgo/v2/ginkgo \
		-procs 4 --compilers 4 \
		--poll-progress-after=120s --poll-progress-interval=30s \
		ci/integration

# ensure $TMPDIR is set - it is present on darwin but not linux
ifeq ($(TMPDIR),)
TMPDIR := /tmp/
endif

PASSWORD_STORE_DIR := $(HOME)/.paas-pass
$(TMPDIR)/paas-aiven-broker_integration.env: Makefile ci/create_integration_envfile.sh
	$(eval export PASSWORD_STORE_DIR=$(PASSWORD_STORE_DIR))
	@ci/create_integration_envfile.sh $@

# This target can be used locally to run the integration tests.
# It will automatically create a temporary environment file, pulling
# the necessary credentials from the password store.
local_integration: $(TMPDIR)/paas-aiven-broker_integration.env
	$(foreach line,$(shell cat $<),$(eval export $(line)))
	$(eval export BROKER_NAME=integration_local_$(shell git rev-parse --short --verify HEAD))
	@$(MAKE) integration

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
