package server

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type HealthzServer interface {
	grpc_health_v1.HealthServer
	Ready()
	NotReady()
}

type healthzServer struct {
	grpc_health_v1.UnimplementedHealthServer
	ready  bool
	logger *zap.Logger
}

func NewHealthzServer(logger *zap.Logger) HealthzServer {
	return &healthzServer{
		logger: logger,
	}
}

func (h *healthzServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log := h.logger.Named("Check")
	log.Debug("received grpc health check")

	if h.ready {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}, nil
}

// Watch is unimplemented for now
func (h *healthzServer) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "health watching is not supported yet")
}

func (h *healthzServer) Ready() {
	h.ready = true
}

func (h *healthzServer) NotReady() {
	h.ready = false
}
