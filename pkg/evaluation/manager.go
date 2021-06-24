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
	EvaluateResource(context.Context, *pb.EvaluateResourceRequest) (*pb.ResourceEvaluationResult, error)
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
	esClient                esutil.Client
	policyManager           policy.Manager
	policyGroupManager      policy.PolicyGroupManager
	policyAssignmentManager policy.AssignmentManager
	grafeasExtensions       grafeas.Extensions
	opa                     opa.Client
	resourceManager         resource.Manager
	indexManager            indexmanager.IndexManager
}

func NewManager(
	logger *zap.Logger,
	esClient esutil.Client,
	policyManager policy.Manager,
	policyGroupManager policy.PolicyGroupManager,
	policyAssignmentManager policy.AssignmentManager,
	grafeasExtensions grafeas.Extensions,
	opa opa.Client,
	resourceManager resource.Manager,
	indexManager indexmanager.IndexManager,
) Manager {
	return &manager{
		logger:                  logger,
		esClient:                esClient,
		policyManager:           policyManager,
		policyGroupManager:      policyGroupManager,
		policyAssignmentManager: policyAssignmentManager,
		grafeasExtensions:       grafeasExtensions,
		opa:                     opa,
		resourceManager:         resourceManager,
		indexManager:            indexManager,
	}
}

func (m *manager) EvaluateResource(ctx context.Context, request *pb.EvaluateResourceRequest) (*pb.ResourceEvaluationResult, error) {
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
	if resourceVersion == nil {
		return nil, util.GrpcErrorWithCode(log, "resource version not found", nil, codes.FailedPrecondition)
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
	occurrences, _, err := m.grafeasExtensions.ListVersionedResourceOccurrences(ctx, request.ResourceUri, "", constants.MaxPageSize)
	if err != nil {
		return nil, util.GrpcInternalError(log, "error listing occurrences", err)
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
		Index: m.indexManager.AliasName(constants.EvaluationsDocumentKind, ""),
		Items: bulkRequestItems,
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
	panic("implement me")
}

func (m *manager) ListResourceEvaluations(ctx context.Context, request *pb.ListResourceEvaluationsRequest) (*pb.ListResourceEvaluationsResponse, error) {
	panic("implement me")
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

	return &pb.EvaluatePolicyResponse{
		Pass: evaluatePolicyResponse.Result.Pass,
		Result: []*pb.EvaluatePolicyResult{
			attestation,
		},
		Explanation: *evaluatePolicyResponse.Explanation,
	}, nil
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
