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
