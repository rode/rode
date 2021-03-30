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

package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	Expect := NewGomegaWithT(t).Expect

	for _, tc := range []struct {
		name        string
		flags       []string
		expected    *Config
		expectError bool
	}{
		{
			name:  "defaults",
			flags: []string{},
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{},
					JWT:   &JWTAuthConfig{},
				},
				Elasticsearch: &ElasticsearchConfig{
					Host:    "http://elasticsearch-master:9200",
					Refresh: "true",
				},
				Grafeas: &GrafeasConfig{
					Host: "localhost:8080",
				},
				Opa: &OpaConfig{
					Host: "http://localhost:8181",
				},
				GrpcPort: 50051,
				HttpPort: 50052,
				Debug:    false,
			},
		},
		{
			name:        "bad gRPC port",
			flags:       []string{"--grpc-port=foo"},
			expectError: true,
		},
		{
			name:        "bad http port",
			flags:       []string{"--http-port=bar"},
			expectError: true,
		},
		{
			name:        "bad debug",
			flags:       []string{"--debug=bar"},
			expectError: true,
		},
		{
			name:  "basic auth",
			flags: []string{"--basic-auth-username=foo", "--basic-auth-password=bar"},
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{
						Username: "foo",
						Password: "bar",
					},
					JWT: &JWTAuthConfig{},
				},
				Elasticsearch: &ElasticsearchConfig{
					Host:    "http://elasticsearch-master:9200",
					Refresh: "true",
				},
				Grafeas: &GrafeasConfig{
					Host: "localhost:8080",
				},
				Opa: &OpaConfig{
					Host: "http://localhost:8181",
				},
				GrpcPort: 50051,
				HttpPort: 50052,
				Debug:    false,
			},
		},
		{
			name:        "basic auth missing username",
			flags:       []string{"--basic-auth-password=bar"},
			expectError: true,
		},
		{
			name:        "basic auth missing password",
			flags:       []string{"--basic-auth-username=foo"},
			expectError: true,
		},
		{
			name:        "jwt required audience without issuer",
			flags:       []string{"--jwt-required-audience=foo"},
			expectError: true,
		},
		{
			name:  "OPA host",
			flags: []string{"--opa-host=opa.test.na:8181"},
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{},
					JWT:   &JWTAuthConfig{},
				},
				Grafeas: &GrafeasConfig{
					Host: "localhost:8080",
				},
				Elasticsearch: &ElasticsearchConfig{
					Host:    "http://elasticsearch-master:9200",
					Refresh: "true",
				},
				Opa: &OpaConfig{
					Host: "opa.test.na:8181",
				},
				GrpcPort: 50051,
				HttpPort: 50052,
				Debug:    false,
			},
		},
		{
			name:        "Elasticsearch config missing username",
			flags:       []string{"--elasticsearch-password=bar"},
			expectError: true,
		},
		{
			name:        "Elasticsearch missing password",
			flags:       []string{"--elasticsearch-username=foo"},
			expectError: true,
		},
		{
			name:        "Elasticsearch bad refresh option",
			flags:       []string{"--elasticsearch-refresh=foo"},
			expectError: true,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conf, err := Build("rode", tc.flags)

			if tc.expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(conf).To(BeEquivalentTo(tc.expected))
			}
		})
	}

	t.Run("jwt", func(t *testing.T) {
		type providerJSON struct {
			Issuer      string   `json:"issuer"`
			AuthURL     string   `json:"authorization_endpoint"`
			TokenURL    string   `json:"token_endpoint"`
			JWKSURL     string   `json:"jwks_uri"`
			UserInfoURL string   `json:"userinfo_endpoint"`
			Algorithms  []string `json:"id_token_signing_alg_values_supported"`
		}

		issuer := "http://localhost:8080/auth/realms/test"
		wellknown := "/.well-known/openid-configuration"
		responseBytes, err := json.Marshal(&providerJSON{
			Issuer:      issuer,
			AuthURL:     "",
			TokenURL:    "",
			JWKSURL:     "",
			UserInfoURL: "",
			Algorithms:  []string{""},
		})
		Expect(err).ToNot(HaveOccurred())

		t.Run("should be successful", func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.Deactivate()

			httpmock.RegisterResponder("GET", issuer+wellknown, func(request *http.Request) (*http.Response, error) {
				return httpmock.NewStringResponse(http.StatusOK, string(responseBytes)), nil
			})

			c, err := Build("rode", []string{fmt.Sprintf("--jwt-issuer=%s", issuer)})
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Auth.JWT.Issuer).To(Equal(issuer))
		})

		t.Run("should be successful with required audience", func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.Deactivate()

			audience := gofakeit.LetterN(10)

			httpmock.RegisterResponder("GET", issuer+wellknown, func(request *http.Request) (*http.Response, error) {
				return httpmock.NewStringResponse(http.StatusOK, string(responseBytes)), nil
			})

			c, err := Build("rode", []string{fmt.Sprintf("--jwt-issuer=%s", issuer), fmt.Sprintf("--jwt-required-audience=%s", audience)})
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Auth.JWT.Issuer).To(Equal(issuer))
			Expect(c.Auth.JWT.RequiredAudience).To(Equal(audience))
		})

		t.Run("should fail if fetching the openid discovery document fails", func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.Deactivate()

			httpmock.RegisterResponder("GET", issuer+wellknown, func(request *http.Request) (*http.Response, error) {
				return httpmock.NewStringResponse(http.StatusInternalServerError, "error"), nil
			})

			_, err := Build("rode", []string{fmt.Sprintf("--jwt-issuer=%s", issuer)})
			Expect(err).To(HaveOccurred())
		})
	})
}
