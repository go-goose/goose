package openstackservice

import (
	"launchpad.net/goose/identity"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/novaservice"
	"launchpad.net/goose/testservices/swiftservice"
	"net/http"
	"strings"
)

// Openstack provides an Openstack service double implementation.
type Openstack struct {
	Identity identityservice.IdentityService
	Nova     *novaservice.Nova
	Swift    *swiftservice.Swift
}

// New creates an instance of a full Openstack service double.
// An initial user with the specified credentials is registered with the identity service.
func New(cred *identity.Credentials) *Openstack {
	openstack := Openstack{
		Identity: identityservice.NewUserPass(),
	}
	userInfo := openstack.Identity.AddUser(cred.User, cred.Secrets, cred.TenantName)
	if cred.TenantName == "" {
		panic("Openstack service double requires a tenant to be specified.")
	}
	openstack.Nova = novaservice.New(cred.URL, "v2", userInfo.TenantId, cred.Region, openstack.Identity)
	// Create the swift service using only the region base so we emulate real world deployments.
	regionParts := strings.Split(cred.Region, ".")
	baseRegion := regionParts[len(regionParts)-1]
	openstack.Swift = swiftservice.New(cred.URL, "v1", userInfo.TenantId, baseRegion, openstack.Identity)
	return &openstack
}

// SetupHTTP attaches all the needed handlers to provide the HTTP API for the Openstack service..
func (openstack *Openstack) SetupHTTP(mux *http.ServeMux) {
	openstack.Identity.SetupHTTP(mux)
	openstack.Nova.SetupHTTP(mux)
	openstack.Swift.SetupHTTP(mux)
}
