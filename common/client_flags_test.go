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

package common

import (
	"flag"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("client flags", func() {
	var (
		flagSet      *flag.FlagSet
		actualConfig *ClientConfig

		expectedClientId     = fake.LetterN(10)
		expectedClientSecret = fake.UUID()
		expectedTokenUrl     = fake.URL()
		expectedScopes       = strings.Join([]string{fake.LetterN(10), fake.LetterN(10)}, " ")

		expectedUsername = fake.LetterN(10)
		expectedPassword = fake.LetterN(10)

		expectedRodeHost = fake.LetterN(10)
	)
	BeforeEach(func() {
		flagSet = flag.NewFlagSet("rode-client", flag.ContinueOnError)

		actualConfig = SetupRodeClientFlags(flagSet)
	})

	DescribeTable("flag parsing",
		func(flags []string, expectedConfig *ClientConfig) {
			err := flagSet.Parse(flags)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualConfig).To(Equal(expectedConfig))
		},
		Entry("defaults", []string{}, &ClientConfig{
			Rode: &RodeClientConfig{
				Host: "rode:50051",
			},
			OIDCAuth:  &OIDCAuthConfig{},
			BasicAuth: &BasicAuthConfig{},
		}),
		Entry("rode config", []string{
			"--rode-host=" + expectedRodeHost,
			"--rode-insecure-disable-transport-security",
		}, &ClientConfig{
			Rode: &RodeClientConfig{
				Host:                     expectedRodeHost,
				DisableTransportSecurity: true,
			},
			OIDCAuth:  &OIDCAuthConfig{},
			BasicAuth: &BasicAuthConfig{},
		}),
		Entry("oidc auth", []string{
			"--oidc-client-id=" + expectedClientId,
			"--oidc-client-secret=" + expectedClientSecret,
			"--oidc-token-url=" + expectedTokenUrl,
			"--oidc-tls-insecure-skip-verify",
			"--oidc-scopes=" + expectedScopes,
		}, &ClientConfig{
			Rode: &RodeClientConfig{
				Host: "rode:50051",
			},
			OIDCAuth: &OIDCAuthConfig{
				ClientID:              expectedClientId,
				ClientSecret:          expectedClientSecret,
				Scopes:                expectedScopes,
				TokenURL:              expectedTokenUrl,
				TlsInsecureSkipVerify: true,
			},
			BasicAuth: &BasicAuthConfig{},
		}),
		Entry("basic auth", []string{
			"--basic-auth-username=" + expectedUsername,
			"--basic-auth-password=" + expectedPassword,
		}, &ClientConfig{
			Rode: &RodeClientConfig{
				Host: "rode:50051",
			},
			OIDCAuth: &OIDCAuthConfig{},
			BasicAuth: &BasicAuthConfig{
				Username: expectedUsername,
				Password: expectedPassword,
			},
		}),
		Entry("proxy auth", []string{
			"--proxy-auth",
		}, &ClientConfig{
			Rode: &RodeClientConfig{
				Host: "rode:50051",
			},
			OIDCAuth:  &OIDCAuthConfig{},
			BasicAuth: &BasicAuthConfig{},
			ProxyAuth: true,
		}),
	)
})
