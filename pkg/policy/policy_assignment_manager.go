package policy

import (
	"context"
	"fmt"

	"github.com/rode/es-index-manager/indexmanager"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/config"
	"github.com/rode/rode/pkg/constants"
	pb "github.com/rode/rode/proto/v1alpha1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AssignmentManager interface {
	CreatePolicyAssignment(context.Context, *pb.PolicyAssignment) (*pb.PolicyAssignment, error)
	GetPolicyAssignment(context.Context, *pb.GetPolicyAssignmentRequest) (*pb.PolicyAssignment, error)
	UpdatePolicyAssignment(context.Context, *pb.PolicyAssignment) (*pb.PolicyAssignment, error)
	DeletePolicyAssignment(context.Context, *pb.DeletePolicyAssignmentRequest) (*emptypb.Empty, error)
	ListPolicyAssignments(context.Context, *pb.ListPolicyAssignmentsRequest) (*pb.ListPolicyAssignmentsResponse, error)
}

type assignmentManager struct {
	logger       *zap.Logger
	esClient     esutil.Client
	esConfig     *config.ElasticsearchConfig
	indexManager indexmanager.IndexManager
	filterer     filtering.Filterer
}

func NewAssignmentManager(
	logger *zap.Logger,
	esClient esutil.Client,
	esConfig *config.ElasticsearchConfig,
	indexManager indexmanager.IndexManager,
	filterer filtering.Filterer,
) AssignmentManager {
	return &assignmentManager{
		logger,
		esClient,
		esConfig,
		indexManager,
		filterer,
	}
}

func (m *assignmentManager) CreatePolicyAssignment(ctx context.Context, assignment *pb.PolicyAssignment) (*pb.PolicyAssignment, error) {
	log := m.logger.Named("CreatePolicyAssignment").
		With(zap.String("policyVersionId", assignment.PolicyVersionId)).
		With(zap.String("policyGroup", assignment.PolicyGroup))
	log.Debug("received request")

	if assignment.PolicyGroup == "" || assignment.PolicyVersionId == "" {
		return nil, createErrorWithCode(log, "Must specify policy group and a policy version", nil, codes.InvalidArgument)
	}

	policyId, version, err := parsePolicyVersionId(assignment.PolicyVersionId)
	if err != nil {
		return nil, createError(log, "error parsing policy version id", err)
	}

	if version == 0 {
		return nil, createErrorWithCode(log, "Assignments must use policy version ids", nil, codes.InvalidArgument)
	}

	// TODO: check if already exists
	// TODO: check that policy group and policy exist
	assignmentId := policyAssignmentId(policyId, assignment.PolicyGroup)
	currentTime := timestamppb.Now()
	assignment.Created = currentTime
	assignment.Updated = currentTime
	assignment.Id = assignmentId

	if _, err := m.esClient.Create(ctx, &esutil.CreateRequest{
		Index:      m.policyAssignmentsAlias(),
		Refresh:    m.esConfig.Refresh.String(),
		Message:    assignment,
		DocumentId: assignmentId,
	}); err != nil {
		return nil, createError(log, "error creating policy assignment", err)
	}

	return assignment, nil
}

func (m *assignmentManager) GetPolicyAssignment(ctx context.Context, request *pb.GetPolicyAssignmentRequest) (*pb.PolicyAssignment, error) {
	log := m.logger.Named("GetPolicyAssignment").With(zap.String("id", request.Id))
	log.Debug("received request")

	response, err := m.esClient.Get(ctx, &esutil.GetRequest{
		Index:      m.policyAssignmentsAlias(),
		DocumentId: request.Id,
	})

	if err != nil {
		return nil, createError(log, "error retrieving policy assignment", err)
	}

	if !response.Found {
		return nil, createErrorWithCode(log, "assignment not found", nil, codes.NotFound)
	}
	var assignment pb.PolicyAssignment
	if err := protojson.Unmarshal(response.Source, &assignment); err != nil {
		return nil, createError(log, "error unmarshalling assignment", err)
	}

	return &assignment, nil
}

func (m *assignmentManager) UpdatePolicyAssignment(ctx context.Context, assignment *pb.PolicyAssignment) (*pb.PolicyAssignment, error) {
	log := m.logger.Named("UpdatePolicyAssignment")
	log.Debug("received request")
	assignmentId := assignment.Id
	currentAssignment, err := m.GetPolicyAssignment(ctx, &pb.GetPolicyAssignmentRequest{Id: assignmentId})
	if err != nil {
		return nil, err
	}

	if currentAssignment.PolicyGroup != assignment.PolicyGroup {
		return nil, createErrorWithCode(log, "Updating policy group is not allowed", nil, codes.InvalidArgument)
	}

	currentAssignment.PolicyVersionId = assignment.PolicyVersionId
	currentAssignment.Updated = timestamppb.Now()

	if _, err := m.esClient.Update(ctx, &esutil.UpdateRequest{
		Index:      m.policyAssignmentsAlias(),
		DocumentId: assignmentId,
		Refresh:    m.esConfig.Refresh.String(),
		Message:    currentAssignment,
	}); err != nil {
		return nil, createError(log, "error updating policy assignment in Elasticsearch", err)
	}

	return currentAssignment, nil
}

func (m *assignmentManager) DeletePolicyAssignment(ctx context.Context, request *pb.DeletePolicyAssignmentRequest) (*emptypb.Empty, error) {
	log := m.logger.Named("DeletePolicyAssignment").With(zap.String("id", request.Id))
	log.Debug("received request")

	err := m.esClient.Delete(ctx, &esutil.DeleteRequest{
		Index: m.policyAssignmentsAlias(),
		Search: &esutil.EsSearch{
			Query: &filtering.Query{
				Term: &filtering.Term{
					"_id": request.Id,
				},
			},
		},
		Refresh: m.esConfig.Refresh.String(),
	})

	if err != nil {
		return nil, createError(log, "error deleting assignment", err)
	}

	return &emptypb.Empty{}, nil
}

func (m *assignmentManager) ListPolicyAssignments(ctx context.Context, request *pb.ListPolicyAssignmentsRequest) (*pb.ListPolicyAssignmentsResponse, error) {
	log := m.logger.Named("ListPolicyAssignments")
	log.Debug("received request", zap.Any("request", request))

	queries := filtering.Must{}

	if request.PolicyGroup != "" {
		queries = append(queries, &filtering.Query{
			Term: &filtering.Term{
				"policyGroup": request.PolicyGroup,
			},
		})
	}

	if request.PolicyVersionId != "" {
		queries = append(queries, &filtering.Query{
			Prefix: &filtering.Term{
				"policyVersionId": request.PolicyVersionId + ".",
			},
		})
	}

	if request.Filter != "" {
		filterQuery, err := m.filterer.ParseExpression(request.Filter)
		if err != nil {
			return nil, createError(log, "error creating filter query", err)
		}

		queries = append(queries, filterQuery)
	}

	searchRequest := &esutil.SearchRequest{
		Index: m.policyAssignmentsAlias(),
		Search: &esutil.EsSearch{
			Query: &filtering.Query{
				Bool: &filtering.Bool{
					Must: &queries,
				},
			},
		},
	}

	if request.PageSize != 0 {
		searchRequest.Pagination = &esutil.SearchPaginationOptions{
			Size:  int(request.PageSize),
			Token: request.PageToken,
		}
	}

	searchResponse, err := m.esClient.Search(ctx, searchRequest)
	if err != nil {
		return nil, createError(log, "error searching for policy assignments", err)
	}

	response := &pb.ListPolicyAssignmentsResponse{
		NextPageToken: searchResponse.NextPageToken,
	}
	for _, hit := range searchResponse.Hits.Hits {
		var assignment pb.PolicyAssignment

		if err := protojson.Unmarshal(hit.Source, &assignment); err != nil {
			return nil, createError(log, "error unmarshalling assignment", err)
		}

		response.PolicyAssignments = append(response.PolicyAssignments, &assignment)
	}

	return response, nil
}

func (m *assignmentManager) policyAssignmentsAlias() string {
	return m.indexManager.AliasName(constants.PolicyAssignmentsDocumentKind, "")
}

func policyAssignmentId(policyId, policyGroupName string) string {
	return fmt.Sprintf("policies/%s/assignments/%s", policyId, policyGroupName)
}

// /policies/{policyId}/assignments/{policyGroupId} -- fine for GetPolicyAssignment, weird for CreatePolicyAssignment -- we want policyVersionId, not policyId
// /policy-assignments/{assignmentId} -- id is `/policies/{policyId}/assignments/{policyGroupId}`, making this path look strange

// CreatePolicyAssignment: /policy-groups/{policyGroupId}:assign
// ListPolicyAssignments: /policy-groups/{policyGroupId}/assignments, /policies/{policyId}/assignments
// GetPolicyAssignment/UpdatePolicyAssignment/DeletePolicyAssignment: /policy-groups/{policyGroupId}/assignments/{policyId}
