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
