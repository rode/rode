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

package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/rode/rode/pkg/resource"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"net/http"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/google/uuid"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"

	"github.com/golang/protobuf/proto"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	"github.com/open-policy-agent/opa/ast"
	"github.com/rode/rode/config"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	rodeProjectSlug                        = "projects/rode"
	rodeElasticsearchOccurrencesAlias      = "grafeas-rode-occurrences"
	rodeElasticsearchPoliciesIndex         = "rode-v1alpha1-policies"
	rodeElasticsearchGenericResourcesIndex = "rode-v1alpha1-generic-resources"
	maxPageSize                            = 1000
	pitKeepAlive                           = "5m"
)

// NewRodeServer constructor for rodeServer
func NewRodeServer(
	logger *zap.Logger,
	grafeasCommon grafeas_proto.GrafeasV1Beta1Client,
	grafeasProjects grafeas_project_proto.ProjectsClient,
	opa opa.Client,
	esClient *elasticsearch.Client,
	filterer filtering.Filterer,
	elasticsearchConfig *config.ElasticsearchConfig,
	resourceManager resource.Manager,
) (pb.RodeServer, error) {
	rodeServer := &rodeServer{
		logger:              logger,
		grafeasCommon:       grafeasCommon,
		grafeasProjects:     grafeasProjects,
		opa:                 opa,
		esClient:            esClient,
		filterer:            filterer,
		elasticsearchConfig: elasticsearchConfig,
		resouceManager:      resourceManager,
	}
	if err := rodeServer.initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize rode server: %s", err)
	}

	return rodeServer, nil
}

type rodeServer struct {
	pb.UnimplementedRodeServer
	logger              *zap.Logger
	esClient            *elasticsearch.Client
	filterer            filtering.Filterer
	grafeasCommon       grafeas_proto.GrafeasV1Beta1Client
	grafeasProjects     grafeas_project_proto.ProjectsClient
	opa                 opa.Client
	elasticsearchConfig *config.ElasticsearchConfig
	resouceManager      resource.Manager
}

func (r *rodeServer) BatchCreateOccurrences(ctx context.Context, occurrenceRequest *pb.BatchCreateOccurrencesRequest) (*pb.BatchCreateOccurrencesResponse, error) {
	log := r.logger.Named("BatchCreateOccurrences")
	log.Debug("received request", zap.Any("BatchCreateOccurrencesRequest", occurrenceRequest))

	if err := r.resouceManager.BatchCreateGenericResources(ctx, occurrenceRequest); err != nil {
		return nil, createError(log, "error creating generic resources", err)
	}

	occurrenceResponse, err := r.grafeasCommon.BatchCreateOccurrences(ctx, &grafeas_proto.BatchCreateOccurrencesRequest{
		Parent:      rodeProjectSlug,
		Occurrences: occurrenceRequest.GetOccurrences(),
	})
	if err != nil {
		return nil, createError(log, "error creating occurrences", err)
	}

	return &pb.BatchCreateOccurrencesResponse{
		Occurrences: occurrenceResponse.GetOccurrences(),
	}, nil
}

func (r *rodeServer) EvaluatePolicy(ctx context.Context, request *pb.EvaluatePolicyRequest) (*pb.EvaluatePolicyResponse, error) {
	var err error
	log := r.logger.Named("EvaluatePolicy").With(zap.String("policy", request.Policy), zap.String("resource", request.ResourceUri))
	log.Debug("evaluate policy request received")

	if request.ResourceUri == "" {
		return nil, createErrorWithCode(log, "resource uri is required", nil, codes.InvalidArgument)
	}

	policy, err := r.GetPolicy(ctx, &pb.GetPolicyRequest{Id: request.Policy})
	if err != nil {
		return nil, createError(log, "error fetching policy", err)
	}

	// check OPA policy has been loaded, using the policy id
	err = r.opa.InitializePolicy(request.Policy, policy.Policy.RegoContent)
	if err != nil {
		return nil, createError(log, "error initializing policy in OPA", err)
	}

	// fetch occurrences from grafeas
	listOccurrencesResponse, err := r.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{
		Parent:   rodeProjectSlug,
		Filter:   fmt.Sprintf(`"resource.uri" == "%s"`, request.ResourceUri),
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
	evaluatePolicyResponse, err = r.opa.EvaluatePolicy(policy.Policy.RegoContent, input)
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

func (r *rodeServer) ListResources(ctx context.Context, request *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	log := r.logger.Named("ListResources")
	log.Debug("received request", zap.Any("ListResourcesRequest", request))

	hits, nextPageToken, err := r.genericList(ctx, log, &genericListOptions{
		index:     rodeElasticsearchOccurrencesAlias,
		filter:    request.Filter,
		pageSize:  request.PageSize,
		pageToken: request.PageToken,
		query: &esutil.EsSearch{
			Collapse: &esutil.EsSearchCollapse{
				Field: "resource.uri",
			},
		},
		sortDirection: esutil.EsSortOrderAscending,
		sortField:     "resource.uri",
	})

	if err != nil {
		return nil, err
	}

	var resources []*grafeas_proto.Resource
	for _, hit := range hits.Hits {
		occurrence := &grafeas_proto.Occurrence{}
		err := protojson.Unmarshal(hit.Source, proto.MessageV2(occurrence))
		if err != nil {
			return nil, createError(log, "error unmarshalling search result", err)
		}

		resources = append(resources, occurrence.Resource)
	}

	return &pb.ListResourcesResponse{
		Resources:     resources,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *rodeServer) ListGenericResources(ctx context.Context, request *pb.ListGenericResourcesRequest) (*pb.ListGenericResourcesResponse, error) {
	log := r.logger.Named("ListGenericResources")
	log.Debug("received request", zap.Any("request", request))

	hits, nextPageToken, err := r.genericList(ctx, log, &genericListOptions{
		index:         rodeElasticsearchGenericResourcesIndex,
		filter:        request.Filter,
		pageSize:      request.PageSize,
		pageToken:     request.PageToken,
		sortDirection: esutil.EsSortOrderAscending,
		sortField:     "name",
	})

	if err != nil {
		return nil, err
	}

	var resources []*pb.GenericResource
	for _, hit := range hits.Hits {
		resources = append(resources, &pb.GenericResource{Name: hit.ID})
	}

	return &pb.ListGenericResourcesResponse{
		GenericResources: resources,
		NextPageToken:    nextPageToken,
	}, nil
}

type indexSetting struct {
	index      string
	properties map[string]interface{}
}

func (r *rodeServer) initialize(ctx context.Context) error {
	log := r.logger.Named("initialize")

	_, err := r.grafeasProjects.GetProject(ctx, &grafeas_project_proto.GetProjectRequest{Name: rodeProjectSlug})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			_, err := r.grafeasProjects.CreateProject(ctx, &grafeas_project_proto.CreateProjectRequest{Project: &grafeas_project_proto.Project{Name: rodeProjectSlug}})
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

	indexSettings := []indexSetting{
		{index: rodeElasticsearchPoliciesIndex, properties: map[string]interface{}{
			"created": map[string]interface{}{
				"type": "date",
			},
		}},
		{index: rodeElasticsearchGenericResourcesIndex, properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type": "keyword",
			},
		}},
	}

	for _, settings := range indexSettings {
		if err := r.createIndex(ctx, settings); err != nil {

			return fmt.Errorf("error creating index %s: %s", settings, err)
		}
	}

	return nil
}

func (r *rodeServer) ListVersionedResourceOccurrences(ctx context.Context, request *pb.ListVersionedResourceOccurrencesRequest) (*pb.ListVersionedResourceOccurrencesResponse, error) {
	log := r.logger.Named("ListVersionedResourceOccurrences")
	log.Debug("received request", zap.Any("ListVersionedResourceOccurrencesRequest", request))

	resourceUri := request.ResourceUri
	if resourceUri == "" {
		return nil, createErrorWithCode(log, "invalid request", errors.New("must set resource_uri"), codes.InvalidArgument)
	}

	log.Debug("listing build occurrences")
	buildOccurrences, err := r.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{
		Parent:   rodeProjectSlug,
		PageSize: maxPageSize,
		Filter:   fmt.Sprintf(`kind == "BUILD" && (resource.uri == "%[1]s" || build.provenance.builtArtifacts.nestedFilter(id == "%[1]s"))`, resourceUri),
	})

	if err != nil {
		return nil, createError(log, "error fetching build occurrences", err)
	}

	resourceUris := map[string]string{
		resourceUri: resourceUri,
	}
	for _, occurrence := range buildOccurrences.Occurrences {
		resourceUris[occurrence.Resource.Uri] = occurrence.Resource.Uri
		for _, artifact := range occurrence.GetBuild().GetProvenance().BuiltArtifacts {
			resourceUris[artifact.Id] = artifact.Id
		}
	}

	var resourceFilters []string
	for k := range resourceUris {
		resourceFilters = append(resourceFilters, fmt.Sprintf(`resource.uri == "%s"`, k))
	}

	filter := strings.Join(resourceFilters, " || ")
	log.Debug("listing occurrences", zap.String("filter", filter))
	allOccurrences, err := r.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{
		Parent:    rodeProjectSlug,
		Filter:    filter,
		PageSize:  request.PageSize,
		PageToken: request.PageToken,
	})

	if err != nil {
		return nil, createError(log, "error listing occurrences", err)
	}

	return &pb.ListVersionedResourceOccurrencesResponse{
		Occurrences:   allOccurrences.Occurrences,
		NextPageToken: allOccurrences.NextPageToken,
	}, nil
}

func (r *rodeServer) ListOccurrences(ctx context.Context, occurrenceRequest *pb.ListOccurrencesRequest) (*pb.ListOccurrencesResponse, error) {
	log := r.logger.Named("ListOccurrences")
	log.Debug("received request", zap.Any("ListOccurrencesRequest", occurrenceRequest))

	request := &grafeas_proto.ListOccurrencesRequest{
		Parent:    rodeProjectSlug,
		Filter:    occurrenceRequest.Filter,
		PageToken: occurrenceRequest.PageToken,
		PageSize:  occurrenceRequest.PageSize,
	}

	listOccurrencesResponse, err := r.grafeasCommon.ListOccurrences(ctx, request)
	if err != nil {
		return nil, createError(log, "error listing occurrences", err)
	}

	return &pb.ListOccurrencesResponse{
		Occurrences:   listOccurrencesResponse.GetOccurrences(),
		NextPageToken: listOccurrencesResponse.GetNextPageToken(),
	}, nil
}

func (r *rodeServer) UpdateOccurrence(ctx context.Context, occurrenceRequest *pb.UpdateOccurrenceRequest) (*grafeas_proto.Occurrence, error) {
	log := r.logger.Named("UpdateOccurrence")
	log.Debug("received request", zap.Any("UpdateOccurrenceRequest", occurrenceRequest))

	name := fmt.Sprintf("projects/rode/occurrences/%s", occurrenceRequest.Id)

	if occurrenceRequest.Occurrence.Name != name {
		log.Error("occurrence name does not contain the occurrence id", zap.String("occurrenceName", occurrenceRequest.Occurrence.Name), zap.String("id", occurrenceRequest.Id))
		return nil, status.Error(codes.InvalidArgument, "occurrence name does not contain the occurrence id")
	}

	updatedOccurrence, err := r.grafeasCommon.UpdateOccurrence(ctx, &grafeas_proto.UpdateOccurrenceRequest{
		Name:       name,
		Occurrence: occurrenceRequest.Occurrence,
		UpdateMask: occurrenceRequest.UpdateMask,
	})
	if err != nil {
		return nil, createError(log, "error updating occurrence", err)
	}

	return updatedOccurrence, nil
}

func (r *rodeServer) ValidatePolicy(ctx context.Context, policy *pb.ValidatePolicyRequest) (*pb.ValidatePolicyResponse, error) {
	log := r.logger.Named("ValidatePolicy")

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

func (r *rodeServer) CreatePolicy(ctx context.Context, policyEntity *pb.PolicyEntity) (*pb.Policy, error) {
	// TODO maybe check if it already exists (if we think a unique name is required)

	log := r.logger.Named("CreatePolicy")
	// Name field is a requirement
	if len(policyEntity.Name) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "policy name not provided")
	}

	// CheckPolicy before writing to elastic
	result, err := r.ValidatePolicy(ctx, &pb.ValidatePolicyRequest{Policy: policyEntity.RegoContent})
	if (err != nil) || !result.Compile {
		message := &pb.ValidatePolicyResponse{
			Policy:  policyEntity.RegoContent,
			Compile: false,
			Errors:  result.Errors,
		}
		s, _ := status.New(codes.InvalidArgument, "failed to compile the provided policy").WithDetails(message)
		return nil, s.Err()
	}
	currentTime := timestamppb.Now()
	policy := &pb.Policy{
		Id:      uuid.New().String(),
		Policy:  policyEntity,
		Created: currentTime,
		Updated: currentTime,
	}
	str, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(proto.MessageV2(policy))
	if err != nil {
		return nil, createError(log, fmt.Sprintf("error marshalling %T to json", policy), err)
	}

	res, err := r.esClient.Index(
		rodeElasticsearchPoliciesIndex,
		bytes.NewReader(str),
		r.esClient.Index.WithContext(ctx),
		r.esClient.Index.WithRefresh(r.elasticsearchConfig.Refresh.String()),
	)
	if err != nil {
		return nil, createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return nil, createError(log, "error indexing document in elasticsearch", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	}

	return policy, nil
}

func (r *rodeServer) GetPolicy(ctx context.Context, getPolicyRequest *pb.GetPolicyRequest) (*pb.Policy, error) {
	log := r.logger.Named("GetPolicy")

	search := &esutil.EsSearch{
		Query: &filtering.Query{
			Term: &filtering.Term{
				"id": getPolicyRequest.Id,
			},
		},
	}

	policy := &pb.Policy{}

	_, err := r.genericGet(ctx, log, search, rodeElasticsearchPoliciesIndex, policy)
	if err != nil {
		return nil, createError(log, "error getting policy", err)
	}

	return policy, nil

}

func (r *rodeServer) DeletePolicy(ctx context.Context, deletePolicyRequest *pb.DeletePolicyRequest) (*emptypb.Empty, error) {
	log := r.logger.Named("DeletePolicy")

	search := &esutil.EsSearch{
		Query: &filtering.Query{
			Term: &filtering.Term{
				"id": deletePolicyRequest.Id,
			},
		},
	}

	encodedBody, requestJSON := esutil.EncodeRequest(search)
	log.Debug("es request payload", zap.Any("payload", requestJSON))

	res, err := r.esClient.DeleteByQuery(
		[]string{rodeElasticsearchPoliciesIndex},
		encodedBody,
		r.esClient.DeleteByQuery.WithContext(ctx),
		r.esClient.DeleteByQuery.WithRefresh(withRefreshBool(r.elasticsearchConfig.Refresh)),
	)
	if err != nil {
		return nil, createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return nil, createError(log, "unexpected response from elasticsearch", err, zap.String("response", res.String()))
	}

	var deletedResults esutil.EsDeleteResponse
	if err = esutil.DecodeResponse(res.Body, &deletedResults); err != nil {
		return nil, createError(log, "error unmarshalling elasticsearch response", err)
	}

	if deletedResults.Deleted == 0 {
		return nil, createError(log, "elasticsearch returned zero deleted documents", nil, zap.Any("response", deletedResults))
	}

	return &emptypb.Empty{}, nil
}

func (r *rodeServer) ListPolicies(ctx context.Context, listPoliciesRequest *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	log := r.logger.Named("List Policies")
	hits, nextPageToken, err := r.genericList(ctx, log, &genericListOptions{
		index:         rodeElasticsearchPoliciesIndex,
		filter:        listPoliciesRequest.Filter,
		pageSize:      listPoliciesRequest.PageSize,
		pageToken:     listPoliciesRequest.PageToken,
		sortDirection: esutil.EsSortOrderDescending,
		sortField:     "created",
	})

	if err != nil {
		return nil, err
	}

	var policies []*pb.Policy
	for _, hit := range hits.Hits {
		hitLogger := log.With(zap.String("policy raw", string(hit.Source)))

		policy := &pb.Policy{}
		err := protojson.Unmarshal(hit.Source, proto.MessageV2(policy))
		if err != nil {
			return nil, createError(hitLogger, "error converting _doc to policy", err)
		}

		hitLogger.Debug("policy hit", zap.Any("policy", policy))

		policies = append(policies, policy)
	}

	return &pb.ListPoliciesResponse{
		Policies:      policies,
		NextPageToken: nextPageToken,
	}, nil
}

// UpdatePolicy will update only the fields provided by the user
func (r *rodeServer) UpdatePolicy(ctx context.Context, updatePolicyRequest *pb.UpdatePolicyRequest) (*pb.Policy, error) {
	log := r.logger.Named("Update Policy")

	// check if the policy exists
	search := &esutil.EsSearch{
		Query: &filtering.Query{
			Term: &filtering.Term{
				"id": updatePolicyRequest.Id,
			},
		},
	}

	policy := &pb.Policy{}
	targetDocumentID, err := r.genericGet(ctx, log, search, rodeElasticsearchPoliciesIndex, policy)
	if err != nil {
		return nil, createError(log, "error fetching policy", err)
	}

	log.Debug("field masks", zap.Any("response", updatePolicyRequest.UpdateMask.Paths))
	// if one of the fields being updated is the rego policy, revalidate the policy
	if contains(updatePolicyRequest.UpdateMask.Paths, "rego_content") {
		_, err = r.ValidatePolicy(ctx, &pb.ValidatePolicyRequest{Policy: updatePolicyRequest.Policy.RegoContent})
		if err != nil {
			return nil, err
		}
	}

	m, err := fieldmask_utils.MaskFromPaths(updatePolicyRequest.UpdateMask.Paths, generator.CamelCase)
	if err != nil {
		return nil, createError(log, "error mapping field masks", err)
	}

	err = fieldmask_utils.StructToStruct(m, updatePolicyRequest.Policy, policy.Policy)
	if err != nil {
		return nil, createError(log, "error copying struct via field masks", err)
	}

	policy.Updated = timestamppb.Now()
	str, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(proto.MessageV2(policy))
	if err != nil {
		return nil, createError(log, fmt.Sprintf("error marshalling %T to json", policy), err)
	}

	res, err := r.esClient.Index(
		rodeElasticsearchPoliciesIndex,
		bytes.NewReader(str),
		r.esClient.Index.WithDocumentID(targetDocumentID),
		r.esClient.Index.WithContext(ctx),
		r.esClient.Index.WithRefresh(r.elasticsearchConfig.Refresh.String()),
	)
	if err != nil {
		return nil, createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return nil, createError(log, "unexpected response from elasticsearch", err, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	}

	return policy, nil
}

func (r *rodeServer) RegisterCollector(ctx context.Context, registerCollectorRequest *pb.RegisterCollectorRequest) (*pb.RegisterCollectorResponse, error) {
	log := r.logger.Named("RegisterCollector")

	if registerCollectorRequest.Id == "" {
		return nil, createErrorWithCode(log, "collector ID is required", nil, codes.InvalidArgument)
	}

	if len(registerCollectorRequest.Notes) == 0 {
		return &pb.RegisterCollectorResponse{}, nil
	}

	// build collection of notes that potentially need to be created
	notesWithIds := make(map[string]*grafeas_proto.Note)
	notesToCreate := make(map[string]*grafeas_proto.Note)
	for _, note := range registerCollectorRequest.Notes {
		noteId := buildNoteIdFromCollectorId(registerCollectorRequest.Id, note)

		if _, ok := notesWithIds[noteId]; ok {
			return nil, createErrorWithCode(log, "cannot use more than one note type when registering a collector", nil, codes.InvalidArgument)
		}

		notesWithIds[noteId] = note
		notesToCreate[noteId] = note
	}

	log = log.With(zap.Any("notes", notesWithIds))

	// find out which notes already exist
	filter := fmt.Sprintf(`name.startsWith("%s/notes/%s-")`, rodeProjectSlug, registerCollectorRequest.Id)
	listNotesResponse, err := r.grafeasCommon.ListNotes(ctx, &grafeas_proto.ListNotesRequest{
		Parent: rodeProjectSlug,
		Filter: filter,
	})
	if err != nil {
		return nil, createError(log, "error listing notes", err)
	}

	// build map of notes that need to be created
	for _, note := range listNotesResponse.Notes {
		noteId := getNoteIdFromNoteName(note.Name)

		if _, ok := notesWithIds[noteId]; ok {
			notesWithIds[noteId].Name = note.Name
			delete(notesToCreate, noteId)
		}
	}

	if len(notesToCreate) != 0 {
		batchCreateNotesResponse, err := r.grafeasCommon.BatchCreateNotes(ctx, &grafeas_proto.BatchCreateNotesRequest{
			Parent: rodeProjectSlug,
			Notes:  notesToCreate,
		})
		if err != nil {
			return nil, createError(log, "error creating notes", err)
		}

		for _, note := range batchCreateNotesResponse.Notes {
			noteId := getNoteIdFromNoteName(note.Name)

			if _, ok := notesWithIds[noteId]; ok {
				notesWithIds[noteId].Name = note.Name
			}
		}
	}

	return &pb.RegisterCollectorResponse{
		Notes: notesWithIds,
	}, nil
}

func (r *rodeServer) createIndex(ctx context.Context, settings indexSetting) error {
	mappings := map[string]interface{}{
		"mappings": map[string]interface{}{
			"_meta": map[string]interface{}{
				"type": "rode",
			},
			"properties": settings.properties,
			"dynamic_templates": []map[string]interface{}{
				{
					"strings_as_keywords": map[string]interface{}{
						"match_mapping_type": "string",
						"mapping": map[string]interface{}{
							"type":  "keyword",
							"norms": false,
						},
					},
				},
			},
		},
	}
	body, _ := esutil.EncodeRequest(mappings)
	response, err := r.esClient.Indices.Create(settings.index, r.esClient.Indices.Create.WithBody(body), r.esClient.Indices.Create.WithContext(ctx))

	if err != nil {
		return err
	}

	if response.IsError() && response.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("unexpected response creating Elasticsearch index: %s", response)
	}

	return nil
}

// validateRodeRequirementsForPolicy ensures that these two rules are followed:
// 1. A policy is expected to return a pass that is simply a boolean representing the AND of all rules.
// 2. A policy is expected to return an array of violations that are maps containing a description id message name pass. pass here is what will be used to determine the overall pass.
func validateRodeRequirementsForPolicy(mod *ast.Module, regoContent string) []error {
	errorsList := []error{}
	// policy must contains a pass block somewhere in the code
	passBlockExists := (len(mod.RuleSet("pass")) > 0)
	// policy must contains a violations block somewhere in the code
	violationsBlockExists := (len(mod.RuleSet("violations")) > 0)
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

func (r *rodeServer) genericGet(ctx context.Context, log *zap.Logger, search *esutil.EsSearch, index string, protoMessage interface{}) (string, error) {
	encodedBody, requestJson := esutil.EncodeRequest(search)
	log = log.With(zap.String("request", requestJson))

	res, err := r.esClient.Search(
		r.esClient.Search.WithContext(ctx),
		r.esClient.Search.WithIndex(index),
		r.esClient.Search.WithBody(encodedBody),
	)
	if err != nil {
		return "", createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return "", createError(log, "error searching elasticsearch for document", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	}

	var searchResults esutil.EsSearchResponse
	if err := esutil.DecodeResponse(res.Body, &searchResults); err != nil {
		return "", createError(log, "error unmarshalling elasticsearch response", err)
	}

	if searchResults.Hits.Total.Value == 0 {
		log.Debug("document not found", zap.Any("search", search))
		return "", status.Error(codes.NotFound, fmt.Sprintf("%T not found", protoMessage))
	}

	return searchResults.Hits.Hits[0].ID, protojson.Unmarshal(searchResults.Hits.Hits[0].Source, proto.MessageV2(protoMessage))
}

type genericListOptions struct {
	index         string
	filter        string
	query         *esutil.EsSearch
	pageSize      int32
	pageToken     string
	sortDirection esutil.EsSortOrder
	sortField     string
}

func (r *rodeServer) genericList(ctx context.Context, log *zap.Logger, options *genericListOptions) (*esutil.EsSearchResponseHits, string, error) {
	body := &esutil.EsSearch{}
	if options.query != nil {
		body = options.query
	}

	if options.filter != "" {
		log = log.With(zap.String("filter", options.filter))
		filterQuery, err := r.filterer.ParseExpression(options.filter)
		if err != nil {
			return nil, "", createError(log, "error while parsing filter expression", err)
		}

		body.Query = filterQuery
	}

	if options.sortField != "" {
		body.Sort = map[string]esutil.EsSortOrder{
			options.sortField: options.sortDirection,
		}
	}

	searchOptions := []func(*esapi.SearchRequest){
		r.esClient.Search.WithContext(ctx),
	}

	var nextPageToken string
	if options.pageToken != "" || options.pageSize != 0 { // handle pagination
		next, extraSearchOptions, err := r.handlePagination(ctx, log, body, options.index, options.pageToken, options.pageSize)
		if err != nil {
			return nil, "", createError(log, "error while handling pagination", err)
		}

		nextPageToken = next
		searchOptions = append(searchOptions, extraSearchOptions...)
	} else {
		searchOptions = append(searchOptions,
			r.esClient.Search.WithIndex(options.index),
			r.esClient.Search.WithSize(maxPageSize),
		)
	}

	encodedBody, requestJson := esutil.EncodeRequest(body)
	log = log.With(zap.String("request", requestJson))
	log.Debug("performing search")

	res, err := r.esClient.Search(
		append(searchOptions, r.esClient.Search.WithBody(encodedBody))...,
	)
	if err != nil {
		return nil, "", createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return nil, "", createError(log, "unexpected response from elasticsearch", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	}

	var searchResults esutil.EsSearchResponse
	if err := esutil.DecodeResponse(res.Body, &searchResults); err != nil {
		return nil, "", createError(log, "error decoding elasticsearch response", err)
	}

	if options.pageToken != "" || options.pageSize != 0 { // if request is paginated, check for last page
		_, from, err := esutil.ParsePageToken(nextPageToken)
		if err != nil {
			return nil, "", createError(log, "error parsing page token", err)
		}

		if from >= searchResults.Hits.Total.Value {
			nextPageToken = ""
		}
	}
	return searchResults.Hits, nextPageToken, nil
}

func (r *rodeServer) handlePagination(ctx context.Context, log *zap.Logger, body *esutil.EsSearch, index, pageToken string, pageSize int32) (string, []func(*esapi.SearchRequest), error) {
	log = log.With(zap.String("pageToken", pageToken), zap.Int32("pageSize", pageSize))

	var (
		pit  string
		from int
		err  error
	)

	// if no pageToken is specified, we need to create a new PIT
	if pageToken == "" {
		res, err := r.esClient.OpenPointInTime(
			r.esClient.OpenPointInTime.WithContext(ctx),
			r.esClient.OpenPointInTime.WithIndex(index),
			r.esClient.OpenPointInTime.WithKeepAlive(pitKeepAlive),
		)
		if err != nil {
			return "", nil, createError(log, "error sending request to elasticsearch", err)
		}
		if res.IsError() {
			return "", nil, createError(log, "unexpected response from elasticsearch", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
		}

		var pitResponse esutil.ESPitResponse
		if err = esutil.DecodeResponse(res.Body, &pitResponse); err != nil {
			return "", nil, createError(log, "error decoding elasticsearch response", err)
		}

		pit = pitResponse.Id
		from = 0
	} else {
		// get the PIT from the provided pageToken
		pit, from, err = esutil.ParsePageToken(pageToken)
		if err != nil {
			return "", nil, createError(log, "error parsing page token", err)
		}
	}

	body.Pit = &esutil.EsSearchPit{
		Id:        pit,
		KeepAlive: pitKeepAlive,
	}

	return esutil.CreatePageToken(pit, from+int(pageSize)), []func(*esapi.SearchRequest){
		r.esClient.Search.WithSize(int(pageSize)),
		r.esClient.Search.WithFrom(from),
	}, err
}

// createError is a helper function that allows you to easily log an error and return a gRPC formatted error.
func createError(log *zap.Logger, message string, err error, fields ...zap.Field) error {
	return createErrorWithCode(log, message, err, codes.Internal, fields...)
}

// createError is a helper function that allows you to easily log an error and return a gRPC formatted error.
func createErrorWithCode(log *zap.Logger, message string, err error, code codes.Code, fields ...zap.Field) error {
	if err == nil {
		log.Error(message, fields...)
		return status.Errorf(code, "%s", message)
	}

	log.Error(message, append(fields, zap.Error(err))...)
	return status.Errorf(code, "%s: %s", message, err)
}

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

func buildNoteIdFromCollectorId(collectorId string, note *grafeas_proto.Note) string {
	switch note.Kind {
	case common_go_proto.NoteKind_VULNERABILITY:
		return fmt.Sprintf("%s-vulnerability", collectorId)
	case common_go_proto.NoteKind_BUILD:
		return fmt.Sprintf("%s-build", collectorId)
	case common_go_proto.NoteKind_IMAGE:
		return fmt.Sprintf("%s-image", collectorId)
	case common_go_proto.NoteKind_PACKAGE:
		return fmt.Sprintf("%s-package", collectorId)
	case common_go_proto.NoteKind_DEPLOYMENT:
		return fmt.Sprintf("%s-deployment", collectorId)
	case common_go_proto.NoteKind_DISCOVERY:
		return fmt.Sprintf("%s-discovery", collectorId)
	case common_go_proto.NoteKind_ATTESTATION:
		return fmt.Sprintf("%s-attestation", collectorId)
	case common_go_proto.NoteKind_INTOTO:
		return fmt.Sprintf("%s-intoto", collectorId)
	}

	return fmt.Sprintf("%s-unspecified", collectorId)
}

func getNoteIdFromNoteName(noteName string) string {
	// note name format: projects/${projectId}/notes/${noteId}
	return strings.TrimPrefix(noteName, rodeProjectSlug+"/notes/")
}
