package main

import (
	"context"
	"fmt"
	"log"
	"net"

	grafeas "github.com/grafeas/grafeas/proto/v1beta1/grafeas_go_proto"
	project "github.com/grafeas/grafeas/proto/v1beta1/project_go_proto"
	pb "github.com/liatrio/rode-api/proto/v1alpha1"
	"google.golang.org/grpc"
)

var client *grafeasClient

type server struct {
	pb.UnimplementedRodeServer
}

type grafeasClient struct {
	client             grafeas.GrafeasV1Beta1Client
	projectClient      project.ProjectsClient
	projectID          string
	projectInitialized bool
}

// GrafeasClient handle into grafeas
// type GrafeasClient interface {
// 	Creator
// 	Lister
// }

// NewGrafeasClient creates a new client
func NewGrafeasClient(endpoint string) (*grafeasClient, error) {

	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := grafeas.NewGrafeasV1Beta1Client(conn)
	projectClient := project.NewProjectsClient(conn)
	c := &grafeasClient{
		client,
		projectClient,
		"projects/rode",
		false,
	}

	return c, nil
}

func (*server) BatchCreateOccurrences(ctx context.Context, occurrenceRequest *grafeas.BatchCreateOccurrencesRequest) (*grafeas.BatchCreateOccurrencesResponse, error) {
	fmt.Println("This works!!!!!")
	fmt.Printf("%#v\n", *occurrenceRequest.Occurrences[0])

	//Forward to grafeas to create occurrence
	occurrenceResponse, err := client.client.BatchCreateOccurrences(ctx, occurrenceRequest)
	if err != nil {
		fmt.Println("Failed to create occurrence")
		return nil, err
	}
	return occurrenceResponse, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	client, err = NewGrafeasClient("localhost:8080")
	if err != nil {

		log.Fatalf("failed to talk to grafeas: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterRodeServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
