// Copyright 2021 Google LLC
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
	"encoding/json"
	"flag"
	"fmt"
	"log"

	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	opaHost  string
	policy   string
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

	response, err := rode.EvaluatePolicy(context.Background(), &pb.EvaluatePolicyRequest{
		Policy:      policy,
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
