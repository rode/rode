package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	// "github.com/rode/rode/hack/util"
	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	rode := pb.NewRodeClient(conn)

	response, err := rode.AttestPolicy(context.Background(), &pb.AttestPolicyRequest{
		Policy: "example1",
		ResourceURI: "harbor.localhost/library/nginx@sha256:0b159cd1ee1203dad901967ac55eee18c24da84ba3be384690304be93538bea8",
	})
	if err != nil {
		log.Fatalf("attest policy returned error: %v", err)
	}

	json, err := json.Marshal(response)
	if err != nil {
		log.Fatal("error json encoding response")
	}
	fmt.Println(string(json))
}