package provider

import (
	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider internals", func() {

	Describe("providerStatesMapping", func() {
		It("should return 'succeeded' when RUNNING", func() {
			state, description := providerStatesMapping(aiven.Running)

			Expect(state).To(Equal(brokerapi.Succeeded))
			Expect(description).To(Equal("Last operation succeeded"))
		})

		It("should return 'in progress' when REBUILDING", func() {
			state, description := providerStatesMapping(aiven.Rebuilding)

			Expect(state).To(Equal(brokerapi.InProgress))
			Expect(description).To(Equal("Rebuilding"))
		})

		It("should return 'in progress' when REBALANCING", func() {
			state, description := providerStatesMapping(aiven.Rebalancing)

			Expect(state).To(Equal(brokerapi.InProgress))
			Expect(description).To(Equal("Rebalancing"))
		})

		It("should return 'failed' when POWEROFF", func() {
			state, description := providerStatesMapping(aiven.PowerOff)

			Expect(state).To(Equal(brokerapi.Failed))
			Expect(description).To(Equal("Last operation failed: service is powered off"))
		})

		It("should return 'in progress' by default", func() {
			state, description := providerStatesMapping("foo")

			Expect(state).To(Equal(brokerapi.InProgress))
			Expect(description).To(Equal("Unknown state: foo"))
		})
	})
})
