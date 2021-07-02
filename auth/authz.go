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

	"github.com/rode/rode/pkg/util"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/scylladb/go-set/strset"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type AuthorizationInterceptor interface {
	Authorize(context.Context) (context.Context, error)
	LoadServicePermissions(serviceInfo map[string]grpc.ServiceInfo) error
}

type authorizationInterceptor struct {
	logger            *zap.Logger
	roleRegistry      RoleRegistry
	methodPermissions map[string][]string
}

func NewAuthorizationInterceptor(logger *zap.Logger, registry RoleRegistry) AuthorizationInterceptor {
	return &authorizationInterceptor{
		logger,
		registry,
		map[string][]string{},
	}
}

func (a *authorizationInterceptor) LoadServicePermissions(serviceInfo map[string]grpc.ServiceInfo) error {
	for serviceName := range serviceInfo {
		fd, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))

		if err != nil {
			return err
		}
		serviceDescriptor := fd.(protoreflect.ServiceDescriptor)
		for i := 0; i < serviceDescriptor.Methods().Len(); i++ {
			method := serviceDescriptor.Methods().Get(i)
			authz := proto.GetExtension(method.Options(), pb.E_Authorization).(*pb.Authorization)
			if authz == nil {
				continue
			}

			fullName := fmt.Sprintf("/%s/%s", serviceName, method.Name())
			a.methodPermissions[fullName] = authz.Permissions
		}
	}

	return nil
}

func (a *authorizationInterceptor) Authorize(ctx context.Context) (context.Context, error) {
	log := a.logger.Named("Authorize")
	methodName, ok := grpc.Method(ctx)
	if !ok {
		return nil, util.GrpcInternalError(log, "unable to determine server method", nil)
	}

	log = log.With(zap.String("method", methodName))
	methodPermissions, ok := a.methodPermissions[methodName]
	if !ok {
		return ctx, nil
	}

	requiredPermissions := strset.New(methodPermissions...)
	roles, ok := ctx.Value(rolesCtxKey).([]Role)
	// the authentication interceptors should always set a fallback role if no credentials were presented
	// however, we need to handle this case regardless
	if !ok {
		return nil, util.GrpcErrorWithCode(log, "no assigned roles", nil, codes.PermissionDenied)
	}

	callerPermissions := strset.New()
	for _, r := range roles {
		var permissions []string
		for _, p := range a.roleRegistry.GetRolePermissions(r) {
			permissions = append(permissions, string(p))
		}

		callerPermissions.Add(permissions...)
	}

	if !callerPermissions.IsSubset(requiredPermissions) {
		return nil, util.GrpcErrorWithCode(log, "missing required permissions for call", nil, codes.PermissionDenied)
	}

	return ctx, nil
}
