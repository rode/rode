package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/brianvoe/gofakeit/v5"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	gomock "github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/mocks"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"io"
	"io/ioutil"
	"net/http"
)

var _ = Describe("rode server", func() {
	const (
		createProjectError = "CREATE_PROJECT_ERROR"
		getProjectError    = "GET_PROJECT_ERROR"
	)

	var (
		log                   *zap.Logger
		rodeServer            pb.RodeServer
		rodeServerError       error
		grafeasClient         *mocks.MockGrafeasV1Beta1Client
		grafeasProjectsClient *mocks.MockProjectsClient
		esClient              *elasticsearch.Client
		esTransport           *mockEsTransport
		mockFilterer          *mocks.MockFilterer
		mockCtrl              *gomock.Controller
		getProjectRequest     = &grafeas_project_proto.GetProjectRequest{Name: "projects/rode"}
	)

	BeforeEach(func() {
		log = logger.Named("rode server test")
		mockCtrl = gomock.NewController(GinkgoT())
		grafeasClient = mocks.NewMockGrafeasV1Beta1Client(mockCtrl)
		grafeasProjectsClient = mocks.NewMockProjectsClient(mockCtrl)

		esTransport = &mockEsTransport{}
		esClient = &elasticsearch.Client{
			Transport: esTransport,
			API:       esapi.New(esTransport),
		}
		mockFilterer = mocks.NewMockFilterer(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("server is initialized", func() {
		It("should check if the rode project exists", func() {
			grafeasProjectsClient.
				EXPECT().
				GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest))

			rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)
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

				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)
			})

			When("create project returns error from Grafeas", func() {
				BeforeEach(func() {
					grafeasProjectsClient.
						EXPECT().
						CreateProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(createProjectRequest)).Return(nil, status.Error(codes.Internal, createProjectError))
				})

				It("should returns error", func() {
					rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)

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
					rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)

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

				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)
			})

			It("should return the Rode server", func() {
				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)

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
				rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)

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

			rodeServer, rodeServerError = NewRodeServer(log, grafeasClient, grafeasProjectsClient, esClient, mockFilterer)
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

		When("policy is evaluated", func() {
			var (
				resourceURI            string
				policy                 string
				listOccurrencesRequest *grafeas_proto.ListOccurrencesRequest
				attestPolicyRequest    *pb.AttestPolicyRequest
			)

			BeforeEach(func() {
				resourceURI = gofakeit.URL()
				policy = gofakeit.Word()
				listOccurrencesRequest = &grafeas_proto.ListOccurrencesRequest{
					Filter: fmt.Sprintf("resource.uri = '%s'", resourceURI),
				}
				attestPolicyRequest = &pb.AttestPolicyRequest{
					ResourceURI: resourceURI,
					Policy:      policy,
				}
			})

			It("should initialize OPA policy", func() {
				// ignore Grafeas list occurrences call
				grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).AnyTimes()

				// expect OPA initPolicy call

				_, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
			})

			When("OPA policy initializes", func() {

				It("should list Grafeas occurrences", func() {
					// expect Grafeas list occurrences call
					grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), listOccurrencesRequest)

					_, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
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

						// expect evalute OPA policy call

						_, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
					})

					When("resource does not have previous attestation occurrence", func() {
						BeforeEach(func() {
							// mock Grafeas list occurrences response
							// mock OPA evalute policy response
						})
						It("should create new attestation occurrence", func() {
							// expect Grafeas create occurrence call
							// _, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
						})
						It("should respond with new attestation occurrence", func() {
							// expect response state to match OPA policy evaluation
							// expect response to include new attestation
							// _, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
						})
					})

					When("resource has previous attestation occurrence", func() {
						BeforeEach(func() {
							// mock Grafeas list occurrences response
						})
						When("OPA policy evaluation is same as previous attestation occurrence", func() {
							BeforeEach(func() {
								// mock OPA evalute policy response
							})
							It("should not create new attestation occurrence", func() {
								// expect Grafeas create occurrence to not be called
								// _, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
							})
							It("should respond with previous the previous attestation occurrence", func() {
								// expect response state to match OPA policy evaluation
								// expect response to include previous attestation
								// _, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
							})
						})

						When("OPA policy evaluation is different than previous attestation occurrence", func() {
							BeforeEach(func() {
								listOccurrencesResponse := &grafeas_proto.ListOccurrencesResponse{
									Occurrences: []*grafeas_proto.Occurrence{
										createRandomOccurrence(grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED),
										createRandomOccurrence(grafeas_common_proto.NoteKind_ATTESTATION),
									},
								}

								grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(listOccurrencesResponse, nil)
								// mock OPA evalute policy response
							})
							It("should create new attestation occurrence", func() {
								// mock Grafeas list occurrences response
								// expect Grafeas create occurrence call
								_, _ = rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
							})
							It("should respond with new attestation occurrence", func() {

								// expect response state to match OPA policy evaluation
								// expect response to include new attestation
								attestPolicyResponse, attestPolicyError := rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)

								Expect(attestPolicyResponse).To(Equal(&pb.AttestPolicyResponse{}))
								Expect(attestPolicyError).ToNot(HaveOccurred())
							})
						})
					})

					When("evalute OPA policy returns error", func() {
						It("should return error", func() {
							// _, attestPolicyError := rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
							// Expect(attestPolicyError).To(HaveOccurred())
						})
					})
				})

				When("Grafeas list occurrences response is error", func() {
					It("should return an error", func() {
						// mock Grafeas list occurrences error response
						grafeasClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("elasticsearch error"))

						_, attestPolicyError := rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)
						Expect(attestPolicyError).To(HaveOccurred())
					})
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
					actualRequest := esTransport.receivedHttpRequests[0]

					Expect(actualRequest.URL.Path).To(Equal("/grafeas-v1beta1-rode-occurrences/_search"))
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

					actualRequest := esTransport.receivedHttpRequests[0]
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

func readEsSearchResponse(request *http.Request) *esSearch {
	requestBody, err := ioutil.ReadAll(request.Body)
	Expect(err).To(BeNil())

	search := &esSearch{}
	err = json.Unmarshal(requestBody, search)
	Expect(err).To(BeNil())

	return search
}