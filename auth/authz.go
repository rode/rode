package auth

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/scylladb/go-set/strset"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, errors.New("unable to determine server method")
	}

	log = log.With(zap.String("method", methodName))
	methodPermissions, ok := a.methodPermissions[methodName]
	if !ok {
		return ctx, nil
	}

	requiredPermissions := strset.New(methodPermissions...)
	roles, ok := ctx.Value(rolesCtxKey).([]Role)
	if !ok {
		return nil, errors.New("missing roles")
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
		return nil, status.Error(codes.PermissionDenied, "missing required permissions for call")
	}

	return ctx, nil
}
