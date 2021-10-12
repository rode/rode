// Copyright 2021 The Rode Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//counterfeiter:generate . AssignmentManager
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

	if err := m.validateAssignment(ctx, log, assignment, m.validateCreate); err != nil {
		return nil, err
	}

	policyId, _, _ := parsePolicyVersionId(assignment.PolicyVersionId)
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

	if request.Id == "" {
		return nil, createErrorWithCode(log, "invalid policy assignment id", nil, codes.InvalidArgument)
	}

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
	if err = protojson.Unmarshal(response.Source, &assignment); err != nil {
		return nil, createError(log, "error unmarshalling assignment", err)
	}

	return &assignment, nil
}

func (m *assignmentManager) UpdatePolicyAssignment(ctx context.Context, assignment *pb.PolicyAssignment) (*pb.PolicyAssignment, error) {
	log := m.logger.Named("UpdatePolicyAssignment")
	log.Debug("received request")
	assignmentId := assignment.Id

	if err := m.validateAssignment(ctx, log, assignment, m.validateUpdate); err != nil {
		return nil, err
	}
	assignment.Updated = timestamppb.Now()

	if _, err := m.esClient.Update(ctx, &esutil.UpdateRequest{
		Index:      m.policyAssignmentsAlias(),
		DocumentId: assignmentId,
		Refresh:    m.esConfig.Refresh.String(),
		Message:    assignment,
	}); err != nil {
		return nil, createError(log, "error updating policy assignment in Elasticsearch", err)
	}

	return assignment, nil
}

func (m *assignmentManager) DeletePolicyAssignment(ctx context.Context, request *pb.DeletePolicyAssignmentRequest) (*emptypb.Empty, error) {
	log := m.logger.Named("DeletePolicyAssignment").With(zap.String("id", request.Id))
	log.Debug("received request")

	// check that assignment exists
	if _, err := m.GetPolicyAssignment(ctx, &pb.GetPolicyAssignmentRequest{Id: request.Id}); err != nil {
		return nil, err
	}

	if err := m.esClient.Delete(ctx, &esutil.DeleteRequest{
		Index: m.policyAssignmentsAlias(),
		Search: &esutil.EsSearch{
			Query: &filtering.Query{
				Term: &filtering.Term{
					"_id": request.Id,
				},
			},
		},
		Refresh: m.esConfig.Refresh.String(),
	}); err != nil {
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

	if request.PolicyId != "" {
		queries = append(queries, &filtering.Query{
			Prefix: &filtering.Term{
				"policyVersionId": request.PolicyId + ".",
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
			Sort: map[string]esutil.EsSortOrder{
				"created": esutil.EsSortOrderDescending,
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

		if err = protojson.Unmarshal(hit.Source, &assignment); err != nil {
			return nil, createError(log, "error unmarshalling assignment", err)
		}

		response.PolicyAssignments = append(response.PolicyAssignments, &assignment)
	}

	return response, nil
}

func (m *assignmentManager) policyAssignmentsAlias() string {
	return m.indexManager.AliasName(constants.PolicyAssignmentsDocumentKind, "")
}

type assignmentValidator = func(ctx context.Context, log *zap.Logger, policyId string, assignment *pb.PolicyAssignment) error

func (m *assignmentManager) validateAssignment(ctx context.Context, log *zap.Logger, assignment *pb.PolicyAssignment, validate assignmentValidator) error {
	if assignment.PolicyGroup == "" || assignment.PolicyVersionId == "" {
		return createErrorWithCode(log, "Must specify policy group and a policy version", nil, codes.InvalidArgument)
	}

	policyId, version, err := parsePolicyVersionId(assignment.PolicyVersionId)
	if err != nil {
		return createError(log, "error parsing policy version id", err)
	}

	if version == 0 {
		return createErrorWithCode(log, "Assignments must use policy version ids", nil, codes.InvalidArgument)
	}

	if err = validate(ctx, log, policyId, assignment); err != nil {
		return err
	}

	response, err := m.esClient.MultiGet(ctx, &esutil.MultiGetRequest{
		Items: []*esutil.EsMultiGetItem{
			{
				Id:    policyId,
				Index: m.indexManager.AliasName(constants.PoliciesDocumentKind, ""),
			},
			{
				Id:      assignment.PolicyVersionId,
				Index:   m.indexManager.AliasName(constants.PoliciesDocumentKind, ""),
				Routing: policyId,
			},
			{
				Id:    assignment.PolicyGroup,
				Index: m.indexManager.AliasName(constants.PolicyGroupsDocumentKind, ""),
			},
		},
	})

	if err != nil {
		return createError(log, "error retrieving policy version and group", err)
	}

	for i, message := range []string{
		fmt.Sprintf("policy with id %s not found", policyId),
		fmt.Sprintf("policy version with id %s not found", assignment.PolicyVersionId),
		fmt.Sprintf("policy group named %s not found", assignment.PolicyGroup),
	} {
		if !response.Docs[i].Found {
			return createErrorWithCode(log, message, nil, codes.FailedPrecondition)
		}
	}

	var policy pb.Policy
	if err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(response.Docs[0].Source, &policy); err != nil {
		return createError(log, "error parsing policy", err)
	}

	if policy.Deleted {
		return createErrorWithCode(log, "cannot create assignment for a policy that has been deleted", nil, codes.FailedPrecondition)
	}

	var group pb.PolicyGroup
	if err = protojson.Unmarshal(response.Docs[2].Source, &group); err != nil {
		return createError(log, "error parsing policy group", err)
	}

	if group.Deleted {
		return createErrorWithCode(log, "cannot assign a policy to a deleted policy group", nil, codes.FailedPrecondition)
	}

	return nil
}

func (m *assignmentManager) validateCreate(ctx context.Context, log *zap.Logger, policyId string, assignment *pb.PolicyAssignment) error {
	assignmentId := policyAssignmentId(policyId, assignment.PolicyGroup)
	existingAssignment, err := m.GetPolicyAssignment(ctx, &pb.GetPolicyAssignmentRequest{Id: assignmentId})
	if existingAssignment != nil {
		return createErrorWithCode(log, "assignment already exists", nil, codes.AlreadyExists)
	}

	if err != nil && status.Convert(err).Code() != codes.NotFound {
		return err
	}

	return nil
}

func (m *assignmentManager) validateUpdate(ctx context.Context, log *zap.Logger, policyId string, assignment *pb.PolicyAssignment) error {
	currentAssignment, err := m.GetPolicyAssignment(ctx, &pb.GetPolicyAssignmentRequest{Id: assignment.Id})
	if err != nil {
		return err
	}

	if currentAssignment.PolicyGroup != assignment.PolicyGroup {
		return createErrorWithCode(log, "Updating policy group is forbidden", nil, codes.InvalidArgument)
	}

	currentPolicyId, _, _ := parsePolicyVersionId(currentAssignment.PolicyVersionId)
	if currentPolicyId != policyId {
		return createErrorWithCode(log, "Updates may only change policy version, not policy", nil, codes.InvalidArgument)
	}

	assignment.Created = currentAssignment.Created // prevent updates from overwriting the original timestamp

	return nil
}

func policyAssignmentId(policyId, policyGroupName string) string {
	return fmt.Sprintf("policies/%s/assignments/%s", policyId, policyGroupName)
}
