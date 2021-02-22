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

// NewClientError constructor
func NewClientError(message string, errorType ClientErrorType, causedBy error) ClientError {
	return clientError{
		message,
		errorType,
		causedBy,
	}
}

// OpaClientErrorType constants
const (
	OpaClientErrorTypeGetPolicy      ClientErrorType = "Get Policy"
	OpaClientErrorTypePolicyNotFound ClientErrorType = "Policy Not Found"
	OpaClientErrorTypePublishPolicy  ClientErrorType = "Publish Policy"
	OpaClientErrorTypeHTTP           ClientErrorType = "HTTP Error"
	OpaClientErrorTypeBadResponse    ClientErrorType = "Bad Response"
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
