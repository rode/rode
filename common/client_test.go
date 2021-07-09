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
	"encoding/base64"
	"errors"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"net/http"
	"strings"
	"time"
)

var _ = Describe("client", func() {
	var (
		actualRodeClient pb.RodeClient
		actualError      error

		expectedConfig *ClientConfig

		fakeListener *bufconn.Listener
		fakeServer   *grpc.Server

		actualAuthorizationHeader string
	)

	BeforeEach(func() {
		httpmock.Activate()
		httpmock.ActivateNonDefault(insecureOauthHttpClient)

		fakeAuthnFunc := func(ctx context.Context) (context.Context, error) {
			actualAuthorizationHeader = metautils.ExtractIncoming(ctx).Get("authorization")

			return ctx, nil
		}

		fakeServer = grpc.NewServer(
			grpc_middleware.WithStreamServerChain(
				grpc_auth.StreamServerInterceptor(fakeAuthnFunc),
			),
			grpc_middleware.WithUnaryServerChain(
				grpc_auth.UnaryServerInterceptor(fakeAuthnFunc),
			),
		)

		pb.RegisterRodeServer(fakeServer, &pb.UnimplementedRodeServer{})

		fakeListener = bufconn.Listen(1024 * 1024)
		dialOptions = []grpc.DialOption{
			grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
				return fakeListener.Dial()
			}),
		}

		expectedConfig = &ClientConfig{
			Rode: &RodeClientConfig{
				Host:                     fakeListener.Addr().String(),
				DisableTransportSecurity: true,
			},
		}
	})

	JustBeforeEach(func() {
		go fakeServer.Serve(fakeListener)
		actualRodeClient, actualError = NewRodeClient(expectedConfig)
	})

	AfterEach(func() {
		httpmock.Deactivate()
		fakeServer.GracefulStop()
	})

	It("should return a rode client", func() {
		Expect(actualRodeClient).ToNot(BeNil())
		Expect(actualError).ToNot(HaveOccurred())
	})

	When("no config is specified", func() {
		BeforeEach(func() {
			expectedConfig = nil
		})

		It("should return an error", func() {
			Expect(actualRodeClient).To(BeNil())
			Expect(actualError).To(HaveOccurred())
		})
	})

	When("no rode config is specified", func() {
		BeforeEach(func() {
			expectedConfig.Rode = nil
		})

		It("should return an error", func() {
			Expect(actualRodeClient).To(BeNil())
			Expect(actualError).To(HaveOccurred())
		})
	})

	When("no rode config is specified", func() {
		BeforeEach(func() {
			expectedConfig.Rode.Host = ""
		})

		It("should return an error", func() {
			Expect(actualRodeClient).To(BeNil())
			Expect(actualError).To(HaveOccurred())
		})
	})

	When("more than one authentication method is specified", func() {
		BeforeEach(func() {
			expectedConfig.BasicAuth = &BasicAuthConfig{}
			expectedConfig.JWTAuth = &JWTAuthConfig{}
		})

		It("should return an error", func() {
			Expect(actualRodeClient).To(BeNil())
			Expect(actualError).To(HaveOccurred())
		})
	})

	When("JWT auth is configured", func() {
		type tokenResponse struct {
			AccessToken string `json:"access_token"`
		}

		var (
			expectedClientId     string
			expectedClientSecret string
			expectedTokenUrl     string
			expectedAccessToken  string
		)

		BeforeEach(func() {
			expectedClientId = fake.LetterN(10)
			expectedClientSecret = fake.UUID()
			expectedTokenUrl = fake.URL()
			expectedAccessToken = fake.UUID()

			expectedConfig.JWTAuth = &JWTAuthConfig{
				ClientID:              expectedClientId,
				ClientSecret:          expectedClientSecret,
				TokenURL:              expectedTokenUrl,
				TlsInsecureSkipVerify: false,
			}

			httpmock.RegisterResponder(http.MethodPost, expectedTokenUrl, func(*http.Request) (*http.Response, error) {
				return httpmock.NewJsonResponse(http.StatusOK, &tokenResponse{
					AccessToken: expectedAccessToken,
				})
			})
		})

		It("should return a rode client", func() {
			Expect(actualRodeClient).ToNot(BeNil())
			Expect(actualError).ToNot(HaveOccurred())
		})

		It("should send a bearer authentication header with each request", func() {
			_, _ = actualRodeClient.GetPolicy(context.Background(), &pb.GetPolicyRequest{})

			Expect(actualAuthorizationHeader).ToNot(BeEmpty())

			parts := strings.Split(actualAuthorizationHeader, " ")

			Expect(parts[0]).To(Equal("Bearer"))
			Expect(parts[1]).To(Equal(expectedAccessToken))
		})

		When("getting the access token fails", func() {
			BeforeEach(func() {
				httpmock.Reset()
				httpmock.RegisterResponder(http.MethodPost, expectedTokenUrl, func(*http.Request) (*http.Response, error) {
					return httpmock.NewStringResponse(http.StatusInternalServerError, "error"), nil
				})
			})

			It("should return an error", func() {
				Expect(actualRodeClient).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("tls verification is disabled", func() {
			BeforeEach(func() {
				expectedConfig.JWTAuth.TlsInsecureSkipVerify = true
			})

			It("should use a http client with tls verification disabled", func() {
				_, _ = actualRodeClient.GetPolicy(context.Background(), &pb.GetPolicyRequest{})

				mockTransport, ok := insecureOauthHttpClient.Transport.(*httpmock.MockTransport)
				Expect(ok).To(BeTrue())

				httpmock.GetCallCountInfo()
				Expect(mockTransport.GetCallCountInfo()).ToNot(HaveLen(0))
			})
		})

		When("the client ID is not specified", func() {
			BeforeEach(func() {
				expectedConfig.JWTAuth.ClientID = ""
			})

			It("should return an error", func() {
				Expect(actualRodeClient).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the client secret is not specified", func() {
			BeforeEach(func() {
				expectedConfig.JWTAuth.ClientSecret = ""
			})

			It("should return an error", func() {
				Expect(actualRodeClient).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the token URL is not specified", func() {
			BeforeEach(func() {
				expectedConfig.JWTAuth.TokenURL = ""
			})

			It("should return an error", func() {
				Expect(actualRodeClient).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})
	})

	When("basic auth is configured", func() {
		var (
			expectedUsername string
			expectedPassword string
		)

		BeforeEach(func() {
			expectedUsername = fake.LetterN(10)
			expectedPassword = fake.LetterN(10)

			expectedConfig.BasicAuth = &BasicAuthConfig{
				Username: expectedUsername,
				Password: expectedPassword,
			}
		})

		It("should return a rode client", func() {
			Expect(actualRodeClient).ToNot(BeNil())
			Expect(actualError).ToNot(HaveOccurred())
		})

		It("should send a basic authentication header with each request", func() {
			_, _ = actualRodeClient.GetPolicy(context.Background(), &pb.GetPolicyRequest{})

			Expect(actualAuthorizationHeader).ToNot(BeEmpty())

			parts := strings.Split(actualAuthorizationHeader, " ")

			Expect(parts[0]).To(Equal("Basic"))

			data, err := base64.StdEncoding.DecodeString(parts[1])
			Expect(err).ToNot(HaveOccurred())

			dataParts := strings.Split(string(data), ":")
			Expect(dataParts[0]).To(Equal(expectedUsername))
			Expect(dataParts[1]).To(Equal(expectedPassword))
		})

		When("the username is missing", func() {
			BeforeEach(func() {
				expectedConfig.BasicAuth.Username = ""
			})

			It("should return an error", func() {
				Expect(actualRodeClient).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})

		When("the password is missing", func() {
			BeforeEach(func() {
				expectedConfig.BasicAuth.Password = ""
			})

			It("should return an error", func() {
				Expect(actualRodeClient).To(BeNil())
				Expect(actualError).To(HaveOccurred())
			})
		})
	})

	When("connecting to the server fails", func() {
		BeforeEach(func() {
			contextDuration = 1 * time.Millisecond
			dialOptions = []grpc.DialOption{
				grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
					return nil, errors.New("failed connecting to server")
				}),
			}
		})

		AfterEach(func() {
			contextDuration = 10 * time.Second
		})

		It("should return an error", func() {
			Expect(actualRodeClient).To(BeNil())
			Expect(actualError).To(HaveOccurred())
		})
	})
})
