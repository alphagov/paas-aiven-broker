package provider

import (
	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider internals", func() {

	DescribeTable("buildServiceName",
		func(prefix, instanceId, expected string) {
			Expect(buildServiceName(prefix, instanceId)).To(Equal(expected))
		},
		Entry("combines prefix and instanceId", "env", "09e1993e-62e2-4040-adf2-4d3ec741efe6", "env-09e1993e-62e2-4040-adf2-4d3ec741efe6"),
		Entry("downcases everything", "Env", "09E1993E-62E2-4040-ADF2-4D3EC741EFE6", "env-09e1993e-62e2-4040-adf2-4d3ec741efe6"),
	)

	DescribeTable("providerStatesMapping",
		func(inputState aiven.ServiceStatus, expectedState brokerapi.LastOperationState, expectedDescription string) {
			state, description := providerStatesMapping(inputState)
			Expect(state).To(Equal(expectedState))
			Expect(description).To(Equal(expectedDescription))
		},
		Entry("returns 'succeeded' when RUNNING", aiven.Running, brokerapi.Succeeded, "Last operation succeeded"),
		Entry("returns 'in progress' when REBUILDING", aiven.Rebuilding, brokerapi.InProgress, "Rebuilding"),
		Entry("returns 'in progress' when REBALANCING", aiven.Rebalancing, brokerapi.InProgress, "Rebalancing"),
		Entry("returns 'failed' when POWEROFF", aiven.PowerOff, brokerapi.Failed, "Last operation failed: service is powered off"),
		Entry("returns 'in progress' by default", aiven.ServiceStatus("foo"), brokerapi.InProgress, "Unknown state: foo"),
	)
})
