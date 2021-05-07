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
		rodeServer            pb.RodeServer
		grafeasClient         *mocks.FakeGrafeasV1Beta1Client
		grafeasProjectsClient *mocks.FakeProjectsClient
		opaClient             *mocks.MockOpaClient
		esClient              *elasticsearch.Client
		esTransport           *mockEsTransport
		mockFilterer          *mocks.MockFilterer
		mockCtrl              *gomock.Controller
		elasticsearchConfig   *config.ElasticsearchConfig
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		grafeasClient = &mocks.FakeGrafeasV1Beta1Client{}
		grafeasProjectsClient = &mocks.FakeProjectsClient{}
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

		rodeServer, _ = NewRodeServer(logger, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer, elasticsearchConfig)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("initialize", func() {
		var (
			actualRodeServer pb.RodeServer
			actualErr        error

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

			actualRodeServer, actualErr = NewRodeServer(logger, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer, elasticsearchConfig)
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
			Expect(actualErr).ToNot(HaveOccurred())
		})

		When("getting the rode project fails", func() {
			BeforeEach(func() {
				expectedGetProjectError = status.Error(codes.Internal, "getting project failed")
			})

			It("should return an error", func() {
				Expect(actualRodeServer).To(BeNil())
				Expect(actualErr).To(HaveOccurred())
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
					Expect(actualErr).To(HaveOccurred())
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
				Expect(actualErr).To(HaveOccurred())
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
				Expect(actualErr).To(HaveOccurred())
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
				Expect(actualErr).To(HaveOccurred())
			})
		})

		When("creating the second index errors", func() {
			BeforeEach(func() {
				esTransport.actions = []func(req *http.Request) (*http.Response, error){
					nil,
					func(req *http.Request) (*http.Response, error) {
						return nil, errors.New(gofakeit.Word())
					},
				}
			})

			It("should return an error", func() {
				Expect(actualRodeServer).To(BeNil())
				Expect(actualErr).To(HaveOccurred())
			})
		})

		When("the first index already exists", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses[0].StatusCode = http.StatusBadRequest
			})

			It("should return the initialized rode server", func() {
				Expect(actualRodeServer).ToNot(BeNil())
				Expect(actualErr).ToNot(HaveOccurred())
			})
		})

		When("the second index already exists", func() {
			BeforeEach(func() {
				esTransport.preparedHttpResponses[1].StatusCode = http.StatusBadRequest
			})

			It("should return the initialized rode server", func() {
				Expect(actualRodeServer).ToNot(BeNil())
				Expect(actualErr).ToNot(HaveOccurred())
			})
		})
	})

	Context("server has been initialized", func() {
		BeforeEach(func() {
			grafeasProjectsClient.
				EXPECT().
				GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest)).
				Return(&grafeas_project_proto.Project{}, nil)

			rodeServer, rodeServerError = NewRodeServer(logger, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer, elasticsearchConfig)
		})

		When("occurrences are created", func() {
			var (
				randomOccurrence                      *grafeas_proto.Occurrence
				grafeasBatchCreateOccurrencesRequest  *grafeas_proto.BatchCreateOccurrencesRequest
				grafeasBatchCreateOccurrencesResponse *grafeas_proto.BatchCreateOccurrencesResponse
			)

			BeforeEach(func() {
				randomOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
				mgetResponse := esutil.EsMultiGetResponse{Docs: []*esutil.EsMultiGetDocument{
					{
						Found: true,
					},
				}}
				esTransport.preparedHttpResponses = append(esTransport.preparedHttpResponses, &http.Response{
					StatusCode: http.StatusOK,
					Body:       structToJsonBody(mgetResponse),
				})

				// expected Grafeas BatchCreateOccurrences request
				grafeasBatchCreateOccurrencesRequest = &grafeas_proto.BatchCreateOccurrencesRequest{
					Parent: "projects/rode",
					Occurrences: []*grafeas_proto.Occurrence{
						randomOccurrence,
					},
				}

				// mocked Grafeas BatchCreateOccurrences response
				grafeasBatchCreateOccurrencesResponse = &grafeas_proto.BatchCreateOccurrencesResponse{
					Occurrences: []*grafeas_proto.Occurrence{
						randomOccurrence,
					},
				}

			})

			It("should send occurrences to Grafeas", func() {
				// ensure Grafeas BatchCreateOccurrences is called with expected request and inject response
				grafeasClient.EXPECT().BatchCreateOccurrences(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(grafeasBatchCreateOccurrencesRequest)).Return(grafeasBatchCreateOccurrencesResponse, nil)

				batchCreateOccurrencesRequest := &pb.BatchCreateOccurrencesRequest{
					Occurrences: []*grafeas_proto.Occurrence{
						randomOccurrence,
					},
				}
				response, err := rodeServer.BatchCreateOccurrences(context.Background(), batchCreateOccurrencesRequest)
				Expect(err).ToNot(HaveOccurred())

				// check response
				Expect(response.Occurrences).To(BeEquivalentTo(grafeasBatchCreateOccurrencesResponse.Occurrences))
			})

			When("an error occurs creating occurrences", func() {
				BeforeEach(func() {
					grafeasClient.
						EXPECT().
						BatchCreateOccurrences(gomock.Any(), gomock.Any()).
						Return(nil, errors.New(gofakeit.Word()))

				})

				It("should return an error", func() {
					batchCreateOccurrencesRequest := &pb.BatchCreateOccurrencesRequest{
						Occurrences: []*grafeas_proto.Occurrence{
							randomOccurrence,
						},
					}
					_, err := rodeServer.BatchCreateOccurrences(context.Background(), batchCreateOccurrencesRequest)

					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("creating generic resources", func() {
			var (
				actualError          error
				expectedResourceName string
				occurrence           *grafeas_go_proto.Occurrence
				request              *pb.BatchCreateOccurrencesRequest
			)

			BeforeEach(func() {
				occurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_BUILD)
				expectedResourceName = gofakeit.URL()
				occurrence.Resource.Uri = fmt.Sprintf("%s@sha256:%s", expectedResourceName, gofakeit.LetterN(10))
				esTransport.preparedHttpResponses = []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: structToJsonBody(&esutil.EsMultiGetResponse{
							Docs: []*esutil.EsMultiGetDocument{{Found: false}},
						}),
					},
					{
						StatusCode: http.StatusOK,
						Body: structToJsonBody(&esutil.EsBulkResponse{
							Items: []*esutil.EsBulkResponseItem{
								{
									Create: &esutil.EsIndexDocResponse{
										Id:     expectedResourceName,
										Status: http.StatusOK,
									},
								},
							},
						}),
					},
				}
				request = &pb.BatchCreateOccurrencesRequest{
					Occurrences: []*grafeas_go_proto.Occurrence{occurrence},
				}

				grafeasClient.EXPECT().BatchCreateOccurrences(gomock.Any(), gomock.Any()).AnyTimes()
			})

			JustBeforeEach(func() {
				_, actualError = rodeServer.BatchCreateOccurrences(context.Background(), request)
			})

			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			When("the generic resources do not exist", func() {
				It("should check if the resources already exist", func() {
					Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodGet))
					Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal(fmt.Sprintf("/%s/_mget", rodeElasticsearchGenericResourcesIndex)))

					requestBody := &esutil.EsMultiGetRequest{}
					readResponseBody(esTransport.receivedHttpRequests[2], &requestBody)
					Expect(requestBody.IDs).To(ConsistOf(expectedResourceName))
				})

				It("should make a bulk request to create all of the resources", func() {
					Expect(esTransport.receivedHttpRequests[3].Method).To(Equal(http.MethodPost))
					Expect(esTransport.receivedHttpRequests[3].URL.Path).To(Equal(fmt.Sprintf("/%s/_bulk", rodeElasticsearchGenericResourcesIndex)))
					Expect(esTransport.receivedHttpRequests[3].URL.Query().Get("refresh")).To(Equal("true"))
				})

				It("should send the API action and document as part of the bulk request", func() {
					body, err := io.ReadAll(esTransport.receivedHttpRequests[3].Body)
					Expect(err).To(BeNil())

					metadata := &esutil.EsBulkQueryFragment{}
					resource := &pb.GenericResource{}
					pieces := bytes.Split(body, []byte{'\n'})

					Expect(json.Unmarshal(pieces[0], metadata)).To(BeNil())
					Expect(json.Unmarshal(pieces[1], resource)).To(BeNil())

					Expect(metadata).To(Equal(&esutil.EsBulkQueryFragment{
						Create: &esutil.EsBulkQueryCreateFragment{
							Id: expectedResourceName,
						},
					}))
					Expect(resource).To(Equal(&pb.GenericResource{
						Name: expectedResourceName,
					}))
				})

			})

			When("the same resource appears multiple times", func() {
				BeforeEach(func() {
					otherOccurrence := createRandomOccurrence(grafeas_common_proto.NoteKind_BUILD)
					otherOccurrence.Resource.Uri = occurrence.Resource.Uri

					request.Occurrences = append(request.Occurrences, otherOccurrence)
				})

				It("should only try to make a single resource", func() {
					requestBody := &esutil.EsMultiGetRequest{}
					readResponseBody(esTransport.receivedHttpRequests[2], &requestBody)
					Expect(requestBody.IDs).To(HaveLen(1))
				})
			})

			When("an error occurs determining the resource uri version", func() {
				BeforeEach(func() {
					occurrence.Resource.Uri = gofakeit.URL()
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("the generic resources already exist", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[0] = &http.Response{
						StatusCode: http.StatusOK,
						Body: structToJsonBody(&esutil.EsMultiGetResponse{
							Docs: []*esutil.EsMultiGetDocument{{Found: true}},
						}),
					}
				})

				It("should not make any further requests to Elasticsearch", func() {
					Expect(esTransport.receivedHttpRequests).To(HaveLen(3))
				})
			})

			When("an unexpected status code is returned from the multi-get", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[0] = &http.Response{
						StatusCode: http.StatusInternalServerError,
					}
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("an error occurs during the multi-get request", func() {
				BeforeEach(func() {
					esTransport.actions = []func(req *http.Request) (*http.Response, error){
						func(req *http.Request) (*http.Response, error) {
							return nil, errors.New(gofakeit.Word())
						},
					}
				})

				It("should return the error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("an the response from the multi-get fails to parse", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[0] = &http.Response{
						StatusCode: http.StatusOK,
						Body:       createInvalidResponseBody(),
					}
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("an unexpected status code is returned from the bulk request", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[1] = &http.Response{
						StatusCode: http.StatusInternalServerError,
					}
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("an error occurs during the bulk request", func() {
				BeforeEach(func() {
					esTransport.actions = []func(req *http.Request) (*http.Response, error){
						func(req *http.Request) (*http.Response, error) {
							return esTransport.preparedHttpResponses[0], nil
						},
						func(req *http.Request) (*http.Response, error) {
							return nil, errors.New(gofakeit.Word())
						},
					}
				})

				It("should return the error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("the response from the bulk request fails to parse", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[1] = &http.Response{
						StatusCode: http.StatusOK,
						Body:       createInvalidResponseBody(),
					}
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("one of the bulk requests fails", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[1] = &http.Response{
						StatusCode: http.StatusOK,
						Body: structToJsonBody(&esutil.EsBulkResponse{
							Items: []*esutil.EsBulkResponseItem{
								{
									Create: &esutil.EsIndexDocResponse{
										Error: &esutil.EsIndexDocError{
											Reason: gofakeit.Word(),
										},
										Status: http.StatusInternalServerError,
									},
								},
							},
						}),
					}
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})

			When("one of the bulk request items tries to create an already existing resource", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[1] = &http.Response{
						StatusCode: http.StatusOK,
						Body: structToJsonBody(&esutil.EsBulkResponse{
							Items: []*esutil.EsBulkResponseItem{
								{
									Create: &esutil.EsIndexDocResponse{
										Error: &esutil.EsIndexDocError{
											Reason: gofakeit.Word(),
										},
										Status: http.StatusConflict,
									},
								},
							},
						}),
					}
				})

				It("should not return an error", func() {
					Expect(actualError).NotTo(HaveOccurred())
				})
			})
		})

		Describe("ListGenericResources", func() {
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
					actualResponse, actualError = rodeServer.ListGenericResources(context.Background(), listRequest)
				})

				It("should not return an error", func() {
					Expect(actualError).NotTo(HaveOccurred())
				})

				It("should search against the generic resources index", func() {
					Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodGet))
					Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal(fmt.Sprintf("/%s/_search", rodeElasticsearchGenericResourcesIndex)))
					Expect(esTransport.receivedHttpRequests[2].URL.Query().Get("size")).To(Equal(strconv.Itoa(maxPageSize)))

					body := readEsSearchResponse(esTransport.receivedHttpRequests[2])

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
							body := readEsSearchResponse(esTransport.receivedHttpRequests[2])

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
							Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal(fmt.Sprintf("/%s/_pit", rodeElasticsearchGenericResourcesIndex)))
							Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodPost))
							Expect(esTransport.receivedHttpRequests[2].URL.Query().Get("keep_alive")).To(Equal("5m"))
						})

						It("should query using the PIT", func() {
							Expect(esTransport.receivedHttpRequests[3].URL.Path).To(Equal("/_search"))
							Expect(esTransport.receivedHttpRequests[3].Method).To(Equal(http.MethodGet))
							request := readEsSearchResponse(esTransport.receivedHttpRequests[3])
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
							Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal("/_search"))
							Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodGet))
							request := readEsSearchResponse(esTransport.receivedHttpRequests[2])
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
							Expect(esTransport.receivedHttpRequests).To(HaveLen(2))
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
							Expect(esTransport.receivedHttpRequests).To(HaveLen(2))
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

		Describe("ListVersionedResourceOccurrences", func() {
			var (
				buildOccurrencesRequest  *grafeas_proto.ListOccurrencesRequest
				buildOccurrencesResponse *grafeas_proto.ListOccurrencesResponse

				allOccurrencesRequest  *grafeas_proto.ListOccurrencesRequest
				allOccurrencesResponse *grafeas_proto.ListOccurrencesResponse

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

				buildOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
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

				allOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
					Occurrences: []*grafeas_proto.Occurrence{
						createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
						createRandomOccurrence(grafeas_common_proto.NoteKind_BUILD),
					},
					NextPageToken: nextPageToken,
				}
			})

			JustBeforeEach(func() {
				actualResponse, actualError = rodeServer.ListVersionedResourceOccurrences(ctx, request)
			})

			AfterEach(func() {
				buildOccurrencesRequest = nil
				allOccurrencesRequest = nil
			})

			Describe("successful calls to Grafeas", func() {
				BeforeEach(func() {
					grafeasClient.EXPECT().
						ListOccurrences(ctx, gomock.Any()).
						DoAndReturn(func(_ context.Context, r *grafeas_proto.ListOccurrencesRequest) (*grafeas_proto.ListOccurrencesResponse, error) {
							if buildOccurrencesRequest == nil {
								buildOccurrencesRequest = r

								return buildOccurrencesResponse, nil
							}

							if allOccurrencesRequest == nil {
								allOccurrencesRequest = r
								return allOccurrencesResponse, nil
							}

							return nil, nil
						}).Times(2)
				})

				When("there are build occurrences related to the resource uri", func() {
					It("should list build occurrences for the resource uri", func() {
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

						Expect(allOccurrencesRequest).NotTo(BeNil())
						Expect(allOccurrencesRequest.Parent).To(Equal("projects/rode"))
						Expect(allOccurrencesRequest.PageSize).To(Equal(pageSize))
						Expect(allOccurrencesRequest.PageToken).To(Equal(currentPageToken))

						filterParts := strings.Split(allOccurrencesRequest.Filter, " || ")
						Expect(filterParts).To(ConsistOf(expectedFilter))
					})
				})

				When("there are no build occurrences", func() {
					BeforeEach(func() {
						buildOccurrencesResponse.Occurrences = []*grafeas_proto.Occurrence{}
					})

					It("should list occurrences for the resource uri", func() {
						Expect(allOccurrencesRequest.Filter).To(Equal(fmt.Sprintf(`resource.uri == "%s"`, resourceUri)))
					})
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
					grafeasClient.EXPECT().
						ListOccurrences(gomock.Any(), gomock.Any()).
						Return(nil, errors.New(gofakeit.Word())).
						Times(1)
				})

				It("should return an error", func() {
					Expect(actualResponse).To(BeNil())
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				})
			})

			When("an error occurs listing all occurrences", func() {
				BeforeEach(func() {
					grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(buildOccurrencesResponse, nil)
					grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(nil, errors.New(gofakeit.Word()))
				})

				It("should return an error", func() {
					Expect(actualResponse).To(BeNil())
					Expect(actualError).To(HaveOccurred())
					Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
				})
			})
		})

		Describe("ListOccurrences", func() {
			var (
				randomOccurrence               *grafeas_proto.Occurrence
				nextPageToken                  string
				currentPageToken               string
				pageSize                       int32
				grafeasListOccurrencesRequest  *grafeas_proto.ListOccurrencesRequest
				grafeasListOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
				uri                            string
			)

			BeforeEach(func() {
				randomOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)

				uri = randomOccurrence.Resource.Uri
				nextPageToken = gofakeit.Word()
				currentPageToken = gofakeit.Word()
				pageSize = gofakeit.Int32()

				// expected Grafeas ListOccurrencesRequest request
				grafeasListOccurrencesRequest = &grafeas_proto.ListOccurrencesRequest{
					Parent:    "projects/rode",
					Filter:    fmt.Sprintf(`"resource.uri" == "%s"`, uri),
					PageToken: currentPageToken,
					PageSize:  pageSize,
				}

				// mocked Grafeas ListOccurrencesResponse response
				grafeasListOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
					Occurrences: []*grafeas_proto.Occurrence{
						randomOccurrence,
					},
					NextPageToken: nextPageToken,
				}

			})

			It("should list occurrences from grafeas", func() {
				// ensure Grafeas ListOccurrences is called with expected request and inject response
				grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(grafeasListOccurrencesRequest)).Return(grafeasListOccurrencesResponse, nil)

				listOccurrencesRequest := &pb.ListOccurrencesRequest{
					Filter:    fmt.Sprintf(`"resource.uri" == "%s"`, uri),
					PageToken: currentPageToken,
					PageSize:  pageSize,
				}
				response, err := rodeServer.ListOccurrences(context.Background(), listOccurrencesRequest)
				Expect(err).ToNot(HaveOccurred())

				// check response
				Expect(response.Occurrences).To(BeEquivalentTo(grafeasListOccurrencesResponse.Occurrences))
				Expect(response.NextPageToken).To(Equal(nextPageToken))
			})

			When("Grafeas returns an error", func() {
				It("should return an error", func() {
					grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(grafeasListOccurrencesRequest)).Return(nil, fmt.Errorf("error occurred"))

					listOccurrencesRequest := &pb.ListOccurrencesRequest{
						Filter:    fmt.Sprintf(`"resource.uri" == "%s"`, uri),
						PageToken: currentPageToken,
						PageSize:  pageSize,
					}
					response, err := rodeServer.ListOccurrences(context.Background(), listOccurrencesRequest)
					Expect(err).ToNot(BeNil())

					// check response
					Expect(response).To(BeNil())
				})
			})
		})

		Describe("UpdateOccurrence", func() {
			var (
				actualError             error
				response                *grafeas_go_proto.Occurrence
				randomOccurrence        *grafeas_proto.Occurrence
				updateOccurrenceRequest *pb.UpdateOccurrenceRequest
				expectedResponse        *grafeas_proto.Occurrence
				grafeasUpdateRequest    *grafeas_proto.UpdateOccurrenceRequest
			)

			BeforeEach(func() {
				randomOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
				occurrenceId := gofakeit.UUID()
				occurrenceName := fmt.Sprintf("projects/rode/occurrences/%s", occurrenceId)
				randomOccurrence.Name = occurrenceName
				updateOccurrenceRequest = &pb.UpdateOccurrenceRequest{
					Id:         occurrenceId,
					Occurrence: randomOccurrence,
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{gofakeit.Word()},
					},
				}

				grafeasUpdateRequest = &grafeas_go_proto.UpdateOccurrenceRequest{
					Name:       occurrenceName,
					Occurrence: randomOccurrence,
					UpdateMask: updateOccurrenceRequest.UpdateMask,
				}

				expectedResponse = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)
			})

			JustBeforeEach(func() {
				response, actualError = rodeServer.UpdateOccurrence(context.Background(), updateOccurrenceRequest)
			})

			When("the occurrence is successfully updated", func() {
				BeforeEach(func() {
					grafeasClient.EXPECT().UpdateOccurrence(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(grafeasUpdateRequest)).Return(expectedResponse, nil)
				})

				It("should return the updated occurrence", func() {
					Expect(actualError).ToNot(HaveOccurred())
					Expect(response).To(Equal(expectedResponse))
				})
			})

			When("Grafeas returns an error", func() {
				BeforeEach(func() {
					grafeasClient.EXPECT().UpdateOccurrence(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(grafeasUpdateRequest)).Return(nil, fmt.Errorf("error occurred"))
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
					Expect(response).To(BeNil())
				})
			})

			When("the occurrence name doesn't contain the occurrence id", func() {
				BeforeEach(func() {
					updateOccurrenceRequest.Id = gofakeit.UUID()
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
			})
		})

		When("policy is evaluated", func() {
			var (
				policy                    string
				resourceURI               string
				listOccurrencesRequest    *grafeas_proto.ListOccurrencesRequest
				evaluatePolicyRequest     *pb.EvaluatePolicyRequest
				opaEvaluatePolicyResponse *opa.EvaluatePolicyResponse
			)

			BeforeEach(func() {
				resourceURI = gofakeit.URL()
				policy = goodPolicy
				occurrences := []*grafeas_proto.Occurrence{
					createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
					createRandomOccurrence(grafeas_common_proto.NoteKind_ATTESTATION),
				}
				listOccurrencesRequest = &grafeas_proto.ListOccurrencesRequest{
					Parent:   "projects/rode",
					PageSize: maxPageSize,
					Filter:   fmt.Sprintf(`"resource.uri" == "%s"`, resourceURI),
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
				createPolicyResponse, _ := rodeServer.CreatePolicy(context.Background(), createPolicyRequest)
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
			})

			It("should initialize OPA policy", func() {
				// ignore non test calls
				grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).AnyTimes()
				opaClient.EXPECT().EvaluatePolicy(gomock.Any(), gomock.Any()).AnyTimes().Return(opaEvaluatePolicyResponse, nil)

				// expect OPA initialize policy call
				opaClient.EXPECT().InitializePolicy(policy, goodPolicy).Return(nil)

				_, _ = rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
			})

			It("should return an error if resource uri is not specified", func() {
				grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Times(0)
				opaClient.EXPECT().EvaluatePolicy(gomock.Any(), gomock.Any()).Times(0)
				opaClient.EXPECT().InitializePolicy(policy, goodPolicy).Times(0)

				evaluatePolicyRequest.ResourceUri = ""
				_, err := rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeEquivalentTo(status.Errorf(codes.InvalidArgument, "resource uri is required")))
			})

			When("OPA policy initializes", func() {

				BeforeEach(func() {
					opaClient.EXPECT().InitializePolicy(gomock.Any(), goodPolicy).AnyTimes().Return(nil)
				})

				It("should list Grafeas occurrences", func() {
					// ingore non test calls
					opaClient.EXPECT().EvaluatePolicy(gomock.Any(), gomock.Any()).AnyTimes().Return(opaEvaluatePolicyResponse, nil)

					// expect Grafeas list occurrences call
					grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), listOccurrencesRequest)

					_, _ = rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
				})

				When("Grafeas list occurrences response is ok", func() {

					It("should evaluate OPA policy", func() {
						// mock Grafeas list occurrences response
						listOccurrencesResponse := &grafeas_proto.ListOccurrencesResponse{
							Occurrences: []*grafeas_proto.Occurrence{
								createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED),
							},
						}
						grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(listOccurrencesResponse, nil)

						opaClient.EXPECT().EvaluatePolicy(gomock.Eq(policy), gomock.Any()).Return(opaEvaluatePolicyResponse, nil)

						_, _ = rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
					})

					When("evalute OPA policy returns error", func() {
						It("should return error", func() {
							grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(&grafeas_proto.ListOccurrencesResponse{Occurrences: []*grafeas_proto.Occurrence{}}, nil)

							opaClient.EXPECT().EvaluatePolicy(gomock.Eq(policy), gomock.Any()).Return(nil, fmt.Errorf("OPA Error"))

							_, evaluatePolicyError := rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
							Expect(evaluatePolicyError).To(HaveOccurred())
							Expect(evaluatePolicyError.Error()).To(ContainSubstring("error evaluating policy"))
						})
					})
				})

				When("Grafeas list occurrences response is error", func() {
					It("should return an error", func() {
						// mock Grafeas list occurrences error response
						grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("elasticsearch error"))

						_, evaluatePolicyError := rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
						Expect(evaluatePolicyError).To(HaveOccurred())
					})
				})
			})

			When("OPA policy is not found", func() {
				It("should return an error", func() {
					opaClient.EXPECT().InitializePolicy(gomock.Any(), goodPolicy).Return(opa.NewClientError("policy not found", opa.OpaClientErrorTypePolicyNotFound, fmt.Errorf("es search result empty")))

					_, evaluatePolicyError := rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
					Expect(evaluatePolicyError).To(HaveOccurred())
				})
			})
		})

		When("listing resources", func() {
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

					listResourcesResponse, listResourcesResponseErr = rodeServer.ListResources(context.Background(), request)
				})

				It("should query the Rode occurrences index", func() {
					actualRequest := esTransport.receivedHttpRequests[2]

					Expect(actualRequest.URL.Path).To(Equal("/grafeas-rode-occurrences/_search"))
				})

				It("should take the first 1000 matches", func() {
					actualRequest := esTransport.receivedHttpRequests[2]
					query := actualRequest.URL.Query()

					Expect(query.Get("size")).To(Equal("1000"))
				})

				It("should collapse fields on resource.uri", func() {
					actualRequest := esTransport.receivedHttpRequests[2]
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

					_, _ = rodeServer.ListResources(context.Background(), request)
				})

				It("should include the Elasticsearch query in the response", func() {
					expectedQuery := &filtering.Query{
						Term: &filtering.Term{
							gofakeit.UUID(): gofakeit.UUID(),
						},
					}
					mockFilterer.EXPECT().ParseExpression(gomock.Any()).Return(expectedQuery, nil)

					_, err := rodeServer.ListResources(context.Background(), request)
					Expect(err).To(BeNil())

					actualRequest := esTransport.receivedHttpRequests[2]
					search := readEsSearchResponse(actualRequest)

					Expect(search.Query).To(Equal(expectedQuery))
				})
			})

			When("elasticsearch returns with an error", func() {
				BeforeEach(func() {
					esTransport.preparedHttpResponses[0] = &http.Response{
						StatusCode: http.StatusInternalServerError,
					}

					listResourcesResponse, listResourcesResponseErr = rodeServer.ListResources(context.Background(), request)
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
					actualResponse, actualError = rodeServer.ListResources(context.Background(), request)
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
						Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal(fmt.Sprintf("/%s/_pit", rodeElasticsearchOccurrencesAlias)))
						Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodPost))
						Expect(esTransport.receivedHttpRequests[2].URL.Query().Get("keep_alive")).To(Equal("5m"))
					})

					It("should query using the PIT", func() {
						Expect(esTransport.receivedHttpRequests[3].URL.Path).To(Equal("/_search"))
						Expect(esTransport.receivedHttpRequests[3].Method).To(Equal(http.MethodGet))
						request := readEsSearchResponse(esTransport.receivedHttpRequests[3])
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
						Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal("/_search"))
						Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodGet))
						request := readEsSearchResponse(esTransport.receivedHttpRequests[2])
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
						Expect(esTransport.receivedHttpRequests).To(HaveLen(2))
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
						Expect(esTransport.receivedHttpRequests).To(HaveLen(2))
					})

					It("should return an error", func() {
						Expect(actualError).To(HaveOccurred())
						Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
						Expect(actualResponse).To(BeNil())
					})
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

				policyResponse, err = rodeServer.CreatePolicy(context.Background(), policyEntity)
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
				policyResponse, err = rodeServer.CreatePolicy(context.Background(), policyEntity)
			})

			It("should have a correct url path", func() {
				Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal("/rode-v1alpha1-policies/_doc"))
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
					getResponse, err = rodeServer.GetPolicy(context.Background(), &pb.GetPolicyRequest{Id: policyResponse.Id})
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
					deleteResponse, err = rodeServer.DeletePolicy(context.Background(), &pb.DeletePolicyRequest{Id: policyResponse.Id})
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
				policyResponseOne, err = rodeServer.CreatePolicy(context.Background(), policyEntityOne)
				policyResponseTwo, err = rodeServer.CreatePolicy(context.Background(), policyEntityTwo)
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
					listResponse, err = rodeServer.ListPolicies(context.Background(), listRequest)
				})
				It("should not return an error", func() {
					Expect(err).To(Not(HaveOccurred()))
				})
				It("should have listed 4 different policies", func() {
					Expect(listResponse.Policies).To(HaveLen(4))
				})
				It("should have generated a filter query", func() {
					actualRequest := esTransport.receivedHttpRequests[2]
					search := readEsSearchResponse(actualRequest)

					Expect(search.Query).To(Equal(expectedQuery))
				})
				It("should have generated a filter query", func() {
					actualRequest := esTransport.receivedHttpRequests[2]
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
				listResponse, actualError = rodeServer.ListPolicies(context.Background(), listRequest)
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
					Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal(fmt.Sprintf("/%s/_pit", rodeElasticsearchPoliciesIndex)))
					Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodPost))
					Expect(esTransport.receivedHttpRequests[2].URL.Query().Get("keep_alive")).To(Equal("5m"))
				})

				It("should query using the PIT", func() {
					Expect(esTransport.receivedHttpRequests[3].URL.Path).To(Equal("/_search"))
					Expect(esTransport.receivedHttpRequests[3].Method).To(Equal(http.MethodGet))
					request := readEsSearchResponse(esTransport.receivedHttpRequests[3])
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
					Expect(esTransport.receivedHttpRequests[2].URL.Path).To(Equal("/_search"))
					Expect(esTransport.receivedHttpRequests[2].Method).To(Equal(http.MethodGet))
					request := readEsSearchResponse(esTransport.receivedHttpRequests[2])
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
					Expect(esTransport.receivedHttpRequests).To(HaveLen(2))
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
					Expect(esTransport.receivedHttpRequests).To(HaveLen(2))
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
				listPoliciesResponse, err = rodeServer.ListPolicies(context.Background(), listRequest)
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
				_, err = rodeServer.ListPolicies(context.Background(), listRequest)
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
				policyResponse, err = rodeServer.CreatePolicy(context.Background(), policyEntity)
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
				policyResponse, err = rodeServer.CreatePolicy(context.Background(), policyEntity)
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
				policyResponse, err = rodeServer.CreatePolicy(context.Background(), policyEntity)
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
				policyResponse, err = rodeServer.CreatePolicy(context.Background(), policyEntity)
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
				validatePolicyResponse, err = rodeServer.ValidatePolicy(context.Background(), validatePolicyRequest)
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
				_, err = rodeServer.ValidatePolicy(context.Background(), validatePolicyRequest)
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
				validatePolicyResponse, err = rodeServer.ValidatePolicy(context.Background(), validatePolicyRequest)
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
				createPolicyResponse, _ = rodeServer.CreatePolicy(context.Background(), createPolicyRequest)
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
				updatePolicyResponse, err = rodeServer.UpdatePolicy(context.Background(), updatePolicyRequest)
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
					updatePolicyResponse, err = rodeServer.UpdatePolicy(context.Background(), updatePolicyRequest)
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
				createPolicyResponse, _ = rodeServer.CreatePolicy(context.Background(), createPolicyRequest)
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
					updatePolicyResponse, err = rodeServer.UpdatePolicy(context.Background(), updatePolicyRequest)
				})
				It("should throw an error ", func() {
					Expect(err).To(HaveOccurred())
					Expect(updatePolicyResponse).To(BeNil())
				})
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
