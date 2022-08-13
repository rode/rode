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

// Package opa provides client make requests to the Open Policy Agent API
package opa

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	pb "github.com/rode/rode/proto/v1alpha1"

	"go.uber.org/zap"
)

//go:generate counterfeiter -generate

// Client is an interface for sending requests to the OPA API
//counterfeiter:generate . Client
type Client interface {
	InitializePolicy(ctx context.Context, policyId, policy string) error
	EvaluatePolicy(ctx context.Context, policyId string, input interface{}) (*EvaluatePolicyResult, error)
}

type client struct {
	logger  *zap.Logger
	queries map[string]rego.PreparedEvalQuery
}

// EvaluatePolicyResult OPA evaluate policy result
type EvaluatePolicyResult struct {
	Pass       bool                          `json:"pass"`
	Violations []*pb.EvaluatePolicyViolation `json:"violations"`
}

// NewClient OpaClient constructor
func NewClient(logger *zap.Logger) Client {
	return &client{
		logger:  logger,
		queries: map[string]rego.PreparedEvalQuery{},
	}
}

// InitializePolicy prepares the OPA policy for evaluation
func (opa *client) InitializePolicy(ctx context.Context, policyId, policy string) error {
	log := opa.logger.Named("InitializePolicy").With(zap.String("policyId", policyId))
	if _, ok := opa.queries[policyId]; ok {
		log.Debug("Parsed query in cache, skipping prepare")
		return nil
	}

	module, err := ast.ParseModule("rode.rego", policy)
	if err != nil {
		return err
	}

	log.Info("Preparing policy for evaluation")
	query, err := rego.New(
		rego.Query(module.Package.Path.String()),
		rego.ParsedModule(module),
	).PrepareForEval(ctx)

	if err != nil {
		return err
	}

	opa.queries[policyId] = query

	return nil
}

// EvaluatePolicy evaluates OPA policy against provided input
func (opa *client) EvaluatePolicy(ctx context.Context, policyId string, input interface{}) (*EvaluatePolicyResult, error) {
	query, ok := opa.queries[policyId]
	if !ok {
		return nil, errors.New("query has not been initialized")
	}

	rs, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, err
	}

	if len(rs) == 0 {
		return nil, errors.New("no evaluation results from policy")
	}

	if len(rs[0].Expressions) == 0 {
		return nil, errors.New("no expression output in result set")
	}

	result := rs[0].Expressions[0].Value
	resultJson, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var policyResult EvaluatePolicyResult
	if err = json.Unmarshal(resultJson, &policyResult); err != nil {
		return nil, err
	}

	return &policyResult, nil
}
