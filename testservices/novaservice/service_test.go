// Nova double testing service - internal direct API tests

package novaservice

import (
	"fmt"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/nova"
	"strings"
)

type NovaServiceSuite struct {
	service           NovaService
	endpointNoVersion string
	endpoint          string
}

var baseURL = "/v2/"
var token = "token"
var hostname = "http://example.com/"

var _ = Suite(&NovaServiceSuite{})

func (s *NovaServiceSuite) SetUpSuite(c *C) {
	s.service = New(hostname, baseURL, token)
	s.endpointNoVersion = hostname + token
	s.endpoint = hostname + strings.TrimLeft(baseURL, "/") + token
}

func (s *NovaServiceSuite) ensureNoFlavor(c *C, flavor nova.FlavorDetail) {
	_, err := s.service.GetFlavor(flavor.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such flavor %q", flavor.Id))
}

func (s *NovaServiceSuite) ensureNoServer(c *C, server nova.ServerDetail) {
	_, err := s.service.GetServer(server.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such server %q", server.Id))
}

func (s *NovaServiceSuite) ensureNoGroup(c *C, group nova.SecurityGroup) {
	_, err := s.service.GetSecurityGroup(group.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such security group %d", group.Id))
}

func (s *NovaServiceSuite) ensureNoRule(c *C, rule nova.SecurityGroupRule) {
	_, err := s.service.GetSecurityGroupRule(rule.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such security group rule %d", rule.Id))
}

func (s *NovaServiceSuite) ensureNoIP(c *C, ip nova.FloatingIP) {
	_, err := s.service.GetFloatingIP(ip.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such floating IP %d", ip.Id))
}

func (s *NovaServiceSuite) addFlavor(c *C, flavor nova.FlavorDetail) {
	s.ensureNoFlavor(c, flavor)
	err := s.service.AddFlavor(flavor)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) addServer(c *C, server nova.ServerDetail) {
	s.ensureNoServer(c, server)
	err := s.service.AddServer(server)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) addGroup(c *C, group nova.SecurityGroup) {
	s.ensureNoGroup(c, group)
	err := s.service.AddSecurityGroup(group)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) addIP(c *C, ip nova.FloatingIP) {
	s.ensureNoIP(c, ip)
	err := s.service.AddFloatingIP(ip)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) removeFlavor(c *C, flavor nova.FlavorDetail) {
	err := s.service.RemoveFlavor(flavor.Id)
	c.Assert(err, IsNil)
	s.ensureNoFlavor(c, flavor)
}

func (s *NovaServiceSuite) removeServer(c *C, server nova.ServerDetail) {
	err := s.service.RemoveServer(server.Id)
	c.Assert(err, IsNil)
	s.ensureNoServer(c, server)
}

func (s *NovaServiceSuite) removeGroup(c *C, group nova.SecurityGroup) {
	err := s.service.RemoveSecurityGroup(group.Id)
	c.Assert(err, IsNil)
	s.ensureNoGroup(c, group)
}

func (s *NovaServiceSuite) removeRule(c *C, rule nova.SecurityGroupRule) {
	err := s.service.RemoveSecurityGroupRule(rule.Id)
	c.Assert(err, IsNil)
	s.ensureNoRule(c, rule)
}

func (s *NovaServiceSuite) removeIP(c *C, ip nova.FloatingIP) {
	err := s.service.RemoveFloatingIP(ip.Id)
	c.Assert(err, IsNil)
	s.ensureNoIP(c, ip)
}

func (s *NovaServiceSuite) TestAddRemoveFlavor(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.addFlavor(c, flavor)
	s.removeFlavor(c, flavor)
}

func (s *NovaServiceSuite) TestAddFlavorCreatesLinks(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.addFlavor(c, flavor)
	defer s.removeFlavor(c, flavor)
	fl, _ := s.service.GetFlavor(flavor.Id)
	url := "/flavors/" + flavor.Id
	links := []nova.Link{
		nova.Link{Href: s.endpoint + url, Rel: "self"},
		nova.Link{Href: s.endpointNoVersion + url, Rel: "bookmark"},
	}
	c.Assert(fl.Links, DeepEquals, links)
}

func (s *NovaServiceSuite) TestAddFlavorWithLinks(c *C) {
	flavor := nova.FlavorDetail{
		Id: "test",
		Links: []nova.Link{
			nova.Link{Href: "href", Rel: "rel"},
		},
	}
	s.addFlavor(c, flavor)
	defer s.removeFlavor(c, flavor)
	fl, _ := s.service.GetFlavor(flavor.Id)
	c.Assert(fl, DeepEquals, flavor)
}

func (s *NovaServiceSuite) TestAddFlavorTwiceFails(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.addFlavor(c, flavor)
	defer s.removeFlavor(c, flavor)
	err := s.service.AddFlavor(flavor)
	c.Assert(err, ErrorMatches, `a flavor with id "test" already exists`)
}

func (s *NovaServiceSuite) TestRemoveFlavorTwiceFails(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.addFlavor(c, flavor)
	s.removeFlavor(c, flavor)
	err := s.service.RemoveFlavor(flavor.Id)
	c.Assert(err, ErrorMatches, `no such flavor "test"`)
}

func (s *NovaServiceSuite) TestAllFlavors(c *C) {
	_, err := s.service.AllFlavors()
	c.Assert(err, ErrorMatches, "no flavors to return")
	flavors := []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1", Links: []nova.Link{}},
		nova.FlavorDetail{Id: "fl2", Links: []nova.Link{}},
	}
	s.addFlavor(c, flavors[0])
	s.addFlavor(c, flavors[1])
	defer s.removeFlavor(c, flavors[0])
	defer s.removeFlavor(c, flavors[1])
	fl, err := s.service.AllFlavors()
	c.Assert(err, IsNil)
	c.Assert(fl, HasLen, len(flavors))
	if fl[0].Id != flavors[0].Id {
		fl[0], fl[1] = fl[1], fl[0]
	}
	c.Assert(fl, DeepEquals, flavors)
}

func (s *NovaServiceSuite) TestAllFlavorsAsEntities(c *C) {
	_, err := s.service.AllFlavorsAsEntities()
	c.Assert(err, ErrorMatches, "no flavors to return")
	entities := []nova.Entity{
		nova.Entity{Id: "fl1", Links: []nova.Link{}},
		nova.Entity{Id: "fl2", Links: []nova.Link{}},
	}
	flavors := []nova.FlavorDetail{
		nova.FlavorDetail{Id: entities[0].Id, Links: entities[0].Links},
		nova.FlavorDetail{Id: entities[1].Id, Links: entities[1].Links},
	}
	s.addFlavor(c, flavors[0])
	s.addFlavor(c, flavors[1])
	defer s.removeFlavor(c, flavors[0])
	defer s.removeFlavor(c, flavors[1])
	ent, err := s.service.AllFlavorsAsEntities()
	c.Assert(err, IsNil)
	c.Assert(ent, HasLen, len(entities))
	if ent[0].Id != entities[0].Id {
		ent[0], ent[1] = ent[1], ent[0]
	}
	c.Assert(ent, DeepEquals, entities)
}

func (s *NovaServiceSuite) TestGetFlavor(c *C) {
	flavor := nova.FlavorDetail{
		Id:    "test",
		Name:  "flavor",
		RAM:   128,
		VCPUs: 2,
		Disk:  123456,
		Links: []nova.Link{},
	}
	s.addFlavor(c, flavor)
	defer s.removeFlavor(c, flavor)
	fl, _ := s.service.GetFlavor(flavor.Id)
	c.Assert(fl, DeepEquals, flavor)
}

func (s *NovaServiceSuite) TestGetFlavorAsEntity(c *C) {
	entity := nova.Entity{
		Id:    "test",
		Name:  "flavor",
		Links: []nova.Link{},
	}
	flavor := nova.FlavorDetail{
		Id:    entity.Id,
		Name:  entity.Name,
		Links: entity.Links,
	}
	s.addFlavor(c, flavor)
	defer s.removeFlavor(c, flavor)
	ent, _ := s.service.GetFlavorAsEntity(flavor.Id)
	c.Assert(ent, DeepEquals, entity)
}

func (s *NovaServiceSuite) TestAddRemoveServer(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.addServer(c, server)
	s.removeServer(c, server)
}

func (s *NovaServiceSuite) TestAddServerCreatesLinks(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	sr, _ := s.service.GetServer(server.Id)
	url := "/servers/" + server.Id
	links := []nova.Link{
		nova.Link{Href: s.endpoint + url, Rel: "self"},
		nova.Link{Href: s.endpointNoVersion + url, Rel: "bookmark"},
	}
	c.Assert(sr.Links, DeepEquals, links)
}

func (s *NovaServiceSuite) TestAddServerWithLinks(c *C) {
	server := nova.ServerDetail{
		Id: "test",
		Links: []nova.Link{
			nova.Link{Href: "href", Rel: "rel"},
		},
	}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	sr, _ := s.service.GetServer(server.Id)
	c.Assert(sr, DeepEquals, server)
}

func (s *NovaServiceSuite) TestAddServerTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	err := s.service.AddServer(server)
	c.Assert(err, ErrorMatches, `a server with id "test" already exists`)
}

func (s *NovaServiceSuite) TestRemoveServerTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.addServer(c, server)
	s.removeServer(c, server)
	err := s.service.RemoveServer(server.Id)
	c.Assert(err, ErrorMatches, `no such server "test"`)
}

func (s *NovaServiceSuite) TestAllServers(c *C) {
	_, err := s.service.AllServers()
	c.Assert(err, ErrorMatches, "no servers to return")
	servers := []nova.ServerDetail{
		nova.ServerDetail{Id: "sr1", Links: []nova.Link{}},
		nova.ServerDetail{Id: "sr2", Links: []nova.Link{}},
	}
	s.addServer(c, servers[0])
	s.addServer(c, servers[1])
	defer s.removeServer(c, servers[0])
	defer s.removeServer(c, servers[1])
	sr, err := s.service.AllServers()
	c.Assert(err, IsNil)
	c.Assert(sr, HasLen, len(servers))
	if sr[0].Id != servers[0].Id {
		sr[0], sr[1] = sr[1], sr[0]
	}
	c.Assert(sr, DeepEquals, servers)
}

func (s *NovaServiceSuite) TestAllServersAsEntities(c *C) {
	_, err := s.service.AllServersAsEntities()
	c.Assert(err, ErrorMatches, "no servers to return")
	entities := []nova.Entity{
		nova.Entity{Id: "sr1", Links: []nova.Link{}},
		nova.Entity{Id: "sr2", Links: []nova.Link{}},
	}
	servers := []nova.ServerDetail{
		nova.ServerDetail{Id: entities[0].Id, Links: entities[0].Links},
		nova.ServerDetail{Id: entities[1].Id, Links: entities[1].Links},
	}
	s.addServer(c, servers[0])
	s.addServer(c, servers[1])
	defer s.removeServer(c, servers[0])
	defer s.removeServer(c, servers[1])
	ent, err := s.service.AllServersAsEntities()
	c.Assert(err, IsNil)
	c.Assert(ent, HasLen, len(entities))
	if ent[0].Id != entities[0].Id {
		ent[0], ent[1] = ent[1], ent[0]
	}
	c.Assert(ent, DeepEquals, entities)
}

func (s *NovaServiceSuite) TestGetServer(c *C) {
	server := nova.ServerDetail{
		Id:          "test",
		Name:        "server",
		AddressIPv4: "1.2.3.4",
		AddressIPv6: "1::fff",
		Created:     "1/1/1",
		Flavor:      nova.Entity{Id: "fl1", Name: "flavor1"},
		Image:       nova.Entity{Id: "im1", Name: "image1"},
		HostId:      "myhost",
		Progress:    123,
		Status:      "st",
		TenantId:    "tenant",
		Updated:     "2/3/4",
		UserId:      "user",
		Links:       []nova.Link{},
	}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	sr, _ := s.service.GetServer(server.Id)
	c.Assert(sr, DeepEquals, server)
}

func (s *NovaServiceSuite) TestGetServerAsEntity(c *C) {
	entity := nova.Entity{
		Id:    "test",
		Name:  "server",
		Links: []nova.Link{},
	}
	server := nova.ServerDetail{
		Id:    entity.Id,
		Name:  entity.Name,
		Links: entity.Links,
	}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	ent, _ := s.service.GetServerAsEntity(server.Id)
	c.Assert(ent, DeepEquals, entity)
}

func (s *NovaServiceSuite) TestAddRemoveSecurityGroup(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.addGroup(c, group)
	s.removeGroup(c, group)
}

func (s *NovaServiceSuite) TestAddSecurityGroupWithRules(c *C) {
	group := nova.SecurityGroup{
		Id:   1,
		Name: "test",
		Rules: []nova.SecurityGroupRule{
			nova.SecurityGroupRule{Id: 10, ParentGroupId: 1},
			nova.SecurityGroupRule{Id: 20, ParentGroupId: 1},
		},
	}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	gr, _ := s.service.GetSecurityGroup(group.Id)
	c.Assert(gr, DeepEquals, group)
}

func (s *NovaServiceSuite) TestAddSecurityGroupTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "test"}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	err := s.service.AddSecurityGroup(group)
	c.Assert(err, ErrorMatches, "a security group with id 1 already exists")
}

func (s *NovaServiceSuite) TestRemoveSecurityGroupTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "test"}
	s.addGroup(c, group)
	s.removeGroup(c, group)
	err := s.service.RemoveSecurityGroup(group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
}

func (s *NovaServiceSuite) TestAllSecurityGroups(c *C) {
	_, err := s.service.AllSecurityGroups()
	c.Assert(err, ErrorMatches, "no security groups to return")
	groups := []nova.SecurityGroup{
		nova.SecurityGroup{Id: 1, Name: "one"},
		nova.SecurityGroup{Id: 2, Name: "two"},
	}
	s.addGroup(c, groups[0])
	s.addGroup(c, groups[1])
	defer s.removeGroup(c, groups[0])
	defer s.removeGroup(c, groups[1])
	gr, err := s.service.AllSecurityGroups()
	c.Assert(err, IsNil)
	c.Assert(gr, HasLen, len(groups))
	if gr[0].Id != groups[0].Id {
		gr[0], gr[1] = gr[1], gr[0]
	}
	c.Assert(gr, DeepEquals, groups)
}

func (s *NovaServiceSuite) TestGetSecurityGroup(c *C) {
	group := nova.SecurityGroup{
		Id:          42,
		TenantId:    "tenant",
		Name:        "group",
		Description: "desc",
		Rules:       []nova.SecurityGroupRule{},
	}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	gr, _ := s.service.GetSecurityGroup(group.Id)
	c.Assert(gr, DeepEquals, group)
}

func (s *NovaServiceSuite) TestAddHasRemoveSecurityGroupRule(c *C) {
	group := nova.SecurityGroup{Id: 1}
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10, ParentGroupId: group.Id}
	s.ensureNoGroup(c, group)
	s.ensureNoRule(c, rule)
	ok := s.service.HasSecurityGroupRule(group.Id, rule.Id)
	c.Assert(ok, Equals, false)
	s.addGroup(c, group)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroupRule(group.Id, rule.Id)
	c.Assert(ok, Equals, true)
	s.removeGroup(c, group)
	ok = s.service.HasSecurityGroupRule(-1, rule.Id)
	c.Assert(ok, Equals, true)
	ok = s.service.HasSecurityGroupRule(group.Id, rule.Id)
	c.Assert(ok, Equals, false)
	s.removeRule(c, rule)
	ok = s.service.HasSecurityGroupRule(-1, rule.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestAddGetIngressSecurityGroupRule(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	ri := nova.RuleInfo{
		FromPort:      1234,
		ToPort:        4321,
		IPProtocol:    "tcp",
		Cidr:          "1.2.3.4/5",
		ParentGroupId: group.Id,
	}
	rule := nova.SecurityGroupRule{
		Id:            10,
		ParentGroupId: group.Id,
		FromPort:      &ri.FromPort,
		ToPort:        &ri.ToPort,
		IPProtocol:    &ri.IPProtocol,
		IPRange:       make(map[string]string),
	}
	rule.IPRange["cidr"] = ri.Cidr
	s.ensureNoRule(c, rule)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.removeRule(c, rule)
	ru, err := s.service.GetSecurityGroupRule(rule.Id)
	c.Assert(err, IsNil)
	c.Assert(ru.Id, Equals, rule.Id)
	c.Assert(ru.ParentGroupId, Equals, rule.ParentGroupId)
	c.Assert(*ru.FromPort, Equals, *rule.FromPort)
	c.Assert(*ru.ToPort, Equals, *rule.ToPort)
	c.Assert(*ru.IPProtocol, Equals, *rule.IPProtocol)
	c.Assert(ru.IPRange, DeepEquals, rule.IPRange)
}

func (s *NovaServiceSuite) TestAddGetGroupSecurityGroupRule(c *C) {
	srcGroup := nova.SecurityGroup{Id: 1, Name: "source", TenantId: "tenant"}
	tgtGroup := nova.SecurityGroup{Id: 2, Name: "target"}
	s.addGroup(c, srcGroup)
	s.addGroup(c, tgtGroup)
	defer s.removeGroup(c, srcGroup)
	defer s.removeGroup(c, tgtGroup)
	ri := nova.RuleInfo{
		FromPort:      1234,
		ToPort:        4321,
		IPProtocol:    "tcp",
		GroupId:       &srcGroup.Id,
		ParentGroupId: tgtGroup.Id,
	}
	rule := nova.SecurityGroupRule{
		Id:            10,
		ParentGroupId: tgtGroup.Id,
		FromPort:      &ri.FromPort,
		ToPort:        &ri.ToPort,
		IPProtocol:    &ri.IPProtocol,
		Group: &nova.SecurityGroupRef{
			TenantId: srcGroup.TenantId,
			Name:     srcGroup.Name,
		},
	}
	s.ensureNoRule(c, rule)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.removeRule(c, rule)
	ru, err := s.service.GetSecurityGroupRule(rule.Id)
	c.Assert(err, IsNil)
	c.Assert(ru.Id, Equals, rule.Id)
	c.Assert(ru.ParentGroupId, Equals, rule.ParentGroupId)
	c.Assert(*ru.FromPort, Equals, *rule.FromPort)
	c.Assert(*ru.ToPort, Equals, *rule.ToPort)
	c.Assert(*ru.IPProtocol, Equals, *rule.IPProtocol)
	c.Assert(*ru.Group, DeepEquals, *rule.Group)
}

func (s *NovaServiceSuite) TestAddSecurityGroupRuleTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.removeRule(c, rule)
	err = s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "a security group rule with id 10 already exists")
}

func (s *NovaServiceSuite) TestAddSecurityGroupRuleToParentTwiceFails(c *C) {
	group := nova.SecurityGroup{
		Id: 1,
		Rules: []nova.SecurityGroupRule{
			nova.SecurityGroupRule{Id: 10},
		},
	}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "cannot add twice rule 10 to security group 1")
}

func (s *NovaServiceSuite) TestAddSecurityGroupRuleWithInvalidParentFails(c *C) {
	invalidGroup := nova.SecurityGroup{Id: 1}
	s.ensureNoGroup(c, invalidGroup)
	ri := nova.RuleInfo{ParentGroupId: invalidGroup.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "cannot add a rule to unknown security group 1")
}

func (s *NovaServiceSuite) TestAddGroupSecurityGroupRuleWithInvalidSourceFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	invalidGroupId := 42
	ri := nova.RuleInfo{
		ParentGroupId: group.Id,
		GroupId:       &invalidGroupId,
	}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "unknown source security group 42")
}

func (s *NovaServiceSuite) TestAddSecurityGroupRuleUpdatesParent(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10, ParentGroupId: group.Id}
	s.ensureNoRule(c, rule)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.removeRule(c, rule)
	group.Rules = []nova.SecurityGroupRule{rule}
	gr, err := s.service.GetSecurityGroup(group.Id)
	c.Assert(err, IsNil)
	c.Assert(gr, DeepEquals, group)
}

func (s *NovaServiceSuite) TestRemoveSecurityGroupRuleTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.AddSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	s.removeRule(c, rule)
	err = s.service.RemoveSecurityGroupRule(rule.Id)
	c.Assert(err, ErrorMatches, "no such security group rule 10")
}

func (s *NovaServiceSuite) TestAddHasRemoveServerSecurityGroup(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.ensureNoServer(c, server)
	s.ensureNoGroup(c, group)
	ok := s.service.HasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	ok = s.service.HasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	ok = s.service.HasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	err := s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, true)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestAddServerSecurityGroupWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.ensureNoServer(c, server)
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	err := s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
}

func (s *NovaServiceSuite) TestAddServerSecurityGroupWithInvalidGroupFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.ensureNoGroup(c, group)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	err := s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
}

func (s *NovaServiceSuite) TestAddServerSecurityGroupTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	err := s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	err = s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `server "sr1" already belongs to group 1`)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) TestRemoveServerSecurityGroupWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.addServer(c, server)
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	err := s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	s.removeServer(c, server)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) TestRemoveServerSecurityGroupWithInvalidGroupFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.addGroup(c, group)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	err := s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	s.removeGroup(c, group)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) TestRemoveServerSecurityGroupTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	s.addGroup(c, group)
	defer s.removeGroup(c, group)
	err := s.service.AddServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `server "sr1" does not belong to group 1`)
}

func (s *NovaServiceSuite) TestAddHasRemoveFloatingIP(c *C) {
	ip := nova.FloatingIP{Id: 1, IP: "1.2.3.4"}
	s.ensureNoIP(c, ip)
	ok := s.service.HasFloatingIP(ip.IP)
	c.Assert(ok, Equals, false)
	s.addIP(c, ip)
	ok = s.service.HasFloatingIP("invalid IP")
	c.Assert(ok, Equals, false)
	ok = s.service.HasFloatingIP(ip.IP)
	c.Assert(ok, Equals, true)
	s.removeIP(c, ip)
	ok = s.service.HasFloatingIP(ip.IP)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestAddFloatingIPTwiceFails(c *C) {
	ip := nova.FloatingIP{Id: 1}
	s.addIP(c, ip)
	defer s.removeIP(c, ip)
	err := s.service.AddFloatingIP(ip)
	c.Assert(err, ErrorMatches, "a floating IP with id 1 already exists")
}

func (s *NovaServiceSuite) TestRemoveFloatingIPTwiceFails(c *C) {
	ip := nova.FloatingIP{Id: 1}
	s.addIP(c, ip)
	s.removeIP(c, ip)
	err := s.service.RemoveFloatingIP(ip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
}

func (s *NovaServiceSuite) TestAllFloatingIPs(c *C) {
	_, err := s.service.AllFloatingIPs()
	c.Assert(err, ErrorMatches, "no floating IPs to return")
	fips := []nova.FloatingIP{
		nova.FloatingIP{Id: 1},
		nova.FloatingIP{Id: 2},
	}
	s.addIP(c, fips[0])
	s.addIP(c, fips[1])
	defer s.removeIP(c, fips[0])
	defer s.removeIP(c, fips[1])
	ips, err := s.service.AllFloatingIPs()
	c.Assert(err, IsNil)
	c.Assert(ips, HasLen, len(fips))
	if ips[0].Id != fips[0].Id {
		ips[0], ips[1] = ips[1], ips[0]
	}
	c.Assert(ips, DeepEquals, fips)
}

func (s *NovaServiceSuite) TestGetFloatingIP(c *C) {
	fip := nova.FloatingIP{
		Id:         1,
		IP:         "1.2.3.4",
		Pool:       "pool",
		InstanceId: "sr1",
		FixedIP:    "4.3.2.1",
	}
	s.addIP(c, fip)
	defer s.removeIP(c, fip)
	ip, _ := s.service.GetFloatingIP(fip.Id)
	c.Assert(ip, DeepEquals, fip)
}

func (s *NovaServiceSuite) TestAddHasRemoveServerFloatingIP(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1, IP: "1.2.3.4"}
	s.ensureNoServer(c, server)
	s.ensureNoIP(c, fip)
	ok := s.service.HasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	ok = s.service.HasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	s.addIP(c, fip)
	defer s.removeIP(c, fip)
	ok = s.service.HasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	err := s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, true)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestAddServerFloatingIPWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.ensureNoServer(c, server)
	s.addIP(c, fip)
	defer s.removeIP(c, fip)
	err := s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
}

func (s *NovaServiceSuite) TestAddServerFloatingIPWithInvalidIPFails(c *C) {
	fip := nova.FloatingIP{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.ensureNoIP(c, fip)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	err := s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
}

func (s *NovaServiceSuite) TestAddServerFloatingIPTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	s.addIP(c, fip)
	defer s.removeIP(c, fip)
	err := s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	err = s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `server "sr1" already has floating IP 1`)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) TestRemoveServerFloatingIPWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.addServer(c, server)
	s.addIP(c, fip)
	defer s.removeIP(c, fip)
	err := s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	s.removeServer(c, server)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) TestRemoveServerFloatingIPWithInvalidIPFails(c *C) {
	fip := nova.FloatingIP{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.addIP(c, fip)
	s.addServer(c, server)
	defer s.removeServer(c, server)
	err := s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	s.removeIP(c, fip)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
	s.addIP(c, fip)
	defer s.removeIP(c, fip)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaServiceSuite) TestRemoveServerFloatingIPTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.addServer(c, server)
	defer s.removeServer(c, server)
	s.addIP(c, fip)
	defer s.removeIP(c, fip)
	err := s.service.AddServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `server "sr1" does not have floating IP 1`)
}
