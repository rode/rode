package config

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestConfig(t *testing.T) {
	Expect := NewGomegaWithT(t).Expect

	for _, tc := range []struct {
		name        string
		flags       []string
		expected    *Config
		expectError bool
	}{
		{
			name:  "defaults",
			flags: []string{},
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{},
					JWT:   &JWTAuthConfig{},
				},
				Grafeas: &GrafeasConfig{
					Host: "localhost:8080",
				},
				Port:  50051,
				Debug: false,
			},
		},
		{
			name:        "bad port",
			flags:       []string{"--port=foo"},
			expectError: true,
		},
		{
			name:        "bad debug",
			flags:       []string{"--debug=bar"},
			expectError: true,
		},
		{
			name:  "basic auth",
			flags: []string{"--basic-auth-username=foo", "--basic-auth-password=bar"},
			expected: &Config{
				Auth: &AuthConfig{
					Basic: &BasicAuthConfig{
						Username: "foo",
						Password: "bar",
					},
					JWT: &JWTAuthConfig{},
				},
				Grafeas: &GrafeasConfig{
					Host: "localhost:8080",
				},
				Port:  50051,
				Debug: false,
			},
		},
		{
			name:        "basic auth missing username",
			flags:       []string{"--basic-auth-password=bar"},
			expectError: true,
		},
		{
			name:        "basic auth missing password",
			flags:       []string{"--basic-auth-username=foo"},
			expectError: true,
		},
		{
			name:        "jwt required audience without issuer",
			flags:       []string{"--jwt-required-audience=foo"},
			expectError: true,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, err := Build("rode", tc.flags)

			if tc.expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(c).To(BeEquivalentTo(tc.expected))
			}
		})
	}
}
