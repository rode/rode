package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/rode/rode/proto/v1alpha1"
	grafeas "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/rode/rode/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	debug       bool
	port        int
	grafeasHost string
)

func main() {
	flag.IntVar(&port, "port", 50051, "the port that the rode API server should listen on")
	flag.BoolVar(&debug, "debug", false, "when set, debug mode will be enabled")
	flag.StringVar(&grafeasHost, "grafeas-host", "localhost:8080", "the host to use to connect to grafeas")

	flag.Parse()

	logger, err := createLogger(debug)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Fatal("failed to listen", zap.NamedError("error", err))
	}

	grafeasClient, err := createGrafeasClient(grafeasHost)
	if err != nil {
		logger.Fatal("failed to connect to grafeas", zap.String("grafeas host", grafeasHost), zap.NamedError("error", err))
	}

	rodeServer := server.NewRodeServer(logger.Named("rode"), grafeasClient)
	healthzServer := server.NewHealthzServer(logger.Named("healthz"))
	s := grpc.NewServer()

	if debug {
		reflection.Register(s)
	}

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
