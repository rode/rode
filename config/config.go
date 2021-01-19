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
	Opa     *OpaConfig
	Port    int
	Debug   bool
}

type GrafeasConfig struct {
	Host string
}

type OpaConfig struct {
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

	conf := &Config{
		Auth: &AuthConfig{
			Basic: &BasicAuthConfig{},
			JWT:   &JWTAuthConfig{},
		},
		Grafeas: &GrafeasConfig{},
		Opa:     &OpaConfig{},
	}

	flags.StringVar(&conf.Auth.Basic.Username, "basic-auth-username", "", "when set, basic auth will be enabled for all endpoints, using the provided username. --basic-auth-password must also be set")
	flags.StringVar(&conf.Auth.Basic.Password, "basic-auth-password", "", "when set, basic auth will be enabled for all endpoints, using the provided password. --basic-auth-username must also be set")
	flags.StringVar(&conf.Auth.JWT.Issuer, "jwt-issuer", "", "when set, jwt based auth will be enabled for all endpoints. the provided issuer will be used to fetch the discovery document in order to validate received jwts")
	flags.StringVar(&conf.Auth.JWT.RequiredAudience, "jwt-required-audience", "", "when set, if jwt based auth is enabled, this audience must be specified within the `aud` claim of any received jwts")

	flags.IntVar(&conf.Port, "port", 50051, "the port that the rode API server should listen on")
	flags.BoolVar(&conf.Debug, "debug", false, "when set, debug mode will be enabled")
	flags.StringVar(&conf.Grafeas.Host, "grafeas-host", "localhost:8080", "the host to use to connect to grafeas")
	flags.StringVar(&conf.Opa.Host, "opa-host", "localhost:8181", "the host to use to connect to Open Policy Agent")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	if (conf.Auth.Basic.Username != "" && conf.Auth.Basic.Password == "") || (conf.Auth.Basic.Username == "" && conf.Auth.Basic.Password != "") {
		return nil, errors.New("when using basic auth, both --basic-auth-username and --basic-auth-password must be set")
	}

	if conf.Auth.JWT.Issuer != "" {
		provider, err := oidc.NewProvider(context.Background(), conf.Auth.JWT.Issuer)
		if err != nil {
			return nil, fmt.Errorf("error initializing oidc provider: %v", err)
		}

		oidcConfig := &oidc.Config{}
		if conf.Auth.JWT.RequiredAudience != "" {
			oidcConfig.ClientID = conf.Auth.JWT.RequiredAudience
		} else {
			oidcConfig.SkipClientIDCheck = true
		}

		conf.Auth.JWT.Verifier = provider.Verifier(oidcConfig)
	} else if conf.Auth.JWT.RequiredAudience != "" {
		return nil, errors.New("the --jwt-required-audience flag cannot be specified without --jwt-issuer")
	}

	return conf, nil
}
