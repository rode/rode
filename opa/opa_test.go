// Copyright 2021 The Rode Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opa

import (
	"encoding/json"
	"errors"
	"fmt"
	pb "github.com/rode/rode/proto/v1alpha1"

	"net/http"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("opa client", func() {
	var (
		Opa     Client
		opaHost string
	)
	const (
		compilablePolicyMissingRodeFields = `
		package play
		default hello = false
		hello {
			m := input.message
			m == "world"
		}`
		opaPolicy = "play"
	)

	BeforeEach(func() {
		opaHost = fmt.Sprintf("http://%s", fake.DomainName())

		Opa = &client{
			logger,
			opaHost,
			false,
			&http.Client{},
		}

	})

	When("a new OPA client is created", func() {
		It("returns OpaClient", func() {
			host := fmt.Sprintf("http://%s", fake.DomainName())
			opa := NewClient(logger, host, false)
			Expect(opa).To(BeAssignableToTypeOf(&client{}))
			Expect(opa.(*client).Host).To(Equal(host))
		})
	})

	Context("an OPA policy is initialized", func() {
		var (
			initializePolicyError ClientError
			getPolicyResponse     *http.Response
			getPolicyError        error
		)

		BeforeEach(func() {
			getPolicyResponse = httpmock.NewStringResponse(200, "{}")
			getPolicyError = nil

			httpmock.RegisterResponder("GET", fmt.Sprintf("%s/v1/policies/%s", opaHost, opaPolicy),
				func(req *http.Request) (*http.Response, error) {
					return getPolicyResponse, getPolicyError
				},
			)
		})

		JustBeforeEach(func() {
			initializePolicyError = Opa.InitializePolicy(opaPolicy, fake.LetterN(200))
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
				Expect(initializePolicyError).To(BeAssignableToTypeOf(clientError{}))
				Expect(initializePolicyError.Type()).To(Equal(OpaClientErrorTypeGetPolicy))
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
				Expect(initializePolicyError.Type()).To(Equal(OpaClientErrorTypeGetPolicy))
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

					httpmock.RegisterResponder(http.MethodPut, fmt.Sprintf("%s/v1/policies/%s", opaHost, opaPolicy),
						func(req *http.Request) (*http.Response, error) {
							// todo assert http put body contains policy package
							return createPolicyResponse, createPolicyError
						},
					)
				})

				XIt("publishes policy to OPA", func() {
					Expect(httpmock.GetTotalCallCount()).To(Equal(2))
				})

				When("publish OPA policy request returns error", func() {
					BeforeEach(func() {
						createPolicyResponse = nil
						createPolicyError = errors.New("error connecting to host")
					})

					XIt("should return a publish policy http error", func() {
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

					XIt("should return a publish policy bad response error", func() {
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
			input                 []byte
			opaResponse           *EvaluatePolicyResponse
			expectedOpaRequest    *EvalutePolicyRequest
			evalutePolicyResponse *EvaluatePolicyResponse
			expectedErr           error
		)

		JustBeforeEach(func() {
			evalutePolicyResponse, expectedErr = Opa.EvaluatePolicy(compilablePolicyMissingRodeFields, input)
		})

		When("OPA returns a valid response", func() {
			BeforeEach(func() {
				input = []byte(fmt.Sprintf(`{"%s":"%s"}`, fake.Word(), fake.Word()))
				opaResponse = &EvaluatePolicyResponse{
					Result: &EvaluatePolicyResult{
						Pass: fake.Bool(),
						Violations: []*pb.EvaluatePolicyViolation{
							{
								Message: fake.Paragraph(1, 1, 10, "."),
							},
						},
					},
				}

				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", opaHost, opaPolicy),
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

			It("should return OPA evaluation response", func() {
				Expect(expectedErr).ToNot(HaveOccurred())
				Expect(evalutePolicyResponse).To(Equal(opaResponse))
			})
		})

		When("OPA returns an invalid status code", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", opaHost, opaPolicy),
					func(req *http.Request) (*http.Response, error) {
						return httpmock.NewStringResponse(500, "OPA error"), nil
					},
				)
			})

			It("should return a http status error", func() {
				Expect(expectedErr).To(HaveOccurred())
				Expect(expectedErr.Error()).To(ContainSubstring("http response status not OK"))
				Expect(evalutePolicyResponse).To(BeNil())
			})
		})

		When("there is a failure in the http request", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", opaHost, opaPolicy),
					func(req *http.Request) (*http.Response, error) {
						return nil, fmt.Errorf("HTTP POST failed")
					},
				)
			})

			It("should return an error", func() {
				Expect(expectedErr).To(HaveOccurred())
				Expect(expectedErr.Error()).To(ContainSubstring("http request to OPA failed"))
				Expect(evalutePolicyResponse).To(BeNil())
			})
		})

		When("response from OPA fails to decode", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", opaHost, opaPolicy),
					func(req *http.Request) (*http.Response, error) {
						return httpmock.NewStringResponse(200, `{"result":{"pass":"foo"}}`), nil
					},
				)
			})

			It("should return a response decode error", func() {
				Expect(expectedErr).To(HaveOccurred())
				Expect(expectedErr.Error()).To(ContainSubstring("failed to decode OPA result"))
				Expect(evalutePolicyResponse).To(BeNil())
			})
		})

		When("invalid input data is provided to the client", func() {
			BeforeEach(func() {
				input = []byte(fake.Word())
				httpmock.RegisterResponder("POST", fmt.Sprintf("%s/v1/data/%s", opaHost, opaPolicy), nil)
			})

			It("should return an error", func() {
				Expect(httpmock.GetTotalCallCount()).To(Equal(0))
				Expect(expectedErr.Error()).To(ContainSubstring("failed to encode OPA input"))
				Expect(expectedErr).To(HaveOccurred())
			})
		})
	})
})
