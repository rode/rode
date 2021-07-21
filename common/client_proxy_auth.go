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

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/credentials"
)

type proxyAuth struct {
	insecure bool
}

func newProxyAuth(insecure bool) credentials.PerRPCCredentials {
	return &proxyAuth{insecure}
}

func (p *proxyAuth) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	authzHeader := metautils.ExtractIncoming(ctx).Get("authorization")
	metadata := map[string]string{}
	if authzHeader != "" {
		metadata["authorization"] = authzHeader
	}

	return metadata, nil
}

func (p *proxyAuth) RequireTransportSecurity() bool {
	return !p.insecure
}
