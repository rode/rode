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
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/google/uuid"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"

	"github.com/golang/protobuf/proto"
	"github.com/open-policy-agent/opa/ast"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	rodeElasticsearchOccurrencesIndex = "grafeas-v1beta1-rode-occurrences"
	rodeElasticsearchPoliciesIndex    = "rode-v1alpha1-policies"
	maxPageSize                       = 1000
)

// NewRodeServer constructor for rodeServer
func NewRodeServer(
	logger *zap.Logger,
	grafeasCommon grafeas_proto.GrafeasV1Beta1Client,
	grafeasProjects grafeas_project_proto.ProjectsClient,
	opa opa.Client,
	esClient *elasticsearch.Client,
	filterer filtering.Filterer,
) (pb.RodeServer, error) {
	rodeServer := &rodeServer{
		logger:          logger,
		grafeasCommon:   grafeasCommon,
		grafeasProjects: grafeasProjects,
		opa:             opa,
		esClient:        esClient,
		filterer:        filterer,
	}
	if err := rodeServer.initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize rode server: %s", err)
	}

	return rodeServer, nil
}

type rodeServer struct {
	pb.UnimplementedRodeServer
	logger          *zap.Logger
	esClient        *elasticsearch.Client
	filterer        filtering.Filterer
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

func (r *rodeServer) EvaluatePolicy(ctx context.Context, request *pb.EvaluatePolicyRequest) (*pb.EvaluatePolicyResponse, error) {
	var err error
	log := r.logger.Named("EvaluatePolicy").With(zap.String("policy", request.Policy), zap.String("resource", request.ResourceURI))
	log.Debug("evaluate policy request received")

	// check OPA policy has been loaded
	err = r.opa.InitializePolicy(request.Policy)
	if err != nil {
		log.Error("error checking if policy exists", zap.Error(err))
		return nil, status.Error(codes.Internal, "check if policy exists failed")
	}

	// fetch occurrences from grafeas
	listOccurrencesResponse, err := r.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{Parent: "projects/rode", Filter: fmt.Sprintf(`"resource.uri" == "%s"`, request.ResourceURI)})
	if err != nil {
		log.Error("list occurrences failed", zap.Error(err), zap.String("resource", request.ResourceURI))
		return nil, status.Error(codes.Internal, "list occurrences failed")
	}
	log.Debug("Occurrences found", zap.Any("occurrences", listOccurrencesResponse))

	// json encode occurrences. list occurrences response should not generate error
	input, _ := protojson.Marshal(proto.MessageV2(listOccurrencesResponse))

	// evalute OPA policy
	evaluatePolicyResponse, err := r.opa.EvaluatePolicy(request.Policy, input)
	if err != nil {
		log.Error("evaluate OPA policy failed")
		return nil, status.Error(codes.Internal, "evaluate OPA policy failed")
	}
	log.Debug("Evalute policy result", zap.Any("policy result", evaluatePolicyResponse))

	attestation := &pb.EvaluatePolicyResult{}
	attestation.Created = timestamppb.Now()
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

	searchQuery := esSearch{
		Collapse: &esCollapse{
			Field: "resource.uri",
		},
	}

	if request.Filter != "" {
		parsedQuery, err := r.filterer.ParseExpression(request.Filter)
		if err != nil {
			log.Error("failed to parse query", zap.Error(err))
			return nil, err
		}

		searchQuery.Query = parsedQuery
	}

	encodedBody, requestJSON := encodeRequest(searchQuery)
	log.Debug("es request payload", zap.Any("payload", requestJSON))
	res, err := r.esClient.Search(
		r.esClient.Search.WithContext(ctx),
		r.esClient.Search.WithIndex(rodeElasticsearchOccurrencesIndex),
		r.esClient.Search.WithBody(encodedBody),
		r.esClient.Search.WithSize(maxPageSize),
	)

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		return nil, fmt.Errorf("error occurred during ES query %v", res)
	}

	var searchResults esSearchResponse
	if err := decodeResponse(res.Body, &searchResults); err != nil {
		return nil, err
	}
	var resources []*grafeas_proto.Resource
	for _, hit := range searchResults.Hits.Hits {
		occurrence := &grafeas_proto.Occurrence{}
		err := protojson.Unmarshal(hit.Source, proto.MessageV2(occurrence))
		if err != nil {
			log.Error("failed to convert", zap.Error(err))
			return nil, err
		}

		resources = append(resources, occurrence.Resource)
	}

	return &pb.ListResourcesResponse{
		Resources:     resources,
		NextPageToken: "",
	}, nil
}

func encodeRequest(body interface{}) (io.Reader, string) {
	b, err := json.Marshal(body)
	if err != nil {
		// we should know that `body` is a serializable struct before invoking `encodeRequest`
		panic(err)
	}

	return bytes.NewReader(b), string(b)
}

func decodeResponse(r io.ReadCloser, i interface{}) error {
	return json.NewDecoder(r).Decode(i)
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
	// Create an index for policy storage
	r.esClient.Indices.Create(rodeElasticsearchPoliciesIndex)

	return nil
}

func (r *rodeServer) ListOccurrences(ctx context.Context, occurrenceRequest *pb.ListOccurrencesRequest) (*pb.ListOccurrencesResponse, error) {
	log := r.logger.Named("ListOccurrences")
	log.Debug("received request", zap.Any("ListOccurrencesRequest", occurrenceRequest))

	requestedFilter := occurrenceRequest.Filter

	listOccurrencesResponse, err := r.grafeasCommon.ListOccurrences(ctx, &grafeas_proto.ListOccurrencesRequest{Parent: "projects/rode", Filter: requestedFilter})
	if err != nil {
		log.Error("list occurrences failed", zap.Error(err), zap.String("filter", occurrenceRequest.Filter))
		return nil, status.Error(codes.Internal, "list occurrences failed")
	}
	return &pb.ListOccurrencesResponse{
		Occurrences:   listOccurrencesResponse.GetOccurrences(),
		NextPageToken: "",
	}, nil
}

func (r *rodeServer) ValidatePolicy(ctx context.Context, policy *pb.ValidatePolicyRequest) (*pb.ValidatePolicyResponse, error) {
	log := r.logger.Named("ValidatePolicy")

	if len(policy.Policy) == 0 {
		return nil, createError(log, "empty policy passed in", nil)
	}

	// Generate the AST
	mod, err := ast.ParseModule("validate_module", string(policy.Policy))
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
	log.Debug("compilation successful")

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

	policy := &pb.Policy{
		Id:      uuid.New().String(),
		Version: 1,
		Policy:  policyEntity,
	}
	str, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(proto.MessageV2(policy))
	if err != nil {
		return nil, createError(log, fmt.Sprintf("error marshalling %T to json", policy), err)
	}
	res, err := r.esClient.Index(
		rodeElasticsearchPoliciesIndex,
		bytes.NewReader(str),
		r.esClient.Index.WithContext(ctx),
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

	search := &esSearch{
		Query: &filtering.Query{
			Term: &filtering.Term{
				"id.keyword": getPolicyRequest.Id,
			},
		},
	}

	policy := &pb.Policy{}
	encodedBody, requestJson := encodeRequest(search)
	res, err := r.esClient.Search(
		r.esClient.Search.WithContext(ctx),
		r.esClient.Search.WithIndex(rodeElasticsearchPoliciesIndex),
		r.esClient.Search.WithBody(encodedBody),
		r.esClient.Search.WithSize(maxPageSize),
	)
	log = log.With(zap.String("request", requestJson))

	if err != nil {
		return nil, createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return nil, createError(log, "error searching elasticsearch for document", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	}

	var searchResults esSearchResponse
	if err := decodeResponse(res.Body, &searchResults); err != nil {
		return nil, createError(log, "error unmarshalling elasticsearch response", err)
	}

	if searchResults.Hits.Total.Value == 0 {
		log.Debug("document not found", zap.Any("search", search))
		return nil, status.Error(codes.NotFound, fmt.Sprintf("%T not found", policy))
	}
	protojson.Unmarshal(searchResults.Hits.Hits[0].Source, proto.MessageV2(policy))

	return policy, nil

}

func (r *rodeServer) DeletePolicy(ctx context.Context, deletePolicyRequest *pb.DeletePolicyRequest) (*emptypb.Empty, error) {
	log := r.logger.Named("DeletePolicy")

	search := &esSearch{
		Query: &filtering.Query{
			Term: &filtering.Term{
				"id.keyword": deletePolicyRequest.Id,
			},
		},
	}

	encodedBody, requestJSON := encodeRequest(search)
	log.Debug("es request payload", zap.Any("payload", requestJSON))

	res, err := r.esClient.DeleteByQuery(
		[]string{rodeElasticsearchPoliciesIndex},
		encodedBody,
		r.esClient.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		return nil, createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return nil, createError(log, "received unexpected response from elasticsearch", nil)
	}

	var deletedResults esDeleteResponse
	if err = decodeResponse(res.Body, &deletedResults); err != nil {
		return nil, createError(log, "error unmarshalling elasticsearch response", err)
	}

	if deletedResults.Deleted == 0 {
		return nil, createError(log, "elasticsearch returned zero deleted documents", nil, zap.Any("response", deletedResults))
	}

	return &emptypb.Empty{}, nil
}

func (r *rodeServer) ListPolicies(ctx context.Context, listPoliciesRequest *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	log := r.logger.Named("List Policies")

	// filtering logic here

	var policies []*pb.Policy
	res, err := r.esClient.Search(
		r.esClient.Search.WithContext(ctx),
		r.esClient.Search.WithIndex(rodeElasticsearchPoliciesIndex),
		r.esClient.Search.WithSize(maxPageSize),
	)

	if err != nil {
		return nil, createError(log, "error sending request to elasticsearch", err)
	}
	if res.IsError() {
		return nil, createError(log, "error searching elasticsearch for document", nil, zap.String("response", res.String()), zap.Int("status", res.StatusCode))
	}

	var searchResults esSearchResponse
	if err := decodeResponse(res.Body, &searchResults); err != nil {
		return nil, createError(log, "error unmarshalling elasticsearch response", err)
	}

	if searchResults.Hits.Total.Value == 0 {
		log.Debug("document not found", zap.Any("search", "filter replace here"))
		return &pb.ListPoliciesResponse{}, nil
	}

	for _, hit := range searchResults.Hits.Hits {
		hitLogger := log.With(zap.String("policy raw", string(hit.Source)))

		policy := &pb.Policy{}
		err := protojson.Unmarshal(hit.Source, proto.MessageV2(policy))
		if err != nil {
			log.Error("failed to convert _doc to policy", zap.Error(err))
			return nil, createError(hitLogger, "error converting _doc to policy", err)
		}

		hitLogger.Debug("policy hit", zap.Any("policy", policy))

		policies = append(policies, policy)
	}

	return &pb.ListPoliciesResponse{Policies: policies}, nil
}

// // Determine method for field masks
// func (r *rodeServer) UpdatePolicy(ctx context.Context, occurrenceRequest *pb.ListOccurrencesRequest) (*pb.ListOccurrencesResponse, error) {

// }

// createError is a helper function that allows you to easily log an error and return a gRPC formatted error.
func createError(log *zap.Logger, message string, err error, fields ...zap.Field) error {
	if err == nil {
		log.Error(message, fields...)
		return status.Errorf(codes.Internal, "%s", message)
	}

	log.Error(message, append(fields, zap.Error(err))...)
	return status.Errorf(codes.Internal, "%s: %s", message, err)
}

type esDeleteResponse struct {
	Deleted int `json:"deleted"`
}
