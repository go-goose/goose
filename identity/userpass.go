package identity

import (
	goosehttp "github.com/go-goose/goose/v5/http"
)

type passwordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authRequest struct {
	PasswordCredentials passwordCredentials `json:"passwordCredentials"`
	TenantName          string              `json:"tenantName"`
}

type authWrapper struct {
	Auth authRequest `json:"auth"`
}

type UserPass struct {
	client goosehttp.HttpClient
}

func (u *UserPass) Auth(creds *Credentials) (*AuthDetails, error) {
	if u.client == nil {
		u.client = goosehttp.New()
	}

	auth := authWrapper{Auth: authRequest{
		PasswordCredentials: passwordCredentials{
			Username: creds.User,
			Password: creds.Secrets,
		},
		TenantName: creds.TenantName}}

	// In Keystone v2 TenantName and TenantID can be interchangeable used.
	if creds.TenantName == "" {
		auth.Auth.TenantName = creds.TenantID
	}

	return keystoneAuth(u.client, auth, creds.URL)
}
