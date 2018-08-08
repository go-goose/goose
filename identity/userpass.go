package identity

import (
	goosehttp "gopkg.in/goose.v2/http"
)

type passwordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authRequest struct {
	PasswordCredentials passwordCredentials `json:"passwordCredentials"`
	TenantName          string              `json:"tenantName"`
	TenantID            string              `json:"tenantID"`
}

type authWrapper struct {
	Auth authRequest `json:"auth"`
}

type UserPass struct {
	client *goosehttp.Client
}

func (u *UserPass) Auth(creds *Credentials) (*AuthDetails, error) {
	if u.client == nil {
		u.client = goosehttp.New()
	}
	// In Keystone v2 TenantName and TenantID can be interchangeable used.
	auth := authWrapper{Auth: authRequest{
		PasswordCredentials: passwordCredentials{
			Username: creds.User,
			Password: creds.Secrets,
		},
		TenantName: creds.TenantName,
		TenantID:   creds.TenantID,
	}}

	return keystoneAuth(u.client, auth, creds.URL)
}
