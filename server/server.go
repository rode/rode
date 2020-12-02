package server

import (
	"context"

	"encoding/json"
	"fmt"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (r *rodeServer) AttestPolicy(ctx context.Context, request *pb.AttestPolicyRequest) (*pb.AttestPolicyResponse, error) {
	log := r.logger.Named("AttestPolicy")
	log.Debug("received requests")

	// check OPA policy has been loaded

	// fetch occurrences from grafeas
	listOccurrencesResponse, err := r.grafeasClient.ListOccurrences(ctx, &grafeas.ListOccurrencesRequest{Filter: fmt.Sprintf("resourceUri:%s", request.ResourceURI)})
	if err != nil {
		log.Error("failed to list occurrences for resource", zap.NamedError("error", err), zap.String("resource", request.ResourceURI))
		return nil, err
	}

	// json encode occurrences
	_, err = json.Marshal(listOccurrencesResponse)
	if err != nil {
		log.Error("failed to encode resource occurrences", zap.NamedError("error", err))
		return nil, err
	}

	// evalute OPA policy

	// create attestation

	return &pb.AttestPolicyResponse{
		Allow:   false,
		Changed: false,
		Attestations: []*pb.AttestPolicyAttestation{
			{
				Allow:      false,
				Created:    timestamppb.Now(),
				Violations: []*pb.AttestPolicyViolation{},
			},
		},
	}, nil
}

func NewRodeServer(logger *zap.Logger, grafeasClient grafeas.GrafeasV1Beta1Client) pb.RodeServer {
	return &rodeServer{
		logger:        logger,
		grafeasClient: grafeasClient,
	}
}
