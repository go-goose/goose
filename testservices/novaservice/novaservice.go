// Nova double testing service - mimics OpenStack Nova compute service
// for testing goose against close-to-live API.

package novaservice

import (
	"launchpad.net/goose/nova"
	"net/http"
)

// NovaService presents an direct-API to manipulate the internal
// state, as well as an HTTP API double for OpenStack Nova.
type NovaService interface {
	// AddFlavor creates a new flavor.
	AddFlavor(flavor nova.FlavorDetail) error

	// GetFlavor retrieves an existing flavor by ID.
	GetFlavor(flavorId string) (nova.FlavorDetail, error)

	// GetFlavorAsEntity returns the stored FlavorDetail as Entity.
	GetFlavorAsEntity(flavorId string) (nova.Entity, error)

	// AllFlavors returns a list of all existing flavors.
	AllFlavors() ([]nova.FlavorDetail, error)

	// AllFlavorsAsEntities returns all flavors as Entity structs.
	AllFlavorsAsEntities() ([]nova.Entity, error)

	// RemoveFlavor deletes an existing flavor.
	RemoveFlavor(flavorId string) error

	// AddServer creates a new server.
	AddServer(server nova.ServerDetail) error

	// GetServer retrieves an existing server by ID.
	GetServer(serverId string) (nova.ServerDetail, error)

	// GetServerAsEntity returns the stored ServerDetail as Entity.
	GetServerAsEntity(serverId string) (nova.Entity, error)

	// AllServers returns a list of all existing servers.
	AllServers() ([]nova.ServerDetail, error)

	// AllServersAsEntities returns all servers as Entity structs.
	AllServersAsEntities() ([]nova.Entity, error)

	// RemoveServer deletes an existing server.
	RemoveServer(serverId string) error

	// AddSecurityGroup creates a new security group.
	AddSecurityGroup(group nova.SecurityGroup) error

	// GetSecurityGroup retrieves an existing group by ID.
	GetSecurityGroup(groupId int) (nova.SecurityGroup, error)

	// AllSecurityGroups returns a list of all existing groups.
	AllSecurityGroups() ([]nova.SecurityGroup, error)

	// RemoveSecurityGroup deletes an existing group.
	RemoveSecurityGroup(groupId int) error

	// AddSecurityGroupRule creates a new rule in an existing group.
	AddSecurityGroupRule(ruleId int, rule nova.RuleInfo) error

	// HasSecurityGroupRule returns whether the given group contains the rule.
	HasSecurityGroupRule(groupId, ruleId int) bool

	// GetSecurityGroupRule retrieves an existing rule by ID.
	GetSecurityGroupRule(ruleId int) (nova.SecurityGroupRule, error)

	// RemoveSecurityGroupRule deletes an existing rule from its group.
	RemoveSecurityGroupRule(ruleId int) error

	// AddServerSecurityGroup attaches an existing server to a group.
	AddServerSecurityGroup(serverId string, groupId int) error

	// HasServerSecurityGroup returns whether the given server belongs to the group.
	HasServerSecurityGroup(serverId string, groupId int) bool

	// RemoveServerSecurityGroup detaches an existing server from a group.
	RemoveServerSecurityGroup(serverId string, groupId int) error

	// AddFloatingIP creates a new floating IP address in the pool.
	AddFloatingIP(ip nova.FloatingIP) error

	// HasFloatingIP returns whether the given floating IP address exists.
	HasFloatingIP(address string) bool

	// GetFloatingIP retrieves the floating IP by ID.
	GetFloatingIP(ipId int) (nova.FloatingIP, error)

	// AllFloatingIPs returns a list of all created floating IPs.
	AllFloatingIPs() ([]nova.FloatingIP, error)

	// RemoveFloatingIP deletes an existing floating IP by ID.
	RemoveFloatingIP(ipId int) error

	// AddServerFloatingIP attaches an existing floating IP to a server.
	AddServerFloatingIP(serverId string, ipId int) error

	// HasServerFloatingIP verifies the given floating IP belongs to a server.
	HasServerFloatingIP(serverId, address string) bool

	// RemoveServerFloatingIP deletes an attached floating IP from a server.
	RemoveServerFloatingIP(serverId string, ipId int) error

	// ServeHTTP is the main entry point in the HTTP request processing.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
