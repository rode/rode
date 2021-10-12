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

package policy

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	immocks "github.com/rode/es-index-manager/mocks"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil/esutilfakes"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering/filteringfakes"
	"github.com/rode/rode/config"
	"github.com/rode/rode/pkg/constants"
	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("AssignmentManager", func() {

	var (
		ctx                            = context.Background()
		expectedPolicyAssignmentsAlias string
		expectedPoliciesAlias          string
		expectedPolicyGroupsAlias      string

		esClient     *esutilfakes.FakeClient
		esConfig     *config.ElasticsearchConfig
		indexManager *immocks.FakeIndexManager
		filterer     *filteringfakes.FakeFilterer

		manager AssignmentManager
	)

	BeforeEach(func() {
		indexManager = &immocks.FakeIndexManager{}
		esClient = &esutilfakes.FakeClient{}
		filterer = &filteringfakes.FakeFilterer{}
		esConfig = randomEsConfig()

		expectedPolicyAssignmentsAlias = fake.LetterN(10)
		expectedPoliciesAlias = fake.LetterN(10)
		expectedPolicyGroupsAlias = fake.LetterN(10)
		indexManager.AliasNameStub = func(documentKind string, _ string) string {
			return map[string]string{
				constants.PoliciesDocumentKind:          expectedPoliciesAlias,
				constants.PolicyGroupsDocumentKind:      expectedPolicyGroupsAlias,
				constants.PolicyAssignmentsDocumentKind: expectedPolicyAssignmentsAlias,
			}[documentKind]
		}

		manager = NewAssignmentManager(logger, esClient, esConfig, indexManager, filterer)
	})

	Context("CreatePolicyAssignment", func() {
		var (
			policyId     string
			policyGroup  string
			assignmentId string
			assignment   *pb.PolicyAssignment

			getAssignmentResponse *esutil.EsGetResponse
			getAssignmentError    error

			multiGetResponse *esutil.EsMultiGetResponse
			multiGetError    error

			createAssigmentError error

			actualAssignment *pb.PolicyAssignment
			actualError      error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			policyGroup = fake.Word()
			assignmentId = fmt.Sprintf("policies/%s/assignments/%s", policyId, policyGroup)

			assignment = &pb.PolicyAssignment{
				PolicyVersionId: fmt.Sprintf("%s.%d", policyId, fake.Number(1, 10)),
				PolicyGroup:     policyGroup,
			}

			getAssignmentResponse = &esutil.EsGetResponse{
				Id:    assignmentId,
				Found: false,
			}
			getAssignmentError = nil

			policyJson, _ := protojson.Marshal(&pb.Policy{Id: policyId})
			policyGroupJson, _ := protojson.Marshal(&pb.PolicyGroup{
				Name: policyGroup,
			})

			multiGetResponse = &esutil.EsMultiGetResponse{
				Docs: []*esutil.EsGetResponse{
					{
						Id:     policyId,
						Found:  true,
						Source: policyJson,
					},
					{
						Id:    assignment.PolicyVersionId,
						Found: true,
					},
					{
						Id:     policyGroup,
						Found:  true,
						Source: policyGroupJson,
					},
				},
			}
			multiGetError = nil
			createAssigmentError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getAssignmentResponse, getAssignmentError)
			esClient.MultiGetReturns(multiGetResponse, multiGetError)
			esClient.CreateReturns(assignmentId, createAssigmentError)

			actualAssignment, actualError = manager.CreatePolicyAssignment(ctx, deepCopyPolicyAssignment(assignment))
		})

		It("should check if the assignment already exists", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, actualRequest := esClient.GetArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
			Expect(actualRequest.DocumentId).To(Equal(assignmentId))
		})

		It("should see if the policy version and group exist", func() {
			Expect(esClient.MultiGetCallCount()).To(Equal(1))
			_, actualRequest := esClient.MultiGetArgsForCall(0)

			Expect(actualRequest.Index).To(BeEmpty())
			Expect(actualRequest.Items).To(HaveLen(3))

			Expect(actualRequest.Items[0].Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.Items[0].Id).To(Equal(policyId))

			Expect(actualRequest.Items[1].Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.Items[1].Id).To(Equal(assignment.PolicyVersionId))
			Expect(actualRequest.Items[1].Routing).To(Equal(policyId))

			Expect(actualRequest.Items[2].Index).To(Equal(expectedPolicyGroupsAlias))
			Expect(actualRequest.Items[2].Id).To(Equal(assignment.PolicyGroup))
		})

		It("should insert the assignment into Elasticsearch", func() {
			Expect(esClient.CreateCallCount()).To(Equal(1))

			_, actualRequest := esClient.CreateArgsForCall(0)

			Expect(actualRequest.DocumentId).To(Equal(assignmentId))
			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
			Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))

			actualMessage := actualRequest.Message.(*pb.PolicyAssignment)
			Expect(actualMessage.Id).To(Equal(assignmentId))
			Expect(actualMessage.PolicyGroup).To(Equal(assignment.PolicyGroup))
			Expect(actualMessage.PolicyVersionId).To(Equal(assignment.PolicyVersionId))
			Expect(actualMessage.Created.IsValid()).To(BeTrue())
			Expect(actualMessage.Updated.IsValid()).To(BeTrue())
		})

		It("should return the created assignment", func() {
			Expect(actualAssignment.Id).To(Equal(assignmentId))
			Expect(actualAssignment.PolicyGroup).To(Equal(assignment.PolicyGroup))
			Expect(actualAssignment.PolicyVersionId).To(Equal(assignment.PolicyVersionId))
			Expect(actualAssignment.Created.IsValid()).To(BeTrue())
			Expect(actualAssignment.Updated.IsValid()).To(BeTrue())
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("the assignment already exists", func() {
			BeforeEach(func() {
				existingAssignment := randomPolicyAssignment(assignmentId)
				assignmentJson, _ := protojson.Marshal(existingAssignment)

				getAssignmentResponse.Found = true
				getAssignmentResponse.Source = assignmentJson
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.AlreadyExists))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("the policy version does not exist", func() {
			BeforeEach(func() {
				multiGetResponse.Docs[0].Found = false
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("the policy group does not exist", func() {
			BeforeEach(func() {
				multiGetResponse.Docs[1].Found = false
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("the policy group is empty", func() {
			BeforeEach(func() {
				assignment.PolicyGroup = ""
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("the policy version id is empty", func() {
			BeforeEach(func() {
				assignment.PolicyVersionId = ""
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("a policy id is sent instead of a policy version id", func() {
			BeforeEach(func() {
				assignment.PolicyVersionId = fake.UUID()
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("the policy version id is invalid", func() {
			BeforeEach(func() {
				assignment.PolicyVersionId = fmt.Sprintf("%s.%s", fake.Word(), fake.Word())
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs checking for an existing assignment", func() {
			BeforeEach(func() {
				getAssignmentError = errors.New("get error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs fetching the policy version and group", func() {
			BeforeEach(func() {
				multiGetError = errors.New("mget error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to create a new assignment", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs creating the assignment in Elasticsearch", func() {
			BeforeEach(func() {
				createAssigmentError = errors.New("create error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy group has been deleted", func() {
			BeforeEach(func() {
				multiGetResponse.Docs[2].Source, _ = protojson.Marshal(&pb.PolicyGroup{
					Name:    policyGroup,
					Deleted: true,
				})
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})
		})

		When("the policy has been deleted", func() {
			BeforeEach(func() {
				multiGetResponse.Docs[0].Source, _ = protojson.Marshal(&pb.Policy{
					Id:      policyId,
					Deleted: true,
				})
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})
		})
	})

	Context("GetPolicyAssignment", func() {
		var (
			expectedId         string
			expectedAssignment *pb.PolicyAssignment

			getAssignmentResponse *esutil.EsGetResponse
			getAssignmentError    error

			actualAssignment *pb.PolicyAssignment
			actualError      error
		)

		BeforeEach(func() {
			expectedId = randomPolicyAssignmentId()
			expectedAssignment = randomPolicyAssignment(expectedId)

			assignmentJson, _ := protojson.Marshal(expectedAssignment)
			getAssignmentResponse = &esutil.EsGetResponse{
				Id:     expectedId,
				Found:  true,
				Source: assignmentJson,
			}
			getAssignmentError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getAssignmentResponse, getAssignmentError)

			actualAssignment, actualError = manager.GetPolicyAssignment(ctx, &pb.GetPolicyAssignmentRequest{
				Id: expectedId,
			})
		})

		It("should lookup the policy assignments alias", func() {
			Expect(indexManager.AliasNameCallCount()).To(Equal(1))

			actualDocumentKind, actualInner := indexManager.AliasNameArgsForCall(0)

			Expect(actualDocumentKind).To(Equal(constants.PolicyAssignmentsDocumentKind))
			Expect(actualInner).To(BeEmpty())
		})

		It("should fetch the assignment by its id", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, actualRequest := esClient.GetArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
			Expect(actualRequest.DocumentId).To(Equal(expectedId))
		})

		It("should return the assignment", func() {
			Expect(actualAssignment).To(Equal(expectedAssignment))
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("the call to Elasticsearch is unsuccessful", func() {
			BeforeEach(func() {
				getAssignmentError = errors.New("get error")
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the assignment is not found", func() {
			BeforeEach(func() {
				getAssignmentResponse.Found = false
				getAssignmentResponse.Source = nil
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.NotFound))
			})
		})

		When("the document source is invalid", func() {
			BeforeEach(func() {
				getAssignmentResponse.Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the assignment is is not set", func() {
			BeforeEach(func() {
				expectedId = ""
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})
		})
	})

	Context("UpdatePolicyAssignment", func() {
		var (
			policyId           string
			assignmentId       string
			currentAssignment  *pb.PolicyAssignment
			updatedAssignment  *pb.PolicyAssignment
			newPolicyVersionId string

			getAssignmentResponse *esutil.EsGetResponse
			getAssignmentError    error
			multiGetResponse      *esutil.EsMultiGetResponse
			multiGetError         error

			updateAssignmentError error

			actualAssignment *pb.PolicyAssignment
			actualError      error
		)

		BeforeEach(func() {
			assignmentId = randomPolicyAssignmentId()
			policyId = fake.UUID()
			currentAssignment = randomPolicyAssignment(assignmentId)
			currentAssignment.PolicyVersionId = fmt.Sprintf("%s.%d", policyId, fake.Number(1, 5))

			newPolicyVersionId = fmt.Sprintf("%s.%d", policyId, fake.Number(6, 10))
			updatedAssignment = deepCopyPolicyAssignment(currentAssignment)
			updatedAssignment.PolicyVersionId = newPolicyVersionId

			assignmentJson, _ := protojson.Marshal(currentAssignment)
			getAssignmentResponse = &esutil.EsGetResponse{
				Id:     assignmentId,
				Found:  true,
				Source: assignmentJson,
			}
			getAssignmentError = nil

			policyJson, _ := protojson.Marshal(&pb.Policy{Id: policyId})
			policyGroupJson, _ := protojson.Marshal(&pb.PolicyGroup{
				Name: currentAssignment.PolicyGroup,
			})
			multiGetResponse = &esutil.EsMultiGetResponse{
				Docs: []*esutil.EsGetResponse{
					{
						Id:     policyId,
						Found:  true,
						Source: policyJson,
					},
					{
						Id:    newPolicyVersionId,
						Found: true,
					},
					{
						Id:     currentAssignment.PolicyGroup,
						Found:  true,
						Source: policyGroupJson,
					},
				},
			}
			multiGetError = nil

			updateAssignmentError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getAssignmentResponse, getAssignmentError)
			esClient.MultiGetReturns(multiGetResponse, multiGetError)
			esClient.UpdateReturns(nil, updateAssignmentError)

			actualAssignment, actualError = manager.UpdatePolicyAssignment(ctx, updatedAssignment)
		})

		It("should fetch the existing policy assignment", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, actualRequest := esClient.GetArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
			Expect(actualRequest.DocumentId).To(Equal(assignmentId))
		})

		It("should fetch the policy group and versioned policy", func() {
			Expect(esClient.MultiGetCallCount()).To(Equal(1))

			_, actualRequest := esClient.MultiGetArgsForCall(0)

			Expect(actualRequest.Items[0].Id).To(Equal(policyId))
			Expect(actualRequest.Items[0].Index).To(Equal(expectedPoliciesAlias))

			Expect(actualRequest.Items[1].Id).To(Equal(newPolicyVersionId))
			Expect(actualRequest.Items[1].Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.Items[1].Routing).To(Equal(policyId))

			Expect(actualRequest.Items[2].Id).To(Equal(currentAssignment.PolicyGroup))
			Expect(actualRequest.Items[2].Index).To(Equal(expectedPolicyGroupsAlias))
		})

		It("should update the policy version id in Elasticsearch", func() {
			Expect(esClient.UpdateCallCount()).To(Equal(1))

			_, actualRequest := esClient.UpdateArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
			Expect(actualRequest.DocumentId).To(Equal(assignmentId))
			Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))
			actualMessage := actualRequest.Message.(*pb.PolicyAssignment)

			Expect(actualMessage.Id).To(Equal(assignmentId))
			Expect(actualMessage.PolicyVersionId).To(Equal(newPolicyVersionId))
			Expect(actualMessage.PolicyGroup).To(Equal(currentAssignment.PolicyGroup))
			Expect(actualMessage.Updated).NotTo(Equal(currentAssignment.Updated))
		})

		It("should return the updated assignment", func() {
			Expect(actualAssignment).NotTo(BeNil())
			Expect(actualAssignment.Id).To(Equal(assignmentId))
			Expect(actualAssignment.PolicyVersionId).To(Equal(newPolicyVersionId))
			Expect(actualAssignment.PolicyGroup).To(Equal(currentAssignment.PolicyGroup))
			Expect(actualAssignment.Created.IsValid()).To(BeTrue())
			Expect(actualAssignment.Updated.IsValid()).To(BeTrue())
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("the policy version id cannot be parsed", func() {
			BeforeEach(func() {
				updatedAssignment.PolicyVersionId = fmt.Sprintf("%s.%s", fake.Word(), fake.Word())
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to update the assignment", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("a policy id is passed instead of a policy version id", func() {
			BeforeEach(func() {
				updatedAssignment.PolicyVersionId = fake.UUID()
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not attempt to update the assignment", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("the policy version does not exist", func() {
			BeforeEach(func() {
				multiGetResponse.Docs[1].Found = false
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})

			It("should not attempt to update the assignment", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("the update tries to change the policy id", func() {
			BeforeEach(func() {
				updatedAssignment.PolicyVersionId = fmt.Sprintf("%s.%d", fake.UUID(), fake.Number(1, 5))
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not attempt to update the assignment", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs fetching the existing policy assignment", func() {
			BeforeEach(func() {
				getAssignmentError = errors.New("error")
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to update the assignment", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs fetching the versioned policy and policy group", func() {
			BeforeEach(func() {
				multiGetError = errors.New("multi-get error")
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to update the assignment", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("the update is for the policy group", func() {
			BeforeEach(func() {
				updatedAssignment.PolicyGroup = fake.Word()
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not attempt to update the assignment", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("the call to update is unsuccessful", func() {
			BeforeEach(func() {
				updateAssignmentError = errors.New("update failed")
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy group has been deleted", func() {
			BeforeEach(func() {
				multiGetResponse.Docs[2].Source, _ = protojson.Marshal(&pb.PolicyGroup{
					Name:    currentAssignment.PolicyGroup,
					Deleted: true,
				})
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})
		})

		When("the policy has been deleted", func() {
			BeforeEach(func() {
				multiGetResponse.Docs[0].Source, _ = protojson.Marshal(&pb.Policy{
					Id:      policyId,
					Deleted: true,
				})
			})

			It("should return an error", func() {
				Expect(actualAssignment).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})
		})
	})

	Context("DeletePolicyAssignment", func() {
		var (
			assignmentId string

			getAssignmentResponse *esutil.EsGetResponse
			getAssignmentError    error
			deleteAssignmentError error

			actualResponse *emptypb.Empty
			actualError    error
		)

		BeforeEach(func() {
			assignmentId = randomPolicyAssignmentId()

			assignmentJson, _ := protojson.Marshal(randomPolicyAssignment(assignmentId))
			getAssignmentResponse = &esutil.EsGetResponse{
				Id:     assignmentId,
				Found:  true,
				Source: assignmentJson,
			}
			getAssignmentError = nil
			deleteAssignmentError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getAssignmentResponse, getAssignmentError)
			esClient.DeleteReturns(deleteAssignmentError)

			actualResponse, actualError = manager.DeletePolicyAssignment(ctx, &pb.DeletePolicyAssignmentRequest{
				Id: assignmentId,
			})
		})

		It("should check that the assignment exists", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, actualRequest := esClient.GetArgsForCall(0)
			Expect(actualRequest.DocumentId).To(Equal(assignmentId))
			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
		})

		It("should delete the assignment", func() {
			Expect(esClient.DeleteCallCount()).To(Equal(1))

			_, actualRequest := esClient.DeleteArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
			Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))
			Expect((*actualRequest.Search.Query.Term)["_id"]).To(Equal(assignmentId))
		})

		It("should not return an error", func() {
			Expect(actualResponse).To(Equal(&emptypb.Empty{}))
			Expect(actualError).To(BeNil())
		})

		When("an error occurs deleting the assignment", func() {
			BeforeEach(func() {
				deleteAssignmentError = errors.New("delete error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("fetching the assignment returns an error", func() {
			BeforeEach(func() {
				getAssignmentError = errors.New("get error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt to delete the assignment", func() {
				Expect(esClient.DeleteCallCount()).To(Equal(0))
			})
		})
	})

	Context("ListPolicyAssignments", func() {
		var (
			request *pb.ListPolicyAssignmentsRequest

			assignmentCount     int
			expectedAssignments []*pb.PolicyAssignment
			searchResponse      *esutil.SearchResponse
			searchError         error

			actualResponse *pb.ListPolicyAssignmentsResponse
			actualError    error
			filterQuery    *filtering.Query
			filterError    error
		)

		BeforeEach(func() {
			request = &pb.ListPolicyAssignmentsRequest{}

			searchResponse = &esutil.SearchResponse{
				Hits: &esutil.EsSearchResponseHits{},
			}
			searchError = nil

			expectedAssignments = []*pb.PolicyAssignment{}
			assignmentCount = fake.Number(2, 5)
			for i := 0; i < assignmentCount; i++ {
				assignmentId := randomPolicyAssignmentId()
				assignment := randomPolicyAssignment(assignmentId)
				expectedAssignments = append(expectedAssignments, assignment)

				assignmentJson, _ := protojson.Marshal(assignment)
				searchResponse.Hits.Hits = append(searchResponse.Hits.Hits, &esutil.EsSearchResponseHit{
					ID:     assignmentId,
					Source: assignmentJson,
				})
			}

			filterQuery = nil
			filterError = nil
		})

		JustBeforeEach(func() {
			filterer.ParseExpressionReturns(filterQuery, filterError)
			esClient.SearchReturns(searchResponse, searchError)

			actualResponse, actualError = manager.ListPolicyAssignments(ctx, request)
		})

		It("should not parse the filter", func() {
			Expect(filterer.ParseExpressionCallCount()).To(Equal(0))
		})

		It("should perform a search against the assignments index", func() {
			Expect(esClient.SearchCallCount()).To(Equal(1))

			_, actualRequest := esClient.SearchArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyAssignmentsAlias))
			Expect(actualRequest.Pagination).To(BeNil())
			Expect(actualRequest.Search.Sort["created"]).To(Equal(esutil.EsSortOrderDescending))
			Expect(*actualRequest.Search.Query.Bool.Must).To(BeEmpty())
		})

		It("should return the assignments", func() {
			Expect(actualResponse.PolicyAssignments).To(Equal(expectedAssignments))
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("pagination options are specified", func() {
			var (
				nextPageToken string
				pageSize      int32
				pageToken     string
			)

			BeforeEach(func() {
				nextPageToken = fake.Word()
				pageSize = int32(fake.Number(10, 100))
				pageToken = fake.Word()

				searchResponse.NextPageToken = nextPageToken

				request.PageSize = pageSize
				request.PageToken = pageToken
			})

			It("should include the page size and token in the search request", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))
				_, actualRequest := esClient.SearchArgsForCall(0)

				Expect(actualRequest.Pagination).NotTo(BeNil())
				Expect(actualRequest.Pagination.Size).To(BeEquivalentTo(pageSize))
				Expect(actualRequest.Pagination.Token).To(Equal(pageToken))
			})

			It("should return the next page token", func() {
				Expect(actualResponse.NextPageToken).To(Equal(nextPageToken))
			})
		})

		When("a policy group is specified", func() {
			var expectedPolicyGroup string

			BeforeEach(func() {
				expectedPolicyGroup = fake.Word()
				request.PolicyGroup = expectedPolicyGroup
			})

			It("should include the policy group in the query", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))
				_, actualRequest := esClient.SearchArgsForCall(0)

				actualQuery := (*actualRequest.Search.Query.Bool.Must)[0].(*filtering.Query)

				Expect((*actualQuery.Term)["policyGroup"]).To(Equal(expectedPolicyGroup))
			})
		})

		When("a policy id is passed", func() {
			var expectedPolicyId string

			BeforeEach(func() {
				expectedPolicyId = fake.UUID()
				request.PolicyId = expectedPolicyId
			})

			It("should include the policy id in the query as a prefix match", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))
				_, actualRequest := esClient.SearchArgsForCall(0)

				actualQuery := (*actualRequest.Search.Query.Bool.Must)[0].(*filtering.Query)

				Expect((*actualQuery.Prefix)["policyVersionId"]).To(Equal(expectedPolicyId + "."))
			})
		})

		When("a filter is applied", func() {
			var expectedFilter string

			BeforeEach(func() {
				expectedFilter = fake.Word()

				request.Filter = expectedFilter
				filterQuery = &filtering.Query{
					Term: &filtering.Term{
						fake.Word(): fake.Word(),
					},
				}
			})

			It("should parse the filter", func() {
				Expect(filterer.ParseExpressionCallCount()).To(Equal(1))

				actualFilter := filterer.ParseExpressionArgsForCall(0)

				Expect(actualFilter).To(Equal(expectedFilter))
			})

			It("should include the filter query in the search", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))
				_, actualRequest := esClient.SearchArgsForCall(0)

				actualQuery := (*actualRequest.Search.Query.Bool.Must)[0].(*filtering.Query)

				Expect(actualQuery).To(Equal(filterQuery))
			})

			When("an error occurs parsing the filter", func() {
				BeforeEach(func() {
					filterError = errors.New("filter error")
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				})

				It("should not perform a search", func() {
					Expect(esClient.SearchCallCount()).To(Equal(0))
				})
			})
		})

		When("one of the assignment documents is malformed", func() {
			BeforeEach(func() {
				randomIndex := fake.Number(0, assignmentCount-1)

				searchResponse.Hits.Hits[randomIndex].Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the search is unsuccessful", func() {
			BeforeEach(func() {
				searchError = errors.New("search error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})
})

func randomPolicyAssignmentId() string {
	return fmt.Sprintf("policies/%s/assignments/%s", fake.UUID(), fake.Word())
}

func randomPolicyAssignment(id string) *pb.PolicyAssignment {
	return &pb.PolicyAssignment{
		Id:              id,
		PolicyVersionId: fmt.Sprintf("%s.%d", fake.UUID(), fake.Number(1, 10)),
		PolicyGroup:     fake.Word(),
		Created:         timestamppb.New(fake.Date()),
		Updated:         timestamppb.New(fake.Date()),
	}
}

func deepCopyPolicyAssignment(assignment *pb.PolicyAssignment) *pb.PolicyAssignment {
	return &pb.PolicyAssignment{
		Id:              assignment.Id,
		PolicyVersionId: assignment.PolicyVersionId,
		PolicyGroup:     assignment.PolicyGroup,
		Created:         assignment.Created,
		Updated:         assignment.Updated,
	}
}
