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
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/rode/rode/test/data"
	. "github.com/rode/rode/test/util"
	"google.golang.org/grpc/codes"
)

var _ = Describe("Evaluations", func() {
	var (
		ctx = context.Background()
	)

	Describe("Evaluating policy", func() {
		When("the policy inputs are valid", func() {
			It("should pass the evaluation", func() {
				policy, err := rode.CreatePolicy(ctx, randomPolicy(data.RequireOccurrences))
				Expect(err).NotTo(HaveOccurred())
				occurrence := randomBuildOccurrence()

				_, err = rode.BatchCreateOccurrences(ctx, &v1alpha1.BatchCreateOccurrencesRequest{
					Occurrences: []*grafeas_proto.Occurrence{occurrence},
				})
				Expect(err).NotTo(HaveOccurred())

				response, err := rode.EvaluatePolicy(ctx, &v1alpha1.EvaluatePolicyRequest{
					Policy:      policy.Id,
					ResourceUri: occurrence.Resource.Uri,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Pass).To(BeTrue())
			})
		})

		When("the policy inputs are invalid", func() {
			It("should fail the evaluation", func() {
				policy, err := rode.CreatePolicy(ctx, randomPolicy(data.RequireOccurrences))
				Expect(err).NotTo(HaveOccurred())

				response, err := rode.EvaluatePolicy(ctx, &v1alpha1.EvaluatePolicyRequest{
					Policy:      policy.Id,
					ResourceUri: randomContainerImageUri(),
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Pass).To(BeFalse())
			})
		})

		When("the resource uri is not set", func() {
			It("should return an error", func() {
				_, err := rode.EvaluatePolicy(ctx, &v1alpha1.EvaluatePolicyRequest{
					Policy: fake.UUID(),
				})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy does not exist", func() {
			It("should return an error", func() {
				_, err := rode.EvaluatePolicy(ctx, &v1alpha1.EvaluatePolicyRequest{
					ResourceUri: randomContainerImageUri(),
					Policy:      fake.UUID(),
				})

				Expect(err).To(HaveGrpcStatus(codes.Internal))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			policy, err := rode.CreatePolicy(ctx, randomPolicy(data.RequireOccurrences))
			Expect(err).NotTo(HaveOccurred())

			_, err = rode.WithRole(entry.Role).EvaluatePolicy(ctx, &v1alpha1.EvaluatePolicyRequest{
				Policy:      policy.Id,
				ResourceUri: randomContainerImageUri(),
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
