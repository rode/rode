package common

import (
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("client flags", func() {
	var (
		flagSet      *flag.FlagSet
		actualConfig *ClientConfig

		expectedClientId     string
		expectedClientSecret string
		expectedTokenUrl     string

		expectedUsername string
		expectedPassword string

		expectedRodeHost string
	)
	BeforeEach(func() {
		flagSet = flag.NewFlagSet("rode-client", flag.ContinueOnError)
		actualConfig = SetupRodeClientFlags(flagSet)

		expectedClientId = fake.LetterN(10)
		expectedClientSecret = fake.UUID()
		expectedTokenUrl = fake.URL()

		expectedUsername = fake.LetterN(10)
		expectedPassword = fake.LetterN(10)

		expectedRodeHost = fake.LetterN(10)
	})

	DescribeTable("flag parsing",
		func(flags []string, expectedConfig *ClientConfig) {
			err := flagSet.Parse(flags)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualConfig).To(BeEquivalentTo(expectedConfig))
		},
		Entry("defaults", []string{}, &ClientConfig{
			Rode: &RodeClientConfig{
				Host: "rode:50051",
			},
			JWTAuth:   &JWTAuthConfig{},
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
			JWTAuth:   &JWTAuthConfig{},
			BasicAuth: &BasicAuthConfig{},
		}),
		Entry("jwt auth", []string{
			"--jwt-client-id=" + expectedClientId,
			"--jwt-client-secret=" + expectedClientSecret,
			"--jwt-token-url=" + expectedTokenUrl,
		}, &ClientConfig{
			Rode: &RodeClientConfig{
				Host: "rode:50051",
			},
			JWTAuth: &JWTAuthConfig{
				ClientID:     expectedClientId,
				ClientSecret: expectedClientSecret,
				TokenURL:     expectedTokenUrl,
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
			JWTAuth: &JWTAuthConfig{},
			BasicAuth: &BasicAuthConfig{
				Username: expectedUsername,
				Password: expectedPassword,
			},
		}),
	)
})
