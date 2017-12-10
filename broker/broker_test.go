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
		plan2            brokerapi.ServicePlan
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
		plan2 = brokerapi.ServicePlan{
			ID:   "plan2",
			Name: "plan2",
		}
		service1 = brokerapi.Service{
			ID:            "service1",
			Name:          "service1",
			PlanUpdatable: true,
			Plans:         []brokerapi.ServicePlan{plan1, plan2},
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

	Describe("Bind", func() {
		var (
			bindingID        string
			appGUID          string
			bindResource     *brokerapi.BindResource
			validBindDetails brokerapi.BindDetails
		)

		BeforeEach(func() {
			bindingID = "bindingID"
			appGUID = "appGUID"
			bindResource = &brokerapi.BindResource{
				AppGuid: appGUID,
			}
			validBindDetails = brokerapi.BindDetails{
				AppGUID:      appGUID,
				PlanID:       plan1.ID,
				ServiceID:    service1.ID,
				BindResource: bindResource,
			}
		})

		It("logs a debug message when binding begins", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Bind(context.Background(), instanceID, bindingID, validBindDetails)

			Expect(log).To(gbytes.Say("binding-start"))
		})

		It("sets a deadline by which the binding request should complete", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Bind(context.Background(), instanceID, bindingID, validBindDetails)

			Expect(fakeProvider.BindCallCount()).To(Equal(1))
			receivedContext, _ := fakeProvider.BindArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct data to the Provider", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Bind(context.Background(), instanceID, bindingID, validBindDetails)

			Expect(fakeProvider.BindCallCount()).To(Equal(1))
			_, bindData := fakeProvider.BindArgsForCall(0)

			expectedBindData := provider.BindData{
				InstanceID:      instanceID,
				BindingID:       bindingID,
				Details:         validBindDetails,
				ProviderCatalog: providerCatalog,
			}

			Expect(bindData).To(Equal(expectedBindData))
		})

		It("errors if binding fails", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.BindReturns(brokerapi.Binding{}, errors.New("ERROR BINDING"))

			_, err := b.Bind(context.Background(), instanceID, bindingID, validBindDetails)

			Expect(err).To(MatchError("ERROR BINDING"))
		})

		It("logs a debug message when binding succeeds", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Bind(context.Background(), instanceID, bindingID, validBindDetails)

			Expect(log).To(gbytes.Say("binding-success"))
		})

		It("returns the binding", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.BindReturns(brokerapi.Binding{
				Credentials: "some-value-of-interface{}-type",
			}, nil)

			Expect(b.Bind(context.Background(), instanceID, bindingID, validBindDetails)).
				To(Equal(brokerapi.Binding{
					Credentials: "some-value-of-interface{}-type",
				}))
		})
	})

	Describe("Unbind", func() {
		var (
			bindingID          string
			validUnbindDetails brokerapi.UnbindDetails
		)

		BeforeEach(func() {
			bindingID = "bindingID"
			validUnbindDetails = brokerapi.UnbindDetails{
				PlanID:    plan1.ID,
				ServiceID: service1.ID,
			}
		})

		It("logs a debug message when unbinding begins", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Unbind(context.Background(), instanceID, bindingID, validUnbindDetails)

			Expect(log).To(gbytes.Say("unbinding-start"))
		})

		It("sets a deadline by which the unbinding request should complete", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Unbind(context.Background(), instanceID, bindingID, validUnbindDetails)

			Expect(fakeProvider.UnbindCallCount()).To(Equal(1))
			receivedContext, _ := fakeProvider.UnbindArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct data to the Provider", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Unbind(context.Background(), instanceID, bindingID, validUnbindDetails)

			Expect(fakeProvider.UnbindCallCount()).To(Equal(1))
			_, unbindData := fakeProvider.UnbindArgsForCall(0)

			expectedUnbindData := provider.UnbindData{
				InstanceID:      instanceID,
				BindingID:       bindingID,
				Details:         validUnbindDetails,
				ProviderCatalog: providerCatalog,
			}

			Expect(unbindData).To(Equal(expectedUnbindData))
		})

		It("errors if unbinding fails", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.UnbindReturns(errors.New("ERROR UNBINDING"))

			err := b.Unbind(context.Background(), instanceID, bindingID, validUnbindDetails)

			Expect(err).To(MatchError("ERROR UNBINDING"))
		})

		It("logs a debug message when unbinding succeeds", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Unbind(context.Background(), instanceID, bindingID, validUnbindDetails)

			Expect(log).To(gbytes.Say("unbinding-success"))
		})
	})

	Describe("Update", func() {
		var updatePlanDetails brokerapi.UpdateDetails

		BeforeEach(func() {
			updatePlanDetails = brokerapi.UpdateDetails{
				ServiceID: service1.ID,
				PlanID:    plan2.ID,
				PreviousValues: brokerapi.PreviousValues{
					ServiceID: service1.ID,
					PlanID:    plan1.ID,
					OrgID:     orgGUID,
					SpaceID:   spaceGUID,
				},
			}
		})

		Describe("Updatability", func() {
			Context("when the plan is not updatable", func() {
				var updateParametersDetails brokerapi.UpdateDetails

				BeforeEach(func() {
					validConfig.Catalog.Catalog.Services[0].PlanUpdatable = false

					updateParametersDetails = brokerapi.UpdateDetails{
						ServiceID:     service1.ID,
						PlanID:        plan1.ID,
						RawParameters: json.RawMessage(`{"new":"parameter"}`),
						PreviousValues: brokerapi.PreviousValues{
							ServiceID: service1.ID,
							PlanID:    plan1.ID,
							OrgID:     orgGUID,
							SpaceID:   spaceGUID,
						},
					}
				})

				It("returns an error when changing the plan", func() {
					b := New(validConfig, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))

					Expect(updatePlanDetails.PlanID).NotTo(Equal(updatePlanDetails.PreviousValues.PlanID))
					_, err := b.Update(context.Background(), instanceID, updatePlanDetails, true)

					Expect(err).To(Equal(brokerapi.ErrPlanChangeNotSupported))
				})

				It("accepts the update request when just changing parameters", func() {
					b := New(validConfig, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))

					Expect(updateParametersDetails.PlanID).To(Equal(updateParametersDetails.PreviousValues.PlanID))
					_, err := b.Update(context.Background(), instanceID, updateParametersDetails, true)

					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		It("logs a debug message when update begins", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Update(context.Background(), instanceID, updatePlanDetails, true)

			Expect(log).To(gbytes.Say("update-start"))
		})

		It("errors if async isn't allowed", func() {
			b := New(validConfig, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))
			asyncAllowed := false

			_, err := b.Update(context.Background(), instanceID, updatePlanDetails, asyncAllowed)

			Expect(err).To(Equal(brokerapi.ErrAsyncRequired))
		})

		It("errors if the service is not in the catalog", func() {
			config := validConfig
			config.Catalog = Catalog{Catalog: brokerapi.CatalogResponse{}}
			b := New(config, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))

			_, err := b.Update(context.Background(), instanceID, updatePlanDetails, true)

			Expect(err).To(MatchError("Error: service " + service1.ID + " not found in the catalog"))
		})

		It("errors if the plan is not in the catalog", func() {
			config := validConfig
			config.Catalog.Catalog.Services[0].Plans = []brokerapi.ServicePlan{}
			b := New(config, &fakes.FakeServiceProvider{}, lager.NewLogger("broker"))

			_, err := b.Update(context.Background(), instanceID, updatePlanDetails, true)

			Expect(err).To(MatchError("Error: plan " + plan2.ID + " not found in service " + service1.ID))
		})

		It("sets a deadline by which the update request should complete", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Update(context.Background(), instanceID, updatePlanDetails, true)

			Expect(fakeProvider.UpdateCallCount()).To(Equal(1))
			receivedContext, _ := fakeProvider.UpdateArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct data to the Provider", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.Update(context.Background(), instanceID, updatePlanDetails, true)

			Expect(fakeProvider.UpdateCallCount()).To(Equal(1))
			_, updateData := fakeProvider.UpdateArgsForCall(0)

			expectedUpdateData := provider.UpdateData{
				InstanceID:      instanceID,
				Details:         updatePlanDetails,
				Service:         service1,
				Plan:            plan2,
				ProviderCatalog: providerCatalog,
			}

			Expect(updateData).To(Equal(expectedUpdateData))
		})

		It("errors if update fails", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.UpdateReturns("", errors.New("ERROR UPDATING"))

			_, err := b.Update(context.Background(), instanceID, updatePlanDetails, true)

			Expect(err).To(MatchError("ERROR UPDATING"))
		})

		It("logs a debug message when updating succeeds", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.Update(context.Background(), instanceID, updatePlanDetails, true)

			Expect(log).To(gbytes.Say("update-success"))
		})

		It("returns the update service spec", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.UpdateReturns("operation data", nil)

			Expect(b.Update(context.Background(), instanceID, updatePlanDetails, true)).
				To(Equal(brokerapi.UpdateServiceSpec{
					IsAsync:       true,
					OperationData: "operation data",
				}))
		})
	})

	Describe("LastOperation", func() {
		var operationData string

		BeforeEach(func() {
			operationData = `{"operation_type": "provision"}`
		})

		It("logs a debug message when the last operation check begins", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.LastOperation(context.Background(), instanceID, operationData)

			Expect(log).To(gbytes.Say("last-operation-start"))
		})

		It("sets a deadline by which the last operation request should complete", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.LastOperation(context.Background(), instanceID, operationData)

			Expect(fakeProvider.LastOperationCallCount()).To(Equal(1))
			receivedContext, _ := fakeProvider.LastOperationArgsForCall(0)

			_, hasDeadline := receivedContext.Deadline()

			Expect(hasDeadline).To(BeTrue())
		})

		It("passes the correct data to the Provider", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))

			b.LastOperation(context.Background(), instanceID, operationData)

			Expect(fakeProvider.LastOperationCallCount()).To(Equal(1))
			_, lastOperationData := fakeProvider.LastOperationArgsForCall(0)

			expectedLastOperationData := provider.LastOperationData{
				InstanceID:      instanceID,
				OperationData:   operationData,
				ProviderCatalog: providerCatalog,
			}

			Expect(lastOperationData).To(Equal(expectedLastOperationData))
		})

		It("errors if last operation fails", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.LastOperationReturns(brokerapi.InProgress, "", errors.New("ERROR LAST OPERATION"))

			_, err := b.LastOperation(context.Background(), instanceID, operationData)

			Expect(err).To(MatchError("ERROR LAST OPERATION"))
		})

		It("logs a debug message when last operation check succeeds", func() {
			logger := lager.NewLogger("broker")
			log := gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			b := New(validConfig, &fakes.FakeServiceProvider{}, logger)

			b.LastOperation(context.Background(), instanceID, operationData)

			Expect(log).To(gbytes.Say("last-operation-success"))
		})

		It("returns the last operation status", func() {
			fakeProvider := &fakes.FakeServiceProvider{}
			b := New(validConfig, fakeProvider, lager.NewLogger("broker"))
			fakeProvider.LastOperationReturns(brokerapi.Succeeded, "Provision successful", nil)

			Expect(b.LastOperation(context.Background(), instanceID, operationData)).
				To(Equal(brokerapi.LastOperation{
					State:       brokerapi.Succeeded,
					Description: "Provision successful",
				}))
		})
	})
})
