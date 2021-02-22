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
	"github.com/brianvoe/gofakeit/v5"
	"github.com/coreos/go-oidc"
	"github.com/dgrijalva/jwt-go"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
	"time"
)

func TestAuth(t *testing.T) {
	Expect := NewGomegaWithT(t).Expect
	ctx := context.Background()

	t.Run("no authentication", func(t *testing.T) {
		authenticator := NewAuthenticator(&config.AuthConfig{
			Basic: &config.BasicAuthConfig{},
			JWT:   &config.JWTAuthConfig{},
		})

		_, err := authenticator.Authenticate(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	t.Run("basic authentication", func(t *testing.T) {
		authConfig := &config.AuthConfig{
			Basic: &config.BasicAuthConfig{
				Username: gofakeit.LetterN(10),
				Password: gofakeit.LetterN(10),
			},
		}
		authenticator := NewAuthenticator(authConfig)

		t.Run("should be successful when using the correct credentials", func(t *testing.T) {
			_, err := authenticator.Authenticate(createCtxWithBasicAuth(ctx, authConfig.Basic.Username, authConfig.Basic.Password))
			Expect(err).ToNot(HaveOccurred())
		})

		t.Run("should fail when using incorrect credentials", func(t *testing.T) {
			_, err := authenticator.Authenticate(createCtxWithBasicAuth(ctx, gofakeit.LetterN(10), gofakeit.LetterN(10)))
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})

		t.Run("should fail when providing no credentials", func(t *testing.T) {
			_, err := authenticator.Authenticate(ctx)
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})

		t.Run("should fail when using incorrect format for basic auth", func(t *testing.T) {
			_, err := authenticator.Authenticate(createCtxWithBasicAuth(ctx, fmt.Sprintf("%s:%s", gofakeit.LetterN(10), gofakeit.LetterN(10)), gofakeit.LetterN(10)))
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})

		t.Run("should fail when base64 decoding fails", func(t *testing.T) {
			meta := metautils.NiceMD(metadata.New(map[string]string{
				"authorization": fmt.Sprintf("Basic %s", gofakeit.LetterN(10)),
			}))

			_, err := authenticator.Authenticate(meta.ToIncoming(ctx))
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})
	})

	t.Run("jwt authentication", func(t *testing.T) {
		issuer := gofakeit.LetterN(10)
		keySet := &fakeKeySet{}
		clientId := gofakeit.LetterN(10)
		verifier := oidc.NewVerifier(issuer, keySet, &oidc.Config{
			ClientID: clientId,
		})

		authConfig := &config.AuthConfig{
			Basic: &config.BasicAuthConfig{},
			JWT: &config.JWTAuthConfig{
				Issuer:   issuer,
				Verifier: verifier,
			},
		}
		authenticator := NewAuthenticator(authConfig)

		t.Run("should be successful when jwt validation is successful", func(t *testing.T) {
			ctx, payload := createCtxWithJWT(ctx, issuer, clientId, time.Now().Add(time.Minute*1).Unix())
			keySet.jwtPayload = payload
			keySet.shouldVerify = true

			_, err := authenticator.Authenticate(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		t.Run("should fail when jwt validation fails", func(t *testing.T) {
			ctx, payload := createCtxWithJWT(ctx, issuer, clientId, time.Now().Add(time.Minute*1).Unix())
			keySet.jwtPayload = payload
			keySet.shouldVerify = false

			_, err := authenticator.Authenticate(ctx)
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})

		t.Run("should fail when a bearer token is not specified", func(t *testing.T) {
			// a basic auth attempt would fail here
			meta := metautils.NiceMD(metadata.New(map[string]string{
				"authorization": fmt.Sprintf("Basic %s", gofakeit.LetterN(10)),
			}))

			_, err := authenticator.Authenticate(meta.ToIncoming(ctx))
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})
	})
}

type fakeKeySet struct {
	shouldVerify bool
	jwtPayload   []byte
}

func (f *fakeKeySet) VerifySignature(context.Context, string) ([]byte, error) {
	if f.shouldVerify {
		return f.jwtPayload, nil
	}

	return nil, errors.New(gofakeit.LetterN(10))
}

func expectUnauthenticatedErrorToHaveOccurred(t *testing.T, err error) {
	Expect := NewGomegaWithT(t).Expect
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

func createCtxWithJWT(ctx context.Context, issuer, audience string, expires int64) (context.Context, []byte) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		Issuer:    issuer,
		Audience:  audience,
		ExpiresAt: expires,
	})
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signedString, _ := token.SignedString(key)

	meta := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", signedString),
	})

	payload, _ := jwt.DecodeSegment(strings.Split(signedString, ".")[1])

	return metautils.NiceMD(meta).ToIncoming(ctx), payload
}
