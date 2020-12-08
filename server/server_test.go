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
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("rode server", func() {
	var (
		rodeServer    pb.RodeServer
		grafeasClient *mocks.MockGrafeasV1Beta1Client
		mockCtrl      *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		grafeasClient = mocks.NewMockGrafeasV1Beta1Client(mockCtrl)
		rodeServer = NewRodeServer(logger.Named("rode server test"), grafeasClient)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("occurrences are created", func() {
		var (
			randomOccurrence                      *grafeas_proto.Occurrence
			grafeasBatchCreateOccurrencesRequest  *grafeas_proto.BatchCreateOccurrencesRequest
			grafeasBatchCreateOccurrencesResponse *grafeas_proto.BatchCreateOccurrencesResponse
		)

		BeforeEach(func() {
			randomOccurrence = createRandomUnspecifiedOccurrence()
			// expected Grafeas BatchCreateOccurrences request
			grafeasBatchCreateOccurrencesRequest = &grafeas_proto.BatchCreateOccurrencesRequest{
				Parent: "projects/rode",
				Occurrences: []*grafeas_proto.Occurrence{
					randomOccurrence,
				},
			}
			// mock Grafeas BatchCreateOccurrences response
			grafeasBatchCreateOccurrencesResponse = &grafeas_proto.BatchCreateOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					randomOccurrence,
				},
			}
		})

		It("should forward the batch create occurrence request to grafeas", func() {
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
			resourceURI             string
			policy                  string
			listOccurrencesRequest  *grafeas_proto.ListOccurrencesRequest
			listOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
		)

		BeforeEach(func() {
			resourceURI = gofakeit.URL()
			policy = gofakeit.Word()
			listOccurrencesRequest = &grafeas_proto.ListOccurrencesRequest{
				Filter: fmt.Sprintf("resource.uri = '%s'", resourceURI),
			}
			listOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					createRandomUnspecifiedOccurrence(),
				},
			}
		})

		It("should check OPA policy is loaded", func() {

		})

		When("OPA policy is not loaded", func() {
			It("initializes OPA policies", func() {

			})
		})

		It("should fetch resource occurrences", func() {
			grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), listOccurrencesRequest).Return(listOccurrencesResponse, nil)

			attestPolicyRequest := &pb.AttestPolicyRequest{
				ResourceURI: resourceURI,
				Policy:      policy,
			}
			_, err := rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)

			Expect(err).ToNot(HaveOccurred())
		})

		When("Grafeas list occurrences fails", func() {
			It("returns an error", func() {
				grafeasClient.EXPECT().ListOccurrences(gomock.AssignableToTypeOf(context.Background()), listOccurrencesRequest).Return(listOccurrencesResponse, fmt.Errorf("Grafeas error"))

				attestPolicyRequest := &pb.AttestPolicyRequest{
					ResourceURI: resourceURI,
					Policy:      policy,
				}
				_, err := rodeServer.AttestPolicy(context.Background(), attestPolicyRequest)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("list occurrences failed"))
			})
		})

		It("should evaluate policy with occurrence data", func() {

		})

		When("policy evaluation fails", func() {
			It("should create failed attestation", func() {

			})
		})

		When("policy evaluation succeeds", func() {
			It("should create success attestation", func() {

			})
		})

		It("should respond with policy attestation response", func() {

		})
	})
})

func createRandomUnspecifiedOccurrence() *grafeas_proto.Occurrence {
	return &grafeas_proto.Occurrence{
		Name: gofakeit.LetterN(10),
		Resource: &grafeas_proto.Resource{
			Uri: gofakeit.LetterN(10),
		},
		NoteName:    gofakeit.LetterN(10),
		Kind:        grafeas_common_proto.NoteKind_NOTE_KIND_UNSPECIFIED,
		Remediation: gofakeit.LetterN(10),
		CreateTime:  timestamppb.New(gofakeit.Date()),
		UpdateTime:  timestamppb.New(gofakeit.Date()),
		Details:     nil,
	}
}
