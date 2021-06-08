package client

import (
	"time"

	goosehttp "gopkg.in/goose.v3/http"
	"gopkg.in/goose.v3/identity"
	"gopkg.in/goose.v3/logging"
)

type AuthCleanup func()

func SetAuthenticationTimeout(timeout time.Duration) AuthCleanup {
	origTimeout := authenticationTimeout
	authenticationTimeout = timeout
	return func() {
		authenticationTimeout = origTimeout
	}
}

func SetAuthenticator(client AuthenticatingClient, auth identity.Authenticator) {
	client.(*authenticatingClient).authMode = auth
}

func NewClientForTest(
	creds *identity.Credentials,
	auth_method identity.AuthMode,
	httpClient goosehttp.HttpClient,
	logger logging.CompatLogger,
) AuthenticatingClient {
	return newClient(creds, auth_method, httpClient, logger)
}
