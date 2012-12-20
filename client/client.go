package client

import (
	goosehttp "launchpad.net/goose/http"
	"log"
	"net/http"
	"path"
)

const (
	// The HTTP request methods.
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
	HEAD   = "HEAD"
	COPY   = "COPY"
)

// Client implementations send service requests to an OpenStack deployment.
type Client interface {
	SendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData) (err error)
}

// Authenticator defines methods used to perform client authentication.
// In the case of unauthentictedClient instances, these are essentially NOOPs.
type Authenticator interface {
	Authenticate() error
	Token() string
	UserId() string
	TenantId() string
	// MakeServiceURL prepares a full URL to a service endpoint, with optional
	// URL parts. It uses the first endpoint it can find for the given service type.
	// Authentication needs to be completed prior to calling this method since it
	// uses data returned from the authentication call to form the URL.
	MakeServiceURL(serviceType string, parts []string) (string, error)
}

type unauthentictedClient struct {
	auth    Authenticator
	client  *goosehttp.Client
	logger  *log.Logger
	baseURL string
}

func NewPublicClient(baseURL string, logger *log.Logger) Client {
	client := unauthentictedClient{baseURL: baseURL, logger: logger}
	client.auth = &client
	return &client
}

func (c *unauthentictedClient) SendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData) (err error) {
	if err = c.auth.Authenticate(); err != nil {
		return
	}

	url, err := c.auth.MakeServiceURL(svcType, []string{apiCall})
	if err != nil {
		return
	}
	if c.client == nil {
		c.client = goosehttp.New(http.Client{CheckRedirect: nil}, c.logger, c.auth.Token())
	}
	if requestData.ReqValue != nil || requestData.RespValue != nil {
		err = c.client.JsonRequest(method, url, requestData)
	} else {
		err = c.client.BinaryRequest(method, url, requestData)
	}
	return
}

func (c *unauthentictedClient) MakeServiceURL(serviceType string, parts []string) (string, error) {
	url := c.baseURL + path.Join(parts...)
	return url, nil
}

func (c *unauthentictedClient) Authenticate() error {
	return nil
}

func (c *unauthentictedClient) Token() string {
	return ""
}

func (c *unauthentictedClient) UserId() string {
	return ""
}

func (c *unauthentictedClient) TenantId() string {
	return ""
}
