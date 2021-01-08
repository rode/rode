package opa

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var logger *zap.Logger

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OPA Suite")
}

var _ = BeforeSuite(func() {
	logger, _ = zap.NewDevelopment()
	httpmock.Activate()
	gofakeit.Seed(time.Now().UnixNano())
})

var _ = BeforeEach(func() {
	// remove any mocks
	httpmock.Reset()
})

var _ = AfterSuite(func() {
	httpmock.DeactivateAndReset()
})
