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
	"crypto/tls"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/credentials"
	"net/http"
)

type jwtAuth struct {
	tokenSource oauth2.TokenSource
	insecure    bool
}

func newJwtAuth(config *JWTAuthConfig, insecure bool) (credentials.PerRPCCredentials, error) {
	clientCredentialsConfig := &clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.TokenURL,
	}

	ctx := context.Background()

	if config.TlsInsecureSkipVerify {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	tokenSource := clientCredentialsConfig.TokenSource(ctx)

	// get an initial token to ensure client credentials are valid
	_, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return &jwtAuth{
		tokenSource: tokenSource,
		insecure:    insecure,
	}, nil
}

func (j jwtAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	token, err := j.tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", token.AccessToken),
	}, nil
}

func (j jwtAuth) RequireTransportSecurity() bool {
	return !j.insecure
}
