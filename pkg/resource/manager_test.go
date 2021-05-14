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
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil/esutilfakes"
	"github.com/rode/rode/config"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
)

var _ = Describe("resource manager", func() {
	var (
		ctx      context.Context
		manager  Manager
		esClient *esutilfakes.FakeClient
		esConfig *config.ElasticsearchConfig
	)

	BeforeEach(func() {
		ctx = context.Background()
		esConfig = &config.ElasticsearchConfig{
			Refresh: config.RefreshTrue,
		}
		esClient = &esutilfakes.FakeClient{}
	})

	JustBeforeEach(func() {
		manager = NewManager(logger, esClient, esConfig)
	})

	Context("BatchCreateGenericResources", func() {
		var (
			actualError error

			expectedBatchCreateOccurrencesRequest *pb.BatchCreateOccurrencesRequest

			expectedMultiGetResponse *esutil.EsMultiGetResponse
			expectedMultiGetError    error

			expectedBulkCreateResponse *esutil.EsBulkResponse
			expectedBulkCreateError    error

			expectedOccurrence *grafeas_go_proto.Occurrence

			expectedResourceName string
		)

		BeforeEach(func() {
			expectedOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
			expectedResourceName = fake.URL()
			expectedOccurrence.Resource.Uri = fmt.Sprintf("%s@sha256:%s", expectedResourceName, fake.LetterN(10))

			expectedBatchCreateOccurrencesRequest = &pb.BatchCreateOccurrencesRequest{
				Occurrences: []*grafeas_go_proto.Occurrence{
					expectedOccurrence,
				},
			}

			// happy path: document needs to be created
			expectedMultiGetResponse = &esutil.EsMultiGetResponse{
				Docs: []*esutil.EsMultiGetDocument{
					{
						Found: false,
					},
				},
			}
			expectedMultiGetError = nil

			// happy path: generic resource document created successfully
			expectedBulkCreateResponse = &esutil.EsBulkResponse{
				Items: []*esutil.EsBulkResponseItem{
					{
						Create: &esutil.EsIndexDocResponse{
							Id:     expectedResourceName,
							Status: http.StatusOK,
						},
					},
				},
			}
			expectedBulkCreateError = nil
		})

		JustBeforeEach(func() {
			esClient.MultiGetReturns(expectedMultiGetResponse, expectedMultiGetError)
			esClient.BulkCreateReturns(expectedBulkCreateResponse, expectedBulkCreateError)

			actualError = manager.BatchCreateGenericResources(ctx, expectedBatchCreateOccurrencesRequest)
		})

		It("should check if the generic resources already exist", func() {
			Expect(esClient.MultiGetCallCount()).To(Equal(1))

			_, multiGetRequest := esClient.MultiGetArgsForCall(0)
			Expect(multiGetRequest.Index).To(Equal(rodeElasticsearchGenericResourcesIndex))
			Expect(multiGetRequest.DocumentIds).To(HaveLen(1))
			Expect(multiGetRequest.DocumentIds).To(ConsistOf(expectedResourceName))
		})

		It("should make a bulk request to create all of the generic resources", func() {
			Expect(esClient.BulkCreateCallCount()).To(Equal(1))

			_, bulkCreateRequest := esClient.BulkCreateArgsForCall(0)
			Expect(bulkCreateRequest.Refresh).To(Equal(esConfig.Refresh.String()))
			Expect(bulkCreateRequest.Index).To(Equal(rodeElasticsearchGenericResourcesIndex))
			Expect(bulkCreateRequest.Items).To(HaveLen(1))

			Expect(bulkCreateRequest.Items[0].DocumentId).To(Equal(expectedResourceName))
			genericResource := bulkCreateRequest.Items[0].Message.(*pb.GenericResource)

			Expect(genericResource.Name).To(Equal(expectedResourceName))
		})

		It("should not return an error", func() {
			Expect(actualError).ToNot(HaveOccurred())
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
				Expect(multiGetRequest.DocumentIds).To(ConsistOf(expectedResourceName))
			})

			It("should only create the generic resource once", func() {
				Expect(esClient.BulkCreateCallCount()).To(Equal(1))

				_, bulkCreateRequest := esClient.BulkCreateArgsForCall(0)
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
				Expect(esClient.BulkCreateCallCount()).To(Equal(0))
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
				Expect(esClient.BulkCreateCallCount()).To(Equal(0))
			})
		})

		When("the bulk create fails", func() {
			BeforeEach(func() {
				expectedBulkCreateError = errors.New("bulk create failed")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("one resource fails to create", func() {
			BeforeEach(func() {
				expectedBulkCreateResponse.Items[0].Create = &esutil.EsIndexDocResponse{
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
				expectedBulkCreateResponse.Items[0].Create = &esutil.EsIndexDocResponse{
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
