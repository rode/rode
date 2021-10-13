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

package v1alpha1_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/proto/v1alpha1"
	. "github.com/rode/rode/test/util"
	"google.golang.org/grpc/codes"
)

const (
	minimalValidPolicy = `
package minimal

pass {
    true
}

violations[result] {
	result = {
		"pass": true,
		"id": "valid",
		"name": "name",
		"description": "description",
		"message": "message",
	}
}
`
	policyUpdates = `
violations[result] {
	result = {
		"pass": true,
		"id": "valid",
		"name": "name",
		"description": "description",
		"message": "message",
	}
}
`
)

var _ = Describe("Policies", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Creating a policy", func() {
		When("the policy is valid", func() {
			It("should be created", func() {
				expectedPolicy := randomPolicy()
				newPolicy, err := rode.CreatePolicy(ctx, expectedPolicy)
				Expect(err).NotTo(HaveOccurred())

				createdPolicy, err := rode.GetPolicy(ctx, &v1alpha1.GetPolicyRequest{
					Id: newPolicy.Id,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createdPolicy.Name).To(Equal(expectedPolicy.Name))
				Expect(createdPolicy.Description).To(Equal(expectedPolicy.Description))
				Expect(createdPolicy.CurrentVersion).To(Equal(uint32(1)))
				Expect(createdPolicy.Policy.Version).To(Equal(uint32(1)))
			})
		})

		When("the policy name is not present", func() {
			It("should return an error", func() {
				policy := randomPolicy()
				policy.Name = ""

				_, err := rode.CreatePolicy(ctx, policy)
				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy is not provided", func() {
			It("should return an error", func() {
				policy := randomPolicy()
				policy.Policy = nil

				_, err := rode.CreatePolicy(ctx, policy)
				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy is invalid", func() {
			It("should return an error", func() {
				policy := randomPolicy()
				policy.Policy.RegoContent = ""

				_, err := rode.CreatePolicy(ctx, policy)
				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).CreatePolicy(ctx, randomPolicy())

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest("PolicyDeveloper", "PolicyAdministrator", "Administrator")...,
		)
	})

	Describe("Deleting a policy", func() {
		When("the policy exists", func() {
			It("should be deleted successfully", func() {
				policy, err := rode.CreatePolicy(ctx, randomPolicy())
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicy(ctx, &v1alpha1.DeletePolicyRequest{
					Id: policy.Id,
				})
				Expect(err).NotTo(HaveOccurred())

				actualPolicy, err := rode.GetPolicy(ctx, &v1alpha1.GetPolicyRequest{
					Id: policy.Id,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(actualPolicy.Deleted).To(BeTrue())
			})
		})

		When("the policy doesn't exist", func() {
			It("should return an error", func() {
				_, err := rode.DeletePolicy(ctx, &v1alpha1.DeletePolicyRequest{
					Id: fake.UUID(),
				})

				Expect(err).To(HaveGrpcStatus(codes.NotFound))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			policy, err := rode.CreatePolicy(ctx, randomPolicy())
			Expect(err).NotTo(HaveOccurred())

			_, err = rode.WithRole(entry.Role).DeletePolicy(ctx, &v1alpha1.DeletePolicyRequest{
				Id: policy.Id,
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest("PolicyAdministrator", "Administrator")...,
		)
	})

	Describe("Updating a policy", func() {
		When("the changes are valid", func() {
			It("should be successful", func() {
				policy, err := rode.CreatePolicy(ctx, randomPolicy())
				Expect(err).NotTo(HaveOccurred())
				expectedDescription := fake.LetterN(20)
				policy.Description = expectedDescription

				updatedPolicy, err := rode.UpdatePolicy(ctx, &v1alpha1.UpdatePolicyRequest{
					Policy: policy,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(updatedPolicy.Description).To(Equal(expectedDescription))
				Expect(updatedPolicy.CurrentVersion).To(Equal(uint32(1)))
			})
		})

		When("the policy content changes", func() {
			It("should update the version", func() {
				policy, err := rode.CreatePolicy(ctx, randomPolicy())
				Expect(err).NotTo(HaveOccurred())
				policy.Policy.RegoContent += policyUpdates

				updatedPolicy, err := rode.UpdatePolicy(ctx, &v1alpha1.UpdatePolicyRequest{
					Policy: policy,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(updatedPolicy.CurrentVersion).To(Equal(uint32(2)))
				Expect(updatedPolicy.Policy.Version).To(Equal(uint32(2)))
			})
		})

		When("the policy content updates are invalid", func() {
			It("should return an error", func() {
				policy, err := rode.CreatePolicy(ctx, randomPolicy())
				Expect(err).NotTo(HaveOccurred())
				policy.Policy.RegoContent = "package invalid"

				_, err = rode.UpdatePolicy(ctx, &v1alpha1.UpdatePolicyRequest{
					Policy: policy,
				})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy doesn't exist", func() {
			It("should return an error", func() {
				_, err := rode.UpdatePolicy(ctx, &v1alpha1.UpdatePolicyRequest{
					Policy: &v1alpha1.Policy{
						Id: fake.UUID(),
					},
				})

				Expect(err).To(HaveGrpcStatus(codes.NotFound))
			})
		})

		When("the policy has been deleted", func() {
			It("should return an error", func() {
				policy, err := rode.CreatePolicy(ctx, randomPolicy())
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicy(ctx, &v1alpha1.DeletePolicyRequest{
					Id: policy.Id,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.UpdatePolicy(ctx, &v1alpha1.UpdatePolicyRequest{
					Policy: policy,
				})

				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			policy, err := rode.CreatePolicy(ctx, randomPolicy())
			Expect(err).NotTo(HaveOccurred())

			policy.Name = fake.LetterN(10)
			_, err = rode.WithRole(entry.Role).UpdatePolicy(ctx, &v1alpha1.UpdatePolicyRequest{
				Policy: policy,
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest("PolicyDeveloper", "PolicyAdministrator", "Administrator")...,
		)
	})

	Describe("Validating a policy", func() {
		When("the policy is valid", func() {
			It("should be successful", func() {
				validation, err := rode.ValidatePolicy(ctx, &v1alpha1.ValidatePolicyRequest{
					Policy: minimalValidPolicy,
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(validation.Compile).To(BeTrue())
				Expect(validation.Errors).To(BeEmpty())
			})
		})

		When("no policy is given", func() {
			It("should return an error", func() {
				_, err := rode.ValidatePolicy(ctx, &v1alpha1.ValidatePolicyRequest{})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy is incomplete", func() {
			It("should return an error", func() {
				_, err := rode.ValidatePolicy(ctx, &v1alpha1.ValidatePolicyRequest{
					Policy: "package incomplete",
				})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).ValidatePolicy(ctx, &v1alpha1.ValidatePolicyRequest{
				Policy: minimalValidPolicy,
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest(
				"ApplicationDeveloper",
				"PolicyDeveloper",
				"PolicyAdministrator",
				"Administrator",
			)...,
		)
	})
})

func randomPolicy() *v1alpha1.Policy {
	return &v1alpha1.Policy{
		Name:        fake.LetterN(10),
		Description: fake.LetterN(20),
		Policy: &v1alpha1.PolicyEntity{
			RegoContent: minimalValidPolicy,
		},
	}
}
