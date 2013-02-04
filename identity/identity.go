// goose/identity - Go package to interact with OpenStack Identity (Keystone) API.

package identity

import (
	"fmt"
	"os"
	"reflect"
)

// AuthMode defines the authentication method to use (see Auth*
// constants below).
type AuthMode int

const (
	AuthLegacy   = AuthMode(iota) // Legacy authentication
	AuthUserPass                  // Username + password authentication
)

func (a AuthMode) String() string {
	switch a {
	case AuthLegacy:
		return "Legacy Authentication"
	case AuthUserPass:
		return "Username/password Authentication"
	}
	panic(fmt.Errorf("Unknown athentication type: %d", a))
}

// AuthDetails defines all the necessary information, needed for an
// authenticated session with OpenStack.
type AuthDetails struct {
	Token       string
	TenantId    string
	UserId      string
	ServiceURLs map[string]string // Service type to endpoint URLs
}

// Credentials defines necessary parameters for authentication.
type Credentials struct {
	URL        string // The URL to authenticate against
	User       string // The username to authenticate as
	Secrets    string // The secrets to pass
	Region     string // Region to send requests to
	TenantName string // The tenant information for this connection
}

// Authenticator is implemented by each authentication method.
type Authenticator interface {
	Auth(creds *Credentials) (*AuthDetails, error)
}

// getConfig returns the value of the first available environment
// variable, among the given ones.
func getConfig(envVars ...string) (value string) {
	value = ""
	for _, v := range envVars {
		value = os.Getenv(v)
		if value != "" {
			break
		}
	}
	return
}

// CredentialsFromEnv creates and initializes the credentials from the
// environment variables.
func CredentialsFromEnv() *Credentials {
	return &Credentials{
		URL:        getConfig("OS_AUTH_URL"),
		User:       getConfig("OS_USERNAME", "NOVA_USERNAME"),
		Secrets:    getConfig("OS_PASSWORD", "NOVA_PASSWORD"),
		Region:     getConfig("OS_REGION_NAME", "NOVA_REGION"),
		TenantName: getConfig("OS_TENANT_NAME", "NOVA_PROJECT_ID"),
	}
}

// CompleteCredentialsFromEnv gets and verifies all the required
// authentication parameters have values in the environment.
func CompleteCredentialsFromEnv() (cred *Credentials, err error) {
	cred = CredentialsFromEnv()
	v := reflect.ValueOf(cred).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.String() == "" {
			err = fmt.Errorf("required environment variable not set for credentials attribute: %s", t.Field(i).Name)
		}
	}
	return
}
