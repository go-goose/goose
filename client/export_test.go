package client

import (
	"launchpad.net/goose/identity"
	"time"
)

func SetAuthenticationTimeout(timeout time.Duration) {
	authenticationTimeout = timeout
}

func SetAuthenticator(client AuthenticatingClient, auth identity.Authenticator) {
	client.(*authenticatingClient).authMode = auth
}
