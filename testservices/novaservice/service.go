// Nova double testing service - internal direct API implementation

package novaservice

import (
	"fmt"
	"launchpad.net/goose/nova"
	"launchpad.net/goose/testservices"
	"launchpad.net/goose/testservices/identityservice"
	"net/url"
	"strings"
)

var _ testservices.HttpService = (*Nova)(nil)
var _ identityservice.ServiceProvider = (*Nova)(nil)

// Nova implements a OpenStack Nova testing service and
// contains the service double's internal state.
type Nova struct {
	testservices.ServiceInstance
	flavors                   map[string]nova.FlavorDetail
	servers                   map[string]nova.ServerDetail
	groups                    map[int]nova.SecurityGroup
	rules                     map[int]nova.SecurityGroupRule
	floatingIPs               map[int]nova.FloatingIP
	serverGroups              map[string][]int
	serverIPs                 map[string][]int
	nextGroupId               int
	nextRuleId                int
	nextIPId                  int
	sendFakeRateLimitResponse bool
}

// endpoint returns either a versioned or non-versioned service
// endpoint URL from the given path.
func (n *Nova) endpointURL(version bool, path string) string {
	ep := "http://" + n.Hostname
	if version {
		ep += n.VersionPath + "/"
	}
	ep += n.TenantId
	if path != "" {
		ep += "/" + strings.TrimLeft(path, "/")
	}
	return ep
}

func (n *Nova) Endpoints() []identityservice.Endpoint {
	ep := identityservice.Endpoint{
		AdminURL:    n.endpointURL(true, ""),
		InternalURL: n.endpointURL(true, ""),
		PublicURL:   n.endpointURL(true, ""),
		Region:      n.Region,
	}
	return []identityservice.Endpoint{ep}
}

// New creates an instance of the Nova object, given the parameters.
func New(hostURL, versionPath, tenantId, region string, identityService identityservice.IdentityService) *Nova {
	url, err := url.Parse(hostURL)
	if err != nil {
		panic(err)
	}
	hostname := url.Host
	if !strings.HasSuffix(hostname, "/") {
		hostname += "/"
	}
	// Real openstack instances have flavours "out of the box". So we add some here.
	defaultFlavors := []nova.FlavorDetail{
		{Id: "1", Name: "m1.tiny"},
		{Id: "2", Name: "m1.small"},
	}
	// Real openstack instances have a default security group "out of the box". So we add it here.
	defaultSecurityGroups := []nova.SecurityGroup{
		{Id: 999, Name: "default", Description: "default group"},
	}
	nova := &Nova{
		flavors:      make(map[string]nova.FlavorDetail),
		servers:      make(map[string]nova.ServerDetail),
		groups:       make(map[int]nova.SecurityGroup),
		rules:        make(map[int]nova.SecurityGroupRule),
		floatingIPs:  make(map[int]nova.FloatingIP),
		serverGroups: make(map[string][]int),
		serverIPs:    make(map[string][]int),
		// The following attribute controls whether rate limit responses are sent back to the caller.
		// This is switched off when we want to ensure the client eventually gets a proper response.
		sendFakeRateLimitResponse: true,
		ServiceInstance: testservices.ServiceInstance{
			IdentityService: identityService,
			Hostname:        hostname,
			VersionPath:     versionPath,
			TenantId:        tenantId,
			Region:          region,
		},
	}
	if identityService != nil {
		identityService.RegisterServiceProvider("nova", "compute", nova)
	}
	for i, flavor := range defaultFlavors {
		nova.buildFlavorLinks(&flavor)
		defaultFlavors[i] = flavor
		err := nova.addFlavor(flavor)
		if err != nil {
			panic(err)
		}
	}
	for _, group := range defaultSecurityGroups {
		err := nova.addSecurityGroup(group)
		if err != nil {
			panic(err)
		}
	}
	return nova
}

// buildFlavorLinks populates the Links field of the passed
// FlavorDetail as needed by OpenStack HTTP API. Call this
// before addFlavor().
func (n *Nova) buildFlavorLinks(flavor *nova.FlavorDetail) {
	url := "/flavors/" + flavor.Id
	flavor.Links = []nova.Link{
		nova.Link{Href: n.endpointURL(true, url), Rel: "self"},
		nova.Link{Href: n.endpointURL(false, url), Rel: "bookmark"},
	}
}

// addFlavor creates a new flavor.
func (n *Nova) addFlavor(flavor nova.FlavorDetail) error {
	if _, err := n.flavor(flavor.Id); err == nil {
		return fmt.Errorf("a flavor with id %q already exists", flavor.Id)
	}
	n.flavors[flavor.Id] = flavor
	return nil
}

// flavor retrieves an existing flavor by ID.
func (n *Nova) flavor(flavorId string) (*nova.FlavorDetail, error) {
	flavor, ok := n.flavors[flavorId]
	if !ok {
		return nil, fmt.Errorf("no such flavor %q", flavorId)
	}
	return &flavor, nil
}

// flavorAsEntity returns the stored FlavorDetail as Entity.
func (n *Nova) flavorAsEntity(flavorId string) (*nova.Entity, error) {
	flavor, err := n.flavor(flavorId)
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
	if _, err := n.flavor(flavorId); err != nil {
		return err
	}
	delete(n.flavors, flavorId)
	return nil
}

// buildServerLinks populates the Links field of the passed
// ServerDetail as needed by OpenStack HTTP API. Call this
// before addServer().
func (n *Nova) buildServerLinks(server *nova.ServerDetail) {
	url := "/servers/" + server.Id
	server.Links = []nova.Link{
		nova.Link{Href: n.endpointURL(true, url), Rel: "self"},
		nova.Link{Href: n.endpointURL(false, url), Rel: "bookmark"},
	}
}

// addServer creates a new server.
func (n *Nova) addServer(server nova.ServerDetail) error {
	if _, err := n.server(server.Id); err == nil {
		return fmt.Errorf("a server with id %q already exists", server.Id)
	}
	n.servers[server.Id] = server
	return nil
}

// server retrieves an existing server by ID.
func (n *Nova) server(serverId string) (*nova.ServerDetail, error) {
	server, ok := n.servers[serverId]
	if !ok {
		return nil, fmt.Errorf("no such server %q", serverId)
	}
	return &server, nil
}

// serverByName retrieves the first existing server with the given name.
func (n *Nova) serverByName(name string) (*nova.ServerDetail, error) {
	for _, server := range n.servers {
		if server.Name == name {
			return &server, nil
		}
	}
	return nil, fmt.Errorf("no such server named %q", name)
}

// serverAsEntity returns the stored ServerDetail as Entity.
func (n *Nova) serverAsEntity(serverId string) (*nova.Entity, error) {
	server, err := n.server(serverId)
	if err != nil {
		return nil, err
	}
	return &nova.Entity{
		Id:    server.Id,
		Name:  server.Name,
		Links: server.Links,
	}, nil
}

// matchServer returns true if the given server is matched by the
// given filter, or false otherwise.
func (n *Nova) matchServer(filter *nova.Filter, server nova.ServerDetail) bool {
	if filter == nil {
		return true // empty filter matches everything
	}
	values := filter.Values
	for _, val := range values[nova.FilterStatus] {
		if server.Status == val {
			return true
		}
	}
	for _, val := range values[nova.FilterServer] {
		if server.Name == val {
			return true
		}
	}
	// TODO(dimitern) maybe implement FilterFlavor, FilterImage,
	// FilterMarker, FilterLimit and FilterChangesSince
	return false
}

// allServers returns a list of all existing servers.
// Filtering is supported, see nova.Filter* for more info.
func (n *Nova) allServers(filter *nova.Filter) []nova.ServerDetail {
	var servers []nova.ServerDetail
	for _, server := range n.servers {
		if n.matchServer(filter, server) {
			servers = append(servers, server)
		}
	}
	return servers
}

// allServersAsEntities returns all servers as Entity structs.
// Filtering is supported, see nova.Filter* for more info.
func (n *Nova) allServersAsEntities(filter *nova.Filter) []nova.Entity {
	var entities []nova.Entity
	for _, server := range n.servers {
		if n.matchServer(filter, server) {
			entities = append(entities, nova.Entity{
				Id:    server.Id,
				Name:  server.Name,
				Links: server.Links,
			})
		}
	}
	return entities
}

// removeServer deletes an existing server.
func (n *Nova) removeServer(serverId string) error {
	if _, err := n.server(serverId); err != nil {
		return err
	}
	delete(n.servers, serverId)
	return nil
}

// addSecurityGroup creates a new security group.
func (n *Nova) addSecurityGroup(group nova.SecurityGroup) error {
	if _, err := n.securityGroup(group.Id); err == nil {
		return fmt.Errorf("a security group with id %d already exists", group.Id)
	}
	group.TenantId = n.TenantId
	if group.Rules == nil {
		group.Rules = []nova.SecurityGroupRule{}
	}
	n.groups[group.Id] = group
	return nil
}

// securityGroup retrieves an existing group by ID.
func (n *Nova) securityGroup(groupId int) (*nova.SecurityGroup, error) {
	group, ok := n.groups[groupId]
	if !ok {
		return nil, fmt.Errorf("no such security group %d", groupId)
	}
	return &group, nil
}

// securityGroupByName retrieves an existing named group.
func (n *Nova) securityGroupByName(groupName string) (*nova.SecurityGroup, error) {
	for _, group := range n.groups {
		if group.Name == groupName {
			return &group, nil
		}
	}
	return nil, fmt.Errorf("no such security group named %q", groupName)
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
	if _, err := n.securityGroup(groupId); err != nil {
		return err
	}
	delete(n.groups, groupId)
	return nil
}

// addSecurityGroupRule creates a new rule in an existing group.
// This can be either an ingress or a group rule (see the notes
// about nova.RuleInfo).
func (n *Nova) addSecurityGroupRule(ruleId int, rule nova.RuleInfo) error {
	if _, err := n.securityGroupRule(ruleId); err == nil {
		return fmt.Errorf("a security group rule with id %d already exists", ruleId)
	}
	group, err := n.securityGroup(rule.ParentGroupId)
	if err != nil {
		return err
	}
	for _, ru := range group.Rules {
		if ru.Id == ruleId {
			return fmt.Errorf("cannot add twice rule %d to security group %d", ru.Id, group.Id)
		}
	}
	var zeroSecurityGroupRef nova.SecurityGroupRef
	newrule := nova.SecurityGroupRule{
		ParentGroupId: rule.ParentGroupId,
		Id:            ruleId,
		Group:         zeroSecurityGroupRef,
	}
	if rule.GroupId != nil {
		sourceGroup, err := n.securityGroup(*rule.GroupId)
		if err != nil {
			return fmt.Errorf("unknown source security group %d", *rule.GroupId)
		}
		newrule.Group = nova.SecurityGroupRef{
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
	_, err := n.securityGroup(groupId)
	return ok && (groupId == -1 || (err == nil && rule.ParentGroupId == groupId))
}

// securityGroupRule retrieves an existing rule by ID.
func (n *Nova) securityGroupRule(ruleId int) (*nova.SecurityGroupRule, error) {
	rule, ok := n.rules[ruleId]
	if !ok {
		return nil, fmt.Errorf("no such security group rule %d", ruleId)
	}
	return &rule, nil
}

// removeSecurityGroupRule deletes an existing rule from its group.
func (n *Nova) removeSecurityGroupRule(ruleId int) error {
	rule, err := n.securityGroupRule(ruleId)
	if err != nil {
		return err
	}
	if group, err := n.securityGroup(rule.ParentGroupId); err == nil {
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
	if _, err := n.server(serverId); err != nil {
		return err
	}
	if _, err := n.securityGroup(groupId); err != nil {
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
	if _, err := n.server(serverId); err != nil {
		return false
	}
	if _, err := n.securityGroup(groupId); err != nil {
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

// allServerSecurityGroups returns all security groups attached to the
// given server.
func (n *Nova) allServerSecurityGroups(serverId string) []nova.SecurityGroup {
	var groups []nova.SecurityGroup
	for _, gid := range n.serverGroups[serverId] {
		group, err := n.securityGroup(gid)
		if err != nil {
			return nil
		}
		groups = append(groups, *group)
	}
	return groups
}

// removeServerSecurityGroup detaches an existing server from a group.
func (n *Nova) removeServerSecurityGroup(serverId string, groupId int) error {
	if _, err := n.server(serverId); err != nil {
		return err
	}
	if _, err := n.securityGroup(groupId); err != nil {
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
	if _, err := n.floatingIP(ip.Id); err == nil {
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

// floatingIP retrieves the floating IP by ID.
func (n *Nova) floatingIP(ipId int) (*nova.FloatingIP, error) {
	ip, ok := n.floatingIPs[ipId]
	if !ok {
		return nil, fmt.Errorf("no such floating IP %d", ipId)
	}
	return &ip, nil
}

// floatingIPByAddr retrieves the floating IP by address.
func (n *Nova) floatingIPByAddr(address string) (*nova.FloatingIP, error) {
	for _, fip := range n.floatingIPs {
		if fip.IP == address {
			return &fip, nil
		}
	}
	return nil, fmt.Errorf("no such floating IP with address %q", address)
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
	if _, err := n.floatingIP(ipId); err != nil {
		return err
	}
	delete(n.floatingIPs, ipId)
	return nil
}

// addServerFloatingIP attaches an existing floating IP to a server.
func (n *Nova) addServerFloatingIP(serverId string, ipId int) error {
	if _, err := n.server(serverId); err != nil {
		return err
	}
	if fip, err := n.floatingIP(ipId); err != nil {
		return err
	} else {
		fip.FixedIP = fip.IP
		fip.InstanceId = serverId
		n.floatingIPs[ipId] = *fip
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
	if _, err := n.server(serverId); err != nil || !n.hasFloatingIP(address) {
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
	if _, err := n.server(serverId); err != nil {
		return err
	}
	if fip, err := n.floatingIP(ipId); err != nil {
		return err
	} else {
		fip.FixedIP = nil
		fip.InstanceId = nil
		n.floatingIPs[ipId] = *fip
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
