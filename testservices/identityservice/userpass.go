package identityservice

import (
	"net/http"
)

// Implement the v2 User Pass form of identity (Keystone)

type UserPassRequest struct {
	Auth struct {
		PasswordCredentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
		TenantName string `json:"tenantName"`
	} `json:"auth"`
}

type UserPass struct {
	users map[string]UserInfo
}

func NewUserPass() *UserPass {
	userpass := &UserPass{users: make(map[string]UserInfo)}
	return userpass
}

func (u *UserPass) AddUser(user, secret, token string) {
	u.users[user] = UserInfo{secret: secret, token: token}
}

func (u *UserPass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
}
