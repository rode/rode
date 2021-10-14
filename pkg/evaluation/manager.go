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

package evaluation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rode/es-index-manager/indexmanager"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/config"
	"github.com/rode/rode/opa"
	"github.com/rode/rode/pkg/constants"
	"github.com/rode/rode/pkg/grafeas"
	"github.com/rode/rode/pkg/policy"
	"github.com/rode/rode/pkg/resource"
	"github.com/rode/rode/pkg/util"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate counterfeiter -generate

//counterfeiter:generate . Manager
type Manager interface {
	EvaluateResource(context.Context, *pb.ResourceEvaluationRequest) (*pb.ResourceEvaluationResult, error)
	GetResourceEvaluation(context.Context, *pb.GetResourceEvaluationRequest) (*pb.ResourceEvaluationResult, error)
	ListResourceEvaluations(context.Context, *pb.ListResourceEvaluationsRequest) (*pb.ListResourceEvaluationsResponse, error)
	EvaluatePolicy(ctx context.Context, request *pb.EvaluatePolicyRequest) (*pb.EvaluatePolicyResponse, error)
}

const (
	evaluationDocumentJoinField    = "join"
	resourceEvaluationRelationName = "resource"
	policyEvaluationRelationName   = "policy"
)

type EvaluationManager Manager

type manager struct {
	logger                  *zap.Logger
	esConfig                *config.ElasticsearchConfig
	esClient                esutil.Client
	policyManager           policy.Manager
	policyGroupManager      policy.PolicyGroupManager
	policyAssignmentManager policy.AssignmentManager
	grafeasExtensions       grafeas.Extensions
	opa                     opa.Client
	resourceManager         resource.Manager
	indexManager            indexmanager.IndexManager
	filterer                filtering.Filterer
}

func NewManager(
	logger *zap.Logger,
	esClient esutil.Client,
	esConfig *config.ElasticsearchConfig,
	policyManager policy.Manager,
	policyGroupManager policy.PolicyGroupManager,
	policyAssignmentManager policy.AssignmentManager,
	grafeasExtensions grafeas.Extensions,
	opa opa.Client,
	resourceManager resource.Manager,
	indexManager indexmanager.IndexManager,
	filterer filtering.Filterer,
) Manager {
	return &manager{
		logger:                  logger,
		esClient:                esClient,
		esConfig:                esConfig,
		policyManager:           policyManager,
		policyGroupManager:      policyGroupManager,
		policyAssignmentManager: policyAssignmentManager,
		grafeasExtensions:       grafeasExtensions,
		opa:                     opa,
		resourceManager:         resourceManager,
		indexManager:            indexManager,
		filterer:                filterer,
	}
}

func (m *manager) EvaluateResource(ctx context.Context, request *pb.ResourceEvaluationRequest) (*pb.ResourceEvaluationResult, error) {
	log := m.logger.Named("EvaluateResource").With(zap.Any("request", request))

	if request.ResourceUri == "" {
		return nil, util.GrpcErrorWithCode(log, "resource uri is required", nil, codes.InvalidArgument)
	}

	if request.PolicyGroup == "" {
		return nil, util.GrpcErrorWithCode(log, "policy group is required", nil, codes.InvalidArgument)
	}

	resourceVersion, err := m.resourceManager.GetResourceVersion(ctx, request.ResourceUri)
	if err != nil {
		return nil, err
	}

	// get the policy group to evaluate against
	policyGroup, err := m.policyGroupManager.GetPolicyGroup(ctx, &pb.GetPolicyGroupRequest{Name: request.PolicyGroup})
	if err != nil {
		return nil, err
	}

	// get policy group assignments
	listPolicyAssignmentsResponse, err := m.policyAssignmentManager.ListPolicyAssignments(ctx, &pb.ListPolicyAssignmentsRequest{
		PolicyGroup: policyGroup.Name,
	})
	if err != nil {
		return nil, err
	}

	if len(listPolicyAssignmentsResponse.PolicyAssignments) == 0 {
		return nil, util.GrpcErrorWithCode(log, fmt.Sprintf("policy group %s has no policy assignments", policyGroup.Name), nil, codes.FailedPrecondition)
	}

	// fetch occurrences from grafeas
	occurrences, nextPage, err := m.grafeasExtensions.ListVersionedResourceOccurrences(ctx, request.ResourceUri, "", constants.MaxPageSize)
	if err != nil {
		return nil, util.GrpcInternalError(log, "error listing occurrences", err)
	}
	if nextPage != "" {
		log.Warn(fmt.Sprintf("listing occurrences for resource %s resulted in more than %d occurrences, proceeding with evaluation anyway", request.ResourceUri, constants.MaxPageSize))
	}

	resourceEvaluation := &pb.ResourceEvaluation{
		Id:              uuid.New().String(),
		Pass:            true, // defaults to true, but will be set to false if any policy evaluations fail
		Source:          request.Source,
		Created:         timestamppb.Now(),
		ResourceVersion: resourceVersion,
		PolicyGroup:     policyGroup.Name,
	}
	bulkRequestItems := []*esutil.BulkRequestItem{
		{
			Operation:  esutil.BULK_CREATE,
			Message:    resourceEvaluation,
			DocumentId: resourceEvaluation.Id,
			Join: &esutil.EsJoin{
				Field: evaluationDocumentJoinField,
				Name:  resourceEvaluationRelationName,
			},
		},
	}

	var policyEvaluations []*pb.PolicyEvaluation
	for _, policyAssignment := range listPolicyAssignmentsResponse.PolicyAssignments {
		policyEntity, err := m.policyManager.GetPolicyVersion(ctx, policyAssignment.PolicyVersionId)
		if err != nil {
			return nil, util.GrpcInternalError(log, "error fetching policy version", err)
		}
		if policyEntity == nil {
			return nil, util.GrpcInternalError(log, "policy version does not exist", nil)
		}

		evaluatePolicyResponse, err := m.evaluatePolicy(ctx, policyAssignment.PolicyVersionId, policyEntity.RegoContent, occurrences)
		if err != nil {
			return nil, util.GrpcInternalError(log, fmt.Sprintf("error evaluating policy version %s", policyAssignment.PolicyVersionId), err)
		}

		if !evaluatePolicyResponse.Result.Pass {
			resourceEvaluation.Pass = false
		}

		policyEvaluation := &pb.PolicyEvaluation{
			Id:                   uuid.New().String(),
			ResourceEvaluationId: resourceEvaluation.Id,
			PolicyVersionId:      policyAssignment.PolicyVersionId,
			Pass:                 evaluatePolicyResponse.Result.Pass,
			Violations:           evaluatePolicyResponse.Result.Violations,
		}

		policyEvaluations = append(policyEvaluations, policyEvaluation)
		bulkRequestItems = append(bulkRequestItems, &esutil.BulkRequestItem{
			Operation:  esutil.BULK_CREATE,
			Message:    policyEvaluation,
			DocumentId: policyEvaluation.Id,
			Join: &esutil.EsJoin{
				Parent: resourceEvaluation.Id,
				Field:  evaluationDocumentJoinField,
				Name:   policyEvaluationRelationName,
			},
		})
	}

	response, err := m.esClient.Bulk(ctx, &esutil.BulkRequest{
		Index:   m.indexManager.AliasName(constants.EvaluationsDocumentKind, ""),
		Items:   bulkRequestItems,
		Refresh: m.esConfig.Refresh.String(),
	})
	if err != nil {
		return nil, util.GrpcInternalError(log, "error storing resource evaluation results", err)
	}
	if err = util.CheckBulkResponseErrors(response); err != nil {
		return nil, util.GrpcInternalError(log, "error storing resource evaluation results", err)
	}

	return &pb.ResourceEvaluationResult{
		ResourceEvaluation: resourceEvaluation,
		PolicyEvaluations:  policyEvaluations,
	}, nil
}

func (m *manager) GetResourceEvaluation(ctx context.Context, request *pb.GetResourceEvaluationRequest) (*pb.ResourceEvaluationResult, error) {
	log := m.logger.Named("GetResourceEvaluation").With(zap.String("id", request.Id))

	searchResponse, err := m.esClient.MultiSearch(ctx, &esutil.MultiSearchRequest{
		Index: m.indexManager.AliasName(constants.EvaluationsDocumentKind, ""),
		Searches: []*esutil.EsSearch{
			{
				Query: &filtering.Query{
					Term: &filtering.Term{
						"_id": request.Id,
					},
				},
			},
			{
				Query: &filtering.Query{
					HasParent: &filtering.HasParent{
						ParentType: resourceEvaluationRelationName,
						Query: &filtering.Query{
							Term: &filtering.Term{
								"_id": request.Id,
							},
						},
					},
				},
				Routing: request.Id,
			},
		},
	})
	if err != nil {
		return nil, util.GrpcInternalError(log, "error searching for resource evaluation", err)
	}

	resourceEvaluationResponse := searchResponse.Responses[0]

	if resourceEvaluationResponse.Hits.Total.Value == 0 {
		return nil, util.GrpcErrorWithCode(log, "resource evaluation not found", nil, codes.NotFound)
	}

	var resourceEvaluation pb.ResourceEvaluation
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(resourceEvaluationResponse.Hits.Hits[0].Source, &resourceEvaluation)
	if err != nil {
		return nil, util.GrpcInternalError(log, "error unmarshalling resource evaluation into json", err)
	}

	policyEvaluationResponse := searchResponse.Responses[1]

	var policyEvaluations []*pb.PolicyEvaluation
	for _, hit := range policyEvaluationResponse.Hits.Hits {
		var policyEvaluation pb.PolicyEvaluation
		err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(hit.Source, &policyEvaluation)
		if err != nil {
			return nil, util.GrpcInternalError(log, "error unmarshalling policy evaluation into json", err)
		}

		policyEvaluations = append(policyEvaluations, &policyEvaluation)
	}

	return &pb.ResourceEvaluationResult{
		ResourceEvaluation: &resourceEvaluation,
		PolicyEvaluations:  policyEvaluations,
	}, nil
}

func (m *manager) ListResourceEvaluations(ctx context.Context, request *pb.ListResourceEvaluationsRequest) (*pb.ListResourceEvaluationsResponse, error) {
	log := m.logger.Named("ListResourceEvaluations").With(zap.String("resourceUri", request.ResourceUri))

	if request.ResourceUri == "" {
		return nil, util.GrpcErrorWithCode(log, "resourceUri is required", nil, codes.InvalidArgument)
	}

	_, err := m.resourceManager.GetResourceVersion(ctx, request.ResourceUri)
	if err != nil {
		return nil, err
	}

	queries := filtering.Must{
		&filtering.Query{
			Term: &filtering.Term{
				"resourceVersion.version": request.ResourceUri,
			},
		},
	}

	if request.Filter != "" {
		filterQuery, err := m.filterer.ParseExpression(request.Filter)
		if err != nil {
			return nil, util.GrpcInternalError(log, "error parsing filter expression", err)
		}

		queries = append(queries, filterQuery)
	}

	searchRequest := &esutil.SearchRequest{
		Index: m.indexManager.AliasName(constants.EvaluationsDocumentKind, ""),
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
		return nil, util.GrpcInternalError(log, "error searching for resource evaluations", err)
	}

	if searchResponse.Hits.Total.Value == 0 {
		return &pb.ListResourceEvaluationsResponse{}, nil
	}

	var (
		resourceEvaluationResults []*pb.ResourceEvaluationResult
		policyEvaluationSearches  []*esutil.EsSearch
	)
	for _, hit := range searchResponse.Hits.Hits {
		var resourceEvaluation pb.ResourceEvaluation
		err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(hit.Source, &resourceEvaluation)
		if err != nil {
			return nil, util.GrpcInternalError(log, "error unmarshalling resource evaluation into json", err)
		}

		resourceEvaluationResults = append(resourceEvaluationResults, &pb.ResourceEvaluationResult{
			ResourceEvaluation: &resourceEvaluation,
		})
		policyEvaluationSearches = append(policyEvaluationSearches, &esutil.EsSearch{
			Query: &filtering.Query{
				HasParent: &filtering.HasParent{
					ParentType: resourceEvaluationRelationName,
					Query: &filtering.Query{
						Term: &filtering.Term{
							"_id": resourceEvaluation.Id,
						},
					},
				},
			},
			Routing: resourceEvaluation.Id,
		})
	}

	multiSearchResponse, err := m.esClient.MultiSearch(ctx, &esutil.MultiSearchRequest{
		Index:    m.indexManager.AliasName(constants.EvaluationsDocumentKind, ""),
		Searches: policyEvaluationSearches,
	})
	if err != nil {
		return nil, util.GrpcInternalError(log, "error searching for policy evaluations", err)
	}

	for i, response := range multiSearchResponse.Responses {
		var policyEvaluations []*pb.PolicyEvaluation
		for _, hit := range response.Hits.Hits {
			var poliyEvaluation pb.PolicyEvaluation
			err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(hit.Source, &poliyEvaluation)
			if err != nil {
				return nil, util.GrpcInternalError(log, "error unmarshalling policy evaluation into json", err)
			}

			policyEvaluations = append(policyEvaluations, &poliyEvaluation)
		}

		resourceEvaluationResults[i].PolicyEvaluations = policyEvaluations
	}

	return &pb.ListResourceEvaluationsResponse{
		ResourceEvaluations: resourceEvaluationResults,
		NextPageToken:       searchResponse.NextPageToken,
	}, nil
}

func (m *manager) EvaluatePolicy(ctx context.Context, request *pb.EvaluatePolicyRequest) (*pb.EvaluatePolicyResponse, error) {
	var err error
	log := m.logger.Named("EvaluatePolicy").With(zap.String("policy", request.Policy), zap.String("resource", request.ResourceUri))
	log.Debug("evaluate policy request received")

	if request.ResourceUri == "" {
		return nil, util.GrpcErrorWithCode(log, "resource uri is required", nil, codes.InvalidArgument)
	}

	policy, err := m.policyManager.GetPolicy(ctx, &pb.GetPolicyRequest{Id: request.Policy})
	if err != nil {
		return nil, util.GrpcInternalError(log, "error fetching policy", err)
	}

	// fetch occurrences from grafeas
	occurrences, _, err := m.grafeasExtensions.ListVersionedResourceOccurrences(ctx, request.ResourceUri, "", constants.MaxPageSize)
	if err != nil {
		return nil, util.GrpcInternalError(log, "error listing occurrences", err)
	}

	log.Debug("Occurrences found", zap.Any("occurrences", occurrences))

	// evaluate OPA policy
	evaluatePolicyResponse, err := m.evaluatePolicy(ctx, policy.Id, policy.Policy.RegoContent, occurrences)
	if err != nil {
		return nil, util.GrpcInternalError(log, "error evaluating policy", err)
	}

	log.Debug("Evaluate policy result", zap.Any("policy result", evaluatePolicyResponse))

	attestation := &pb.EvaluatePolicyResult{}
	attestation.Created = timestamppb.Now()
	if evaluatePolicyResponse.Result != nil {
		attestation.Pass = evaluatePolicyResponse.Result.Pass

		for _, violation := range evaluatePolicyResponse.Result.Violations {
			attestation.Violations = append(attestation.Violations, &pb.EvaluatePolicyViolation{
				Id:          violation.Id,
				Name:        violation.Name,
				Description: violation.Description,
				Message:     violation.Message,
				Link:        violation.Link,
				Pass:        violation.Pass,
			})
		}
	} else {
		evaluatePolicyResponse.Result = &opa.EvaluatePolicyResult{
			Pass: false,
		}
	}

	response := &pb.EvaluatePolicyResponse{
		Pass: evaluatePolicyResponse.Result.Pass,
		Result: []*pb.EvaluatePolicyResult{
			attestation,
		},
	}

	if evaluatePolicyResponse.Explanation != nil {
		response.Explanation = *evaluatePolicyResponse.Explanation
	}

	return response, nil
}

func (m *manager) evaluatePolicy(ctx context.Context, policyId, rego string, occurrences []*grafeas_go_proto.Occurrence) (*opa.EvaluatePolicyResponse, error) {
	// check OPA policy has been loaded, using the policy id
	initializePolicyErr := m.opa.InitializePolicy(policyId, rego)
	if initializePolicyErr != nil {
		return nil, fmt.Errorf("error initializing policy in OPA: %v", initializePolicyErr)
	}

	input, _ := protojson.Marshal(&pb.EvaluatePolicyInput{
		Occurrences: occurrences,
	})

	evaluatePolicyResponse, err := m.opa.EvaluatePolicy(rego, input)
	if err != nil {
		return nil, fmt.Errorf("error evaluating policy in OPA: %v", err)
	}

	return evaluatePolicyResponse, nil
}
