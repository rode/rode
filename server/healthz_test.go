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

package server

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var _ = Describe("healthz server", func() {
	var healthzServer HealthzServer

	BeforeEach(func() {
		healthzServer = NewHealthzServer(logger.Named("healthz server test"))
	})

	When("a health check is received", func() {
		It("should respond with 'SERVING' when ready", func() {
			healthzServer.Ready()
			res, err := healthzServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})

			Expect(err).ToNot(HaveOccurred())
			Expect(res.Status).To(BeEquivalentTo(grpc_health_v1.HealthCheckResponse_SERVING))
		})

		It("should respond with 'NOT_SERVING' when not ready", func() {
			healthzServer.NotReady()
			res, err := healthzServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})

			Expect(err).ToNot(HaveOccurred())
			Expect(res.Status).To(BeEquivalentTo(grpc_health_v1.HealthCheckResponse_NOT_SERVING))
		})
	})
})
