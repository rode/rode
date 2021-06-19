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
	"regexp"

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

//counterfeiter:generate . PolicyGroupManager
type PolicyGroupManager interface {
	CreatePolicyGroup(context.Context, *pb.PolicyGroup) (*pb.PolicyGroup, error)
	ListPolicyGroups(context.Context, *pb.ListPolicyGroupsRequest) (*pb.ListPolicyGroupsResponse, error)
	GetPolicyGroup(context.Context, *pb.GetPolicyGroupRequest) (*pb.PolicyGroup, error)
	UpdatePolicyGroup(context.Context, *pb.PolicyGroup) (*pb.PolicyGroup, error)
	DeletePolicyGroup(context.Context, *pb.DeletePolicyGroupRequest) (*emptypb.Empty, error)
}

var policyGroupNamePattern = regexp.MustCompile("^[a-z0-9-_]+$")

type policyGroupManager struct {
	logger       *zap.Logger
	esClient     esutil.Client
	esConfig     *config.ElasticsearchConfig
	indexManager indexmanager.IndexManager
	filterer     filtering.Filterer
}

func NewPolicyGroupManager(
	logger *zap.Logger,
	esClient esutil.Client,
	esConfig *config.ElasticsearchConfig,
	indexManager indexmanager.IndexManager,
	filterer filtering.Filterer,
) PolicyGroupManager {
	return &policyGroupManager{
		logger,
		esClient,
		esConfig,
		indexManager,
		filterer,
	}
}

func (m *policyGroupManager) CreatePolicyGroup(ctx context.Context, policyGroup *pb.PolicyGroup) (*pb.PolicyGroup, error) {
	log := m.logger.Named("CreatePolicyGroup").With(zap.String("name", policyGroup.Name))
	log.Debug("received request")

	existingPolicyGroup, err := m.GetPolicyGroup(ctx, &pb.GetPolicyGroupRequest{Name: policyGroup.Name})
	if existingPolicyGroup != nil {
		return nil, createErrorWithCode(log, "policy group already exists", nil, codes.AlreadyExists)
	}

	if err != nil && status.Convert(err).Code() != codes.NotFound {
		return nil, err
	}

	if !policyGroupNamePattern.MatchString(policyGroup.Name) {
		return nil, createErrorWithCode(log, "policy group name can only contain lowercase alphanumeric characters, dashes, and underscores.", nil, codes.InvalidArgument)
	}

	currentTime := timestamppb.Now()
	policyGroup.Created = currentTime
	policyGroup.Updated = currentTime

	if _, err := m.esClient.Create(ctx, &esutil.CreateRequest{
		Index:      m.policyGroupsAlias(),
		Refresh:    m.esConfig.Refresh.String(),
		Message:    policyGroup,
		DocumentId: policyGroup.Name,
	}); err != nil {
		return nil, createError(log, "error creating policy group", err)
	}

	return policyGroup, nil
}

func (m *policyGroupManager) ListPolicyGroups(ctx context.Context, request *pb.ListPolicyGroupsRequest) (*pb.ListPolicyGroupsResponse, error) {
	log := m.logger.Named("ListPolicyGroups")
	log.Debug("received request")

	queries := filtering.Must{
		&filtering.Query{
			Term: &filtering.Term{
				"deleted": "false",
			},
		},
	}

	if request.Filter != "" {
		query, err := m.filterer.ParseExpression(request.Filter)
		if err != nil {
			return nil, createError(log, "error creating filter query", err)
		}

		queries = append(queries, query)
	}

	searchRequest := &esutil.SearchRequest{
		Index: m.policyGroupsAlias(),
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
		return nil, createError(log, "error searching for policy groups", err)
	}

	response := &pb.ListPolicyGroupsResponse{
		NextPageToken: searchResponse.NextPageToken,
	}
	for _, hit := range searchResponse.Hits.Hits {
		var policyGroup pb.PolicyGroup
		if err := protojson.Unmarshal(hit.Source, &policyGroup); err != nil {
			return nil, createError(log, "error unmarshalling policy group", err)
		}

		response.PolicyGroups = append(response.PolicyGroups, &policyGroup)
	}

	return response, nil
}

func (m *policyGroupManager) GetPolicyGroup(ctx context.Context, request *pb.GetPolicyGroupRequest) (*pb.PolicyGroup, error) {
	log := m.logger.Named("GetPolicyGroup").With(zap.String("name", request.Name))
	log.Debug("received request")

	if request.Name == "" {
		return nil, createErrorWithCode(log, "policy group name must be supplied", nil, codes.InvalidArgument)
	}

	response, err := m.esClient.Get(ctx, &esutil.GetRequest{
		Index:      m.policyGroupsAlias(),
		DocumentId: request.Name,
	})

	if err != nil {
		return nil, createError(log, "error retrieving policy group from elasticsearch", err)
	}

	if !response.Found {
		return nil, createErrorWithCode(log, "policy group not found", nil, codes.NotFound)
	}

	var policyGroup pb.PolicyGroup
	if err := protojson.Unmarshal(response.Source, &policyGroup); err != nil {
		return nil, createError(log, "error unmarshalling policy group", err)
	}

	return &policyGroup, nil
}

func (m *policyGroupManager) UpdatePolicyGroup(ctx context.Context, policyGroup *pb.PolicyGroup) (*pb.PolicyGroup, error) {
	log := m.logger.Named("UpdatePolicyGroup").With(zap.String("name", policyGroup.Name))
	log.Debug("received request")
	currentPolicyGroup, err := m.GetPolicyGroup(ctx, &pb.GetPolicyGroupRequest{Name: policyGroup.Name})
	if err != nil {
		return nil, err
	}

	if currentPolicyGroup.Deleted {
		return nil, createErrorWithCode(log, "cannot update a deleted policy group", nil, codes.FailedPrecondition)
	}

	currentPolicyGroup.Description = policyGroup.Description // only description is editable
	currentPolicyGroup.Updated = timestamppb.Now()

	if _, err := m.esClient.Update(ctx, &esutil.UpdateRequest{
		Index:      m.policyGroupsAlias(),
		DocumentId: currentPolicyGroup.Name,
		Refresh:    m.esConfig.Refresh.String(),
		Message:    currentPolicyGroup,
	}); err != nil {
		return nil, createError(log, "error updating policy group", err)
	}

	return currentPolicyGroup, nil
}

func (m *policyGroupManager) DeletePolicyGroup(ctx context.Context, request *pb.DeletePolicyGroupRequest) (*emptypb.Empty, error) {
	log := m.logger.Named("DeletePolicyGroup").With(zap.String("name", request.Name))
	log.Debug("received request")
	currentPolicyGroup, err := m.GetPolicyGroup(ctx, &pb.GetPolicyGroupRequest{Name: request.Name})
	if err != nil {
		return nil, err
	}

	currentPolicyGroup.Deleted = true

	if _, err = m.esClient.Update(ctx, &esutil.UpdateRequest{
		Index:      m.policyGroupsAlias(),
		DocumentId: request.Name,
		Refresh:    m.esConfig.Refresh.String(),
		Message:    currentPolicyGroup,
	}); err != nil {
		return nil, createError(log, "error marking policy group as deleted", err)
	}

	return &emptypb.Empty{}, nil
}

func (m *policyGroupManager) policyGroupsAlias() string {
	return m.indexManager.AliasName(constants.PolicyGroupsDocumentKind, "")
}
