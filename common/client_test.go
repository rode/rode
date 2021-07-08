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
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pb "github.com/rode/rode/proto/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
)

var _ = Describe("client", func() {
	var (
		actualRodeClient pb.RodeClient
		actualError      error

		expectedConfig *ClientConfig

		fakeListener *bufconn.Listener
		fakeServer   *grpc.Server
	)

	BeforeEach(func() {
		httpmock.Activate()
		httpmock.ActivateNonDefault(insecureOauthHttpClient)

		fakeServer = grpc.NewServer()
		pb.RegisterRodeServer(fakeServer, &pb.UnimplementedRodeServer{})

		fakeListener = bufconn.Listen(1024 * 1024)
		dialOptions = []grpc.DialOption{
			grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
				return fakeListener.Dial()
			}),
		}

		go fakeServer.Serve(fakeListener)

		expectedConfig = &ClientConfig{
			Rode: &RodeClientConfig{
				Host:                     fakeListener.Addr().String(),
				DisableTransportSecurity: true,
			},
		}
	})

	JustBeforeEach(func() {
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

	// TODO: figure out how to make this not take 10 seconds
	When("connecting to the server fails", func() {
		BeforeEach(func() {
			dialOptions = []grpc.DialOption{
				grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
					return nil, errors.New("failed connecting to server")
				}),
			}
		})

		It("should return an error", func() {
			Expect(actualRodeClient).To(BeNil())
			Expect(actualError).To(HaveOccurred())
		})
	})
})
