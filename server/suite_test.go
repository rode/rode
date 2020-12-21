package server

import (
	"github.com/brianvoe/gofakeit/v5"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
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
