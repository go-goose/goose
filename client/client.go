package client

import (
	"errors"
	"fmt"
	gooseerrors "launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"launchpad.net/goose/identity"
	"log"
	"net/http"
)

const (
	apiTokens = "/tokens"

	// The HTTP request methods.
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
	HEAD   = "HEAD"
	COPY   = "COPY"
)

type Client interface {
	MakeServiceURL(serviceType string, parts []string) (string, error)
	SendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData) (err error)
}

type OpenStackClient struct {
	client *goosehttp.Client
	logger *log.Logger

	creds *identity.Credentials
	auth  identity.Authenticator

	//TODO - store service urls by region.
	ServiceURLs map[string]string
	Token       string
	TenantId    string
	UserId      string
}

func NewClient(creds *identity.Credentials, auth_method identity.AuthMethod, logger *log.Logger) *OpenStackClient {
	client_creds := *creds
	client_creds.URL = client_creds.URL + apiTokens
	client := OpenStackClient{creds: &client_creds, logger: logger}
	switch auth_method {
	default:
		panic(fmt.Errorf("Invalid identity authorisation method: %d", auth_method))
	case identity.AuthLegacy:
		client.auth = &identity.Legacy{}
	case identity.AuthUserPass:
		client.auth = &identity.UserPass{}
	}
	return &client
}

func (c *OpenStackClient) Authenticate() (err error) {
	err = nil
	if c.auth == nil {
		return fmt.Errorf("Authentication method has not been specified")
	}
	authDetails, err := c.auth.Auth(c.creds)
	if err != nil {
		err = gooseerrors.Newf(err, nil, "authentication failed")
		return
	}

	c.Token = authDetails.Token
	c.TenantId = authDetails.TenantId
	c.UserId = authDetails.UserId
	c.ServiceURLs = authDetails.ServiceURLs
	return nil
}

func (c *OpenStackClient) IsAuthenticated() bool {
	return c.Token != ""
}

// MakeServiceURL prepares a full URL to a service endpoint, with optional
// URL parts. It uses the first endpoint it can find for the given service type.
func (c *OpenStackClient) MakeServiceURL(serviceType string, parts []string) (string, error) {
	url, ok := c.ServiceURLs[serviceType]
	if !ok {
		return "", errors.New("no endpoints known for service type: " + serviceType)
	}
	for _, part := range parts {
		url += part
	}
	return url, nil
}

func (c *OpenStackClient) SendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData) (err error) {
	if c.creds != nil && !c.IsAuthenticated() {
		err = c.Authenticate()
		if err != nil {
			return
		}
	}

	url, err := c.MakeServiceURL(svcType, []string{apiCall})
	if err != nil {
		return
	}

	if c.client == nil {
		c.client = goosehttp.New(http.Client{CheckRedirect: nil}, c.logger, c.Token)
	}
	if requestData.ReqValue != nil || requestData.RespValue != nil {
		err = c.client.JsonRequest(method, url, requestData)
	} else {
		err = c.client.BinaryRequest(method, url, requestData)
	}
	return
}
