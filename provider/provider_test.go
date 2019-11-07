package provider_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alphagov/paas-aiven-broker/provider"
	"github.com/alphagov/paas-aiven-broker/provider/aiven"
	"github.com/alphagov/paas-aiven-broker/provider/aiven/fakes"
	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Provider", func() {

	var (
		aivenProvider   *provider.AivenProvider
		fakeAivenClient *fakes.FakeClient
		config          *provider.Config
	)

	BeforeEach(func() {
		planSpecificConfig1 := provider.PlanSpecificConfig{}
		planSpecificConfig1.AivenPlan = "startup-1"
		planSpecificConfig1.ElasticsearchVersion = "6"

		planSpecificConfig2 := provider.PlanSpecificConfig{}
		planSpecificConfig2.AivenPlan = "startup-2"
		planSpecificConfig2.ElasticsearchVersion = "6"

		config = &provider.Config{
			Cloud:             "aws-eu-west-1",
			ServiceNamePrefix: "env",
			Catalog: provider.Catalog{
				Services: []provider.Service{
					{
						Service: brokerapi.Service{ID: "uuid-1"},
						Plans: []provider.Plan{
							{
								ServicePlan: brokerapi.ServicePlan{
									ID:   "uuid-2",
									Name: "elasticsearch",
								},
								PlanSpecificConfig: planSpecificConfig1,
							},
							{
								ServicePlan: brokerapi.ServicePlan{
									ID:   "uuid-3",
									Name: "elasticsearch",
								},
								PlanSpecificConfig: planSpecificConfig2,
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
		Context("passes the correct parameters to the Aiven client", func() {
			provisionData := provider.ProvisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				Service:    brokerapi.Service{ID: "uuid-1", Name: "elasticsearch"},
				Plan:       brokerapi.ServicePlan{ID: "uuid-2"},
			}
			It("includes ip whitelist", func() {
				os.Setenv("IP_WHITELIST", "1.2.3.4,5.6.7.8")
				_, _, err := aivenProvider.Provision(context.Background(), provisionData)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeAivenClient.CreateServiceCallCount()).To(Equal(1))

				userConfig := aiven.UserConfig{}
				userConfig.ElasticsearchVersion = "6"
				userConfig.IPFilter = []string{"1.2.3.4", "5.6.7.8"}

				expectedParameters := &aiven.CreateServiceInput{
					Cloud:       "aws-eu-west-1",
					Plan:        "startup-1",
					ServiceName: "env-09e1993e-62e2-4040-adf2-4d3ec741efe6",
					ServiceType: "elasticsearch",
					UserConfig:  userConfig,
				}
				Expect(fakeAivenClient.CreateServiceArgsForCall(0)).To(Equal(expectedParameters))
				os.Unsetenv("IP_WHITELIST")
			})
			It("excludes ip whitelist when not set", func() {
				os.Unsetenv("IP_WHITELIST")
				_, _, err := aivenProvider.Provision(context.Background(), provisionData)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeAivenClient.CreateServiceCallCount()).To(Equal(1))

				userConfig := aiven.UserConfig{}
				userConfig.ElasticsearchVersion = "6"
				userConfig.IPFilter = []string{}

				expectedParameters := &aiven.CreateServiceInput{
					Cloud:       "aws-eu-west-1",
					Plan:        "startup-1",
					ServiceName: "env-09e1993e-62e2-4040-adf2-4d3ec741efe6",
					ServiceType: "elasticsearch",
					UserConfig:  userConfig,
				}
				Expect(fakeAivenClient.CreateServiceArgsForCall(0)).To(Equal(expectedParameters))
			})
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
				ServiceName: "env-09e1993e-62e2-4040-adf2-4d3ec741efe6",
			}
			Expect(fakeAivenClient.DeleteServiceArgsForCall(0)).To(Equal(expectedParameters))
		})

		It("errors if the client errors", func() {
			deprovisionData := provider.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			fakeAivenClient.DeleteServiceReturnsOnCall(0, errors.New("some-error"))

			_, err := aivenProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).To(HaveOccurred())
		})

		It("Returns a specific error if the instance has already been deleted", func() {
			deprovisionData := provider.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			fakeAivenClient.DeleteServiceReturnsOnCall(0, aiven.ErrInstanceDoesNotExist)

			_, err := aivenProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
		})
	})

	Describe("Bind", func() {
		const (
			testInstanceID = "09E1993E-62E2-4040-ADF2-4D3EC741EFE6"
			testBindingID  = "D26EA3FB-AA78-451C-9ED0-233935ED388F"
			stubPassword   = "superdupersecret"
		)
		var (
			testESServer    *ghttp.Server
			testESHost      string
			testESPort      string
			versionResponse http.HandlerFunc
			bindData        provider.BindData
			bindCtx         context.Context
			bindCancel      context.CancelFunc
		)

		BeforeEach(func() {
			testESServer = ghttp.NewTLSServer()
			http.DefaultClient = testESServer.HTTPTestServer.Client()
			versionResponse = ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/"),
				ghttp.VerifyBasicAuth(testBindingID, stubPassword),
				ghttp.RespondWith(200, `{"version":{"number":"1.2.3"}}`),
			)
			testESServer.AppendHandlers(versionResponse)

			esURL, err := url.Parse(testESServer.URL())
			Expect(err).NotTo(HaveOccurred())
			parts := strings.SplitN(esURL.Host, ":", 2)
			Expect(parts).To(HaveLen(2))
			testESHost, testESPort = parts[0], parts[1]

			fakeAivenClient.CreateServiceUserReturnsOnCall(0, stubPassword, nil)
			fakeAivenClient.GetServiceConnectionDetailsReturnsOnCall(0, testESHost, testESPort, nil)

			bindCtx, bindCancel = context.WithTimeout(context.Background(), 5*time.Second)
			bindData = provider.BindData{
				InstanceID: testInstanceID,
				BindingID:  testBindingID,
			}
		})

		AfterEach(func() {
			if testESServer != nil {
				testESServer.Close()
			}
			bindCancel()
		})

		It("passes the correct parameters to the Aiven client", func() {
			actualBinding, err := aivenProvider.Bind(bindCtx, bindData)
			Expect(err).ToNot(HaveOccurred())

			expectedCreateServiceUserParameters := &aiven.CreateServiceUserInput{
				ServiceName: "env-" + strings.ToLower(testInstanceID),
				Username:    testBindingID,
			}
			Expect(fakeAivenClient.CreateServiceUserArgsForCall(0)).To(Equal(expectedCreateServiceUserParameters))

			expectedGetServiceConnectionDetailsParameters := &aiven.GetServiceInput{
				ServiceName: "env-" + strings.ToLower(testInstanceID),
			}
			Expect(fakeAivenClient.GetServiceConnectionDetailsArgsForCall(0)).To(Equal(expectedGetServiceConnectionDetailsParameters))

			expectedBinding := brokerapi.Binding{
				Credentials: provider.Credentials{
					URI:      fmt.Sprintf("https://%s:%s@%s:%s", testBindingID, stubPassword, testESHost, testESPort),
					Hostname: testESHost,
					Port:     testESPort,
					Username: testBindingID,
					Password: stubPassword,
				},
			}
			Expect(actualBinding).To(Equal(expectedBinding))
		})

		It("errors if the client fails to create the service user", func() {
			fakeAivenClient.CreateServiceUserReturnsOnCall(0, "", errors.New("some-error"))

			_, err := aivenProvider.Bind(bindCtx, bindData)
			Expect(err).To(HaveOccurred())
		})

		It("errors if the client fails to get the service", func() {
			fakeAivenClient.GetServiceConnectionDetailsReturnsOnCall(0, "", "", errors.New("some-error"))

			_, err := aivenProvider.Bind(bindCtx, bindData)
			Expect(err).To(HaveOccurred())
		})

		Describe("polling ES until the credentials work", func() {
			var (
				unauthorizedResponse http.HandlerFunc
			)

			BeforeEach(func() {
				unauthorizedResponse = ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.VerifyBasicAuth(testBindingID, stubPassword),
					ghttp.RespondWith(401, `<html><body><h1>401 Unauthorized</h1>You need a valid user and password to access this content.</body></html>`),
				)
				// Replace the handler set above
				testESServer.SetHandler(0, unauthorizedResponse)
			})

			It("retries calling the ES server if it gets a 401 initially", func() {
				testESServer.AppendHandlers(versionResponse)
				_, err := aivenProvider.Bind(bindCtx, bindData)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeAivenClient.DeleteServiceUserCallCount()).To(Equal(0))
			})

			It("gives up polling and returns anyway after a timeout", func() {
				testESServer.AppendHandlers(unauthorizedResponse)

				ctx, cancel := context.WithTimeout(bindCtx, 900*time.Millisecond)
				defer cancel()

				_, err := aivenProvider.Bind(ctx, bindData)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns any other errors encountered while polling", func() {
				testESServer.AppendHandlers(unauthorizedResponse)

				ctx, cancel := context.WithCancel(bindCtx)
				cancel() // cancelling to ensure a non-timeout error is triggered

				_, err := aivenProvider.Bind(ctx, bindData)
				Expect(err).To(HaveOccurred())
			})
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
				ServiceName: "env-09e1993e-62e2-4040-adf2-4d3ec741efe6",
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
			os.Setenv("IP_WHITELIST", "1.2.3.4,5.6.7.8")
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

			userConfig := aiven.UserConfig{}
			userConfig.ElasticsearchVersion = "6"
			userConfig.IPFilter = []string{"1.2.3.4", "5.6.7.8"}

			expectedParameters := &aiven.UpdateServiceInput{
				ServiceName: "env-09e1993e-62e2-4040-adf2-4d3ec741efe6",
				Plan:        "startup-2",
				UserConfig:  userConfig,
			}
			Expect(fakeAivenClient.UpdateServiceArgsForCall(0)).To(Equal(expectedParameters))
		})

		It("should return an error if the client returns error", func() {
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

		It("should returns StatusUnprocessableEntity (422) if the client returns invalid update error", func() {
			updateData := provider.UpdateData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				Details: brokerapi.UpdateDetails{
					ServiceID:      "uuid-1",
					PlanID:         "uuid-3",
					PreviousValues: brokerapi.PreviousValues{PlanID: "uuid-2"},
				},
			}
			fakeAivenClient.UpdateServiceReturnsOnCall(0, "", aiven.ErrInvalidUpdate{"not-valid"})

			_, err := aivenProvider.Update(context.Background(), updateData)

			expectedErr := brokerapi.NewFailureResponseBuilder(
				aiven.ErrInvalidUpdate{"not-valid"},
				http.StatusUnprocessableEntity,
				"plan-change-not-supported",
			).WithErrorKey("PlanChangeNotSupported").Build()

			Expect(err).To(MatchError(expectedErr))
			Expect(fakeAivenClient.UpdateServiceCallCount()).To(Equal(1))
		})
	})

	Describe("LastOperation", func() {
		It("should return succeeded when the service is running", func() {
			expectedGetServiceStatusParameters := &aiven.GetServiceInput{
				ServiceName: "env-09e1993e-62e2-4040-adf2-4d3ec741efe6",
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
					ServiceName: "env-09e1993e-62e2-4040-adf2-4d3ec741efe6",
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

	Describe("The ParseIPWhitelist function", func() {
		It("parses an empty string as an empty list", func() {
			Expect(provider.ParseIPWhitelist("")).
				To(BeEmpty())
		})

		It("parses a single IP", func() {
			Expect(provider.ParseIPWhitelist("127.0.0.1")).
				To(Equal([]string{"127.0.0.1"}))
		})

		It("parses multiple IPs", func() {
			Expect(provider.ParseIPWhitelist("127.0.0.1,99.99.99.99")).
				To(Equal([]string{"127.0.0.1", "99.99.99.99"}))
		})

		It("returns an error for IPs containing the wrong number of octets", func() {
			var err error
			By("not permitting too many octets")
			_, err = provider.ParseIPWhitelist("127.0.0.0.1")
			Expect(err).To(HaveOccurred())
			By("not permitting too few octets")
			_, err = provider.ParseIPWhitelist("127.0.1")
			Expect(err).To(HaveOccurred())
			By("not permitting too few octets even when valid IPs are present")
			_, err = provider.ParseIPWhitelist("8.8.8.8,127.0.1")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error for garbage IPs", func() {
			_, err := provider.ParseIPWhitelist("ojnratuh53ggijntboijngk3,0ij90490ti9jo43p;';;1;'")
			Expect(err).To(HaveOccurred())
		})

	})
})
