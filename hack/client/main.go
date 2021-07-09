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
	"github.com/rode/rode/common"
	pb "github.com/rode/rode/proto/v1alpha1"
	"log"
)

func main() {
	// configure the rode client with OIDC auth
	rodeClient, err := common.NewRodeClient(&common.ClientConfig{
		Rode: &common.RodeClientConfig{
			Host:                     "localhost:50051",
			DisableTransportSecurity: true,
		},
		OIDCAuth: &common.OIDCAuthConfig{
			ClientID:              "rode-ui",
			ClientSecret:          "8a487ca4-6a96-4dc2-affa-35ea7f67e50c",
			TokenURL:              "https://keycloak.test/auth/realms/rode-demo/protocol/openid-connect/token",
			TlsInsecureSkipVerify: true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// sample request to test authentication
	policyGroups, err := rodeClient.ListPolicyGroups(context.Background(), &pb.ListPolicyGroupsRequest{})
	if err != nil {
		log.Fatal(fmt.Errorf("error listing policy groups: %v", err))
	}

	log.Println(policyGroups)
}
