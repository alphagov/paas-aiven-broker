package provider_test

import (
	"context"
	"errors"

	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/alphagov/paas-aiven-broker/provider/aiven/fakes"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider", func() {

	var (
		aivenProvider   *provider.AivenProvider
		fakeAivenClient *fakes.FakeClient
		config          *provider.Config
	)

	BeforeEach(func() {
		config = &provider.Config{
			Cloud:             "aws-eu-west-1",
			ServiceNamePrefix: "env",
			Catalog: provider.Catalog{
				Services: []provider.Service{
					{
						Plans: []provider.Plan{
							{
								PlanSpecificConfig: provider.PlanSpecificConfig{
									AivenPlan:            "startup-1",
									ElasticsearchVersion: "6",
								},
							},
						},
					},
				},
			},
		}
		fakeAivenClient = &fakes.FakeClient{}
		aivenProvider = &provider.AivenProvider{
			Client: fakeAivenClient,
			Config: config,
		}
	})

	Describe("Provision", func() {
		It("passes the correct parameters to the Aiven client", func() {
			provisionData := provider.ProvisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			_, _, err := aivenProvider.Provision(context.Background(), provisionData)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAivenClient.CreateServiceCallCount()).To(Equal(1))

			expectedParameters := &aiven.CreateServiceInput{
				Cloud:       "aws-eu-west-1",
				Plan:        "startup-1",
				ServiceName: "env-69bc39f8",
				ServiceType: "elasticsearch",
				UserConfig: aiven.UserConfig{
					ElasticsearchVersion: "6",
				},
			}
			Expect(fakeAivenClient.CreateServiceArgsForCall(0)).To(Equal(expectedParameters))
		})

		It("errors if the client errors", func() {
			provisionData := provider.ProvisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			fakeAivenClient.CreateServiceReturnsOnCall(0, "", errors.New("some-error"))

			_, _, err := aivenProvider.Provision(context.Background(), provisionData)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Deprovision", func() {
		It("passes the correct parameters to the Aiven client", func() {
			deprovisionData := provider.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			_, err := aivenProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAivenClient.DeleteServiceCallCount()).To(Equal(1))

			expectedParameters := &aiven.DeleteServiceInput{
				ServiceName: "env-69bc39f8",
			}
			Expect(fakeAivenClient.DeleteServiceArgsForCall(0)).To(Equal(expectedParameters))
		})

		It("errors if the client errors", func() {
			deprovisionData := provider.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			fakeAivenClient.DeleteServiceReturnsOnCall(0, "", errors.New("some-error"))

			_, err := aivenProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("LastOperation", func() {
		It("should return succeeded when the service is running", func() {
			expectedParameters := &aiven.GetServiceStatusInput{
				ServiceName: "env-69bc39f8",
			}

			lastOperationData := provider.LastOperationData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}

			fakeAivenClient.GetServiceStatusReturnsOnCall(0, aiven.Running, nil)
			actualLastOperationState, description, err := aivenProvider.LastOperation(context.Background(), lastOperationData)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAivenClient.GetServiceStatusArgsForCall(0)).To(Equal(expectedParameters))
			Expect(actualLastOperationState).To(Equal(brokerapi.Succeeded))
			Expect(description).To(Equal("Last operation succeeded"))
		})

		It("should return an error if the client fails to get status information", func() {
			lastOperationData := provider.LastOperationData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}

			fakeAivenClient.GetServiceStatusReturnsOnCall(0, aiven.Running, errors.New("some-error"))
			actualLastOperationState, description, err := aivenProvider.LastOperation(context.Background(), lastOperationData)

			Expect(err).To(MatchError("some-error"))
			Expect(actualLastOperationState).To(Equal(brokerapi.LastOperationState("")))
			Expect(description).To(Equal(""))
		})
	})

	Describe("ProviderStatesMapping", func() {
		It("should return 'succeeded' when RUNNING", func() {
			state, description := provider.ProviderStatesMapping(aiven.Running)

			Expect(state).To(Equal(brokerapi.Succeeded))
			Expect(description).To(Equal("Last operation succeeded"))
		})

		It("should return 'in progress' when REBUILDING", func() {
			state, description := provider.ProviderStatesMapping(aiven.Rebuilding)

			Expect(state).To(Equal(brokerapi.InProgress))
			Expect(description).To(Equal("Rebuilding"))
		})

		It("should return 'in progress' when REBALANCING", func() {
			state, description := provider.ProviderStatesMapping(aiven.Rebalancing)

			Expect(state).To(Equal(brokerapi.InProgress))
			Expect(description).To(Equal("Rebalancing"))
		})

		It("should return 'failed' when POWEROFF", func() {
			state, description := provider.ProviderStatesMapping(aiven.PowerOff)

			Expect(state).To(Equal(brokerapi.Failed))
			Expect(description).To(Equal("Last operation failed: service is powered off"))
		})

		It("should return 'in progress' by default", func() {
			state, description := provider.ProviderStatesMapping("foo")

			Expect(state).To(Equal(brokerapi.InProgress))
			Expect(description).To(Equal("Unknown state: foo"))
		})
	})
})
