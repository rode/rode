package util

import (
	"os"
	"strings"

	"github.com/onsi/ginkgo/extensions/table"
	"github.com/rode/rode/common"
	"github.com/rode/rode/proto/v1alpha1"
	"github.com/scylladb/go-set/strset"
)

type RodeClientSet struct {
	Anonymous           v1alpha1.RodeClient
	Collector           v1alpha1.RodeClient
	Enforcer            v1alpha1.RodeClient
	PolicyDeveloper     v1alpha1.RodeClient
	PolicyAdministrator v1alpha1.RodeClient
	Administrator       v1alpha1.RodeClient
	v1alpha1.RodeClient // embed a privileged client so that callers don't have specify a role
}

var (
	tokenUrl  = "https://keycloak.test/auth/realms/rode-demo/protocol/openid-connect/token"
	rodeHost  = "localhost:50051"
	rodeRoles = strset.New(
		"Anonymous",
		"Collector",
		"Enforcer",
		"PolicyDeveloper",
		"PolicyAdministrator",
		"Administrator",
	)
)

func NewRodeClientSet() (*RodeClientSet, error) {
	clientSet := &RodeClientSet{}

	var err error
	clientSet.Anonymous, err = common.NewRodeClient(&common.ClientConfig{
		Rode: &common.RodeClientConfig{
			Host:                     rodeHost,
			DisableTransportSecurity: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if clientSet.Collector, err = newRodeClient("Collector"); err != nil {
		return nil, err
	}
	if clientSet.Enforcer, err = newRodeClient("Enforcer"); err != nil {
		return nil, err
	}
	if clientSet.PolicyDeveloper, err = newRodeClient("Policy Developer"); err != nil {
		return nil, err
	}
	if clientSet.PolicyAdministrator, err = newRodeClient("Policy Administrator"); err != nil {
		return nil, err
	}
	administrator, err := newRodeClient("Administrator")
	if err != nil {
		return nil, err
	}
	clientSet.Administrator = administrator
	clientSet.RodeClient = administrator

	return clientSet, nil
}

func (rcs *RodeClientSet) WithRole(roleName string) v1alpha1.RodeClient {
	switch roleName {
	case "Collector":
		return rcs.Collector
	case "Enforcer":
		return rcs.Enforcer
	case "PolicyDeveloper":
		return rcs.PolicyDeveloper
	case "PolicyAdministrator":
		return rcs.PolicyAdministrator
	case "Administrator":
		return rcs.Administrator
	default:
		return rcs.Anonymous
	}
}

type AuthzTestEntry struct {
	Role      string
	Permitted bool
}

func NewAuthzTableTest(roles []string) []table.TableEntry {
	permittedRoles := strset.New(roles...)
	forbiddenRoles := strset.Difference(rodeRoles, permittedRoles)

	var entries []table.TableEntry

	permittedRoles.Each(func(role string) bool {
		entry := &AuthzTestEntry{
			Permitted: true,
			Role:      role,
		}

		entries = append(entries, table.Entry(role, entry))
		return true
	})

	forbiddenRoles.Each(func(role string) bool {
		entry := &AuthzTestEntry{
			Permitted: false,
			Role:      role,
		}
		entries = append(entries, table.Entry(role, entry))
		return true
	})

	return entries
}

func newRodeClient(role string) (v1alpha1.RodeClient, error) {
	fallbackCredentials := strings.Replace(strings.ToLower(role), " ", "-", -1)
	envBase := strings.ToUpper(strings.Replace(role, " ", "_", -1))

	clientId := getEnvOrDefault(envBase+"_CLIENT_ID", fallbackCredentials)
	clientSecret := getEnvOrDefault(envBase+"_CLIENT_SECRET", fallbackCredentials)

	config := &common.ClientConfig{
		Rode: &common.RodeClientConfig{
			Host:                     rodeHost,
			DisableTransportSecurity: true,
		},
		OIDCAuth: &common.OIDCAuthConfig{
			ClientID:              clientId,
			ClientSecret:          clientSecret,
			TokenURL:              tokenUrl,
			TlsInsecureSkipVerify: true,
		},
	}

	return common.NewRodeClient(config)
}

func getEnvOrDefault(envName, fallback string) string {
	val, ok := os.LookupEnv(envName)
	if ok {
		return val
	}

	return fallback
}
