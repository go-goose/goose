package testservices

import (
	"launchpad.net/goose/testservices/identityservice"
	"net/http"
)

// An HttpService provides the HTTP API for a service double.
type HttpService interface {
	SetupHTTP(mux *http.ServeMux)
}

// A ServiceInstance is an Openstack module, one of nova, swift, glance.
type ServiceInstance struct {
	identityservice.ServiceProvider
	IdentityService identityservice.IdentityService
	Hostname        string
	VersionPath     string
	TenantId        string
	Region          string
}
