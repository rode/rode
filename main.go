package main

import (
	"context"
	"github.com/grafeas/grafeas/proto/v1beta1/grafeas_go_proto"
	pb "github.com/liatrio/rode-collector-service/proto/v1alpha1"
	"google.golang.org/grpc"
	"log"
	"net"
)

type server struct {
	pb.UnimplementedRodeCollectorServiceServer
}

func (*server) BatchCreateOccurrences(context.Context, *grafeas_go_proto.BatchCreateOccurrencesRequest) (*grafeas_go_proto.BatchCreateOccurrencesResponse, error) {
	return nil, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterRodeCollectorServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
