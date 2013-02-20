package client

import (
	"launchpad.net/goose/identity"
	"time"
)

func SetAuthenticationTimeout(timeout time.Duration) func() {
	origTimeout := authenticationTimeout
	authenticationTimeout = timeout
	return func() {
		authenticationTimeout = origTimeout
	}
}

func SetAuthenticator(client AuthenticatingClient, auth identity.Authenticator) {
	client.(*authenticatingClient).authMode = auth
}
