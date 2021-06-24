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

package evaluation

import (
	"context"
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/es-index-manager/mocks"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil/esutilfakes"
	"github.com/rode/rode/opa"
	"github.com/rode/rode/opa/opafakes"
	"github.com/rode/rode/pkg/constants"
	"github.com/rode/rode/pkg/grafeas/grafeasfakes"
	"github.com/rode/rode/pkg/policy/policyfakes"
	"github.com/rode/rode/pkg/resource/resourcefakes"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("evaluation manager", func() {
	var (
		ctx = context.Background()

		esClient                *esutilfakes.FakeClient
		policyManager           *policyfakes.FakeManager
		policyGroupManager      *policyfakes.FakePolicyGroupManager
		policyAssignmentManager *policyfakes.FakeAssignmentManager
		grafeasExtensions       *grafeasfakes.FakeExtensions
		opaClient               *opafakes.FakeClient
		resourceManager         *resourcefakes.FakeManager
		indexManager            *mocks.FakeIndexManager

		manager Manager
	)

	BeforeEach(func() {
		esClient = &esutilfakes.FakeClient{}
		policyManager = &policyfakes.FakeManager{}
		policyGroupManager = &policyfakes.FakePolicyGroupManager{}
		policyAssignmentManager = &policyfakes.FakeAssignmentManager{}
		grafeasExtensions = &grafeasfakes.FakeExtensions{}
		opaClient = &opafakes.FakeClient{}
		resourceManager = &resourcefakes.FakeManager{}
		indexManager = &mocks.FakeIndexManager{}

		manager = NewManager(logger, esClient, policyManager, policyGroupManager, policyAssignmentManager, grafeasExtensions, opaClient, resourceManager, indexManager)
	})

	Context("EvaluatePolicy", func() {
		var (
			policyId       string
			version        uint32
			policy         *pb.Policy
			getPolicyError error

			resourceUri string
			request     *pb.EvaluatePolicyRequest

			opaInitializePolicyError opa.ClientError

			opaEvaluatePolicyResponse *opa.EvaluatePolicyResponse
			opaEvaluatePolicyError    error

			listVersionedResourceOccurrencesResponse []*grafeas_proto.Occurrence
			listVersionedResourceOccurrencesError    error

			actualResponse *pb.EvaluatePolicyResponse
			actualError    error

			expectedPolicyRego string
		)

		BeforeEach(func() {
			expectedPolicyRego = fake.LetterN(10)
			policyId = fake.UUID()
			version = fake.Uint32()
			resourceUri = fake.URL()

			policy = createRandomPolicy(policyId, version)
			policy.Policy = createRandomPolicyEntity(expectedPolicyRego, version)
			getPolicyError = nil

			opaInitializePolicyError = nil

			listVersionedResourceOccurrencesResponse = []*grafeas_proto.Occurrence{
				createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
				createRandomOccurrence(grafeas_common_proto.NoteKind_ATTESTATION),
			}
			listVersionedResourceOccurrencesError = nil

			opaEvaluatePolicyResponse = &opa.EvaluatePolicyResponse{
				Result: &opa.EvaluatePolicyResult{
					Pass:       true,
					Violations: []*pb.EvaluatePolicyViolation{},
				},
				Explanation: &[]string{fake.Word()},
			}
			opaEvaluatePolicyError = nil

			request = &pb.EvaluatePolicyRequest{
				Policy:      policyId,
				ResourceUri: resourceUri,
			}
		})

		JustBeforeEach(func() {
			policyManager.GetPolicyReturns(policy, getPolicyError)

			opaClient.InitializePolicyReturns(opaInitializePolicyError)
			grafeasExtensions.ListVersionedResourceOccurrencesReturns(listVersionedResourceOccurrencesResponse, "", listVersionedResourceOccurrencesError)
			opaClient.EvaluatePolicyReturns(opaEvaluatePolicyResponse, opaEvaluatePolicyError)

			actualResponse, actualError = manager.EvaluatePolicy(ctx, request)
		})

		When("evaluation is successful", func() {
			It("should fetch the policy and current policy version from Elasticsearch", func() {
				Expect(policyManager.GetPolicyCallCount()).To(Equal(1))

				_, actualRequest := policyManager.GetPolicyArgsForCall(0)

				Expect(actualRequest.Id).To(Equal(policyId))
			})

			It("should initialize the policy in Open Policy Agent", func() {
				Expect(opaClient.InitializePolicyCallCount()).To(Equal(1))

				actualPolicyId, policyContent := opaClient.InitializePolicyArgsForCall(0)

				Expect(actualPolicyId).To(Equal(policyId))
				Expect(policyContent).To(Equal(expectedPolicyRego))
			})

			It("should fetch versioned resource occurrences from Grafeas", func() {
				Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(Equal(1))

				_, actualResourceUri, actualPageToken, actualPageSize := grafeasExtensions.ListVersionedResourceOccurrencesArgsForCall(0)

				Expect(actualResourceUri).To(Equal(resourceUri))
				Expect(actualPageToken).To(BeEmpty())
				Expect(actualPageSize).To(BeEquivalentTo(constants.MaxPageSize))
			})

			It("should evaluate the policy in Open Policy Agent", func() {
				Expect(opaClient.EvaluatePolicyCallCount()).To(Equal(1))
				actualPolicy, actualInput := opaClient.EvaluatePolicyArgsForCall(0)

				expectedInput, err := protojson.Marshal(&pb.EvaluatePolicyInput{
					Occurrences: listVersionedResourceOccurrencesResponse,
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(actualPolicy).To(Equal(expectedPolicyRego))
				Expect(actualInput).To(MatchJSON(expectedInput))
			})

			It("should return the evaluation results", func() {
				Expect(actualResponse).NotTo(BeNil())
				Expect(actualResponse.Pass).To(BeTrue())
				Expect(actualResponse.Explanation[0]).To(Equal((*opaEvaluatePolicyResponse.Explanation)[0]))
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		When("the request doesn't contain a resource uri", func() {
			BeforeEach(func() {
				request.ResourceUri = ""
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not try to fetch or evaluate policy", func() {
				Expect(policyManager.GetPolicyCallCount()).To(Equal(0))
				Expect(opaClient.EvaluatePolicyCallCount()).To(Equal(0))
			})
		})

		When("an error occurs fetching policy", func() {
			BeforeEach(func() {
				getPolicyError = errors.New("get policy error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to evaluate policy", func() {
				Expect(opaClient.EvaluatePolicyCallCount()).To(Equal(0))
			})
		})

		When("an error occurs initializing policy in Open Policy Agent", func() {
			BeforeEach(func() {
				opaInitializePolicyError = opa.NewClientError(fake.Word(), opa.OpaClientErrorTypeHTTP, nil)
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to evaluate policy", func() {
				Expect(opaClient.EvaluatePolicyCallCount()).To(Equal(0))
			})
		})

		When("an error occurs listing occurrences", func() {
			BeforeEach(func() {
				listVersionedResourceOccurrencesError = errors.New("grafeas error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to evaluate policy", func() {
				Expect(opaClient.EvaluatePolicyCallCount()).To(Equal(0))
			})
		})

		When("an error occurs evaluating policy", func() {
			BeforeEach(func() {
				opaEvaluatePolicyError = errors.New("evaluate error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy result is absent", func() {
			BeforeEach(func() {
				opaEvaluatePolicyResponse.Result = nil
			})

			It("should not pass", func() {
				Expect(actualResponse.Pass).To(BeFalse())
			})
		})

		When("there are violations", func() {
			var (
				expectedViolationsCount int
				expectedViolations      []*pb.EvaluatePolicyViolation
			)

			BeforeEach(func() {
				expectedViolationsCount = fake.Number(1, 5)
				expectedViolations = []*pb.EvaluatePolicyViolation{}

				for i := 0; i < expectedViolationsCount; i++ {
					expectedViolations = append(expectedViolations, randomViolation())
				}
				opaEvaluatePolicyResponse.Result.Violations = expectedViolations
			})

			It("should include them in the evaluation result", func() {
				Expect(actualResponse.Result).To(HaveLen(1))
				actualViolations := actualResponse.Result[0].Violations
				Expect(actualViolations).To(ConsistOf(expectedViolations))
			})
		})
	})
})

func createRandomOccurrence(kind grafeas_common_proto.NoteKind) *grafeas_proto.Occurrence {
	return &grafeas_proto.Occurrence{
		Name: fake.LetterN(10),
		Resource: &grafeas_proto.Resource{
			Uri: fmt.Sprintf("%s@sha256:%s", fake.URL(), fake.LetterN(10)),
		},
		NoteName:    fake.LetterN(10),
		Kind:        kind,
		Remediation: fake.LetterN(10),
		CreateTime:  timestamppb.New(fake.Date()),
		UpdateTime:  timestamppb.New(fake.Date()),
		Details:     nil,
	}
}

func randomViolation() *pb.EvaluatePolicyViolation {
	return &pb.EvaluatePolicyViolation{
		Id:          fake.LetterN(10),
		Name:        fake.Word(),
		Description: fake.Word(),
		Message:     fake.Word(),
		Pass:        fake.Bool(),
	}
}

func createRandomPolicy(id string, version uint32) *pb.Policy {
	return &pb.Policy{
		Id:             id,
		Name:           fake.Word(),
		Description:    fake.Word(),
		CurrentVersion: version,
	}
}

func createRandomPolicyEntity(policy string, version uint32) *pb.PolicyEntity {
	return &pb.PolicyEntity{
		Version:     version,
		RegoContent: policy,
		SourcePath:  fake.URL(),
		Message:     fake.Word(),
	}
}
