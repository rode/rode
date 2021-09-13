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
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/golang-jwt/jwt"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var _ = Describe("Auth", func() {
	var (
		authConfig *config.AuthConfig
		ctx context.Context
		authenticator Authenticator

		actualCtx context.Context
		actualError error

	)

	BeforeEach(func() {
		ctx = context.Background()
		registry := NewRoleRegistry()
		authConfig = &config.AuthConfig{
			Basic: &config.BasicAuthConfig{},
			OIDC:  &config.OIDCAuthConfig{},
		}

		authenticator = NewAuthenticator(authConfig, logger, registry)
	})

	JustBeforeEach(func() {
		actualCtx, actualError = authenticator.Authenticate(ctx)
	})

	When("no authentication is configured", func() {
		It("should allow the request", func() {
			Expect(actualError).ToNot(HaveOccurred())
		})
	})

	When("no authentication is configured but the authorization header is set", func() {
		BeforeEach(func() {
			meta := metautils.NiceMD(metadata.New(map[string]string{
				"authorization": fake.Word(),
			}))

			ctx = meta.ToIncoming(ctx)
		})

		It("should allow the request", func() {
			Expect(actualError).ToNot(HaveOccurred())
		})
	})

	Context("basic authentication", func() {
		BeforeEach(func() {
			authConfig.Basic.Username = fake.LetterN(10)
			authConfig.Basic.Password = fake.LetterN(10)
		})

		When("the correct credentials are presented", func() {
			BeforeEach(func() {
				ctx = createCtxWithBasicAuth(ctx, authConfig.Basic.Username, authConfig.Basic.Password)
			})

			It("should allow the request", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		When("the credentials are incorrect", func() {
			BeforeEach(func() {
				ctx = createCtxWithBasicAuth(ctx, fake.LetterN(10), fake.LetterN(10))
			})

			It("should deny the request", func() {
				expectUnauthenticatedErrorToHaveOccurred(actualError)
			})
		})

		When("the authorization header is malformed", func() {
			BeforeEach(func() {
				meta := metautils.NiceMD(metadata.New(map[string]string{
					"authorization": "Basic",
				}))
				ctx = meta.ToIncoming(ctx)
			})

			It("should deny the request", func() {
				expectUnauthenticatedErrorToHaveOccurred(actualError)
			})
		})

		When("the authorization header doesn't have the correct format", func() {
			BeforeEach(func() {
				ctx = createCtxWithBasicAuth(ctx, fmt.Sprintf("%s:%s", fake.LetterN(10), fake.LetterN(10)), fake.LetterN(10))
			})

			It("should deny the request", func() {
				expectUnauthenticatedErrorToHaveOccurred(actualError)
			})
		})

		When("the base64 decode fails", func() {
			BeforeEach(func() {
				meta := metautils.NiceMD(metadata.New(map[string]string{
					"authorization": fmt.Sprintf("Basic %s", fake.LetterN(10)),
				}))
				ctx = meta.ToIncoming(ctx)
			})

			It("should deny the request", func() {
				expectUnauthenticatedErrorToHaveOccurred(actualError)
			})
		})
	})

	Context("OIDC authentication", func() {
		var (
			issuer string
			keySet *fakeKeySet
			clientId string
			verifier *oidc.IDTokenVerifier
			payload []byte
		)

		BeforeEach(func() {
			issuer = fake.LetterN(10)
			keySet = &fakeKeySet{}
			clientId = fake.LetterN(10)
			verifier = oidc.NewVerifier(issuer, keySet, &oidc.Config{
				ClientID: clientId,
			})

			authConfig.OIDC.Issuer = issuer
			authConfig.OIDC.Verifier = verifier
			authConfig.OIDC.RoleClaimPath = "roles"
		})

		When("jwt validation is successful", func() {
			var (
				role string
			)

			BeforeEach(func() {
				role = fake.RandomString([]string{
					string(RoleAdministrator),
					string(RoleApplicationDeveloper),
					string(RoleEnforcer),
				})

				ctx, payload = createCtxWithJWT(ctx, issuer, clientId, role, time.Now().Add(time.Minute*1).Unix())
				keySet.jwtPayload = payload
				keySet.shouldVerify = true
			})

			It("should allow the request", func() {
				Expect(actualError).NotTo(HaveOccurred())
				Expect(actualCtx.Value(rolesCtxKey)).To(Equal([]Role{Role(role)}))
			})
		})

		When("there the role claim does not contain any known roles", func() {
			BeforeEach(func() {
				ctx, payload = createCtxWithJWT(ctx, issuer, clientId, fake.Word(), time.Now().Add(time.Minute*1).Unix())

				keySet.jwtPayload = payload
				keySet.shouldVerify = true
			})

			It("should set the anonymous role", func() {
				Expect(actualError).NotTo(HaveOccurred())
				Expect(actualCtx.Value(rolesCtxKey)).To(Equal([]Role{RoleAnonymous}))
			})
		})

		When("the roles claim is missing", func() {
			BeforeEach(func() {
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, &jwt.StandardClaims{
					Issuer:    issuer,
					Audience:  clientId,
					ExpiresAt: time.Now().Add(time.Minute * 1).Unix(),
				})

				key, _ := rsa.GenerateKey(rand.Reader, 2048)
				signedToken, _ := token.SignedString(key)
				meta := metautils.NiceMD(metadata.New(map[string]string{
					"authorization": fmt.Sprintf("Bearer %s", signedToken),
				}))
				ctx = meta.ToIncoming(ctx)

				payload, _ = jwt.DecodeSegment(strings.Split(signedToken, ".")[1])
				keySet.jwtPayload = payload
				keySet.shouldVerify = true
			})

			It("should deny the request", func() {
				expectUnauthenticatedErrorToHaveOccurred(actualError)
			})
		})

		When("jwt validation fails", func() {
			BeforeEach(func() {
				ctx, payload = createCtxWithJWT(ctx, issuer, clientId, string(RoleAdministrator), time.Now().Add(time.Minute*1).Unix())
				keySet.jwtPayload = payload
				keySet.shouldVerify = false
			})

			It("should deny the request", func() {
				expectUnauthenticatedErrorToHaveOccurred(actualError)
			})
		})

		When("a bearer token is not specified", func() {
			BeforeEach(func() {
				meta := metautils.NiceMD(metadata.New(map[string]string{
					"authorization": fmt.Sprintf("Basic %s", fake.LetterN(10)),
				}))
				ctx = meta.ToIncoming(ctx)
			})

			It("should deny the request", func() {
				expectUnauthenticatedErrorToHaveOccurred(actualError)
			})
		})
	})
})

type fakeKeySet struct {
	shouldVerify bool
	jwtPayload   []byte
}

func (f *fakeKeySet) VerifySignature(context.Context, string) ([]byte, error) {
	if f.shouldVerify {
		return f.jwtPayload, nil
	}

	return nil, errors.New(fake.LetterN(10))
}

type fakeClaims struct {
	*jwt.StandardClaims
	Roles []string `json:"roles"`
}

func expectUnauthenticatedErrorToHaveOccurred( err error) {
	Expect(err).To(HaveOccurred())
	s, ok := status.FromError(err)

	Expect(ok).To(BeTrue(), "expected error to have been produced from the grpc/status package")
	Expect(s.Code()).To(Equal(codes.Unauthenticated))
}

func createCtxWithBasicAuth(ctx context.Context, username, password string) context.Context {
	token := fmt.Sprintf("%s:%s", username, password)
	enc := base64.StdEncoding.EncodeToString([]byte(token))

	meta := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Basic %s", enc),
	})

	return metautils.NiceMD(meta).ToIncoming(ctx)
}

func createCtxWithJWT(ctx context.Context, issuer, audience, role string, expires int64) (context.Context, []byte) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &fakeClaims{
		StandardClaims: &jwt.StandardClaims{
			Issuer:    issuer,
			Audience:  audience,
			ExpiresAt: expires,
		},
		Roles: []string{role},
	})
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signedString, _ := token.SignedString(key)

	meta := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", signedString),
	})

	payload, _ := jwt.DecodeSegment(strings.Split(signedString, ".")[1])

	return metautils.NiceMD(meta).ToIncoming(ctx), payload
}
