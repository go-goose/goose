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
	// buildFlavorLinks populates the Links field as needed.
	buildFlavorLinks(flavor *nova.FlavorDetail)

	// addFlavor creates a new flavor.
	addFlavor(flavor nova.FlavorDetail) error

	// getFlavor retrieves an existing flavor by ID.
	getFlavor(flavorId string) (*nova.FlavorDetail, error)

	// getFlavorAsEntity returns the stored FlavorDetail as Entity.
	getFlavorAsEntity(flavorId string) (*nova.Entity, error)

	// allFlavors returns a list of all existing flavors.
	allFlavors() []nova.FlavorDetail

	// allFlavorsAsEntities returns all flavors as Entity structs.
	allFlavorsAsEntities() []nova.Entity

	// removeFlavor deletes an existing flavor.
	removeFlavor(flavorId string) error

	// buildServerLinks populates the Links field as needed.
	buildServerLinks(server *nova.ServerDetail)

	// addServer creates a new server.
	addServer(server nova.ServerDetail) error

	// getServer retrieves an existing server by ID.
	getServer(serverId string) (*nova.ServerDetail, error)

	// getServerAsEntity returns the stored ServerDetail as Entity.
	getServerAsEntity(serverId string) (*nova.Entity, error)

	// allServers returns a list of all existing servers.
	allServers() []nova.ServerDetail

	// allServersAsEntities returns all servers as Entity structs.
	allServersAsEntities() []nova.Entity

	// removeServer deletes an existing server.
	removeServer(serverId string) error

	// addSecurityGroup creates a new security group.
	addSecurityGroup(group nova.SecurityGroup) error

	// getSecurityGroup retrieves an existing group by ID.
	getSecurityGroup(groupId int) (*nova.SecurityGroup, error)

	// allSecurityGroups returns a list of all existing groups.
	allSecurityGroups() []nova.SecurityGroup

	// removeSecurityGroup deletes an existing group.
	removeSecurityGroup(groupId int) error

	// addSecurityGroupRule creates a new rule in an existing group.
	addSecurityGroupRule(ruleId int, rule nova.RuleInfo) error

	// hasSecurityGroupRule returns whether the given group contains the rule.
	hasSecurityGroupRule(groupId, ruleId int) bool

	// getSecurityGroupRule retrieves an existing rule by ID.
	getSecurityGroupRule(ruleId int) (*nova.SecurityGroupRule, error)

	// removeSecurityGroupRule deletes an existing rule from its group.
	removeSecurityGroupRule(ruleId int) error

	// addServerSecurityGroup attaches an existing server to a group.
	addServerSecurityGroup(serverId string, groupId int) error

	// hasServerSecurityGroup returns whether the given server belongs to the group.
	hasServerSecurityGroup(serverId string, groupId int) bool

	// removeServerSecurityGroup detaches an existing server from a group.
	removeServerSecurityGroup(serverId string, groupId int) error

	// addFloatingIP creates a new floating IP address in the pool.
	addFloatingIP(ip nova.FloatingIP) error

	// hasFloatingIP returns whether the given floating IP address exists.
	hasFloatingIP(address string) bool

	// getFloatingIP retrieves the floating IP by ID.
	getFloatingIP(ipId int) (*nova.FloatingIP, error)

	// allFloatingIPs returns a list of all created floating IPs.
	allFloatingIPs() []nova.FloatingIP

	// removeFloatingIP deletes an existing floating IP by ID.
	removeFloatingIP(ipId int) error

	// addServerFloatingIP attaches an existing floating IP to a server.
	addServerFloatingIP(serverId string, ipId int) error

	// hasServerFloatingIP verifies the given floating IP belongs to a server.
	hasServerFloatingIP(serverId, address string) bool

	// removeServerFloatingIP deletes an attached floating IP from a server.
	removeServerFloatingIP(serverId string, ipId int) error

	// ServeHTTP is the main entry point in the HTTP request processing.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
