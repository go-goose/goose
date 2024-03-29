package testservices

import (
	"net/http"

	"github.com/go-goose/goose/v5/testservices/hook"
	"github.com/go-goose/goose/v5/testservices/identityservice"
)

// An HttpService provides the HTTP API for a service double.
type HttpService interface {
	SetupHTTP(mux *http.ServeMux)
	Stop()
}

// A ServiceInstance is an Openstack module, one of nova, swift, glance.
type ServiceInstance struct {
	identityservice.ServiceProvider
	hook.TestService
	IdentityService identityservice.IdentityService
	// For Keystone V3, V2 is also accepted as an identity service
	// this represents that possibility.
	FallbackIdentityService identityservice.IdentityService
	Scheme                  string
	Hostname                string
	VersionPath             string
	TenantId                string
	Region                  string
	RegionID                string
}

// Internal Openstack errors.
var RateLimitExceededError = NewRateLimitExceededError()
var TooManyRequestsError = NewTooManyRequestsError()
var ForbiddenRateLimitError = NewForbiddenRateLimitError()
var ServiceUnavailRateLimitError = NewServiceUnavailRateLimitError()

// NoMoreFloatingIPs corresponds to "HTTP 404 Zero floating ips available."
var NoMoreFloatingIPs = NewNoMoreFloatingIpsError()

// IPLimitExceeded corresponds to "HTTP 413 Maximum number of floating ips exceeded"
var IPLimitExceeded = NewIPLimitExceededError()

// AvailabilityZoneIsNotAvailable corresponds to
// "HTTP 400 The requested availability zone is not available"
var AvailabilityZoneIsNotAvailable = NewAvailabilityZoneIsNotAvailableError()
