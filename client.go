package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// ErrorContextf prefixes any error stored in err with text formatted
// according to the format specifier. If err does not contain an error,
// ErrorContextf does nothing.
func ErrorContextf(err *error, format string, args ...interface{}) {
	if *err != nil {
		*err = errors.New(fmt.Sprintf(format, args...) + ": " + (*err).Error())
	}
}

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

func GetEnvVars() (username, password, tenant, region, authUrl string) {
	username = getConfig("OS_USERNAME", "NOVA_USERNAME")
	password = getConfig("OS_PASSWORD", "NOVA_PASSWORD")
	tenant = getConfig("OS_TENANT_NAME", "NOVA_PROJECT_ID")
	region = getConfig("OS_REGION_NAME", "NOVA_REGION")
	authUrl = getConfig("OS_AUTH_URL")
	return
}

const (
	OS_API_TOKENS          = "/tokens"
	OS_API_FLAVORS         = "/flavors"
	OS_API_FLAVORS_DETAIL  = "/flavors/detail"
	OS_API_SERVERS         = "/servers"
	OS_API_SERVERS_DETAIL  = "/servers/detail"
	OS_API_SECURITY_GROUPS = "/os-security-groups"

	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
)

type endpoint struct {
	AdminURL    string
	Region      string
	InternalURL string
	Id          string
	PublicURL   string
}

type service struct {
	Name      string
	Type      string
	Endpoints []endpoint
}

type token struct {
	Expires string
	Id      string
	Tenant  struct {
		Enabled     bool
		Description string
		Name        string
		Id          string
	}
}

type user struct {
	Username string
	Roles    []struct {
		Name string
	}
	Id   string
	Name string
}

type metadata struct {
	IsAdmin bool
	Roles   []string
}

type OpenStackClient struct {
	// URL to the OpenStack Identity service (Keystone)
	IdentityEndpoint string
	// Which region to use
	Region string

	client *http.Client

	Services map[string]service
	Token    token
	User     user
	Metadata metadata
}

func (c *OpenStackClient) Authenticate(username, password, tenant string) (err error) {
	err = nil
	var req struct {
		Auth struct {
			Credentials struct {
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"passwordCredentials"`
			Tenant string `json:"tenantName"`
		} `json:"auth"`
	}
	req.Auth.Credentials.Username = username
	req.Auth.Credentials.Password = password
	req.Auth.Tenant = tenant

	var resp struct {
		Access struct {
			Token          token
			ServiceCatalog []service
			User           user
			Metadata       metadata
		}
	}

	err = c.request(POST, c.IdentityEndpoint+OS_API_TOKENS, req, &resp, http.StatusOK)
	if err != nil {
		ErrorContextf(&err, "authentication failed")
		return
	}

	c.Token = resp.Access.Token
	c.User = resp.Access.User
	c.Metadata = resp.Access.Metadata
	if c.Services == nil {
		c.Services = make(map[string]service)
	}
	for _, s := range resp.Access.ServiceCatalog {
		// Filter endpoints outside our region
		for i, e := range s.Endpoints {
			if e.Region != c.Region {
				s.Endpoints = append(s.Endpoints[:i], s.Endpoints[i+1:]...)
			}
		}
		c.Services[s.Type] = s
	}
	return nil
}

func (c *OpenStackClient) IsAuthenticated() bool {
	return c.Token.Id != ""
}

type Link struct {
	Href string
	Rel  string
	Type string
}

type Entity struct {
	Id    string
	Links []Link
	Name  string
}

func (c *OpenStackClient) ListFlavors() (flavors []Entity, err error) {

	var resp struct {
		Flavors []Entity
	}
	err = c.authRequest(GET, "compute", OS_API_FLAVORS, nil, &resp, http.StatusOK)
	if err != nil {
		ErrorContextf(&err, "failed to get list of flavors")
		return
	}

	return resp.Flavors, nil
}

func (c *OpenStackClient) ListFlavorsDetail() (flavors []Entity, err error) {

	var resp struct {
		Flavors []Entity
	}
	err = c.authRequest(GET, "compute", OS_API_FLAVORS_DETAIL, nil, &resp, http.StatusOK)
	if err != nil {
		ErrorContextf(&err, "failed to get list of flavors details")
		return
	}

	return resp.Flavors, nil
}

func (c *OpenStackClient) ListServers() (servers []Entity, err error) {

	var resp struct {
		Servers []Entity
	}
	err = c.authRequest(GET, "compute", OS_API_SERVERS, nil, &resp, http.StatusOK)
	if err != nil {
		ErrorContextf(&err, "failed to get list of servers")
		return
	}

	return resp.Servers, nil
}

type ServerDetail struct {
	AddressIPv4 string
	AddressIPv6 string
	Created     string
	Flavor      Entity
	HostId      string
	Id          string
	Image       Entity
	Links       []Link
	Name        string
	Progress    int
	Status      string
	TenantId    string `json:"tenant_id"`
	Updated     string
	UserId      string `json:"user_id"`
}

func (c *OpenStackClient) ListServersDetail() (servers []ServerDetail, err error) {

	var resp struct {
		Servers []ServerDetail
	}
	err = c.authRequest(GET, "compute", OS_API_SERVERS_DETAIL, nil, &resp, http.StatusOK)
	if err != nil {
		ErrorContextf(&err, "failed to get list of servers details")
		return
	}

	return resp.Servers, nil
}

func (c *OpenStackClient) GetServer(serverId string) (ServerDetail, error) {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	err := c.authRequest(GET, "compute", url, nil, &resp, http.StatusOK)
	if err != nil {
		ErrorContextf(&err, "failed to get details for serverId=%s", serverId)
		return ServerDetail{}, err
	}

	return resp.Server, nil
}

func (c *OpenStackClient) DeleteServer(serverId string) error {

	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", OS_API_SERVERS, serverId)
	err := c.authRequest(DELETE, "compute", url, nil, &resp, http.StatusNoContent)
	if err != nil {
		ErrorContextf(&err, "failed to delete server with serverId=%s", serverId)
		return err
	}

	return nil
}

//func (c *OpenStackClient) RunServer(imageId, flavorId, name string, securit

type SecurityGroup struct {
	Rules []struct {
		FromPort      int               `json:"from_port"`
		IPProtocol    string            `json:"ip_protocol"`
		ToPort        int               `json:"to_port"`
		ParentGroupId int               `json:"parent_group_id"`
		IPRange       map[string]string `json:"ip_range"`
		Id            int
	}
	TenantId    string `json:"tenant_id"`
	Id          int
	Name        string
	Description string
}

func (c *OpenStackClient) ListSecurityGroups() (groups []SecurityGroup, err error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	err = c.authRequest(GET, "compute", OS_API_SECURITY_GROUPS, nil, &resp, http.StatusOK)
	if err != nil {
		ErrorContextf(&err, "failed to list security groups")
		return nil, err
	}

	return resp.Groups, nil
}

////////////////////////////////////////////////////////////////////////
// Private helpers

// request sends an HTTP request with the given method to the given URL,
// containing an optional body (serialized to JSON), and returning either
// an error or the (deserialized) response body
func (c *OpenStackClient) request(method, url string, body interface{}, resp interface{}, expectedStatus int) (err error) {
	err = nil
	if c.client == nil {
		c.client = &http.Client{CheckRedirect: nil}
	}

	var req *http.Request
	if body != nil {
		var jsonBody []byte
		jsonBody, err = json.Marshal(body)
		if err != nil {
			ErrorContextf(&err, "failed marshalling the request body")
			return
		}

		reqBody := strings.NewReader(string(jsonBody))
		req, err = http.NewRequest(method, url, reqBody)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		ErrorContextf(&err, "failed creating the request")
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	if c.IsAuthenticated() {
		req.Header.Add("X-Auth-Token", c.Token.Id)
	}

	rawResp, err := c.client.Do(req)
	if err != nil {
		ErrorContextf(&err, "failed executing the request")
		return
	}
	if rawResp.StatusCode != expectedStatus {
		defer rawResp.Body.Close()
		respBody, _ := ioutil.ReadAll(rawResp.Body)
		err = errors.New(
			fmt.Sprintf(
				"request (%s) returned unexpected status: %s - %s",
				url,
				rawResp.Status,
				respBody))
		return
	}

	var respBody []byte
	defer rawResp.Body.Close()
	respBody, err = ioutil.ReadAll(rawResp.Body)
	if err != nil {
		ErrorContextf(&err, "failed reading the response body")
		return
	}

	if len(respBody) > 0 {
		err = json.Unmarshal(respBody, &resp)
		if err != nil {
			ErrorContextf(&err, "failed unmarshaling the response body: %s", respBody)
		}
	} else {
		resp = nil
	}

	return
}

// makeUrl prepares a full URL to a service endpoint, with optional
// URL parts, appended to it and optional query string params. It
// uses the first endpoint it can find for the given service type
func (c *OpenStackClient) makeUrl(serviceType string, parts []string, params url.Values) (string, error) {
	s, ok := c.Services[serviceType]
	if !ok || len(s.Endpoints) == 0 {
		return "", errors.New("no endpoints known for service type: " + serviceType)
	}
	url := s.Endpoints[0].PublicURL
	for _, part := range parts {
		url += part
	}
	if params != nil {
		url += "?" + params.Encode()
	}
	return url, nil
}

func (c *OpenStackClient) authRequest(method, svcType, apiCall string, params url.Values, resp interface{}, expectedStatus int) (err error) {

	if !c.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	url, err := c.makeUrl(svcType, []string{apiCall}, params)
	if err != nil {
		ErrorContextf(&err, "cannot find a '%s' node endpoint", svcType)
		return
	}

	err = c.request(method, url, nil, &resp, expectedStatus)
	if err != nil {
		ErrorContextf(&err, "request failed")
	}
	return
}

func main() {
	username, password, tenant, region, auth_url := GetEnvVars()
	for _, p := range []string{username, password, tenant, region, auth_url} {
		if p == "" {
			panic("required environment var(s) missing!")
		}
	}
	client := &OpenStackClient{IdentityEndpoint: auth_url, Region: region}
	err := client.Authenticate(username, password, tenant)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("authenticated successfully: token=%s\n", client.Token.Id)
	servers, err := client.ListServers()
	if err != nil {
		panic(err.Error())
	}
	server, err := client.GetServer(servers[0].Id)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\n\n%+v\n", server)
	groups, err := client.ListSecurityGroups()
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\n\n%+v\n", groups)
}
