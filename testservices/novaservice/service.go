// Nova double testing service - internal direct API implementation

package novaservice

import (
	"launchpad.net/goose/nova"
)

type Nova struct {
	flavors      map[string]Flavor
	servers      map[string]Server
	groups       map[int]nova.SecurityGroup
	floatingIPs  map[int]nova.FloatingIP
	groupRules   map[int]int
	serverGroups map[string]int
	serverIPs    map[string]int
	hostname     string
	baseURL      string
	token        string
}

// New creates an instance of the Nova object, given the parameters.
func New(hostname, baseURL, token string) *Nova {
	nova := &Nova{
		flavors:      make(map[string]Flavor),
		servers:      make(map[string]Server),
		groups:       make(map[int]nova.SecurityGroup),
		floatingIPs:  make(map[int]nova.FloatingIP),
		groupRules:   make(map[int]int),
		serverGroups: make(map[string]int),
		serverIPs:    make(map[string]int),
		hostname:     hostname,
		baseURL:      baseURL,
		token:        token,
	}
	return nova
}

// AddFlavor creates a new flavor.
func (n *Nova) AddFlavor(flavor Flavor) error {
	return nil
}

// HasFlavor verifies the given flavor exists or not.
func (n *Nova) HasFlavor(flavorId string) bool {
	return false
}

// GetFlavor retrieves an existing flavor by ID.
func (n *Nova) GetFlavor(flavorId string) (Flavor, error) {
	return Flavor{}, nil
}

// AllFlavors returns a list of all existing flavors.
func (n *Nova) AllFlavors() ([]Flavor, error) {
	return nil, nil
}

// RemoveFlavor deletes an existing flavor.
func (n *Nova) RemoveFlavor(flavorId string) error {
	return nil
}

// AddServer creates a new server.
func (n *Nova) AddServer(server Server) error {
	return nil
}

// HasServer verifies the given server exists or not.
func (n *Nova) HasServer(serverId string) bool {
	return false
}

// GetServer retrieves an existing server by ID.
func (n *Nova) GetServer(serverId string) (Server, error) {
	return Server{}, nil
}

// AllServers returns a list of all existing servers.
func (n *Nova) AllServers() ([]Server, error) {
	return nil, nil
}

// RemoveServer deletes an existing server.
func (n *Nova) RemoveServer(serverId string) error {
	return nil
}

// AddSecurityGroup creates a new security group.
func (n *Nova) AddSecurityGroup(group nova.SecurityGroup) error {
	return nil
}

// HasSecurityGroup verifies the given security group exists.
func (n *Nova) HasSecurityGroup(groupId int) bool {
	return false
}

// GetSecurityGroup retrieves an existing group by ID.
func (n *Nova) GetSecurityGroup(groupId int) (nova.SecurityGroup, error) {
	return nova.SecurityGroup{}, nil
}

// AllSecurityGroups returns a list of all existing groups.
func (n *Nova) AllSecurityGroups() ([]nova.SecurityGroup, error) {
	return nil, nil
}

// RemoveSecurityGroup deletes an existing group.
func (n *Nova) RemoveSecurityGroup(groupId int) error {
	return nil
}

// AddSecurityGroupRule creates a new rule in an existing group.
func (n *Nova) AddSecurityGroupRule(groupId int, rule nova.RuleInfo) error {
	return nil
}

// HasSecurityGroupRule verifies the given group contains the given rule.
func (n *Nova) HasSecurityGroupRule(groupId, ruleId int) bool {
	return false
}

// GetSecurityGroupRule retrieves an existing rule by ID.
func (n *Nova) GetSecurityGroupRule(ruleId int) (nova.SecurityGroupRule, error) {
	return nova.SecurityGroupRule{}, nil
}

// RemoveSecurityGroupRule deletes an existing rule from its group.
func (n *Nova) RemoveSecurityGroupRule(groupId, ruleId int) error {
	return nil
}

// AddServerSecurityGroup attaches an existing server to a group.
func (n *Nova) AddServerSecurityGroup(serverId string, groupId int) error {
	return nil
}

// HasServerSecurityGroup verifies the given server is part of the group.
func (n *Nova) HasServerSecurityGroup(serverId string, groupId int) bool {
	return false
}

// RemoveServerSecurityGroup detaches an existing server from a group.
func (n *Nova) RemoveServerSecurityGroup(serverId string, groupId int) error {
	return nil
}

// AddFloatingIP creates a new floating IP address in the pool.
func (n *Nova) AddFloatingIP(ip nova.FloatingIP) error {
	return nil
}

// HasFloatingIP verifies the given floating IP address exists.
func (n *Nova) HasFloatingIP(address string) bool {
	return false
}

// GetFloatingIP retrieves the floating IP by ID.
func (n *Nova) GetFloatingIP(ipId int) (nova.FloatingIP, error) {
	return nova.FloatingIP{}, nil
}

// AllFlotingIPs returns a list of all created floating IPs.
func (n *Nova) AllFlotingIPs() ([]nova.FloatingIP, error) {
	return nil, nil
}

// RemoveFloatingIP deletes an existing floating IP by ID.
func (n *Nova) RemoveFloatingIP(ipId int) error {
	return nil
}

// AddServerFloatingIP attaches an existing floating IP to a server.
func (n *Nova) AddServerFloatingIP(serverId string, ipId int) error {
	return nil
}

// HasServerFloatingIP verifies the given floating IP belongs to a server.
func (n *Nova) HasServerFloatingIP(serverId, address string) bool {
	return false
}

// RemoveServerFloatingIP deletes an attached floating IP from a server.
func (n *Nova) RemoveServerFloatingIP(serverId, ipId int) error {
	return nil
}
