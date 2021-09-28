package v1alpha1_test

import (
	"flag"
	"github.com/brianvoe/gofakeit/v6"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/common"
	"github.com/rode/rode/proto/v1alpha1"

	"log"
	"os"
	"testing"
)

var (
	fake = gofakeit.New(0)
	rode v1alpha1.RodeClient
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		log.Println("Skipping integration tests because the -short flag was passed")
		os.Exit(0)
	}

	var err error
	rode, err = common.NewRodeClient(&common.ClientConfig{
		Rode: &common.RodeClientConfig{
			Host:                     "localhost:50051",
			DisableTransportSecurity: true,
		},
	})

	if err != nil {
		log.Fatal("Error creating Rode client", err)
	}

	os.Exit(m.Run())
}

func TestRode_v1alpha1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rode v1alpha1 Suite")
}
