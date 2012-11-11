package identityservice

import (
	"net/http"
)

type Legacy struct {
	tokens        map[string]UserInfo
	managementURL string
}

func NewLegacy() *Legacy {
	service := &Legacy{}
	service.tokens = make(map[string]UserInfo)
	return service
}

func (lis *Legacy) SetManagementURL(URL string) {
	lis.managementURL = URL
}

func (lis *Legacy) AddUser(user, secret string) string {
	token := randomHexToken()
	lis.tokens[user] = UserInfo{secret: secret, token: token}
	return token
}

func (lis *Legacy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Auth-User")
	info, ok := lis.tokens[username]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	auth_key := r.Header.Get("X-Auth-Key")
	if auth_key != info.secret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	header := w.Header()
	header.Set("X-Auth-Token", info.token)
	header.Set("X-Server-Management-Url", lis.managementURL)
	w.WriteHeader(http.StatusNoContent)
}
