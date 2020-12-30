package opa

import "fmt"

// ClientError interface for errors created by client
type ClientError interface {
	error
	Type() ClientErrorType
	CausedBy() error
}

type clientError struct {
	message   string
	errorType ClientErrorType
	causedBy  error
}

// ClientErrorType defines error types
type ClientErrorType string

// OpaClientErrorType constants
const (
	OpaClientErrorTypePolicyExits   ClientErrorType = "Policy Exists"
	OpaClientErrorTypePublishPolicy ClientErrorType = "Publish Policy"
	OpaClientErrorTypeHTTP          ClientErrorType = "HTTP Error"
	OpaClientErrorTypeBadResponse   ClientErrorType = "Bad Response"
)

// Error returns formatted error message
func (err clientError) Error() string {
	if err.causedBy == nil {
		return err.message
	}
	return fmt.Sprintf("%s: %s", err.message, err.causedBy.Error())
}

// Type gets error type
func (err clientError) Type() ClientErrorType {
	return err.errorType
}

// CausedBy gets error that caused this error if one exists
func (err clientError) CausedBy() error {
	return err.causedBy
}
