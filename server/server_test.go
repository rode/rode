package server

import (
	"context"
	"fmt"

	"github.com/brianvoe/gofakeit/v5"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/mocks"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("rode server", func() {
	var (
		rodeServer            pb.RodeServer
		rodeServerError       error
		grafeasClient         *mocks.MockGrafeasV1Beta1Client
		grafeasProjectsClient *mocks.MockProjectsClient
		mockCtrl              *gomock.Controller
		getProjectRequest     = &grafeas_project_proto.GetProjectRequest{Name: "projects/rode"}
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		grafeasClient = mocks.NewMockGrafeasV1Beta1Client(mockCtrl)
		grafeasProjectsClient = mocks.NewMockProjectsClient(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("server is initialized", func() {
		It("checks if rode project exists", func() {
			grafeasProjectsClient.
				EXPECT().
				GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest))

			rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})
		})

		When("rode project does not exists", func() {
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

			It("creates rode project", func() {
				grafeasProjectsClient.
					EXPECT().
					CreateProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(createProjectRequest))

				rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})
			})

			When("create project returns error from Grafeas", func() {
				BeforeEach(func() {
					grafeasProjectsClient.
						EXPECT().
						CreateProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(createProjectRequest)).Return(nil, status.Error(codes.Internal, "CREATE_PROJECT_ERROR"))
				})

				It("returns error", func() {
					rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})

					Expect(rodeServerError).To(HaveOccurred())
					Expect(rodeServerError.Error()).To(ContainSubstring("CREATE_PROJECT_ERROR"))
				})
			})

			When("create project succeeds", func() {
				BeforeEach(func() {
					grafeasProjectsClient.
						EXPECT().
						CreateProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(createProjectRequest)).Return(&grafeas_project_proto.Project{}, nil)
				})

				It("returns Rode server", func() {
					rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})

					Expect(rodeServer).ToNot(BeNil())
					Expect(rodeServerError).To(BeNil())
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

			It("does not create project", func() {
				grafeasProjectsClient.
					EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).MaxTimes(0)

				rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})
			})

			It("returns Rode server", func() {
				rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})

				Expect(rodeServer).ToNot(BeNil())
				Expect(rodeServerError).To(BeNil())
			})
		})

		When("get rode project returns error", func() {
			BeforeEach(func() {
				grafeasProjectsClient.
					EXPECT().
					GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest)).
					Return(nil, status.Error(codes.Internal, "GET_PROJECT_ERROR"))
			})

			It("returns and error", func() {
				rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})

				Expect(rodeServerError).To(HaveOccurred())
				Expect(rodeServerError.Error()).To(ContainSubstring("GET_PROJECT_ERROR"))
			})
		})
	})

	Context("server has been initialized", func() {
		BeforeEach(func() {
			grafeasProjectsClient.
				EXPECT().
				GetProject(gomock.AssignableToTypeOf(context.Background()), gomock.Eq(getProjectRequest)).
				Return(&grafeas_project_proto.Project{}, nil)

			rodeServer, rodeServerError = NewRodeServer(logger.Named("rode server test"), GrafeasClients{grafeasClient, grafeasProjectsClient})
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

			It("sends occurrences to Grafeas", func() {
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
