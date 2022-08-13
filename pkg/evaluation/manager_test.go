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
	"encoding/json"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/es-index-manager/mocks"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil/esutilfakes"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering/filteringfakes"
	"github.com/rode/rode/config"
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
		esConfig                *config.ElasticsearchConfig
		policyManager           *policyfakes.FakeManager
		policyGroupManager      *policyfakes.FakePolicyGroupManager
		policyAssignmentManager *policyfakes.FakeAssignmentManager
		grafeasExtensions       *grafeasfakes.FakeExtensions
		opaClient               *opafakes.FakeClient
		resourceManager         *resourcefakes.FakeManager
		indexManager            *mocks.FakeIndexManager
		filterer                *filteringfakes.FakeFilterer

		manager Manager
	)

	BeforeEach(func() {
		ctx = context.Background()

		esClient = &esutilfakes.FakeClient{}
		esConfig = &config.ElasticsearchConfig{
			Refresh: config.RefreshTrue,
		}
		policyManager = &policyfakes.FakeManager{}
		policyGroupManager = &policyfakes.FakePolicyGroupManager{}
		policyAssignmentManager = &policyfakes.FakeAssignmentManager{}
		grafeasExtensions = &grafeasfakes.FakeExtensions{}
		opaClient = &opafakes.FakeClient{}
		resourceManager = &resourcefakes.FakeManager{}
		indexManager = &mocks.FakeIndexManager{}
		filterer = &filteringfakes.FakeFilterer{}

		expectedEvaluationsAlias = fake.LetterN(10)
		indexManager.AliasNameReturns(expectedEvaluationsAlias)

		manager = NewManager(logger, esClient, esConfig, policyManager, policyGroupManager, policyAssignmentManager, grafeasExtensions, opaClient, resourceManager, indexManager, filterer)
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

			expectedEvaluatePolicyResponse *opa.EvaluatePolicyResult
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

			expectedEvaluatePolicyResponse = &opa.EvaluatePolicyResult{
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

			_, policyId, rego := opaClient.InitializePolicyArgsForCall(0)

			Expect(policyId).To(Equal(expectedPolicyVersionId))
			Expect(rego).To(Equal(expectedPolicyRego))
		})

		It("should evaluate the policy in OPA", func() {
			Expect(opaClient.EvaluatePolicyCallCount()).To(Equal(1))

			_, rego, input := opaClient.EvaluatePolicyArgsForCall(0)

			Expect(rego).To(Equal(expectedPolicyVersionId))
			actualInputJson, _ := json.Marshal(input)
			expectedInputJson, _ := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(&pb.EvaluatePolicyInput{
				Occurrences: expectedOccurrences,
			})
			Expect(actualInputJson).To(MatchJSON(expectedInputJson))
		})

		It("should store the evaluation results in elasticsearch", func() {
			Expect(esClient.BulkCallCount()).To(Equal(1))

			_, bulkRequest := esClient.BulkArgsForCall(0)

			Expect(bulkRequest.Items).To(HaveLen(2)) // one for resource evaluation, one for single policy evaluation
			Expect(bulkRequest.Index).To(Equal(expectedEvaluationsAlias))
			Expect(bulkRequest.Refresh).To(Equal(esConfig.Refresh.String()))

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
			Expect(policyEvaluation.Violations).To(Equal(expectedEvaluatePolicyResponse.Violations))
		})

		It("should return the resource evaluation result and policy evaluation results", func() {
			Expect(actualResourceEvaluationResult).ToNot(BeNil())
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("the resource uri is missing", func() {
			BeforeEach(func() {
				expectedResourceEvaluationRequest.ResourceUri = ""
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not continue with the request", func() {
				Expect(resourceManager.GetResourceVersionCallCount()).To(BeZero())
				Expect(policyGroupManager.GetPolicyGroupCallCount()).To(BeZero())
				Expect(policyAssignmentManager.ListPolicyAssignmentsCallCount()).To(BeZero())
				Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(BeZero())
				Expect(policyManager.GetPolicyVersionCallCount()).To(BeZero())
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("the policy group name is missing", func() {
			BeforeEach(func() {
				expectedResourceEvaluationRequest.PolicyGroup = ""
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not continue with the request", func() {
				Expect(resourceManager.GetResourceVersionCallCount()).To(BeZero())
				Expect(policyGroupManager.GetPolicyGroupCallCount()).To(BeZero())
				Expect(policyAssignmentManager.ListPolicyAssignmentsCallCount()).To(BeZero())
				Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(BeZero())
				Expect(policyManager.GetPolicyVersionCallCount()).To(BeZero())
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("fetching the resource version fails", func() {
			BeforeEach(func() {
				expectedGetResourceVersionError = errors.New("error fetching resource version")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not continue with the request", func() {
				Expect(policyGroupManager.GetPolicyGroupCallCount()).To(BeZero())
				Expect(policyAssignmentManager.ListPolicyAssignmentsCallCount()).To(BeZero())
				Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(BeZero())
				Expect(policyManager.GetPolicyVersionCallCount()).To(BeZero())
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("fetching the policy group fails", func() {
			BeforeEach(func() {
				expectedGetPolicyGroupError = errors.New("error fetching policy group")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not continue with the request", func() {
				Expect(policyAssignmentManager.ListPolicyAssignmentsCallCount()).To(BeZero())
				Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(BeZero())
				Expect(policyManager.GetPolicyVersionCallCount()).To(BeZero())
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("fetching the policy assignments fails", func() {
			BeforeEach(func() {
				expectedListPolicyAssignmentsError = errors.New("error fetching policy assignments")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not continue with the request", func() {
				Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(BeZero())
				Expect(policyManager.GetPolicyVersionCallCount()).To(BeZero())
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("there are no policy assignments for the specified group", func() {
			BeforeEach(func() {
				expectedPolicyAssignments = []*pb.PolicyAssignment{}
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})

			It("should not continue with the request", func() {
				Expect(grafeasExtensions.ListVersionedResourceOccurrencesCallCount()).To(BeZero())
				Expect(policyManager.GetPolicyVersionCallCount()).To(BeZero())
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("fetching the occurrences fails", func() {
			BeforeEach(func() {
				expectedListVersionedResourceOccurrencesError = errors.New("error fetching occurrences")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not continue with the request", func() {
				Expect(policyManager.GetPolicyVersionCallCount()).To(BeZero())
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("fetching the policy version fails", func() {
			BeforeEach(func() {
				expectedGetPolicyVersionError = errors.New("error fetching policy version")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to evaluate a policy", func() {
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("the policy version is not found", func() {
			BeforeEach(func() {
				expectedPolicyEntity = nil
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to evaluate a policy", func() {
				Expect(opaClient.InitializePolicyCallCount()).To(BeZero())
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("initializing the policy fails", func() {
			BeforeEach(func() {
				expectedInitializePolicyError = opa.NewClientError("error initializing policy", opa.OpaClientErrorTypeLoadPolicy, errors.New("error initializing policy"))
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to evaluate a policy", func() {
				Expect(opaClient.EvaluatePolicyCallCount()).To(BeZero())
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("evaluating the policy fails", func() {
			BeforeEach(func() {
				expectedEvaluatePolicyError = errors.New("error evaluating policy")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to store evaluation results", func() {
				Expect(esClient.BulkCallCount()).To(BeZero())
			})
		})

		When("the policy does not pass", func() {
			BeforeEach(func() {
				expectedEvaluatePolicyResponse.Pass = false
			})

			It("should mark the resource evaluation as failed", func() {
				_, bulkRequest := esClient.BulkArgsForCall(0)

				resourceEvaluationItem := bulkRequest.Items[0]
				resourceEvaluation := resourceEvaluationItem.Message.(*pb.ResourceEvaluation)

				Expect(resourceEvaluation.Pass).To(BeFalse())

				policyEvaluationItem := bulkRequest.Items[1]
				policyEvaluation := policyEvaluationItem.Message.(*pb.PolicyEvaluation)

				Expect(policyEvaluation.Pass).To(BeFalse())
			})
		})

		When("storing the evaluation results fails", func() {
			BeforeEach(func() {
				expectedBulkError = errors.New("error during bulk insert")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the bulk insert has errors", func() {
			BeforeEach(func() {
				expectedBulkResponse.Items = []*esutil.EsBulkResponseItem{
					{
						Index: &esutil.EsIndexDocResponse{
							Error: &esutil.EsIndexDocError{
								Type:   fake.LetterN(10),
								Reason: fake.LetterN(10),
							},
						},
					},
				}
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("multiple policies are assigned to the policy group", func() {
			BeforeEach(func() {
				// the new fake values can be the same as the previous one, all that matters is that the additional policy fails
				expectedPolicyAssignments = append(expectedPolicyAssignments, &pb.PolicyAssignment{
					Id:              fake.UUID(),
					PolicyVersionId: expectedPolicyVersionId,
					PolicyGroup:     expectedPolicyGroupName,
				})

				policyManager.GetPolicyVersionReturnsOnCall(1, expectedPolicyEntity, expectedGetPolicyVersionError)
				opaClient.InitializePolicyReturnsOnCall(1, expectedInitializePolicyError)
			})

			When("one of the policies fails", func() {
				BeforeEach(func() {
					opaClient.EvaluatePolicyReturnsOnCall(1, &opa.EvaluatePolicyResult{
						Pass: false,
					}, expectedEvaluatePolicyError)
				})

				It("should store the overall resource evaluation as a failure", func() {
					Expect(esClient.BulkCallCount()).To(Equal(1))

					_, bulkRequest := esClient.BulkArgsForCall(0)

					Expect(bulkRequest.Items).To(HaveLen(3)) // one for resource evaluation, one for each policy evaluation
					Expect(bulkRequest.Index).To(Equal(expectedEvaluationsAlias))

					resourceEvaluationItem := bulkRequest.Items[0]
					resourceEvaluation := resourceEvaluationItem.Message.(*pb.ResourceEvaluation)

					Expect(resourceEvaluation.Pass).To(BeFalse())

					passingPolicyEvaluationItem := bulkRequest.Items[1]
					passingPolicyEvaluation := passingPolicyEvaluationItem.Message.(*pb.PolicyEvaluation)

					Expect(passingPolicyEvaluation.Pass).To(BeTrue())

					failingPolicyEvaluationItem := bulkRequest.Items[2]
					failingPolicyEvaluation := failingPolicyEvaluationItem.Message.(*pb.PolicyEvaluation)

					Expect(failingPolicyEvaluation.Pass).To(BeFalse())
				})
			})
		})
	})

	Context("ListResourceEvaluations", func() {
		var (
			actualListResourceEvaluationsResponse *pb.ListResourceEvaluationsResponse
			actualError                           error

			expectedListResourceEvaluationsRequest *pb.ListResourceEvaluationsRequest

			expectedSearchResponse *esutil.SearchResponse
			expectedSearchError    error

			expectedResourceEvaluation   *pb.ResourceEvaluation
			expectedResourceEvaluationId string
			expectedResourceUri          string

			expectedFilterQuery *filtering.Query
			expectedFilterError error

			expectedGetResourceVersionError error

			expectedPolicyEvaluation *pb.PolicyEvaluation

			expectedMultiSearchResponse *esutil.EsMultiSearchResponse
			expectedMultiSearchError    error
		)

		BeforeEach(func() {
			expectedResourceUri = fake.LetterN(10)
			expectedResourceEvaluationId = fake.LetterN(10)
			expectedListResourceEvaluationsRequest = &pb.ListResourceEvaluationsRequest{
				ResourceUri: expectedResourceUri,
			}

			expectedResourceEvaluation = &pb.ResourceEvaluation{
				Id:   expectedResourceEvaluationId,
				Pass: fake.Bool(),
			}
			resourceEvaluationJson, _ := protojson.Marshal(expectedResourceEvaluation)
			expectedSearchResponse = &esutil.SearchResponse{
				Hits: &esutil.EsSearchResponseHits{
					Hits: []*esutil.EsSearchResponseHit{
						{
							Source: resourceEvaluationJson,
							ID:     expectedResourceEvaluationId,
						},
					},
					Total: &esutil.EsSearchResponseTotal{
						Value: 1,
					},
				},
			}
			expectedSearchError = nil

			expectedGetResourceVersionError = nil

			expectedPolicyEvaluation = &pb.PolicyEvaluation{
				Id:                   fake.UUID(),
				ResourceEvaluationId: expectedResourceEvaluationId,
				Pass:                 fake.Bool(),
			}
			policyEvaluationJson, _ := protojson.Marshal(expectedPolicyEvaluation)
			expectedMultiSearchResponse = &esutil.EsMultiSearchResponse{
				Responses: []*esutil.EsMultiSearchResponseHitsSummary{
					{
						Hits: &esutil.EsMultiSearchResponseHits{
							Hits: []*esutil.EsMultiSearchResponseHit{
								{
									Source: policyEvaluationJson,
								},
							},
							Total: &esutil.EsSearchResponseTotal{
								Value: 1,
							},
						},
					},
				},
			}
			expectedMultiSearchError = nil
		})

		JustBeforeEach(func() {
			resourceManager.GetResourceVersionReturns(nil, expectedGetResourceVersionError)
			filterer.ParseExpressionReturns(expectedFilterQuery, expectedFilterError)
			esClient.SearchReturns(expectedSearchResponse, expectedSearchError)
			esClient.MultiSearchReturns(expectedMultiSearchResponse, expectedMultiSearchError)

			actualListResourceEvaluationsResponse, actualError = manager.ListResourceEvaluations(ctx, expectedListResourceEvaluationsRequest)
		})

		It("should check for the existence of the resource version", func() {
			Expect(resourceManager.GetResourceVersionCallCount()).To(Equal(1))

			_, resourceUri := resourceManager.GetResourceVersionArgsForCall(0)
			Expect(resourceUri).To(Equal(expectedResourceUri))
		})

		It("should perform a search for resource evaluations", func() {
			Expect(esClient.SearchCallCount()).To(Equal(1))

			_, searchRequest := esClient.SearchArgsForCall(0)
			Expect(searchRequest.Index).To(Equal(expectedEvaluationsAlias))

			// no pagination options were specified
			Expect(searchRequest.Pagination).To(BeNil())

			// should sort by timestamp
			Expect(searchRequest.Search.Sort["created"]).To(Equal(esutil.EsSortOrderDescending))

			// no filter was specified, so we should only have one query
			Expect(*searchRequest.Search.Query.Bool.Must).To(HaveLen(1))

			// the only query should specify a term query for the resource uri
			term := (*searchRequest.Search.Query.Bool.Must)[0].(*filtering.Query).Term
			Expect((*term)["resourceVersion.version"]).To(Equal(expectedResourceUri))
		})

		It("should search for associated policy evaluations", func() {
			Expect(esClient.MultiSearchCallCount()).To(Equal(1))

			_, multiSearchRequest := esClient.MultiSearchArgsForCall(0)

			Expect(multiSearchRequest.Index).To(Equal(expectedEvaluationsAlias))

			Expect(multiSearchRequest.Searches).To(HaveLen(1))
			Expect(multiSearchRequest.Searches[0].Routing).To(Equal(expectedResourceEvaluationId))
			Expect(multiSearchRequest.Searches[0].Query.HasParent.ParentType).To(Equal(resourceEvaluationRelationName))
			Expect((*multiSearchRequest.Searches[0].Query.HasParent.Query.Term)["_id"]).To(Equal(expectedResourceEvaluationId))
		})

		It("should not attempt to parse a filter", func() {
			Expect(filterer.ParseExpressionCallCount()).To(Equal(0))
		})

		It("should return the resource evaluations and no error", func() {
			Expect(actualListResourceEvaluationsResponse.ResourceEvaluations).To(HaveLen(1))
			Expect(actualListResourceEvaluationsResponse.ResourceEvaluations[0].ResourceEvaluation).To(Equal(expectedResourceEvaluation))
			Expect(actualListResourceEvaluationsResponse.ResourceEvaluations[0].PolicyEvaluations).To(HaveLen(1))
			Expect(actualListResourceEvaluationsResponse.ResourceEvaluations[0].PolicyEvaluations[0]).To(Equal(expectedPolicyEvaluation))

			Expect(actualError).ToNot(HaveOccurred())
		})

		When("a filter is specified", func() {
			BeforeEach(func() {
				expectedListResourceEvaluationsRequest.Filter = fake.LetterN(10)

				expectedFilterQuery = &filtering.Query{
					Term: &filtering.Term{
						fake.LetterN(10): fake.LetterN(10),
					},
				}
				expectedFilterError = nil
			})

			It("should attempt to parse the filter", func() {
				Expect(filterer.ParseExpressionCallCount()).To(Equal(1))

				filter := filterer.ParseExpressionArgsForCall(0)
				Expect(filter).To(Equal(expectedListResourceEvaluationsRequest.Filter))
			})

			It("should use the filter query when searching", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))

				_, searchRequest := esClient.SearchArgsForCall(0)

				Expect(*searchRequest.Search.Query.Bool.Must).To(HaveLen(2))

				filterQuery := (*searchRequest.Search.Query.Bool.Must)[1].(*filtering.Query)
				Expect(filterQuery).To(Equal(expectedFilterQuery))
			})

			When("an error occurs while attempting to parse the filter", func() {
				BeforeEach(func() {
					expectedFilterError = errors.New("error parsing filter")
				})

				It("should not attempt a search", func() {
					Expect(esClient.SearchCallCount()).To(Equal(0))
				})

				It("should return an error", func() {
					Expect(actualListResourceEvaluationsResponse).To(BeNil())
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				})
			})
		})

		When("an error occurs while searching", func() {
			BeforeEach(func() {
				expectedSearchError = errors.New("error searching")
			})

			It("should return an error", func() {
				Expect(actualListResourceEvaluationsResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("pagination is used", func() {
			BeforeEach(func() {
				expectedListResourceEvaluationsRequest.PageSize = int32(fake.Number(1, 10))
				expectedListResourceEvaluationsRequest.PageToken = fake.LetterN(10)
			})

			It("should use pagination when searching", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))

				_, searchRequest := esClient.SearchArgsForCall(0)

				Expect(searchRequest.Pagination).ToNot(BeNil())
				Expect(searchRequest.Pagination.Size).To(BeEquivalentTo(expectedListResourceEvaluationsRequest.PageSize))
				Expect(searchRequest.Pagination.Token).To(Equal(expectedListResourceEvaluationsRequest.PageToken))
			})
		})

		When("the resource uri is not specified", func() {
			BeforeEach(func() {
				expectedListResourceEvaluationsRequest.ResourceUri = ""
			})

			It("should return an error", func() {
				Expect(actualListResourceEvaluationsResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not perform a search", func() {
				Expect(esClient.SearchCallCount()).To(Equal(0))
			})
		})

		When("an error occurs while fetching the resource version", func() {
			BeforeEach(func() {
				expectedGetResourceVersionError = errors.New("error searching for resource version")
			})

			It("should return an error", func() {
				Expect(actualListResourceEvaluationsResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not perform a search", func() {
				Expect(esClient.SearchCallCount()).To(Equal(0))
			})
		})

		When("no resource evaluations are found", func() {
			BeforeEach(func() {
				expectedSearchResponse.Hits.Total.Value = 0
				expectedSearchResponse.Hits.Hits = []*esutil.EsSearchResponseHit{}
			})

			It("should not perform a multisearch", func() {
				Expect(esClient.MultiSearchCallCount()).To(BeZero())
			})

			It("should return a response with an empty list", func() {
				Expect(actualListResourceEvaluationsResponse.ResourceEvaluations).To(BeEmpty())
				Expect(actualError).ToNot(HaveOccurred())
			})
		})
	})

	Context("GetResourceEvaluation", func() {
		var (
			actualResourceEvaluationResult *pb.ResourceEvaluationResult
			actualError                    error

			expectedMultiSearchResponse *esutil.EsMultiSearchResponse
			expectedMultiSearchError    error

			expectedResourceEvaluation *pb.ResourceEvaluation
			expectedPolicyEvaluation   *pb.PolicyEvaluation

			expectedResourceEvaluationId string
		)

		BeforeEach(func() {
			expectedResourceEvaluationId = fake.UUID()
			expectedResourceEvaluation = &pb.ResourceEvaluation{
				Id:   expectedResourceEvaluationId,
				Pass: fake.Bool(),
			}

			expectedPolicyEvaluation = &pb.PolicyEvaluation{
				Id:                   fake.UUID(),
				ResourceEvaluationId: expectedResourceEvaluationId,
				Pass:                 fake.Bool(),
			}

			resourceEvaluationJson, _ := protojson.Marshal(expectedResourceEvaluation)
			policyEvaluationJson, _ := protojson.Marshal(expectedPolicyEvaluation)

			expectedMultiSearchResponse = &esutil.EsMultiSearchResponse{
				Responses: []*esutil.EsMultiSearchResponseHitsSummary{
					{
						Hits: &esutil.EsMultiSearchResponseHits{
							Total: &esutil.EsSearchResponseTotal{
								Value: 1,
							},
							Hits: []*esutil.EsMultiSearchResponseHit{
								{
									Source: resourceEvaluationJson,
								},
							},
						},
					},
					{
						Hits: &esutil.EsMultiSearchResponseHits{
							Total: &esutil.EsSearchResponseTotal{
								Value: 1,
							},
							Hits: []*esutil.EsMultiSearchResponseHit{
								{
									Source: policyEvaluationJson,
								},
							},
						},
					},
				},
			}
			expectedMultiSearchError = nil
		})

		JustBeforeEach(func() {
			esClient.MultiSearchReturns(expectedMultiSearchResponse, expectedMultiSearchError)

			actualResourceEvaluationResult, actualError = manager.GetResourceEvaluation(ctx, &pb.GetResourceEvaluationRequest{
				Id: expectedResourceEvaluationId,
			})
		})

		It("should perform a multi search for the resource evaluation and child policy evaluations", func() {
			Expect(esClient.MultiSearchCallCount()).To(Equal(1))

			_, searchRequest := esClient.MultiSearchArgsForCall(0)

			Expect(searchRequest.Index).To(Equal(expectedEvaluationsAlias))
			Expect(searchRequest.Searches).To(HaveLen(2))

			resourceEvaluationSearch := searchRequest.Searches[0]

			Expect((*resourceEvaluationSearch.Query.Term)["_id"]).To(Equal(expectedResourceEvaluationId))

			policyEvaluationSearch := searchRequest.Searches[1]

			Expect(policyEvaluationSearch.Query.HasParent.ParentType).To(Equal(resourceEvaluationRelationName))
			Expect(policyEvaluationSearch.Routing).To(Equal(expectedResourceEvaluationId))
			Expect((*policyEvaluationSearch.Query.HasParent.Query.Term)["_id"]).To(Equal(expectedResourceEvaluationId))
		})

		It("should return the result and no error", func() {
			Expect(actualResourceEvaluationResult.ResourceEvaluation).To(Equal(expectedResourceEvaluation))
			Expect(actualResourceEvaluationResult.PolicyEvaluations).To(HaveLen(1))
			Expect(actualResourceEvaluationResult.PolicyEvaluations[0]).To(Equal(expectedPolicyEvaluation))
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("an error occurs while searching for the resource evaluation", func() {
			BeforeEach(func() {
				expectedMultiSearchError = errors.New("error performing msearch")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the resource evaluation is not found", func() {
			BeforeEach(func() {
				expectedMultiSearchResponse.Responses[0].Hits.Total.Value = 0
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.NotFound))
			})
		})

		When("the resource evaluation json is invalid", func() {
			BeforeEach(func() {
				expectedMultiSearchResponse.Responses[0].Hits.Hits[0].Source = []byte("invalid json")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy evaluation json is invalid", func() {
			BeforeEach(func() {
				expectedMultiSearchResponse.Responses[1].Hits.Hits[0].Source = []byte("invalid json")
			})

			It("should return an error", func() {
				Expect(actualResourceEvaluationResult).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("EvaluatePolicy", func() {
		var (
			policyId        string
			policyVersionId string
			version         uint32
			policy          *pb.Policy
			getPolicyError  error

			resourceUri string
			request     *pb.EvaluatePolicyRequest

			opaInitializePolicyError opa.ClientError

			opaEvaluatePolicyResponse *opa.EvaluatePolicyResult
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
			policyVersionId = policy.Policy.Id

			opaInitializePolicyError = nil

			listVersionedResourceOccurrencesResponse = []*grafeas_proto.Occurrence{
				createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
				createRandomOccurrence(grafeas_common_proto.NoteKind_ATTESTATION),
			}
			listVersionedResourceOccurrencesError = nil

			opaEvaluatePolicyResponse = &opa.EvaluatePolicyResult{
				Pass:       true,
				Violations: []*pb.EvaluatePolicyViolation{},
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

			It("should initialize the versioned policy in Open Policy Agent", func() {
				Expect(opaClient.InitializePolicyCallCount()).To(Equal(1))

				_, actualPolicyVersionId, policyContent := opaClient.InitializePolicyArgsForCall(0)

				Expect(actualPolicyVersionId).To(Equal(policyVersionId))
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
				_, actualPolicyVersionId, actualInput := opaClient.EvaluatePolicyArgsForCall(0)

				expectedInputJson, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(&pb.EvaluatePolicyInput{
					Occurrences: listVersionedResourceOccurrencesResponse,
				})
				Expect(err).NotTo(HaveOccurred())
				actualInputJson, err := json.Marshal(actualInput)
				Expect(err).NotTo(HaveOccurred())

				Expect(actualPolicyVersionId).To(Equal(policyVersionId))
				Expect(actualInputJson).To(MatchJSON(expectedInputJson))
			})

			It("should return the evaluation results", func() {
				Expect(actualResponse).NotTo(BeNil())
				Expect(actualResponse.Pass).To(BeTrue())
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
				opaEvaluatePolicyResponse = nil
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
				opaEvaluatePolicyResponse.Violations = expectedViolations
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
		Id:          fmt.Sprintf("%s.%d", fake.UUID(), fake.Number(1, 10)),
		Version:     version,
		RegoContent: policy,
		SourcePath:  fake.URL(),
		Message:     fake.Word(),
	}
}
