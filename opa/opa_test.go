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
	"context"
	_ "embed"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	//go:embed test/valid.rego
	validPolicy string
	//go:embed test/fail.rego
	failingPolicy string
	//go:embed test/results.rego
	invalidResults string
)

var _ = Describe("OPA client", func() {
	var (
		opaClient Client
		ctx       context.Context

		policyId string
	)

	BeforeEach(func() {
		ctx = context.Background()
		policyId = fake.UUID()

		opaClient = NewClient(logger)
	})

	Context("InitializePolicy", func() {
		var (
			actualError error
			policy      string
		)

		BeforeEach(func() {
			policy = validPolicy
		})

		JustBeforeEach(func() {
			actualError = opaClient.InitializePolicy(ctx, policyId, policy)
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("the Rego code is invalid", func() {
			BeforeEach(func() {
				policy = fake.Word()
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the parsed query is already in the cache", func() {
			BeforeEach(func() {
				Expect(opaClient.InitializePolicy(ctx, policyId, policy)).NotTo(HaveOccurred())
			})

			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})
	})

	Context("EvaluatePolicy", func() {
		var (
			actualResult *EvaluatePolicyResult
			actualError  error
			input        interface{}
		)

		BeforeEach(func() {
			input = map[string]interface{}{}
		})

		var initializePolicy = func(policy string) {
			Expect(opaClient.InitializePolicy(ctx, policyId, policy)).NotTo(HaveOccurred())
		}

		JustBeforeEach(func() {
			actualResult, actualError = opaClient.EvaluatePolicy(ctx, policyId, input)
		})

		When("the policy query passes", func() {
			BeforeEach(func() {
				initializePolicy(validPolicy)
			})

			It("should return the successful evaluation result", func() {
				Expect(actualError).NotTo(HaveOccurred())
				Expect(actualResult.Pass).To(BeTrue())
			})
		})

		When("the policy evaluation fails", func() {
			BeforeEach(func() {
				initializePolicy(failingPolicy)
			})

			It("should indicate the inputs failed the policy", func() {
				Expect(actualError).NotTo(HaveOccurred())
				Expect(actualResult.Pass).To(BeFalse())
			})
		})

		When("the input is invalid", func() {
			BeforeEach(func() {
				input = newInvalidInput()
				initializePolicy(validPolicy)
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the policy results don't match the expected format", func() {
			BeforeEach(func() {
				initializePolicy(invalidResults)
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the policy hasn't been initialized", func() {
			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})
	})
})

type invalidOpaInput struct {
	Field *invalidOpaInput
}

// OPA marshals and unmarshals inputs as JSON, so the only inputs that raise are ones that can't be represented as JSON.
// Cyclic data structures are an example
func newInvalidInput() *invalidOpaInput {
	input := &invalidOpaInput{}
	input.Field = input

	return input
}
