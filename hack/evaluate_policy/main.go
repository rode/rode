package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	opaHost string
	policy string
	resource string
)

func init() {
	flag.StringVar(&opaHost, "opa-host", "localhost:50051", "OPA host")
	flag.StringVar(&policy, "policy", "example", "OPA policy name")
	flag.StringVar(&resource, "resource", "harbor.localhost/library/nginx:latest", "resource URI")
}

func main() {
	flag.Parse()

	conn, err := grpc.Dial(
		opaHost,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	rode := pb.NewRodeClient(conn)

	response, err := rode.AttestPolicy(context.Background(), &pb.AttestPolicyRequest{
		Policy: policy,
		ResourceURI: resource,
	})
	if err != nil {
		log.Fatalf("attest policy returned error: %v", err)
	}

	if response.Explanation != nil {
		explanation, _ := json.Marshal(response.Explanation)
		fmt.Println(string(explanation))
		response.Explanation = nil
	}

	json, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(response)
	if err != nil {
		log.Fatal("error json encoding response")
	}
	fmt.Println(string(json))
}