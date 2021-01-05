package main

import (
	"context"
	"fmt"
	"github.com/rode/rode/auth"
	"github.com/rode/rode/config"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
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

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.Port))
	if err != nil {
		logger.Fatal("failed to listen", zap.NamedError("error", err))
	}

	grafeasClient, err := createGrafeasClient(c.Grafeas.Host)
	if err != nil {
		logger.Fatal("failed to connect to grafeas", zap.String("grafeas host", c.Grafeas.Host), zap.NamedError("error", err))
	}

	authenticator := auth.NewAuthenticator(c.Auth)
	s := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_auth.StreamServerInterceptor(authenticator.Authenticate),
		),
		grpc.UnaryInterceptor(
			grpc_auth.UnaryServerInterceptor(authenticator.Authenticate),
		),
	)

	if c.Debug {
		reflection.Register(s)
	}

	rodeServer := server.NewRodeServer(logger.Named("rode"), grafeasClient)
	healthzServer := server.NewHealthzServer(logger.Named("healthz"))

	pb.RegisterRodeServer(s, rodeServer)
	grpc_health_v1.RegisterHealthServer(s, healthzServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Fatal("failed to serve", zap.NamedError("error", err))
		}
	}()

	logger.Info("listening", zap.String("host", lis.Addr().String()))
	healthzServer.Ready()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	terminationSignal := <-sig

	logger.Info("shutting down...", zap.String("termination signal", terminationSignal.String()))
	healthzServer.NotReady()

	s.GracefulStop()
}

func createGrafeasClient(grafeasEndpoint string) (grafeas.GrafeasV1Beta1Client, error) {
	connection, err := grpc.Dial(grafeasEndpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := grafeas.NewGrafeasV1Beta1Client(connection)

	// test grafeas connection
	_, err = client.ListOccurrences(context.Background(), &grafeas.ListOccurrencesRequest{
		Parent: "projects/rode",
	})
	return client, err
}

func createLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}
