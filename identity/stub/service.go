package identitystub

import (
	"net/http"
)

type UserInfo struct {
	secret string
	token  string
}

type LegacyIdentityService struct {
	tokens        map[string]UserInfo
	managementURL string
}

func NewLegacyIdentityService(managementURL string) *LegacyIdentityService {
	service := &LegacyIdentityService{}
	service.tokens = make(map[string]UserInfo)
	service.managementURL = managementURL
	return service
}

func (lis *LegacyIdentityService) AddUser(user, secret, token string) {
	lis.tokens[user] = UserInfo{secret: secret, token: token}
}

func (lis *LegacyIdentityService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
