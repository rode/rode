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
}

func HaveGrpcStatus(expected codes.Code) types.GomegaMatcher {
	return &haveGrpcStatusMatcher{expected}
}

func (h *haveGrpcStatusMatcher) Match(actual interface{}) (bool, error) {
	if actual == nil {
		return false, nil
	}
	actualError, ok := actual.(error)
	if !ok {
		return false, fmt.Errorf("expected %v to be an error, but was %[1]T", actualError)
	}

	statusError, ok := status.FromError(actualError)
	if !ok {
		return false, fmt.Errorf("%v was not a gRPC status", actualError)
	}

	return statusError.Code() == h.expected, nil
}

func (h *haveGrpcStatusMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%v to be a gRPC status with code\n\t%s", actual, h.expected.String())
}

func (h *haveGrpcStatusMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%v not to equal code \n\t%s", actual, h.expected.String())
}
