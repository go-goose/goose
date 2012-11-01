package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"errors"
)
// ErrorContextf prefixes any error stored in err with text formatted
// according to the format specifier. If err does not contain an error,
// ErrorContextf does nothing.
func ErrorContextf(err *error, format string, args ...interface{}) error {
	if *err != nil {
		*err = errors.New(fmt.Sprintf(format, args...) + ": " + (*err).Error())
	}
	return *err
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

func getEnvVars() (username, password, tenant, authUrl string) {
	username = getConfig("OS_USERNAME", "NOVA_USERNAME")
	password = getConfig("OS_PASSWORD", "NOVA_PASSWORD")
	tenant = getConfig("OS_TENANT_NAME", "NOVA_PROJECT_ID")
	authUrl = getConfig("OS_AUTH_URL")
	return
}

const (
	OS_API_TOKENS = "tokens"
)

type Endpoint struct {
	AdminURL string
	Region string
	InternalURL string
	Id string
	PublicURL string
}

type Service struct {
	Name string
	Type string
	Endpoints []Endpoint
}

type Token struct {
	Expires string
	Id string
	Tenant struct {
		Enabled bool
		Description string
		Name string
		Id string
	}
}

type User struct {
	Username string
	Roles []struct {
		Name string
	}
	Id string
	Name string
}

type Metadata struct {
	IsAdmin bool
	Roles []string
}

type OpenStackClient struct {
	// URL to the OpenStack Identity service (Keystone)
	IdentityEndpoint string

	client *http.Client

	Services map[string]Service
	Token Token
	User User
	Metadata Metadata
}

func (c *OpenStackClient) request(url string, body interface{}) ([]byte, error) {
	if c.client == nil {
		c.client = &http.Client{CheckRedirect: nil}
	}

	json, err := json.Marshal(body)
	if err != nil {
		return nil, ErrorContextf(&err, "failed marshalling the request body")
	}

	reqBody := strings.NewReader(string(json))
	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, ErrorContextf(&err, "failed creating the request")
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, ErrorContextf(&err, "failed executing the request")
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrorContextf(&err, "failed reading the response body")
	}

	return respBody, nil
}

func (c *OpenStackClient) Authenticate(username, password, tenant string) error {
	var req struct {
		Auth struct {
			PasswordCredentials struct {
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"passwordCredentials"`
			TenantName string `json:"tenantName"`
	    } `json:"auth"`
	}
	req.Auth.PasswordCredentials.Username = username
	req.Auth.PasswordCredentials.Password = password
	req.Auth.TenantName = tenant

	respBody, err := c.request(c.IdentityEndpoint + OS_API_TOKENS, req)
	if err != nil {
		return ErrorContextf(&err, "authentication failed")
	}

	var resp struct {
		Access struct {
			Token Token
			ServiceCatalog []Service
			User User
			Metadata Metadata
		}
	}
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return ErrorContextf(&err, "failed unmarshaling the response body")
	}

	if c.Services == nil {
		c.Services = make(map[string]Service)
	}
	for _, s := range resp.Access.ServiceCatalog {
		c.Services[s.Name] = s
	}
	c.Token = resp.Access.Token
	c.User = resp.Access.User
	c.Metadata = resp.Access.Metadata
	return nil
}

func main() {
	username, password, tenant, auth_url := getEnvVars()
	client := &OpenStackClient{IdentityEndpoint: auth_url}
	err := client.Authenticate(username, password, tenant)
	if err != nil {
		panic("Error: " + err.Error())
	}
	fmt.Printf("authenticated successfully: token=%s\n", client.Token.Id)
}
