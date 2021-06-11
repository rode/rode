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
	"errors"
	"fmt"
	"github.com/rode/rode/pkg/constants"
	"github.com/rode/rode/pkg/grafeas"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/open-policy-agent/opa/ast"
	"github.com/rode/es-index-manager/indexmanager"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/config"
	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	policiesDocumentKind      = "policies"
	policyDocumentJoinField   = "join"
	policyRelationName        = "policy"
	policyVersionRelationName = "version"
)

//go:generate counterfeiter -generate

//counterfeiter:generate . Manager
type Manager interface {
	CreatePolicy(context.Context, *pb.Policy) (*pb.Policy, error)
	GetPolicy(context.Context, *pb.GetPolicyRequest) (*pb.Policy, error)
	DeletePolicy(context.Context, *pb.DeletePolicyRequest) (*emptypb.Empty, error)
	ListPolicies(context.Context, *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error)
	UpdatePolicy(context.Context, *pb.UpdatePolicyRequest) (*pb.Policy, error)
	EvaluatePolicy(context.Context, *pb.EvaluatePolicyRequest) (*pb.EvaluatePolicyResponse, error)
	ValidatePolicy(context.Context, *pb.ValidatePolicyRequest) (*pb.ValidatePolicyResponse, error)
}

type manager struct {
	logger *zap.Logger

	esClient          esutil.Client
	esConfig          *config.ElasticsearchConfig
	indexManager      indexmanager.IndexManager
	filterer          filtering.Filterer
	opa               opa.Client
	grafeasExtensions grafeas.Extensions
}

func NewManager(
	logger *zap.Logger,
	esClient esutil.Client,
	esConfig *config.ElasticsearchConfig,
	indexManager indexmanager.IndexManager,
	filterer filtering.Filterer,
	opa opa.Client,
	grafeasExtensions grafeas.Extensions,
) Manager {
	return &manager{
		logger:            logger,
		esClient:          esClient,
		esConfig:          esConfig,
		indexManager:      indexManager,
		filterer:          filterer,
		opa:               opa,
		grafeasExtensions: grafeasExtensions,
	}
}

var newUuid = uuid.New

func (m *manager) CreatePolicy(ctx context.Context, policy *pb.Policy) (*pb.Policy, error) {
	log := m.logger.Named("CreatePolicy")
	log.Debug("received request", zap.Any("request", policy))

	if len(policy.Name) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "policy name not provided")
	}

	if policy.Policy == nil {
		return nil, status.Errorf(codes.InvalidArgument, "policy entity not provided")
	}

	policyId := newUuid().String()
	log = log.With(zap.String("id", policyId))

	version := uint32(1)
	policyVersionId := policyVersionId(policyId, version)
	currentTime := timestamppb.Now()

	policy.Id = policyId
	policy.Created = currentTime
	policy.Updated = currentTime
	policy.CurrentVersion = version

	policyVersion := policy.Policy
	policy.Policy = nil // remove current policy
	policyVersion.Version = version
	policyVersion.Message = "Initial policy creation"
	policyVersion.Created = currentTime

	log.Debug("validating policy")
	if err := m.validatePolicy(ctx, log, policyVersion); err != nil {
		return nil, err
	}

	log.Debug("performing bulk request")
	response, err := m.esClient.Bulk(ctx, &esutil.BulkRequest{
		Index:   m.policiesAlias(),
		Refresh: m.esConfig.Refresh.String(),
		Items: []*esutil.BulkRequestItem{
			{
				Operation:  esutil.BULK_CREATE,
				Message:    policy,
				DocumentId: policyId,
				Join: &esutil.EsJoin{
					Field: policyDocumentJoinField,
					Name:  policyRelationName,
				},
			},
			{
				Operation:  esutil.BULK_CREATE,
				DocumentId: policyCounterId(policyId),
				// the counter document's version is used to track the current policy version, but the document itself is empty
				Message: &emptypb.Empty{},
			},
			{
				Operation:  esutil.BULK_CREATE,
				Message:    policyVersion,
				DocumentId: policyVersionId,
				Join: &esutil.EsJoin{
					Parent: policyId,
					Field:  policyDocumentJoinField,
					Name:   policyVersionRelationName,
				},
			},
		},
	})

	if err != nil {
		return nil, createError(log, "error creating policy", err)
	}

	if err := checkBulkResponseErrors(response); err != nil {
		return nil, createError(log, "policy creation failed", err)
	}

	policy.Policy = policyVersion

	log.Debug("successfully created policy")
	return policy, nil
}

func (m *manager) GetPolicy(ctx context.Context, request *pb.GetPolicyRequest) (*pb.Policy, error) {
	log := m.logger.Named("GetPolicy")
	log.Debug("received request")
	policyId, version, err := parsePolicyVersionId(request.Id)

	if err != nil {
		return nil, createErrorWithCode(log, "invalid policy id", err, codes.InvalidArgument)
	}

	log = log.With(zap.String("id", policyId))

	policy, err := m.getPolicy(ctx, log, policyId)
	if err != nil {
		return nil, err
	}

	if version == 0 {
		version = policy.CurrentVersion
	}

	log = log.With(zap.Uint32("version", version))

	response, err := m.esClient.Get(ctx, &esutil.GetRequest{
		Routing:    policyId,
		Index:      m.policiesAlias(),
		DocumentId: policyVersionId(policy.Id, version),
	})

	if err != nil {
		return nil, createError(log, "unable to determine current policy version", err)
	}

	if !response.Found {
		return nil, createError(log, "policy entity not found", nil)
	}

	var policyEntity pb.PolicyEntity
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(response.Source, &policyEntity)
	if err != nil {
		return nil, createError(log, "error unmarshalling policy version", err)
	}

	policy.Policy = &policyEntity

	return policy, nil
}

func (m *manager) DeletePolicy(ctx context.Context, request *pb.DeletePolicyRequest) (*emptypb.Empty, error) {
	log := m.logger.Named("DeletePolicy").With(zap.String("id", request.Id))
	log.Debug("received request")

	policy, err := m.getPolicy(ctx, log, request.Id)
	if err != nil {
		return nil, err
	}

	policy.Deleted = true

	if _, err := m.esClient.Update(ctx, &esutil.UpdateRequest{
		Index:      m.policiesAlias(),
		DocumentId: request.Id,
		Refresh:    m.esConfig.Refresh.String(),
		Message:    policy,
	}); err != nil {
		return nil, createError(log, "error deleting policy and its versions", err)
	}

	return &emptypb.Empty{}, nil
}

func (m *manager) ListPolicies(ctx context.Context, request *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	log := m.logger.Named("ListPolicies")
	log.Debug("received request", zap.Any("request", request))

	queries := filtering.Must{
		&filtering.Query{
			Term: &filtering.Term{
				policyDocumentJoinField: policyRelationName,
			},
		},
		// exclude policies that were soft-deleted
		&filtering.Query{
			Term: &filtering.Term{
				"deleted": "false",
			},
		},
	}

	if request.Filter != "" {
		filterQuery, err := m.filterer.ParseExpression(request.Filter)
		if err != nil {
			return nil, createError(log, "error creating filter query", err)
		}

		queries = append(queries, filterQuery)
	}

	searchRequest := &esutil.SearchRequest{
		Index: m.policiesAlias(),
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

	response, err := m.esClient.Search(ctx, searchRequest)
	if err != nil {
		return nil, createError(log, "error searching for policies", err)
	}

	if len(response.Hits.Hits) == 0 {
		return &pb.ListPoliciesResponse{}, nil
	}

	policies := make([]*pb.Policy, 0)
	versionItems := make([]*esutil.EsMultiGetItem, 0)
	for _, hit := range response.Hits.Hits {
		var policy pb.Policy
		err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(hit.Source, &policy)
		if err != nil {
			return nil, createError(log, "error unmarshalling policy", err)
		}

		policy.Id = hit.ID
		policies = append(policies, &policy)

		versionItems = append(versionItems, &esutil.EsMultiGetItem{
			Id:      policyVersionId(policy.Id, policy.CurrentVersion),
			Routing: policy.Id,
		})
	}

	versionsResponse, err := m.esClient.MultiGet(ctx, &esutil.MultiGetRequest{
		Index: m.policiesAlias(),
		Items: versionItems,
	})

	if err != nil {
		return nil, createError(log, "error fetching policy versions", err)
	}

	for i, document := range versionsResponse.Docs {
		if !document.Found {
			return nil, createError(log, fmt.Sprintf("missing policy version with id %s", document.Id), nil)
		}

		var entity pb.PolicyEntity
		err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(document.Source, &entity)
		if err != nil {
			return nil, createError(log, "error unmarshalling policy entity", err)
		}

		policies[i].Policy = &entity
	}

	return &pb.ListPoliciesResponse{
		Policies:      policies,
		NextPageToken: response.NextPageToken,
	}, nil

}

func (m *manager) UpdatePolicy(ctx context.Context, request *pb.UpdatePolicyRequest) (*pb.Policy, error) {
	log := m.logger.Named("UpdatePolicy").With(zap.String("policyId", request.Policy.Id))
	log.Debug("Received request")

	policyId := request.Policy.Id
	log.Debug("Fetching existing policy")
	currentPolicy, err := m.GetPolicy(ctx, &pb.GetPolicyRequest{Id: policyId})
	if err != nil {
		return nil, err
	}

	if currentPolicy.Deleted {
		return nil, createErrorWithCode(log, "cannot update a deleted policy", nil, codes.FailedPrecondition)
	}

	updatedPolicy := request.Policy
	currentTime := timestamppb.Now()
	// Update the existing policy to disallow clients from updating server-managed fields (e.g., timestamps)
	currentPolicy.Name = updatedPolicy.Name
	currentPolicy.Description = updatedPolicy.Description
	currentPolicy.Updated = currentTime
	policyVersion := currentPolicy.Policy

	bulkItems := []*esutil.BulkRequestItem{
		{
			Operation:  esutil.BULK_INDEX,
			Message:    currentPolicy,
			DocumentId: policyId,
			Join: &esutil.EsJoin{
				Field: policyDocumentJoinField,
				Name:  policyRelationName,
			},
		},
	}

	if hasPolicyContentChanges(currentPolicy, updatedPolicy) {
		policyVersion = updatedPolicy.Policy
		log.Debug("Policy content has changed. A new version will be created")

		if err := m.validatePolicy(ctx, log, policyVersion); err != nil {
			return nil, err
		}

		newVersion, err := m.incrementPolicyVersion(ctx, log, policyId)
		if err != nil {
			return nil, err
		}

		currentPolicy.CurrentVersion = newVersion
		policyVersion.Created = currentTime
		policyVersion.Version = newVersion

		if policyVersion.Message == "" {
			policyVersion.Message = fmt.Sprintf("Updated policy")
		}

		bulkItems = append(bulkItems, &esutil.BulkRequestItem{
			Operation:  esutil.BULK_CREATE,
			Message:    policyVersion,
			DocumentId: policyVersionId(policyId, newVersion),
			Join: &esutil.EsJoin{
				Parent: policyId,
				Field:  policyDocumentJoinField,
				Name:   policyVersionRelationName,
			},
		})
	}

	currentPolicy.Policy = nil // unset the embedded policy entity so that it isn't persisted to Elasticsearch
	log.Debug("Sending bulk request to Elasticsearch")
	response, err := m.esClient.Bulk(ctx, &esutil.BulkRequest{
		Index:   m.policiesAlias(),
		Refresh: m.esConfig.Refresh.String(),
		Items:   bulkItems,
	})

	if err != nil {
		return nil, createError(log, "error updating policy", err)
	}

	if err := checkBulkResponseErrors(response); err != nil {
		return nil, createError(log, "failed to update policy", err)
	}

	currentPolicy.Policy = policyVersion

	return currentPolicy, nil
}

func (m *manager) ValidatePolicy(_ context.Context, policy *pb.ValidatePolicyRequest) (*pb.ValidatePolicyResponse, error) {
	log := m.logger.Named("ValidatePolicy")

	if len(policy.Policy) == 0 {
		return nil, createErrorWithCode(log, "empty policy passed in", nil, codes.InvalidArgument)
	}

	// Generate the AST
	mod, err := ast.ParseModule("validate_module", policy.Policy)
	if err != nil {
		log.Debug("failed to parse the policy", zap.Any("policy", err))
		message := &pb.ValidatePolicyResponse{
			Policy:  policy.Policy,
			Compile: false,
			Errors:  []string{err.Error()},
		}
		s, _ := status.New(codes.InvalidArgument, "failed to parse the policy").WithDetails(message)
		return message, s.Err()
	}

	// Create a new compiler instance and compile the module
	c := ast.NewCompiler()

	mods := map[string]*ast.Module{
		"validate_module": mod,
	}

	if c.Compile(mods); c.Failed() {
		log.Debug("compilation error", zap.Any("payload", c.Errors))
		length := len(c.Errors)
		errorsList := make([]string, length)

		for i := range c.Errors {
			errorsList = append(errorsList, c.Errors[i].Error())
		}

		message := &pb.ValidatePolicyResponse{
			Policy:  policy.Policy,
			Compile: false,
			Errors:  errorsList,
		}
		s, _ := status.New(codes.InvalidArgument, "failed to compile the policy").WithDetails(message)
		return message, s.Err()

	}

	internalErrors := validateRodeRequirementsForPolicy(mod)
	if len(internalErrors) != 0 {
		var stringifiedErrorList []string
		for _, err := range internalErrors {
			stringifiedErrorList = append(stringifiedErrorList, err.Error())
		}
		message := &pb.ValidatePolicyResponse{
			Policy:  policy.Policy,
			Compile: false,
			Errors:  stringifiedErrorList,
		}
		s, _ := status.New(codes.InvalidArgument, "policy compiled successfully but is missing Rode required fields").WithDetails(message)
		return message, s.Err()
	}

	return &pb.ValidatePolicyResponse{
		Policy:  policy.Policy,
		Compile: true,
		Errors:  nil,
	}, nil
}

func (m *manager) EvaluatePolicy(ctx context.Context, request *pb.EvaluatePolicyRequest) (*pb.EvaluatePolicyResponse, error) {
	var err error
	log := m.logger.Named("EvaluatePolicy").With(zap.String("policy", request.Policy), zap.String("resource", request.ResourceUri))
	log.Debug("evaluate policy request received")

	if request.ResourceUri == "" {
		return nil, createErrorWithCode(log, "resource uri is required", nil, codes.InvalidArgument)
	}

	policy, err := m.GetPolicy(ctx, &pb.GetPolicyRequest{Id: request.Policy})
	if err != nil {
		return nil, createError(log, "error fetching policy", err)
	}

	// check OPA policy has been loaded, using the policy id
	err = m.opa.InitializePolicy(request.Policy, policy.Policy.RegoContent)
	if err != nil {
		return nil, createError(log, "error initializing policy in OPA", err)
	}

	// fetch occurrences from grafeas
	occurrences, _, err := m.grafeasExtensions.ListVersionedResourceOccurrences(ctx, request.ResourceUri, "", constants.MaxPageSize)
	if err != nil {
		return nil, createError(log, "error listing occurrences", err)
	}

	log.Debug("Occurrences found", zap.Any("occurrences", occurrences))

	input, _ := protojson.Marshal(&pb.EvaluatePolicyInput{
		Occurrences: occurrences,
	})

	evaluatePolicyResponse := &opa.EvaluatePolicyResponse{
		Result: &opa.EvaluatePolicyResult{
			Pass:       false,
			Violations: []*opa.EvaluatePolicyViolation{},
		},
	}
	// evaluate OPA policy
	evaluatePolicyResponse, err = m.opa.EvaluatePolicy(policy.Policy.RegoContent, input)
	if err != nil {
		return nil, createError(log, "error evaluating policy", err)
	}

	log.Debug("Evaluate policy result", zap.Any("policy result", evaluatePolicyResponse))

	attestation := &pb.EvaluatePolicyResult{}
	attestation.Created = timestamppb.Now()
	if evaluatePolicyResponse.Result != nil {
		attestation.Pass = evaluatePolicyResponse.Result.Pass

		for _, violation := range evaluatePolicyResponse.Result.Violations {
			attestation.Violations = append(attestation.Violations, &pb.EvaluatePolicyViolation{
				Id:          violation.ID,
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

func (m *manager) getPolicy(ctx context.Context, log *zap.Logger, id string) (*pb.Policy, error) {
	if id == "" {
		return nil, createErrorWithCode(log, "must specify policy id", nil, codes.InvalidArgument)
	}

	response, err := m.esClient.Get(ctx, &esutil.GetRequest{
		Index:      m.policiesAlias(),
		DocumentId: id,
	})

	if err != nil {
		return nil, createError(log, "error fetching policy", err)
	}

	if !response.Found {
		return nil, createErrorWithCode(log, "policy not found", err, codes.NotFound)
	}

	var policy pb.Policy
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(response.Source, &policy)
	if err != nil {
		return nil, createError(log, "error unmarshalling policy", err)
	}

	return &policy, nil
}

func (m *manager) validatePolicy(ctx context.Context, log *zap.Logger, policy *pb.PolicyEntity) error {
	result, err := m.ValidatePolicy(ctx, &pb.ValidatePolicyRequest{Policy: policy.RegoContent})

	message := &pb.ValidatePolicyResponse{
		Policy: policy.RegoContent,
	}

	if result != nil && !result.Compile {
		message.Errors = result.Errors
	}

	if err != nil || (result != nil && !result.Compile) {
		s, _ := status.New(codes.InvalidArgument, "failed to compile the provided policy").WithDetails(message)
		log.Error("failed to compile policy", zap.Error(s.Err()))

		return s.Err()
	}

	return nil
}

func (m *manager) policiesAlias() string {
	return m.indexManager.AliasName(policiesDocumentKind, "")
}

func (m *manager) incrementPolicyVersion(ctx context.Context, log *zap.Logger, policyId string) (uint32, error) {
	updateResponse, err := m.esClient.Update(ctx, &esutil.UpdateRequest{
		Index:      m.policiesAlias(),
		DocumentId: policyCounterId(policyId),
		Refresh:    m.esConfig.Refresh.String(),
		Message:    &emptypb.Empty{},
	})

	if err != nil {
		return 0, createError(log, "error updating policy version counter", err)
	}

	return uint32(updateResponse.Version), nil
}

func createErrorWithCode(log *zap.Logger, message string, err error, code codes.Code, fields ...zap.Field) error {
	if err == nil {
		log.Error(message, fields...)
		return status.Errorf(code, "%s", message)
	}

	log.Error(message, append(fields, zap.Error(err))...)
	return status.Errorf(code, "%s: %s", message, err)
}

func createError(log *zap.Logger, message string, err error, fields ...zap.Field) error {
	return createErrorWithCode(log, message, err, codes.Internal, fields...)
}

// validateRodeRequirementsForPolicy ensures that these two rules are followed:
// 1. A policy is expected to return a pass that is simply a boolean representing the AND of all rules.
// 2. A policy is expected to return an array of violations that are maps containing a description id message name pass. pass here is what will be used to determine the overall pass.
func validateRodeRequirementsForPolicy(mod *ast.Module) []error {
	errorsList := []error{}
	// policy must contains a pass block somewhere in the code
	passBlockExists := len(mod.RuleSet("pass")) > 0
	// policy must contains a violations block somewhere in the code
	violationsBlockExists := len(mod.RuleSet("violations")) > 0
	// missing field from result return response
	returnFieldsExist := false

	violations := mod.RuleSet("violations")

	for x, r := range violations {
		if r.Head.Key == nil || r.Head.Key.Value.String() != "result" {
			// found a violations block that does not return a result object, break immediately
			break
		}
		if !validateResultTermsInBody(r.Body) {
			break
		}
		// if the end of the loop is reached, all violations blocks have the required fields
		if x == len(violations)-1 {
			returnFieldsExist = true
		}
	}

	if !passBlockExists {
		err := errors.New(`all policies must contain a "pass" block that returns a boolean result of the policy`)
		errorsList = append(errorsList, err)
	}
	if !violationsBlockExists {
		err := errors.New(`all policies must contain a "violations" block that returns a map of results`)
		errorsList = append(errorsList, err)
	}
	if !returnFieldsExist {
		err := errors.New(`all "violations" blocks must return a "result" that contains pass, id, message, and name fields`)
		errorsList = append(errorsList, err)
	}

	return errorsList
}

func validateResultTermsInBody(body ast.Body) bool {
	for _, b := range body {
		// find the assignment
		if b.Operator().String() == "assign" || b.Operator().String() == "eq" {
			terms := (b.Terms).([]*ast.Term)
			for i, t := range terms {
				object, ok := t.Value.(ast.Object)
				if ok {
					// look at the previous terms to check that it was assigned to result
					if terms[i-1].String() == "result" {
						keyMap := make(map[string]interface{})
						for _, key := range object.Keys() {
							keyVal, err := strconv.Unquote(key.Value.String())
							if err != nil {
								keyVal = key.Value.String()
							}
							keyMap[keyVal] = object.Get(key)
						}

						_, passExists := keyMap["pass"]
						_, nameExists := keyMap["name"]
						_, idExists := keyMap["id"]
						_, messageExists := keyMap["message"]

						if !passExists || !nameExists || !idExists || !messageExists {
							return false
						}
					}
				} else {
					continue
				}
			}
		}
	}
	return true
}

func policyVersionId(policyId string, version uint32) string {
	return fmt.Sprintf("%s.%d", policyId, version)
}

func policyCounterId(policyId string) string {
	return fmt.Sprintf("%s.counter", policyId)
}

// Parses a policy or policy version id
// If the separator is present, treat the id as containing the policy id and the version
func parsePolicyVersionId(id string) (string, uint32, error) {
	if !strings.ContainsRune(id, '.') {
		return id, 0, nil
	}

	pieces := strings.Split(id, ".")
	if len(pieces) != 2 {
		return "", 0, errors.New("invalid policy version id")
	}

	version, err := strconv.Atoi(pieces[1])
	if err != nil {
		return "", 0, err
	}

	return pieces[0], uint32(version), nil
}

func checkBulkResponseErrors(response *esutil.EsBulkResponse) error {
	var bulkErrors []error
	for _, item := range response.Items {
		if item.Create != nil && item.Create.Error != nil {
			bulkErrors = append(bulkErrors, fmt.Errorf("error creating new policy version [%d] %s: %s", item.Create.Status, item.Create.Error.Type, item.Create.Error.Reason))
		} else if item.Index != nil && item.Index.Error != nil {
			bulkErrors = append(bulkErrors, fmt.Errorf("error updating policy [%d] %s: %s", item.Index.Status, item.Index.Error.Type, item.Index.Error.Reason))
		}
	}

	if len(bulkErrors) > 0 {
		return fmt.Errorf("errors: %v", bulkErrors)
	}

	return nil
}

func hasPolicyContentChanges(currentPolicy, updatedPolicy *pb.Policy) bool {
	updated := updatedPolicy.Policy
	current := currentPolicy.Policy

	return !(updated.RegoContent == current.RegoContent && updated.SourcePath == current.SourcePath)
}
