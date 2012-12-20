// goose - Go packages to interact with the OpenStack API.
//
//   https://launchpad.net/goose/
//
// Copyright (c) 2012 Canonical Ltd.
//

package client

import (
	"errors"
	"fmt"
	gooseerrors "launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"launchpad.net/goose/identity"
	"log"
	"net/http"
	"path"
)

// API URL parts and request types.
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

// OpenStackClient implements a subset of the OpenStack API, which is
// needed for juju, including authentication, compute and swift services.
type OpenStackClient struct {
	client *goosehttp.Client
	logger *log.Logger

	creds *identity.Credentials
	auth  identity.Authenticator

	// Discovered services and their endpoints.
	// TODO - store service urls by region.
	ServiceURLs map[string]string
	// Current session authentication token.
	Token string
	// Tenant identifier for that user.
	TenantId string
	// User account identifier.
	UserId string
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

// Authenticate establishes an authenticated session with OpenStack
// Identity service. It uses OS_* and NOVA_* environment variables to
// discover the username, password, tenant and region.
func (c *OpenStackClient) Authenticate() (err error) {
	err = nil
	if c.auth == nil {
		return fmt.Errorf("Authentication method has not been specified")
	}
	authDetails, err := c.auth.Auth(c.creds)
	if err != nil {
		err = gooseerrors.Newf(err, "authentication failed")
		return
	}

	c.Token = authDetails.Token
	c.TenantId = authDetails.TenantId
	c.UserId = authDetails.UserId
	c.ServiceURLs = authDetails.ServiceURLs
	return nil
}

// IsAuthenticated returns true if there is an establised session.
func (c *OpenStackClient) IsAuthenticated() bool {
	return c.Token != ""
}

// MakeServiceURL prepares a full URL to a service endpoint, with optional
// URL parts. It uses the first endpoint it can find for the given service type.
func (c *OpenStackClient) MakeServiceURL(serviceType string, parts []string) (string, error) {
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

// SendRequest sends an HTTP request with the given method, service
// type (to get an endpoint to it), URL suffix of the API call and
// extended request data.
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
