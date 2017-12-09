package broker_test

import (
	"context"
	"encoding/json"
	"errors"

	"code.cloudfoundry.org/lager"
	. "github.com/henrytk/broker-skeleton/broker"
	"github.com/henrytk/broker-skeleton/provider"
	"github.com/henrytk/broker-skeleton/provider/fakes"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Broker", func() {
	var (
		validConfig      Config
		instanceID       string
		orgGUID          string
		spaceGUID        string
		plan1            brokerapi.ServicePlan
		service1         brokerapi.Service
		providerCatalog  provider.ProviderCatalog
		providerPlan1    provider.ProviderPlan
		providerService1 provider.ProviderService
	)

	BeforeEach(func() {
		instanceID = "instanceID"
		orgGUID = "org-guid"
		spaceGUID = "space-guid"
		plan1 = brokerapi.ServicePlan{
			ID:   "plan1",
			Name: "plan1",
		}
		service1 = brokerapi.Service{
			ID:    "service1",
			Name:  "service1",
			Plans: []brokerapi.ServicePlan{plan1},
		}
		providerPlan1 = provider.ProviderPlan{
			ID:             plan1.ID,
			ProviderConfig: json.RawMessage(`{"this-is": "some-provider-specific-plan-config"}`),
		}
		providerService1 = provider.ProviderService{
			ID:             service1.ID,
			ProviderConfig: json.RawMessage(`{"this-is": "some-provider-specific-service-config"}`),
			Plans:          []provider.ProviderPlan{providerPlan1},
		}
		providerCatalog = provider.ProviderCatalog{
			Services: []provider.ProviderService{providerService1},
		}
		validConfig = Config{
			Catalog: Catalog{
				brokerapi.CatalogResponse{
					Services: []brokerapi.Service{service1},
				},
			},
			Provider: provider.Provider{Catalog: providerCatalog},
		}
	})

	Describe("Provision", func() {
		var validProvisionDetails brokerapi.ProvisionDetails

		BeforeEach(func() {
			validProvisionDetails = brokerapi.ProvisionDetails{
				ServiceID:        service1.ID,
				PlanID:           plan1.ID,
				OrganizationGUID: orgGUID,
				SpaceGUID:        spaceGUID,
			}
		})

		It("logs a debug message when provision begins", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Provision(context.Background(), instanceID, validProvisionDetails, true)

			Expect(log).To(gbytes.Say("provision-start"))
		})

		It("errors if async isn't allowed", func() {
			b := New(validConfig, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))
			asyncAllowed := false

			_, err := b.Provision(context.Background(), instanceID, validProvisionDetails, asyncAllowed)

			Expect(err).To(Equal(brokerapi.ErrAsyncRequired))
		})

		It("errors if the service is not in the catalog", func() {
			config := validConfig
			config.Catalog = Catalog{Catalog: brokerapi.CatalogResponse{}}
			b := New(config, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))

			_, err := b.Provision(context.Background(), instanceID, validProvisionDetails, true)

			Expect(err).To(MatchError("Error: service " + service1.ID + " not found in the catalog"))
		})

		It("errors if the plan is not in the catalog", func() {
			config := validConfig
			config.Catalog.Catalog.Services[0].Plans = []brokerapi.ServicePlan{}
			b := New(config, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))

			_, err := b.Provision(context.Background(), instanceID, validProvisionDetails, true)

			Expect(err).To(MatchError("Error: plan " + plan1.ID + " not found in service " + service1.ID))
		})

		It("sets a deadline by which the provision request should complete", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Provision(context.Background(), instanceID, validProvisionDetails, true)

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			receivedContext, _ := fakeProvider.ProvisionArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct data to the Provider", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Provision(context.Background(), instanceID, validProvisionDetails, true)

			Expect(fakeProvider.ProvisionCallCount()).To(Equal(1))
			_, provisionData := fakeProvider.ProvisionArgsForCall(0)

			expectedProvisionData := provider.ProvisionData{
				InstanceID:      instanceID,
				Details:         validProvisionDetails,
				Service:         service1,
				Plan:            plan1,
				ProviderCatalog: providerCatalog,
			}

			Expect(provisionData).To(Equal(expectedProvisionData))
		})

		It("errors if provisioning fails", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.ProvisionReturns("", "", errors.New("ERROR PROVISIONING"))

			_, err := b.Provision(context.Background(), instanceID, validProvisionDetails, true)

			Expect(err).To(MatchError("ERROR PROVISIONING"))
		})

		It("logs a debug message when provisioning succeeds", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Provision(context.Background(), instanceID, validProvisionDetails, true)

			Expect(log).To(gbytes.Say("provision-success"))
		})

		It("returns the provisioned service spec", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.ProvisionReturns("dashboard URL", "operation data", nil)

			Expect(b.Provision(context.Background(), instanceID, validProvisionDetails, true)).
				To(Equal(brokerapi.ProvisionedServiceSpec{
					IsAsync:       true,
					DashboardURL:  "dashboard URL",
					OperationData: "operation data",
				}))
		})
	})

	Describe("Deprovision", func() {
		var validDeprovisionDetails brokerapi.DeprovisionDetails

		BeforeEach(func() {
			validDeprovisionDetails = brokerapi.DeprovisionDetails{
				ServiceID: service1.ID,
				PlanID:    plan1.ID,
			}
		})

		It("logs a debug message when deprovision begins", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Deprovision(context.Background(), instanceID, validDeprovisionDetails, true)

			Expect(log).To(gbytes.Say("deprovision-start"))
		})

		It("errors if async isn't allowed", func() {
			b := New(validConfig, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))
			asyncAllowed := false

			_, err := b.Deprovision(context.Background(), instanceID, validDeprovisionDetails, asyncAllowed)

			Expect(err).To(Equal(brokerapi.ErrAsyncRequired))
		})

		It("sets a deadline by which the deprovision request should complete", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Deprovision(context.Background(), instanceID, validDeprovisionDetails, true)

			Expect(fakeProvider.DeprovisionCallCount()).To(Equal(1))
			receivedContext, _ := fakeProvider.DeprovisionArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct data to the Provider", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Deprovision(context.Background(), instanceID, validDeprovisionDetails, true)

			Expect(fakeProvider.DeprovisionCallCount()).To(Equal(1))
			_, deprovisionData := fakeProvider.DeprovisionArgsForCall(0)

			expectedDeprovisionData := provider.DeprovisionData{
				InstanceID:      instanceID,
				Details:         validDeprovisionDetails,
				ProviderCatalog: providerCatalog,
			}

			Expect(deprovisionData).To(Equal(expectedDeprovisionData))
		})

		It("errors if deprovisioning fails", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.DeprovisionReturns("", errors.New("ERROR DEPROVISIONING"))

			_, err := b.Deprovision(context.Background(), instanceID, validDeprovisionDetails, true)

			Expect(err).To(MatchError("ERROR DEPROVISIONING"))
		})

		It("logs a debug message when deprovisioning succeeds", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Deprovision(context.Background(), instanceID, validDeprovisionDetails, true)

			Expect(log).To(gbytes.Say("deprovision-success"))
		})

		It("returns the deprovisioned service spec", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.DeprovisionReturns("operation data", nil)

			Expect(b.Deprovision(context.Background(), instanceID, validDeprovisionDetails, true)).
				To(Equal(brokerapi.DeprovisionServiceSpec{
					IsAsync:       true,
					OperationData: "operation data",
				}))
		})
	})
})
