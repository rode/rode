package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/brianvoe/gofakeit/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestAuth(t *testing.T) {
	Expect := NewGomegaWithT(t).Expect
	ctx := context.Background()

	t.Run("no authentication", func(t *testing.T) {
		a := NewAuthenticator(&config.AuthConfig{})

		_, err := a.Authenticate(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	t.Run("basic authentication", func(t *testing.T) {
		c := &config.AuthConfig{
			BasicAuthUsername: gofakeit.LetterN(10),
			BasicAuthPassword: gofakeit.LetterN(10),
		}
		a := NewAuthenticator(c)

		t.Run("should be successful when using the correct credentials", func(t *testing.T) {
			_, err := a.Authenticate(createCtxWithBasicAuth(ctx, c.BasicAuthUsername, c.BasicAuthPassword))
			Expect(err).ToNot(HaveOccurred())
		})

		t.Run("should fail when using incorrect credentials", func(t *testing.T) {
			_, err := a.Authenticate(createCtxWithBasicAuth(ctx, gofakeit.LetterN(10), gofakeit.LetterN(10)))
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})

		t.Run("should fail when providing no credentials", func(t *testing.T) {
			_, err := a.Authenticate(ctx)
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})

		t.Run("should fail when using incorrect format for basic auth", func(t *testing.T) {
			_, err := a.Authenticate(createCtxWithBasicAuth(ctx, fmt.Sprintf("%s:%s", gofakeit.LetterN(10), gofakeit.LetterN(10)), gofakeit.LetterN(10)))
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})

		t.Run("should fail when base64 decoding fails", func(t *testing.T) {
			meta := metautils.NiceMD(metadata.New(map[string]string{
				"authorization": fmt.Sprintf("Basic %s", gofakeit.LetterN(10)),
			}))

			_, err := a.Authenticate(meta.ToIncoming(ctx))
			expectUnauthenticatedErrorToHaveOccurred(t, err)
		})
	})
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
