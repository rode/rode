package main

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc"
	"github.com/rode/rode/hack/util"
	pb "github.com/rode/rode/proto/v1alpha1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"log"
)

type jwtAuth struct {
	issuer       string
	clientId     string
	clientSecret string
}

func (j jwtAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	provider, err := oidc.NewProvider(ctx, j.issuer)
	if err != nil {
		return nil, err
	}

	res, err := (&clientcredentials.Config{
		ClientID:     j.clientId,
		ClientSecret: j.clientSecret,
		TokenURL:     provider.Endpoint().TokenURL,
	}).Token(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", res.AccessToken),
	}, nil
}

func (j jwtAuth) RequireTransportSecurity() bool {
	return false // no transport security required for testing locally. don't do this in production
}

func main() {
	conn, err := grpc.Dial(util.Address,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(&jwtAuth{
			issuer:       "http://localhost:8080/auth/realms/test",
			clientId:     "test-openid-client",
			clientSecret: "secret",
		}),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewRodeClient(conn)

	util.CreateOccurrences(c)
}
