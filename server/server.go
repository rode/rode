package server

import (
	"context"
	"fmt"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
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
	listOccurrencesResponse, err := r.grafeasClient.ListOccurrences(ctx, &grafeas.ListOccurrencesRequest{Filter: fmt.Sprintf("resource.uri = '%s'", request.ResourceURI)})
	if err != nil {
		log.Error("list occurrences failed", zap.Error(err), zap.String("resource", request.ResourceURI))
		return nil, status.Error(codes.Internal, "list occurrences failed")
	}

	// json encode occurrences. list occurrences response should not generate error
	_, _ = protojson.Marshal(proto.MessageV2(listOccurrencesResponse))

	// evalute OPA policy

	// create attestation

	return &pb.AttestPolicyResponse{}, nil
}

func NewRodeServer(logger *zap.Logger, grafeasClient grafeas.GrafeasV1Beta1Client) pb.RodeServer {
	return &rodeServer{
		logger:        logger,
		grafeasClient: grafeasClient,
	}
}
