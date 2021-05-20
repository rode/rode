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

package resource

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
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
)

var _ = Describe("resource manager", func() {
	var (
		ctx          context.Context
		manager      Manager
		esClient     *esutilfakes.FakeClient
		esConfig     *config.ElasticsearchConfig
		indexManager *immocks.FakeIndexManager
		mockFilterer *filteringfakes.FakeFilterer

		genericResourcesAlias string
	)

	BeforeEach(func() {
		ctx = context.Background()
		esConfig = &config.ElasticsearchConfig{
			Refresh: config.RefreshTrue,
		}
		esClient = &esutilfakes.FakeClient{}
		indexManager = &immocks.FakeIndexManager{}
		mockFilterer = &filteringfakes.FakeFilterer{}

		genericResourcesAlias = fake.LetterN(10)
		indexManager.AliasNameReturns(genericResourcesAlias)
	})

	JustBeforeEach(func() {
		manager = NewManager(logger, esClient, esConfig, indexManager, mockFilterer)
	})

	Context("BatchCreateGenericResources", func() {
		var (
			actualError error

			expectedBatchCreateOccurrencesRequest *pb.BatchCreateOccurrencesRequest

			expectedMultiGetResponse *esutil.EsMultiGetResponse
			expectedMultiGetError    error

			expectedBulkResponse *esutil.EsBulkResponse
			expectedBulkError    error

			expectedOccurrence *grafeas_go_proto.Occurrence

			expectedResourceName string
			expectedResourceId   string
		)

		BeforeEach(func() {
			expectedOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
			expectedResourceName = fake.URL()
			expectedResourceId = fmt.Sprintf("DOCKER:%s", expectedResourceName)
			expectedOccurrence.Resource.Uri = fmt.Sprintf("%s@sha256:%s", expectedResourceName, fake.LetterN(10))

			expectedBatchCreateOccurrencesRequest = &pb.BatchCreateOccurrencesRequest{
				Occurrences: []*grafeas_go_proto.Occurrence{
					expectedOccurrence,
				},
			}

			// happy path: document needs to be created
			expectedMultiGetResponse = &esutil.EsMultiGetResponse{
				Docs: []*esutil.EsGetResponse{
					{
						Found: false,
					},
				},
			}
			expectedMultiGetError = nil

			// happy path: generic resource document created successfully
			expectedBulkResponse = &esutil.EsBulkResponse{
				Items: []*esutil.EsBulkResponseItem{
					{
						Create: &esutil.EsIndexDocResponse{
							Id:     expectedResourceName,
							Status: http.StatusOK,
						},
					},
				},
			}
			expectedBulkError = nil
		})

		JustBeforeEach(func() {
			esClient.MultiGetReturns(expectedMultiGetResponse, expectedMultiGetError)
			esClient.BulkReturns(expectedBulkResponse, expectedBulkError)

			actualError = manager.BatchCreateGenericResources(ctx, expectedBatchCreateOccurrencesRequest)
		})

		It("should check if the generic resources already exist", func() {
			Expect(esClient.MultiGetCallCount()).To(Equal(1))

			_, multiGetRequest := esClient.MultiGetArgsForCall(0)
			Expect(multiGetRequest.Index).To(Equal(genericResourcesAlias))
			Expect(multiGetRequest.DocumentIds).To(HaveLen(1))
			Expect(multiGetRequest.DocumentIds).To(ConsistOf(expectedResourceId))
		})

		It("should make a bulk request to create all of the generic resources", func() {
			Expect(esClient.BulkCallCount()).To(Equal(1))

			_, bulkCreateRequest := esClient.BulkArgsForCall(0)
			Expect(bulkCreateRequest.Refresh).To(Equal(esConfig.Refresh.String()))
			Expect(bulkCreateRequest.Index).To(Equal(genericResourcesAlias))
			Expect(bulkCreateRequest.Items).To(HaveLen(1))

			Expect(bulkCreateRequest.Items[0].DocumentId).To(Equal(expectedResourceId))
			genericResource := bulkCreateRequest.Items[0].Message.(*pb.GenericResource)

			Expect(genericResource.Name).To(Equal(expectedResourceName))
			Expect(genericResource.Type).To(Equal(pb.ResourceType_DOCKER))
		})

		It("should not return an error", func() {
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("a non docker resource is referenced", func() {
			BeforeEach(func() {
				expectedOccurrence.Resource.Uri = fmt.Sprintf("git://github.com/rode/rode@%s", fake.LetterN(10))
			})

			It("should create a generic resource with the correct type", func() {
				Expect(esClient.BulkCallCount()).To(Equal(1))

				_, bulkCreateRequest := esClient.BulkArgsForCall(0)
				genericResource := bulkCreateRequest.Items[0].Message.(*pb.GenericResource)

				Expect(genericResource.Type).To(Equal(pb.ResourceType_GIT))
			})
		})

		When("the same resource appears multiple times", func() {
			BeforeEach(func() {
				otherOccurrence := createRandomOccurrence(grafeas_common_proto.NoteKind_BUILD)
				otherOccurrence.Resource.Uri = expectedOccurrence.Resource.Uri

				expectedBatchCreateOccurrencesRequest.Occurrences = append(expectedBatchCreateOccurrencesRequest.Occurrences, otherOccurrence)
			})

			It("should only search for the existing resource once", func() {
				Expect(esClient.MultiGetCallCount()).To(Equal(1))

				_, multiGetRequest := esClient.MultiGetArgsForCall(0)
				Expect(multiGetRequest.DocumentIds).To(HaveLen(1))
				Expect(multiGetRequest.DocumentIds).To(ConsistOf(expectedResourceId))
			})

			It("should only create the generic resource once", func() {
				Expect(esClient.BulkCallCount()).To(Equal(1))

				_, bulkCreateRequest := esClient.BulkArgsForCall(0)
				Expect(bulkCreateRequest.Items).To(HaveLen(1))
			})
		})

		When("an error occurs determining the resource uri version", func() {
			BeforeEach(func() {
				expectedOccurrence.Resource.Uri = fake.URL()
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the generic resources already exist", func() {
			BeforeEach(func() {
				expectedMultiGetResponse.Docs[0].Found = true
			})

			It("should not attempt to create any resources", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the multi get request fails", func() {
			BeforeEach(func() {
				expectedMultiGetError = errors.New("multi get failed")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("should not attempt to create any resources", func() {
				Expect(esClient.BulkCallCount()).To(Equal(0))
			})
		})

		When("the bulk create fails", func() {
			BeforeEach(func() {
				expectedBulkError = errors.New("bulk create failed")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("one resource fails to create", func() {
			BeforeEach(func() {
				expectedBulkResponse.Items[0].Create = &esutil.EsIndexDocResponse{
					Error: &esutil.EsIndexDocError{
						Reason: fake.Word(),
					},
					Status: http.StatusInternalServerError,
				}
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("attempting to create a generic resource that already exists", func() {
			BeforeEach(func() {
				expectedBulkResponse.Items[0].Create = &esutil.EsIndexDocResponse{
					Error: &esutil.EsIndexDocError{
						Reason: fake.Word(),
					},
					Status: http.StatusConflict,
				}
			})
		})

		It("should not return an error", func() {
			Expect(actualError).ToNot(HaveOccurred())
		})
	})

	Context("ListGenericResources", func() {
		var (
			expectedListGenericResourcesRequest *pb.ListGenericResourcesRequest

			expectedSearchResponse *esutil.SearchResponse
			expectedSearchError    error

			expectedGenericResource *pb.GenericResource

			expectedFilterQuery *filtering.Query
			expectedFilterError error

			actualListGenericResourcesResponse *pb.ListGenericResourcesResponse
			actualError                        error
		)

		BeforeEach(func() {
			expectedListGenericResourcesRequest = &pb.ListGenericResourcesRequest{}

			expectedGenericResource = &pb.GenericResource{
				Name: fake.LetterN(10),
				Type: pb.ResourceType(fake.Number(0, 6)),
			}
			genericResourceJson, _ := protojson.Marshal(expectedGenericResource)
			expectedSearchResponse = &esutil.SearchResponse{
				Hits: &esutil.EsSearchResponseHits{
					Hits: []*esutil.EsSearchResponseHit{
						{
							Source: genericResourceJson,
						},
					},
				},
			}
			expectedSearchError = nil
		})

		JustBeforeEach(func() {
			mockFilterer.ParseExpressionReturns(expectedFilterQuery, expectedFilterError)
			esClient.SearchReturns(expectedSearchResponse, expectedSearchError)

			actualListGenericResourcesResponse, actualError = manager.ListGenericResources(ctx, expectedListGenericResourcesRequest)
		})

		It("should perform a search", func() {
			Expect(esClient.SearchCallCount()).To(Equal(1))

			_, searchRequest := esClient.SearchArgsForCall(0)

			// no pagination options were specified
			Expect(searchRequest.Pagination).To(BeNil())

			// no filter was specified, so we should only have one query
			Expect(*searchRequest.Search.Query.Bool.Must).To(HaveLen(1))

			// the only query should specify the join field
			query := (*searchRequest.Search.Query.Bool.Must)[0].(*filtering.Query)
			Expect((*query.Term)[genericResourceDocumentJoinField]).To(Equal(genericResourceRelationName))
		})

		It("should not attempt to parse a filter", func() {
			Expect(mockFilterer.ParseExpressionCallCount()).To(Equal(0))
		})

		It("should return the generic resources and no error", func() {
			Expect(actualListGenericResourcesResponse.GenericResources).To(HaveLen(1))
			Expect(actualListGenericResourcesResponse.GenericResources[0]).To(Equal(expectedGenericResource))

			Expect(actualError).ToNot(HaveOccurred())
		})

		When("a filter is specified", func() {
			BeforeEach(func() {
				expectedListGenericResourcesRequest.Filter = fake.LetterN(10)

				expectedFilterQuery = &filtering.Query{
					Term: &filtering.Term{
						fake.LetterN(10): fake.LetterN(10),
					},
				}
				expectedFilterError = nil
			})

			It("should attempt to parse the filter", func() {
				Expect(mockFilterer.ParseExpressionCallCount()).To(Equal(1))

				filter := mockFilterer.ParseExpressionArgsForCall(0)
				Expect(filter).To(Equal(expectedListGenericResourcesRequest.Filter))
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
					Expect(actualListGenericResourcesResponse).To(BeNil())
					Expect(actualError).To(HaveOccurred())
				})
			})
		})

		When("pagination is used", func() {
			BeforeEach(func() {
				expectedListGenericResourcesRequest.PageSize = int32(fake.Number(1, 10))
				expectedListGenericResourcesRequest.PageToken = fake.LetterN(10)
			})

			It("should use pagination when searching", func() {
				Expect(esClient.SearchCallCount()).To(Equal(1))

				_, searchRequest := esClient.SearchArgsForCall(0)

				Expect(searchRequest.Pagination).ToNot(BeNil())
				Expect(searchRequest.Pagination.Size).To(BeEquivalentTo(expectedListGenericResourcesRequest.PageSize))
				Expect(searchRequest.Pagination.Token).To(Equal(expectedListGenericResourcesRequest.PageToken))
			})
		})
	})
})

func createRandomOccurrence(kind grafeas_common_proto.NoteKind) *grafeas_go_proto.Occurrence {
	return &grafeas_go_proto.Occurrence{
		Name: fake.LetterN(10),
		Resource: &grafeas_go_proto.Resource{
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
