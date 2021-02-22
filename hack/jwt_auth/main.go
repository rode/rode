// Copyright 2021 The Rode Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
