package config

import (
	"errors"
	"flag"
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
	BasicAuthUsername string
	BasicAuthPassword string
}

func Build(name string, args []string) (*Config, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)

	c := &Config{
		Auth:    &AuthConfig{},
		Grafeas: &GrafeasConfig{},
	}

	flags.StringVar(&c.Auth.BasicAuthUsername, "basic-auth-username", "", "when set, basic auth will be enabled for all endpoints using the provided username. --basic-auth-password must also be set")
	flags.StringVar(&c.Auth.BasicAuthPassword, "basic-auth-password", "", "when set, basic auth will be enabled for all endpoints using the provided password. --basic-auth-username must also be set")
	flags.IntVar(&c.Port, "port", 50051, "the port that the rode API server should listen on")
	flags.BoolVar(&c.Debug, "debug", false, "when set, debug mode will be enabled")
	flags.StringVar(&c.Grafeas.Host, "grafeas-host", "localhost:8080", "the host to use to connect to grafeas")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	if (c.Auth.BasicAuthUsername != "" && c.Auth.BasicAuthPassword == "") || (c.Auth.BasicAuthUsername == "" && c.Auth.BasicAuthPassword != "") {
		return nil, errors.New("when using basic auth, both --basic-auth-username and --basic-auth-password must be set")
	}

	return c, nil
}
