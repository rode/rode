package server

import (
	"context"

	"github.com/brianvoe/gofakeit/v5"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("rode server", func() {
	var (
		rodeServer    pb.RodeServer
		grafeasClient *mockGrafeasClient
	)

	BeforeEach(func() {
		grafeasClient = &mockGrafeasClient{}
		rodeServer = NewRodeServer(logger.Named("rode server test"), grafeasClient)
	})

	When("occurrences are created", func() {
		var (
			randomOccurrence *grafeas_proto.Occurrence
		)

		JustBeforeEach(func() {
			randomOccurrence = createRandomUnspecifiedOccurrence()
			grafeasClient.preparedBatchCreateOccurrenceResponse = &grafeas_proto.BatchCreateOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					randomOccurrence,
				},
			}
		})

		It("should forward the batch create occurrence request to grafeas", func() {
			response, err := rodeServer.BatchCreateOccurrences(context.Background(), &pb.BatchCreateOccurrencesRequest{
				Occurrences: []*grafeas_proto.Occurrence{
					randomOccurrence,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// ensure occurrences were forwarded to grafeas
			Expect(grafeasClient.receivedBatchCreateOccurrenceRequest.Occurrences).To(HaveLen(1))
			Expect(grafeasClient.receivedBatchCreateOccurrenceRequest.Occurrences[0]).To(BeEquivalentTo(randomOccurrence))

			// ensure correct project was set
			Expect(grafeasClient.receivedBatchCreateOccurrenceRequest.Parent).To(BeEquivalentTo("projects/rode"))

			// check response
			Expect(response.GetOccurrences()[0]).To(BeEquivalentTo(randomOccurrence))
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
