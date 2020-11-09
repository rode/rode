package server

import (
	"context"
	"github.com/brianvoe/gofakeit/v5"
	"github.com/golang/protobuf/ptypes/empty"
	grafeas "github.com/liatrio/rode-api/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"testing"
	"time"
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
	receivedBatchCreateOccurrenceRequest  *grafeas.BatchCreateOccurrencesRequest
	preparedBatchCreateOccurrenceResponse *grafeas.BatchCreateOccurrencesResponse
}

func (c *mockGrafeasClient) GetOccurrence(context.Context, *grafeas.GetOccurrenceRequest, ...grpc.CallOption) (*grafeas.Occurrence, error) {
	return nil, nil
}

func (c *mockGrafeasClient) ListOccurrences(context.Context, *grafeas.ListOccurrencesRequest, ...grpc.CallOption) (*grafeas.ListOccurrencesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) DeleteOccurrence(context.Context, *grafeas.DeleteOccurrenceRequest, ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func (c *mockGrafeasClient) CreateOccurrence(context.Context, *grafeas.CreateOccurrenceRequest, ...grpc.CallOption) (*grafeas.Occurrence, error) {
	return nil, nil
}

func (c *mockGrafeasClient) BatchCreateOccurrences(ctx context.Context, req *grafeas.BatchCreateOccurrencesRequest, opt ...grpc.CallOption) (*grafeas.BatchCreateOccurrencesResponse, error) {
	c.receivedBatchCreateOccurrenceRequest = req

	// if we have a prepared response, send it. otherwise, return nil
	if c.preparedBatchCreateOccurrenceResponse != nil {
		return c.preparedBatchCreateOccurrenceResponse, nil
	}

	return nil, nil
}

func (c *mockGrafeasClient) UpdateOccurrence(context.Context, *grafeas.UpdateOccurrenceRequest, ...grpc.CallOption) (*grafeas.Occurrence, error) {
	return nil, nil
}

func (c *mockGrafeasClient) GetOccurrenceNote(context.Context, *grafeas.GetOccurrenceNoteRequest, ...grpc.CallOption) (*grafeas.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) GetNote(context.Context, *grafeas.GetNoteRequest, ...grpc.CallOption) (*grafeas.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) ListNotes(context.Context, *grafeas.ListNotesRequest, ...grpc.CallOption) (*grafeas.ListNotesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) DeleteNote(context.Context, *grafeas.DeleteNoteRequest, ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func (c *mockGrafeasClient) CreateNote(context.Context, *grafeas.CreateNoteRequest, ...grpc.CallOption) (*grafeas.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) BatchCreateNotes(context.Context, *grafeas.BatchCreateNotesRequest, ...grpc.CallOption) (*grafeas.BatchCreateNotesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) UpdateNote(context.Context, *grafeas.UpdateNoteRequest, ...grpc.CallOption) (*grafeas.Note, error) {
	return nil, nil
}

func (c *mockGrafeasClient) ListNoteOccurrences(context.Context, *grafeas.ListNoteOccurrencesRequest, ...grpc.CallOption) (*grafeas.ListNoteOccurrencesResponse, error) {
	return nil, nil
}

func (c *mockGrafeasClient) GetVulnerabilityOccurrencesSummary(context.Context, *grafeas.GetVulnerabilityOccurrencesSummaryRequest, ...grpc.CallOption) (*grafeas.VulnerabilityOccurrencesSummary, error) {
	return nil, nil
}
