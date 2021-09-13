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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	DescribeTable("Build", func(tc *testCase) {
		actualConfig, err := Build("rode", tc.flags)

		if tc.expectError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(actualConfig).To(Equal(tc.expected))
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("defaults", &testCase{
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{},
					OIDC: &OIDCAuthConfig{
						RoleClaimPath: "roles",
					},
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
				Port:  50051,
				Debug: false,
			},
		}),
		Entry("bad port", &testCase{
			flags:       []string{"--port=foo"},
			expectError: true,
		}),
		Entry("bad debug value", &testCase{
			flags:       []string{"--debug=bar"},
			expectError: true,
		}),
		Entry("basic auth", &testCase{
			flags: []string{"--basic-auth-username=foo", "--basic-auth-password=bar"},
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{
						Username: "foo",
						Password: "bar",
					},
					OIDC: &OIDCAuthConfig{
						RoleClaimPath: "roles",
					},
					Enabled: true,
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
				Port:  50051,
				Debug: false,
			},
		}),
		Entry("basic auth missing username", &testCase{
			flags:       []string{"--basic-auth-password=bar"},
			expectError: true,
		}),
		Entry("basic auth missing password", &testCase{
			flags:       []string{"--basic-auth-username=foo"},
			expectError: true,
		}),
		Entry("OIDC required audience without issuer", &testCase{
			flags:       []string{"--oidc-required-audience=foo"},
			expectError: true,
		}),
		Entry("OPA host", &testCase{
			flags: []string{"--opa-host=opa.test.na:8181"},
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{},
					OIDC: &OIDCAuthConfig{
						RoleClaimPath: "roles",
					},
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
				Port:  50051,
				Debug: false,
			},
		}),
		Entry("Elasticsearch config missing username", &testCase{
			flags:       []string{"--elasticsearch-password=bar"},
			expectError: true,
		}),
		Entry("Elasticsearch missing password", &testCase{
			flags:       []string{"--elasticsearch-username=foo"},
			expectError: true,
		}),
		Entry("Elasticsearch bad refresh option", &testCase{
			flags:       []string{"--elasticsearch-refresh=foo"},
			expectError: true,
		}),
	)

	Describe("OIDC", func() {
		var (
			issuer = "http://localhost:8080/auth/realms/test"
			wellknown = "/.well-known/openid-configuration"
			responseBytes []byte
			flags []string

			actualConfig *Config
			actualError error
		)

		BeforeEach(func() {
			var err error
			flags = []string{fmt.Sprintf("--oidc-issuer=%s", issuer)}
			responseBytes, err = json.Marshal(&providerJSON{
				Issuer:      issuer,
				AuthURL:     "",
				TokenURL:    "",
				JWKSURL:     "",
				UserInfoURL: "",
				Algorithms:  []string{""},
			})
			Expect(err).NotTo(HaveOccurred())
			httpmock.Activate()
		})

		JustBeforeEach(func() {
			actualConfig, actualError = Build("rode", flags)
		})

		AfterEach(func() {
			httpmock.Deactivate()
		})

		When("the configuration is correct", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("GET", issuer+wellknown, func(request *http.Request) (*http.Response, error) {
					return httpmock.NewStringResponse(http.StatusOK, string(responseBytes)), nil
				})
			})

			It("should be successful", func() {
				Expect(actualError).NotTo(HaveOccurred())
				Expect(actualConfig.Auth.OIDC.Issuer).To(Equal(issuer))
			})
		})

		When("a required audience is set", func() {
			var audience string

			BeforeEach(func() {
				audience = fake.LetterN(10)
				httpmock.RegisterResponder("GET", issuer+wellknown, func(request *http.Request) (*http.Response, error) {
					return httpmock.NewStringResponse(http.StatusOK, string(responseBytes)), nil
				})

				flags = []string{
					fmt.Sprintf("--oidc-issuer=%s", issuer),
					fmt.Sprintf("--oidc-required-audience=%s", audience),
				}
			})

			It("should be successful", func() {
				Expect(actualError).NotTo(HaveOccurred())
				Expect(actualConfig.Auth.OIDC.Issuer).To(Equal(issuer))
				Expect(actualConfig.Auth.OIDC.RequiredAudience).To(Equal(audience))
			})
		})

		When("fetching the OIDC discovery document fails", func() {
			BeforeEach(func() {
				httpmock.RegisterResponder("GET", issuer+wellknown, func(request *http.Request) (*http.Response, error) {
					return httpmock.NewStringResponse(http.StatusInternalServerError, "error"), nil
				})
			})

			It("should fail", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("TLS verification is off", func() {
			var actualTransport *http.Transport

			BeforeEach(func() {
				oidcClientContext = func(ctx context.Context, client *http.Client) context.Context {
					actualTransport = client.Transport.(*http.Transport)
					httpmock.ActivateNonDefault(client)

					httpmock.RegisterResponder("GET", issuer+wellknown, func(request *http.Request) (*http.Response, error) {
						return httpmock.NewStringResponse(http.StatusOK, string(responseBytes)), nil
					})

					return oidc.ClientContext(ctx, client)
				}

				flags = []string{
					"--oidc-tls-insecure-skip-verify=true",
					fmt.Sprintf("--oidc-issuer=%s", issuer),
				}
			})

			It("should provide a custom HTTP client", func() {
				Expect(actualError).NotTo(HaveOccurred())
				Expect(actualTransport).NotTo(BeNil())
				Expect(actualTransport.TLSClientConfig.InsecureSkipVerify).To(BeTrue())
			})
		})
	})
})

type providerJSON struct {
	Issuer      string   `json:"issuer"`
	AuthURL     string   `json:"authorization_endpoint"`
	TokenURL    string   `json:"token_endpoint"`
	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}

type testCase struct {
	flags       []string
	expected    *Config
	expectError bool
}
