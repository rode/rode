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
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/proto/v1alpha1"
	. "github.com/rode/rode/test/util"
	"google.golang.org/grpc/codes"
)

var _ = Describe("Policy Groups", func() {
	var (
		ctx                   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Creating a policy group", func() {
		When("the policy group name is valid", func() {
			It("should create the policy group", func() {
				expectedGroup := randomPolicyGroup()
				createdPolicyGroup, err := rode.CreatePolicyGroup(ctx, expectedGroup)
				Expect(err).NotTo(HaveOccurred())

				group, err := rode.GetPolicyGroup(ctx, &v1alpha1.GetPolicyGroupRequest{Name: createdPolicyGroup.Name})
				Expect(err).NotTo(HaveOccurred())
				Expect(group.Name).To(Equal(expectedGroup.Name))
			})
		})

		When("the name is unset", func() {
			It("should return an error", func() {
				_, err := rode.CreatePolicyGroup(ctx, &v1alpha1.PolicyGroup{})
				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the name is invalid", func() {
			It("should return an error", func() {
				_, err := rode.CreatePolicyGroup(ctx, &v1alpha1.PolicyGroup{Name: "Policy Group"})
				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("there is an existing group with the same name", func() {
			It("should return an error", func() {
				expectedGroup := randomPolicyGroup();

				_, err := rode.CreatePolicyGroup(ctx, expectedGroup)
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.CreatePolicyGroup(ctx, expectedGroup)
				Expect(err).To(HaveGrpcStatus(codes.AlreadyExists))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).CreatePolicyGroup(ctx, randomPolicyGroup())

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest([]string{"Administrator", "PolicyAdministrator"})...,
		)
	})

	Describe("Deleting a policy group", func() {
		When("the policy group is deleted", func() {
			It("should mark the policy group as deleted", func() {
				expectedGroup := randomPolicyGroup();

				group, err := rode.CreatePolicyGroup(ctx, expectedGroup)
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicyGroup(ctx, &v1alpha1.DeletePolicyGroupRequest{Name: group.Name})
				Expect(err).NotTo(HaveOccurred())

				actualGroup, err := rode.GetPolicyGroup(ctx, &v1alpha1.GetPolicyGroupRequest{Name: group.Name})
				Expect(err).NotTo(HaveOccurred())
				Expect(actualGroup.Deleted).To(BeTrue())
			})
		})

		When("the policy group has been deleted", func() {
			It("should not allow another group of the same name", func() {
				expectedGroup := randomPolicyGroup();

				group, err := rode.CreatePolicyGroup(ctx, expectedGroup)
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicyGroup(ctx, &v1alpha1.DeletePolicyGroupRequest{Name: group.Name})
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.CreatePolicyGroup(ctx, &v1alpha1.PolicyGroup{Name: group.Name})
				Expect(err).To(HaveGrpcStatus(codes.AlreadyExists))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			expectedGroup := randomPolicyGroup()
			group, err := rode.CreatePolicyGroup(ctx, expectedGroup)
			Expect(err).NotTo(HaveOccurred())

			_, err = rode.WithRole(entry.Role).DeletePolicyGroup(ctx, &v1alpha1.DeletePolicyGroupRequest{
				Name: group.Name,
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

	Describe("Updating a policy group", func() {
		When("the description has changed", func() {
			It("should be updated", func() {
				expectedGroup := randomPolicyGroup();

				createdPolicyGroup, err := rode.CreatePolicyGroup(ctx, expectedGroup)
				Expect(err).NotTo(HaveOccurred())

				expectedDescription := fake.Sentence(5)
				updated := &v1alpha1.PolicyGroup{
					Name:        createdPolicyGroup.Name,
					Description: expectedDescription,
				}
				actualUpdated, err := rode.UpdatePolicyGroup(ctx, updated)

				Expect(err).NotTo(HaveOccurred())
				Expect(actualUpdated.Description).To(Equal(expectedDescription))
			})
		})

		When("the group doesn't exist", func() {
			It("should not allow the update", func() {
				_, err := rode.UpdatePolicyGroup(ctx, randomPolicyGroup())

				Expect(err).To(HaveGrpcStatus(codes.NotFound))
			})
		})

		When("the group has been deleted", func() {
			It("should not allow the update", func() {
				expectedGroup := randomPolicyGroup()

				group, err := rode.CreatePolicyGroup(ctx, expectedGroup)
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.DeletePolicyGroup(ctx, &v1alpha1.DeletePolicyGroupRequest{Name: group.Name})
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.UpdatePolicyGroup(ctx, expectedGroup)
				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			expectedGroup := randomPolicyGroup()
			group, err := rode.CreatePolicyGroup(ctx, expectedGroup)
			Expect(err).NotTo(HaveOccurred())

			group.Description = fake.LetterN(10)
			_, err = rode.WithRole(entry.Role).UpdatePolicyGroup(ctx, group)

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

func randomPolicyGroup() *v1alpha1.PolicyGroup {
	return &v1alpha1.PolicyGroup{
		Name: strings.ToLower(fake.LetterN(10)),
		Description: fake.LetterN(10),
	}
}
