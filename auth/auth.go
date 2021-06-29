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
	"github.com/rode/rode/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var rolesCtxKey = "roles"

type authenticator struct {
	authConfig *config.AuthConfig
}

type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
}

func NewAuthenticator(authConfig *config.AuthConfig) Authenticator {
	return &authenticator{
		authConfig: authConfig,
	}
}

func (a *authenticator) Authenticate(ctx context.Context) (context.Context, error) {
	if a.authConfig.Basic.Username != "" && a.authConfig.Basic.Password != "" {
		return a.basic(ctx)
	}

	if a.authConfig.JWT.Issuer != "" {
		return a.jwt(ctx)
	}

	return ctx, nil
}

func (a *authenticator) basic(ctx context.Context) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "basic")
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "error decoding auth token: %v", err)
	}

	parts := strings.Split(string(data), ":")
	if len(parts) != 2 {
		return nil, status.Errorf(codes.Unauthenticated, "expected auth token to follow format ${username}:${password}")
	}

	if a.authConfig.Basic.Username == parts[0] && a.authConfig.Basic.Password == parts[1] {
		return ctx, nil
	}

	return nil, status.Error(codes.Unauthenticated, "invalid username or password")
}

func (a *authenticator) jwt(ctx context.Context) (context.Context, error) {
	rawToken, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	token, err := a.authConfig.JWT.Verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "error validating jwt: %v", err)
	}

	var claims map[string]interface{}
	if err = token.Claims(&claims); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "error unmarshalling claims: %v", err)
	}

	allRoles, ok := gabs.Wrap(claims).Path(a.authConfig.JWT.RoleClaimPath).Data().([]interface{})
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing roles claim")
	}

	var roles []Role
	registry := NewRoleRegistry()
	for _, s := range allRoles {
		if r := registry.GetRoleByName(s.(string)); r != "" {
			roles = append(roles, r)
		}
	}

	// TODO: no roles = set Anonymous
	return context.WithValue(ctx, rolesCtxKey, roles), nil
}
