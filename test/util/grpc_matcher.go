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

package util

import (
	"fmt"

	"github.com/onsi/gomega/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type haveGrpcStatusMatcher struct {
	expected codes.Code
	actual   codes.Code
}

func HaveGrpcStatus(expected codes.Code) types.GomegaMatcher {
	return &haveGrpcStatusMatcher{expected: expected}
}

func (h *haveGrpcStatusMatcher) Match(actual interface{}) (bool, error) {
	statusError, err := toGrpcStatus(actual)
	if err != nil {
		return false, err
	}

	h.actual = statusError.Code()

	return h.actual == h.expected, nil
}

func (h *haveGrpcStatusMatcher) FailureMessage(_ interface{}) string {
	return fmt.Sprintf("Expected gRPC status code to be %[1]s (%[1]d), but was %[2]s (%[2]d)", h.expected, h.actual)
}

func (h *haveGrpcStatusMatcher) NegatedFailureMessage(_ interface{}) string {
	return fmt.Sprintf("Expected gRPC status code not to equal %[1]s (%[1]d)", h.actual)
}

func toGrpcStatus(actual interface{}) (*status.Status, error) {
	if actual == nil {
		return nil, fmt.Errorf("expected a gRPC status, but was nil")
	}

	actualError, ok := actual.(error)
	if !ok {
		return nil, fmt.Errorf("expected %v to be an error, but was of type %[1]T", actual)
	}

	statusError, ok := status.FromError(actualError)
	if !ok {
		return nil, fmt.Errorf("'%v' was an error, but not a gRPC status", actualError)
	}

	return statusError, nil
}
