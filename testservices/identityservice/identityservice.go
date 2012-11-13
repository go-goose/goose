package identityservice

import (
	"net/http"
)

type IdentityService interface {
	AddUser(user, secret string) (token string)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
