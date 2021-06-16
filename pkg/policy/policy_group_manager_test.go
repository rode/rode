package policy

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	immocks "github.com/rode/es-index-manager/mocks"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/pkg/constants"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil/esutilfakes"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering/filteringfakes"
	"github.com/rode/rode/config"
	pb "github.com/rode/rode/proto/v1alpha1"
)

var _ = Describe("PolicyGroupManager", func() {

	var (
		manager                   PolicyGroupManager
		ctx                       = context.Background()
		expectedPolicyGroupsAlias string

		esClient     *esutilfakes.FakeClient
		esConfig     *config.ElasticsearchConfig
		indexManager *immocks.FakeIndexManager
		filterer     *filteringfakes.FakeFilterer
	)

	BeforeEach(func() {
		esClient = &esutilfakes.FakeClient{}
		esConfig = &config.ElasticsearchConfig{
			Refresh: config.RefreshOption(fake.RandomString([]string{config.RefreshTrue, config.RefreshFalse, config.RefreshWaitFor})),
		}
		indexManager = &immocks.FakeIndexManager{}
		filterer = &filteringfakes.FakeFilterer{}

		expectedPolicyGroupsAlias = fake.LetterN(10)
		indexManager.AliasNameReturns(expectedPolicyGroupsAlias)

		manager = NewPolicyGroupManager(logger, esClient, esConfig, indexManager, filterer)
	})

	Context("CreatePolicyGroup", func() {
		var (
			policyGroupName string

			createPolicyRequest *pb.PolicyGroup
			actualPolicyGroup   *pb.PolicyGroup

			getPolicyGroupResponse *esutil.EsGetResponse
			getPolicyGroupError    error

			createPolicyGroupError error

			actualError error
		)

		BeforeEach(func() {
			policyGroupName = fake.Word()
			createPolicyRequest = randomPolicyGroup(policyGroupName)

			getPolicyGroupResponse = &esutil.EsGetResponse{
				Id:    policyGroupName,
				Found: false,
			}
			getPolicyGroupError = nil
			createPolicyGroupError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getPolicyGroupResponse, getPolicyGroupError)
			esClient.CreateReturns("", createPolicyGroupError)

			actualPolicyGroup, actualError = manager.CreatePolicyGroup(ctx, deepCopyPolicyGroup(createPolicyRequest))
		})

		It("should check to see if the policy name is in use", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, actualRequest := esClient.GetArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyGroupsAlias))
			Expect(actualRequest.DocumentId).To(Equal(policyGroupName))
		})

		It("should create the policy group document", func() {
			Expect(esClient.CreateCallCount()).To(Equal(1))

			_, actualRequest := esClient.CreateArgsForCall(0)

			Expect(actualRequest.DocumentId).To(Equal(policyGroupName))
			Expect(actualRequest.Index).To(Equal(expectedPolicyGroupsAlias))
			Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))

			actualMessage := actualRequest.Message.(*pb.PolicyGroup)
			Expect(actualMessage.Name).To(Equal(policyGroupName))
			Expect(actualMessage.Description).To(Equal(createPolicyRequest.Description))
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		It("should return the new policy", func() {
			Expect(actualPolicyGroup.Name).To(Equal(policyGroupName))
			Expect(actualPolicyGroup.Description).To(Equal(createPolicyRequest.Description))
			Expect(actualPolicyGroup.Created.IsValid()).To(BeTrue())
			Expect(actualPolicyGroup.Updated.IsValid()).To(BeTrue())
		})

		When("the name is invalid", func() {
			BeforeEach(func() {
				createPolicyRequest.Name = fake.URL()
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not insert the policy group", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("a policy group with that name already exists", func() {
			BeforeEach(func() {
				getPolicyGroupResponse.Found = true
				policyGroup := randomPolicyGroup(policyGroupName)
				policyGroupJson, _ := protojson.Marshal(policyGroup)
				getPolicyGroupResponse.Source = policyGroupJson
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.AlreadyExists))
			})

			It("should not insert the policy group", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs while checking if there's an existing policy group by that name", func() {
			BeforeEach(func() {
				getPolicyGroupError = errors.New("get policy group error")
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not insert the policy group", func() {
				Expect(esClient.CreateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs creating the document in Elasticsearch", func() {
			BeforeEach(func() {
				createPolicyGroupError = errors.New("create error")
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("ListPolicyGroups", func() {
		var (
			request        *pb.ListPolicyGroupsRequest
			actualResponse *pb.ListPolicyGroupsResponse
			actualError    error

			policyGroupCount     int
			expectedPolicyGroups []*pb.PolicyGroup

			expectedFilterQuery *filtering.Query
			expectedFilterError error

			searchResponse *esutil.SearchResponse
			searchError    error
		)

		BeforeEach(func() {
			request = &pb.ListPolicyGroupsRequest{}
			policyGroupCount = fake.Number(2, 5)
			expectedPolicyGroups = []*pb.PolicyGroup{}
			searchResponse = &esutil.SearchResponse{
				Hits: &esutil.EsSearchResponseHits{},
			}
			searchError = nil

			for i := 0; i < policyGroupCount; i++ {
				policyGroupName := fake.Word()
				policyGroup := randomPolicyGroup(policyGroupName)
				expectedPolicyGroups = append(expectedPolicyGroups, policyGroup)

				policyGroupJson, _ := protojson.Marshal(policyGroup)
				searchResponse.Hits.Hits = append(searchResponse.Hits.Hits, &esutil.EsSearchResponseHit{
					ID:     policyGroupName,
					Source: policyGroupJson,
				})
			}

			expectedFilterQuery = nil
			expectedFilterError = nil
		})

		JustBeforeEach(func() {
			filterer.ParseExpressionReturns(expectedFilterQuery, expectedFilterError)
			esClient.SearchReturns(searchResponse, searchError)

			actualResponse, actualError = manager.ListPolicyGroups(ctx, request)
		})

		It("should issue a search for all policy groups", func() {
			Expect(filterer.ParseExpressionCallCount()).To(Equal(0))
			Expect(esClient.SearchCallCount()).To(Equal(1))

			_, actualRequest := esClient.SearchArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyGroupsAlias))
			Expect(actualRequest.Pagination).To(BeNil())
			Expect(actualRequest.Search.Query).To(BeNil())
			Expect(actualRequest.Search.Sort["created"]).To(Equal(esutil.EsSortOrderDescending))
		})

		It("should return the policy groups", func() {
			Expect(actualResponse).NotTo(BeNil())
			Expect(actualResponse.PolicyGroups).To(ConsistOf(expectedPolicyGroups))
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("a filter is applied", func() {
			var expectedFilter string

			BeforeEach(func() {
				expectedFilter = fake.Word()
				request.Filter = expectedFilter

				expectedFilterQuery = &filtering.Query{
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

			It("should include the query in the search", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))

				_, actualRequest := esClient.SearchArgsForCall(0)

				Expect(actualRequest.Search.Query).To(Equal(expectedFilterQuery))
			})

			When("an error occurs parsing the filter expression", func() {
				BeforeEach(func() {
					expectedFilterError = errors.New("parse error")
				})

				It("should return an error", func() {
					Expect(actualResponse).To(BeNil())
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				})

				It("should not query Elasticsearch", func() {
					Expect(esClient.SearchCallCount()).To(Equal(0))
				})
			})
		})

		When("pagination options are specified", func() {
			var (
				pageToken     string
				pageSize      int32
				nextPageToken string
			)

			BeforeEach(func() {
				nextPageToken = fake.Word()
				pageSize = int32(fake.Number(10, 100))
				pageToken = fake.Word()

				request.PageSize = pageSize
				request.PageToken = pageToken

				searchResponse.NextPageToken = nextPageToken
			})

			It("should include the page size and token in the search request", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))

				_, actualRequest := esClient.SearchArgsForCall(0)

				Expect(actualRequest.Pagination.Token).To(Equal(pageToken))
				Expect(actualRequest.Pagination.Size).To(BeEquivalentTo(pageSize))
			})

			It("should return the next page token", func() {
				Expect(actualResponse.NextPageToken).To(Equal(nextPageToken))
			})
		})

		When("there are no policy groups", func() {
			BeforeEach(func() {
				searchResponse.Hits.Hits = nil
			})

			It("should return an empty list", func() {
				Expect(actualResponse.PolicyGroups).To(BeEmpty())
			})
		})

		When("an error occurs searching for policy groups", func() {
			BeforeEach(func() {
				searchError = errors.New("search error")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("a policy group document is malformed", func() {
			BeforeEach(func() {
				randomIndex := fake.Number(0, policyGroupCount-1)

				searchResponse.Hits.Hits[randomIndex].Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("GetPolicyGroup", func() {
		var (
			policyGroupName     string
			expectedPolicyGroup *pb.PolicyGroup

			getPolicyGroupResponse *esutil.EsGetResponse
			getPolicyGroupError    error

			actualPolicyGroup *pb.PolicyGroup
			actualError       error
		)

		BeforeEach(func() {
			policyGroupName = fake.Word()
			expectedPolicyGroup = randomPolicyGroup(policyGroupName)
			policyGroupJson, _ := protojson.Marshal(expectedPolicyGroup)

			getPolicyGroupResponse = &esutil.EsGetResponse{
				Id:     policyGroupName,
				Found:  true,
				Source: policyGroupJson,
			}
			getPolicyGroupError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getPolicyGroupResponse, getPolicyGroupError)

			actualPolicyGroup, actualError = manager.GetPolicyGroup(ctx, &pb.GetPolicyGroupRequest{Name: policyGroupName})
		})

		It("should get the alias name from the index manager", func() {
			Expect(indexManager.AliasNameCallCount()).To(Equal(1))
			documentKind, inner := indexManager.AliasNameArgsForCall(0)

			Expect(documentKind).To(Equal(constants.PolicyGroupsDocumentKind))
			Expect(inner).To(BeEmpty())
		})

		It("should fetch the policy group from Elasticsearch", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, actualRequest := esClient.GetArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyGroupsAlias))
			Expect(actualRequest.DocumentId).To(Equal(policyGroupName))
		})

		It("should return the policy group", func() {
			Expect(actualPolicyGroup).NotTo(BeNil())
			Expect(actualPolicyGroup.Name).To(Equal(policyGroupName))
			Expect(actualPolicyGroup.Description).To(Equal(expectedPolicyGroup.Description))
			Expect(actualPolicyGroup.Created.IsValid()).To(BeTrue())
			Expect(actualPolicyGroup.Updated.IsValid()).To(BeTrue())
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("the name is empty", func() {
			BeforeEach(func() {
				policyGroupName = ""
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})

			It("should not try to fetch the policy group document", func() {
				Expect(esClient.GetCallCount()).To(Equal(0))
			})
		})

		When("an error occurs fetching the policy group document", func() {
			BeforeEach(func() {
				getPolicyGroupError = errors.New("get policy group error")
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy group document cannot be deserialized", func() {
			BeforeEach(func() {
				getPolicyGroupResponse.Source = invalidJson
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the policy group document is not found", func() {
			BeforeEach(func() {
				getPolicyGroupResponse.Found = false
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.NotFound))
			})
		})
	})

	Context("UpdatePolicyGroup", func() {
		var (
			policyGroupName     string
			existingPolicyGroup *pb.PolicyGroup
			updatedPolicyGroup  *pb.PolicyGroup

			actualPolicyGroup *pb.PolicyGroup
			actualError       error

			getPolicyGroupResponse *esutil.EsGetResponse
			getPolicyGroupError    error

			updatePolicyGroupError error
		)

		BeforeEach(func() {
			policyGroupName = fake.Word()
			existingPolicyGroup = randomPolicyGroup(policyGroupName)
			updatedPolicyGroup = deepCopyPolicyGroup(existingPolicyGroup)
			updatedPolicyGroup.Description = fake.Sentence(5)

			policyGroupJson, _ := protojson.Marshal(existingPolicyGroup)
			getPolicyGroupResponse = &esutil.EsGetResponse{
				Id:     policyGroupName,
				Found:  true,
				Source: policyGroupJson,
			}
			getPolicyGroupError = nil
			updatePolicyGroupError = nil
		})

		JustBeforeEach(func() {
			esClient.GetReturns(getPolicyGroupResponse, getPolicyGroupError)
			esClient.UpdateReturns(nil, updatePolicyGroupError)

			actualPolicyGroup, actualError = manager.UpdatePolicyGroup(ctx, deepCopyPolicyGroup(updatedPolicyGroup))
		})

		It("should fetch the current policy group", func() {
			Expect(esClient.GetCallCount()).To(Equal(1))

			_, actualRequest := esClient.GetArgsForCall(0)

			Expect(actualRequest.Index).To(Equal(expectedPolicyGroupsAlias))
			Expect(actualRequest.DocumentId).To(Equal(policyGroupName))
		})

		It("should update Elasticsearch with the new description", func() {
			Expect(esClient.UpdateCallCount()).To(Equal(1))

			_, actualRequest := esClient.UpdateArgsForCall(0)

			Expect(actualRequest.DocumentId).To(Equal(policyGroupName))
			Expect(actualRequest.Index).To(Equal(expectedPolicyGroupsAlias))
			Expect(actualRequest.Refresh).To(Equal(esConfig.Refresh.String()))

			actualMessage := actualRequest.Message.(*pb.PolicyGroup)
			Expect(actualMessage.Name).To(Equal(policyGroupName))
			Expect(actualMessage.Description).To(Equal(updatedPolicyGroup.Description))
			Expect(actualMessage.Updated.IsValid()).To(BeTrue())
		})

		It("should return the updated policy group", func() {
			Expect(actualPolicyGroup).NotTo(BeNil())
			Expect(actualPolicyGroup.Name).To(Equal(policyGroupName))
			Expect(actualPolicyGroup.Description).To(Equal(updatedPolicyGroup.Description))
			Expect(actualPolicyGroup.Updated.IsValid()).To(BeTrue())
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("an error occurs fetching the policy group", func() {
			BeforeEach(func() {
				getPolicyGroupError = errors.New("get error")
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})

			It("should not try to update the policy group", func() {
				Expect(esClient.UpdateCallCount()).To(Equal(0))
			})
		})

		When("an error occurs updating the policy group", func() {
			BeforeEach(func() {
				updatePolicyGroupError = errors.New("update error")
			})

			It("should return an error", func() {
				Expect(actualPolicyGroup).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})
})

func randomPolicyGroup(name string) *pb.PolicyGroup {
	return &pb.PolicyGroup{
		Name:        name,
		Description: fake.Sentence(5),
		Created:     timestamppb.New(fake.Date()),
		Updated:     timestamppb.New(fake.Date()),
	}
}

func deepCopyPolicyGroup(group *pb.PolicyGroup) *pb.PolicyGroup {
	return &pb.PolicyGroup{
		Name:        group.Name,
		Description: group.Description,
	}
}
