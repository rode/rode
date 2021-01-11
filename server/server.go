package server

import (
	"context"
	"fmt"

	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewRodeServer constructor for rodeServer
func NewRodeServer(logger *zap.Logger, grafeasCommon grafeas_proto.GrafeasV1Beta1Client, grafeasProjects grafeas_project_proto.ProjectsClient, opa opa.Client) (pb.RodeServer, error) {
	rodeServer := &rodeServer{
		logger:          logger,
		grafeasCommon:   grafeasCommon,
		grafeasProjects: grafeasProjects,
		opa:             opa,
	}
	if err := rodeServer.initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize rode server: %s", err)
	}
	return rodeServer, nil
}

type rodeServer struct {
	pb.UnimplementedRodeServer
	logger          *zap.Logger
	grafeasCommon   grafeas_proto.GrafeasV1Beta1Client
	grafeasProjects grafeas_project_proto.ProjectsClient
	opa             opa.Client
}

func (r *rodeServer) BatchCreateOccurrences(ctx context.Context, occurrenceRequest *pb.BatchCreateOccurrencesRequest) (*pb.BatchCreateOccurrencesResponse, error) {
	log := r.logger.Named("BatchCreateOccurrences")
	log.Debug("received request", zap.Any("BatchCreateOccurrencesRequest", occurrenceRequest))

	//Forward to grafeas to create occurrence
	occurrenceResponse, err := r.grafeasCommon.BatchCreateOccurrences(ctx, &grafeas_proto.BatchCreateOccurrencesRequest{
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
	listOccurrencesResponse, err := r.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{Filter: fmt.Sprintf("resource.uri = '%s'", request.ResourceURI)})
	if err != nil {
		log.Error("list occurrences failed", zap.Error(err), zap.String("resource", request.ResourceURI))
		return nil, status.Error(codes.Internal, "list occurrences failed")
	}

	// json encode occurrences. list occurrences response should not generate error
	input, _ := protojson.Marshal(proto.MessageV2(listOccurrencesResponse))

	// evalute OPA policy
	evaluatePolicyResult, err := r.opa.EvaluatePolicy(request.Policy, input)
	if err != nil {
		log.Error("evaluate OPA policy failed")
		return nil, status.Error(codes.Internal, "evaluate OPA policy failed")
	}

	attestation := &pb.AttestPolicyAttestation{}
	attestation.Created = timestamppb.Now()
	for _, violation := range evaluatePolicyResult.Violations {
		attestation.Violations = append(attestation.Violations, &pb.AttestPolicyViolation{
			Id:          violation.ID,
			Name:        violation.Name,
			Description: violation.Description,
			Message:     violation.Message,
			Link:        violation.Link,
		})
	}

	return &pb.AttestPolicyResponse{
		Pass: evaluatePolicyResult.Pass,
		Attestations: []*pb.AttestPolicyAttestation{
			attestation,
		},
	}, nil
}

func (r *rodeServer) initialize(ctx context.Context) error {
	log := r.logger.Named("initialize")

	_, err := r.grafeasProjects.GetProject(ctx, &grafeas_project_proto.GetProjectRequest{Name: "projects/rode"})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			_, err := r.grafeasProjects.CreateProject(ctx, &grafeas_project_proto.CreateProjectRequest{Project: &grafeas_project_proto.Project{Name: "projects/rode"}})
			if err != nil {
				log.Error("failed to create rode project", zap.Error(err))
				return err
			}
			log.Info("created rode project")
		} else {
			log.Error("error checking if rode project exists", zap.Error(err))
			return err
		}
	}

	return nil
}
