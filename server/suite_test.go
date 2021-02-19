package server

import (
	"github.com/brianvoe/gofakeit/v5"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"net/http"
	"testing"
	"time"
)

var logger *zap.Logger

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
	logger, _ = zap.NewDevelopment()
	gofakeit.Seed(time.Now().UnixNano())
})

// https://github.com/rode/grafeas-elasticsearch/blob/624ccb5d038b55d90fb7c6b3b5378125d7ad0aa5/go/v1beta1/storage/storage_suite_test.go#L20-L48
type mockEsTransport struct {
	receivedHttpRequests  []*http.Request
	preparedHttpResponses []*http.Response
	actions               []func(req *http.Request) (*http.Response, error)
}

func (m *mockEsTransport) Perform(req *http.Request) (*http.Response, error) {
	m.receivedHttpRequests = append(m.receivedHttpRequests, req)

	// if we have an action, return its result
	if len(m.actions) != 0 {
		action := m.actions[0]
		if action != nil {
			m.actions = append(m.actions[:0], m.actions[1:]...)
			return action(req)
		}
	}

	// if we have a prepared response, send it
	if len(m.preparedHttpResponses) != 0 {
		res := m.preparedHttpResponses[0]
		m.preparedHttpResponses = append(m.preparedHttpResponses[:0], m.preparedHttpResponses[1:]...)

		return res, nil
	}

	// return nil if we don't know what to do
	return nil, nil
}
