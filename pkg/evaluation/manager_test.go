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
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
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
		ctx context.Context

		expectedEvaluationsAlias string

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
		ctx = context.Background()

		esClient = &esutilfakes.FakeClient{}
		policyManager = &policyfakes.FakeManager{}
		policyGroupManager = &policyfakes.FakePolicyGroupManager{}
		policyAssignmentManager = &policyfakes.FakeAssignmentManager{}
		grafeasExtensions = &grafeasfakes.FakeExtensions{}
		opaClient = &opafakes.FakeClient{}
		resourceManager = &resourcefakes.FakeManager{}
		indexManager = &mocks.FakeIndexManager{}

		expectedEvaluationsAlias = fake.LetterN(10)
		indexManager.AliasNameReturns(expectedEvaluationsAlias)

		manager = NewManager(logger, esClient, policyManager, policyGroupManager, policyAssignmentManager, grafeasExtensions, opaClient, resourceManager, indexManager)
	})

	Context("EvaluateResource", func() {
		var (
			actualResourceEvaluationResult *pb.ResourceEvaluationResult
			actualError                    error

			expectedPolicyGroupName           string
			expectedResourceUri               string
			expectedResourceEvaluationRequest *pb.ResourceEvaluationRequest

			expectedResourceVersion         *pb.ResourceVersion
			expectedGetResourceVersionError error

			expectedPolicyGroup         *pb.PolicyGroup
			expectedGetPolicyGroupError error

			expectedPolicyAssignments          []*pb.PolicyAssignment
			expectedListPolicyAssignmentsError error

			expectedPolicyVersionId string
			expectedPolicyRego      string

			expectedOccurrences                           []*grafeas_proto.Occurrence
			expectedListVersionedResourceOccurrencesError error

			expectedPolicyEntity          *pb.PolicyEntity
			expectedGetPolicyVersionError error

			expectedInitializePolicyError opa.ClientError

			expectedEvaluatePolicyResponse *opa.EvaluatePolicyResponse
			expectedEvaluatePolicyError    error

			expectedBulkResponse *esutil.EsBulkResponse
			expectedBulkError    error
		)

		BeforeEach(func() {
			expectedPolicyGroupName = fake.LetterN(10)
			expectedResourceUri = fake.LetterN(10)
			expectedResourceEvaluationRequest = &pb.ResourceEvaluationRequest{
				ResourceUri: expectedResourceUri,
				PolicyGroup: expectedPolicyGroupName,
				Source: &pb.ResourceEvaluationSource{
					Name: fake.LetterN(10),
					Url:  fake.LetterN(10),
				},
			}

			expectedResourceVersion = &pb.ResourceVersion{
				Version: expectedResourceUri,
			}
			expectedGetResourceVersionError = nil

			expectedPolicyGroup = &pb.PolicyGroup{
				Name: expectedPolicyGroupName,
			}
			expectedGetPolicyGroupError = nil

			expectedPolicyVersion := fake.Number(1, 9)
			expectedPolicyVersionId = fmt.Sprintf("%s.%d", fake.UUID(), expectedPolicyVersion)
			expectedPolicyAssignments = []*pb.PolicyAssignment{
				{
					Id:              fake.UUID(),
					PolicyVersionId: expectedPolicyVersionId,
					PolicyGroup:     expectedPolicyGroupName,
				},
			}
			expectedListPolicyAssignmentsError = nil

			expectedOccurrences = []*grafeas_proto.Occurrence{
				createRandomOccurrence(grafeas_common_proto.NoteKind_DISCOVERY),
			}
			expectedListVersionedResourceOccurrencesError = nil

			expectedPolicyRego = fake.LetterN(10)
			expectedPolicyEntity = createRandomPolicyEntity(expectedPolicyRego, uint32(expectedPolicyVersion))
			expectedGetPolicyVersionError = nil

			expectedInitializePolicyError = nil

			expectedEvaluatePolicyResponse = &opa.EvaluatePolicyResponse{
				Result: &opa.EvaluatePolicyResult{
					Pass: true,
					Violations: []*pb.EvaluatePolicyViolation{
						{
							Id:          fake.LetterN(10),
							Name:        fake.LetterN(10),
							Description: fake.LetterN(10),
							Message:     fake.LetterN(10),
							Link:        fake.LetterN(10),
							Pass:        true,
						},
					},
				},
			}
			expectedEvaluatePolicyError = nil

			expectedBulkResponse = &esutil.EsBulkResponse{
				Items:  []*esutil.EsBulkResponseItem{},
				Errors: false,
			}
			expectedBulkError = nil
		})

		JustBeforeEach(func() {
			resourceManager.GetResourceVersionReturns(expectedResourceVersion, expectedGetResourceVersionError)
			policyGroupManager.GetPolicyGroupReturns(expectedPolicyGroup, expectedGetPolicyGroupError)
			policyAssignmentManager.ListPolicyAssignmentsReturns(&pb.ListPolicyAssignmentsResponse{PolicyAssignments: expectedPolicyAssignments}, expectedListPolicyAssignmentsError)
			grafeasExtensions.ListVersionedResourceOccurrencesReturns(expectedOccurrences, "", expectedListVersionedResourceOccurrencesError)

			policyManager.GetPolicyVersionReturnsOnCall(0, expectedPolicyEntity, expectedGetPolicyVersionError)
			opaClient.InitializePolicyReturnsOnCall(0, expectedInitializePolicyError)
			opaClient.EvaluatePolicyReturnsOnCall(0, expectedEvaluatePolicyResponse, expectedEvaluatePolicyError)

			esClient.BulkReturns(expectedBulkResponse, expectedBulkError)

			actualResourceEvaluationResult, actualError = manager.EvaluateResource(ctx, expectedResourceEvaluationRequest)
		})

		It("should fetch the resource version using the provided URI", func() {
			Expect(resourceManager.GetResourceVersionCallCount()).To(Equal(1))

			_, resourceUri := resourceManager.GetResourceVersionArgsForCall(0)

			Expect(resourceUri).To(Equal(expectedResourceUri))
		})

		It("should fetch the provided policy group", func() {
			Expect(policyGroupManager.GetPolicyGroupCallCount()).To(Equal(1))

			_, getPolicyGroupRequest := policyGroupManager.GetPolicyGroupArgsForCall(0)

			Expect(getPolicyGroupRequest.Name).To(Equal(expectedPolicyGroupName))
		})

		It("should fetch policy assignments for the provided policy group", func() {
			Expect(policyAssignmentManager.ListPolicyAssignmentsCallCount()).To(Equal(1))

			_, listPolicyAssignmentsRequest := policyAssignmentManager.ListPolicyAssignmentsArgsForCall(0)

			Expect(listPolicyAssignmentsRequest.PolicyGroup).To(Equal(expectedPolicyGroupName))
		})

		It("should fetch the versioned resource occurrences for the provided resource uri", func() {
			Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(Equal(1))

			_, resourceUri, _, _ := grafeasExtensions.ListVersionedResourceOccurrencesArgsForCall(0)

			Expect(resourceUri).To(Equal(expectedResourceUri))
		})

		It("should fetch the policy version assigned to the group", func() {
			Expect(policyManager.GetPolicyVersionCallCount()).To(Equal(1))

			_, policyVersionId := policyManager.GetPolicyVersionArgsForCall(0)

			Expect(policyVersionId).To(Equal(expectedPolicyVersionId))
		})

		It("should initialize the policy in OPA", func() {
			Expect(opaClient.InitializePolicyCallCount()).To(Equal(1))

			policyId, rego := opaClient.InitializePolicyArgsForCall(0)

			Expect(policyId).To(Equal(expectedPolicyVersionId))
			Expect(rego).To(Equal(expectedPolicyRego))
		})

		It("should evaluate the policy in OPA", func() {
			Expect(opaClient.EvaluatePolicyCallCount()).To(Equal(1))

			rego, input := opaClient.EvaluatePolicyArgsForCall(0)

			Expect(rego).To(Equal(expectedPolicyRego))
			expectedInput, _ := protojson.Marshal(&pb.EvaluatePolicyInput{
				Occurrences: expectedOccurrences,
			})
			Expect(input).To(MatchJSON(expectedInput))
		})

		It("should store the evaluation results in elasticsearch", func() {
			Expect(esClient.BulkCallCount()).To(Equal(1))

			_, bulkRequest := esClient.BulkArgsForCall(0)

			Expect(bulkRequest.Items).To(HaveLen(2)) // one for resource evaluation, one for single policy evaluation
			Expect(bulkRequest.Index).To(Equal(expectedEvaluationsAlias))

			resourceEvaluationItem := bulkRequest.Items[0]
			resourceEvaluation := resourceEvaluationItem.Message.(*pb.ResourceEvaluation)

			Expect(resourceEvaluationItem.Operation).To(Equal(esutil.BULK_CREATE))
			Expect(resourceEvaluationItem.DocumentId).To(Equal(resourceEvaluation.Id))
			Expect(resourceEvaluationItem.Join.Name).To(Equal(resourceEvaluationRelationName))
			Expect(resourceEvaluationItem.Join.Field).To(Equal(evaluationDocumentJoinField))
			Expect(resourceEvaluationItem.Join.Parent).To(BeEmpty())

			Expect(resourceEvaluation.Pass).To(BeTrue())
			Expect(resourceEvaluation.Source).To(Equal(expectedResourceEvaluationRequest.Source))
			Expect(resourceEvaluation.ResourceVersion).To(Equal(expectedResourceVersion))
			Expect(resourceEvaluation.PolicyGroup).To(Equal(expectedPolicyGroupName))

			policyEvaluationItem := bulkRequest.Items[1]
			policyEvaluation := policyEvaluationItem.Message.(*pb.PolicyEvaluation)

			Expect(policyEvaluationItem.Operation).To(Equal(esutil.BULK_CREATE))
			Expect(policyEvaluationItem.DocumentId).To(Equal(policyEvaluation.Id))
			Expect(policyEvaluationItem.Join.Name).To(Equal(policyEvaluationRelationName))
			Expect(policyEvaluationItem.Join.Field).To(Equal(evaluationDocumentJoinField))
			Expect(policyEvaluationItem.Join.Parent).To(Equal(resourceEvaluation.Id))

			Expect(policyEvaluation.ResourceEvaluationId).To(Equal(resourceEvaluation.Id))
			Expect(policyEvaluation.PolicyVersionId).To(Equal(expectedPolicyVersionId))
			Expect(policyEvaluation.Pass).To(BeTrue())
			Expect(policyEvaluation.Violations).To(Equal(expectedEvaluatePolicyResponse.Result.Violations))
		})

		It("should return the resource evaluation result and policy evaluation results", func() {
			Expect(actualResourceEvaluationResult).ToNot(BeNil())
			Expect(actualError).ToNot(HaveOccurred())
		})
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
