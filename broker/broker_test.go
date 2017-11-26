package broker_test

import (
	"context"

	"code.cloudfoundry.org/lager"
	. "github.com/henrytk/broker-skeleton/broker"
	"github.com/henrytk/broker-skeleton/provider/fakes"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Broker", func() {
	var validConfig Config

	BeforeEach(func() {
		validConfig = Config{
			Catalog: Catalog{
				brokerapi.CatalogResponse{
					Services: []brokerapi.Service{
						brokerapi.Service{
							ID:   "service1",
							Name: "service1",
							Plans: []brokerapi.ServicePlan{
								brokerapi.ServicePlan{
									ID:   "plan1",
									Name: "plan1",
								},
							},
						},
					},
				},
			},
		}
	})

	Describe("Provision", func() {
		var validProvisionDetails brokerapi.ProvisionDetails

		BeforeEach(func() {
			validProvisionDetails = brokerapi.ProvisionDetails{
				ServiceID:        "service1",
				PlanID:           "plan1",
				OrganizationGUID: "org-guid",
				SpaceGUID:        "space-guid",
			}
		})

		It("logs a debug message when provision begins", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeProvider{}, logger)

			b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

			Expect(log).To(gbytes.Say("provision-start"))
		})

		It("errors if async isn't allowed", func() {
			b := New(validConfig, &fakes.FakeProvider{}, lager.NewLogger("broker"))
			asyncAllowed := false

			_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, asyncAllowed)

			Expect(err).To(Equal(brokerapi.ErrAsyncRequired))
		})

		It("errors if the plan config cannot be retrieved", func() {
			b := New(Config{}, &fakes.FakeProvider{}, lager.NewLogger("broker"))

			_, err := b.Provision(context.Background(), "instanceid", validProvisionDetails, true)

			Expect(err).To(MatchError("service plan plan1: no plan found"))
		})
	})
})
