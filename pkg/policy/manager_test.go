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
	_ "embed"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	immocks "github.com/rode/es-index-manager/mocks"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil/esutilfakes"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering/filteringfakes"
	"github.com/rode/rode/config"
	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	//go:embed test/good.rego
	goodPolicy string
	//go:embed test/minimal.rego
	minimalPolicy string
	//go:embed test/missing_rode_fields.rego
	compilablePolicyMissingRodeFields string
	//go:embed test/missing_results_fields.rego
	compilablePolicyMissingResultsFields string
	//go:embed test/missing_results_return.rego
	compilablePolicyMissingResultsReturn string
	//go:embed test/uncompilable.rego
	uncompilablePolicy string
	unparseablePolicy  = `
		package play
		default hello = false
		hello
			m := input.message
			m == "world"
		}`
)

var _ = Describe("PolicyManager", func() {
	var (
		ctx                   = context.Background()
		expectedPoliciesAlias string

		esClient     *esutilfakes.FakeClient
		esConfig     *config.ElasticsearchConfig
		indexManager *immocks.FakeIndexManager
		filterer     *filteringfakes.FakeFilterer

		manager Manager
	)

	BeforeEach(func() {
		esClient = &esutilfakes.FakeClient{}
		indexManager = &immocks.FakeIndexManager{}
		filterer = &filteringfakes.FakeFilterer{}
		esConfig = randomEsConfig()

		expectedPoliciesAlias = fake.LetterN(10)
		indexManager.AliasNameReturns(expectedPoliciesAlias)

		manager = NewManager(logger, esClient, esConfig, indexManager, filterer)
	})

	Context("CreatePolicy", func() {
		var (
			policyId        string
			policyVersionId string
			version         uint32
			policy          *pb.Policy
			policyEntity    *pb.PolicyEntity

			bulkResponse      *esutil.EsBulkResponse
			bulkResponseError error

			actualPolicy *pb.Policy
			actualError  error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			version = 1 // initial version is always 1
			newUuid = func() uuid.UUID {
				return uuid.MustParse(policyId)
			}
			policyVersionId = fmt.Sprintf("%s.%d", policyId, version)

			policy = createRandomPolicy(fake.UUID(), version)
			policyEntity = createRandomPolicyEntity(goodPolicy, version)

			policy.Policy = policyEntity

			bulkResponse = &esutil.EsBulkResponse{
				Errors: false,
				Items: []*esutil.EsBulkResponseItem{
					{
						Create: &esutil.EsIndexDocResponse{
							Status: http.StatusOK,
						},
					},
					{
						Create: &esutil.EsIndexDocResponse{
							Status: http.StatusOK,
						},
					},
				},
			}
			bulkResponseError = nil
		})

		JustBeforeEach(func() {
			if esClient.BulkStub == nil {
				esClient.BulkReturns(bulkResponse, bulkResponseError)
			}

			actualPolicy, actualError = manager.CreatePolicy(ctx, deepCopyPolicy(policy))
		})

		When("the policy is valid", func() {
			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should send a bulk request to create the policy and its initial version", func() {
				Expect(esClient.BulkCallCount()).To(Equal(1))

				_, actualRequest := esClient.BulkArgsForCall(0)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))
				Expect(actualRequest.Items).To(HaveLen(3))

				createPolicyItem := actualRequest.Items[0]
				Expect(createPolicyItem.Operation).To(Equal(esutil.BULK_CREATE))
				Expect(createPolicyItem.DocumentId).To(Equal(policyId))
				Expect(createPolicyItem.Join.Field).To(Equal("join"))
				Expect(createPolicyItem.Join.Name).To(Equal("policy"))

				createCounterItem := actualRequest.Items[1]
				Expect(createCounterItem.Operation).To(Equal(esutil.BULK_CREATE))
				Expect(createCounterItem.DocumentId).To(Equal(policyId + ".counter"))
				Expect(createCounterItem.Message).To(Equal(&emptypb.Empty{}))
				Expect(createCounterItem.Routing).To(Equal(policyId))

				createPolicyEntityItem := actualRequest.Items[2]
				Expect(createPolicyEntityItem.Operation).To(Equal(esutil.BULK_CREATE))
				Expect(createPolicyEntityItem.DocumentId).To(Equal(policyVersionId))
				Expect(createPolicyEntityItem.Join.Field).To(Equal("join"))
				Expect(createPolicyEntityItem.Join.Name).To(Equal("version"))
			})

			Describe("policy content in parent document", func() {
				var actualPolicyMessage *pb.Policy

				BeforeEach(func() {
					esClient.BulkCalls(func(ctx context.Context, request *esutil.BulkRequest) (*esutil.EsBulkResponse, error) {
						message, ok := request.Items[0].Message.(*pb.Policy)
						Expect(ok).To(BeTrue())
						actualPolicyMessage = deepCopyPolicy(message)

						return bulkResponse, bulkResponseError
					})
				})

				It("should remove the policy content from the parent document", func() {
					Expect(actualPolicyMessage).NotTo(BeNil())
					Expect(actualPolicyMessage.Policy).To(BeNil())
				})
			})

			It("should return the policy at its current version", func() {
				Expect(actualPolicy.Id).To(Equal(policyId))
				Expect(actualPolicy.Name).To(Equal(policy.Name))
				Expect(actualPolicy.Description).To(Equal(policy.Description))
				Expect(actualPolicy.CurrentVersion).To(Equal(version))
				Expect(actualPolicy.Created.IsValid()).To(BeTrue())
				Expect(actualPolicy.Updated.IsValid()).To(BeTrue())

				Expect(actualPolicy.Policy).NotTo(BeNil())
				Expect(actualPolicy.Policy.Id).To(Equal(policyVersionId))
				Expect(actualPolicy.Policy.RegoContent).To(Equal(policyEntity.RegoContent))
				Expect(actualPolicy.Policy.SourcePath).To(Equal(policyEntity.SourcePath))
				Expect(actualPolicy.Policy.Created).To(Equal(actualPolicy.Created))
				Expect(actualPolicy.Policy.Version).To(Equal(version))
				Expect(actualPolicy.Policy.Message).To(Equal("Initial policy creation"))
			})
		})

		When("the policy is invalid", func() {
			BeforeEach(func() {
				policy.Policy.RegoContent = unparseablePolicy
			})

			It("should return an error with details", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.InvalidArgument))

				Expect(status.Details()).To(HaveLen(1))
				detailsMsg := status.Details()[0].(*pb.ValidatePolicyResponse)

				Expect(detailsMsg.Policy).To(Equal(policyEntity.RegoContent))
				Expect(detailsMsg.Compile).To(BeFalse())
				Expect(detailsMsg.Errors).To(HaveLen(1))
			})

			It("should not create any documents", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("validating the policy causes an error", func() {
			BeforeEach(func() {
				policy.Policy.RegoContent = unparseablePolicy
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not create any documents", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the policy name is unset", func() {
			BeforeEach(func() {
				policy.Name = ""
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not create any documents", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the policy entity is not set", func() {
			BeforeEach(func() {
				policy.Policy = nil
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not create any documents", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the bulk create fails", func() {
			BeforeEach(func() {
				bulkResponseError = errors.New("bulk error")
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy creation in the bulk request fails", func() {
			BeforeEach(func() {
				bulkResponse.Items[0].Create.Status = http.StatusInternalServerError
				bulkResponse.Items[0].Create.Error = &esutil.EsIndexDocError{
					Type:   fake.Word(),
					Reason: fake.Word(),
				}
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy entity creation in the bulk request fails", func() {
			BeforeEach(func() {
				bulkResponse.Items[1].Create.Status = http.StatusInternalServerError
				bulkResponse.Items[1].Create.Error = &esutil.EsIndexDocError{
					Type:   fake.Word(),
					Reason: fake.Word(),
				}
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())

				status := getGRPCStatusFromError(actualError)
				Expect(status.Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("GetPolicyVersion", func() {
		var (
			actualPolicyEntity *pb.PolicyEntity
			actualError        error

			expectedPolicyId        string
			expectedPolicyVersionId string

			expectedGetResponse *esutil.EsGetResponse
			expectedGetError    error

			expectedPolicyEntity  *pb.PolicyEntity
			expectedPolicyVersion uint32
		)

		BeforeEach(func() {
			expectedPolicyVersion = uint32(fake.Number(1, 10))
			expectedPolicyEntity = createRandomPolicyEntity(fake.LetterN(10), expectedPolicyVersion)

			expectedPolicyId = fake.UUID()
			expectedPolicyVersionId = fmt.Sprintf("%s.%d", expectedPolicyId, expectedPolicyVersion)

			policyEntityJson, _ := protojson.Marshal(expectedPolicyEntity)
			expectedGetResponse = &esutil.EsGetResponse{
				Id:     expectedPolicyVersionId,
				Found:  true,
				Source: policyEntityJson,
			}
			expectedGetError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(expectedGetResponse, expectedGetError)

			actualPolicyEntity, actualError = manager.GetPolicyVersion(ctx, expectedPolicyVersionId)
		})

		It("should query elasticsearch for the policy entity", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, getRequest := esClient.GetArgsForCall(0)

			Expect(getRequest.Routing).To(Equal(expectedPolicyId))
			Expect(getRequest.DocumentId).To(Equal(expectedPolicyVersionId))
			Expect(getRequest.Index).To(Equal(expectedPoliciesAlias))
		})

		It("should return the policy entity and no error", func() {
			Expect(actualPolicyEntity).To(Equal(expectedPolicyEntity))
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("the policy version id is invalid", func() {
			BeforeEach(func() {
				expectedPolicyVersionId = fake.LetterN(10) + "." + fake.LetterN(10)
			})

			It("should return an error", func() {
				Expect(actualPolicyEntity).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("an error occurs while fetching the policy version from ES", func() {
			BeforeEach(func() {
				expectedGetError = errors.New("get failed")
			})

			It("should return an error", func() {
				Expect(actualPolicyEntity).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the policy version is not found", func() {
			BeforeEach(func() {
				expectedGetResponse.Found = false
			})

			It("should return nil", func() {
				Expect(actualPolicyEntity).To(BeNil())
				Expect(actualError).ToNot(HaveOccurred())
			})
		})

		When("an error occurs while unmarshaling the policy entity", func() {
			BeforeEach(func() {
				expectedGetResponse.Source = []byte("invalid json")
			})

			It("should return an error", func() {
				Expect(actualPolicyEntity).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})
	})

	Context("GetPolicy", func() {
		var (
			policyId             string
			policyVersionId      string
			version              uint32
			request              *pb.GetPolicyRequest
			expectedPolicy       *pb.Policy
			expectedPolicyEntity *pb.PolicyEntity

			actualError  error
			actualPolicy *pb.Policy

			getPolicyResponse *esutil.EsGetResponse
			getPolicyError    error

			getPolicyEntityResponse *esutil.EsGetResponse
			getPolicyEntityError    error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			version = uint32(fake.Number(1, 10))
			policyVersionId = fmt.Sprintf("%s.%d", policyId, version)

			request = &pb.GetPolicyRequest{
				Id: policyId,
			}

			expectedPolicy = createRandomPolicy(policyId, version)
			policyJson, _ := protojson.Marshal(expectedPolicy)

			getPolicyResponse = &esutil.EsGetResponse{
				Id:     policyId,
				Found:  true,
				Source: policyJson,
			}
			getPolicyError = nil

			expectedPolicyEntity = createRandomPolicyEntity(goodPolicy, version)
			expectedPolicyEntity.Id = policyVersionId
			policyEntityJson, _ := protojson.Marshal(expectedPolicyEntity)
			getPolicyEntityResponse = &esutil.EsGetResponse{
				Id:     policyVersionId,
				Found:  true,
				Source: policyEntityJson,
			}
			getPolicyEntityError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturnsOnCall(0, getPolicyResponse, getPolicyError)
			esClient.GetReturnsOnCall(1, getPolicyEntityResponse, getPolicyEntityError)

			actualPolicy, actualError = manager.GetPolicy(ctx, request)
		})

		When("the policy exists", func() {
			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should query Elasticsearch for the policy", func() {
				Expect(indexManager.AliasNameCallCount()).To(Equal(2))

				actualDocumentKind, inner := indexManager.AliasNameArgsForCall(0)
				Expect(actualDocumentKind).To(Equal("policies"))
				Expect(inner).To(Equal(""))

				Expect(esClient.GetCallCount()).To(Equal(2))

				_, actualRequest := esClient.GetArgsForCall(0)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.DocumentId).To(Equal(policyId))
			})

			It("should query Elasticsearch for the versioned policy entity", func() {
				_, actualRequest := esClient.GetArgsForCall(1)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.DocumentId).To(Equal(policyVersionId))
				Expect(actualRequest.Routing).To(Equal(policyId))
			})

			It("should return the policy at its current version", func() {
				Expect(actualPolicy).NotTo(BeNil())
				Expect(actualPolicy.Id).To(Equal(policyId))
				Expect(actualPolicy.Name).To(Equal(expectedPolicy.Name))
				Expect(actualPolicy.Description).To(Equal(expectedPolicy.Description))
				Expect(actualPolicy.CurrentVersion).To(Equal(version))

				Expect(actualPolicy.Policy).NotTo(BeNil())
				Expect(actualPolicy.Policy.Id).To(Equal(policyVersionId))
				Expect(actualPolicy.Policy.Version).To(Equal(version))
				Expect(actualPolicy.Policy.RegoContent).To(Equal(goodPolicy))
			})
		})

		When("a policy version id is passed", func() {
			BeforeEach(func() {
				request.Id = policyVersionId
				expectedPolicy.CurrentVersion = fake.Uint32()
				policyJson, _ := protojson.Marshal(expectedPolicy)

				getPolicyResponse.Source = policyJson
			})

			It("should fetch the policy", func() {
				Expect(esClient.GetCallCount()).To(Equal(2))

				_, actualRequest := esClient.GetArgsForCall(0)

				Expect(actualRequest.DocumentId).To(Equal(policyId))
			})

			It("should fetch the policy at the specified version", func() {
				_, actualRequest := esClient.GetArgsForCall(1)

				Expect(actualRequest.DocumentId).To(Equal(policyVersionId))
				Expect(actualRequest.Routing).To(Equal(policyId))
			})
		})

		When("a policy version id is not a number", func() {
			BeforeEach(func() {
				request.Id = fmt.Sprintf("%s.%s", fake.UUID(), fake.Word())
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})
		})

		When("a policy version id does not have the expected number of components", func() {
			BeforeEach(func() {
				request.Id = fmt.Sprintf("%s..", fake.UUID())
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})
		})

		When("an error occurs fetching policy", func() {
			BeforeEach(func() {
				getPolicyError = errors.New("get policy error")
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to fetch the policy entity", func() {
				Expect(esClient.GetCallCount()).To(Equal(1))
			})
		})

		When("the policy is not found", func() {
			BeforeEach(func() {
				getPolicyResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.NotFound))
			})

			It("should not try to fetch the policy entity", func() {
				Expect(esClient.GetCallCount()).To(Equal(1))
			})
		})

		When("the policy document is invalid", func() {
			BeforeEach(func() {
				getPolicyResponse.Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to fetch the policy version", func() {
				Expect(esClient.GetCallCount()).To(Equal(1))
			})
		})

		When("an error occurs fetching the policy entity", func() {
			BeforeEach(func() {
				getPolicyEntityError = errors.New("get policy entity error")
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy entity is not found", func() {
			BeforeEach(func() {
				getPolicyEntityResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy entity document is invalid", func() {
			BeforeEach(func() {
				getPolicyEntityResponse.Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualPolicy).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("ListPolicies", func() {
		var (
			request *pb.ListPoliciesRequest

			expectedFilterQuery *filtering.Query
			expectedFilterError error

			searchResponse *esutil.SearchResponse
			searchError    error

			multiGetResponse *esutil.EsMultiGetResponse
			multiGetError    error

			policyCount    int
			policies       []*pb.Policy
			policyVersions []*pb.PolicyEntity

			actualResponse *pb.ListPoliciesResponse
			actualError    error
		)

		BeforeEach(func() {
			request = &pb.ListPoliciesRequest{}

			policies = []*pb.Policy{}
			policyVersions = []*pb.PolicyEntity{}
			policyCount = fake.Number(2, 5)

			searchResponse = &esutil.SearchResponse{
				Hits: &esutil.EsSearchResponseHits{},
			}
			searchError = nil

			multiGetResponse = &esutil.EsMultiGetResponse{}
			multiGetError = nil

			for i := 0; i < policyCount; i++ {
				policy := createRandomPolicy(fake.UUID(), fake.Uint32())
				policyVersion := createRandomPolicyEntity(goodPolicy, policy.CurrentVersion)

				policies = append(policies, policy)
				policyVersions = append(policyVersions, policyVersion)

				policyJson, _ := protojson.Marshal(policy)
				searchResponse.Hits.Hits = append(searchResponse.Hits.Hits, &esutil.EsSearchResponseHit{
					ID:     policy.Id,
					Source: policyJson,
				})

				versionJson, _ := protojson.Marshal(policyVersion)
				multiGetResponse.Docs = append(multiGetResponse.Docs, &esutil.EsGetResponse{
					Found:  true,
					Source: versionJson,
				})
			}
		})

		JustBeforeEach(func() {
			filterer.ParseExpressionReturns(expectedFilterQuery, expectedFilterError)
			esClient.SearchReturns(searchResponse, searchError)
			esClient.MultiGetReturns(multiGetResponse, multiGetError)

			actualResponse, actualError = manager.ListPolicies(ctx, request)
		})

		It("query for all policies", func() {
			Expect(indexManager.AliasNameCallCount()).To(Equal(2))
			Expect(esClient.SearchCallCount()).To(Equal(1))
			Expect(filterer.ParseExpressionCallCount()).To(Equal(0))

			_, actualRequest := esClient.SearchArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.Pagination).To(BeNil())
			Expect(actualRequest.Search.Sort["created"]).To(Equal(esutil.EsSortOrderDescending))
			Expect(*actualRequest.Search.Query.Bool.Must).To(HaveLen(2))

			actualJoinQuery := (*actualRequest.Search.Query.Bool.Must)[0].(*filtering.Query)
			Expect((*actualJoinQuery.Term)["join"]).To(Equal("policy"))

			actualSoftDeleteQuery := (*actualRequest.Search.Query.Bool.Must)[1].(*filtering.Query)
			Expect((*actualSoftDeleteQuery.Term)["deleted"]).To(Equal("false"))
		})

		It("should perform a multi-get to fetch current policy versions", func() {
			Expect(esClient.MultiGetCallCount()).To(Equal(1))
			_, actualRequest := esClient.MultiGetArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.DocumentIds).To(BeNil())

			var expectedMultiGetItems []*esutil.EsMultiGetItem
			for i := 0; i < policyCount; i++ {
				policy := policies[i]

				expectedMultiGetItems = append(expectedMultiGetItems, &esutil.EsMultiGetItem{
					Id:      fmt.Sprintf("%s.%d", policy.Id, policy.CurrentVersion),
					Routing: policy.Id,
				})
			}
			Expect(actualRequest.Items).To(ConsistOf(expectedMultiGetItems))
		})

		It("should not return an error", func() {
			Expect(actualError).To(BeNil())
		})

		It("should return the list of policies", func() {
			for i, policy := range policies {
				policy.Policy = policyVersions[i]
			}

			Expect(actualResponse).NotTo(BeNil())
			Expect(actualResponse.Policies).To(ConsistOf(policies))
		})

		When("a filter is applied", func() {
			var (
				expectedFilter string
			)

			BeforeEach(func() {
				expectedFilter = fake.LetterN(10)
				request.Filter = expectedFilter

				expectedFilterQuery = &filtering.Query{
					Term: &filtering.Term{
						fake.Word(): fake.Word(),
					},
				}
				expectedFilterError = nil
			})

			It("should parse the filter into a query expression", func() {
				Expect(filterer.ParseExpressionCallCount()).To(Equal(1))

				actualFilter := filterer.ParseExpressionArgsForCall(0)
				Expect(actualFilter).To(Equal(expectedFilter))
			})

			It("should include the filter query", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))
				_, actualRequest := esClient.SearchArgsForCall(0)

				Expect(*actualRequest.Search.Query.Bool.Must).To(HaveLen(3))
				actualFilterQuery := (*actualRequest.Search.Query.Bool.Must)[2].(*filtering.Query)
				Expect(actualFilterQuery).To(Equal(expectedFilterQuery))
			})

			When("an error occurs parsing the filter", func() {
				BeforeEach(func() {
					expectedFilterError = errors.New("parse error")
				})

				It("should return an error", func() {
					Expect(actualResponse).To(BeNil())
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				})

				It("should not try to search for policies or policy versions", func() {
					Expect(esClient.SearchCallCount()).To(Equal(0))
					Expect(esClient.MultiGetCallCount()).To(Equal(0))
				})
			})
		})

		When("a pagination options are specified", func() {
			var nextPageToken string

			BeforeEach(func() {
				nextPageToken = fake.Word()
				request.PageSize = int32(fake.Number(10, 100))
				request.PageToken = fake.Word()

				searchResponse.NextPageToken = nextPageToken
			})

			It("should include the page size and page token in the search request", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))
				_, actualRequest := esClient.SearchArgsForCall(0)

				Expect(actualRequest.Pagination).NotTo(BeNil())
				Expect(actualRequest.Pagination.Size).To(BeEquivalentTo(request.PageSize))
				Expect(actualRequest.Pagination.Token).To(Equal(request.PageToken))
			})

			It("should return the next page token", func() {
				Expect(actualResponse.NextPageToken).To(Equal(nextPageToken))
			})
		})

		When("there are no policies", func() {
			BeforeEach(func() {
				searchResponse.Hits.Hits = nil
			})

			It("should not return any policies", func() {
				Expect(actualResponse.Policies).To(BeEmpty())
			})

			It("should not try to fetch versions", func() {
				Expect(esClient.MultiGetCallCount()).To(Equal(0))
			})
		})

		When("an error occurs searching for policies", func() {
			BeforeEach(func() {
				searchError = errors.New("search error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to fetch policy versions", func() {
				Expect(esClient.MultiGetCallCount()).To(Equal(0))
			})
		})

		When("a policy document is malformed", func() {
			BeforeEach(func() {
				randomIndex := fake.Number(0, policyCount-1)

				searchResponse.Hits.Hits[randomIndex].Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to fetch policy versions", func() {
				Expect(esClient.MultiGetCallCount()).To(Equal(0))
			})
		})

		When("an error occurs search for policy versions", func() {
			BeforeEach(func() {
				multiGetError = errors.New("multiget error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("a policy version document is missing", func() {
			BeforeEach(func() {
				randomIndex := fake.Number(0, policyCount-1)

				multiGetResponse.Docs[randomIndex].Found = false
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("a policy version document is malformed", func() {
			BeforeEach(func() {
				randomIndex := fake.Number(0, policyCount-1)

				multiGetResponse.Docs[randomIndex].Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("ListPolicyVersions", func() {
		var (
			policyId       string
			policyVersions []*pb.PolicyEntity
			versionCount   int
			request        *pb.ListPolicyVersionsRequest

			searchResponse *esutil.SearchResponse
			searchError    error

			filterQuery *filtering.Query
			filterError error

			actualResponse *pb.ListPolicyVersionsResponse
			actualError    error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			request = &pb.ListPolicyVersionsRequest{
				Id: policyId,
			}
			policyVersions = []*pb.PolicyEntity{}
			versionCount = fake.Number(2, 5)
			searchResponse = &esutil.SearchResponse{
				Hits: &esutil.EsSearchResponseHits{
					Total: &esutil.EsSearchResponseTotal{
						Value: versionCount,
					},
				},
			}
			searchError = nil
			filterQuery = nil
			filterError = nil

			for i := 0; i < versionCount; i++ {
				version := fake.Uint32()
				policy := createRandomPolicyEntity(minimalPolicy, version)
				policy.Id = fmt.Sprintf("%s.%d", policyId, version)

				policyVersions = append(policyVersions, policy)
				versionJson, _ := protojson.Marshal(policy)
				searchResponse.Hits.Hits = append(searchResponse.Hits.Hits, &esutil.EsSearchResponseHit{
					ID:     policy.Id,
					Source: versionJson,
				})
			}
		})

		JustBeforeEach(func() {
			filterer.ParseExpressionReturns(filterQuery, filterError)
			esClient.SearchReturns(searchResponse, searchError)

			actualResponse, actualError = manager.ListPolicyVersions(ctx, request)
		})

		It("should not apply a filter", func() {
			Expect(filterer.ParseExpressionCallCount()).To(Equal(0))
		})

		It("should search for all versions of a policy", func() {
			Expect(esClient.SearchCallCount()).To(Equal(1))

			_, actualRequest := esClient.SearchArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.Pagination).To(BeNil())
			Expect(actualRequest.Search.Sort["created"]).To(Equal(esutil.EsSortOrderDescending))
			Expect(*actualRequest.Search.Query.Bool.Must).To(HaveLen(1))

			actualQuery := (*actualRequest.Search.Query.Bool.Must)[0].(*filtering.Query)
			Expect(actualQuery.HasParent.ParentType).To(Equal("policy"))
			Expect((*actualQuery.HasParent.Query.Term)["_id"]).To(Equal(policyId))
		})

		It("should return all versions for the given policy id", func() {
			Expect(actualResponse.Versions).To(ConsistOf(policyVersions))
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("a filter is supplied", func() {
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

			It("should parse the filter expression", func() {
				Expect(filterer.ParseExpressionCallCount()).To(Equal(1))

				actualFilter := filterer.ParseExpressionArgsForCall(0)

				Expect(actualFilter).To(Equal(expectedFilter))
			})

			It("should add the filter query to the search", func() {
				_, actualRequest := esClient.SearchArgsForCall(0)
				Expect(*actualRequest.Search.Query.Bool.Must).To(HaveLen(2))

				actualFilterQuery := (*actualRequest.Search.Query.Bool.Must)[1].(*filtering.Query)
				Expect(actualFilterQuery).To(Equal(filterQuery))
			})
		})

		When("the filter is invalid", func() {
			BeforeEach(func() {
				request.Filter = fake.Word()
				filterError = errors.New("parse error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not attempt a search", func() {
				Expect(esClient.SearchCallCount()).To(Equal(0))
			})
		})

		When("a pagination options are supplied", func() {
			var (
				pageSize      int32
				pageToken     string
				nextPageToken string
			)

			BeforeEach(func() {
				pageSize = int32(fake.Number(10, 100))
				pageToken = fake.Word()
				nextPageToken = fake.Word()

				request.PageSize = pageSize
				request.PageToken = pageToken
				searchResponse.NextPageToken = nextPageToken
			})

			It("should include the page size and token in the search", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))
				_, actualRequest := esClient.SearchArgsForCall(0)

				Expect(actualRequest.Pagination).NotTo(BeNil())
				Expect(actualRequest.Pagination.Token).To(Equal(pageToken))
				Expect(actualRequest.Pagination.Size).To(BeEquivalentTo(pageSize))
			})

			It("should include the next page token in the response", func() {
				Expect(actualResponse.NextPageToken).To(Equal(nextPageToken))
			})
		})

		When("the search fails", func() {
			BeforeEach(func() {
				searchError = errors.New("search error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("one of the version documents is invalid", func() {
			BeforeEach(func() {
				randomIndex := fake.Number(0, versionCount-1)

				searchResponse.Hits.Hits[randomIndex].Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("UpdatePolicy", func() {
		var (
			policyId       string
			currentVersion uint32
			newVersion     uint32

			currentPolicy        *pb.Policy
			currentPolicyVersion *pb.PolicyEntity

			updatedPolicy        *pb.Policy
			updatedPolicyVersion *pb.PolicyEntity

			getPolicyResponse *esutil.EsGetResponse
			getPolicyError    error

			getPolicyVersionResponse *esutil.EsGetResponse
			getPolicyVersionError    error

			updateResponse *esutil.EsIndexDocResponse
			updateError    error

			bulkResponse *esutil.EsBulkResponse
			bulkError    error

			request *pb.UpdatePolicyRequest

			actualResponse *pb.Policy
			actualError    error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			currentVersion = fake.Uint32()
			newVersion = fake.Uint32()

			currentPolicy = createRandomPolicy(policyId, currentVersion)
			currentPolicyVersion = createRandomPolicyEntity(goodPolicy, currentVersion)
			currentPolicy.Policy = currentPolicyVersion

			updatedPolicy = deepCopyPolicy(currentPolicy)
			updatedPolicy.Name = fake.Word()
			updatedPolicy.Description = fake.Word()
			updatedPolicyVersion = createRandomPolicyEntity(minimalPolicy, newVersion)
			updatedPolicy.Policy = updatedPolicyVersion

			policyJson, _ := protojson.Marshal(currentPolicy)
			getPolicyResponse = &esutil.EsGetResponse{
				Id:     policyId,
				Found:  true,
				Source: policyJson,
			}
			getPolicyError = nil

			versionJson, _ := protojson.Marshal(currentPolicyVersion)
			getPolicyVersionResponse = &esutil.EsGetResponse{
				Id:     fmt.Sprintf("%s.%d", policyId, currentVersion),
				Found:  true,
				Source: versionJson,
			}
			getPolicyVersionError = nil

			updateResponse = &esutil.EsIndexDocResponse{
				Id:      policyId + ".counter",
				Version: int(newVersion),
			}
			updateError = nil

			bulkResponse = &esutil.EsBulkResponse{
				Items: []*esutil.EsBulkResponseItem{
					{
						Index: &esutil.EsIndexDocResponse{
							Status: http.StatusOK,
						},
					},
					{
						Create: &esutil.EsIndexDocResponse{
							Status: http.StatusOK,
						},
					},
				},
				Errors: false,
			}
			bulkError = nil

			request = &pb.UpdatePolicyRequest{
				Policy: deepCopyPolicy(updatedPolicy),
			}
		})

		JustBeforeEach(func() {
			esClient.GetReturnsOnCall(0, getPolicyResponse, getPolicyError)
			esClient.GetReturnsOnCall(1, getPolicyVersionResponse, getPolicyVersionError)

			esClient.UpdateReturns(updateResponse, updateError)
			if esClient.BulkStub == nil {
				esClient.BulkReturns(bulkResponse, bulkError)
			}

			actualResponse, actualError = manager.UpdatePolicy(ctx, request)
		})

		It("should fetch the current policy and version", func() {
			Expect(esClient.GetCallCount()).To(Equal(2))

			_, actualGetPolicyRequest := esClient.GetArgsForCall(0)
			Expect(actualGetPolicyRequest.DocumentId).To(Equal(policyId))
			Expect(actualGetPolicyRequest.Index).To(Equal(expectedPoliciesAlias))
			Expect(actualGetPolicyRequest.Routing).To(BeEmpty())

			_, actualGetPolicyVersionRequest := esClient.GetArgsForCall(1)
			Expect(actualGetPolicyVersionRequest.DocumentId).To(Equal(fmt.Sprintf("%s.%d", policyId, currentVersion)))
			Expect(actualGetPolicyVersionRequest.Index).To(Equal(expectedPoliciesAlias))
			Expect(actualGetPolicyVersionRequest.Routing).To(Equal(policyId))
		})

		It("should update the counter document", func() {
			Expect(esClient.UpdateCallCount()).To(Equal(1))

			_, actualRequest := esClient.UpdateArgsForCall(0)

			Expect(actualRequest.Message).To(Equal(&emptypb.Empty{}))
			Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.DocumentId).To(Equal(fmt.Sprintf("%s.counter", policyId)))
			Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))
			Expect(actualRequest.Routing).To(Equal(policyId))
		})

		It("should send a bulk request to update the policy and create a new version", func() {
			Expect(esClient.BulkCallCount()).To(Equal(1))

			_, actualRequest := esClient.BulkArgsForCall(0)

			Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))
			Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
			Expect(actualRequest.Items).To(HaveLen(2))

			actualUpdate := actualRequest.Items[0]
			Expect(actualUpdate.Operation).To(Equal(esutil.BULK_INDEX))
			Expect(actualUpdate.DocumentId).To(Equal(policyId))
			Expect(actualUpdate.Join.Name).To(Equal("policy"))
			Expect(actualUpdate.Join.Field).To(Equal("join"))
			actualPolicy := actualUpdate.Message.(*pb.Policy)
			Expect(actualPolicy.CurrentVersion).To(Equal(newVersion))

			actualCreate := actualRequest.Items[1]
			Expect(actualCreate.DocumentId).To(Equal(fmt.Sprintf("%s.%d", policyId, newVersion)))
			Expect(actualCreate.Join.Field).To(Equal("join"))
			Expect(actualCreate.Join.Name).To(Equal("version"))
			Expect(actualCreate.Join.Parent).To(Equal(policyId))
			Expect(actualCreate.Operation).To(Equal(esutil.BULK_CREATE))
			actualPolicyVersion := actualCreate.Message.(*pb.PolicyEntity)

			Expect(actualPolicyVersion.RegoContent).To(Equal(minimalPolicy))
			Expect(actualPolicyVersion.Version).To(Equal(newVersion))
			Expect(actualPolicyVersion.Created.IsValid()).To(BeTrue())
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		It("should return the updated policy", func() {
			Expect(actualResponse.CurrentVersion).To(Equal(newVersion))
			Expect(actualResponse.Updated).NotTo(Equal(currentPolicy.Updated))

			Expect(actualResponse.Policy.Id).To(Equal(fmt.Sprintf("%s.%d", policyId, newVersion)))
			Expect(actualResponse.Policy.Version).To(Equal(newVersion))
			Expect(actualResponse.Policy.RegoContent).To(Equal(minimalPolicy))
			Expect(actualResponse.Policy.Created.IsValid()).To(BeTrue())
		})

		When("the policy was previously deleted", func() {
			BeforeEach(func() {
				currentPolicy.Deleted = true
				policyJson, _ := protojson.Marshal(currentPolicy)

				getPolicyResponse.Source = policyJson
			})

			It("should return a validation error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.FailedPrecondition))
			})

			It("should not update the counter", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})

			It("should not make a bulk request", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the policy doesn't exist", func() {
			BeforeEach(func() {
				getPolicyResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.NotFound))
			})

			It("should not persist updates", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the policy version doesn't exist", func() {
			BeforeEach(func() {
				getPolicyVersionResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not persist updates", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("an error occurs retrieving the policy", func() {
			BeforeEach(func() {
				getPolicyError = errors.New("get policy error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not persist updates", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("an error occurs fetching the policy version", func() {
			BeforeEach(func() {
				getPolicyVersionError = errors.New("get policy version error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not persist updates", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("an error occurs updating the policy counter document", func() {
			BeforeEach(func() {
				updateError = errors.New("update error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not persist updates", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("an error occurs during the bulk update", func() {
			BeforeEach(func() {
				bulkError = errors.New("bulk error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy update fails", func() {
			BeforeEach(func() {
				bulkResponse.Items[0].Index.Error = &esutil.EsIndexDocError{
					Type:   fake.Word(),
					Reason: fake.Word(),
				}
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("creating the new policy version fails", func() {
			BeforeEach(func() {
				bulkResponse.Items[1].Create.Error = &esutil.EsIndexDocError{
					Type:   fake.Word(),
					Reason: fake.Word(),
				}
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("an update message isn't specified", func() {
			BeforeEach(func() {
				request.Policy.Policy.Message = ""
			})

			It("should add a message", func() {
				expectedMessage := "Updated policy"
				_, actualRequest := esClient.BulkArgsForCall(0)

				actualVersion := (actualRequest.Items[1].Message).(*pb.PolicyEntity)

				Expect(actualVersion.Message).To(Equal(expectedMessage))
				Expect(actualResponse.Policy.Message).To(Equal(expectedMessage))
			})
		})

		When("the policy content hasn't changed", func() {
			BeforeEach(func() {
				request.Policy.Policy.RegoContent = currentPolicy.Policy.RegoContent
				request.Policy.Policy.SourcePath = currentPolicy.Policy.SourcePath
			})

			It("should not update the counter document", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})

			It("should not create a new policy version", func() {
				Expect(esClient.BulkCallCount()).To(Equal(1))
				_, actualRequest := esClient.BulkArgsForCall(0)

				Expect(actualRequest.Items).To(HaveLen(1))
			})

			It("should not change the version", func() {
				Expect(actualResponse.CurrentVersion).To(Equal(currentVersion))
			})
		})

		When("the new Rego code is invalid", func() {
			BeforeEach(func() {
				request.Policy.Policy.RegoContent = uncompilablePolicy
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not persist updates", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("persisting policy updates", func() {
			var actualPolicyMessage *pb.Policy

			BeforeEach(func() {
				esClient.BulkCalls(func(ctx context.Context, request *esutil.BulkRequest) (*esutil.EsBulkResponse, error) {
					message, ok := request.Items[0].Message.(*pb.Policy)
					Expect(ok).To(BeTrue())
					actualPolicyMessage = deepCopyPolicy(message)

					return bulkResponse, bulkError
				})
			})

			It("should remove the policy content from the parent document", func() {
				Expect(actualPolicyMessage).NotTo(BeNil())
				Expect(actualPolicyMessage.Policy).To(BeNil())
			})
		})
	})

	Context("DeletePolicy", func() {
		var (
			policyId string
			request  *pb.DeletePolicyRequest

			getPolicyResponse *esutil.EsGetResponse
			getPolicyError    error

			updateError error
			actualError error
		)

		BeforeEach(func() {
			policyId = fake.UUID()
			policy := createRandomPolicy(policyId, fake.Uint32())
			policyJson, _ := protojson.Marshal(policy)
			getPolicyResponse = &esutil.EsGetResponse{
				Id:     policyId,
				Found:  true,
				Source: policyJson,
			}
			getPolicyError = nil

			updateError = nil
			request = &pb.DeletePolicyRequest{Id: policyId}
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getPolicyResponse, getPolicyError)
			esClient.UpdateReturns(nil, updateError)
			_, actualError = manager.DeletePolicy(ctx, request)
		})

		When("a policy is deleted", func() {
			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should fetch the policy", func() {
				Expect(esClient.GetCallCount()).To(Equal(1))
				_, actualRequest := esClient.GetArgsForCall(0)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.DocumentId).To(Equal(policyId))
			})

			It("should set the deleted flag on the policy", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(1))
				_, actualRequest := esClient.UpdateArgsForCall(0)

				Expect(actualRequest.Index).To(Equal(expectedPoliciesAlias))
				Expect(actualRequest.DocumentId).To(Equal(policyId))
				Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))
				actualPolicy := actualRequest.Message.(*pb.Policy)
				Expect(actualPolicy.Deleted).To(BeTrue())
			})
		})

		When("the policy id isn't specified", func() {
			BeforeEach(func() {
				request.Id = ""
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})
		})

		When("the policy isn't found", func() {
			BeforeEach(func() {
				getPolicyResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.NotFound))
			})
		})

		When("an error occurs retrieving the policy", func() {
			BeforeEach(func() {
				getPolicyError = errors.New("get error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("an error occurs setting the delete flag on the policy", func() {
			BeforeEach(func() {
				updateError = errors.New("update error")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("ValidatePolicy", func() {
		var (
			request *pb.ValidatePolicyRequest

			actualResponse *pb.ValidatePolicyResponse
			actualError    error
		)

		BeforeEach(func() {
			request = &pb.ValidatePolicyRequest{
				Policy: goodPolicy,
			}
		})

		JustBeforeEach(func() {
			actualResponse, actualError = manager.ValidatePolicy(ctx, request)
		})

		When("the policy is valid", func() {
			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should indicate successful compilation in the response", func() {
				Expect(actualResponse.Compile).To(BeTrue())
			})

			It("should not return any policy errors", func() {
				Expect(actualResponse.Errors).To(BeEmpty())
			})
		})

		When("the policy is empty", func() {
			BeforeEach(func() {
				request.Policy = ""
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})
		})

		When("the policy fails to compile", func() {
			BeforeEach(func() {
				request.Policy = uncompilablePolicy
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should indicate that compilation failed in the response", func() {
				Expect(actualResponse.Compile).To(BeFalse())
			})

			It("should return the compilation errors", func() {
				Expect(len(actualResponse.Errors)).To(BeNumerically(">", 0))
			})
		})

		When("the policy is missing required fields in the result", func() {
			BeforeEach(func() {
				request.Policy = compilablePolicyMissingResultsFields
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message about the missing field", func() {
				Expect(actualResponse.Errors).To(HaveLen(1))
			})
		})

		When("the policy does not contain a rule that returns results", func() {
			BeforeEach(func() {
				request.Policy = compilablePolicyMissingResultsReturn
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message about the missing result", func() {
				Expect(actualResponse.Errors).To(HaveLen(1))
			})
		})

		When("the policy does not have pass or violations rules", func() {
			BeforeEach(func() {
				request.Policy = compilablePolicyMissingRodeFields
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message about the missing rules", func() {
				Expect(actualResponse.Errors).To(HaveLen(3))
			})
		})

		When("the policy cannot be parsed", func() {
			BeforeEach(func() {
				request.Policy = unparseablePolicy
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should include an error message", func() {
				Expect(actualResponse.Errors).To(HaveLen(1))
			})
		})
	})
})

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

func deepCopyPolicy(policy *pb.Policy) *pb.Policy {
	policyJson, err := protojson.Marshal(policy)
	Expect(err).NotTo(HaveOccurred())

	var newPolicy pb.Policy
	Expect(protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(policyJson, &newPolicy)).NotTo(HaveOccurred())

	return &newPolicy
}
