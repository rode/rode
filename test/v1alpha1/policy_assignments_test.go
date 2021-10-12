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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/proto/v1alpha1"
	. "github.com/rode/rode/test/util"
	"google.golang.org/grpc/codes"
)

var _ = Describe("Policy Assignments", func() {
	var (
		ctx                    = context.Background()
		policyGroup            string
		randomPolicyAssignment = func(policyVersionId string) *v1alpha1.PolicyAssignment {
			return &v1alpha1.PolicyAssignment{
				PolicyGroup:     policyGroup,
				PolicyVersionId: policyVersionId,
			}
		}
		createPolicy = func() (string, *v1alpha1.Policy) {
			policy := randomPolicy()

			createdPolicy, err := rode.CreatePolicy(ctx, policy)
			Expect(err).NotTo(HaveOccurred())

			return createdPolicy.Policy.Id, createdPolicy
		}
	)

	BeforeSuite(func() {
		group := randomPolicyGroup()
		policyGroup = group.Name

		_, err := rode.CreatePolicyGroup(ctx, group)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Creating a policy assignment", func() {
		When("the policy and group both exist", func() {
			It("should create the assignment", func() {
				policyVersionId, _ := createPolicy()
				expectedAssignment := randomPolicyAssignment(policyVersionId)

				createdAssignment, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)
				Expect(err).NotTo(HaveOccurred())

				assignment, err := rode.GetPolicyAssignment(ctx, &v1alpha1.GetPolicyAssignmentRequest{
					Id: createdAssignment.Id,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(assignment.PolicyGroup).To(Equal(expectedAssignment.PolicyGroup))
				Expect(assignment.PolicyVersionId).To(Equal(expectedAssignment.PolicyVersionId))
			})
		})

		When("the policy group is not set", func() {
			It("should return an error", func() {
				expectedAssignment := randomPolicyAssignment(randomAssignmentId())
				expectedAssignment.PolicyGroup = ""

				_, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy version is not set", func() {
			It("should return an error", func() {
				expectedAssignment := randomPolicyAssignment("")

				_, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy version id format is invalid", func() {
			It("should return an error", func() {
				expectedAssignment := randomPolicyAssignment(fmt.Sprintf("%s.%[1]s", fake.UUID()))

				_, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)

				Expect(err).To(HaveGrpcStatus(codes.Internal))
			})
		})

		When("the version is invalid", func() {
			It("should return an error", func() {
				invalidVersionId := fmt.Sprintf("%s.0", fake.UUID())
				expectedAssignment := randomPolicyAssignment(invalidVersionId)

				_, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy has been deleted", func() {
			It("should return an error", func() {
				policyVersionId, policy := createPolicy()
				expectedAssignment := randomPolicyAssignment(policyVersionId)
				_, err := rode.DeletePolicy(ctx, &v1alpha1.DeletePolicyRequest{
					Id: policy.Id,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.CreatePolicyAssignment(ctx, expectedAssignment)

				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		When("the policy group has been deleted", func() {
			It("should return an error", func() {
				policyVersionId, _ := createPolicy()
				group := randomPolicyGroup()
				_, err := rode.CreatePolicyGroup(ctx, group)
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicyGroup(ctx, &v1alpha1.DeletePolicyGroupRequest{
					Name: group.Name,
				})
				Expect(err).NotTo(HaveOccurred())

				assignment := &v1alpha1.PolicyAssignment{
					PolicyVersionId: policyVersionId,
					PolicyGroup: group.Name,
				}
				_, err = rode.CreatePolicyAssignment(ctx, assignment)
				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		When("the assignment already exists", func() {
			It("should return an error", func() {
				policyVersionId, _ := createPolicy()
				expectedAssignment := randomPolicyAssignment(policyVersionId)

				_, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.CreatePolicyAssignment(ctx, expectedAssignment)

				Expect(err).To(HaveGrpcStatus(codes.AlreadyExists))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			policyVersionId, _ := createPolicy()
			expectedAssignment := randomPolicyAssignment(policyVersionId)

			_, err := rode.WithRole(entry.Role).CreatePolicyAssignment(ctx, expectedAssignment)

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest([]string{"Administrator", "PolicyAdministrator"})...,
		)
	})

	Describe("Deleting a policy assignment", func() {
		When("the policy assignment exists", func() {
			It("should be deleted successfully", func() {
				policyVersionId, _ := createPolicy()
				expectedAssignment := randomPolicyAssignment(policyVersionId)

				assignment, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicyAssignment(ctx, &v1alpha1.DeletePolicyAssignmentRequest{
					Id: assignment.Id,
				})

				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("the assignment doesn't exist", func() {
			It("should return an error", func() {
				_, err := rode.DeletePolicyAssignment(ctx, &v1alpha1.DeletePolicyAssignmentRequest{
					Id: fake.UUID(),
				})

				Expect(err).To(HaveGrpcStatus(codes.NotFound))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			policyVersionId, _ := createPolicy()
			expectedAssignment := randomPolicyAssignment(policyVersionId)

			assignment, err := rode.CreatePolicyAssignment(ctx, expectedAssignment)
			Expect(err).NotTo(HaveOccurred())

			_, err = rode.WithRole(entry.Role).DeletePolicyAssignment(ctx, &v1alpha1.DeletePolicyAssignmentRequest{
				Id: assignment.Id,
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest([]string{"Administrator", "PolicyAdministrator"})...,
		)
	})

	Describe("Updating a policy assignment", func() {
		var (
			expectedAssignment *v1alpha1.PolicyAssignment
			actualAssignment   *v1alpha1.PolicyAssignment
			policyVersionId    string
			policy             *v1alpha1.Policy
		)

		BeforeEach(func() {
			policyVersionId, policy = createPolicy()
			expectedAssignment = randomPolicyAssignment(policyVersionId)

			var err error
			actualAssignment, err = rode.CreatePolicyAssignment(ctx, expectedAssignment)
			Expect(err).NotTo(HaveOccurred())
		})

		When("the policy version has changed", func() {
			It("should update the assignment", func() {
				policy.Policy.RegoContent += policyUpdates
				updatedPolicy, err := rode.UpdatePolicy(ctx, &v1alpha1.UpdatePolicyRequest{Policy: policy})
				Expect(err).NotTo(HaveOccurred())

				actualAssignment.PolicyVersionId = updatedPolicy.Policy.Id
				_, err = rode.UpdatePolicyAssignment(ctx, actualAssignment)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("the new policy version id references a different policy", func() {
			It("should return an error", func() {
				actualAssignment.PolicyVersionId = randomAssignmentId()

				_, err := rode.UpdatePolicyAssignment(ctx, actualAssignment)

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy version id does not contain a version", func() {
			It("should return an error", func() {
				actualAssignment.PolicyVersionId = fmt.Sprintf("%s.0", fake.UUID())

				_, err := rode.UpdatePolicyAssignment(ctx, actualAssignment)

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy version id is invalid", func() {
			It("should return an error", func() {
				actualAssignment.PolicyVersionId = fmt.Sprintf("%s.%[1]s", fake.UUID())

				_, err := rode.UpdatePolicyAssignment(ctx, expectedAssignment)

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy version does not exist", func() {
			It("should return an error", func() {
				actualAssignment.PolicyVersionId = fmt.Sprintf("%s.2", policy.Id)

				_, err := rode.UpdatePolicyAssignment(ctx, actualAssignment)

				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		When("the update tries to change the policy group", func() {
			It("should return an error", func() {
				actualAssignment.PolicyGroup = fake.Word()

				_, err := rode.UpdatePolicyAssignment(ctx, actualAssignment)

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy has been deleted", func() {
			It("should return an error", func() {
				_, err := rode.DeletePolicy(ctx, &v1alpha1.DeletePolicyRequest{
					Id: policy.Id,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.UpdatePolicyAssignment(ctx, actualAssignment)

				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		When("the policy group has been deleted", func() {
			It("should return an error", func() {
				// create a new group so that this test doesn't interfere with the others
				group := randomPolicyGroup()
				_, err := rode.CreatePolicyGroup(ctx, group)
				Expect(err).NotTo(HaveOccurred())

				actualAssignment, err = rode.CreatePolicyAssignment(ctx, &v1alpha1.PolicyAssignment{
					PolicyGroup: group.Name,
					PolicyVersionId: policy.Policy.Id,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicyGroup(ctx, &v1alpha1.DeletePolicyGroupRequest{
					Name: group.Name,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.UpdatePolicyAssignment(ctx, actualAssignment)

				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).UpdatePolicyAssignment(ctx, actualAssignment)

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest([]string{"Administrator", "PolicyAdministrator"})...,
		)
	})
})

func randomAssignmentId() string {
	return fmt.Sprintf("%s.%d", fake.UUID(), fake.Number(1, 10))
}
