package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
}

type authorizationInterceptor struct{
	logger *zap.Logger
}

func NewAuthorizationInterceptor(logger *zap.Logger) AuthorizationInterceptor {
	return &authorizationInterceptor{logger}
}

func (a *authorizationInterceptor) Authorize(ctx context.Context) (context.Context, error)  {
	log := a.logger.Named("Authorize")
	methodFullName, ok := grpc.Method(ctx)
	if !ok {
		return nil, errors.New("unable to determine server method")
	}

	log = log.With(zap.String("method", methodFullName))
	 log.Debug("Validating request authorization")

	parts := strings.Split(methodFullName, "/")
	serviceName := parts[1]
	methodName := parts[2]

	fd, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, fmt.Errorf("error looking up service descriptor: %v", err)
	}
	svcd := fd.(protoreflect.ServiceDescriptor)
	method := svcd.Methods().ByName(protoreflect.Name(methodName))
	authz := proto.GetExtension(method.Options(), pb.E_Authorization).(*pb.Authorization)
	requiredPermissions := strset.New(authz.Permissions...)

	roles, ok := ctx.Value(rolesCtxKey).([]Role)
	if !ok {
		return nil, errors.New("missing roles")
	}

	registry := NewRoleRegistry()
	callerPermissions := strset.New()
	for _, r := range roles {
		var permissions []string
		for _, p := range registry.GetRolePermissions(r) {
			permissions = append(permissions, string(p))
		}

		callerPermissions.Add(permissions...)
	}

	if !callerPermissions.IsSubset(requiredPermissions) {
		return nil, status.Error(codes.PermissionDenied, "missing required permissions for call")
	}

	return ctx, nil
}
