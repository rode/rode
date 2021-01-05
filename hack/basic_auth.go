package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/rode/rode/hack/util"
	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc"
	"log"
)

type basicAuth struct {
	username, password string
}

func (b basicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	token := fmt.Sprintf("%s:%s", b.username, b.password)
	enc := base64.StdEncoding.EncodeToString([]byte(token))

	return map[string]string{
		"authorization": fmt.Sprintf("Basic %s", enc),
	}, nil
}

func (basicAuth) RequireTransportSecurity() bool {
	return false // no transport security required for testing locally. don't do this in production
}

const (
	address = "localhost:50051"
)

func main() {
	conn, err := grpc.Dial(address,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(&basicAuth{
			username: "foo",
			password: "bar",
		}),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewRodeClient(conn)

	util.CreateOccurrences(c)
}
