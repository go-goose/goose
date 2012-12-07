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

// AddFlavor creates a new flavor.
func (n *Nova) AddFlavor(flavor nova.FlavorDetail) error {
	if _, ok := n.flavors[flavor.Id]; ok {
		return fmt.Errorf("a flavor with id %q already exists", flavor.Id)
	}
	// build the links, if not given
	if flavor.Links == nil {
		ep := n.hostname
		ver := strings.TrimLeft(n.baseURL, "/")
		url := n.token + "/flavors/" + flavor.Id
		flavor.Links = []nova.Link{
			nova.Link{Href: ep + ver + url, Rel: "self"},
			nova.Link{Href: ep + url, Rel: "bookmark"},
		}
	}
	n.flavors[flavor.Id] = flavor
	return nil
}

// GetFlavor retrieves an existing flavor by ID.
func (n *Nova) GetFlavor(flavorId string) (nova.FlavorDetail, error) {
	flavor, ok := n.flavors[flavorId]
	if !ok {
		return flavor, fmt.Errorf("no such flavor %q", flavorId)
	}
	return flavor, nil
}

// GetFlavorAsEntity returns the stored FlavorDetail as Entity.
func (n *Nova) GetFlavorAsEntity(flavorId string) (nova.Entity, error) {
	flavor, err := n.GetFlavor(flavorId)
	if err != nil {
		return nova.Entity{}, err
	}
	return nova.Entity{
		Id:    flavor.Id,
		Name:  flavor.Name,
		Links: flavor.Links,
	}, nil
}

// AllFlavors returns a list of all existing flavors.
func (n *Nova) AllFlavors() ([]nova.FlavorDetail, error) {
	if len(n.flavors) == 0 {
		return nil, fmt.Errorf("no flavors to return")
	}
	flavors := []nova.FlavorDetail{}
	for _, flavor := range n.flavors {
		flavors = append(flavors, flavor)
	}
	return flavors, nil
}

// AllFlavorsAsEntities returns all flavors as Entity structs.
func (n *Nova) AllFlavorsAsEntities() ([]nova.Entity, error) {
	if len(n.flavors) == 0 {
		return nil, fmt.Errorf("no flavors to return")
	}
	entities := []nova.Entity{}
	for _, flavor := range n.flavors {
		entities = append(entities, nova.Entity{
			Id:    flavor.Id,
			Name:  flavor.Name,
			Links: flavor.Links,
		})
	}
	return entities, nil
}

// RemoveFlavor deletes an existing flavor.
func (n *Nova) RemoveFlavor(flavorId string) error {
	if _, ok := n.flavors[flavorId]; !ok {
		return fmt.Errorf("no such flavor %q", flavorId)
	}
	delete(n.flavors, flavorId)
	return nil
}

// AddServer creates a new server.
func (n *Nova) AddServer(server nova.ServerDetail) error {
	if _, ok := n.servers[server.Id]; ok {
		return fmt.Errorf("a server with id %q already exists", server.Id)
	}
	// build the links, if not given
	if server.Links == nil {
		ep := n.hostname
		ver := strings.TrimLeft(n.baseURL, "/")
		url := n.token + "/servers/" + server.Id
		server.Links = []nova.Link{
			nova.Link{Href: ep + ver + url, Rel: "self"},
			nova.Link{Href: ep + url, Rel: "bookmark"},
		}
	}
	n.servers[server.Id] = server
	return nil
}

// GetServer retrieves an existing server by ID.
func (n *Nova) GetServer(serverId string) (nova.ServerDetail, error) {
	server, ok := n.servers[serverId]
	if !ok {
		return nova.ServerDetail{}, fmt.Errorf("no such server %q", serverId)
	}
	return server, nil
}

// GetServerAsEntity returns the stored ServerDetail as Entity.
func (n *Nova) GetServerAsEntity(serverId string) (nova.Entity, error) {
	server, err := n.GetServer(serverId)
	if err != nil {
		return nova.Entity{}, err
	}
	return nova.Entity{
		Id:    server.Id,
		Name:  server.Name,
		Links: server.Links,
	}, nil
}

// AllServers returns a list of all existing servers.
func (n *Nova) AllServers() ([]nova.ServerDetail, error) {
	if len(n.servers) == 0 {
		return nil, fmt.Errorf("no servers to return")
	}
	servers := []nova.ServerDetail{}
	for _, server := range n.servers {
		servers = append(servers, server)
	}
	return servers, nil
}

// AllServersAsEntities returns all servers as Entity structs.
func (n *Nova) AllServersAsEntities() ([]nova.Entity, error) {
	if len(n.servers) == 0 {
		return nil, fmt.Errorf("no servers to return")
	}
	entities := []nova.Entity{}
	for _, server := range n.servers {
		entities = append(entities, nova.Entity{
			Id:    server.Id,
			Name:  server.Name,
			Links: server.Links,
		})
	}
	return entities, nil
}

// RemoveServer deletes an existing server.
func (n *Nova) RemoveServer(serverId string) error {
	if _, ok := n.servers[serverId]; !ok {
		return fmt.Errorf("no such server %q", serverId)
	}
	delete(n.servers, serverId)
	return nil
}

// AddSecurityGroup creates a new security group.
func (n *Nova) AddSecurityGroup(group nova.SecurityGroup) error {
	if _, ok := n.groups[group.Id]; ok {
		return fmt.Errorf("a security group with id %d already exists", group.Id)
	}
	n.groups[group.Id] = group
	return nil
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
	groups := []nova.SecurityGroup{}
	for _, group := range n.groups {
		groups = append(groups, group)
	}
	return groups, nil
}

// RemoveSecurityGroup deletes an existing group.
func (n *Nova) RemoveSecurityGroup(groupId int) error {
	if _, ok := n.groups[groupId]; !ok {
		return fmt.Errorf("no such security group %d", groupId)
	}
	delete(n.groups, groupId)
	return nil
}

// AddSecurityGroupRule creates a new rule in an existing group.
// This can be either an ingress or a group rule (see the notes
// about nova.RuleInfo).
func (n *Nova) AddSecurityGroupRule(ruleId int, rule nova.RuleInfo) error {
	if _, ok := n.rules[ruleId]; ok {
		return fmt.Errorf("a security group rule with id %d already exists", ruleId)
	}
	group, ok := n.groups[rule.ParentGroupId]
	if !ok {
		return fmt.Errorf("cannot add a rule to unknown security group %d", rule.ParentGroupId)
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
		sourceGroup, ok := n.groups[*rule.GroupId]
		if !ok {
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
	n.groups[group.Id] = group
	n.rules[newrule.Id] = newrule
	return nil
}

// HasSecurityGroupRule returns whether the given group contains the given rule,
// or (when groupId=-1) whether the given rule exists.
func (n *Nova) HasSecurityGroupRule(groupId, ruleId int) bool {
	rule, ok := n.rules[ruleId]
	_, err := n.GetSecurityGroup(groupId)
	return ok && (groupId == -1 || (err == nil && rule.ParentGroupId == groupId))
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
	if group, ok := n.groups[rule.ParentGroupId]; ok {
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
	if _, err := n.GetServer(serverId); err != nil {
		return fmt.Errorf("no such server %q", serverId)
	}
	if _, err := n.GetSecurityGroup(groupId); err != nil {
		return fmt.Errorf("no such security group %d", groupId)
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

// HasServerSecurityGroup returns whether the given server belongs to the group.
func (n *Nova) HasServerSecurityGroup(serverId string, groupId int) bool {
	if _, err := n.GetServer(serverId); err != nil {
		return false
	}
	if _, err := n.GetSecurityGroup(groupId); err != nil {
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
	if _, err := n.GetServer(serverId); err != nil {
		return fmt.Errorf("no such server %q", serverId)
	}
	if _, err := n.GetSecurityGroup(groupId); err != nil {
		return fmt.Errorf("no such security group %d", groupId)
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

// AddFloatingIP creates a new floating IP address in the pool.
func (n *Nova) AddFloatingIP(ip nova.FloatingIP) error {
	if _, ok := n.floatingIPs[ip.Id]; ok {
		return fmt.Errorf("a floating IP with id %d already exists", ip.Id)
	}
	n.floatingIPs[ip.Id] = ip
	return nil
}

// HasFloatingIP returns whether the given floating IP address exists.
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
	ip, ok := n.floatingIPs[ipId]
	if !ok {
		return nova.FloatingIP{}, fmt.Errorf("no such floating IP %d", ipId)
	}
	return ip, nil
}

// AllFloatingIPs returns a list of all created floating IPs.
func (n *Nova) AllFloatingIPs() ([]nova.FloatingIP, error) {
	if len(n.floatingIPs) == 0 {
		return nil, fmt.Errorf("no floating IPs to return")
	}
	fips := []nova.FloatingIP{}
	for _, fip := range n.floatingIPs {
		fips = append(fips, fip)
	}
	return fips, nil
}

// RemoveFloatingIP deletes an existing floating IP by ID.
func (n *Nova) RemoveFloatingIP(ipId int) error {
	if _, ok := n.floatingIPs[ipId]; !ok {
		return fmt.Errorf("no such floating IP %d", ipId)
	}
	delete(n.floatingIPs, ipId)
	return nil
}

// AddServerFloatingIP attaches an existing floating IP to a server.
func (n *Nova) AddServerFloatingIP(serverId string, ipId int) error {
	if _, err := n.GetServer(serverId); err != nil {
		return fmt.Errorf("no such server %q", serverId)
	}
	if _, err := n.GetFloatingIP(ipId); err != nil {
		return fmt.Errorf("no such floating IP %d", ipId)
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

// HasServerFloatingIP verifies the given floating IP belongs to a server.
func (n *Nova) HasServerFloatingIP(serverId, address string) bool {
	if _, err := n.GetServer(serverId); err != nil || !n.HasFloatingIP(address) {
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

// RemoveServerFloatingIP deletes an attached floating IP from a server.
func (n *Nova) RemoveServerFloatingIP(serverId string, ipId int) error {
	if _, err := n.GetServer(serverId); err != nil {
		return fmt.Errorf("no such server %q", serverId)
	}
	if _, err := n.GetFloatingIP(ipId); err != nil {
		return fmt.Errorf("no such floating IP %d", ipId)
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
