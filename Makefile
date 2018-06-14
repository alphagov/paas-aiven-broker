.PHONY: test unit integration

test: unit integration

integration:
	ginkgo ci/integration

unit:
	$(eval export SERVICE_NAME_PREFIX=test)
	$(eval export AIVEN_API_TOKEN=token)
	$(eval export AIVEN_PROJECT=project)
	ginkgo -r --skipPackage=ci
