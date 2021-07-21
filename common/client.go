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

package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc"
)

var (
	contextDuration = 10 * time.Second
)

func NewRodeClient(config *ClientConfig, dialOptions ...grpc.DialOption) (pb.RodeClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), contextDuration)
	defer cancel()

	if config == nil || config.Rode == nil || config.Rode.Host == "" {
		return nil, errors.New("rode host must be specified")
	}

	multipleAuthEnabled := config.oidcAuthIsConfigured() && config.basicAuthIsConfigured() ||
		config.ProxyAuth && (config.oidcAuthIsConfigured() || config.basicAuthIsConfigured())

	if multipleAuthEnabled {
		return nil, errors.New("only one authentication method can be used")
	}

	dialOptions = append(dialOptions, grpc.WithBlock())

	if config.Rode.DisableTransportSecurity {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}

	if config.oidcAuthIsConfigured() {
		oidcCredentials, err := newOidcAuth(config.OIDCAuth, config.Rode.DisableTransportSecurity)
		if err != nil {
			return nil, fmt.Errorf("error configuring OIDC auth: %v", err)
		}

		dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(oidcCredentials))
	}

	if config.basicAuthIsConfigured() {
		if config.BasicAuth.Username == "" || config.BasicAuth.Password == "" {
			return nil, errors.New("both username and password must be set for basic auth")
		}

		dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(newBasicAuth(config.BasicAuth, config.Rode.DisableTransportSecurity)))
	}

	if config.ProxyAuth {
		dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(newProxyAuth(config.Rode.DisableTransportSecurity)))
	}

	conn, err := grpc.DialContext(ctx, config.Rode.Host, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("error connecting to rode server: %v", err)
	}

	return pb.NewRodeClient(conn), nil
}
