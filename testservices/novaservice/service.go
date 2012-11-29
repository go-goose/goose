// Nova double testing service - internal direct API implementation

package novaservice

import (
	"fmt"
	"launchpad.net/goose/nova"
)

type Nova struct {
	flavors      map[string]Flavor
	servers      map[string]Server
	groups       map[int]nova.SecurityGroup
	rules        map[int]nova.SecurityGroupRule
	floatingIPs  map[int]nova.FloatingIP
	serverGroups map[string][]int
	serverIPs    map[string][]int
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
		rules:        make(map[int]nova.SecurityGroupRule),
		floatingIPs:  make(map[int]nova.FloatingIP),
		serverGroups: make(map[string][]int),
		serverIPs:    make(map[string][]int),
		hostname:     hostname,
		baseURL:      baseURL,
		token:        token,
	}
	return nova
}

// AddFlavor creates a new flavor.
func (n *Nova) AddFlavor(flavor Flavor) error {
	id := ""
	if flavor.entity == nil && flavor.detail == nil {
		return fmt.Errorf("refusing to add a nil flavor")
	} else if flavor.entity != nil {
		id = flavor.entity.Id
	} else {
		id = flavor.detail.Id
	}
	if n.HasFlavor(id) {
		return fmt.Errorf("a flavor with id %q already exists", id)
	}
	n.flavors[id] = flavor
	return nil
}

// HasFlavor verifies the given flavor exists or not.
func (n *Nova) HasFlavor(flavorId string) bool {
	_, ok := n.flavors[flavorId]
	return ok
}

// GetFlavor retrieves an existing flavor by ID.
func (n *Nova) GetFlavor(flavorId string) (Flavor, error) {
	flavor, ok := n.flavors[flavorId]
	if !ok {
		return Flavor{}, fmt.Errorf("no such flavor %q", flavorId)
	}
	return flavor, nil
}

// AllFlavors returns a list of all existing flavors.
func (n *Nova) AllFlavors() ([]Flavor, error) {
	if len(n.flavors) == 0 {
		return nil, fmt.Errorf("no flavors to return")
	}
	flavors := make([]Flavor, len(n.flavors))
	for _, flavor := range n.flavors {
		flavors = append(flavors, flavor)
	}
	return flavors, nil
}

// RemoveFlavor deletes an existing flavor.
func (n *Nova) RemoveFlavor(flavorId string) error {
	if !n.HasFlavor(flavorId) {
		return fmt.Errorf("no such flavor %q", flavorId)
	}
	delete(n.flavors, flavorId)
	return nil
}

// AddServer creates a new server.
func (n *Nova) AddServer(server Server) error {
	id := ""
	if server.server == nil && server.detail == nil {
		return fmt.Errorf("refusing to add a nil server")
	} else if server.server != nil {
		id = server.server.Id
	} else {
		id = server.detail.Id
	}
	if n.HasServer(id) {
		return fmt.Errorf("a server with id %q already exists", id)
	}
	n.servers[id] = server
	return nil
}

// HasServer verifies the given server exists or not.
func (n *Nova) HasServer(serverId string) bool {
	_, ok := n.servers[serverId]
	return ok
}

// GetServer retrieves an existing server by ID.
func (n *Nova) GetServer(serverId string) (Server, error) {
	server, ok := n.servers[serverId]
	if !ok {
		return Server{}, fmt.Errorf("no such server %q", serverId)
	}
	return server, nil
}

// AllServers returns a list of all existing servers.
func (n *Nova) AllServers() ([]Server, error) {
	if len(n.servers) == 0 {
		return nil, fmt.Errorf("no servers to return")
	}
	servers := make([]Server, len(n.servers))
	for _, server := range n.servers {
		servers = append(servers, server)
	}
	return servers, nil
}

// RemoveServer deletes an existing server.
func (n *Nova) RemoveServer(serverId string) error {
	if !n.HasServer(serverId) {
		return fmt.Errorf("no such server %q", serverId)
	}
	delete(n.servers, serverId)
	return nil
}

// AddSecurityGroup creates a new security group.
func (n *Nova) AddSecurityGroup(group nova.SecurityGroup) error {
	if n.HasSecurityGroup(group.Id) {
		return fmt.Errorf("group with id %d already exists", group.Id)
	}
	n.groups[group.Id] = group
	return nil
}

// HasSecurityGroup verifies the given security group exists.
func (n *Nova) HasSecurityGroup(groupId int) bool {
	_, ok := n.groups[groupId]
	return ok
}

// GetSecurityGroup retrieves an existing group by ID.
func (n *Nova) GetSecurityGroup(groupId int) (nova.SecurityGroup, error) {
	group, ok := n.groups[groupId]
	if !ok {
		return nova.SecurityGroup{}, fmt.Errorf("no such security group %d", groupId)
	}
	return group, nil
}

// AllSecurityGroups returns a list of all existing groups.
func (n *Nova) AllSecurityGroups() ([]nova.SecurityGroup, error) {
	if len(n.groups) == 0 {
		return nil, fmt.Errorf("no security groups to return")
	}
	groups := make([]nova.SecurityGroup, len(n.groups))
	for _, group := range n.groups {
		groups = append(groups, group)
	}
	return groups, nil
}

// RemoveSecurityGroup deletes an existing group.
func (n *Nova) RemoveSecurityGroup(groupId int) error {
	if !n.HasSecurityGroup(groupId) {
		return fmt.Errorf("no such security group %d", groupId)
	}
	delete(n.groups, groupId)
	return nil
}

// AddSecurityGroupRule creates a new rule in an existing group.
func (n *Nova) AddSecurityGroupRule(ruleId int, rule nova.RuleInfo) error {
	_, ok := n.rules[ruleId]
	if ok {
		return fmt.Errorf("a security group rule with id %d already exists", ruleId)
	}
	group, ok := n.groups[rule.ParentGroupId]
	if !ok {
		return fmt.Errorf("trying to add a rule to unknown security group %d", rule.ParentGroupId)
	}
	for _, ru := range group.Rules {
		if ru.Id == ruleId {
			return fmt.Errorf("cannot add twice rule %d to security group %d", ru.Id, group.Id)
		}
	}
	newrule := nova.SecurityGroupRule{
		ParentGroupId: rule.ParentGroupId,
		Id:            ruleId,
	}
	if rule.GroupId != nil {
		newrule.FromPort = &rule.FromPort
		newrule.ToPort = &rule.ToPort
		newrule.IPProtocol = &rule.IPProtocol
		newrule.IPRange = make(map[string]string)
		newrule.IPRange["cidr"] = rule.Cidr
	}
	group.Rules = append(group.Rules, newrule)
	n.groups[group.Id] = group
	n.rules[newrule.Id] = newrule
	return nil
}

// HasSecurityGroupRule verifies the given group contains the given rule.
// If groupId is -1, it verifies if the rule exists only.
func (n *Nova) HasSecurityGroupRule(groupId, ruleId int) bool {
	rule, ok := n.rules[groupId]
	if !ok {
		return false
	}
	if groupId != -1 {
		if !n.HasSecurityGroup(groupId) {
			return false
		}
		return rule.ParentGroupId == groupId
	}
	return true
}

// GetSecurityGroupRule retrieves an existing rule by ID.
func (n *Nova) GetSecurityGroupRule(ruleId int) (nova.SecurityGroupRule, error) {
	rule, ok := n.rules[ruleId]
	if !ok {
		return nova.SecurityGroupRule{}, fmt.Errorf("no such security group rule %d", ruleId)
	}
	return rule, nil
}

// RemoveSecurityGroupRule deletes an existing rule from its group.
func (n *Nova) RemoveSecurityGroupRule(ruleId int) error {
	rule, ok := n.rules[ruleId]
	if !ok {
		return fmt.Errorf("no such security group rule %d", ruleId)
	}
	group, ok := n.groups[rule.ParentGroupId]
	if ok {
		idx := -1
		for ri, ru := range group.Rules {
			if ru.Id == ruleId {
				idx = ri
				break
			}
		}
		if idx != -1 {
			group.Rules = append(group.Rules[:idx], group.Rules[idx+1:]...)
			n.groups[group.Id] = group
		}
		// Silently ignore missing rules...
	}
	// ...or groups
	delete(n.rules, ruleId)
	return nil
}

// AddServerSecurityGroup attaches an existing server to a group.
func (n *Nova) AddServerSecurityGroup(serverId string, groupId int) error {
	if !n.HasServer(serverId) {
		return fmt.Errorf("no such server %q", serverId)
	}
	groups, ok := n.serverGroups[serverId]
	if ok {
		for _, gid := range groups {
			if gid == groupId {
				return fmt.Errorf("server %q already belongs to group %d", serverId, groupId)
			}
		}
	}
	if !n.HasSecurityGroup(groupId) {
		return fmt.Errorf("no such security group %d", groupId)
	}
	groups = append(groups, groupId)
	n.serverGroups[serverId] = groups
	return nil
}

// HasServerSecurityGroup verifies the given server is part of the group.
func (n *Nova) HasServerSecurityGroup(serverId string, groupId int) bool {
	if !n.HasServer(serverId) || !n.HasSecurityGroup(groupId) {
		return false
	}
	groups, ok := n.serverGroups[serverId]
	if !ok {
		return false
	}
	for _, gid := range groups {
		if gid == groupId {
			return true
		}
	}
	return false
}

// RemoveServerSecurityGroup detaches an existing server from a group.
func (n *Nova) RemoveServerSecurityGroup(serverId string, groupId int) error {
	groups, ok := n.serverGroups[serverId]
	if !ok {
		return fmt.Errorf("server %q does not belong to any groups", serverId)
	}
	idx := -1
	for gi, gid := range groups {
		if gid == groupId {
			idx = gi
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("server %q does not belong to group %d", serverId, groupId)
	}
	if !n.HasSecurityGroup(groupId) {
		return fmt.Errorf("no such security group %d", groupId)
	}
	groups = append(groups[:idx], groups[idx+1:]...)
	n.serverGroups[serverId] = groups
	return nil
}

// AddFloatingIP creates a new floating IP address in the pool.
func (n *Nova) AddFloatingIP(ip nova.FloatingIP) error {
	_, ok := n.floatingIPs[ip.Id]
	if ok {
		fmt.Errorf("floating IP with id %d already exists", ip.Id)
	}
	n.floatingIPs[ip.Id] = ip
	return nil
}

// HasFloatingIP verifies the given floating IP address exists.
func (n *Nova) HasFloatingIP(address string) bool {
	if len(n.floatingIPs) == 0 {
		return false
	}
	for _, fip := range n.floatingIPs {
		if fip.IP == address {
			return true
		}
	}
	return false
}

// GetFloatingIP retrieves the floating IP by ID.
func (n *Nova) GetFloatingIP(ipId int) (nova.FloatingIP, error) {
	for fipid, fip := range n.floatingIPs {
		if fipid == ipId {
			return fip, nil
		}
	}
	return nova.FloatingIP{}, fmt.Errorf("no such floating IP %d", ipId)
}

// AllFlotingIPs returns a list of all created floating IPs.
func (n *Nova) AllFlotingIPs() ([]nova.FloatingIP, error) {
	if len(n.floatingIPs) == 0 {
		return nil, fmt.Errorf("no floating IPs to return")
	}
	fips := make([]nova.FloatingIP, len(n.floatingIPs))
	for _, fip := range n.floatingIPs {
		fips = append(fips, fip)
	}
	return fips, nil
}

// RemoveFloatingIP deletes an existing floating IP by ID.
func (n *Nova) RemoveFloatingIP(ipId int) error {
	_, ok := n.floatingIPs[ipId]
	if !ok {
		fmt.Errorf("no such floating IP %d", ipId)
	}
	delete(n.floatingIPs, ipId)
	return nil
}

// AddServerFloatingIP attaches an existing floating IP to a server.
func (n *Nova) AddServerFloatingIP(serverId string, ipId int) error {
	if !n.HasServer(serverId) {
		return fmt.Errorf("no such server %q", serverId)
	}
	_, ok := n.floatingIPs[ipId]
	if !ok {
		return fmt.Errorf("no such floating IP %d", ipId)
	}
	fips, ok := n.serverIPs[serverId]
	if ok {
		return fmt.Errorf("server %q already has floating IP %d", serverId, ipId)
	}
	fips = append(fips, ipId)
	n.serverIPs[serverId] = fips
	return nil
}

// HasServerFloatingIP verifies the given floating IP belongs to a server.
func (n *Nova) HasServerFloatingIP(serverId, address string) bool {
	if !n.HasServer(serverId) {
		return false
	}
	fips, ok := n.serverIPs[serverId]
	if !ok {
		return false
	}
	for _, fipId := range fips {
		fip, ok := n.floatingIPs[fipId]
		if !ok {
			return false
		}
		if fip.IP == address {
			return true
		}
	}
	return false
}

// RemoveServerFloatingIP deletes an attached floating IP from a server.
func (n *Nova) RemoveServerFloatingIP(serverId string, ipId int) error {
	if !n.HasServer(serverId) {
		return fmt.Errorf("no such server %q", serverId)
	}
	fips, ok := n.serverIPs[serverId]
	if !ok {
		return fmt.Errorf("server %q does not have any floating IPs to remove", serverId)
	}
	idx := -1
	for fi, fipId := range fips {
		if fipId == ipId {
			_, ok := n.floatingIPs[fipId]
			if !ok {
				return fmt.Errorf("no such floating IP %d", ipId)
			}
			idx = fi
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("server %q does not have floating IP %d", serverId, ipId)
	}
	fips = append(fips[:idx], fips[idx+1:]...)
	n.serverIPs[serverId] = fips
	return nil
}
