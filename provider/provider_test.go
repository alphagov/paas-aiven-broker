package provider_test

import (
	"context"
	"errors"
	"time"

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
						Service: brokerapi.Service{ID: "uuid-1"},
						Plans: []provider.Plan{
							{
								ServicePlan: brokerapi.ServicePlan{ID: "uuid-2"},
								PlanSpecificConfig: provider.PlanSpecificConfig{
									AivenPlan:            "startup-1",
									ElasticsearchVersion: "6",
								},
							},
							{
								ServicePlan: brokerapi.ServicePlan{ID: "uuid-3"},
								PlanSpecificConfig: provider.PlanSpecificConfig{
									AivenPlan:            "startup-2",
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
				Service:    brokerapi.Service{ID: "uuid-1"},
				Plan:       brokerapi.ServicePlan{ID: "uuid-2"},
			}
			_, _, err := aivenProvider.Provision(context.Background(), provisionData)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAivenClient.CreateServiceCallCount()).To(Equal(1))

			expectedParameters := &aiven.CreateServiceInput{
				Cloud:       "aws-eu-west-1",
				Plan:        "startup-1",
				ServiceName: "env69bc39f8",
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
				ServiceName: "env69bc39f8",
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

	Describe("Bind", func() {
		It("passes the correct parameters to the Aiven client", func() {
			bindData := provider.BindData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "D26EA3FB-AA78-451C-9ED0-233935ED388F",
			}

			fakeAivenClient.CreateServiceUserReturnsOnCall(0, "superdupersecret", nil)
			fakeAivenClient.GetServiceConnectionDetailsReturnsOnCall(0, "example.com", "23362", nil)

			actualBinding, err := aivenProvider.Bind(context.Background(), bindData)
			Expect(err).ToNot(HaveOccurred())

			expectedCreateServiceUserParameters := &aiven.CreateServiceUserInput{
				ServiceName: "env69bc39f8",
				Username:    bindData.BindingID,
			}
			Expect(fakeAivenClient.CreateServiceUserArgsForCall(0)).To(Equal(expectedCreateServiceUserParameters))

			expectedGetServiceConnectionDetailsParameters := &aiven.GetServiceInput{
				ServiceName: "env69bc39f8",
			}
			Expect(fakeAivenClient.GetServiceConnectionDetailsArgsForCall(0)).To(Equal(expectedGetServiceConnectionDetailsParameters))

			expectedBinding := brokerapi.Binding{
				Credentials: provider.Credentials{
					Uri: "https://D26EA3FB-AA78-451C-9ED0-233935ED388F:superdupersecret@example.com:23362",
					UriParams: aiven.ServiceUriParams{
						Host:     "example.com",
						Port:     "23362",
						User:     "D26EA3FB-AA78-451C-9ED0-233935ED388F",
						Password: "superdupersecret",
					},
				},
			}
			Expect(actualBinding).To(Equal(expectedBinding))
		})

		It("errors if the client fails to create the service user", func() {
			bindData := provider.BindData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "D26EA3FB-AA78-451C-9ED0-233935ED388F",
			}
			fakeAivenClient.CreateServiceUserReturnsOnCall(0, "", errors.New("some-error"))

			_, err := aivenProvider.Bind(context.Background(), bindData)
			Expect(err).To(HaveOccurred())
		})

		It("errors if the client fails to get the service", func() {
			bindData := provider.BindData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "D26EA3FB-AA78-451C-9ED0-233935ED388F",
			}
			fakeAivenClient.GetServiceConnectionDetailsReturnsOnCall(0, "", "", errors.New("some-error"))

			_, err := aivenProvider.Bind(context.Background(), bindData)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Unbind", func() {
		It("passes the correct parameters to the Aiven client", func() {
			unbindData := provider.UnbindData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "D26EA3FB-AA78-451C-9ED0-233935ED388F",
			}
			err := aivenProvider.Unbind(context.Background(), unbindData)
			Expect(err).ToNot(HaveOccurred())

			expectedDeleteServiceUserParameters := &aiven.DeleteServiceUserInput{
				ServiceName: "env69bc39f8",
				Username:    unbindData.BindingID,
			}

			Expect(fakeAivenClient.DeleteServiceUserCallCount()).To(Equal(1))
			Expect(fakeAivenClient.DeleteServiceUserArgsForCall(0)).To(Equal(expectedDeleteServiceUserParameters))
		})

		It("errors if the client errors", func() {
			unbindData := provider.UnbindData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				BindingID:  "D26EA3FB-AA78-451C-9ED0-233935ED388F",
			}
			fakeAivenClient.DeleteServiceUserReturnsOnCall(0, "", errors.New("some-error"))

			err := aivenProvider.Unbind(context.Background(), unbindData)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Update", func() {
		It("should pass the correct parameters to the Aiven client", func() {
			updateData := provider.UpdateData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				Details: brokerapi.UpdateDetails{
					ServiceID:      "uuid-1",
					PlanID:         "uuid-3",
					PreviousValues: brokerapi.PreviousValues{PlanID: "uuid-2"},
				},
			}
			_, err := aivenProvider.Update(context.Background(), updateData)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAivenClient.UpdateServiceCallCount()).To(Equal(1))

			expectedParameters := &aiven.UpdateServiceInput{
				ServiceName: "env69bc39f8",
				Plan:        "startup-2",
			}
			Expect(fakeAivenClient.UpdateServiceArgsForCall(0)).To(Equal(expectedParameters))
		})

		It("should return an error if the client fails to update", func() {
			updateData := provider.UpdateData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				Details: brokerapi.UpdateDetails{
					ServiceID:      "uuid-1",
					PlanID:         "uuid-3",
					PreviousValues: brokerapi.PreviousValues{PlanID: "uuid-2"},
				},
			}
			fakeAivenClient.UpdateServiceReturnsOnCall(0, "", errors.New("some bad thing"))

			_, err := aivenProvider.Update(context.Background(), updateData)

			Expect(err).To(HaveOccurred())
			Expect(fakeAivenClient.UpdateServiceCallCount()).To(Equal(1))
		})
	})

	Describe("LastOperation", func() {
		It("should return succeeded when the service is running", func() {
			expectedGetServiceStatusParameters := &aiven.GetServiceInput{
				ServiceName: "env69bc39f8",
			}

			lastOperationData := provider.LastOperationData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}

			twoMinutesAgo := time.Now().Add(-1 * 2 * time.Minute)
			fakeAivenClient.GetServiceStatusReturnsOnCall(0, aiven.Running, twoMinutesAgo, nil)
			actualLastOperationState, description, err := aivenProvider.LastOperation(context.Background(), lastOperationData)

			Expect(err).ToNot(HaveOccurred())

			Expect(fakeAivenClient.GetServiceStatusArgsForCall(0)).To(Equal(expectedGetServiceStatusParameters))
			Expect(actualLastOperationState).To(Equal(brokerapi.Succeeded))
			Expect(description).To(Equal("Last operation succeeded"))
		})

		// After an update operation the API immediately reports the state as 'RUNNING', which
		// would cause the broker to think it has completed updating. It takes a few seconds for
		// it to report as 'REBUILDING'. We thought we could use the `plan` data from the API to check
		// for when it is running with the new plan, but unfortunately the API shows the new plan
		// immediately (even when it says it is 'RUNNING').
		Context("when the state is RUNNING, but the service has only just been updated", func() {
			It("should report it 'in progress' for up to 60 seconds after the updated time", func() {
				expectedGetServiceParameters := &aiven.GetServiceInput{
					ServiceName: "env69bc39f8",
				}

				lastOperationData := provider.LastOperationData{
					InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				}

				thirtySecondsAgo := time.Now().Add(-1 * 30 * time.Second)
				fakeAivenClient.GetServiceStatusReturnsOnCall(0, aiven.Running, thirtySecondsAgo, nil)
				actualLastOperationState, description, err := aivenProvider.LastOperation(context.Background(), lastOperationData)

				Expect(err).ToNot(HaveOccurred())
				Expect(fakeAivenClient.GetServiceStatusArgsForCall(0)).To(Equal(expectedGetServiceParameters))

				Expect(actualLastOperationState).To(Equal(brokerapi.InProgress))
				Expect(description).To(Equal("Preparing to apply update"))
			})
		})

		It("should return an error if the client fails to get service state", func() {
			lastOperationData := provider.LastOperationData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}

			twoMinutesAgo := time.Now().Add(-1 * 2 * time.Minute)
			fakeAivenClient.GetServiceStatusReturnsOnCall(0, aiven.Running, twoMinutesAgo, errors.New("some-error"))

			actualLastOperationState, description, err := aivenProvider.LastOperation(context.Background(), lastOperationData)

			Expect(err).To(MatchError("some-error"))
			Expect(actualLastOperationState).To(Equal(brokerapi.LastOperationState("")))
			Expect(description).To(Equal(""))
		})
	})
})
