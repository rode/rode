package server

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var logger *zap.Logger

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	logger, _ = zap.NewDevelopment()
	gofakeit.Seed(time.Now().UnixNano())
})

type mockGrafeasClient struct {
	receivedBatchCreateOccurrenceRequest  *grafeas_proto.BatchCreateOccurrencesRequest
	preparedBatchCreateOccurrenceResponse *grafeas_proto.BatchCreateOccurrencesResponse
}

func (c *mockGrafeasClient) GetOccurrence(context.Context, *grafeas_proto.GetOccurrenceRequest, ...grpc.CallOption) (*grafeas_proto.Occurrence, error) {
	return nil, nil
}

func (c *mockGrafeasClient) ListOccurrences(context.Context, *grafeas_proto.ListOccurrencesRequest, ...grpc.CallOption) (*grafeas_proto.ListOccurrencesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) DeleteOccurrence(context.Context, *grafeas_proto.DeleteOccurrenceRequest, ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func (c *mockGrafeasClient) CreateOccurrence(context.Context, *grafeas_proto.CreateOccurrenceRequest, ...grpc.CallOption) (*grafeas_proto.Occurrence, error) {
	return nil, nil
}

func (c *mockGrafeasClient) BatchCreateOccurrences(ctx context.Context, req *grafeas_proto.BatchCreateOccurrencesRequest, opt ...grpc.CallOption) (*grafeas_proto.BatchCreateOccurrencesResponse, error) {
	c.receivedBatchCreateOccurrenceRequest = req

	// if we have a prepared response, send it. otherwise, return nil
	if c.preparedBatchCreateOccurrenceResponse != nil {
		return c.preparedBatchCreateOccurrenceResponse, nil
	}

	return nil, nil
}

func (c *mockGrafeasClient) UpdateOccurrence(context.Context, *grafeas_proto.UpdateOccurrenceRequest, ...grpc.CallOption) (*grafeas_proto.Occurrence, error) {
	return nil, nil
}

func (c *mockGrafeasClient) GetOccurrenceNote(context.Context, *grafeas_proto.GetOccurrenceNoteRequest, ...grpc.CallOption) (*grafeas_proto.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) GetNote(context.Context, *grafeas_proto.GetNoteRequest, ...grpc.CallOption) (*grafeas_proto.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) ListNotes(context.Context, *grafeas_proto.ListNotesRequest, ...grpc.CallOption) (*grafeas_proto.ListNotesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) DeleteNote(context.Context, *grafeas_proto.DeleteNoteRequest, ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func (c *mockGrafeasClient) CreateNote(context.Context, *grafeas_proto.CreateNoteRequest, ...grpc.CallOption) (*grafeas_proto.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) BatchCreateNotes(context.Context, *grafeas_proto.BatchCreateNotesRequest, ...grpc.CallOption) (*grafeas_proto.BatchCreateNotesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) UpdateNote(context.Context, *grafeas_proto.UpdateNoteRequest, ...grpc.CallOption) (*grafeas_proto.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) ListNoteOccurrences(context.Context, *grafeas_proto.ListNoteOccurrencesRequest, ...grpc.CallOption) (*grafeas_proto.ListNoteOccurrencesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) GetVulnerabilityOccurrencesSummary(context.Context, *grafeas_proto.GetVulnerabilityOccurrencesSummaryRequest, ...grpc.CallOption) (*grafeas_proto.VulnerabilityOccurrencesSummary, error) {
	return nil, nil
}
