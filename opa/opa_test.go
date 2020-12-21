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

	})

	When("a new OPA policy is intialized", func() {
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

	When("an OPA policy is evaluated", func() {
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

			opaResult, expectedErr = Opa.EvaluatePolicy(opaPolicy, input)
		})

		It("should call OPA data endpoint", func() {
			Expect(httpmock.GetTotalCallCount()).To(Equal(1))
		})

		It("should return OPA evaluation results", func() {
			Expect(expectedErr).To(Not(HaveOccurred()))
			Expect(opaResult).To(Equal(opaResponse.Result))
		})
	})
})
