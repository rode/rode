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
package auth

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/health/grpc_health_v1" // imported so that the health service is in the proto registry
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	rodeServiceName       = "rode.v1alpha1.Rode"
	grpcHealthServiceName = "grpc.health.v1.Health"
)

var _ = Describe("AuthorizationInterceptor", func() {
	var (
		ctx         context.Context
		interceptor AuthorizationInterceptor
		registry    = NewRoleRegistry()
	)

	BeforeEach(func() {
		interceptor = NewAuthorizationInterceptor(logger, registry)
	})

	Context("LoadServicePermissions", func() {
		var (
			actualError error
			serviceInfo map[string]grpc.ServiceInfo
		)
		BeforeEach(func() {
			serviceInfo = map[string]grpc.ServiceInfo{
				rodeServiceName: {},
			}
		})

		JustBeforeEach(func() {
			actualError = interceptor.LoadServicePermissions(serviceInfo)
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		When("the an RPC does not have the permissions extension", func() {
			BeforeEach(func() {
				serviceInfo[grpcHealthServiceName] = grpc.ServiceInfo{}
			})

			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		When("the service is not registered", func() {
			BeforeEach(func() {
				serviceInfo[fake.Word()] = grpc.ServiceInfo{}
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
			})
		})
	})

	Context("Authorize", func() {
		var (
			actualCtx   context.Context
			actualError error

			methodName      string
			serverTransport *fakeServerTransportStream
		)

		BeforeEach(func() {
			methodName = fake.RandomString([]string{
				grpcMethodName("ListPolicies"),
				grpcMethodName("GetPolicy"),
				grpcMethodName("UpdatePolicy"),
			})
			serverTransport = &fakeServerTransportStream{method: methodName}
			ctx = context.WithValue(
				grpc.NewContextWithServerTransportStream(context.Background(), serverTransport),
				rolesCtxKey,
				[]Role{RoleAdministrator},
			)
		})

		JustBeforeEach(func() {
			Expect(interceptor.LoadServicePermissions(map[string]grpc.ServiceInfo{
				rodeServiceName: {},
			})).NotTo(HaveOccurred())

			actualCtx, actualError = interceptor.Authorize(ctx)
		})

		It("should not return an error", func() {
			Expect(actualError).NotTo(HaveOccurred())
		})

		It("should return the same context", func() {
			Expect(actualCtx).To(Equal(ctx))
		})

		When("the gRPC method isn't in the context", func() {
			BeforeEach(func() {
				ctx = context.Background()
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.Internal))
			})
		})

		When("the role claim isn't in the context", func() {
			BeforeEach(func() {
				ctx = grpc.NewContextWithServerTransportStream(context.Background(), serverTransport)
			})

			It("should return an error", func() {
				Expect(actualCtx).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.PermissionDenied))
			})

		})

		When("the caller doesn't have the required permissions", func() {
			BeforeEach(func() {
				serverTransport.method = grpcMethodName("UpdatePolicy")
				ctx = context.WithValue(ctx, rolesCtxKey, []Role{RoleAnonymous})
			})

			It("should return an error", func() {
				Expect(actualCtx).To(BeNil())
				Expect(actualError).To(HaveOccurred())
				Expect(getGRPCStatusFromError(actualError).Code()).To(Equal(codes.PermissionDenied))
			})
		})

		When("the rpc has no associated permissions", func() {
			BeforeEach(func() {
				serverTransport.method = fake.Word()
			})

			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should return the context", func() {
				Expect(actualCtx).To(Equal(ctx))
			})
		})
	})
})

func getGRPCStatusFromError(err error) *status.Status {
	s, ok := status.FromError(err)
	Expect(ok).To(BeTrue(), "Expected error to be a gRPC status")

	return s
}

func grpcMethodName(name string) string {
	return fmt.Sprintf("/%s/%s", rodeServiceName, name)
}

type fakeServerTransportStream struct {
	method string
}

func (f *fakeServerTransportStream) Method() string {
	return f.method
}

func (f *fakeServerTransportStream) SetHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeServerTransportStream) SendHeader(_ metadata.MD) error {
	return nil
}

func (f *fakeServerTransportStream) SetTrailer(_ metadata.MD) error {
	return nil
}
