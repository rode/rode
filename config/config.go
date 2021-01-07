package config

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/coreos/go-oidc"
)

type Config struct {
	Auth    *AuthConfig
	Grafeas *GrafeasConfig
	Port    int
	Debug   bool
}

type GrafeasConfig struct {
	Host string
}

type AuthConfig struct {
	Basic *BasicAuthConfig
	JWT   *JWTAuthConfig
}

type BasicAuthConfig struct {
	Username string
	Password string
}

type JWTAuthConfig struct {
	Issuer           string
	RequiredAudience string
	Verifier         *oidc.IDTokenVerifier
}

func Build(name string, args []string) (*Config, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)

	c := &Config{
		Auth: &AuthConfig{
			Basic: &BasicAuthConfig{},
			JWT:   &JWTAuthConfig{},
		},
		Grafeas: &GrafeasConfig{},
	}

	flags.StringVar(&c.Auth.Basic.Username, "basic-auth-username", "", "when set, basic auth will be enabled for all endpoints, using the provided username. --basic-auth-password must also be set")
	flags.StringVar(&c.Auth.Basic.Password, "basic-auth-password", "", "when set, basic auth will be enabled for all endpoints, using the provided password. --basic-auth-username must also be set")
	flags.StringVar(&c.Auth.JWT.Issuer, "jwt-issuer", "", "when set, jwt based auth will be enabled for all endpoints. the provided issuer will be used to fetch the discovery document in order to validate received jwts")
	flags.StringVar(&c.Auth.JWT.RequiredAudience, "jwt-required-audience", "", "when set, if jwt based auth is enabled, this audience must be specified within the `aud` claim of any received jwts")

	flags.IntVar(&c.Port, "port", 50051, "the port that the rode API server should listen on")
	flags.BoolVar(&c.Debug, "debug", false, "when set, debug mode will be enabled")
	flags.StringVar(&c.Grafeas.Host, "grafeas-host", "localhost:8080", "the host to use to connect to grafeas")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	if (c.Auth.Basic.Username != "" && c.Auth.Basic.Password == "") || (c.Auth.Basic.Username == "" && c.Auth.Basic.Password != "") {
		return nil, errors.New("when using basic auth, both --basic-auth-username and --basic-auth-password must be set")
	}

	if c.Auth.JWT.Issuer != "" {
		provider, err := oidc.NewProvider(context.Background(), c.Auth.JWT.Issuer)
		if err != nil {
			return nil, fmt.Errorf("error initializing oidc provider: %v", err)
		}

		oidcConfig := &oidc.Config{}
		if c.Auth.JWT.RequiredAudience != "" {
			oidcConfig.ClientID = c.Auth.JWT.RequiredAudience
		} else {
			oidcConfig.SkipClientIDCheck = true
		}

		c.Auth.JWT.Verifier = provider.Verifier(oidcConfig)
	} else if c.Auth.JWT.RequiredAudience != "" {
		return nil, errors.New("the --jwt-required-audience flag cannot be specified without --jwt-issuer")
	}

	return c, nil
}
