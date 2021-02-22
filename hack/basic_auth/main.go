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

func main() {
	conn, err := grpc.Dial(util.Address,
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
