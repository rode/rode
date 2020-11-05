package main

import (
	"context"
	"github.com/liatrio/rode-api/server"
	"log"
	"net"

	pb "github.com/liatrio/rode-api/proto/v1alpha1"
	grafeas "github.com/liatrio/rode-api/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grafeasClient, err := createGrafeasClient("localhost:8080")
	if err != nil {
		log.Fatalf("failed to connect to grafeas: %v", err)
	}

	rodeServer := server.NewRodeServer(grafeasClient)
	s := grpc.NewServer()

	pb.RegisterRodeServer(s, rodeServer)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
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
