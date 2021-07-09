package common

import "flag"

func SetupRodeClientFlags(flags *flag.FlagSet) *ClientConfig {
	conf := &ClientConfig{
		Rode:      &RodeClientConfig{},
		JWTAuth:   &JWTAuthConfig{},
		BasicAuth: &BasicAuthConfig{},
	}

	flags.StringVar(&conf.Rode.Host, "rode-host", "rode:50051", "the host to use to connect to rode")
	flags.BoolVar(&conf.Rode.DisableTransportSecurity, "rode-insecure-disable-transport-security", false, "when set, the connection to rode will not use transport security")

	flags.StringVar(&conf.JWTAuth.ClientID, "jwt-client-id", "", "the client ID to use when requesting a JWT via the client_credentials grant")
	flags.StringVar(&conf.JWTAuth.ClientSecret, "jwt-client-secret", "", "the client secret to use when requesting a JWT via the client_credentials grant")
	flags.StringVar(&conf.JWTAuth.TokenURL, "jwt-token-url", "", "the URL to use to retrieve an access token via the client_credentials grant")

	flags.StringVar(&conf.BasicAuth.Username, "basic-auth-username", "", "the username to use for basic authentication")
	flags.StringVar(&conf.BasicAuth.Password, "basic-auth-password", "", "the password to use for basic authentication")

	return conf
}
