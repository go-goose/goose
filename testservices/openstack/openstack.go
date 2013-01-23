package openstack

import (
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/novaservice"
	"launchpad.net/goose/testservices/swiftservice"
	"net/http"
)

// Openstack provides an Openstack service double implementation.
type Openstack struct {
	identity identityservice.IdentityService
	nova     *novaservice.Nova
	swift    *swiftservice.Swift
}

// New creates an instance of a full Openstack service double.
// An initial user with the specified credentials is registered with the identity service.
func New(baseURL string, username, password, region string) *Openstack {
	openstack := Openstack{
		identity: identityservice.NewUserPass(),
	}
	userInfo := openstack.identity.AddUser(username, password)
	openstack.nova = novaservice.New(baseURL, "v2", userInfo.TenantId, region, openstack.identity)
	openstack.swift = swiftservice.New(baseURL, "v1", userInfo.TenantId, region, openstack.identity)
	return &openstack
}

// setupHTTP attaches all the needed handlers to provide the HTTP API for the Openstack service..
func (openstack *Openstack) SetupHTTP(mux *http.ServeMux) {
	openstack.identity.SetupHTTP(mux)
	openstack.nova.SetupHTTP(mux)
	openstack.swift.SetupHTTP(mux)
}
