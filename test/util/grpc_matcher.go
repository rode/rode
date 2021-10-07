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
