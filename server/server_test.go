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
	"fmt"
	"strings"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	gomock "github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/mocks"
	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"io"
	"io/ioutil"
	"net/http"
)

var _ = Describe("rode server", func() {
	const (
		createProjectError = "CREATE_PROJECT_ERROR"
		getProjectError    = "GET_PROJECT_ERROR"
		goodPolicy         = `
		package play
		default hello = false
		hello {
			m := input.message
			m == "world"
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
		log                   *zap.Logger
		rodeServer            pb.RodeServer
		rodeServerError       error
		grafeasClient         *mocks.MockGrafeasClient
		grafeasProjectsClient *mocks.MockGrafeasProjectsClient
		opaClient             *mocks.MockOpaClient
		esClient              *elasticsearch.Client
		esTransport           *mockEsTransport
		mockFilterer          *mocks.MockFilterer
		mockCtrl              *gomock.Controller
		getProjectRequest     = &grafeas_project_proto.GetProjectRequest{Name: "projects/rode"}
	)

	BeforeEach(func() {
		log = logger.Named("rode server test")
		mockCtrl = gomock.NewController(GinkgoT())
		grafeasClient = mocks.NewMockGrafeasClient(mockCtrl)
		grafeasProjectsClient = mocks.NewMockGrafeasProjectsClient(mockCtrl)
		opaClient = mocks.NewMockOpaClient(mockCtrl)

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
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("server is initialized", func() {
		It("should check if the rode project exists", func() {
			grafeasProjectsClient.
				EXPECT().
				GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest))

			rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)
		})

		When("the rode project does not exist", func() {
			var (
				createProjectRequest = &grafeas_project_proto.CreateProjectRequest{
					Project: &grafeas_project_proto.Project{Name: "projects/rode"},
				}
			)
			BeforeEach(func() {
				grafeasProjectsClient.
					EXPECT().
					GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest)).
					Return(nil, status.Error(codes.NotFound, "Not found"))
			})

			It("should create the rode project", func() {
				grafeasProjectsClient.
					EXPECT().
					CreateProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(createProjectRequest))

				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)
			})

			When("create project returns error from Grafeas", func() {
				BeforeEach(func() {
					grafeasProjectsClient.
						EXPECT().
						CreateProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(createProjectRequest)).Return(nil, status.Error(codes.Internal, createProjectError))
				})

				It("should returns error", func() {
					rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)

					Expect(rodeServerError).To(HaveOccurred())
					Expect(rodeServerError.Error()).To(ContainSubstring(createProjectError))
				})
			})

			When("create project succeeds", func() {
				BeforeEach(func() {
					grafeasProjectsClient.
						EXPECT().
						CreateProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(createProjectRequest)).Return(&grafeas_project_proto.Project{}, nil)
				})

				It("should return the Rode server", func() {
					rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)

					Expect(rodeServer).ToNot(BeNil())
					Expect(rodeServerError).ToNot(HaveOccurred())
				})
			})
		})

		When("rode project exists", func() {
			BeforeEach(func() {
				grafeasProjectsClient.
					EXPECT().
					GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest)).
					Return(&grafeas_project_proto.Project{}, nil)
			})

			It("should not attempt to create the project", func() {
				grafeasProjectsClient.
					EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).MaxTimes(0)

				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)
			})

			It("should return the Rode server", func() {
				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)

				Expect(rodeServer).ToNot(BeNil())
				Expect(rodeServerError).To(BeNil())
			})
		})

		When("fetching the rode project it returns an error", func() {
			BeforeEach(func() {
				grafeasProjectsClient.
					EXPECT().
					GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest)).
					Return(nil, status.Error(codes.Internal, getProjectError))
			})

			It("should return an error", func() {
				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)

				Expect(rodeServerError).To(HaveOccurred())
				Expect(rodeServerError.Error()).To(ContainSubstring(getProjectError))
			})
		})
	})

	Context("server has been initialized", func() {
		BeforeEach(func() {
			grafeasProjectsClient.
				EXPECT().
				GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest)).
				Return(&grafeas_project_proto.Project{}, nil)

			rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, opaClient, esClient, mockFilterer)
		})

		When("occurrences are created", func() {
			var (
				randomOccurrence                      *grafeas_proto.Occurrence
				grafeasBatchCreateOccurrencesRequest  *grafeas_proto.BatchCreateOccurrencesRequest
				grafeasBatchCreateOccurrencesResponse *grafeas_proto.BatchCreateOccurrencesResponse
			)

			BeforeEach(func() {
				randomOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)

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
		})

		When("listing occurrences for a resource", func() {
			var (
				randomOccurrence               *grafeas_proto.Occurrence
				grafeasListOccurrencesRequest  *grafeas_proto.ListOccurrencesRequest
				grafeasListOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
				uri                            string
			)

			BeforeEach(func() {
				randomOccurrence = createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED)

				uri = randomOccurrence.Resource.Uri

				// expected Grafeas ListOccurrencesRequest request
				grafeasListOccurrencesRequest = &grafeas_proto.ListOccurrencesRequest{
					Parent: "projects/rode",
					Filter: fmt.Sprintf(`"resource.uri" == "%s"`, uri),
				}

				// mocked Grafeas ListOccurrencesResponse response
				grafeasListOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
					Occurrences: []*grafeas_proto.Occurrence{
						randomOccurrence,
					},
				}

			})

			It("should list occurrences from grafeas", func() {
				// ensure Grafeas ListOccurrences is called with expected request and inject response
				grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(grafeasListOccurrencesRequest)).Return(grafeasListOccurrencesResponse, nil)

				listOccurrencesRequest := &pb.ListOccurrencesRequest{
					Filter: fmt.Sprintf(`"resource.uri" == "%s"`, uri),
				}
				response, err := rodeServer.ListOccurrences(context.Background(), listOccurrencesRequest)
				Expect(err).ToNot(HaveOccurred())

				// check response
				Expect(response.Occurrences).To(BeEquivalentTo(grafeasListOccurrencesResponse.Occurrences))
			})

			When("Grafeas returns an error", func() {
				It("should return an error", func() {
					grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(grafeasListOccurrencesRequest)).Return(nil, fmt.Errorf("error occurred"))

					listOccurrencesRequest := &pb.ListOccurrencesRequest{
						Filter: fmt.Sprintf(`"resource.uri" == "%s"`, uri),
					}
					response, err := rodeServer.ListOccurrences(context.Background(), listOccurrencesRequest)
					Expect(err).ToNot(BeNil())

					// check response
					Expect(response).To(BeNil())
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
				policy = gofakeit.Word()
				listOccurrencesRequest = &grafeas_proto.ListOccurrencesRequest{
					Parent: "projects/rode",
					Filter: fmt.Sprintf(`"resource.uri" == "%s"`, resourceURI),
				}
				evaluatePolicyRequest = &pb.EvaluatePolicyRequest{
					ResourceURI: resourceURI,
					Policy:      policy,
				}
				opaEvaluatePolicyResponse = &opa.EvaluatePolicyResponse{
					Result: &opa.EvaluatePolicyResult{
						Pass: false,
					},
					Explanation: &[]string{},
				}
			})

			It("should initialize OPA policy", func() {
				// ignore non test calls
				grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).AnyTimes()
				opaClient.EXPECT().EvaluatePolicy(gomock.Any(), gomock.Any()).AnyTimes().Return(opaEvaluatePolicyResponse, nil)

				// expect OPA initialize policy call
				opaClient.EXPECT().InitializePolicy(policy).Return(nil)

				_, _ = rodeServer.EvaluatePolicy(context.Background(), evaluatePolicyRequest)
			})

			When("OPA policy initializes", func() {

				BeforeEach(func() {
					opaClient.EXPECT().InitializePolicy(gomock.Any()).AnyTimes().Return(nil)
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
							Expect(evaluatePolicyError.Error()).To(ContainSubstring("evaluate OPA policy failed"))
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
					opaClient.EXPECT().InitializePolicy(gomock.Any()).Return(opa.NewClientError("policy not found", opa.OpaClientErrorTypePolicyNotFound, fmt.Errorf("es search result empty")))

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
					actualRequest := esTransport.receivedHttpRequests[1]

					Expect(actualRequest.URL.Path).To(Equal("/grafeas-rode-occurrences/_search"))
				})

				It("should take the first 1000 matches", func() {
					actualRequest := esTransport.receivedHttpRequests[1]
					query := actualRequest.URL.Query()

					Expect(query.Get("size")).To(Equal("1000"))
				})

				It("should collapse fields on resource.uri", func() {
					actualRequest := esTransport.receivedHttpRequests[1]
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

					rodeServer.ListResources(context.Background(), request)
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

					actualRequest := esTransport.receivedHttpRequests[1]
					search := readEsSearchResponse(actualRequest)

					Expect(search.Query).To(Equal(expectedQuery))
				})
			})

			When("Elasticsearch returns with an error", func() {
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
				Expect(esTransport.receivedHttpRequests[1].URL.Path).To(Equal("/rode-v1alpha1-policies/_doc"))
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

			When("attemtping to list the policies", func() {
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

					filter = `name=="abc`
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
					Expect(len(listResponse.Policies)).To((Equal(4)))
				})
				It("should have generated a filter query", func() {
					actualRequest := esTransport.receivedHttpRequests[1]
					search := readEsSearchResponse(actualRequest)

					Expect(search.Query).To(Equal(expectedQuery))
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
							Paths: []string{"regoContent"},
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
	})
})

func createRandomOccurrence(kind grafeas_common_proto.NoteKind) *grafeas_proto.Occurrence {
	return &grafeas_proto.Occurrence{
		Name: gofakeit.LetterN(10),
		Resource: &grafeas_proto.Resource{
			Uri: gofakeit.URL(),
		},
		NoteName:    gofakeit.LetterN(10),
		Kind:        kind,
		Remediation: gofakeit.LetterN(10),
		CreateTime:  timestamppb.New(gofakeit.Date()),
		UpdateTime:  timestamppb.New(gofakeit.Date()),
		Details:     nil,
	}
}

func createEsIndexResponse(index string) *esIndexResponse {
	return &esIndexResponse{
		Acknowledged:       true,
		ShardsAcknowledged: true,
		Index:              index,
	}
}

type esIndexResponse struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
}

func createEsDeleteDocResponse() *esDeleteDocResponse {
	return &esDeleteDocResponse{
		Took:             int(gofakeit.Int16()),
		TimedOut:         false,
		Total:            int(gofakeit.Int16()),
		Deleted:          int(gofakeit.Int16()),
		Batches:          int(gofakeit.Int16()),
		VersionConflicts: int(gofakeit.Int16()),
		Noops:            int(gofakeit.Int16()),
		Retries: struct {
			Bulk   int "json:\"bulk\""
			Search int "json:\"search\""
		}{
			Bulk:   int(gofakeit.Int16()),
			Search: int(gofakeit.Int16()),
		},
		ThrottledMillis:      int(gofakeit.Int16()),
		RequestsPerSecond:    gofakeit.Float64(),
		ThrottledUntilMillis: int(gofakeit.Int16()),
		Failures:             nil,
	}
}

type esDeleteDocResponse struct {
	Took             int  `json:"took"`
	TimedOut         bool `json:"timed_out"`
	Total            int  `json:"total"`
	Deleted          int  `json:"deleted"`
	Batches          int  `json:"batches"`
	VersionConflicts int  `json:"version_conflicts"`
	Noops            int  `json:"noops"`
	Retries          struct {
		Bulk   int `json:"bulk"`
		Search int `json:"search"`
	} `json:"retries"`
	ThrottledMillis      int           `json:"throttled_millis"`
	RequestsPerSecond    float64       `json:"requests_per_second"`
	ThrottledUntilMillis int           `json:"throttled_until_millis"`
	Failures             []interface{} `json:"failures"`
}

func structToJsonBody(i interface{}) io.ReadCloser {
	b, err := json.Marshal(i)
	Expect(err).ToNot(HaveOccurred())

	return ioutil.NopCloser(strings.NewReader(string(b)))
}

func createEsSearchResponse(occurrences []*grafeas_proto.Occurrence) io.ReadCloser {
	var occurrenceHits []*esSearchResponseHit

	for _, occurrence := range occurrences {
		source, err := protojson.Marshal(proto.MessageV2(occurrence))
		Expect(err).To(BeNil())

		response := &esSearchResponseHit{
			ID:     gofakeit.UUID(),
			Source: source,
		}

		occurrenceHits = append(occurrenceHits, response)
	}

	response := &esSearchResponse{
		Hits: &esSearchResponseHits{
			Hits: occurrenceHits,
		},
		Took: gofakeit.Number(1, 10),
	}

	responseBody, err := json.Marshal(response)
	Expect(err).To(BeNil())

	return ioutil.NopCloser(bytes.NewReader(responseBody))
}

func createEsSearchResponseForPolicy(occurrences []*pb.Policy) io.ReadCloser {
	var occurrenceHits []*esSearchResponseHit

	for _, occurrence := range occurrences {
		source, err := protojson.Marshal(proto.MessageV2(occurrence))
		Expect(err).To(BeNil())

		response := &esSearchResponseHit{
			ID:     gofakeit.UUID(),
			Source: source,
		}

		occurrenceHits = append(occurrenceHits, response)
	}

	response := &esSearchResponse{
		Hits: &esSearchResponseHits{
			Total: &esSearchResponseTotal{
				Value: len(occurrences),
			},
			Hits: occurrenceHits,
		},
		Took: gofakeit.Number(1, 10),
	}

	responseBody, err := json.Marshal(response)
	Expect(err).To(BeNil())

	return ioutil.NopCloser(bytes.NewReader(responseBody))
}

func readEsSearchResponse(request *http.Request) *esSearch {
	requestBody, err := ioutil.ReadAll(request.Body)
	Expect(err).To(BeNil())

	search := &esSearch{}
	err = json.Unmarshal(requestBody, search)
	Expect(err).To(BeNil())

	return search
}

func createRandomPolicyEntity(policy string) *pb.PolicyEntity {
	return &pb.PolicyEntity{
		Name:        gofakeit.LetterN(10),
		Description: gofakeit.LetterN(50),
		RegoContent: policy,
		SourcePath:  gofakeit.URL(),
	}
}
