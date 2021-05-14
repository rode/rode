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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rode/rode/pkg/resource/resourcefakes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/config"
	"github.com/rode/rode/mocks"
	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/build_go_proto"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/provenance_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("rode server", func() {
	const (
		createProjectError = "CREATE_PROJECT_ERROR"
		getProjectError    = "GET_PROJECT_ERROR"
		goodPolicy         = `
		package harborfail
		pass = true {
				count(violation_count) == 0
		}
		violation_count[v] {
				violations[v].pass == false
		}
		#######################################################################################
		note_name_dne[o]{
			m := input.occurrences
			#trace(sprintf("this %v",[count({x | m[x]; f(m[x]) })]))
			o = count({x | m[x]; f(m[x]) })
		}
		f(x) = true { x.note_name == "not this" }
		# No occurrence should be missing a note name
		violations[result] {
			result = {
				"pass": note_name_dne[i] == 0,
				"id": "note_names_exist",
				"name": "Occurrences containing note names",
				"description": "Verify that all occurrences contain a note name",
				"message": sprintf("found %v occurrences with missing note names", [note_name_dne[i]]),
			}
		}
		###################################################################################
		uses_gcr[o]{
			m := input.occurrences
			#trace(sprintf("this %v",[count({x | m[x]; f(m[x]) })]))
			o = count({x | m[x]; g(m[x]) })
		}
		g(x) = true { contains(x.resource.uri, "gcr.io") }
		# All occurrences should have a resource uri containing a gcr endpoint
		violations[result] {
			result = {
				"pass": uses_gcr[i] == 0,
				"id": "uses_gcr",
				"name": "Occurrences use GCR URIs",
				"description": "Verify that all occurrences contain a resource uri from gcr",
				"message": sprintf("found %v occurrences with non gcr resource uris", [uses_gcr[i]]),
			}
		}`
		compilablePolicyMissingRodeFields = `
		package play
		default hello = false
		hello {
			m := input.message
			m == "world"
		}`
		compilablePolicyMissingResultsFields = `
		package harborfail
		pass = true {
				3 == 3
		}

		violations[result] {
			result := {
				"pass": true,
				"name": "Occurrences containing note names",
				"description": "Verify that all occurrences contain a note name",
				"message": sprintf("found %v occurrences with missing note names", ["hi"]),
			}
		}`
		compilablePolicyMissingResultsReturn = `
		package harborfail
		pass = true {
				3 == 3
		}

		violations {
			a := {
				"pass": true,
				"name": "Occurrences containing note names",
				"description": "Verify that all occurrences contain a note name",
				"message": sprintf("found %v occurrences with missing note names", ["hi"]),
			}
		}`
		unparseablePolicy = `
		package play
		default hello = false
		hello 
			m := input.message
			m == "world"
		}`
		uncompilablePolicy = `
		package play
		default hello = false
		hello {
			m := input.message
			m2 == "world"
		}`
	)

	var (
		server                pb.RodeServer
		grafeasClient         *mocks.FakeGrafeasV1Beta1Client
		grafeasProjectsClient *mocks.FakeProjectsClient
		opaClient             *mocks.MockOpaClient
		esClient              *elasticsearch.Client
		esTransport           *mockEsTransport
		mockFilterer          *mocks.MockFilterer
		mockCtrl              *gomock.Controller
		elasticsearchConfig   *config.ElasticsearchConfig
		resourceManager       *resourcefakes.FakeManager
		ctx                   context.Context
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		grafeasClient = &mocks.FakeGrafeasV1Beta1Client{}
		grafeasProjectsClient = &mocks.FakeProjectsClient{}
		resourceManager = &resourcefakes.FakeManager{}
		opaClient = mocks.NewMockOpaClient(mockCtrl)
		elasticsearchConfig = &config.ElasticsearchConfig{
			Refresh: "true",
		}

		esTransport = &mockEsTransport{}
		esClient = &elasticsearch.Client{
			Transport: esTransport,
			API:       esapi.New(esTransport),
		}
		mockFilterer = mocks.NewMockFilterer(mockCtrl)
		esTransport.preparedHttpResponses = []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       structToJsonBody(createEsIndexResponse("rode-v1alpha1-policies")),
			},
			{
				StatusCode: http.StatusOK,
				Body:       structToJsonBody(createEsIndexResponse("rode-v1alpha1-generic-resources")),
			},
		}

		ctx = context.Background()

		// not using the constructor as it has side effects. side effects are tested under the "initialize" context
		server = &rodeServer{
			logger:              logger,
			grafeasCommon:       grafeasClient,
			grafeasProjects:     grafeasProjectsClient,
			opa:                 opaClient,
			esClient:            esClient,
			filterer:            mockFilterer,
			elasticsearchConfig: elasticsearchConfig,
			resouceManager:      resourceManager,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("initialize", func() {
		var (
			actualRodeServer pb.RodeServer
			actualError      error

			expectedProject         *grafeas_project_proto.Project
			expectedGetProjectError error

			expectedCreateProjectError error
		)

		BeforeEach(func() {
			expectedProject = &grafeas_project_proto.Project{
				Name: fmt.Sprintf("projects/%s", gofakeit.LetterN(10)),
			}
			expectedGetProjectError = nil
			expectedCreateProjectError = nil
		})

		JustBeforeEach(func() {
			grafeasProjectsClient.GetProjectReturns(expectedProject, expectedGetProjectError)
			grafeasProjectsClient.CreateProjectReturns(expectedProject, expectedCreateProjectError)

			actualRodeServer, actualError = NewRodeServer(logger, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer, elasticsearchConfig, resourceManager)
		})

		It("should check if the rode project exists", func() {
			Expect(grafeasProjectsClient.GetProjectCallCount()).To(Equal(1))

			_, getProjectRequest, _ := grafeasProjectsClient.GetProjectArgsForCall(0)
			Expect(getProjectRequest.Name).To(Equal(rodeProjectSlug))
		})

		// happy path: project already exists
		It("should not create a project", func() {
			Expect(grafeasProjectsClient.CreateProjectCallCount()).To(Equal(0))
		})

		It("should create an index for policies", func() {
			Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodPut))
			Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal("/rode-v1alpha1-policies"))
			payload := map[string]interface{}{}
			readResponseBody(esTransport.receivedHttpRequests[0], &payload)
			Expect(payload).To(MatchAllKeys(Keys{
				"mappings": MatchAllKeys(Keys{
					"_meta": MatchAllKeys(Keys{
						"type": Equal("rode"),
					}),
					"properties": MatchAllKeys(Keys{
						"created": MatchAllKeys(Keys{
							"type": Equal("date"),
						}),
					}),
					"dynamic_templates": ConsistOf(MatchAllKeys(Keys{
						"strings_as_keywords": MatchAllKeys(Keys{
							"match_mapping_type": Equal("string"),
							"mapping": MatchAllKeys(Keys{
								"norms": Equal(false),
								"type":  Equal("keyword"),
							}),
						}),
					})),
				}),
			}))
		})

		It("should create an index for generic resources", func() {
			Expect(esTransport.receivedHttpRequests[1].Method).To(Equal(http.MethodPut))
			Expect(esTransport.receivedHttpRequests[1].URL.Path).To(Equal("/rode-v1alpha1-generic-resources"))
			payload := map[string]interface{}{}
			readResponseBody(esTransport.receivedHttpRequests[1], &payload)
			Expect(payload).To(MatchAllKeys(Keys{
				"mappings": MatchAllKeys(Keys{
					"_meta": MatchAllKeys(Keys{
						"type": Equal("rode"),
					}),
					"properties": MatchAllKeys(Keys{
						"name": MatchAllKeys(Keys{
							"type": Equal("keyword"),
						}),
					}),
					"dynamic_templates": ConsistOf(MatchAllKeys(Keys{
						"strings_as_keywords": MatchAllKeys(Keys{
							"match_mapping_type": Equal("string"),
							"mapping": MatchAllKeys(Keys{
								"norms": Equal(false),
								"type":  Equal("keyword"),
							}),
						}),
					})),
				}),
			}))
		})

		It("should return the initialized rode server", func() {
			Expect(actualRodeServer).ToNot(BeNil())
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("getting the rode project fails", func() {
			BeforeEach(func() {
				expectedGetProjectError = status.Error(codes.Internal, "getting project failed")
			})

			It("should return an error", func() {
				Expect(actualRodeServer).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not create a project", func() {
				Expect(grafeasProjectsClient.CreateProjectCallCount()).To(Equal(0))
			})

			It("should not attempt to create indices", func() {
				Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
			})
		})

		When("the rode project does not exist", func() {
			BeforeEach(func() {
				expectedGetProjectError = status.Error(codes.NotFound, "not found")
			})

			It("should create the rode project", func() {
				Expect(grafeasProjectsClient.CreateProjectCallCount()).To(Equal(1))

				_, createProjectRequest, _ := grafeasProjectsClient.CreateProjectArgsForCall(0)
				Expect(createProjectRequest.Project.Name).To(Equal(rodeProjectSlug))
			})

			When("creating the rode project fails", func() {
				BeforeEach(func() {
					expectedCreateProjectError = errors.New("create project failed")
				})

				It("should return an error", func() {
					Expect(actualRodeServer).To(BeNil())
					Expect(actualError).To(HaveOccurred())
				})

				It("should not attempt to create indices", func() {
					Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
				})
			})
		})

		When("creating the first index fails", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses[0].StatusCode = http.StatusInternalServerError
			})

			It("should return an error", func() {
				Expect(actualRodeServer).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not attempt to create another index", func() {
				Expect(esTransport.receivedHttpRequests).To(HaveLen(1))
			})
		})

		When("creating the first index errors", func() {
			BeforeEach(func() {
				esTransport.actions = []func(req *http.Request) (*http.Response, error){
					func(req *http.Request) (*http.Response, error) {
						return nil, errors.New(gofakeit.Word())
					},
				}
			})

			It("should return an error", func() {
				Expect(actualRodeServer).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not attempt to create another index", func() {
				Expect(esTransport.receivedHttpRequests).To(HaveLen(1))
			})
		})

		When("creating the second index fails", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses[1].StatusCode = http.StatusInternalServerError
			})

			It("should return an error", func() {
				Expect(actualRodeServer).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the first index already exists", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses[0].StatusCode = http.StatusBadRequest
			})

			It("should return the initialized rode server", func() {
				Expect(actualRodeServer).ToNot(BeNil())
				Expect(actualError).ToNot(HaveOccurred())
			})
		})

		When("the second index already exists", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses[1].StatusCode = http.StatusBadRequest
			})

			It("should return the initialized rode server", func() {
				Expect(actualRodeServer).ToNot(BeNil())
				Expect(actualError).ToNot(HaveOccurred())
			})
		})
	})

	Context("BatchCreateOccurrences", func() {
		var (
			actualRodeBatchCreateOccurrencesResponse *pb.BatchCreateOccurrencesResponse
			actualError                              error

			expectedRodeBatchCreateOccurrencesRequest *pb.BatchCreateOccurrencesRequest

			expectedOccurrence *grafeas_proto.Occurrence

			expectedGrafeasBatchCreateOccurrencesResponse *grafeas_proto.BatchCreateOccurrencesResponse
			expectedGrafeasBatchCreateOccurrencesError    error

			expectedBatchCreateResourcesError error

			expectedResourceName string
		)

		BeforeEach(func() {
			expectedOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
			expectedResourceName = gofakeit.URL()
			expectedOccurrence.Resource.Uri = fmt.Sprintf("%s@sha256:%s", expectedResourceName, gofakeit.LetterN(10))

			expectedGrafeasBatchCreateOccurrencesResponse = &grafeas_proto.BatchCreateOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					expectedOccurrence,
				},
			}
			expectedGrafeasBatchCreateOccurrencesError = nil

			expectedRodeBatchCreateOccurrencesRequest = &pb.BatchCreateOccurrencesRequest{
				Occurrences: []*grafeas_proto.Occurrence{
					expectedOccurrence,
				},
			}

			expectedBatchCreateResourcesError = nil
		})

		JustBeforeEach(func() {
			resourceManager.BatchCreateGenericResourcesReturns(expectedBatchCreateResourcesError)
			grafeasClient.BatchCreateOccurrencesReturns(expectedGrafeasBatchCreateOccurrencesResponse, expectedGrafeasBatchCreateOccurrencesError)

			actualRodeBatchCreateOccurrencesResponse, actualError = server.BatchCreateOccurrences(ctx, expectedRodeBatchCreateOccurrencesRequest)
		})

		It("should create generic resources from the received occurrences", func() {
			Expect(resourceManager.BatchCreateGenericResourcesCallCount()).To(Equal(1))

			_, batchCreateGenericResourcesOccurrenceRequest := resourceManager.BatchCreateGenericResourcesArgsForCall(0)
			Expect(batchCreateGenericResourcesOccurrenceRequest).To(BeEquivalentTo(expectedRodeBatchCreateOccurrencesRequest))
		})

		It("should send occurrences to Grafeas", func() {
			Expect(grafeasClient.BatchCreateOccurrencesCallCount()).To(Equal(1))

			_, batchCreateOccurrencesRequest, _ := grafeasClient.BatchCreateOccurrencesArgsForCall(0)
			Expect(batchCreateOccurrencesRequest.Occurrences).To(HaveLen(1))
			Expect(batchCreateOccurrencesRequest.Occurrences[0]).To(BeEquivalentTo(expectedOccurrence))
		})

		It("should return the created occurrences", func() {
			Expect(actualRodeBatchCreateOccurrencesResponse.Occurrences).To(HaveLen(1))
			Expect(actualRodeBatchCreateOccurrencesResponse.Occurrences[0]).To(BeEquivalentTo(expectedOccurrence))
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("an error occurs while creating generic resources", func() {
			BeforeEach(func() {
				expectedBatchCreateResourcesError = errors.New("error batch creating generic resources")
			})

			It("should not attempt to create occurrences in grafeas", func() {
				Expect(grafeasClient.BatchCreateOccurrencesCallCount()).To(Equal(0))
			})

			It("should return an error", func() {
				Expect(actualRodeBatchCreateOccurrencesResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("an error occurs while creating occurrences", func() {
			BeforeEach(func() {
				expectedGrafeasBatchCreateOccurrencesError = errors.New("error batch creating occurrences")
			})

			It("should return an error", func() {
				Expect(actualRodeBatchCreateOccurrencesResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})
	})

	Context("ListGenericResources", func() {
		When("querying for generic resources", func() {
			var (
				actualError      error
				actualResponse   *pb.ListGenericResourcesResponse
				listRequest      *pb.ListGenericResourcesRequest
				genericResources []*pb.GenericResource
			)

			BeforeEach(func() {
				genericResources = []*pb.GenericResource{}
				for i := 0; i < gofakeit.Number(3, 5); i++ {
					genericResources = append(genericResources, &pb.GenericResource{Name: gofakeit.LetterN(10)})
				}

				esTransport.preparedHttpResponses = []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       createEsSearchResponseForGenericResource(genericResources),
					},
				}

				listRequest = &pb.ListGenericResourcesRequest{}
			})

			JustBeforeEach(func() {
				actualResponse, actualError = server.ListGenericResources(context.Background(), listRequest)
			})

			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should search against the generic resources index", func() {
				Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodGet))
				Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal(fmt.Sprintf("/%s/_search", rodeElasticsearchGenericResourcesIndex)))
				Expect(esTransport.receivedHttpRequests[0].URL.Query().Get("size")).To(Equal(strconv.Itoa(maxPageSize)))

				body := readEsSearchResponse(esTransport.receivedHttpRequests[0])

				Expect(body).To(Equal(&esutil.EsSearch{
					Sort: map[string]esutil.EsSortOrder{
						"name": esutil.EsSortOrderAscending,
					},
				}))
			})

			It("should return all of the resources", func() {
				var expectedNames []string
				var actualNames []string

				for _, resource := range genericResources {
					expectedNames = append(expectedNames, resource.Name)
				}

				for _, actual := range actualResponse.GenericResources {
					actualNames = append(actualNames, actual.Name)
				}

				Expect(actualNames).To(ConsistOf(expectedNames))
			})

			When("a filter is provided", func() {
				var (
					expectedFilter string
				)

				BeforeEach(func() {
					expectedFilter = gofakeit.LetterN(10)
					listRequest.Filter = expectedFilter
				})

				When("the filter is valid", func() {
					var expectedQuery *filtering.Query

					BeforeEach(func() {
						expectedQuery = &filtering.Query{
							Term: &filtering.Term{
								gofakeit.LetterN(10): gofakeit.LetterN(10),
							},
						}
						mockFilterer.EXPECT().ParseExpression(expectedFilter).Return(expectedQuery, nil)
					})

					It("should include the filter query in the request body", func() {
						body := readEsSearchResponse(esTransport.receivedHttpRequests[0])

						Expect(body).To(Equal(&esutil.EsSearch{
							Query: expectedQuery,
							Sort: map[string]esutil.EsSortOrder{
								"name": esutil.EsSortOrderAscending,
							},
						}))
					})
				})

				When("the filter is invalid", func() {
					BeforeEach(func() {
						mockFilterer.EXPECT().ParseExpression(expectedFilter).Return(nil, errors.New(gofakeit.Word()))
					})

					It("should return an error", func() {
						Expect(actualError).To(HaveOccurred())
					})

					It("should set a gRPC status", func() {
						status := getGRPCStatusFromError(actualError)

						Expect(status.Code()).To(Equal(codes.Internal))
					})
				})
			})

			When("an unexpected status code is returned from the search", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[0].StatusCode = http.StatusInternalServerError
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should set a gRPC status", func() {
					status := getGRPCStatusFromError(actualError)

					Expect(status.Code()).To(Equal(codes.Internal))
				})
			})

			When("an unparseable response is returned from Elasticsearch", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[0].Body = createInvalidResponseBody()
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should set a gRPC status", func() {
					status := getGRPCStatusFromError(actualError)

					Expect(status.Code()).To(Equal(codes.Internal))
				})
			})

			When("an error occurs during the search", func() {
				BeforeEach(func() {
					esTransport.actions = []func(req *http.Request) (*http.Response, error){
						func(req *http.Request) (*http.Response, error) {
							return nil, errors.New(gofakeit.Word())
						},
					}
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should set a gRPC status", func() {
					status := getGRPCStatusFromError(actualError)

					Expect(status.Code()).To(Equal(codes.Internal))
				})
			})

			When("listing generic resources with pagination", func() {
				var (
					expectedPageToken string
					expectedPageSize  int32
					expectedPitId     string
					expectedFrom      int
				)

				BeforeEach(func() {
					expectedPageSize = int32(gofakeit.Number(5, 20))
					expectedPitId = gofakeit.LetterN(20)
					expectedFrom = gofakeit.Number(int(expectedPageSize), 100)

					listRequest.PageSize = expectedPageSize

					esTransport.preparedHttpResponses[0].Body = createPaginatedEsSearchResponseForGenericResource(genericResources, gofakeit.Number(1000, 10000))
				})

				When("a page token is not specified", func() {
					BeforeEach(func() {
						esTransport.preparedHttpResponses = append([]*http.Response{
							{
								StatusCode: http.StatusOK,
								Body: structToJsonBody(&esutil.ESPitResponse{
									Id: expectedPitId,
								}),
							},
						}, esTransport.preparedHttpResponses...)
					})

					It("should create a PIT in Elasticsearch", func() {
						Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal(fmt.Sprintf("/%s/_pit", rodeElasticsearchGenericResourcesIndex)))
						Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodPost))
						Expect(esTransport.receivedHttpRequests[0].URL.Query().Get("keep_alive")).To(Equal("5m"))
					})

					It("should query using the PIT", func() {
						Expect(esTransport.receivedHttpRequests[1].URL.Path).To(Equal("/_search"))
						Expect(esTransport.receivedHttpRequests[1].Method).To(Equal(http.MethodGet))
						request := readEsSearchResponse(esTransport.receivedHttpRequests[1])
						Expect(request.Pit.Id).To(Equal(expectedPitId))
						Expect(request.Pit.KeepAlive).To(Equal("5m"))
					})

					It("should not return an error", func() {
						Expect(actualError).To(BeNil())
					})

					It("should return the next page token", func() {
						nextPitId, nextFrom, err := esutil.ParsePageToken(actualResponse.NextPageToken)

						Expect(err).ToNot(HaveOccurred())
						Expect(nextPitId).To(Equal(expectedPitId))
						Expect(nextFrom).To(BeEquivalentTo(expectedPageSize))
					})
				})

				When("a valid token is specified", func() {
					BeforeEach(func() {
						expectedPageToken = esutil.CreatePageToken(expectedPitId, expectedFrom)

						listRequest.PageToken = expectedPageToken
					})

					It("should query Elasticsearch using the PIT", func() {
						Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal("/_search"))
						Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodGet))
						request := readEsSearchResponse(esTransport.receivedHttpRequests[0])
						Expect(request.Pit.Id).To(Equal(expectedPitId))
						Expect(request.Pit.KeepAlive).To(Equal("5m"))
					})

					It("should return the next page token", func() {
						nextPitId, nextFrom, err := esutil.ParsePageToken(actualResponse.NextPageToken)

						Expect(err).ToNot(HaveOccurred())
						Expect(nextPitId).To(Equal(expectedPitId))
						Expect(nextFrom).To(BeEquivalentTo(expectedPageSize + int32(expectedFrom)))
					})
				})

				When("an invalid token is passed (bad format)", func() {
					BeforeEach(func() {
						listRequest.PageToken = gofakeit.LetterN(10)
					})

					It("should not make any further Elasticsearch queries", func() {
						Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
					})

					It("should return an error", func() {
						Expect(actualError).To(HaveOccurred())
						Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
						Expect(actualResponse).To(BeNil())
					})
				})

				When("an invalid token is passed (bad from)", func() {
					BeforeEach(func() {
						listRequest.PageToken = esutil.CreatePageToken(expectedPitId, expectedFrom) + gofakeit.LetterN(5)
					})

					It("should not make any further Elasticsearch queries", func() {
						Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
					})

					It("should return an error", func() {
						Expect(actualError).To(HaveOccurred())
						Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
						Expect(actualResponse).To(BeNil())
					})
				})
			})
		})
	})

	Context("ListVersionedResourceOccurrences", func() {
		var (
			listBuildOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
			listBuildOccurrencesError    error

			listAllOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
			listAllOccurrencesError    error

			gitResourceUri string

			nextPageToken    string
			currentPageToken string
			pageSize         int32
			ctx              context.Context
			resourceUri      string
			request          *pb.ListVersionedResourceOccurrencesRequest
			actualResponse   *pb.ListVersionedResourceOccurrencesResponse
			actualError      error
		)

		BeforeEach(func() {
			ctx = context.Background()
			resourceUri = gofakeit.URL()
			nextPageToken = gofakeit.Word()
			currentPageToken = gofakeit.Word()
			pageSize = gofakeit.Int32()

			request = &pb.ListVersionedResourceOccurrencesRequest{
				ResourceUri: resourceUri,
				PageToken:   currentPageToken,
				PageSize:    pageSize,
			}

			gitResourceUri = fmt.Sprintf("git://%s", gofakeit.DomainName())

			listBuildOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					{
						Resource: &grafeas_proto.Resource{
							Uri: gitResourceUri,
						},
						Kind: grafeas_common_proto.NoteKind_BUILD,
						Details: &grafeas_proto.Occurrence_Build{
							Build: &build_go_proto.Details{
								Provenance: &provenance_go_proto.BuildProvenance{
									BuiltArtifacts: []*provenance_go_proto.Artifact{
										{
											Id: resourceUri,
										},
									},
								},
							},
						},
					},
				},
			}
			listBuildOccurrencesError = nil

			listAllOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
					createRandomOccurrence(grafeas_common_proto.NoteKind_BUILD),
				},
				NextPageToken: nextPageToken,
			}
			listAllOccurrencesError = nil
		})

		JustBeforeEach(func() {
			grafeasClient.ListOccurrencesReturnsOnCall(0, listBuildOccurrencesResponse, listBuildOccurrencesError)
			grafeasClient.ListOccurrencesReturnsOnCall(1, listAllOccurrencesResponse, listAllOccurrencesError)

			actualResponse, actualError = server.ListVersionedResourceOccurrences(ctx, request)
		})

		It("should list build occurrences for the resource uri", func() {
			_, buildOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(0)

			Expect(buildOccurrencesRequest).NotTo(BeNil())
			Expect(buildOccurrencesRequest.Parent).To(Equal("projects/rode"))
			Expect(buildOccurrencesRequest.Filter).To(ContainSubstring(fmt.Sprintf(`build.provenance.builtArtifacts.nestedFilter(id == "%s")`, resourceUri)))
			Expect(buildOccurrencesRequest.Filter).To(ContainSubstring(fmt.Sprintf(`resource.uri == "%s"`, resourceUri)))
			Expect(buildOccurrencesRequest.PageSize).To(Equal(int32(1000)))
		})

		It("should use the build occurrence to find all occurrences", func() {
			expectedFilter := []string{
				fmt.Sprintf(`resource.uri == "%s"`, resourceUri),
				fmt.Sprintf(`resource.uri == "%s"`, gitResourceUri),
			}

			_, allOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(1)

			Expect(allOccurrencesRequest).NotTo(BeNil())
			Expect(allOccurrencesRequest.Parent).To(Equal("projects/rode"))
			Expect(allOccurrencesRequest.PageSize).To(Equal(pageSize))
			Expect(allOccurrencesRequest.PageToken).To(Equal(currentPageToken))

			filterParts := strings.Split(allOccurrencesRequest.Filter, " || ")
			Expect(filterParts).To(ConsistOf(expectedFilter))
		})

		It("should return the occurrences and page token from the call to list all occurrences", func() {
			Expect(actualResponse.Occurrences).To(BeEquivalentTo(listAllOccurrencesResponse.Occurrences))
			Expect(actualResponse.NextPageToken).To(BeEquivalentTo(listAllOccurrencesResponse.NextPageToken))
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("there are no build occurrences", func() {
			BeforeEach(func() {
				listBuildOccurrencesResponse.Occurrences = []*grafeas_proto.Occurrence{}
			})

			It("should list occurrences for the resource uri", func() {
				_, allOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(1)

				Expect(allOccurrencesRequest.Filter).To(Equal(fmt.Sprintf(`resource.uri == "%s"`, resourceUri)))
			})
		})

		When("the resource uri is not specified", func() {
			BeforeEach(func() {
				request.ResourceUri = ""
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.InvalidArgument))
			})
		})

		When("an error occurs listing build occurrences", func() {
			BeforeEach(func() {
				listBuildOccurrencesError = errors.New("error listing build occurrences")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("an error occurs listing all occurrences", func() {
			BeforeEach(func() {
				listAllOccurrencesError = errors.New("error listing all occurrences")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})
	})

	Context("ListOccurrences", func() {
		var (
			actualResponse *pb.ListOccurrencesResponse
			actualError    error

			expectedOccurrence *grafeas_proto.Occurrence
			expectedPageToken  string
			expectedPageSize   int32
			expectedFilter     string

			expectedListOccurrencesRequest *pb.ListOccurrencesRequest

			expectedGrafeasListOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
			expectedGrafeasListOccurrencesError    error
		)

		BeforeEach(func() {
			expectedOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)

			expectedPageToken = gofakeit.Word()
			expectedPageSize = gofakeit.Int32()
			expectedFilter = fmt.Sprintf(`"resource.uri" == "%s"`, expectedOccurrence.Resource.Uri)

			expectedListOccurrencesRequest = &pb.ListOccurrencesRequest{
				Filter:    expectedFilter,
				PageToken: expectedPageToken,
				PageSize:  expectedPageSize,
			}

			expectedGrafeasListOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					expectedOccurrence,
				},
				NextPageToken: gofakeit.Word(),
			}

			expectedGrafeasListOccurrencesError = nil
		})

		JustBeforeEach(func() {
			grafeasClient.ListOccurrencesReturns(expectedGrafeasListOccurrencesResponse, expectedGrafeasListOccurrencesError)

			actualResponse, actualError = server.ListOccurrences(context.Background(), expectedListOccurrencesRequest)
		})

		It("should list occurrences from grafeas", func() {
			Expect(grafeasClient.ListOccurrencesCallCount()).To(Equal(1))
			_, listOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(0)

			Expect(listOccurrencesRequest.Parent).To(Equal(rodeProjectSlug))
			Expect(listOccurrencesRequest.Filter).To(Equal(expectedFilter))
			Expect(listOccurrencesRequest.PageToken).To(Equal(expectedPageToken))
			Expect(listOccurrencesRequest.PageSize).To(Equal(expectedPageSize))
		})

		It("should return the results from grafeas", func() {
			Expect(actualResponse.Occurrences).To(BeEquivalentTo(expectedGrafeasListOccurrencesResponse.Occurrences))
			Expect(actualResponse.NextPageToken).To(Equal(expectedGrafeasListOccurrencesResponse.NextPageToken))
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("Grafeas returns an error", func() {
			BeforeEach(func() {
				expectedGrafeasListOccurrencesError = errors.New("error listing occurrences")
			})

			It("should return an error", func() {
				Expect(actualResponse).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})
	})

	Context("UpdateOccurrence", func() {
		var (
			actualError    error
			actualResponse *grafeas_go_proto.Occurrence

			expectedOccurrence              *grafeas_proto.Occurrence
			expectedUpdateOccurrenceRequest *pb.UpdateOccurrenceRequest

			expectedGrafeasUpdateOccurrenceResponse *grafeas_proto.Occurrence
			expectedGrafeasUpdateOccurrenceError    error
		)

		BeforeEach(func() {
			expectedOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
			occurrenceId := gofakeit.UUID()
			occurrenceName := fmt.Sprintf("projects/rode/occurrences/%s", occurrenceId)
			expectedOccurrence.Name = occurrenceName
			expectedUpdateOccurrenceRequest = &pb.UpdateOccurrenceRequest{
				Id:         occurrenceId,
				Occurrence: expectedOccurrence,
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{gofakeit.Word()},
				},
			}

			expectedGrafeasUpdateOccurrenceResponse = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
			expectedGrafeasUpdateOccurrenceError = nil
		})

		JustBeforeEach(func() {
			grafeasClient.UpdateOccurrenceReturns(expectedGrafeasUpdateOccurrenceResponse, expectedGrafeasUpdateOccurrenceError)

			actualResponse, actualError = server.UpdateOccurrence(context.Background(), expectedUpdateOccurrenceRequest)
		})

		It("should update the occurrence in grafeas", func() {
			Expect(grafeasClient.UpdateOccurrenceCallCount()).To(Equal(1))

			_, updateOccurrenceRequest, _ := grafeasClient.UpdateOccurrenceArgsForCall(0)
			Expect(updateOccurrenceRequest.Name).To(Equal(expectedOccurrence.Name))
			Expect(updateOccurrenceRequest.Occurrence).To(Equal(expectedOccurrence))
			Expect(updateOccurrenceRequest.UpdateMask).To(Equal(expectedUpdateOccurrenceRequest.UpdateMask))
		})

		It("should return the updated occurrence", func() {
			Expect(actualError).ToNot(HaveOccurred())
			Expect(actualResponse).To(Equal(expectedGrafeasUpdateOccurrenceResponse))
		})

		When("Grafeas returns an error", func() {
			BeforeEach(func() {
				expectedGrafeasUpdateOccurrenceError = errors.New("error updating occurrence")
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(actualResponse).To(BeNil())
			})
		})

		When("the occurrence name doesn't contain the occurrence id", func() {
			BeforeEach(func() {
				expectedUpdateOccurrenceRequest.Id = gofakeit.UUID()
			})

			It("should return an error", func() {
				Expect(actualError).ToNot(BeNil())
			})

			It("should return a status code of invalid argument", func() {
				s, ok := status.FromError(actualError)
				Expect(ok).To(BeTrue(), "Expected error to be a gRPC status")

				Expect(s.Code()).To(Equal(codes.InvalidArgument))
				Expect(s.Message()).To(ContainSubstring("occurrence name does not contain the occurrence id"))
			})

			It("should not attempt to update the occurrence", func() {
				Expect(grafeasClient.UpdateOccurrenceCallCount()).To(Equal(0))
			})
		})
	})

	Context("EvaluatePolicy", func() {
		var (
			policy                    string
			resourceURI               string
			evaluatePolicyRequest     *pb.EvaluatePolicyRequest
			opaEvaluatePolicyResponse *opa.EvaluatePolicyResponse

			expectedListOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
			expectedListOccurrencesError    error
		)

		BeforeEach(func() {
			resourceURI = gofakeit.URL()
			policy = goodPolicy
			occurrences := []*grafeas_proto.Occurrence{
				createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
				createRandomOccurrence(grafeas_common_proto.NoteKind_ATTESTATION),
			}

			evaluatePolicyRequest = &pb.EvaluatePolicyRequest{
				ResourceUri: resourceURI,
				Policy:      policy,
			}
			opaEvaluatePolicyResponse = &opa.EvaluatePolicyResponse{
				Result: &opa.EvaluatePolicyResult{
					Pass: false,
				},
				Explanation: &[]string{},
			}
			createPolicyRequest := createRandomPolicyEntity(goodPolicy)
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
				},
			}
			createPolicyResponse, _ := server.CreatePolicy(context.Background(), createPolicyRequest)
			esTransport.preparedHttpResponses = []*http.Response{

				{
					StatusCode: http.StatusOK,
					Body:       createEsSearchResponseForPolicy([]*pb.Policy{createPolicyResponse}),
				},
				{
					StatusCode: http.StatusOK,
					Body:       createEsSearchResponse(occurrences),
				},
			}

			expectedListOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED),
				},
			}
			expectedListOccurrencesError = nil
		})

		JustBeforeEach(func() {
			grafeasClient.ListOccurrencesReturns(expectedListOccurrencesResponse, expectedListOccurrencesError)
		})

		It("should initialize OPA policy", func() {
			// ignore non test calls
			opaClient.EXPECT().EvaluatePolicy(gomock.Any(), gomock.Any()).AnyTimes().Return(opaEvaluatePolicyResponse, nil)

			// expect OPA initialize policy call
			opaClient.EXPECT().InitializePolicy(policy, goodPolicy).Return(nil)

			_, _ = server.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
		})

		It("should return an error if resource uri is not specified", func() {
			opaClient.EXPECT().EvaluatePolicy(gomock.Any(), gomock.Any()).Times(0)
			opaClient.EXPECT().InitializePolicy(policy, goodPolicy).Times(0)

			evaluatePolicyRequest.ResourceUri = ""
			_, err := server.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeEquivalentTo(status.Errorf(codes.InvalidArgument, "resource uri is required")))
			Expect(grafeasClient.ListOccurrencesCallCount()).To(Equal(0))
		})

		When("OPA policy initializes", func() {
			BeforeEach(func() {
				opaClient.EXPECT().InitializePolicy(gomock.Any(), goodPolicy).AnyTimes().Return(nil)
			})

			It("should list Grafeas occurrences", func() {
				// ingore non test calls
				opaClient.EXPECT().EvaluatePolicy(gomock.Any(), gomock.Any()).AnyTimes().Return(opaEvaluatePolicyResponse, nil)

				_, _ = server.EvaluatePolicy(context.Background(), evaluatePolicyRequest)

				Expect(grafeasClient.ListOccurrencesCallCount()).To(Equal(1))
				_, listOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(0)

				Expect(listOccurrencesRequest.Parent).To(Equal(rodeProjectSlug))
				Expect(listOccurrencesRequest.PageSize).To(Equal(int32(maxPageSize)))
				Expect(listOccurrencesRequest.Filter).To(Equal(fmt.Sprintf(`"resource.uri" == "%s"`, resourceURI)))
			})

			When("Grafeas list occurrences response is ok", func() {
				It("should evaluate OPA policy", func() {
					opaClient.EXPECT().EvaluatePolicy(gomock.Eq(policy), gomock.Any()).Return(opaEvaluatePolicyResponse, nil)

					_, _ = server.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
				})

				When("evalute OPA policy returns error", func() {
					It("should return error", func() {
						opaClient.EXPECT().EvaluatePolicy(gomock.Eq(policy), gomock.Any()).Return(nil, fmt.Errorf("OPA Error"))

						_, evaluatePolicyError := server.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
						Expect(evaluatePolicyError).To(HaveOccurred())
						Expect(evaluatePolicyError.Error()).To(ContainSubstring("error evaluating policy"))
					})
				})
			})

			When("Grafeas list occurrences response is error", func() {
				BeforeEach(func() {
					expectedListOccurrencesError = errors.New("error listing occurrences")
				})

				It("should return an error", func() {
					_, evaluatePolicyError := server.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
					Expect(evaluatePolicyError).To(HaveOccurred())
				})
			})
		})

		When("OPA policy is not found", func() {
			It("should return an error", func() {
				opaClient.EXPECT().InitializePolicy(gomock.Any(), goodPolicy).Return(opa.NewClientError("policy not found", opa.OpaClientErrorTypePolicyNotFound, fmt.Errorf("es search result empty")))

				_, evaluatePolicyError := server.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
				Expect(evaluatePolicyError).To(HaveOccurred())
			})
		})
	})

	Context("ListResources", func() {
		var (
			occurrences              []*grafeas_proto.Occurrence
			request                  *pb.ListResourcesRequest
			listResourcesResponse    *pb.ListResourcesResponse
			listResourcesResponseErr error
		)

		BeforeEach(func() {
			request = &pb.ListResourcesRequest{}
			occurrences = []*grafeas_proto.Occurrence{
				createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
				createRandomOccurrence(grafeas_common_proto.NoteKind_ATTESTATION),
			}
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       createEsSearchResponse(occurrences),
				},
			}
		})

		When("querying for resources without a filter", func() {
			BeforeEach(func() {
				mockFilterer.EXPECT().ParseExpression(gomock.Any()).Times(0)

				listResourcesResponse, listResourcesResponseErr = server.ListResources(context.Background(), request)
			})

			It("should query the Rode occurrences index", func() {
				actualRequest := esTransport.receivedHttpRequests[0]

				Expect(actualRequest.URL.Path).To(Equal("/grafeas-rode-occurrences/_search"))
			})

			It("should take the first 1000 matches", func() {
				actualRequest := esTransport.receivedHttpRequests[0]
				query := actualRequest.URL.Query()

				Expect(query.Get("size")).To(Equal("1000"))
			})

			It("should collapse fields on resource.uri", func() {
				actualRequest := esTransport.receivedHttpRequests[0]
				search := readEsSearchResponse(actualRequest)

				Expect(search.Collapse.Field).To(Equal("resource.uri"))
			})

			It("should not return an error", func() {
				Expect(listResourcesResponseErr).To(BeNil())
			})

			It("should return all resources from the query", func() {
				var expectedResourceUris []string

				for _, occurrence := range occurrences {
					expectedResourceUris = append(expectedResourceUris, occurrence.Resource.Uri)
				}

				var actualResourceUris []string
				for _, resource := range listResourcesResponse.Resources {
					actualResourceUris = append(actualResourceUris, resource.Uri)
				}

				Expect(actualResourceUris).To(ConsistOf(expectedResourceUris))
			})
		})

		When("querying for resources with a filter", func() {
			BeforeEach(func() {
				request.Filter = gofakeit.UUID()
			})

			It("should pass the filter to the filterer", func() {
				mockFilterer.EXPECT().ParseExpression(request.Filter)

				_, _ = server.ListResources(context.Background(), request)
			})

			It("should include the Elasticsearch query in the response", func() {
				expectedQuery := &filtering.Query{
					Term: &filtering.Term{
						gofakeit.UUID(): gofakeit.UUID(),
					},
				}
				mockFilterer.EXPECT().ParseExpression(gomock.Any()).Return(expectedQuery, nil)

				_, err := server.ListResources(context.Background(), request)
				Expect(err).To(BeNil())

				actualRequest := esTransport.receivedHttpRequests[0]
				search := readEsSearchResponse(actualRequest)

				Expect(search.Query).To(Equal(expectedQuery))
			})
		})

		When("elasticsearch returns with an error", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses[0] = &http.Response{
					StatusCode: http.StatusInternalServerError,
				}

				listResourcesResponse, listResourcesResponseErr = server.ListResources(context.Background(), request)
			})

			It("should return the error", func() {
				Expect(listResourcesResponseErr).ToNot(BeNil())
			})

			It("should not return a protobuf response", func() {
				Expect(listResourcesResponse).To(BeNil())
			})
		})

		When("listing resources with pagination", func() {
			var (
				actualResponse *pb.ListResourcesResponse
				actualError    error

				expectedPageToken string
				expectedPageSize  int32
				expectedPitId     string
				expectedFrom      int
			)

			BeforeEach(func() {
				expectedPageSize = int32(gofakeit.Number(5, 20))
				expectedPitId = gofakeit.LetterN(20)
				expectedFrom = gofakeit.Number(int(expectedPageSize), 100)

				request.PageSize = expectedPageSize

				esTransport.preparedHttpResponses[0].Body = createPaginatedEsSearchResponse(occurrences, gofakeit.Number(1000, 2000))
			})

			JustBeforeEach(func() {
				actualResponse, actualError = server.ListResources(context.Background(), request)
			})

			When("a page token is not specified", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses = append([]*http.Response{
						{
							StatusCode: http.StatusOK,
							Body: structToJsonBody(&esutil.ESPitResponse{
								Id: expectedPitId,
							}),
						},
					}, esTransport.preparedHttpResponses...)
				})

				It("should create a PIT in Elasticsearch", func() {
					Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal(fmt.Sprintf("/%s/_pit", rodeElasticsearchOccurrencesAlias)))
					Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodPost))
					Expect(esTransport.receivedHttpRequests[0].URL.Query().Get("keep_alive")).To(Equal("5m"))
				})

				It("should query using the PIT", func() {
					Expect(esTransport.receivedHttpRequests[1].URL.Path).To(Equal("/_search"))
					Expect(esTransport.receivedHttpRequests[1].Method).To(Equal(http.MethodGet))
					request := readEsSearchResponse(esTransport.receivedHttpRequests[1])
					Expect(request.Pit.Id).To(Equal(expectedPitId))
					Expect(request.Pit.KeepAlive).To(Equal("5m"))
				})

				It("should not return an error", func() {
					Expect(actualError).To(BeNil())
				})

				It("should return the next page token", func() {
					nextPitId, nextFrom, err := esutil.ParsePageToken(actualResponse.NextPageToken)

					Expect(err).ToNot(HaveOccurred())
					Expect(nextPitId).To(Equal(expectedPitId))
					Expect(nextFrom).To(BeEquivalentTo(expectedPageSize))
				})
			})

			When("a valid token is specified", func() {
				BeforeEach(func() {
					expectedPageToken = esutil.CreatePageToken(expectedPitId, expectedFrom)

					request.PageToken = expectedPageToken
				})

				It("should query Elasticsearch using the PIT", func() {
					Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal("/_search"))
					Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodGet))
					request := readEsSearchResponse(esTransport.receivedHttpRequests[0])
					Expect(request.Pit.Id).To(Equal(expectedPitId))
					Expect(request.Pit.KeepAlive).To(Equal("5m"))
				})

				It("should return the next page token", func() {
					nextPitId, nextFrom, err := esutil.ParsePageToken(actualResponse.NextPageToken)

					Expect(err).ToNot(HaveOccurred())
					Expect(nextPitId).To(Equal(expectedPitId))
					Expect(nextFrom).To(BeEquivalentTo(expectedPageSize + int32(expectedFrom)))
				})
			})

			When("an invalid token is passed (bad format)", func() {
				BeforeEach(func() {
					request.PageToken = gofakeit.LetterN(10)
				})

				It("should not make any further Elasticsearch queries", func() {
					Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
					Expect(actualResponse).To(BeNil())
				})
			})

			When("an invalid token is passed (bad from)", func() {
				BeforeEach(func() {
					request.PageToken = esutil.CreatePageToken(expectedPitId, expectedFrom) + gofakeit.LetterN(5)
				})

				It("should not make any further Elasticsearch queries", func() {
					Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
					Expect(actualResponse).To(BeNil())
				})
			})
		})
	})

	Context("RegisterCollector", func() {
		var (
			actualRegisterCollectorResponse *pb.RegisterCollectorResponse
			actualRegisterCollectorError    error

			expectedCollectorId              string
			expectedRegisterCollectorRequest *pb.RegisterCollectorRequest

			expectedListNotesResponse *grafeas_proto.ListNotesResponse
			expectedListNotesError    error

			expectedBatchCreateNotesResponse *grafeas_proto.BatchCreateNotesResponse
			expectedBatchCreateNotesError    error

			expectedNotes []*grafeas_proto.Note
		)

		BeforeEach(func() {
			expectedDiscoveryNote := &grafeas_proto.Note{
				ShortDescription: "Harbor Image Scan",
				Kind:             grafeas_common_proto.NoteKind_DISCOVERY,
			}
			expectedAttestationNote := &grafeas_proto.Note{
				ShortDescription: "Harbor Attestation",
				Kind:             grafeas_common_proto.NoteKind_ATTESTATION,
			}
			expectedNotes = []*grafeas_proto.Note{
				expectedDiscoveryNote,
				expectedAttestationNote,
			}
			expectedCollectorId = gofakeit.LetterN(10)
			expectedRegisterCollectorRequest = &pb.RegisterCollectorRequest{
				Id:    expectedCollectorId,
				Notes: expectedNotes,
			}

			// happy path: notes do not already exist
			expectedListNotesResponse = &grafeas_proto.ListNotesResponse{
				Notes: []*grafeas_proto.Note{},
			}
			expectedListNotesError = nil

			// when notes are returned, their name should not be empty
			expectedCreatedDiscoveryNote := deepCopyNote(expectedDiscoveryNote)
			expectedCreatedAttestationNote := deepCopyNote(expectedAttestationNote)

			expectedCreatedDiscoveryNote.Name = fmt.Sprintf("%s/notes/%s", rodeProjectSlug, buildNoteIdFromCollectorId(expectedCollectorId, expectedCreatedDiscoveryNote))
			expectedCreatedAttestationNote.Name = fmt.Sprintf("%s/notes/%s", rodeProjectSlug, buildNoteIdFromCollectorId(expectedCollectorId, expectedCreatedAttestationNote))

			expectedBatchCreateNotesResponse = &grafeas_proto.BatchCreateNotesResponse{
				Notes: []*grafeas_proto.Note{
					expectedCreatedDiscoveryNote,
					expectedCreatedAttestationNote,
				},
			}
			expectedBatchCreateNotesError = nil
		})

		JustBeforeEach(func() {
			grafeasClient.ListNotesReturns(expectedListNotesResponse, expectedListNotesError)
			grafeasClient.BatchCreateNotesReturns(expectedBatchCreateNotesResponse, expectedBatchCreateNotesError)

			actualRegisterCollectorResponse, actualRegisterCollectorError = server.RegisterCollector(ctx, expectedRegisterCollectorRequest)
		})

		It("should search grafeas for the notes", func() {
			Expect(grafeasClient.ListNotesCallCount()).To(Equal(1))

			_, listNotesRequest, _ := grafeasClient.ListNotesArgsForCall(0)
			Expect(listNotesRequest.Parent).To(Equal(rodeProjectSlug))
			Expect(listNotesRequest.Filter).To(Equal(fmt.Sprintf(`name.startsWith("%s/notes/%s-")`, rodeProjectSlug, expectedCollectorId)))
		})

		It("should create the missing notes", func() {
			Expect(grafeasClient.BatchCreateNotesCallCount()).To(Equal(1))

			_, batchCreateNotesRequest, _ := grafeasClient.BatchCreateNotesArgsForCall(0)
			Expect(batchCreateNotesRequest.Parent).To(Equal(rodeProjectSlug))
			Expect(batchCreateNotesRequest.Notes).To(ConsistOf(expectedNotes))
		})

		It("should return the collector's notes", func() {
			Expect(actualRegisterCollectorResponse).ToNot(BeNil())
			Expect(actualRegisterCollectorResponse.Notes).To(HaveLen(len(expectedNotes)))
			for _, note := range expectedNotes {
				note.Name = buildNoteIdFromCollectorId(expectedCollectorId, note)
				Expect(actualRegisterCollectorResponse.Notes).To(ContainElement(note))
			}

			Expect(actualRegisterCollectorError).ToNot(HaveOccurred())
		})

		When("a note already exists", func() {
			BeforeEach(func() {
				expectedNoteThatAlreadyExists := deepCopyNote(expectedNotes[0])
				expectedNoteThatAlreadyExists.Name = fmt.Sprintf("%s/notes/%s", rodeProjectSlug, buildNoteIdFromCollectorId(expectedCollectorId, expectedNoteThatAlreadyExists))

				expectedListNotesResponse.Notes = []*grafeas_proto.Note{
					expectedNoteThatAlreadyExists,
				}
			})

			It("should not attempt to create that note", func() {
				Expect(grafeasClient.BatchCreateNotesCallCount()).To(Equal(1))

				_, batchCreateNotesRequest, _ := grafeasClient.BatchCreateNotesArgsForCall(0)
				Expect(batchCreateNotesRequest.Parent).To(Equal(rodeProjectSlug))
				Expect(batchCreateNotesRequest.Notes).To(HaveLen(1))
				Expect(batchCreateNotesRequest.Notes).To(ContainElement(expectedNotes[1]))
			})
		})

		When("both notes already exist", func() {
			BeforeEach(func() {
				var notesThatAlreadyExist []*grafeas_proto.Note
				for _, note := range expectedNotes {
					noteThatAlreadyExists := deepCopyNote(note)
					noteThatAlreadyExists.Name = fmt.Sprintf("%s/notes/%s", rodeProjectSlug, buildNoteIdFromCollectorId(expectedCollectorId, noteThatAlreadyExists))

					notesThatAlreadyExist = append(notesThatAlreadyExist, noteThatAlreadyExists)
				}

				expectedListNotesResponse.Notes = notesThatAlreadyExist
			})

			It("should not attempt to create any notes", func() {
				Expect(grafeasClient.BatchCreateNotesCallCount()).To(Equal(0))
			})
		})
	})

	When("creating a policy without a name", func() {
		var (
			policyEntity   *pb.PolicyEntity
			policyResponse *pb.Policy
			err            error
		)

		BeforeEach(func() {
			policyEntity = &pb.PolicyEntity{
				Name: "",
			}

			policyResponse, err = server.CreatePolicy(context.Background(), policyEntity)
		})

		It("should throw a Bad Request Error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeEquivalentTo(status.Errorf(codes.InvalidArgument, "policy name not provided")))
			Expect(policyResponse).To(BeNil())
		})
	})

	When("creating a policy", func() {
		var (
			policyEntity   *pb.PolicyEntity
			policyResponse *pb.Policy
			err            error
		)

		BeforeEach(func() {
			policyEntity = createRandomPolicyEntity(goodPolicy)
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
				},
			}
			policyResponse, err = server.CreatePolicy(context.Background(), policyEntity)
		})

		It("should have a correct url path", func() {
			Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal("/rode-v1alpha1-policies/_doc"))
		})

		It("should match the policy entity", func() {
			Expect(err).To(Not(HaveOccurred()))
			Expect(policyResponse.Policy).To(BeEquivalentTo(policyEntity))
		})

		When("attemtpting to retrieve the same policy", func() {
			var (
				getResponse *pb.Policy
				err         error
			)
			BeforeEach(func() {

				esTransport.preparedHttpResponses = []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       createEsSearchResponseForPolicy([]*pb.Policy{policyResponse}),
					},
				}
				getResponse, err = server.GetPolicy(context.Background(), &pb.GetPolicyRequest{Id: policyResponse.Id})
			})
			It("should not return an error", func() {
				Expect(err).To(Not(HaveOccurred()))
			})
			It("should have the same id as the one originally created", func() {
				Expect(getResponse.Id).To(Equal(policyResponse.Id))
			})
		})

		When("attempting to delete the same policy", func() {
			var deleteResponse *emptypb.Empty

			BeforeEach(func() {
				esTransport.preparedHttpResponses = []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       structToJsonBody(createEsDeleteDocResponse()),
					},
				}
				deleteResponse, err = server.DeletePolicy(context.Background(), &pb.DeletePolicyRequest{Id: policyResponse.Id})
			})
			It("should not return an error", func() {
				Expect(err).To(Not(HaveOccurred()))
			})
			It("should have returned an empty response", func() {
				Expect(deleteResponse).To(BeEquivalentTo(&emptypb.Empty{}))
			})
		})
	})

	When("creating multiple policies sequentially", func() {
		var (
			policyEntityOne   *pb.PolicyEntity
			policyEntityTwo   *pb.PolicyEntity
			policyResponseOne *pb.Policy
			policyResponseTwo *pb.Policy
			err               error
		)

		BeforeEach(func() {
			policyEntityOne = createRandomPolicyEntity(goodPolicy)
			policyEntityTwo = createRandomPolicyEntity(goodPolicy)
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
				},
				{
					StatusCode: http.StatusOK,
				},
			}
			policyResponseOne, err = server.CreatePolicy(context.Background(), policyEntityOne)
			policyResponseTwo, err = server.CreatePolicy(context.Background(), policyEntityTwo)
		})
		It("should not return an error", func() {
			Expect(err).To(Not(HaveOccurred()))
		})
		It("should have created two different policies", func() {
			Expect(policyResponseOne.Id).To(Not(Equal(policyResponseTwo.Id)))
		})

		When("attempting to list the policies", func() {
			var (
				listRequest   *pb.ListPoliciesRequest
				listResponse  *pb.ListPoliciesResponse
				policiesList  []*pb.Policy
				err           error
				filter        string
				expectedQuery *filtering.Query
			)

			BeforeEach(func() {
				policiesList = append(policiesList, policyResponseOne)
				policiesList = append(policiesList, policyResponseOne)

				filter = `name=="abc"`
				expectedQuery := &filtering.Query{
					Term: &filtering.Term{
						"name": "abc",
					},
				}

				listRequest = &pb.ListPoliciesRequest{Filter: filter}
				esTransport.preparedHttpResponses = []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       createEsSearchResponseForPolicy(policiesList),
					},
					{
						StatusCode: http.StatusOK,
						Body:       createEsSearchResponseForPolicy(policiesList),
					},
				}

				mockFilterer.EXPECT().ParseExpression(gomock.Any()).Return(expectedQuery, nil)
				listResponse, err = server.ListPolicies(context.Background(), listRequest)
			})

			It("should not return an error", func() {
				Expect(err).To(Not(HaveOccurred()))
			})

			It("should have listed 4 different policies", func() {
				Expect(listResponse.Policies).To(HaveLen(4))
			})

			It("should have generated a filter query", func() {
				actualRequest := esTransport.receivedHttpRequests[0]
				search := readEsSearchResponse(actualRequest)

				Expect(search.Query).To(Equal(expectedQuery))
			})

			It("should have generated a filter query", func() {
				actualRequest := esTransport.receivedHttpRequests[0]
				search := readEsSearchResponse(actualRequest)

				Expect(search.Query).To(Equal(expectedQuery))
			})
		})
	})

	When("listing policies with pagination", func() {
		var (
			listRequest  *pb.ListPoliciesRequest
			listResponse *pb.ListPoliciesResponse
			actualError  error

			expectedPageToken string
			expectedPageSize  int32
			expectedPitId     string
			expectedFrom      int
		)

		BeforeEach(func() {
			expectedPageSize = int32(gofakeit.Number(5, 20))
			expectedPitId = gofakeit.LetterN(20)
			expectedFrom = gofakeit.Number(int(expectedPageSize), 100)

			listRequest = &pb.ListPoliciesRequest{
				PageSize: expectedPageSize,
			}

			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: createPaginatedEsSearchResponseForPolicy([]*pb.Policy{
						{
							Id:     gofakeit.UUID(),
							Policy: createRandomPolicyEntity(goodPolicy),
						},
					}, gofakeit.Number(1000, 10000)),
				},
			}
		})

		JustBeforeEach(func() {
			listResponse, actualError = server.ListPolicies(context.Background(), listRequest)
		})

		When("a page token is not specified", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses = append([]*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: structToJsonBody(&esutil.ESPitResponse{
							Id: expectedPitId,
						}),
					},
				}, esTransport.preparedHttpResponses...)
			})

			It("should create a PIT in Elasticsearch", func() {
				Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal(fmt.Sprintf("/%s/_pit", rodeElasticsearchPoliciesIndex)))
				Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodPost))
				Expect(esTransport.receivedHttpRequests[0].URL.Query().Get("keep_alive")).To(Equal("5m"))
			})

			It("should query using the PIT", func() {
				Expect(esTransport.receivedHttpRequests[1].URL.Path).To(Equal("/_search"))
				Expect(esTransport.receivedHttpRequests[1].Method).To(Equal(http.MethodGet))
				request := readEsSearchResponse(esTransport.receivedHttpRequests[1])
				Expect(request.Pit.Id).To(Equal(expectedPitId))
				Expect(request.Pit.KeepAlive).To(Equal("5m"))
			})

			It("should not return an error", func() {
				Expect(actualError).To(BeNil())
			})

			It("should return the next page token", func() {
				nextPitId, nextFrom, err := esutil.ParsePageToken(listResponse.NextPageToken)

				Expect(err).ToNot(HaveOccurred())
				Expect(nextPitId).To(Equal(expectedPitId))
				Expect(nextFrom).To(BeEquivalentTo(expectedPageSize))
			})
		})

		When("a valid token is specified", func() {
			BeforeEach(func() {
				expectedPageToken = esutil.CreatePageToken(expectedPitId, expectedFrom)

				listRequest.PageToken = expectedPageToken
			})

			It("should query Elasticsearch using the PIT", func() {
				Expect(esTransport.receivedHttpRequests[0].URL.Path).To(Equal("/_search"))
				Expect(esTransport.receivedHttpRequests[0].Method).To(Equal(http.MethodGet))
				request := readEsSearchResponse(esTransport.receivedHttpRequests[0])
				Expect(request.Pit.Id).To(Equal(expectedPitId))
				Expect(request.Pit.KeepAlive).To(Equal("5m"))
			})

			It("should return the next page token", func() {
				nextPitId, nextFrom, err := esutil.ParsePageToken(listResponse.NextPageToken)

				Expect(err).ToNot(HaveOccurred())
				Expect(nextPitId).To(Equal(expectedPitId))
				Expect(nextFrom).To(BeEquivalentTo(int(expectedPageSize) + expectedFrom))
			})

			When("the user reaches the last page of results", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[0].Body = createPaginatedEsSearchResponseForPolicy([]*pb.Policy{
						{
							Id:     gofakeit.UUID(),
							Policy: createRandomPolicyEntity(goodPolicy),
						},
					}, gofakeit.Number(1, int(expectedPageSize)+expectedFrom-1))
				})

				It("should return an empty next page token", func() {
					Expect(listResponse.NextPageToken).To(Equal(""))
				})
			})
		})

		When("an invalid token is passed (bad format)", func() {
			BeforeEach(func() {
				listRequest.PageToken = gofakeit.LetterN(10)
			})

			It("should not make any further Elasticsearch queries", func() {
				Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				Expect(listResponse).To(BeNil())
			})
		})

		When("an invalid token is passed (bad from)", func() {
			BeforeEach(func() {
				listRequest.PageToken = esutil.CreatePageToken(expectedPitId, expectedFrom) + gofakeit.LetterN(5)
			})

			It("should not make any further Elasticsearch queries", func() {
				Expect(esTransport.receivedHttpRequests).To(HaveLen(0))
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				Expect(listResponse).To(BeNil())
			})
		})
	})

	When("attempting to list an empty policy index", func() {
		var (
			listRequest          *pb.ListPoliciesRequest
			policiesList         []*pb.Policy
			listPoliciesResponse *pb.ListPoliciesResponse
			err                  error
		)
		BeforeEach(func() {
			listRequest = &pb.ListPoliciesRequest{}
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       createEsSearchResponseForPolicy(policiesList),
				},
			}
			listPoliciesResponse, err = server.ListPolicies(context.Background(), listRequest)
		})
		It("should return an error", func() {
			Expect(err).To(Not(HaveOccurred()))
		})
		It("should return an empty list", func() {
			Expect(len(listPoliciesResponse.Policies)).To(Equal(0))
		})

	})

	When("attempting to list policies and elasticsearch is unreachable", func() {
		var (
			listRequest  *pb.ListPoliciesRequest
			policiesList []*pb.Policy
			err          error
		)
		BeforeEach(func() {
			listRequest = &pb.ListPoliciesRequest{}
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       createEsSearchResponseForPolicy(policiesList),
				},
			}
			_, err = server.ListPolicies(context.Background(), listRequest)
		})
		It("should return an error", func() {
			Expect(err).To(HaveOccurred())
		})

	})

	When("creating an unparseable policy", func() {
		var (
			policyEntity   *pb.PolicyEntity
			policyResponse *pb.Policy
			err            error
		)

		BeforeEach(func() {
			policyEntity = createRandomPolicyEntity(unparseablePolicy)
			policyResponse, err = server.CreatePolicy(context.Background(), policyEntity)
		})

		It("should throw a compilation error", func() {
			Expect(policyResponse).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	When("creating an compilable policy with missing Rode Requirements", func() {
		var (
			policyEntity   *pb.PolicyEntity
			policyResponse *pb.Policy
			err            error
		)

		BeforeEach(func() {
			policyEntity = createRandomPolicyEntity(compilablePolicyMissingRodeFields)
			policyResponse, err = server.CreatePolicy(context.Background(), policyEntity)
		})

		It("should throw a compilation error", func() {
			Expect(policyResponse).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	When("creating an compilable policy with missing results return", func() {
		var (
			policyEntity   *pb.PolicyEntity
			policyResponse *pb.Policy
			err            error
		)

		BeforeEach(func() {
			policyEntity = createRandomPolicyEntity(compilablePolicyMissingResultsReturn)
			policyResponse, err = server.CreatePolicy(context.Background(), policyEntity)
		})

		It("should throw a compilation error", func() {
			Expect(policyResponse).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	When("creating an compilable policy with missing required fields in the result object", func() {
		var (
			policyEntity   *pb.PolicyEntity
			policyResponse *pb.Policy
			err            error
		)

		BeforeEach(func() {
			policyEntity = createRandomPolicyEntity(compilablePolicyMissingResultsFields)
			policyResponse, err = server.CreatePolicy(context.Background(), policyEntity)
		})

		It("should throw a compilation error", func() {
			Expect(policyResponse).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	When("validating a good policy", func() {
		var (
			validatePolicyRequest  *pb.ValidatePolicyRequest
			validatePolicyResponse *pb.ValidatePolicyResponse
			err                    error
		)

		BeforeEach(func() {
			validatePolicyRequest = &pb.ValidatePolicyRequest{Policy: goodPolicy}
			validatePolicyResponse, err = server.ValidatePolicy(context.Background(), validatePolicyRequest)
		})

		It("should not throw an error", func() {
			Expect(err).To(Not(HaveOccurred()))
		})
		It("should return a successful compilation", func() {
			Expect(validatePolicyResponse.Compile).To(BeTrue())
		})
		It("should return an empty error array", func() {
			Expect(validatePolicyResponse.Errors).To(BeEmpty())
		})
	})

	When("validating an empty policy", func() {
		var (
			validatePolicyRequest *pb.ValidatePolicyRequest
			err                   error
		)

		BeforeEach(func() {
			validatePolicyRequest = &pb.ValidatePolicyRequest{Policy: ""}
			_, err = server.ValidatePolicy(context.Background(), validatePolicyRequest)
		})

		It("should throw an error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("validating an uncompilable policy", func() {
		var (
			validatePolicyRequest  *pb.ValidatePolicyRequest
			validatePolicyResponse *pb.ValidatePolicyResponse
			err                    error
		)

		BeforeEach(func() {
			validatePolicyRequest = &pb.ValidatePolicyRequest{Policy: uncompilablePolicy}
			validatePolicyResponse, err = server.ValidatePolicy(context.Background(), validatePolicyRequest)
		})

		It("should throw an error", func() {
			Expect(err).To(HaveOccurred())
		})
		It("should return an unsuccessful compilation", func() {
			Expect(validatePolicyResponse.Compile).To(BeFalse())
		})
		It("should not return an empty error array", func() {
			Expect(len(validatePolicyResponse.Errors)).To(Not(Equal(0)))
		})
	})

	When("updating the name of a policy", func() {
		var (
			createPolicyRequest  *pb.PolicyEntity
			createPolicyResponse *pb.Policy
			updatePolicyRequest  *pb.UpdatePolicyRequest
			updatePolicyResponse *pb.Policy
			err                  error
			initialPolicyName    string
		)

		BeforeEach(func() {
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
				},
			}

			createPolicyRequest = createRandomPolicyEntity(goodPolicy)
			createPolicyResponse, _ = server.CreatePolicy(context.Background(), createPolicyRequest)
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       createEsSearchResponseForPolicy([]*pb.Policy{createPolicyResponse}),
				},
				{
					StatusCode: http.StatusOK,
				},
			}

			initialPolicyName = createPolicyResponse.Policy.Name
			updatePolicyRequest = &pb.UpdatePolicyRequest{
				Id: createPolicyResponse.Id,
				Policy: &pb.PolicyEntity{
					Name: "random name",
				},
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{"name"},
				},
			}
			updatePolicyResponse, err = server.UpdatePolicy(context.Background(), updatePolicyRequest)
		})

		It("should not throw an error", func() {
			Expect(err).To(Not(HaveOccurred()))
		})
		It("should now have a new policy name", func() {
			Expect(initialPolicyName).To(Not(Equal(updatePolicyResponse.Policy.Name)))
		})
		When("the original policy does not exist", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses = []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       createEsSearchResponseForPolicy([]*pb.Policy{}),
					},
					{
						StatusCode: http.StatusOK,
					},
				}
				updatePolicyRequest = &pb.UpdatePolicyRequest{
					Id: gofakeit.LetterN(10),
					Policy: &pb.PolicyEntity{
						Name: "random name",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"name"},
					},
				}
				updatePolicyResponse, err = server.UpdatePolicy(context.Background(), updatePolicyRequest)
			})
			It("should throw an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("updating the rego content of a policy", func() {
		var (
			createPolicyRequest  *pb.PolicyEntity
			createPolicyResponse *pb.Policy
			updatePolicyRequest  *pb.UpdatePolicyRequest
			updatePolicyResponse *pb.Policy
			err                  error
		)

		BeforeEach(func() {
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
				},
			}

			createPolicyRequest = createRandomPolicyEntity(goodPolicy)
			createPolicyResponse, _ = server.CreatePolicy(context.Background(), createPolicyRequest)
			esTransport.preparedHttpResponses = []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       createEsSearchResponseForPolicy([]*pb.Policy{createPolicyResponse}),
				},
				{
					StatusCode: http.StatusOK,
				},
			}

		})
		When("the policy does not compile", func() {
			BeforeEach(func() {
				updatePolicyRequest = &pb.UpdatePolicyRequest{
					Id: createPolicyResponse.Id,
					Policy: &pb.PolicyEntity{
						RegoContent: uncompilablePolicy,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"rego_content"},
					},
				}
				updatePolicyResponse, err = server.UpdatePolicy(context.Background(), updatePolicyRequest)
			})
			It("should throw an error ", func() {
				Expect(err).To(HaveOccurred())
				Expect(updatePolicyResponse).To(BeNil())
			})
		})

	})
})

func createRandomOccurrence(kind grafeas_common_proto.NoteKind) *grafeas_proto.Occurrence {
	return &grafeas_proto.Occurrence{
		Name: gofakeit.LetterN(10),
		Resource: &grafeas_proto.Resource{
			Uri: fmt.Sprintf("%s@sha256:%s", gofakeit.URL(), gofakeit.LetterN(10)),
		},
		NoteName:    gofakeit.LetterN(10),
		Kind:        kind,
		Remediation: gofakeit.LetterN(10),
		CreateTime:  timestamppb.New(gofakeit.Date()),
		UpdateTime:  timestamppb.New(gofakeit.Date()),
		Details:     nil,
	}
}

func createEsIndexResponse(index string) *esutil.EsIndexResponse {
	return &esutil.EsIndexResponse{
		Acknowledged:       true,
		ShardsAcknowledged: true,
		Index:              index,
	}
}

func createEsDeleteDocResponse() *esutil.EsDeleteResponse {
	return &esutil.EsDeleteResponse{
		Took:                 int(gofakeit.Int16()),
		TimedOut:             false,
		Total:                int(gofakeit.Int16()),
		Deleted:              int(gofakeit.Int16()),
		Batches:              int(gofakeit.Int16()),
		VersionConflicts:     int(gofakeit.Int16()),
		Noops:                int(gofakeit.Int16()),
		ThrottledMillis:      int(gofakeit.Int16()),
		RequestsPerSecond:    gofakeit.Float64(),
		ThrottledUntilMillis: int(gofakeit.Int16()),
		Failures:             nil,
	}
}

func structToJsonBody(i interface{}) io.ReadCloser {
	b, err := json.Marshal(i)
	Expect(err).ToNot(HaveOccurred())

	return io.NopCloser(strings.NewReader(string(b)))
}

func createEsSearchResponse(occurrences []*grafeas_proto.Occurrence) io.ReadCloser {
	return createPaginatedEsSearchResponse(occurrences, len(occurrences))
}

func createPaginatedEsSearchResponse(occurrences []*grafeas_proto.Occurrence, totalResults int) io.ReadCloser {
	var occurrenceHits []*esutil.EsSearchResponseHit

	for _, occurrence := range occurrences {
		source, err := protojson.Marshal(proto.MessageV2(occurrence))
		Expect(err).To(BeNil())

		response := &esutil.EsSearchResponseHit{
			ID:     gofakeit.UUID(),
			Source: source,
		}

		occurrenceHits = append(occurrenceHits, response)
	}

	response := &esutil.EsSearchResponse{
		Hits: &esutil.EsSearchResponseHits{
			Total: &esutil.EsSearchResponseTotal{
				Value: totalResults,
			},
			Hits: occurrenceHits,
		},
		Took: gofakeit.Number(1, 10),
	}

	responseBody, err := json.Marshal(response)
	Expect(err).To(BeNil())

	return io.NopCloser(bytes.NewReader(responseBody))
}

func createEsSearchResponseForGenericResource(resources []*pb.GenericResource) io.ReadCloser {
	return createPaginatedEsSearchResponseForGenericResource(resources, len(resources))
}

func createPaginatedEsSearchResponseForGenericResource(resources []*pb.GenericResource, totalValue int) io.ReadCloser {
	var hits []*esutil.EsSearchResponseHit

	for _, resource := range resources {
		source, err := protojson.Marshal(proto.MessageV2(resource))
		Expect(err).To(BeNil())

		response := &esutil.EsSearchResponseHit{
			ID:     resource.Name,
			Source: source,
		}

		hits = append(hits, response)
	}

	response := &esutil.EsSearchResponse{
		Hits: &esutil.EsSearchResponseHits{
			Hits: hits,
			Total: &esutil.EsSearchResponseTotal{
				Value: totalValue,
			},
		},
		Took: gofakeit.Number(1, 10),
	}

	responseBody, err := json.Marshal(response)
	Expect(err).To(BeNil())

	return io.NopCloser(bytes.NewReader(responseBody))
}

func createEsSearchResponseForPolicy(policies []*pb.Policy) io.ReadCloser {
	return createPaginatedEsSearchResponseForPolicy(policies, len(policies))
}

func createPaginatedEsSearchResponseForPolicy(policies []*pb.Policy, totalValue int) io.ReadCloser {
	var policyHits []*esutil.EsSearchResponseHit

	for _, occurrence := range policies {
		source, err := protojson.Marshal(proto.MessageV2(occurrence))
		Expect(err).To(BeNil())

		response := &esutil.EsSearchResponseHit{
			ID:     gofakeit.UUID(),
			Source: source,
		}

		policyHits = append(policyHits, response)
	}

	response := &esutil.EsSearchResponse{
		Hits: &esutil.EsSearchResponseHits{
			Total: &esutil.EsSearchResponseTotal{
				Value: totalValue,
			},
			Hits: policyHits,
		},
		Took: gofakeit.Number(1, 10),
	}

	responseBody, err := json.Marshal(response)
	Expect(err).To(BeNil())

	return io.NopCloser(bytes.NewReader(responseBody))
}

func readEsSearchResponse(request *http.Request) *esutil.EsSearch {
	search := &esutil.EsSearch{}
	readResponseBody(request, search)

	return search
}

func readResponseBody(request *http.Request, v interface{}) {
	requestBody, err := io.ReadAll(request.Body)
	Expect(err).To(BeNil())

	err = json.Unmarshal(requestBody, v)
	Expect(err).To(BeNil())
}

func createInvalidResponseBody() io.ReadCloser {
	return io.NopCloser(strings.NewReader("{"))
}

func getGRPCStatusFromError(err error) *status.Status {
	s, ok := status.FromError(err)
	Expect(ok).To(BeTrue(), "Expected error to be a gRPC status")

	return s
}

func createRandomPolicyEntity(policy string) *pb.PolicyEntity {
	return &pb.PolicyEntity{
		Name:        gofakeit.LetterN(10),
		Description: gofakeit.LetterN(50),
		RegoContent: policy,
		SourcePath:  gofakeit.URL(),
	}
}

func deepCopyNote(note *grafeas_proto.Note) *grafeas_proto.Note {
	return &grafeas_proto.Note{
		Name:             note.Name,
		ShortDescription: note.ShortDescription,
		LongDescription:  note.LongDescription,
		Kind:             note.Kind,
	}
}
