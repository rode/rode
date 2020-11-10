package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/liatrio/rode-api/server"
	"go.uber.org/zap"
	"log"
	"net"

	pb "github.com/liatrio/rode-api/proto/v1alpha1"
	grafeas "github.com/liatrio/rode-api/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"google.golang.org/grpc"
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
	logger.Info("listening", zap.String("host", lis.Addr().String()))

	grafeasClient, err := createGrafeasClient(grafeasHost)
	if err != nil {
		logger.Fatal("failed to connect to grafeas", zap.String("grafeas host", grafeasHost), zap.NamedError("error", err))
	}

	rodeServer := server.NewRodeServer(logger, grafeasClient)
	s := grpc.NewServer()

	pb.RegisterRodeServer(s, rodeServer)
	if err := s.Serve(lis); err != nil {
		logger.Fatal("failed to serve", zap.NamedError("error", err))
	}
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
