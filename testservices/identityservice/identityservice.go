package identityservice

import (
	"net/http"
)

type IdentityService interface {
	AddUser(user, secret, token string)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
