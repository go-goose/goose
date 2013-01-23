package identityservice

import (
	"net/http"
)

// An IdentityService provides user authentication for an Openstack instance.
type IdentityService interface {
	HttpService
	AddUser(user, secret string) *UserInfo
	FindUser(token string) (*UserInfo, error)
	RegisterServiceProvider(name, serviceType string, serviceProvider ServiceProvider)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// An HttpService provides the HTTP API for a service double.
type HttpService interface {
	SetupHTTP(mux *http.ServeMux)
}

// A ServiceProvider is an Openstack module which has service endpoints.
type ServiceProvider interface {
	Endpoints() []Endpoint
}

// A ServiceInstance is an Openstack module, one of nova, swift, glance.
type ServiceInstance struct {
	ServiceProvider
	IdentityService IdentityService
	Hostname        string
	VersionPath     string
	TenantId        string
	Region          string
}
