// Nova double testing service - internal direct API implementation

package novaservice

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/go-goose/goose/v5/errors"
	"github.com/go-goose/goose/v5/nova"
	"github.com/go-goose/goose/v5/testservices"
	"github.com/go-goose/goose/v5/testservices/identityservice"
	"github.com/go-goose/goose/v5/testservices/neutronmodel"
)

var _ testservices.HttpService = (*Nova)(nil)
var _ identityservice.ServiceProvider = (*Nova)(nil)

// Nova implements a OpenStack Nova testing service and
// contains the service double's internal state.
type Nova struct {
	testservices.ServiceInstance
	neutronModel              *neutronmodel.NeutronModel
	flavors                   map[string]nova.FlavorDetail
	servers                   map[string]nova.ServerDetail
	groups                    map[string]nova.SecurityGroup
	rules                     map[string]nova.SecurityGroupRule
	floatingIPs               map[string]nova.FloatingIP
	networks                  map[string]nova.Network
	serverGroups              map[string][]string
	serverIPs                 map[string][]string
	availabilityZones         map[string]nova.AvailabilityZone
	serverIdToOSInterfaces    map[string][]nova.OSInterface
	serverIdToAttachedVolumes map[string][]nova.VolumeAttachment
	nextServerId              int
	nextGroupId               int
	nextRuleId                int
	nextIPId                  int
	nextAttachmentId          int
	nextOSInterfaceId         int
	useNeutronNetworking      bool
	noValidHostZone           nova.AvailabilityZone
	serverStatus              string
}

func errorJSONEncode(err error) (int, string) {
	serverError, ok := err.(*testservices.ServerError)
	if !ok {
		serverError = testservices.NewInternalServerError(err.Error())
	}
	return serverError.Code(), serverError.AsJSON()
}

// endpoint returns either a versioned or non-versioned service
// endpoint URL from the given path.
func (n *Nova) endpointURL(version bool, path string) string {
	ep := n.Scheme + "://" + n.Hostname
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

func (n *Nova) V3Endpoints() []identityservice.V3Endpoint {
	url := n.endpointURL(true, "")
	return identityservice.NewV3Endpoints(url, url, url, n.RegionID)
}

// New creates an instance of the Nova object, given the parameters.
func New(hostURL, versionPath, tenantId, region string, identityService, fallbackIdentity identityservice.IdentityService) *Nova {
	URL, err := url.Parse(hostURL)
	if err != nil {
		panic(err)
	}
	hostname := URL.Host
	if !strings.HasSuffix(hostname, "/") {
		hostname += "/"
	}
	// Real openstack instances have flavours "out of the box". So we add some here.
	defaultFlavors := []nova.FlavorDetail{
		{Id: "1", Name: "m1.tiny", RAM: 512, VCPUs: 1, Disk: 5},
		{Id: "2", Name: "m1.small", RAM: 2048, VCPUs: 1, Disk: 10},
		{Id: "3", Name: "m1.medium", RAM: 4096, VCPUs: 2, Disk: 15},
	}
	// Real openstack instances have a default security group "out of the box". So we add it here.
	defaultSecurityGroups := []nova.SecurityGroup{
		{Id: "999", Name: "default", Description: "default group"},
	}
	novaService := &Nova{
		flavors:                   make(map[string]nova.FlavorDetail),
		servers:                   make(map[string]nova.ServerDetail),
		groups:                    make(map[string]nova.SecurityGroup),
		rules:                     make(map[string]nova.SecurityGroupRule),
		floatingIPs:               make(map[string]nova.FloatingIP),
		networks:                  make(map[string]nova.Network),
		serverGroups:              make(map[string][]string),
		serverIPs:                 make(map[string][]string),
		availabilityZones:         make(map[string]nova.AvailabilityZone),
		serverIdToOSInterfaces:    make(map[string][]nova.OSInterface),
		serverIdToAttachedVolumes: make(map[string][]nova.VolumeAttachment),
		useNeutronNetworking:      false,
		ServiceInstance: testservices.ServiceInstance{
			IdentityService:         identityService,
			FallbackIdentityService: fallbackIdentity,
			Scheme:                  URL.Scheme,
			Hostname:                hostname,
			VersionPath:             versionPath,
			TenantId:                tenantId,
			Region:                  region,
		},
	}
	if identityService != nil {
		identityService.RegisterServiceProvider("nova", "compute", novaService)
	}
	for i, flavor := range defaultFlavors {
		novaService.buildFlavorLinks(&flavor)
		defaultFlavors[i] = flavor
		err := novaService.addFlavor(flavor)
		if err != nil {
			panic(err)
		}
	}
	for _, group := range defaultSecurityGroups {
		err := novaService.addSecurityGroup(group)
		if err != nil {
			panic(err)
		}
	}
	// Add a sample default network
	var id = "1"
	novaService.networks[id] = nova.Network{
		Id:    id,
		Label: "net",
		Cidr:  "10.0.0.0/24",
	}
	return novaService
}

func (n *Nova) Stop() {
	// noop
}

// AddNeutronModel setups up the test double to use Neutron networking
// which requires shared data between the nova and neutron test doubles.
func (n *Nova) AddNeutronModel(neutronModel *neutronmodel.NeutronModel) {
	n.neutronModel = neutronModel
	n.useNeutronNetworking = true
}

// SetAvailabilityZones sets the availability zones for setting
// availability zones.
//
// Note: this is implemented as a public method rather than as
// an HTTP API for two reasons: availability zones are created
// indirectly via "host aggregates", which are a cloud-provider
// concept that we have not implemented, and because we want to
// be able to synthesize zone state changes.
func (n *Nova) SetAvailabilityZones(zones ...nova.AvailabilityZone) {
	n.availabilityZones = make(map[string]nova.AvailabilityZone)
	for _, z := range zones {
		n.availabilityZones[z.Name] = z
	}
}

// SetAZForNoValidHosts sets an availability zone to cause a
// No valid host failures.
//
// Note: this is implemented as a public method rather than as
// an HTTP API for the same reasons as SetAvailabilityZones, as
// well as defining an availability zone to cause 'No valid host'
// failures.
func (n *Nova) SetAZForNoValidHosts(zone nova.AvailabilityZone) {
	n.noValidHostZone = zone
	// ensure the zone for failure, is on the list of
	// possible zones
	if _, ok := n.availabilityZones[zone.Name]; !ok {
		n.availabilityZones[zone.Name] = zone
	}
}

// SetServerStatus sets the ServerDetail.Status to a new
// value.
//
// Note: this is implemented as a public method rather than as
// an HTTP API to allow for changing the status inside of the
// returned data structure, not accomplished by the testservice
// hooks
func (n *Nova) SetServerStatus(status string) {
	n.serverStatus = status
}

// buildFlavorLinks populates the Links field of the passed
// FlavorDetail as needed by OpenStack HTTP API. Call this
// before addFlavor().
func (n *Nova) buildFlavorLinks(flavor *nova.FlavorDetail) {
	url := "/flavors/" + flavor.Id
	flavor.Links = []nova.Link{
		{Href: n.endpointURL(true, url), Rel: "self"},
		{Href: n.endpointURL(false, url), Rel: "bookmark"},
	}
}

// addFlavor creates a new flavor.
func (n *Nova) addFlavor(flavor nova.FlavorDetail) error {
	if err := n.ProcessFunctionHook(n, flavor); err != nil {
		return err
	}
	if _, err := n.flavor(flavor.Id); err == nil {
		return testservices.NewAddFlavorError(flavor.Id)
	}
	n.flavors[flavor.Id] = flavor
	return nil
}

// flavor retrieves an existing flavor by ID.
func (n *Nova) flavor(flavorId string) (*nova.FlavorDetail, error) {
	if err := n.ProcessFunctionHook(n, flavorId); err != nil {
		return nil, err
	}
	flavor, ok := n.flavors[flavorId]
	if !ok {
		return nil, testservices.NewNoSuchFlavorError(flavorId)
	}
	return &flavor, nil
}

// flavorAsEntity returns the stored FlavorDetail as Entity.
func (n *Nova) flavorAsEntity(flavorId string) (*nova.Entity, error) {
	if err := n.ProcessFunctionHook(n, flavorId); err != nil {
		return nil, err
	}
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
	if err := n.ProcessFunctionHook(n, flavorId); err != nil {
		return err
	}
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
		{Href: n.endpointURL(true, url), Rel: "self"},
		{Href: n.endpointURL(false, url), Rel: "bookmark"},
	}
}

// addServer creates a new server.
func (n *Nova) addServer(server nova.ServerDetail) error {
	if err := n.ProcessFunctionHook(n, &server); err != nil {
		return err
	}
	if _, err := n.server(server.Id); err == nil {
		return testservices.NewServerAlreadyExistsError(server.Id)
	}
	n.servers[server.Id] = server
	return nil
}

// updateServerName creates a new server.
func (n *Nova) updateServerName(serverId, name string) error {
	if err := n.ProcessFunctionHook(n, serverId); err != nil {
		return err
	}
	server, err := n.server(serverId)
	if err != nil {
		return testservices.NewServerByIDNotFoundError(serverId)
	}
	server.Name = name
	n.servers[serverId] = *server
	return nil
}

// server retrieves an existing server by ID.
func (n *Nova) server(serverId string) (*nova.ServerDetail, error) {
	if err := n.ProcessFunctionHook(n, serverId); err != nil {
		return nil, err
	}
	server, ok := n.servers[serverId]
	if !ok {
		return nil, testservices.NewServerByIDNotFoundError(serverId)
	}
	groups := n.allServerSecurityGroups(serverId)
	if len(groups) > 0 {
		groupNames := make([]nova.SecurityGroupName, len(groups))
		for i, group := range groups {
			groupNames[i] = nova.SecurityGroupName{Name: group.Name}
		}
		server.Groups = &groupNames
	} else {
		server.Groups = nil
	}
	if server.AvailabilityZone != "" && server.AvailabilityZone == n.noValidHostZone.Name {
		server.Status = nova.StatusError
		server.Fault = &nova.ServerFault{
			Code:    500,
			Message: "No valid host was found. There are not enough hosts available.",
		}
	} else if n.serverStatus != "" {
		server.Status = n.serverStatus
	} else {
		server.Status = nova.StatusActive
	}
	return &server, nil
}

// serverByName retrieves the first existing server with the given name.
func (n *Nova) serverByName(name string) (*nova.ServerDetail, error) {
	if err := n.ProcessFunctionHook(n, name); err != nil {
		return nil, err
	}
	for _, server := range n.servers {
		if server.Name == name {
			return &server, nil
		}
	}
	return nil, testservices.NewServerByNameNotFoundError(name)
}

// serverAsEntity returns the stored ServerDetail as Entity.
func (n *Nova) serverAsEntity(serverId string) (*nova.Entity, error) {
	if err := n.ProcessFunctionHook(n, serverId); err != nil {
		return nil, err
	}
	server, err := n.server(serverId)
	if err != nil {
		return nil, err
	}
	return &nova.Entity{
		Id:    server.Id,
		UUID:  server.UUID,
		Name:  server.Name,
		Links: server.Links,
	}, nil
}

// filter is used internally by matchServers.
type filter map[string]string

// matchServers returns a list of matching servers, after applying the
// given filter. Each separate filter is combined with a logical AND.
// Each filter can have only one value. A nil filter matches all servers.
//
// This is tested to match OpenStack behavior. Regular expression
// matching is supported for FilterServer only, and the supported
// syntax is limited to whatever DB backend is used (see SQL
// REGEXP/RLIKE).
//
// Example:
//
// f := filter{
//     nova.FilterStatus: nova.StatusActive,
//     nova.FilterServer: `foo.*`,
// }
//
// This will match all servers with status "ACTIVE", and names starting
// with "foo".
func (n *Nova) matchServers(f filter) ([]nova.ServerDetail, error) {
	if err := n.ProcessFunctionHook(n, f); err != nil {
		return nil, err
	}
	var servers []nova.ServerDetail
	for _, server := range n.servers {
		servers = append(servers, server)
	}
	if len(f) == 0 {
		return servers, nil // empty filter matches everything
	}
	if status := f[nova.FilterStatus]; status != "" {
		matched := []nova.ServerDetail{}
		for _, server := range servers {
			if server.Status == status {
				matched = append(matched, server)
			}
		}
		if len(matched) == 0 {
			// no match, so no need to look further
			return nil, nil
		}
		servers = matched
	}
	if nameRex := f[nova.FilterServer]; nameRex != "" {
		matched := []nova.ServerDetail{}
		rex, err := regexp.Compile(nameRex)
		if err != nil {
			return nil, err
		}
		for _, server := range servers {
			if rex.MatchString(server.Name) {
				matched = append(matched, server)
			}
		}
		if len(matched) == 0 {
			// no match, here so ignore other results
			return nil, nil
		}
		servers = matched
	}
	return servers, nil
	// TODO(dimitern) - 2013-02-11 bug=1121690
	// implement FilterFlavor, FilterImage, FilterMarker, FilterLimit and FilterChangesSince
}

// allServers returns a list of all existing servers.
// Filtering is supported, see filter type for more info.
func (n *Nova) allServers(f filter) ([]nova.ServerDetail, error) {
	return n.matchServers(f)
}

// allServersAsEntities returns all servers as Entity structs.
// Filtering is supported, see filter type for more info.
func (n *Nova) allServersAsEntities(f filter) ([]nova.Entity, error) {
	var entities []nova.Entity
	servers, err := n.matchServers(f)
	if err != nil {
		return nil, err
	}
	for _, server := range servers {
		entities = append(entities, nova.Entity{
			Id:    server.Id,
			UUID:  server.UUID,
			Name:  server.Name,
			Links: server.Links,
		})
	}
	return entities, nil
}

// removeServer deletes an existing server.
func (n *Nova) removeServer(serverId string) error {
	if err := n.ProcessFunctionHook(n, serverId); err != nil {
		return err
	}
	if _, err := n.server(serverId); err != nil {
		return err
	}
	delete(n.servers, serverId)
	return nil
}

func (n *Nova) updateSecurityGroup(group nova.SecurityGroup) error {
	if err := n.ProcessFunctionHook(n, group); err != nil {
		return err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.UpdateNovaSecurityGroup(group)
	}
	existingGroup, err := n.securityGroup(group.Id)
	if err != nil {
		return testservices.NewSecurityGroupByIDNotFoundError(group.Id)
	}
	existingGroup.Name = group.Name
	existingGroup.Description = group.Description
	n.groups[group.Id] = *existingGroup
	return nil
}

// addSecurityGroup creates a new security group.
func (n *Nova) addSecurityGroup(group nova.SecurityGroup) error {
	if err := n.ProcessFunctionHook(n, group); err != nil {
		return err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.AddNovaSecurityGroup(group)
	}
	if _, err := n.securityGroup(group.Id); err == nil {
		return testservices.NewSecurityGroupAlreadyExistsError(group.Id)
	}
	group.TenantId = n.TenantId
	if group.Rules == nil {
		group.Rules = []nova.SecurityGroupRule{}
	}
	n.groups[group.Id] = group
	return nil
}

// securityGroup retrieves an existing group by ID.
func (n *Nova) securityGroup(groupId string) (*nova.SecurityGroup, error) {
	if err := n.ProcessFunctionHook(n, groupId); err != nil {
		return nil, err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.NovaSecurityGroup(groupId)
	}
	group, ok := n.groups[groupId]
	if !ok {
		return nil, testservices.NewSecurityGroupByIDNotFoundError(groupId)
	}
	return &group, nil
}

// securityGroupByName retrieves an existing named group.
func (n *Nova) securityGroupByName(groupName string) (*nova.SecurityGroup, error) {
	if err := n.ProcessFunctionHook(n, groupName); err != nil {
		return nil, err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.NovaSecurityGroupByName(groupName)
	}
	for _, group := range n.groups {
		if group.Name == groupName {
			return &group, nil
		}
	}
	return nil, testservices.NewSecurityGroupByNameNotFoundError(groupName)
}

// allSecurityGroups returns a list of all existing groups.
func (n *Nova) allSecurityGroups() []nova.SecurityGroup {
	var groups []nova.SecurityGroup
	if n.useNeutronNetworking {
		return n.neutronModel.AllNovaSecurityGroups()
	}
	for _, group := range n.groups {
		groups = append(groups, group)
	}
	return groups
}

// removeSecurityGroup deletes an existing group.
func (n *Nova) removeSecurityGroup(groupId string) error {
	if err := n.ProcessFunctionHook(n, groupId); err != nil {
		return err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.RemoveSecurityGroup(groupId)
	}
	if _, err := n.securityGroup(groupId); err != nil {
		return err
	}
	delete(n.groups, groupId)
	return nil
}

// addSecurityGroupRule creates a new rule in an existing group.
// This can be either an ingress or a group rule (see the notes
// about nova.RuleInfo).
func (n *Nova) addSecurityGroupRule(ruleId string, rule nova.RuleInfo) error {
	if err := n.ProcessFunctionHook(n, ruleId, rule); err != nil {
		return err
	}
	if _, err := n.securityGroupRule(ruleId); err == nil {
		return testservices.NewSecurityGroupRuleAlreadyExistsError(ruleId)
	}
	group, err := n.securityGroup(rule.ParentGroupId)
	if err != nil {
		return err
	}
	for _, ru := range group.Rules {
		if ru.Id == ruleId {
			return testservices.NewCannotAddTwiceRuleToGroupError(ru.Id, group.Id)
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
			return testservices.NewUnknownSecurityGroupError(*rule.GroupId)
		}
		newrule.Group = nova.SecurityGroupRef{
			TenantId: sourceGroup.TenantId,
			Name:     sourceGroup.Name,
		}
	} else if rule.Cidr == "" {
		// http://pad.lv/1226996
		// It seems that if you don't supply Cidr or GroupId then
		// Openstack treats the Cidr as 0.0.0.0/0
		// However, since that is not clearly specified we just panic()
		// because we don't want to rely on that behavior
		panic(fmt.Sprintf("Neither Cidr nor GroupId are set for this SecurityGroup Rule: %v", rule))
	}
	if rule.FromPort != 0 {
		newrule.FromPort = &rule.FromPort
	}
	if rule.ToPort != 0 {
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
// or (when groupId="-1") whether the given rule exists.
func (n *Nova) hasSecurityGroupRule(groupId, ruleId string) bool {
	rule, ok := n.rules[ruleId]
	_, err := n.securityGroup(groupId)
	return ok && (groupId == "-1" || (err == nil && rule.ParentGroupId == groupId))
}

// securityGroupRule retrieves an existing rule by ID.
func (n *Nova) securityGroupRule(ruleId string) (*nova.SecurityGroupRule, error) {
	if err := n.ProcessFunctionHook(n, ruleId); err != nil {
		return nil, err
	}
	rule, ok := n.rules[ruleId]
	if !ok {
		return nil, testservices.NewSecurityGroupRuleNotFoundError(ruleId)
	}
	return &rule, nil
}

// removeSecurityGroupRule deletes an existing rule from its group.
func (n *Nova) removeSecurityGroupRule(ruleId string) error {
	if err := n.ProcessFunctionHook(n, ruleId); err != nil {
		return err
	}
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
func (n *Nova) addServerSecurityGroup(serverId string, groupId string) error {
	if err := n.ProcessFunctionHook(n, serverId, groupId); err != nil {
		return err
	}
	if n.useNeutronNetworking {
		if _, err := n.neutronModel.NovaSecurityGroup(groupId); err != nil {
			return err
		}
	} else {
		if _, err := n.securityGroup(groupId); err != nil {
			return err
		}
	}
	if _, err := n.server(serverId); err != nil {
		return err
	}
	groups, ok := n.serverGroups[serverId]
	if ok {
		for _, gid := range groups {
			if gid == groupId {
				return testservices.NewServerBelongsToGroupError(serverId, groupId)
			}
		}
	}
	groups = append(groups, groupId)
	n.serverGroups[serverId] = groups
	return nil
}

// hasServerSecurityGroup returns whether the given server belongs to the group.
func (n *Nova) hasServerSecurityGroup(serverId string, groupId string) bool {
	if n.useNeutronNetworking {
		if _, err := n.neutronModel.NovaSecurityGroup(groupId); err != nil {
			return false
		}
	} else {
		if _, err := n.securityGroup(groupId); err != nil {
			return false
		}
	}
	if _, err := n.server(serverId); err != nil {
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
func (n *Nova) removeServerSecurityGroup(serverId string, groupId string) error {
	if err := n.ProcessFunctionHook(n, serverId, groupId); err != nil {
		return err
	}
	if n.useNeutronNetworking {
		if _, err := n.neutronModel.NovaSecurityGroup(groupId); err != nil {
			return err
		}
	} else {
		if _, err := n.securityGroup(groupId); err != nil {
			return err
		}
	}
	if _, err := n.server(serverId); err != nil {
		return err
	}
	groups, ok := n.serverGroups[serverId]
	if !ok {
		return testservices.NewServerDoesNotBelongToGroupsError(serverId)
	}
	idx := -1
	for gi, gid := range groups {
		if gid == groupId {
			idx = gi
			break
		}
	}
	if idx == -1 {
		return testservices.NewServerDoesNotBelongToGroupError(serverId, groupId)
	}
	groups = append(groups[:idx], groups[idx+1:]...)
	n.serverGroups[serverId] = groups
	return nil
}

// addFloatingIP creates a new floating IP address in the pool.
func (n *Nova) addFloatingIP(ip nova.FloatingIP) error {
	if err := n.ProcessFunctionHook(n, ip); err != nil {
		return err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.AddNovaFloatingIP(ip)
	}
	if _, err := n.floatingIP(ip.Id); err == nil {
		return testservices.NewFloatingIPExistsError(ip.Id)
	}
	n.floatingIPs[ip.Id] = ip
	return nil
}

// hasFloatingIP returns whether the given floating IP address exists.
func (n *Nova) hasFloatingIP(address string) bool {
	if n.useNeutronNetworking {
		return n.neutronModel.HasFloatingIP(address)
	}
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
func (n *Nova) floatingIP(ipId string) (*nova.FloatingIP, error) {
	if err := n.ProcessFunctionHook(n, ipId); err != nil {
		return nil, err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.NovaFloatingIP(ipId)
	}
	ip, ok := n.floatingIPs[ipId]
	if !ok {
		return nil, testservices.NewFloatingIPNotFoundError(ipId)
	}
	return &ip, nil
}

// floatingIPByAddr retrieves the floating IP by address.
func (n *Nova) floatingIPByAddr(address string) (*nova.FloatingIP, error) {
	if err := n.ProcessFunctionHook(n, address); err != nil {
		return nil, err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.NovaFloatingIPByAddr(address)
	}
	for _, fip := range n.floatingIPs {
		if fip.IP == address {
			return &fip, nil
		}
	}
	return nil, testservices.NewFloatingIPNotFoundError(address)
}

// allFloatingIPs returns a list of all created floating IPs.
func (n *Nova) allFloatingIPs() []nova.FloatingIP {
	if n.useNeutronNetworking {
		return n.neutronModel.AllNovaFloatingIPs()
	}
	var fips []nova.FloatingIP
	for _, fip := range n.floatingIPs {
		fips = append(fips, fip)
	}
	return fips
}

// removeFloatingIP deletes an existing floating IP by ID.
func (n *Nova) removeFloatingIP(ipId string) error {
	if err := n.ProcessFunctionHook(n, ipId); err != nil {
		return err
	}
	if n.useNeutronNetworking {
		return n.neutronModel.RemoveFloatingIP(ipId)
	}
	if _, err := n.floatingIP(ipId); err != nil {
		return err
	}
	delete(n.floatingIPs, ipId)
	return nil
}

// addServerFloatingIP attaches an existing floating IP to a server.
func (n *Nova) addServerFloatingIP(serverId string, ipId string) error {
	if err := n.ProcessFunctionHook(n, serverId, ipId); err != nil {
		return err
	}
	if _, err := n.server(serverId); err != nil {
		return err
	}
	fixedIP := "4.3.2.1" // not important really, unused
	var fip *nova.FloatingIP
	var err error
	if n.useNeutronNetworking {
		fip, err = n.neutronModel.NovaFloatingIP(ipId)
		if err != nil {
			return err
		}
		fip.FixedIP = &fixedIP
		if err := n.neutronModel.UpdateNovaFloatingIP(fip); err != nil {
			return err
		}
	} else {
		fip, err = n.floatingIP(ipId)
		if err != nil {
			return err
		} else {
			fip.FixedIP = &fixedIP
			fip.InstanceId = &serverId
			n.floatingIPs[ipId] = *fip
		}
	}
	fips, ok := n.serverIPs[serverId]
	if ok {
		for _, fipId := range fips {
			if fipId == ipId {
				return testservices.NewServerHasFloatingIPError(serverId, ipId)
			}
		}
	}
	fips = append(fips, ipId)
	n.serverIPs[serverId] = fips
	if err := n.addFloatingIPToServerAddresses(serverId, fip.IP); err != nil {
		return err
	}
	return nil
}

// addFloatingIPToServerAddresses adds a floating ip address to the servers list
// of Addresses to facilitate juju openstack provider tests.
func (n *Nova) addFloatingIPToServerAddresses(serverId, address string) error {
	server, err := n.server(serverId)
	if err != nil {
		return err
	}
	newAddresses := server.Addresses["private"]
	if strings.Contains(address, ":") {
		newAddresses = append(newAddresses, nova.IPAddress{6, address, "floating"})
	} else {
		newAddresses = append(newAddresses, nova.IPAddress{4, address, "floating"})
	}
	server.Addresses["private"] = newAddresses
	n.servers[serverId] = *server
	return nil
}

// hasServerFloatingIP verifies the given floating IP belongs to a server.
func (n *Nova) hasServerFloatingIP(serverId, address string) bool {
	if _, err := n.server(serverId); err != nil {
		return false
	}
	var fip *nova.FloatingIP
	var err error
	if n.useNeutronNetworking {
		fip, err = n.neutronModel.NovaFloatingIPByAddr(address)
	} else {
		fip, err = n.floatingIPByAddr(address)
	}
	if err != nil {
		return false
	}
	fips, ok := n.serverIPs[serverId]
	if !ok {
		return false
	}
	for _, fipId := range fips {
		if fipId == fip.Id {
			return true
		}
	}
	return false
}

// removeFloatingIPFromServerAddresses removes a floating ip address from the
// servers list of Addresses to facilitate juju openstack provider tests.
func (n *Nova) removeFloatingIPFromServerAddresses(serverId, address string) error {
	server, err := n.server(serverId)
	if err != nil {
		return err
	}
	serverAddresses := []nova.IPAddress{}
	for _, serverAddress := range server.Addresses["private"] {
		if serverAddress.Address != address {
			serverAddresses = append(serverAddresses, serverAddress)
		}
	}
	if len(serverAddresses) != 0 {
		server.Addresses["private"] = serverAddresses
	} else {
		server.Addresses["private"] = []nova.IPAddress{}
	}
	n.servers[serverId] = *server
	return nil
}

// removeServerFloatingIP deletes an attached floating IP from a server.
func (n *Nova) removeServerFloatingIP(serverId string, ipId string) error {
	if err := n.ProcessFunctionHook(n, serverId); err != nil {
		return err
	}
	if _, err := n.server(serverId); err != nil {
		return err
	}
	var fip *nova.FloatingIP
	var err error
	if n.useNeutronNetworking {
		fip, err = n.neutronModel.NovaFloatingIP(ipId)
		if err != nil {
			return err
		}
		fip.FixedIP = nil
		if err = n.neutronModel.UpdateNovaFloatingIP(fip); err != nil {
			return err
		}
	} else {
		if fip, err = n.floatingIP(ipId); err != nil {
			return err
		} else {
			fip.FixedIP = nil
			fip.InstanceId = nil
			n.floatingIPs[ipId] = *fip
		}
	}
	if err := n.removeFloatingIPFromServerAddresses(serverId, fip.IP); err != nil {
		return err
	}
	fips, ok := n.serverIPs[serverId]
	if !ok {
		return testservices.NewNoFloatingIPsToRemoveError(serverId)
	}
	idx := -1
	for fi, fipId := range fips {
		if fipId == ipId {
			idx = fi
			break
		}
	}
	if idx == -1 {
		return testservices.NewNoFloatingIPsError(serverId, ipId)
	}
	fips = append(fips[:idx], fips[idx+1:]...)
	n.serverIPs[serverId] = fips
	return nil
}

// allNetworks returns a list of all existing networks.
func (n *Nova) allNetworks() (networks []nova.Network) {
	if n.useNeutronNetworking {
		return n.neutronModel.AllNovaNetworks()
	} else {
		for _, net := range n.networks {
			networks = append(networks, net)
		}
		return networks
	}
}

// networks returns the named network if it exists
func (n *Nova) network(name string) (*nova.Network, error) {
	if n.useNeutronNetworking {
		return n.neutronModel.NovaNetwork(name)
	} else {
		net, ok := n.networks[name]
		var err error
		if !ok {
			err = errors.NewNotFoundf(nil, nil, "network")
		}
		return &net, err
	}
}

// allAvailabilityZones returns a list of all existing availability zones,
// sorted by name.
func (n *Nova) allAvailabilityZones() (zones []nova.AvailabilityZone) {
	for _, zone := range n.availabilityZones {
		zones = append(zones, zone)
	}
	sort.Sort(azByName(zones))
	return zones
}

type azByName []nova.AvailabilityZone

func (a azByName) Len() int {
	return len(a)
}

func (a azByName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

func (a azByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// setServerMetadata sets metadata on a server.
func (n *Nova) setServerMetadata(serverId string, metadata map[string]string) error {
	if err := n.ProcessFunctionHook(n, serverId, metadata); err != nil {
		return err
	}
	server, err := n.server(serverId)
	if err != nil {
		return err
	}
	if server.Metadata == nil {
		server.Metadata = make(map[string]string)
	}
	for k, v := range metadata {
		server.Metadata[k] = v
	}
	n.servers[serverId] = *server
	return nil
}

// AddOSInterface adds a os-interface attachment to a server.
func (n *Nova) AddOSInterface(serverID string, osInterfaces ...nova.OSInterface) error {
	for _, osInter := range osInterfaces {
		n.nextOSInterfaceId++

		port := &osInter
		port.PortID = strconv.Itoa(n.nextOSInterfaceId)

		n.serverIdToOSInterfaces[serverID] = append(n.serverIdToOSInterfaces[serverID], *port)
	}

	return nil
}

// RemoveOSInterface removes a os-interface attachment from a server based
// on the matching criteria.
func (n *Nova) RemoveOSInterface(serverID, ipAddress string) error {
	interfaces, ok := n.serverIdToOSInterfaces[serverID]
	if !ok {
		return testservices.NewServerByIDNotFoundError(serverID)
	}

	for i, v := range interfaces {
		if v.IPAddress == ipAddress {
			interfaces = append(interfaces[:i], interfaces[i+1:]...)
			n.serverIdToOSInterfaces[serverID] = interfaces
			return nil
		}
	}

	return testservices.NewNoSuchOSInterfaceError(ipAddress)
}

func (n *Nova) allOSInterfaces() []nova.OSInterface {
	var results []nova.OSInterface
	for serverID := range n.servers {
		results = append(results, n.serverOSInterfaces(serverID)...)
	}
	return results
}

func (n *Nova) serverOSInterfaces(serverID string) []nova.OSInterface {
	if interfaces, ok := n.serverIdToOSInterfaces[serverID]; ok {
		return interfaces
	}
	return make([]nova.OSInterface, 0)
}

func (n *Nova) serverOSInterface(serverID string, ipAddress string) (nova.OSInterface, error) {
	for _, osInterface := range n.serverOSInterfaces(serverID) {
		if osInterface.IPAddress == ipAddress {
			return osInterface, nil
		}
	}
	return nova.OSInterface{}, testservices.NewNoSuchOSInterfaceError(ipAddress)
}

func (n *Nova) hasServerOSInterface(serverID string, ipAddress string) bool {
	for _, osInterface := range n.serverOSInterfaces(serverID) {
		for _, ips := range osInterface.FixedIPs {
			if ips.IPAddress == ipAddress {
				return true
			}
		}
	}
	return false
}
