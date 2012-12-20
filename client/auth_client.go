package client

import (
	"errors"
	"fmt"
	gooseerrors "launchpad.net/goose/errors"
	"launchpad.net/goose/identity"
	"log"
	"path"
)

const (
	apiTokens = "/tokens"
)

type authClient struct {
	unauthentictedClient

	creds      *identity.Credentials
	authMethod identity.Authenticator

	//TODO - store service urls by region.
	ServiceURLs map[string]string
	tokenId     string
	tenantId    string
	userId      string
}

func NewClient(creds *identity.Credentials, auth_method identity.AuthMethod, logger *log.Logger) *authClient {
	client_creds := *creds
	client_creds.URL = client_creds.URL + apiTokens
	client := authClient{creds: &client_creds}
	client.unauthentictedClient = unauthentictedClient{
		auth: &client, logger: logger,
	}
	switch auth_method {
	default:
		panic(fmt.Errorf("Invalid identity authorisation method: %d", auth_method))
	case identity.AuthLegacy:
		client.authMethod = &identity.Legacy{}
	case identity.AuthUserPass:
		client.authMethod = &identity.UserPass{}
	}
	return &client
}

func (c *authClient) MakeServiceURL(serviceType string, parts []string) (string, error) {
	if !c.IsAuthenticated() {
		return "", errors.New("cannot get endpoint URL without being authenticated")
	}
	url, ok := c.ServiceURLs[serviceType]
	if !ok {
		return "", errors.New("no endpoints known for service type: " + serviceType)
	}
	url += path.Join(append([]string{"/"}, parts...)...)
	return url, nil
}

func (c *authClient) Token() string {
	return c.tokenId
}

func (c *authClient) UserId() string {
	return c.userId
}

func (c *authClient) TenantId() string {
	return c.tenantId
}

func (c *authClient) Authenticate() (err error) {
	if c.creds == nil || c.IsAuthenticated() {
		return nil
	}
	err = nil
	if c.authMethod == nil {
		return fmt.Errorf("Authentication method has not been specified")
	}
	authDetails, err := c.authMethod.Auth(c.creds)
	if err != nil {
		err = gooseerrors.Newf(err, "authentication failed")
		return
	}

	c.tokenId = authDetails.Token
	c.tenantId = authDetails.TenantId
	c.userId = authDetails.UserId
	c.ServiceURLs = authDetails.ServiceURLs
	return nil
}

func (c *authClient) IsAuthenticated() bool {
	return c.tokenId != ""
}
