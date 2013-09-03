package identity

import (
	goosehttp "launchpad.net/goose/http"
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
	Client *goosehttp.Client
}

func (u *UserPass) Auth(creds *Credentials) (*AuthDetails, error) {
	if u.Client == nil {
		u.Client = goosehttp.New()
	}
	auth := authWrapper{Auth: authRequest{
		PasswordCredentials: passwordCredentials{
			Username: creds.User,
			Password: creds.Secrets,
		},
		TenantName: creds.TenantName}}

	return keystoneAuth(u.Client, auth, creds.URL)
}
