// Nova double testing service - mimics OpenStack Nova compute service
// for testing goose against close-to-live API.

package novaservice

import (
	"launchpad.net/goose/nova"
	"net/http"
)

// Flavor holds either one or both of the flavor information.
type Flavor struct {
	entity *nova.Entity
	detail *nova.FlavorDetail
}

// Server holds either one or both of the server information.
type Server struct {
	server *nova.Entity
	detail *nova.ServerDetail
}

// NovaService presents an direct-API to manipulate the internal
// state, as well as an HTTP API double for OpenStack Nova.
type NovaService interface {
	// AddFlavor creates a new flavor.
	AddFlavor(flavor Flavor) error

	// HasFlavor verifies the given flavor exists or not.
	HasFlavor(flavorId string) bool

	// GetFlavor retrieves an existing flavor by ID.
	GetFlavor(flavorId string) (Flavor, error)

	// AllFlavors returns a list of all existing flavors.
	AllFlavors() ([]Flavor, error)

	// RemoveFlavor deletes an existing flavor.
	RemoveFlavor(flavorId string) error

	// AddServer creates a new server.
	AddServer(server Server) error

	// HasServer verifies the given server exists or not.
	HasServer(serverId string) bool

	// GetServer retrieves an existing server by ID.
	GetServer(serverId string) (Server, error)

	// AllServers returns a list of all existing servers.
	AllServers() ([]Server, error)

	// RemoveServer deletes an existing server.
	RemoveServer(serverId string) error

	// AddSecurityGroup creates a new security group.
	AddSecurityGroup(group nova.SecurityGroup) error

	// HasSecurityGroup verifies the given security group exists.
	HasSecurityGroup(groupId int) bool

	// GetSecurityGroup retrieves an existing group by ID.
	GetSecurityGroup(groupId int) (nova.SecurityGroup, error)

	// AllSecurityGroups returns a list of all existing groups.
	AllSecurityGroups() ([]nova.SecurityGroup, error)

	// RemoveSecurityGroup deletes an existing group.
	RemoveSecurityGroup(groupId int) error

	// AddSecurityGroupRule creates a new rule in an existing group.
	AddSecurityGroupRule(groupId int, rule nova.RuleInfo) error

	// HasSecurityGroupRule verifies the given group contains the given rule.
	HasSecurityGroupRule(groupId, ruleId int) bool

	// GetSecurityGroupRule retrieves an existing rule by ID.
	GetSecurityGroupRule(ruleId int) (nova.SecurityGroupRule, error)

	// RemoveSecurityGroupRule deletes an existing rule from its group.
	RemoveSecurityGroupRule(groupId, ruleId int) error

	// AddServerSecurityGroup attaches an existing server to a group.
	AddServerSecurityGroup(serverId string, groupId int) error

	// HasServerSecurityGroup verifies the given server is part of the group.
	HasServerSecurityGroup(serverId string, groupId int) bool

	// RemoveServerSecurityGroup detaches an existing server from a group.
	RemoveServerSecurityGroup(serverId string, groupId int) error

	// AddFloatingIP creates a new floating IP address in the pool.
	AddFloatingIP(ip nova.FloatingIP) error

	// HasFloatingIP verifies the given floating IP address exists.
	HasFloatingIP(address string) bool

	// GetFloatingIP retrieves the floating IP by ID.
	GetFloatingIP(ipId int) (nova.FloatingIP, error)

	// AllFlotingIPs returns a list of all created floating IPs.
	AllFlotingIPs() ([]nova.FloatingIP, error)

	// RemoveFloatingIP deletes an existing floating IP by ID.
	RemoveFloatingIP(ipId int) error

	// AddServerFloatingIP attaches an existing floating IP to a server.
	AddServerFloatingIP(serverId string, ipId int) error

	// HasServerFloatingIP verifies the given floating IP belongs to a server.
	HasServerFloatingIP(serverId, address string) bool

	// RemoveServerFloatingIP deletes an attached floating IP from a server.
	RemoveServerFloatingIP(serverId, ipId int) error

	// ServeHTTP is the main entry point in the HTTP request processing.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
