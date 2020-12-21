package opa

import (
	"encoding/json"
	"fmt"

	"net/http"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("opa client", func() {
	var (
		Opa OpaClient
	)

	BeforeEach(func() {
		Opa = OpaClient{
			logger: logger,
			Host:   fmt.Sprintf("http://%s", gofakeit.DomainName()),
		}
	})

	When("a new OPA client is created", func() {
		It("returns OpaClient", func() {
			host := fmt.Sprintf("http://%s", gofakeit.DomainName())
			opa := NewOPAClient(logger, host)
			Expect(opa).To(BeAssignableToTypeOf(&OpaClient{}))
			Expect(opa.Host).To((Equal(host)))
		})
	})

	Context("a new OPA policy is intialized", func() {
		It("should fetch policy from OPA", func() {

		})

		When("policy exists", func() {
			It("should do nothing", func() {

			})
		})

		When("policy does not exist", func() {
			It("should fetch rules from Elasticsearch", func() {

			})
		})
	})

	Context("an OPA policy is evaluated", func() {
		var (
			opaPolicy          string
			input              string
			opaResponse        *OpaEvaluatePolicyResponse
			expectedOpaRequest *OpaEvalutePolicyRequest
			opaResult          *OpaEvaluatePolicyResult
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
				opaResponse = &OpaEvaluatePolicyResponse{
					Result: &OpaEvaluatePolicyResult{
						Pass: gofakeit.Bool(),
						Violations: []*OpaEvaluatePolicyViolation{
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
						// return nil, fmt.Errorf("HTTP error")
					},
				)
			})
			It("should return an error", func() {
				Expect(expectedErr).To(HaveOccurred())
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
				Expect(expectedErr).To(HaveOccurred())
			})
		})
	})
})
