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
	"encoding/base64"
	"strings"

	"github.com/Jeffail/gabs/v2"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/rode/rode/config"
	"github.com/rode/rode/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

var rolesCtxKey = "roles"

type authenticator struct {
	authConfig   *config.AuthConfig
	logger       *zap.Logger
	roleRegistry RoleRegistry
}

type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
}

func NewAuthenticator(authConfig *config.AuthConfig, logger *zap.Logger, registry RoleRegistry) Authenticator {
	return &authenticator{
		authConfig,
		logger,
		registry,
	}
}

func (a *authenticator) Authenticate(ctx context.Context) (context.Context, error) {
	log := a.logger.Named("Authenticate")
	authzHeader := metautils.ExtractIncoming(ctx).Get("authorization")

	if authzHeader == "" {
		return context.WithValue(ctx, rolesCtxKey, []Role{RoleAnonymous}), nil
	}

	if a.authConfig.Basic.Username != "" && a.authConfig.Basic.Password != "" {
		return a.basic(ctx, log.With(zap.String("authMethod", "basic")))
	}

	if a.authConfig.OIDC.Issuer != "" {
		return a.oidc(ctx, log.With(zap.String("authMethod", "oidc")))
	}

	return ctx, nil
}

func (a *authenticator) basic(ctx context.Context, log *zap.Logger) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "basic")
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, util.GrpcErrorWithCode(log, "error decoding auth token", err, codes.Unauthenticated)
	}

	parts := strings.Split(string(data), ":")
	if len(parts) != 2 {
		return nil, util.GrpcErrorWithCode(log, "expected auth token to follow format ${username}:${password}", nil, codes.Unauthenticated)
	}

	if a.authConfig.Basic.Username == parts[0] && a.authConfig.Basic.Password == parts[1] {
		return context.WithValue(ctx, rolesCtxKey, []Role{RoleAdministrator}), nil
	}

	return nil, util.GrpcErrorWithCode(log, "invalid username or password", nil, codes.Unauthenticated)
}

func (a *authenticator) oidc(ctx context.Context, log *zap.Logger) (context.Context, error) {
	rawToken, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	token, err := a.authConfig.OIDC.Verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, util.GrpcErrorWithCode(log, "error validating jwt", err, codes.Unauthenticated)
	}

	var claims map[string]interface{}
	if err = token.Claims(&claims); err != nil {
		return nil, util.GrpcErrorWithCode(log, "error unmarshalling claims", err, codes.Unauthenticated)
	}

	allRoles, ok := gabs.Wrap(claims).Path(a.authConfig.OIDC.RoleClaimPath).Data().([]interface{})
	if !ok {
		return nil, util.GrpcErrorWithCode(log, "missing roles claim", nil, codes.Unauthenticated)
	}

	var roles []Role
	for _, roleName := range allRoles {
		if role := a.roleRegistry.GetRoleByName(roleName.(string)); role != "" {
			roles = append(roles, role)
		}
	}

	if len(roles) == 0 {
		roles = append(roles, RoleAnonymous)
	}

	return context.WithValue(ctx, rolesCtxKey, roles), nil
}
