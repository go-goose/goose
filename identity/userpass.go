package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Title   string `json:"title"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("Failed: %d %s: %s", e.Code, e.Title, e.Message)
}

type ErrorWrapper struct {
	Error ErrorResponse `json:"error"`
}

type PasswordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthRequest struct {
	PasswordCredentials PasswordCredentials `json:"passwordCredentials"`
	TenantName          string              `json:"tenantName"`
}

type AuthWrapper struct {
	Auth AuthRequest `json:"auth"`
}

type Endpoint struct {
	AdminURL    string `json:"adminURL"`
	InternalURL string `json:"internalURL"`
	PublicURL   string `json:"publicURL"`
	Region      string `json:"region"`
}

type Service struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Endpoints []Endpoint
}

type TokenResponse struct {
	Expires string `json:"expires"` // should this be a date object?
	Id      string `json:"id"`      // Actual token string
	Tenant  struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
}

type RoleResponse struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	TenantId string `json:"tenantId"`
}

type UserResponse struct {
	Id    string         `json:"id"`
	Name  string         `json:"name"`
	Roles []RoleResponse `json:"roles"`
}

type AccessWrapper struct {
	Access AccessResponse `json:"access"`
}

type AccessResponse struct {
	ServiceCatalog []Service     `json:"serviceCatalog"`
	Token          TokenResponse `json:"token"`
	User           UserResponse  `json:"user"`
}

type UserPass struct {
}

func (l *UserPass) Auth(creds Credentials) (*AuthDetails, error) {
	client := &http.Client{}
	auth := AuthWrapper{Auth: AuthRequest{
		PasswordCredentials: PasswordCredentials{
			Username: creds.User,
			Password: creds.Secrets,
		},
		TenantName: "tenant-name"}}
	auth_json, err := json.Marshal(auth)
	request, err := http.NewRequest("POST", creds.URL+"/tokens", bytes.NewBuffer(auth_json))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		// Check if we have a JSON representation of the failure, if so, return it
		if response.Header.Get("Content-Type") == "application/json" {
			var wrappedErr ErrorWrapper
			if err := json.Unmarshal(content, &wrappedErr); err == nil {
				return nil, &wrappedErr.Error
			}
		}
		// We weren't able to parse the response, so just return our own error
		return nil, fmt.Errorf("Failed to Authenticate (code %d %s): %s",
			response.StatusCode, response.Status, content)
	}
	if response.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("Failed to Authenticate. Did not get JSON back: %s", content)
	}
	var access AccessWrapper
	if err := json.Unmarshal(content, &access); err != nil {
		return nil, err
	}
	details := &AuthDetails{}
	details.Token = access.Access.Token.Id
	if details.Token == "" {
		return nil, fmt.Errorf("Did not get valid Token from auth request")
	}
	details.ServiceURLs = make(map[string]string, len(access.Access.ServiceCatalog))
	for _, service := range access.Access.ServiceCatalog {
		details.ServiceURLs[service.Type] = service.Endpoints[0].PublicURL
	}

	return details, nil
}
