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
	"strings"
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

// Client implementations sends service requests to an OpenStack deployment.
type Client interface {
	SendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData) (err error)
	// MakeServiceURL prepares a full URL to a service endpoint, with optional
	// URL parts. It uses the first endpoint it can find for the given service type.
	MakeServiceURL(serviceType string, parts []string) (string, error)
}

// AuthenticatingClient sends service requests to an OpenStack deployment after first validating
// a user's credentials.
type AuthenticatingClient interface {
	Client
	Authenticate() error
	IsAuthenticated() bool
	Token() string
	UserId() string
	TenantId() string
}

// This client sends requests without authenticating.
type client struct {
	httpClient *goosehttp.Client
	logger     *log.Logger
	baseURL    string
}

var _ Client = (*client)(nil)

// This client authenticates before sending requests.
type authenticatingClient struct {
	client

	creds    *identity.Credentials
	authMode identity.Authenticator

	auth              AuthenticatingClient
	regionServiceURLs map[string]identity.ServiceURLs // Service type to endpoint URLs for each region
	tokenId           string
	tenantId          string
	userId            string
}

var _ AuthenticatingClient = (*authenticatingClient)(nil)

func NewPublicClient(baseURL string, logger *log.Logger) Client {
	client := client{baseURL: baseURL, logger: logger}
	return &client
}

func NewClient(creds *identity.Credentials, auth_method identity.AuthMode, logger *log.Logger) AuthenticatingClient {
	client_creds := *creds
	client_creds.URL = client_creds.URL + apiTokens
	client := authenticatingClient{
		creds:             &client_creds,
		client:            client{logger: logger},
		regionServiceURLs: make(map[string]identity.ServiceURLs),
	}
	client.auth = &client
	switch auth_method {
	default:
		panic(fmt.Errorf("Invalid identity authorisation method: %d", auth_method))
	case identity.AuthLegacy:
		client.authMode = &identity.Legacy{}
	case identity.AuthUserPass:
		client.authMode = &identity.UserPass{}
	}
	return &client
}

func (c *client) sendRequest(method, url, token string, requestData *goosehttp.RequestData) (err error) {
	if c.httpClient == nil {
		c.httpClient = goosehttp.New(http.Client{CheckRedirect: nil}, c.logger, token)
	}
	if requestData.ReqValue != nil || requestData.RespValue != nil {
		err = c.httpClient.JsonRequest(method, url, requestData)
	} else {
		err = c.httpClient.BinaryRequest(method, url, requestData)
	}
	return
}

func (c *client) SendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData) error {
	url, _ := c.MakeServiceURL(svcType, []string{apiCall})
	return c.sendRequest(method, url, "", requestData)
}

func (c *client) MakeServiceURL(serviceType string, parts []string) (string, error) {
	urlParts := parts
	if !strings.HasSuffix(c.baseURL, "/") {
		urlParts = append([]string{"/"}, parts...)
	}
	return c.baseURL + path.Join(urlParts...), nil
}

func (c *authenticatingClient) SendRequest(method, svcType, apiCall string, requestData *goosehttp.RequestData) (err error) {
	if err = c.Authenticate(); err != nil {
		return
	}

	url, err := c.MakeServiceURL(svcType, []string{apiCall})
	if err != nil {
		return
	}
	return c.sendRequest(method, url, c.tokenId, requestData)
}

func (c *authenticatingClient) MakeServiceURL(serviceType string, parts []string) (string, error) {
	if !c.IsAuthenticated() {
		return "", errors.New("cannot get endpoint URL without being authenticated")
	}
	serviceURLs, err := c.serviceURLs()
	if err != nil {
		return "", err
	}
	url, ok := serviceURLs[serviceType]
	if !ok {
		return "", errors.New("no endpoints known for service type: " + serviceType)
	}
	url += path.Join(append([]string{"/"}, parts...)...)
	return url, nil
}

// Return the relevant service endpoint URLs for this client's region.
// The region comes from the client credentials.
func (c *authenticatingClient) serviceURLs() (identity.ServiceURLs, error) {
	var serviceURLs identity.ServiceURLs = nil
	for region, urls := range c.regionServiceURLs {
		if regionMatches(c.creds.Region, region) {
			if serviceURLs == nil {
				serviceURLs = make(identity.ServiceURLs)
			}
			for serviceType, endpointURL := range urls {
				serviceURLs[serviceType] = endpointURL
			}
		}
	}
	if serviceURLs == nil {
		var knownRegions []string
		for r := range c.regionServiceURLs {
			knownRegions = append(knownRegions, r)
		}
		return nil, fmt.Errorf("invalid region '%s', valid regions are %s",
			c.creds.Region, strings.Join(knownRegions, ", "))
	}
	return serviceURLs, nil
}

func regionMatches(userRegion, endpointRegion string) bool {
	// The user specified region (from the credentials config) matches
	// the endpoint region if the user region equals or ends with the endpoint region.
	// eg  user region "az-1.region-a.geo-1" matches endpoint region "region-a.geo-1"
	return strings.HasSuffix(userRegion, endpointRegion)
}

func (c *authenticatingClient) Token() string {
	return c.tokenId
}

func (c *authenticatingClient) UserId() string {
	return c.userId
}

func (c *authenticatingClient) TenantId() string {
	return c.tenantId
}

func (c *authenticatingClient) IsAuthenticated() bool {
	return c.tokenId != ""
}

func (c *authenticatingClient) Authenticate() (err error) {
	if c.creds == nil || c.IsAuthenticated() {
		return nil
	}
	err = nil
	if c.authMode == nil {
		return fmt.Errorf("Authentication method has not been specified")
	}
	authDetails, err := c.authMode.Auth(c.creds)
	if err != nil {
		err = gooseerrors.Newf(err, "authentication failed")
		return
	}

	c.tokenId = authDetails.Token
	c.tenantId = authDetails.TenantId
	c.userId = authDetails.UserId
	c.regionServiceURLs = authDetails.RegionServiceURLs
	return nil
}
