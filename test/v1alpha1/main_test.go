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

package v1alpha1_test

import (
	"flag"
	"log"
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/test/util"
)

var (
	fake = gofakeit.New(0)
	rode *util.RodeClientSet
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		log.Println("Skipping integration tests because the -short flag was passed")
		os.Exit(0)
	}

	var err error
	if rode, err = util.NewRodeClientSet(); err != nil {
		log.Fatal("Error creating Rode clients", err)
	}

	os.Exit(m.Run())
}

func TestRode_v1alpha1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rode v1alpha1 Suite")
}
