package opa

import (
	"encoding/json"
	"errors"
	"fmt"

	"net/http"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("opa client", func() {
	var (
		Opa       Client
		opaPolicy string
	)

	BeforeEach(func() {
		Opa = Client{
			logger: logger,
			Host:   fmt.Sprintf("http://%s", gofakeit.DomainName()),
		}
		opaPolicy = gofakeit.Word()
	})

	When("a new OPA client is created", func() {
		It("returns OpaClient", func() {
			host := fmt.Sprintf("http://%s", gofakeit.DomainName())
			opa := NewClient(logger, host)
			Expect(opa).To(BeAssignableToTypeOf(&Client{}))
			Expect(opa.Host).To((Equal(host)))
		})
	})

	Context("an OPA policy is intialized", func() {
		var (
			initializePolicyError ClientError
			getPolicyResponse     *http.Response
			getPolicyError        error
		)

		BeforeEach(func() {
			getPolicyResponse = httpmock.NewStringResponse(200, "{}")
			getPolicyError = nil

			httpmock.RegisterResponder("GET", fmt.Sprintf("%s/v1/policies/%s", Opa.Host, opaPolicy),
				func(req *http.Request) (*http.Response, error) {
					return getPolicyResponse, getPolicyError
				},
			)
		})

		JustBeforeEach(func() {
			initializePolicyError = Opa.InitializePolicy(opaPolicy)
		})

		It("should check if policy exists in OPA", func() {
			Expect(httpmock.GetTotalCallCount()).To(Equal(1))
		})

		When("fetch policy request returns error", func() {
			BeforeEach(func() {
				getPolicyResponse = nil
				getPolicyError = errors.New("error connecting to host")
			})

			It("should return an http request error", func() {
				Expect(initializePolicyError).To(HaveOccurred())
				Expect(initializePolicyError.Type()).To(Equal(OpaClientErrorTypePolicyExists))
				Expect(initializePolicyError.CausedBy()).To(HaveOccurred())
				Expect(initializePolicyError.CausedBy()).To(BeAssignableToTypeOf(clientError{}))
				Expect(initializePolicyError.CausedBy().(clientError).Type()).To(Equal(OpaClientErrorTypeHTTP))
			})
		})

		When("fetch policy response status is not OK", func() {
			BeforeEach(func() {
				getPolicyResponse = httpmock.NewStringResponse(http.StatusInternalServerError, "OPA Error")
			})

			It("should return an bad response error", func() {
				Expect(initializePolicyError).To(HaveOccurred())
				Expect(initializePolicyError.Type()).To(Equal(OpaClientErrorTypePolicyExists))
				Expect(initializePolicyError.CausedBy()).To(HaveOccurred())
				Expect(initializePolicyError.CausedBy()).To(BeAssignableToTypeOf(clientError{}))
				Expect(initializePolicyError.CausedBy().(clientError).Type()).To(Equal(OpaClientErrorTypeBadResponse))
			})
		})

		When("policy exists", func() {
			It("should do nothing", func() {
				Expect(initializePolicyError).NotTo(HaveOccurred())
			})
		})

		When("policy does not exist", func() {
			BeforeEach(func() {
				getPolicyResponse = httpmock.NewStringResponse(404, "{}")
			})

			It("should fetch policy violations from Elasticsearch", func() {
			})

			When("policy violations are returned from elasticsearch", func() {
				var (
					createPolicyResponse *http.Response
					createPolicyError    error
				)

				BeforeEach(func() {
					createPolicyResponse = httpmock.NewStringResponse(200, "{}")
					createPolicyError = nil

					httpmock.RegisterResponder(http.MethodPut, fmt.Sprintf("%s/v1/policies/%s", Opa.Host, opaPolicy),
						func(req *http.Request) (*http.Response, error) {
							// todo assert http put body contains policy package
							return createPolicyResponse, createPolicyError
						},
					)
				})

				JustBeforeEach(func() {
				})

				It("publishes policy to OPA", func() {
					Expect(httpmock.GetTotalCallCount()).To(Equal(2))
				})

				When("publish OPA policy request returns error", func() {
					BeforeEach(func() {
						createPolicyResponse = nil
						createPolicyError = errors.New("error connecting to host")
					})

					It("should return a publish policy http error", func() {
						Expect(initializePolicyError).To(HaveOccurred())
						Expect(initializePolicyError.Type()).To(Equal(OpaClientErrorTypePublishPolicy))
						Expect(initializePolicyError.CausedBy()).To(HaveOccurred())
						Expect(initializePolicyError.CausedBy()).To(BeAssignableToTypeOf(clientError{}))
						Expect(initializePolicyError.CausedBy().(clientError).Type()).To(Equal(OpaClientErrorTypeHTTP))
					})
				})

				When("publish OPA policy status response is not OK", func() {
					BeforeEach(func() {
						createPolicyResponse = httpmock.NewStringResponse(http.StatusInternalServerError, "OPA error")
					})

					It("should return a publish policy bad response error", func() {
						Expect(initializePolicyError).To(HaveOccurred())
						Expect(initializePolicyError.Type()).To(Equal(OpaClientErrorTypePublishPolicy))
						Expect(initializePolicyError.CausedBy()).To(HaveOccurred())
						Expect(initializePolicyError.CausedBy()).To(BeAssignableToTypeOf(clientError{}))
						Expect(initializePolicyError.CausedBy().(clientError).Type()).To(Equal(OpaClientErrorTypeBadResponse))
					})
				})

			})
		})
	})

	Context("an OPA policy is evaluated", func() {
		var (
			input              string
			opaResponse        *EvaluatePolicyResponse
			expectedOpaRequest *EvalutePolicyRequest
			opaResult          *EvaluatePolicyResult
			expectedErr        error
		)

		BeforeEach(func() {
			opaPolicy = gofakeit.Word()
		})

		JustBeforeEach(func() {
			opaResult, expectedErr = Opa.EvaluatePolicy(opaPolicy, input)
		})

		When("OPA returns a valid response", func() {
			BeforeEach(func() {
				input = fmt.Sprintf(`{"%s":"%s"}`, gofakeit.Word(), gofakeit.Word())
				opaResponse = &EvaluatePolicyResponse{
					Result: &EvaluatePolicyResult{
						Pass: gofakeit.Bool(),
						Violations: []*EvaluatePolicyViolation{
							{
								Message: gofakeit.Paragraph(1, 1, 10, "."),
							},
						},
					},
				}

				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", Opa.Host, opaPolicy),
					func(req *http.Request) (*http.Response, error) {
						err := json.NewDecoder(req.Body).Decode(&expectedOpaRequest)
						Expect(err).To(Not(HaveOccurred()))
						Expect(expectedOpaRequest.Input).To(Equal(json.RawMessage(input)))

						return httpmock.NewJsonResponse(200, &opaResponse)
					},
				)

			})

			It("should call OPA data endpoint", func() {
				Expect(httpmock.GetTotalCallCount()).To(Equal(1))
			})

			It("should return OPA evaluation results", func() {
				Expect(expectedErr).ToNot(HaveOccurred())
				Expect(opaResult).To(Equal(opaResponse.Result))
			})
		})

		When("OPA returns an invalid status code", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", Opa.Host, opaPolicy),
					func(req *http.Request) (*http.Response, error) {
						return httpmock.NewStringResponse(500, "OPA error"), nil
					},
				)
			})

			It("should return a http status error", func() {
				Expect(expectedErr).To(HaveOccurred())
				Expect(expectedErr.Error()).To(ContainSubstring("http response status not OK"))
				Expect(opaResult).To(BeNil())
			})
		})

		When("there is a failure in the http request", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", Opa.Host, opaPolicy),
					func(req *http.Request) (*http.Response, error) {
						return nil, fmt.Errorf("HTTP POST failed")
					},
				)
			})

			It("should return an error", func() {
				Expect(expectedErr).To(HaveOccurred())
				Expect(expectedErr.Error()).To(ContainSubstring("http request to OPA failed"))
				Expect(opaResult).To(BeNil())
			})
		})

		When("response from OPA fails to decode", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", Opa.Host, opaPolicy),
					func(req *http.Request) (*http.Response, error) {
						return httpmock.NewStringResponse(200, `{"result":{"pass":"foo"}}`), nil
					},
				)
			})

			It("should return a response decode error", func() {
				Expect(expectedErr).To(HaveOccurred())
				Expect(expectedErr.Error()).To(ContainSubstring("failed to decode OPA result"))
				Expect(opaResult).To(BeNil())
			})
		})

		When("invalid input data is provided to the client", func() {
			BeforeEach(func() {
				input = gofakeit.Word()
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", Opa.Host, opaPolicy), nil)
			})

			It("should return an error", func() {
				Expect(httpmock.GetTotalCallCount()).To(Equal(0))
				Expect(expectedErr.Error()).To(ContainSubstring("failed to encode OPA input"))
				Expect(expectedErr).To(HaveOccurred())
			})
		})
	})
})
