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
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/proto/v1alpha1"
	common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	package_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/package_go_proto"
	vulnerability_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/vulnerability_go_proto"
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
				policy, err := rode.CreatePolicy(ctx, randomPolicy(data.RequireOccurrencesPolicy))
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
				policy, err := rode.CreatePolicy(ctx, randomPolicy(data.RequireOccurrencesPolicy))
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
			policy, err := rode.CreatePolicy(ctx, randomPolicy(data.RequireOccurrencesPolicy))
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

	Describe("Evaluating a resource", func() {
		var (
			buildOccurrence *grafeas_proto.Occurrence
			resourceUri     string
			policyGroup     string
			once            sync.Once
		)

		var setup = func() {
			buildOccurrence = randomBuildOccurrence()
			_, err := rode.BatchCreateOccurrences(ctx, &v1alpha1.BatchCreateOccurrencesRequest{
				Occurrences: []*grafeas_proto.Occurrence{buildOccurrence},
			})
			Expect(err).NotTo(HaveOccurred())
			resourceUri = buildOccurrence.Resource.Uri

			group, err := rode.CreatePolicyGroup(ctx, randomPolicyGroup())
			Expect(err).NotTo(HaveOccurred())
			policyGroup = group.Name

			minimalPolicy, err := rode.CreatePolicy(ctx, randomPolicy(data.MinimalPolicy))
			Expect(err).NotTo(HaveOccurred())

			noVulnsPolicy, err := rode.CreatePolicy(ctx, randomPolicy(data.NoVulnerabilitiesPolicy))
			Expect(err).NotTo(HaveOccurred())

			_, err = rode.CreatePolicyAssignment(ctx, &v1alpha1.PolicyAssignment{
				PolicyGroup:     group.Name,
				PolicyVersionId: minimalPolicy.Policy.Id,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = rode.CreatePolicyAssignment(ctx, &v1alpha1.PolicyAssignment{
				PolicyGroup:     group.Name,
				PolicyVersionId: noVulnsPolicy.Policy.Id,
			})
			Expect(err).NotTo(HaveOccurred())
		}

		BeforeEach(func() {
			// work around for the lack of BeforeAll in ginkgo v1, can be replaced once v2 is released
			once.Do(setup)
		})

		When("the resource passes all policies", func() {
			It("should pass the evaluation", func() {
				response, err := rode.EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
					PolicyGroup: policyGroup,
					ResourceUri: resourceUri,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(response.ResourceEvaluation.Pass).To(BeTrue())

				evaluation, err := rode.GetResourceEvaluation(ctx, &v1alpha1.GetResourceEvaluationRequest{
					Id: response.ResourceEvaluation.Id,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(evaluation.ResourceEvaluation.Pass).To(BeTrue())
			})
		})

		When("the resource fails a policy", func() {
			It("should pass the evaluation", func() {
				failingResourceUri := buildOccurrence.GetBuild().Provenance.BuiltArtifacts[0].Id

				_, err := rode.BatchCreateOccurrences(ctx, &v1alpha1.BatchCreateOccurrencesRequest{
					Occurrences: []*grafeas_proto.Occurrence{
						{
							Name: fake.LetterN(10),
							Resource: &grafeas_proto.Resource{
								Uri: failingResourceUri,
							},
							NoteName: fmt.Sprintf("projects/rode/notes/%s", fake.LetterN(15)),
							Kind:     common_proto.NoteKind_VULNERABILITY,
							Details: &grafeas_proto.Occurrence_Vulnerability{
								Vulnerability: &vulnerability_proto.Details{
									Type:              "git",
									EffectiveSeverity: vulnerability_proto.Severity_HIGH,
									ShortDescription:  fake.LetterN(10),
									PackageIssue: []*vulnerability_proto.PackageIssue{
										{
											AffectedLocation: &vulnerability_proto.VulnerabilityLocation{
												CpeUri:  fake.URL(),
												Package: fake.Word(),
												Version: &package_proto.Version{
													Name: fake.Word(),
													Kind: package_proto.Version_NORMAL,
												},
											},
										},
									},
								},
							},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				response, err := rode.EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
					PolicyGroup: policyGroup,
					ResourceUri: failingResourceUri,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(response.ResourceEvaluation.Pass).To(BeFalse())
			})
		})

		When("the resource uri is not specified", func() {
			It("should return an error", func() {
				_, err := rode.EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
					PolicyGroup: policyGroup,
				})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the policy group is not specified", func() {
			It("should return an error", func() {
				_, err := rode.EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
					ResourceUri: resourceUri,
				})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("the resource version does not exist", func() {
			It("should return an error", func() {
				_, err := rode.EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
					PolicyGroup: policyGroup,
					ResourceUri: randomContainerImageUri(),
				})

				Expect(err).To(HaveGrpcStatus(codes.NotFound))
			})
		})

		When("the policy group does not exist", func() {
			It("should return an error", func() {
				_, err := rode.EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
					PolicyGroup: strings.ToLower(fake.LetterN(10)),
					ResourceUri: resourceUri,
				})

				Expect(err).To(HaveGrpcStatus(codes.NotFound))
			})
		})

		When("the policy group does not contain any assignments", func() {
			It("should return an error", func() {
				group, err := rode.CreatePolicyGroup(ctx, randomPolicyGroup())
				Expect(err).NotTo(HaveOccurred())

				_, err = rode.EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
					PolicyGroup: group.Name,
					ResourceUri: resourceUri,
				})

				Expect(err).To(HaveGrpcStatus(codes.FailedPrecondition))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).EvaluateResource(ctx, &v1alpha1.ResourceEvaluationRequest{
				PolicyGroup: policyGroup,
				ResourceUri: resourceUri,
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest(
				"Enforcer",
				"PolicyDeveloper",
				"PolicyAdministrator",
				"Administrator",
			)...,
		)
	})
})
