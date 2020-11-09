package main

import (
	"context"
	"github.com/liatrio/rode-api/server"
	"go.uber.org/zap"
	"log"
	"net"

	pb "github.com/liatrio/rode-api/proto/v1alpha1"
	grafeas "github.com/liatrio/rode-api/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"google.golang.org/grpc"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Fatal("failed to listen", zap.NamedError("error", err))
	}

	grafeasClient, err := createGrafeasClient("localhost:8080")
	if err != nil {
		logger.Fatal("failed to connect to grafeas", zap.NamedError("error", err))
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
