package identity

import (
	"fmt"
	"net/http"
	goosehttp "launchpad.net/goose/http"
)

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

type ServiceResponse struct {
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
		Description string `json:"description"`
		Enabled bool `json:"enabled"`
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
	ServiceCatalog []ServiceResponse `json:"serviceCatalog"`
	Token          TokenResponse `json:"token"`
	User           UserResponse  `json:"user"`
}

type UserPass struct {
	client *goosehttp.GooseHTTPClient
}

func (u *UserPass) Auth(creds *Credentials) (*AuthDetails, error) {
	if u.client == nil {
		u.client = &goosehttp.GooseHTTPClient{http.Client{CheckRedirect: nil}}
	}
	auth := AuthWrapper{Auth: AuthRequest{
		PasswordCredentials: PasswordCredentials{
			Username: creds.User,
			Password: creds.Secrets,
		},
		TenantName: creds.TenantName}}

	var accessWrapper AccessWrapper
	requestData := goosehttp.RequestData{ReqValue: auth, RespValue: &accessWrapper}
	err := u.client.JsonRequest("POST", creds.URL, &requestData)
	if err != nil {
		return nil, err
	}

	details := &AuthDetails{}
	access := accessWrapper.Access
	respToken := access.Token
	if respToken.Id == "" {
		return nil, fmt.Errorf("Did not get valid Token from auth request")
	}
	details.TokenId = respToken.Id
	details.TenantId = respToken.Tenant.Id
	details.UserId = access.User.Id
	details.ServiceURLs = make(map[string]string, len(access.ServiceCatalog))
	for _, service := range access.ServiceCatalog {
		for i, e := range service.Endpoints {
			if e.Region != creds.Region {
				service.Endpoints = append(service.Endpoints[:i], service.Endpoints[i+1:]...)
			}
		}
		details.ServiceURLs[service.Type] = service.Endpoints[0].PublicURL
	}

	return details, nil
}
