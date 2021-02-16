// Package opa provides client make requests to the Open Policy Agent API
package opa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// Client is an interface for sending requests to the OPA API
type Client interface {
	PolicyExists(policy string) (bool, error)
	InitializePolicy(policy string) ClientError
	EvaluatePolicy(policy string, input []byte) (*EvaluatePolicyResponse, error)
}

type client struct {
	logger *zap.Logger
	Host   string
}

// EvalutePolicyRequest OPA evalute policy request
type EvalutePolicyRequest struct {
	Input json.RawMessage `json:"input"`
}

// EvaluatePolicyResponse OPA evaluate policy response
type EvaluatePolicyResponse struct {
	Result      *EvaluatePolicyResult `json:"result"`
	Explanation *[]string             `json:"explanation"`
}

// EvaluatePolicyResult OPA evaluate policy result
type EvaluatePolicyResult struct {
	Pass       bool                       `json:"pass"`
	Violations []*EvaluatePolicyViolation `json:"violations"`
}

// EvaluatePolicyViolation OPA evaulate policy violation
type EvaluatePolicyViolation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Message     string `json:"message"`
	Link        string `json:"link"`
	Pass        bool   `json:"pass"`
}

// PolicyViolation Rego rule conditions
type PolicyViolation struct {
	Conditions []byte
}

// Write Rego rule to IO writer
func (v *PolicyViolation) Write(w io.Writer) {
	w.Write([]byte("violation[] {\n"))
	w.Write(v.Conditions)
	w.Write([]byte("\n}\n\n"))
}

// NewClient OpaClient constructor
func NewClient(logger *zap.Logger, host string) Client {
	client := &client{
		logger: logger,
		Host:   host,
	}
	return client
}

// InitializePolicy initializes OPA policy if it does not already exist
func (opa *client) InitializePolicy(policy string) ClientError {
	_ = opa.logger.Named("Initialize Policy")

	exists, err := opa.PolicyExists(policy)
	if err != nil {
		return clientError{"error checking if policy exists", OpaClientErrorTypePolicyExists, err}
	}

	if !exists {
		// fetch violations from ES
		violations := []PolicyViolation{}
		err = opa.publishPolicy(policy, violations)
		if err != nil {
			return clientError{"error publishing policy", OpaClientErrorTypePublishPolicy, err}
		}
	}

	return nil
}

// EvaluatePolicy evalutes OPA policy agains provided input
func (opa *client) EvaluatePolicy(policy string, input []byte) (*EvaluatePolicyResponse, error) {
	log := opa.logger.Named("Evalute Policy")
	request, err := json.Marshal(&EvalutePolicyRequest{Input: json.RawMessage(input)})
	if err != nil {
		log.Error("failed to encode OPA input", zap.Error(err), zap.String("input", string(input)))
		return nil, fmt.Errorf("failed to encode OPA input: %s", err)
	}
	httpResponse, err := http.Post(opa.getURL(fmt.Sprintf("v1/data/%s?explain=full&pretty", policy)), "application/json", bytes.NewReader(request))
	if err != nil {
		log.Error("http request to OPA failed", zap.Error(err))
		return nil, fmt.Errorf("http request to OPA failed: %s", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		log.Error("http response status from OPA not OK", zap.Any("status", httpResponse.Status))
		return nil, fmt.Errorf("http response status not OK")
	}

	response := &EvaluatePolicyResponse{}
	err = json.NewDecoder(httpResponse.Body).Decode(&response)
	if err != nil {
		log.Error("failed to decode OPA result", zap.Error(err))
		return nil, fmt.Errorf("failed to decode OPA result: %s", err)
	}

	return response, nil
}

// policyExists tests if OPA policy exists
func (opa *client) PolicyExists(policy string) (bool, error) {
	log := opa.logger.Named("Policy Exists")
	response, err := http.Get(opa.getURL(fmt.Sprintf("v1/policies/%s", policy)))
	if err != nil {
		log.Error("error sending get policy request to OPA", zap.Error(err))
		return false, clientError{"error sending get policy request to OPA", OpaClientErrorTypeHTTP, err}
	}
	switch response.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		log.Error("unhandled get policy response from OPA", zap.Any("status", response.Status))
		return false, clientError{fmt.Sprintf("unhandled get policy response from OPA: %s", response.Status), OpaClientErrorTypeBadResponse, nil}
	}
}

// publishPolicy publishes attester violation rules to OPA policy
func (opa *client) publishPolicy(policy string, violations []PolicyViolation) error {
	log := opa.logger.Named("Publish Policy")
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("package %s\n\n", policy))
	buf.WriteString("pass = true { count(violation) == 0 }\n\n")
	for _, violation := range violations {
		violation.Write(buf)
	}
	request, err := http.NewRequest(http.MethodPut, opa.getURL(fmt.Sprintf("v1/policies/%s", policy)), buf)
	if err != nil {
		log.Error("error creating create policy request", zap.Error(err))
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Error("error sending create policy request", zap.Error(err))
		return clientError{"error sending create policy request", OpaClientErrorTypeHTTP, err}
	}
	if response.StatusCode != http.StatusOK {
		log.Error("unhandled create policy response status", zap.Any("status", response.Status))
		return clientError{"unhandled create policy resposne status", OpaClientErrorTypeBadResponse, nil}
	}
	return nil
}

// getURL for given OPA API path
func (opa *client) getURL(path string) string {
	return fmt.Sprintf("%s/%s", opa.Host, path)
}
