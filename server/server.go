package server

import (
	"context"

	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
)

type rodeServer struct {
	pb.UnimplementedRodeServer
	logger        *zap.Logger
	grafeasClient grafeas.GrafeasV1Beta1Client
}

func (r *rodeServer) BatchCreateOccurrences(ctx context.Context, occurrenceRequest *pb.BatchCreateOccurrencesRequest) (*pb.BatchCreateOccurrencesResponse, error) {
	log := r.logger.Named("BatchCreateOccurrences")
	log.Debug("received request", zap.Any("BatchCreateOccurrencesRequest", occurrenceRequest))

	//Forward to grafeas to create occurrence
	occurrenceResponse, err := r.grafeasClient.BatchCreateOccurrences(ctx, &grafeas.BatchCreateOccurrencesRequest{
		Parent:      "projects/rode",
		Occurrences: occurrenceRequest.GetOccurrences(),
	})
	if err != nil {
		log.Error("failed to create occurrences", zap.NamedError("error", err))
		return nil, err
	}

	return &pb.BatchCreateOccurrencesResponse{
		Occurrences: occurrenceResponse.GetOccurrences(),
	}, nil
}

func NewRodeServer(logger *zap.Logger, grafeasClient grafeas.GrafeasV1Beta1Client) pb.RodeServer {
	return &rodeServer{
		logger:        logger,
		grafeasClient: grafeasClient,
	}
}
