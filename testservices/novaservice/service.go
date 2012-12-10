// Nova double testing service - internal direct API implementation

package novaservice

import (
	"fmt"
	"launchpad.net/goose/nova"
	"strings"
)

// Nova contains the service double's internal state.
type Nova struct {
	flavors      map[string]nova.FlavorDetail
	servers      map[string]nova.ServerDetail
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
		flavors:      make(map[string]nova.FlavorDetail),
		servers:      make(map[string]nova.ServerDetail),
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

// buildFlavorLinks populates the Links field of the passed
// FlavorDetail as needed by OpenStack HTTP API. Call this
// before addFlavor().
func (n *Nova) buildFlavorLinks(flavor *nova.FlavorDetail) {
	ep := n.hostname
	ver := strings.TrimLeft(n.baseURL, "/")
	url := n.token + "/flavors/" + flavor.Id
	flavor.Links = []nova.Link{
		nova.Link{Href: ep + ver + url, Rel: "self"},
		nova.Link{Href: ep + url, Rel: "bookmark"},
	}
}

// addFlavor creates a new flavor.
func (n *Nova) addFlavor(flavor nova.FlavorDetail) error {
	if _, err := n.getFlavor(flavor.Id); err == nil {
		return fmt.Errorf("a flavor with id %q already exists", flavor.Id)
	}
	n.flavors[flavor.Id] = flavor
	return nil
}

// getFlavor retrieves an existing flavor by ID.
func (n *Nova) getFlavor(flavorId string) (*nova.FlavorDetail, error) {
	flavor, ok := n.flavors[flavorId]
	if !ok {
		return nil, fmt.Errorf("no such flavor %q", flavorId)
	}
	return &flavor, nil
}

// getFlavorAsEntity returns the stored FlavorDetail as Entity.
func (n *Nova) getFlavorAsEntity(flavorId string) (*nova.Entity, error) {
	flavor, err := n.getFlavor(flavorId)
	if err != nil {
		return nil, err
	}
	return &nova.Entity{
		Id:    flavor.Id,
		Name:  flavor.Name,
		Links: flavor.Links,
	}, nil
}

// allFlavors returns a list of all existing flavors.
func (n *Nova) allFlavors() []nova.FlavorDetail {
	var flavors []nova.FlavorDetail
	for _, flavor := range n.flavors {
		flavors = append(flavors, flavor)
	}
	return flavors
}

// allFlavorsAsEntities returns all flavors as Entity structs.
func (n *Nova) allFlavorsAsEntities() []nova.Entity {
	var entities []nova.Entity
	for _, flavor := range n.flavors {
		entities = append(entities, nova.Entity{
			Id:    flavor.Id,
			Name:  flavor.Name,
			Links: flavor.Links,
		})
	}
	return entities
}

// removeFlavor deletes an existing flavor.
func (n *Nova) removeFlavor(flavorId string) error {
	if _, err := n.getFlavor(flavorId); err != nil {
		return err
	}
	delete(n.flavors, flavorId)
	return nil
}

// buildServerLinks populates the Links field of the passed
// ServerDetail as needed by OpenStack HTTP API. Call this
// before addServer().
func (n *Nova) buildServerLinks(server *nova.ServerDetail) {
	ep := n.hostname
	ver := strings.TrimLeft(n.baseURL, "/")
	url := n.token + "/servers/" + server.Id
	server.Links = []nova.Link{
		nova.Link{Href: ep + ver + url, Rel: "self"},
		nova.Link{Href: ep + url, Rel: "bookmark"},
	}
}

// addServer creates a new server.
func (n *Nova) addServer(server nova.ServerDetail) error {
	if _, err := n.getServer(server.Id); err == nil {
		return fmt.Errorf("a server with id %q already exists", server.Id)
	}
	n.servers[server.Id] = server
	return nil
}

// getServer retrieves an existing server by ID.
func (n *Nova) getServer(serverId string) (*nova.ServerDetail, error) {
	server, ok := n.servers[serverId]
	if !ok {
		return nil, fmt.Errorf("no such server %q", serverId)
	}
	return &server, nil
}

// getServerAsEntity returns the stored ServerDetail as Entity.
func (n *Nova) getServerAsEntity(serverId string) (*nova.Entity, error) {
	server, err := n.getServer(serverId)
	if err != nil {
		return nil, err
	}
	return &nova.Entity{
		Id:    server.Id,
		Name:  server.Name,
		Links: server.Links,
	}, nil
}

// allServers returns a list of all existing servers.
func (n *Nova) allServers() []nova.ServerDetail {
	var servers []nova.ServerDetail
	for _, server := range n.servers {
		servers = append(servers, server)
	}
	return servers
}

// allServersAsEntities returns all servers as Entity structs.
func (n *Nova) allServersAsEntities() []nova.Entity {
	var entities []nova.Entity
	for _, server := range n.servers {
		entities = append(entities, nova.Entity{
			Id:    server.Id,
			Name:  server.Name,
			Links: server.Links,
		})
	}
	return entities
}

// removeServer deletes an existing server.
func (n *Nova) removeServer(serverId string) error {
	if _, err := n.getServer(serverId); err != nil {
		return err
	}
	delete(n.servers, serverId)
	return nil
}

// addSecurityGroup creates a new security group.
func (n *Nova) addSecurityGroup(group nova.SecurityGroup) error {
	if _, err := n.getSecurityGroup(group.Id); err == nil {
		return fmt.Errorf("a security group with id %d already exists", group.Id)
	}
	n.groups[group.Id] = group
	return nil
}

// getSecurityGroup retrieves an existing group by ID.
func (n *Nova) getSecurityGroup(groupId int) (*nova.SecurityGroup, error) {
	group, ok := n.groups[groupId]
	if !ok {
		return nil, fmt.Errorf("no such security group %d", groupId)
	}
	return &group, nil
}

// allSecurityGroups returns a list of all existing groups.
func (n *Nova) allSecurityGroups() []nova.SecurityGroup {
	var groups []nova.SecurityGroup
	for _, group := range n.groups {
		groups = append(groups, group)
	}
	return groups
}

// removeSecurityGroup deletes an existing group.
func (n *Nova) removeSecurityGroup(groupId int) error {
	if _, err := n.getSecurityGroup(groupId); err != nil {
		return err
	}
	delete(n.groups, groupId)
	return nil
}

// addSecurityGroupRule creates a new rule in an existing group.
// This can be either an ingress or a group rule (see the notes
// about nova.RuleInfo).
func (n *Nova) addSecurityGroupRule(ruleId int, rule nova.RuleInfo) error {
	if _, err := n.getSecurityGroupRule(ruleId); err == nil {
		return fmt.Errorf("a security group rule with id %d already exists", ruleId)
	}
	group, err := n.getSecurityGroup(rule.ParentGroupId)
	if err != nil {
		return err
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
		sourceGroup, err := n.getSecurityGroup(*rule.GroupId)
		if err != nil {
			return fmt.Errorf("unknown source security group %d", *rule.GroupId)
		}
		newrule.Group = &nova.SecurityGroupRef{
			TenantId: sourceGroup.TenantId,
			Name:     sourceGroup.Name,
		}
	}
	if rule.FromPort > 0 {
		newrule.FromPort = &rule.FromPort
	}
	if rule.ToPort > 0 {
		newrule.ToPort = &rule.ToPort
	}
	if rule.IPProtocol != "" {
		newrule.IPProtocol = &rule.IPProtocol
	}
	if rule.Cidr != "" {
		newrule.IPRange = make(map[string]string)
		newrule.IPRange["cidr"] = rule.Cidr
	}

	group.Rules = append(group.Rules, newrule)
	n.groups[group.Id] = *group
	n.rules[newrule.Id] = newrule
	return nil
}

// hasSecurityGroupRule returns whether the given group contains the given rule,
// or (when groupId=-1) whether the given rule exists.
func (n *Nova) hasSecurityGroupRule(groupId, ruleId int) bool {
	rule, ok := n.rules[ruleId]
	_, err := n.getSecurityGroup(groupId)
	return ok && (groupId == -1 || (err == nil && rule.ParentGroupId == groupId))
}

// getSecurityGroupRule retrieves an existing rule by ID.
func (n *Nova) getSecurityGroupRule(ruleId int) (*nova.SecurityGroupRule, error) {
	rule, ok := n.rules[ruleId]
	if !ok {
		return nil, fmt.Errorf("no such security group rule %d", ruleId)
	}
	return &rule, nil
}

// removeSecurityGroupRule deletes an existing rule from its group.
func (n *Nova) removeSecurityGroupRule(ruleId int) error {
	rule, err := n.getSecurityGroupRule(ruleId)
	if err != nil {
		return err
	}
	if group, err := n.getSecurityGroup(rule.ParentGroupId); err == nil {
		idx := -1
		for ri, ru := range group.Rules {
			if ru.Id == ruleId {
				idx = ri
				break
			}
		}
		if idx != -1 {
			group.Rules = append(group.Rules[:idx], group.Rules[idx+1:]...)
			n.groups[group.Id] = *group
		}
		// Silently ignore missing rules...
	}
	// ...or groups
	delete(n.rules, ruleId)
	return nil
}

// addServerSecurityGroup attaches an existing server to a group.
func (n *Nova) addServerSecurityGroup(serverId string, groupId int) error {
	if _, err := n.getServer(serverId); err != nil {
		return err
	}
	if _, err := n.getSecurityGroup(groupId); err != nil {
		return err
	}
	groups, ok := n.serverGroups[serverId]
	if ok {
		for _, gid := range groups {
			if gid == groupId {
				return fmt.Errorf("server %q already belongs to group %d", serverId, groupId)
			}
		}
	}
	groups = append(groups, groupId)
	n.serverGroups[serverId] = groups
	return nil
}

// hasServerSecurityGroup returns whether the given server belongs to the group.
func (n *Nova) hasServerSecurityGroup(serverId string, groupId int) bool {
	if _, err := n.getServer(serverId); err != nil {
		return false
	}
	if _, err := n.getSecurityGroup(groupId); err != nil {
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

// removeServerSecurityGroup detaches an existing server from a group.
func (n *Nova) removeServerSecurityGroup(serverId string, groupId int) error {
	if _, err := n.getServer(serverId); err != nil {
		return err
	}
	if _, err := n.getSecurityGroup(groupId); err != nil {
		return err
	}
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
	groups = append(groups[:idx], groups[idx+1:]...)
	n.serverGroups[serverId] = groups
	return nil
}

// addFloatingIP creates a new floating IP address in the pool.
func (n *Nova) addFloatingIP(ip nova.FloatingIP) error {
	if _, err := n.getFloatingIP(ip.Id); err == nil {
		return fmt.Errorf("a floating IP with id %d already exists", ip.Id)
	}
	n.floatingIPs[ip.Id] = ip
	return nil
}

// hasFloatingIP returns whether the given floating IP address exists.
func (n *Nova) hasFloatingIP(address string) bool {
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

// getFloatingIP retrieves the floating IP by ID.
func (n *Nova) getFloatingIP(ipId int) (*nova.FloatingIP, error) {
	ip, ok := n.floatingIPs[ipId]
	if !ok {
		return nil, fmt.Errorf("no such floating IP %d", ipId)
	}
	return &ip, nil
}

// allFloatingIPs returns a list of all created floating IPs.
func (n *Nova) allFloatingIPs() []nova.FloatingIP {
	var fips []nova.FloatingIP
	for _, fip := range n.floatingIPs {
		fips = append(fips, fip)
	}
	return fips
}

// removeFloatingIP deletes an existing floating IP by ID.
func (n *Nova) removeFloatingIP(ipId int) error {
	if _, err := n.getFloatingIP(ipId); err != nil {
		return err
	}
	delete(n.floatingIPs, ipId)
	return nil
}

// addServerFloatingIP attaches an existing floating IP to a server.
func (n *Nova) addServerFloatingIP(serverId string, ipId int) error {
	if _, err := n.getServer(serverId); err != nil {
		return err
	}
	if _, err := n.getFloatingIP(ipId); err != nil {
		return err
	}
	fips, ok := n.serverIPs[serverId]
	if ok {
		for _, fipId := range fips {
			if fipId == ipId {
				return fmt.Errorf("server %q already has floating IP %d", serverId, ipId)
			}
		}
	}
	fips = append(fips, ipId)
	n.serverIPs[serverId] = fips
	return nil
}

// hasServerFloatingIP verifies the given floating IP belongs to a server.
func (n *Nova) hasServerFloatingIP(serverId, address string) bool {
	if _, err := n.getServer(serverId); err != nil || !n.hasFloatingIP(address) {
		return false
	}
	fips, ok := n.serverIPs[serverId]
	if !ok {
		return false
	}
	for _, fipId := range fips {
		fip := n.floatingIPs[fipId]
		if fip.IP == address {
			return true
		}
	}
	return false
}

// removeServerFloatingIP deletes an attached floating IP from a server.
func (n *Nova) removeServerFloatingIP(serverId string, ipId int) error {
	if _, err := n.getServer(serverId); err != nil {
		return err
	}
	if _, err := n.getFloatingIP(ipId); err != nil {
		return err
	}
	fips, ok := n.serverIPs[serverId]
	if !ok {
		return fmt.Errorf("server %q does not have any floating IPs to remove", serverId)
	}
	idx := -1
	for fi, fipId := range fips {
		if fipId == ipId {
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
