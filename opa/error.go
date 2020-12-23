package opa

import "fmt"

// OpaClientError interface for errors created by client
type OpaClientError interface {
	error
	Type() OpaClientErrorType
	CausedBy() error
}

type opaClientError struct {
	message   string
	errorType OpaClientErrorType
	causedBy  error
}

// OpaClientErrorType defines error types
type OpaClientErrorType string

// OpaClientErrorType constants
const (
	OpaClientErrorTypePolicyExits   OpaClientErrorType = "Policy Exists"
	OpaClientErrorTypePublishPolicy OpaClientErrorType = "Publish Policy"
	OpaClientErrorTypeHTTP          OpaClientErrorType = "HTTP Error"
	OpaClientErrorTypeBadResponse   OpaClientErrorType = "Bad Response"
)

func (err opaClientError) Error() string {
	if err.causedBy == nil {
		return err.message
	}
	return fmt.Sprintf("%s: %s", err.message, err.causedBy.Error())
}

func (err opaClientError) Type() OpaClientErrorType {
	return err.errorType
}

// Is tests if error is of given type
func (err opaClientError) CausedBy() error {
	return err.causedBy
}
