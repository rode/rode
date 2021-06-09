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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/rode/es-index-manager/indexmanager"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/esutil"
	"github.com/rode/rode/pkg/policy"
	"github.com/rode/rode/pkg/resource"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/sync/errgroup"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/rode/rode/auth"
	"github.com/rode/rode/config"
	"github.com/rode/rode/opa"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	grafeas_project_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto"
	"github.com/rode/rode/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	c, err := config.Build(os.Args[0], os.Args[1:])
	if err != nil {
		log.Fatalf("failed to build config: %v", err)
	}

	logger, err := createLogger(c.Debug)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	address := fmt.Sprintf(":%d", c.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	authenticator := auth.NewAuthenticator(c.Auth)
	recoveryHandler := grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
		logger.Error("Panic in gRPC handler", zap.Any("panic", p))

		return status.Errorf(codes.Internal, "Unexpected error")
	})
	s := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_auth.StreamServerInterceptor(authenticator.Authenticate),
				grpc_recovery.StreamServerInterceptor(recoveryHandler),
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_auth.UnaryServerInterceptor(authenticator.Authenticate),
				grpc_recovery.UnaryServerInterceptor(recoveryHandler),
			),
		),
	)
	if c.Debug {
		reflection.Register(s)
	}

	grafeasClientCommon, grafeasClientProjects, err := createGrafeasClients(c.Grafeas.Host)
	if err != nil {
		logger.Fatal("failed to connect to grafeas", zap.String("grafeas host", c.Grafeas.Host), zap.Error(err))
	}
	opaClient := opa.NewClient(logger.Named("opa"), c.Opa.Host, c.Debug)

	esClient, err := createESClient(logger, c.Elasticsearch.Host, c.Elasticsearch.Username, c.Elasticsearch.Password)
	if err != nil {
		logger.Fatal("failed to create Elasticsearch client", zap.Error(err))
	}

	esutilClient := esutil.NewClient(logger.Named("ESClient"), esClient)
	indexManager := indexmanager.NewIndexManager(logger.Named("IndexManager"), esClient, &indexmanager.Config{
		IndexPrefix:  "rode",
		MappingsPath: "mappings",
	})

	filterer := filtering.NewFilterer()

	resourceManager := resource.NewManager(logger.Named("Resource Manager"), esutilClient, c.Elasticsearch, indexManager, filterer)
	policyManager := policy.NewManager(logger.Named("PolicyManager"), esutilClient, c.Elasticsearch, indexManager, filterer, opaClient, grafeasClientCommon)
	rodeServer, err := server.NewRodeServer(logger.Named("rode"), grafeasClientCommon, grafeasClientProjects, resourceManager, indexManager, policyManager)
	if err != nil {
		logger.Fatal("failed to create Rode server", zap.Error(err))
	}
	healthzServer := server.NewHealthzServer(logger.Named("healthz"))

	pb.RegisterRodeServer(s, rodeServer)
	grpc_health_v1.RegisterHealthServer(s, healthzServer)

	mux := cmux.New(lis)
	grpcListener := mux.Match(cmux.HTTP2())
	httpListener := mux.Match(cmux.HTTP1())

	grpcGateway, err := createGrpcGateway(context.Background(), lis.Addr().String())
	if err != nil {
		logger.Fatal("failed to start gateway", zap.Error(err))
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", grpcGateway)

	httpServer := &http.Server{
		Handler: httpMux,
	}

	servers := new(errgroup.Group)
	servers.Go(func() error {
		return s.Serve(grpcListener)
	})
	servers.Go(func() error {
		return httpServer.Serve(httpListener)
	})
	servers.Go(func() error {
		return mux.Serve()
	})

	logger.Info("listening", zap.String("host", lis.Addr().String()))
	healthzServer.Ready()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	terminationSignal := <-sig

	logger.Info("shutting down...", zap.String("termination signal", terminationSignal.String()))
	healthzServer.NotReady()

	s.GracefulStop()
	httpServer.Shutdown(context.Background())
}

func createGrafeasClients(grafeasEndpoint string) (grafeas_proto.GrafeasV1Beta1Client, grafeas_project_proto.ProjectsClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	connection, err := grpc.DialContext(ctx, grafeasEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	grafeasClient := grafeas_proto.NewGrafeasV1Beta1Client(connection)
	projectsClient := grafeas_project_proto.NewProjectsClient(connection)

	return grafeasClient, projectsClient, nil
}

func createGrpcGateway(ctx context.Context, grpcAddress string) (http.Handler, error) {
	conn, err := grpc.DialContext(
		context.Background(),
		grpcAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}
	gwmux := runtime.NewServeMux()
	if err := pb.RegisterRodeHandler(ctx, gwmux, conn); err != nil {
		return nil, err
	}

	return http.Handler(gwmux), nil
}

func createLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}

// https://github.com/rode/grafeas-elasticsearch/blob/bcdf8c2a4e1ec473e18794f6ca8e1718180051e7/go/v1beta1/main/main.go#L44
func createESClient(logger *zap.Logger, elasticsearchEndpoint, username, password string) (*elasticsearch.Client, error) {
	c, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			elasticsearchEndpoint,
		},
		Username: username,
		Password: password,
	})

	if err != nil {
		return nil, err
	}

	res, err := c.Info()
	if err != nil {
		return nil, err
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	logger.Debug("Successful Elasticsearch connection", zap.String("ES Server version", r["version"].(map[string]interface{})["number"].(string)))

	return c, nil
}
