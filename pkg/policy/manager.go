package policy

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/open-policy-agent/opa/ast"
	"github.com/rode/es-index-manager/indexmanager"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/config"
	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	rodeProjectSlug           = "projects/rode"
	policiesDocumentKind      = "policies"
	maxPageSize               = 1000
	pitKeepAlive              = "5m"
	policyDocumentJoinField   = "join"
	policyRelationName        = "policy"
	policyVersionRelationName = "version"
)

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

	esClient      esutil.Client
	esConfig      *config.ElasticsearchConfig
	indexManager  indexmanager.IndexManager
	filterer      filtering.Filterer
	opa           opa.Client
	grafeasCommon grafeas_proto.GrafeasV1Beta1Client
}

func NewManager(
	logger *zap.Logger,
	esClient esutil.Client,
	esConfig *config.ElasticsearchConfig,
	indexManager indexmanager.IndexManager,
	filterer filtering.Filterer,
	opa opa.Client,
	grafeasCommon grafeas_proto.GrafeasV1Beta1Client,
) Manager {
	return &manager{
		logger:        logger,
		esClient:      esClient,
		esConfig:      esConfig,
		indexManager:  indexManager,
		filterer:      filterer,
		opa:           opa,
		grafeasCommon: grafeasCommon,
	}
}

func (m *manager) CreatePolicy(ctx context.Context, policy *pb.Policy) (*pb.Policy, error) {
	log := m.logger.Named("CreatePolicy")
	// Name field is a requirement
	if len(policy.Name) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "policy name not provided")
	}

	policyId := uuid.New().String()
	version := int32(1)
	policyVersionId := policyVersionId(policyId, version)
	currentTime := timestamppb.Now()

	policy.Id = policyId
	policy.Created = currentTime
	policy.Updated = currentTime
	policy.CurrentVersion = version

	policyVersion := policy.Policy
	policy.Policy = nil // remove current policy
	policyVersion.Version = version
	policyVersion.Message = "Initial creation"
	policyVersion.Created = currentTime

	// CheckPolicy before writing to elastic
	result, err := m.ValidatePolicy(ctx, &pb.ValidatePolicyRequest{Policy: policyVersion.RegoContent})
	if (err != nil) || !result.Compile {
		message := &pb.ValidatePolicyResponse{
			Policy:  policyVersion.RegoContent,
			Compile: false,
			Errors:  result.Errors,
		}
		s, _ := status.New(codes.InvalidArgument, "failed to compile the provided policy").WithDetails(message)
		return nil, s.Err()
	}

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

	var bulkErrors []error
	for _, item := range response.Items {
		if item.Create.Error != nil {
			bulkErrors = append(bulkErrors, fmt.Errorf("error creating policy [%d] %s: %s", item.Create.Status, item.Create.Error.Type, item.Create.Error.Reason))
		}
	}

	if len(bulkErrors) > 0 {
		return nil, createError(log, "failed to create policy", fmt.Errorf("bulk creation errors: %v", bulkErrors))
	}

	//policy := &pb.Policy{
	//	Id:      uuid.New().String(),
	//	Policy:  policyEntity,
	//	Created: currentTime,
	//	Updated: currentTime,
	//}
	//str, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(proto.MessageV2(policy))
	//if err != nil {
	//	return nil, createError(log, fmt.Sprintf("error marshalling %T to json", policy), err)
	//}
	//
	//res, err := m.esClient.Index(
	//	m.indexManager.AliasName(policiesDocumentKind, ""),
	//	bytes.NewReader(str),
	//	m.esClient.Index.WithContext(ctx),
	//	m.esClient.Index.WithRefresh(m.esConfig.Refresh.String()),
	//)
	//if err != nil {
	//	return nil, createError(log, "error sending request to elasticsearch", err)
	//}
	//if res.IsError() {
	//	return nil, createError(log, "error indexing document in elasticsearch", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	//}
	//
	policy.Policy = policyVersion

	return policy, nil
}

func (m *manager) GetPolicy(ctx context.Context, request *pb.GetPolicyRequest) (*pb.Policy, error) {
	log := m.logger.Named("GetPolicy").With(zap.String("id", request.Id))

	response, err := m.esClient.Get(ctx, &esutil.GetRequest{
		Index:      m.policiesAlias(),
		DocumentId: request.Id,
	})

	if err != nil {
		return nil, createError(log, "error", err)
	}

	if !response.Found {
		return nil, createErrorWithCode(log, "policy not found", err, codes.NotFound)
	}

	var policy pb.Policy
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(response.Source, &policy)
	if err != nil {
		return nil, createError(log, "error unmarshalling policy", err)
	}

	response, err = m.esClient.Get(ctx, &esutil.GetRequest{
		Join: &esutil.EsJoin{
			Field:  policyDocumentJoinField,
			Name:   policyVersionRelationName,
			Parent: request.Id,
		},
		Index:      m.policiesAlias(),
		DocumentId: policyVersionId(policy.Id, policy.CurrentVersion),
	})

	if err != nil {
		return nil, createError(log, "unable to determine current policy version", err)
	}

	if !response.Found {
		return nil, createError(log, "policy entity not found", nil)
	}

	log = log.With(zap.Int32("version", policy.CurrentVersion))

	var policyEntity pb.PolicyEntity
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(response.Source, &policyEntity)
	if err != nil {
		return nil, createError(log, "error unmarshalling policy version", err)
	}

	policy.Policy = &policyEntity

	return &policy, nil
}

func (m *manager) DeletePolicy(ctx context.Context, request *pb.DeletePolicyRequest) (*emptypb.Empty, error) {

	//response, err := m.esClient.Get(ctx, &esutil.GetRequest{
	//	Index: m.indexManager.AliasName(policiesDocumentKind, ""),
	//	DocumentId: request.Id,
	//})

	deletePolicyVersionsQuery := &filtering.Query{
		HasParent: &filtering.HasParent{
			ParentType: policyRelationName,
			Query: &filtering.Query{
				Term: &filtering.Term{
					"_id": request.Id,
				},
			},
		},
	}

	deletePolicyQuery := &filtering.Query{
		Term: &filtering.Term{
			"_id": request.Id,
		},
	}

	err := m.esClient.Delete(ctx, &esutil.DeleteRequest{
		Index: m.policiesAlias(),
		Search: &esutil.EsSearch{
			Query: &filtering.Query{
				Bool: &filtering.Bool{
					Should: &filtering.Should{
						deletePolicyVersionsQuery,
						deletePolicyQuery,
					},
				},
			},
		},
		Refresh: m.esConfig.Refresh.String(),
	})

	if err != nil {
		return nil, err
	}

	//log := m.logger.Named("DeletePolicy")
	//
	//search := &esutil.EsSearch{
	//	Query: &filtering.Query{
	//		Term: &filtering.Term{
	//			"id": deletePolicyRequest.Id,
	//		},
	//	},
	//}
	//
	//encodedBody, requestJSON := esutil.EncodeRequest(search)
	//log.Debug("es request payload", zap.Any("payload", requestJSON))
	//
	//res, err := m.esClient.DeleteByQuery(
	//	[]string{m.indexManager.IndexName(policiesDocumentKind, "")},
	//	encodedBody,
	//	m.esClient.DeleteByQuery.WithContext(ctx),
	//	m.esClient.DeleteByQuery.WithRefresh(withRefreshBool(m.esConfig.Refresh)),
	//)
	//if err != nil {
	//	return nil, createError(log, "error sending request to elasticsearch", err)
	//}
	//if res.IsError() {
	//	return nil, createError(log, "unexpected response from elasticsearch", err, zap.String("response", res.String()))
	//}
	//
	//var deletedResults esutil.EsDeleteResponse
	//if err = esutil.DecodeResponse(res.Body, &deletedResults); err != nil {
	//	return nil, createError(log, "error unmarshalling elasticsearch response", err)
	//}
	//
	//if deletedResults.Deleted == 0 {
	//	return nil, createError(log, "elasticsearch returned zero deleted documents", nil, zap.Any("response", deletedResults))
	//}
	//
	return &emptypb.Empty{}, nil
}

func (m *manager) ListPolicies(ctx context.Context, listPoliciesRequest *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	//log := m.logger.Named("List Policies")
	//hits, nextPageToken, err := m.genericList(ctx, log, &genericListOptions{
	//	index:         m.indexManager.AliasName(policiesDocumentKind, ""),
	//	filter:        listPoliciesRequest.Filter,
	//	pageSize:      listPoliciesRequest.PageSize,
	//	pageToken:     listPoliciesRequest.PageToken,
	//	sortDirection: esutil.EsSortOrderDescending,
	//	sortField:     "created",
	//})
	//
	//if err != nil {
	//	return nil, err
	//}
	//
	//var policies []*pb.Policy
	//for _, hit := range hits.Hits {
	//	hitLogger := log.With(zap.String("policy raw", string(hit.Source)))
	//
	//	policy := &pb.Policy{}
	//	err := protojson.Unmarshal(hit.Source, proto.MessageV2(policy))
	//	if err != nil {
	//		return nil, createError(hitLogger, "error converting _doc to policy", err)
	//	}
	//
	//	hitLogger.Debug("policy hit", zap.Any("policy", policy))
	//
	//	policies = append(policies, policy)
	//}
	//
	//return &pb.ListPoliciesResponse{
	//	Policies:      policies,
	//	NextPageToken: nextPageToken,
	//}, nil
	return nil, nil
}

func (m *manager) UpdatePolicy(ctx context.Context, updatePolicyRequest *pb.UpdatePolicyRequest) (*pb.Policy, error) {
	//log := m.logger.Named("Update Policy")
	//
	//// check if the policy exists
	//search := &esutil.EsSearch{
	//	Query: &filtering.Query{
	//		Term: &filtering.Term{
	//			"id": updatePolicyRequest.Id,
	//		},
	//	},
	//}
	//
	//policy := &pb.Policy{}
	//targetDocumentID, err := m.genericGet(ctx, log, search, m.indexManager.IndexName(policiesDocumentKind, ""), policy)
	//if err != nil {
	//	return nil, createError(log, "error fetching policy", err)
	//}
	//
	//log.Debug("field masks", zap.Any("response", updatePolicyRequest.UpdateMask.Paths))
	//// if one of the fields being updated is the rego policy, revalidate the policy
	//if contains(updatePolicyRequest.UpdateMask.Paths, "rego_content") {
	//	_, err = m.ValidatePolicy(ctx, &pb.ValidatePolicyRequest{Policy: updatePolicyRequest.Policy.RegoContent})
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	//
	//mask, err := fieldmask_utils.MaskFromPaths(updatePolicyRequest.UpdateMask.Paths, generator.CamelCase)
	//if err != nil {
	//	return nil, createError(log, "error mapping field masks", err)
	//}
	//
	//err = fieldmask_utils.StructToStruct(mask, updatePolicyRequest.Policy, policy.Policy)
	//if err != nil {
	//	return nil, createError(log, "error copying struct via field masks", err)
	//}
	//
	//policy.Updated = timestamppb.Now()
	//str, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(proto.MessageV2(policy))
	//if err != nil {
	//	return nil, createError(log, fmt.Sprintf("error marshalling %T to json", policy), err)
	//}
	//
	//res, err := m.esClient.Index(
	//	m.indexManager.IndexName(policiesDocumentKind, ""),
	//	bytes.NewReader(str),
	//	m.esClient.Index.WithDocumentID(targetDocumentID),
	//	m.esClient.Index.WithContext(ctx),
	//	m.esClient.Index.WithRefresh(m.esConfig.Refresh.String()),
	//)
	//if err != nil {
	//	return nil, createError(log, "error sending request to elasticsearch", err)
	//}
	//if res.IsError() {
	//	return nil, createError(log, "unexpected response from elasticsearch", err, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	//}
	//
	//return policy, nil
	return nil, nil
}

func (m *manager) ValidatePolicy(ctx context.Context, policy *pb.ValidatePolicyRequest) (*pb.ValidatePolicyResponse, error) {
	log := m.logger.Named("ValidatePolicy")

	if len(policy.Policy) == 0 {
		return nil, createError(log, "empty policy passed in", nil)
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

	internalErrors := validateRodeRequirementsForPolicy(mod, policy.Policy)
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
	listOccurrencesResponse, err := m.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{
		Parent:   rodeProjectSlug,
		Filter:   fmt.Sprintf(`resource.uri == "%s"`, request.ResourceUri),
		PageSize: maxPageSize,
	})
	if err != nil {
		return nil, createError(log, "error listing occurrences", err)
	}

	log.Debug("Occurrences found", zap.Any("occurrences", listOccurrencesResponse))

	input, _ := protojson.Marshal(proto.MessageV2(listOccurrencesResponse))

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

func (m *manager) policiesAlias() string {
	return m.indexManager.AliasName(policiesDocumentKind, "")
}

// TODO: move this and createError to a common util package
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
func validateRodeRequirementsForPolicy(mod *ast.Module, regoContent string) []error {
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
		err := errors.New("all policies must contain a \"pass\" block that returns a boolean result of the policy")
		errorsList = append(errorsList, err)
	}
	if !violationsBlockExists {
		err := errors.New("all policies must contain a \"violations\" block that returns a map of results")
		errorsList = append(errorsList, err)
	}
	if !returnFieldsExist {
		err := errors.New("all \"violations\" blocks must return a \"result\" that contains pass, id, message, and name fields")
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

//func (m *manager) genericGet(ctx context.Context, log *zap.Logger, search *esutil.EsSearch, index string, protoMessage interface{}) (string, error) {
//	encodedBody, requestJson := esutil.EncodeRequest(search)
//	log = log.With(zap.String("request", requestJson))
//
//	res, err := m.esClient.Search(
//		m.esClient.Search.WithContext(ctx),
//		m.esClient.Search.WithIndex(index),
//		m.esClient.Search.WithBody(encodedBody),
//	)
//	if err != nil {
//		return "", createError(log, "error sending request to elasticsearch", err)
//	}
//	if res.IsError() {
//		return "", createError(log, "error searching elasticsearch for document", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
//	}
//
//	var searchResults esutil.EsSearchResponse
//	if err := esutil.DecodeResponse(res.Body, &searchResults); err != nil {
//		return "", createError(log, "error unmarshalling elasticsearch response", err)
//	}
//
//	if searchResults.Hits.Total.Value == 0 {
//		log.Debug("document not found", zap.Any("search", search))
//		return "", status.Error(codes.NotFound, fmt.Sprintf("%T not found", protoMessage))
//	}
//
//	return searchResults.Hits.Hits[0].ID, protojson.Unmarshal(searchResults.Hits.Hits[0].Source, proto.MessageV2(protoMessage))
//}
//
//type genericListOptions struct {
//	index         string
//	filter        string
//	query         *esutil.EsSearch
//	pageSize      int32
//	pageToken     string
//	sortDirection esutil.EsSortOrder
//	sortField     string
//}
//
//func (m *manager) genericList(ctx context.Context, log *zap.Logger, options *genericListOptions) (*esutil.EsSearchResponseHits, string, error) {
//	body := &esutil.EsSearch{}
//	if options.query != nil {
//		body = options.query
//	}
//
//	if options.filter != "" {
//		log = log.With(zap.String("filter", options.filter))
//		filterQuery, err := m.filterer.ParseExpression(options.filter)
//		if err != nil {
//			return nil, "", createError(log, "error while parsing filter expression", err)
//		}
//
//		body.Query = filterQuery
//	}
//
//	if options.sortField != "" {
//		body.Sort = map[string]esutil.EsSortOrder{
//			options.sortField: options.sortDirection,
//		}
//	}
//
//	searchOptions := []func(*esapi.SearchRequest){
//		m.esClient.Search.WithContext(ctx),
//	}
//
//	var nextPageToken string
//	if options.pageToken != "" || options.pageSize != 0 { // handle pagination
//		next, extraSearchOptions, err := m.handlePagination(ctx, log, body, options.index, options.pageToken, options.pageSize)
//		if err != nil {
//			return nil, "", createError(log, "error while handling pagination", err)
//		}
//
//		nextPageToken = next
//		searchOptions = append(searchOptions, extraSearchOptions...)
//	} else {
//		searchOptions = append(searchOptions,
//			m.esClient.Search.WithIndex(options.index),
//			m.esClient.Search.WithSize(maxPageSize),
//		)
//	}
//
//	encodedBody, requestJson := esutil.EncodeRequest(body)
//	log = log.With(zap.String("request", requestJson))
//	log.Debug("performing search")
//
//	res, err := m.esClient.Search(
//		append(searchOptions, m.esClient.Search.WithBody(encodedBody))...,
//	)
//	if err != nil {
//		return nil, "", createError(log, "error sending request to elasticsearch", err)
//	}
//	if res.IsError() {
//		return nil, "", createError(log, "unexpected response from elasticsearch", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
//	}
//
//	var searchResults esutil.EsSearchResponse
//	if err := esutil.DecodeResponse(res.Body, &searchResults); err != nil {
//		return nil, "", createError(log, "error decoding elasticsearch response", err)
//	}
//
//	if options.pageToken != "" || options.pageSize != 0 { // if request is paginated, check for last page
//		_, from, err := esutil.ParsePageToken(nextPageToken)
//		if err != nil {
//			return nil, "", createError(log, "error parsing page token", err)
//		}
//
//		if from >= searchResults.Hits.Total.Value {
//			nextPageToken = ""
//		}
//	}
//	return searchResults.Hits, nextPageToken, nil
//}
//
//func (m *manager) handlePagination(ctx context.Context, log *zap.Logger, body *esutil.EsSearch, index, pageToken string, pageSize int32) (string, []func(*esapi.SearchRequest), error) {
//	log = log.With(zap.String("pageToken", pageToken), zap.Int32("pageSize", pageSize))
//
//	var (
//		pit  string
//		from int
//		err  error
//	)
//
//	// if no pageToken is specified, we need to create a new PIT
//	if pageToken == "" {
//		res, err := m.esClient.OpenPointInTime(
//			m.esClient.OpenPointInTime.WithContext(ctx),
//			m.esClient.OpenPointInTime.WithIndex(index),
//			m.esClient.OpenPointInTime.WithKeepAlive(pitKeepAlive),
//		)
//		if err != nil {
//			return "", nil, createError(log, "error sending request to elasticsearch", err)
//		}
//		if res.IsError() {
//			return "", nil, createError(log, "unexpected response from elasticsearch", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
//		}
//
//		var pitResponse esutil.ESPitResponse
//		if err = esutil.DecodeResponse(res.Body, &pitResponse); err != nil {
//			return "", nil, createError(log, "error decoding elasticsearch response", err)
//		}
//
//		pit = pitResponse.Id
//		from = 0
//	} else {
//		// get the PIT from the provided pageToken
//		pit, from, err = esutil.ParsePageToken(pageToken)
//		if err != nil {
//			return "", nil, createError(log, "error parsing page token", err)
//		}
//	}
//
//	body.Pit = &esutil.EsSearchPit{
//		Id:        pit,
//		KeepAlive: pitKeepAlive,
//	}
//
//	return esutil.CreatePageToken(pit, from+int(pageSize)), []func(*esapi.SearchRequest){
//		m.esClient.Search.WithSize(int(pageSize)),
//		m.esClient.Search.WithFrom(from),
//	}, err
//}

// contains returns a boolean describing whether or not a string slice contains a particular string
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func withRefreshBool(o config.RefreshOption) bool {
	if o == config.RefreshFalse {
		return false
	}
	return true
}

func policyVersionId(policyId string, version int32) string {
	return fmt.Sprintf("%s.%d", policyId, version)
}
