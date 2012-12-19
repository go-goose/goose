// Nova double testing service - internal direct API tests

package novaservice

import (
	"fmt"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/nova"
)

type NovaSuite struct {
	service *Nova
}

const (
	versionPath = "v2"
	token       = "token"
	hostname    = "http://example.com/"
	tenantId    = "tenant_id"
)

var _ = Suite(&NovaSuite{})

func (s *NovaSuite) SetUpSuite(c *C) {
	s.service = New(hostname, versionPath, token, tenantId)
}

func (s *NovaSuite) ensureNoFlavor(c *C, flavor nova.FlavorDetail) {
	_, err := s.service.flavor(flavor.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such flavor %q", flavor.Id))
}

func (s *NovaSuite) ensureNoServer(c *C, server nova.ServerDetail) {
	_, err := s.service.server(server.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such server %q", server.Id))
}

func (s *NovaSuite) ensureNoGroup(c *C, group nova.SecurityGroup) {
	_, err := s.service.securityGroup(group.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such security group %d", group.Id))
}

func (s *NovaSuite) ensureNoRule(c *C, rule nova.SecurityGroupRule) {
	_, err := s.service.securityGroupRule(rule.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such security group rule %d", rule.Id))
}

func (s *NovaSuite) ensureNoIP(c *C, ip nova.FloatingIP) {
	_, err := s.service.floatingIP(ip.Id)
	c.Assert(err, ErrorMatches, fmt.Sprintf("no such floating IP %d", ip.Id))
}

func (s *NovaSuite) createFlavor(c *C, flavor nova.FlavorDetail) {
	s.ensureNoFlavor(c, flavor)
	err := s.service.addFlavor(flavor)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) createServer(c *C, server nova.ServerDetail) {
	s.ensureNoServer(c, server)
	err := s.service.addServer(server)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) createGroup(c *C, group nova.SecurityGroup) {
	s.ensureNoGroup(c, group)
	err := s.service.addSecurityGroup(group)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) createIP(c *C, ip nova.FloatingIP) {
	s.ensureNoIP(c, ip)
	err := s.service.addFloatingIP(ip)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) deleteFlavor(c *C, flavor nova.FlavorDetail) {
	err := s.service.removeFlavor(flavor.Id)
	c.Assert(err, IsNil)
	s.ensureNoFlavor(c, flavor)
}

func (s *NovaSuite) deleteServer(c *C, server nova.ServerDetail) {
	err := s.service.removeServer(server.Id)
	c.Assert(err, IsNil)
	s.ensureNoServer(c, server)
}

func (s *NovaSuite) deleteGroup(c *C, group nova.SecurityGroup) {
	err := s.service.removeSecurityGroup(group.Id)
	c.Assert(err, IsNil)
	s.ensureNoGroup(c, group)
}

func (s *NovaSuite) deleteRule(c *C, rule nova.SecurityGroupRule) {
	err := s.service.removeSecurityGroupRule(rule.Id)
	c.Assert(err, IsNil)
	s.ensureNoRule(c, rule)
}

func (s *NovaSuite) deleteIP(c *C, ip nova.FloatingIP) {
	err := s.service.removeFloatingIP(ip.Id)
	c.Assert(err, IsNil)
	s.ensureNoIP(c, ip)
}

func (s *NovaSuite) TestAddRemoveFlavor(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.createFlavor(c, flavor)
	s.deleteFlavor(c, flavor)
}

func (s *NovaSuite) TestBuildLinksAndAddFlavor(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.service.buildFlavorLinks(&flavor)
	s.createFlavor(c, flavor)
	defer s.deleteFlavor(c, flavor)
	fl, _ := s.service.flavor(flavor.Id)
	url := "/flavors/" + flavor.Id
	links := []nova.Link{
		nova.Link{Href: s.service.endpoint(true, url), Rel: "self"},
		nova.Link{Href: s.service.endpoint(false, url), Rel: "bookmark"},
	}
	c.Assert(fl.Links, DeepEquals, links)
}

func (s *NovaSuite) TestAddFlavorWithLinks(c *C) {
	flavor := nova.FlavorDetail{
		Id: "test",
		Links: []nova.Link{
			nova.Link{Href: "href", Rel: "rel"},
		},
	}
	s.createFlavor(c, flavor)
	defer s.deleteFlavor(c, flavor)
	fl, _ := s.service.flavor(flavor.Id)
	c.Assert(*fl, DeepEquals, flavor)
}

func (s *NovaSuite) TestAddFlavorTwiceFails(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.createFlavor(c, flavor)
	defer s.deleteFlavor(c, flavor)
	err := s.service.addFlavor(flavor)
	c.Assert(err, ErrorMatches, `a flavor with id "test" already exists`)
}

func (s *NovaSuite) TestRemoveFlavorTwiceFails(c *C) {
	flavor := nova.FlavorDetail{Id: "test"}
	s.createFlavor(c, flavor)
	s.deleteFlavor(c, flavor)
	err := s.service.removeFlavor(flavor.Id)
	c.Assert(err, ErrorMatches, `no such flavor "test"`)
}

func (s *NovaSuite) TestAllFlavors(c *C) {
	flavors := s.service.allFlavors()
	c.Assert(flavors, HasLen, 0)
	flavors = []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1"},
		nova.FlavorDetail{Id: "fl2"},
	}
	s.createFlavor(c, flavors[0])
	defer s.deleteFlavor(c, flavors[0])
	s.createFlavor(c, flavors[1])
	defer s.deleteFlavor(c, flavors[1])
	fl := s.service.allFlavors()
	c.Assert(fl, HasLen, len(flavors))
	if fl[0].Id != flavors[0].Id {
		fl[0], fl[1] = fl[1], fl[0]
	}
	c.Assert(fl, DeepEquals, flavors)
}

func (s *NovaSuite) TestAllFlavorsAsEntities(c *C) {
	entities := s.service.allFlavorsAsEntities()
	c.Assert(entities, HasLen, 0)
	entities = []nova.Entity{
		nova.Entity{Id: "fl1"},
		nova.Entity{Id: "fl2"},
	}
	flavors := []nova.FlavorDetail{
		nova.FlavorDetail{Id: entities[0].Id},
		nova.FlavorDetail{Id: entities[1].Id},
	}
	s.createFlavor(c, flavors[0])
	defer s.deleteFlavor(c, flavors[0])
	s.createFlavor(c, flavors[1])
	defer s.deleteFlavor(c, flavors[1])
	ent := s.service.allFlavorsAsEntities()
	c.Assert(ent, HasLen, len(entities))
	if ent[0].Id != entities[0].Id {
		ent[0], ent[1] = ent[1], ent[0]
	}
	c.Assert(ent, DeepEquals, entities)
}

func (s *NovaSuite) TestGetFlavor(c *C) {
	flavor := nova.FlavorDetail{
		Id:    "test",
		Name:  "flavor",
		RAM:   128,
		VCPUs: 2,
		Disk:  123,
	}
	s.createFlavor(c, flavor)
	defer s.deleteFlavor(c, flavor)
	fl, _ := s.service.flavor(flavor.Id)
	c.Assert(*fl, DeepEquals, flavor)
}

func (s *NovaSuite) TestGetFlavorAsEntity(c *C) {
	entity := nova.Entity{
		Id:   "test",
		Name: "flavor",
	}
	flavor := nova.FlavorDetail{
		Id:   entity.Id,
		Name: entity.Name,
	}
	s.createFlavor(c, flavor)
	defer s.deleteFlavor(c, flavor)
	ent, _ := s.service.flavorAsEntity(flavor.Id)
	c.Assert(*ent, DeepEquals, entity)
}

func (s *NovaSuite) TestAddRemoveServer(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.createServer(c, server)
	s.deleteServer(c, server)
}

func (s *NovaSuite) TestBuildLinksAndAddServer(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.service.buildServerLinks(&server)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	sr, _ := s.service.server(server.Id)
	url := "/servers/" + server.Id
	links := []nova.Link{
		nova.Link{Href: s.service.endpoint(true, url), Rel: "self"},
		nova.Link{Href: s.service.endpoint(false, url), Rel: "bookmark"},
	}
	c.Assert(sr.Links, DeepEquals, links)
}

func (s *NovaSuite) TestAddServerWithLinks(c *C) {
	server := nova.ServerDetail{
		Id: "test",
		Links: []nova.Link{
			nova.Link{Href: "href", Rel: "rel"},
		},
	}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	sr, _ := s.service.server(server.Id)
	c.Assert(*sr, DeepEquals, server)
}

func (s *NovaSuite) TestAddServerTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	err := s.service.addServer(server)
	c.Assert(err, ErrorMatches, `a server with id "test" already exists`)
}

func (s *NovaSuite) TestRemoveServerTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "test"}
	s.createServer(c, server)
	s.deleteServer(c, server)
	err := s.service.removeServer(server.Id)
	c.Assert(err, ErrorMatches, `no such server "test"`)
}

func (s *NovaSuite) TestAllServers(c *C) {
	servers := s.service.allServers()
	c.Assert(servers, HasLen, 0)
	servers = []nova.ServerDetail{
		nova.ServerDetail{Id: "sr1"},
		nova.ServerDetail{Id: "sr2"},
	}
	s.createServer(c, servers[0])
	defer s.deleteServer(c, servers[1])
	s.createServer(c, servers[1])
	defer s.deleteServer(c, servers[0])
	sr := s.service.allServers()
	c.Assert(sr, HasLen, len(servers))
	if sr[0].Id != servers[0].Id {
		sr[0], sr[1] = sr[1], sr[0]
	}
	c.Assert(sr, DeepEquals, servers)
}

func (s *NovaSuite) TestAllServersAsEntities(c *C) {
	entities := s.service.allServersAsEntities()
	c.Assert(entities, HasLen, 0)
	entities = []nova.Entity{
		nova.Entity{Id: "sr1"},
		nova.Entity{Id: "sr2"},
	}
	servers := []nova.ServerDetail{
		nova.ServerDetail{Id: entities[0].Id},
		nova.ServerDetail{Id: entities[1].Id},
	}
	s.createServer(c, servers[0])
	defer s.deleteServer(c, servers[0])
	s.createServer(c, servers[1])
	defer s.deleteServer(c, servers[1])
	ent := s.service.allServersAsEntities()
	c.Assert(ent, HasLen, len(entities))
	if ent[0].Id != entities[0].Id {
		ent[0], ent[1] = ent[1], ent[0]
	}
	c.Assert(ent, DeepEquals, entities)
}

func (s *NovaSuite) TestGetServer(c *C) {
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
	}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	sr, _ := s.service.server(server.Id)
	c.Assert(*sr, DeepEquals, server)
}

func (s *NovaSuite) TestGetServerAsEntity(c *C) {
	entity := nova.Entity{
		Id:   "test",
		Name: "server",
	}
	server := nova.ServerDetail{
		Id:   entity.Id,
		Name: entity.Name,
	}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	ent, _ := s.service.serverAsEntity(server.Id)
	c.Assert(*ent, DeepEquals, entity)
}

func (s *NovaSuite) TestAddRemoveSecurityGroup(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.createGroup(c, group)
	s.deleteGroup(c, group)
}

func (s *NovaSuite) TestAddSecurityGroupWithRules(c *C) {
	group := nova.SecurityGroup{
		Id:   1,
		Name: "test",
		Rules: []nova.SecurityGroupRule{
			nova.SecurityGroupRule{Id: 10, ParentGroupId: 1},
			nova.SecurityGroupRule{Id: 20, ParentGroupId: 1},
		},
	}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	gr, _ := s.service.securityGroup(group.Id)
	c.Assert(*gr, DeepEquals, group)
}

func (s *NovaSuite) TestAddSecurityGroupTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "test"}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	err := s.service.addSecurityGroup(group)
	c.Assert(err, ErrorMatches, "a security group with id 1 already exists")
}

func (s *NovaSuite) TestRemoveSecurityGroupTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "test"}
	s.createGroup(c, group)
	s.deleteGroup(c, group)
	err := s.service.removeSecurityGroup(group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
}

func (s *NovaSuite) TestAllSecurityGroups(c *C) {
	groups := s.service.allSecurityGroups()
	c.Assert(groups, HasLen, 0)
	groups = []nova.SecurityGroup{
		nova.SecurityGroup{Id: 1, Name: "one"},
		nova.SecurityGroup{Id: 2, Name: "two"},
	}
	s.createGroup(c, groups[0])
	defer s.deleteGroup(c, groups[0])
	s.createGroup(c, groups[1])
	defer s.deleteGroup(c, groups[1])
	gr := s.service.allSecurityGroups()
	c.Assert(gr, HasLen, len(groups))
	if gr[0].Id != groups[0].Id {
		gr[0], gr[1] = gr[1], gr[0]
	}
	c.Assert(gr, DeepEquals, groups)
}

func (s *NovaSuite) TestGetSecurityGroup(c *C) {
	group := nova.SecurityGroup{
		Id:          42,
		TenantId:    "tenant",
		Name:        "group",
		Description: "desc",
		Rules:       []nova.SecurityGroupRule{},
	}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	gr, _ := s.service.securityGroup(group.Id)
	c.Assert(*gr, DeepEquals, group)
}

func (s *NovaSuite) TestGetSecurityGroupByName(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "test"}
	s.ensureNoGroup(c, group)
	gr, err := s.service.securityGroupByName(group.Name)
	c.Assert(err, ErrorMatches, `no such security group named "test"`)
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	gr, err = s.service.securityGroupByName(group.Name)
	c.Assert(err, IsNil)
	c.Assert(*gr, DeepEquals, group)
}

func (s *NovaSuite) TestAddHasRemoveSecurityGroupRule(c *C) {
	group := nova.SecurityGroup{Id: 1}
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10, ParentGroupId: group.Id}
	s.ensureNoGroup(c, group)
	s.ensureNoRule(c, rule)
	ok := s.service.hasSecurityGroupRule(group.Id, rule.Id)
	c.Assert(ok, Equals, false)
	s.createGroup(c, group)
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	ok = s.service.hasSecurityGroupRule(group.Id, rule.Id)
	c.Assert(ok, Equals, true)
	s.deleteGroup(c, group)
	ok = s.service.hasSecurityGroupRule(-1, rule.Id)
	c.Assert(ok, Equals, true)
	ok = s.service.hasSecurityGroupRule(group.Id, rule.Id)
	c.Assert(ok, Equals, false)
	s.deleteRule(c, rule)
	ok = s.service.hasSecurityGroupRule(-1, rule.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaSuite) TestAddGetIngressSecurityGroupRule(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
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
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.deleteRule(c, rule)
	ru, err := s.service.securityGroupRule(rule.Id)
	c.Assert(err, IsNil)
	c.Assert(ru.Id, Equals, rule.Id)
	c.Assert(ru.ParentGroupId, Equals, rule.ParentGroupId)
	c.Assert(*ru.FromPort, Equals, *rule.FromPort)
	c.Assert(*ru.ToPort, Equals, *rule.ToPort)
	c.Assert(*ru.IPProtocol, Equals, *rule.IPProtocol)
	c.Assert(ru.IPRange, DeepEquals, rule.IPRange)
}

func (s *NovaSuite) TestAddGetGroupSecurityGroupRule(c *C) {
	srcGroup := nova.SecurityGroup{Id: 1, Name: "source", TenantId: "tenant"}
	tgtGroup := nova.SecurityGroup{Id: 2, Name: "target"}
	s.createGroup(c, srcGroup)
	defer s.deleteGroup(c, srcGroup)
	s.createGroup(c, tgtGroup)
	defer s.deleteGroup(c, tgtGroup)
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
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.deleteRule(c, rule)
	ru, err := s.service.securityGroupRule(rule.Id)
	c.Assert(err, IsNil)
	c.Assert(ru.Id, Equals, rule.Id)
	c.Assert(ru.ParentGroupId, Equals, rule.ParentGroupId)
	c.Assert(*ru.FromPort, Equals, *rule.FromPort)
	c.Assert(*ru.ToPort, Equals, *rule.ToPort)
	c.Assert(*ru.IPProtocol, Equals, *rule.IPProtocol)
	c.Assert(*ru.Group, DeepEquals, *rule.Group)
}

func (s *NovaSuite) TestAddSecurityGroupRuleTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.deleteRule(c, rule)
	err = s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "a security group rule with id 10 already exists")
}

func (s *NovaSuite) TestAddSecurityGroupRuleToParentTwiceFails(c *C) {
	group := nova.SecurityGroup{
		Id: 1,
		Rules: []nova.SecurityGroupRule{
			nova.SecurityGroupRule{Id: 10},
		},
	}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "cannot add twice rule 10 to security group 1")
}

func (s *NovaSuite) TestAddSecurityGroupRuleWithInvalidParentFails(c *C) {
	invalidGroup := nova.SecurityGroup{Id: 1}
	s.ensureNoGroup(c, invalidGroup)
	ri := nova.RuleInfo{ParentGroupId: invalidGroup.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "no such security group 1")
}

func (s *NovaSuite) TestAddGroupSecurityGroupRuleWithInvalidSourceFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	invalidGroupId := 42
	ri := nova.RuleInfo{
		ParentGroupId: group.Id,
		GroupId:       &invalidGroupId,
	}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, ErrorMatches, "unknown source security group 42")
}

func (s *NovaSuite) TestAddSecurityGroupRuleUpdatesParent(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10, ParentGroupId: group.Id}
	s.ensureNoRule(c, rule)
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	defer s.deleteRule(c, rule)
	group.Rules = []nova.SecurityGroupRule{rule}
	gr, err := s.service.securityGroup(group.Id)
	c.Assert(err, IsNil)
	c.Assert(*gr, DeepEquals, group)
}

func (s *NovaSuite) TestRemoveSecurityGroupRuleTwiceFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	ri := nova.RuleInfo{ParentGroupId: group.Id}
	rule := nova.SecurityGroupRule{Id: 10}
	s.ensureNoRule(c, rule)
	err := s.service.addSecurityGroupRule(rule.Id, ri)
	c.Assert(err, IsNil)
	s.deleteRule(c, rule)
	err = s.service.removeSecurityGroupRule(rule.Id)
	c.Assert(err, ErrorMatches, "no such security group rule 10")
}

func (s *NovaSuite) TestAddHasRemoveServerSecurityGroup(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.ensureNoServer(c, server)
	s.ensureNoGroup(c, group)
	ok := s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	err := s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, true)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaSuite) TestAddServerSecurityGroupWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.ensureNoServer(c, server)
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	err := s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
}

func (s *NovaSuite) TestAddServerSecurityGroupWithInvalidGroupFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.ensureNoGroup(c, group)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	err := s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
}

func (s *NovaSuite) TestAddServerSecurityGroupTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	err := s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	err = s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `server "sr1" already belongs to group 1`)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) TestAllServerSecurityGroups(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	srvGroups := s.service.allServerSecurityGroups(server.Id)
	c.Assert(srvGroups, HasLen, 0)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	groups := []nova.SecurityGroup{
		nova.SecurityGroup{Id: 1, Name: "gr1"},
		nova.SecurityGroup{Id: 2, Name: "gr2"},
	}
	for _, group := range groups {
		s.createGroup(c, group)
		defer s.deleteGroup(c, group)
		err := s.service.addServerSecurityGroup(server.Id, group.Id)
		defer s.service.removeServerSecurityGroup(server.Id, group.Id)
		c.Assert(err, IsNil)
	}
	srvGroups = s.service.allServerSecurityGroups(server.Id)
	c.Assert(srvGroups, HasLen, len(groups))
	if srvGroups[0].Id != groups[0].Id {
		srvGroups[0], srvGroups[1] = srvGroups[1], srvGroups[0]
	}
	c.Assert(srvGroups, DeepEquals, groups)
}

func (s *NovaSuite) TestRemoveServerSecurityGroupWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.createServer(c, server)
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	err := s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	s.deleteServer(c, server)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) TestRemoveServerSecurityGroupWithInvalidGroupFails(c *C) {
	group := nova.SecurityGroup{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.createGroup(c, group)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	err := s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	s.deleteGroup(c, group)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) TestRemoveServerSecurityGroupTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	group := nova.SecurityGroup{Id: 1}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	s.createGroup(c, group)
	defer s.deleteGroup(c, group)
	err := s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, ErrorMatches, `server "sr1" does not belong to group 1`)
}

func (s *NovaSuite) TestAddHasRemoveFloatingIP(c *C) {
	ip := nova.FloatingIP{Id: 1, IP: "1.2.3.4"}
	s.ensureNoIP(c, ip)
	ok := s.service.hasFloatingIP(ip.IP)
	c.Assert(ok, Equals, false)
	s.createIP(c, ip)
	ok = s.service.hasFloatingIP("invalid IP")
	c.Assert(ok, Equals, false)
	ok = s.service.hasFloatingIP(ip.IP)
	c.Assert(ok, Equals, true)
	s.deleteIP(c, ip)
	ok = s.service.hasFloatingIP(ip.IP)
	c.Assert(ok, Equals, false)
}

func (s *NovaSuite) TestAddFloatingIPTwiceFails(c *C) {
	ip := nova.FloatingIP{Id: 1}
	s.createIP(c, ip)
	defer s.deleteIP(c, ip)
	err := s.service.addFloatingIP(ip)
	c.Assert(err, ErrorMatches, "a floating IP with id 1 already exists")
}

func (s *NovaSuite) TestRemoveFloatingIPTwiceFails(c *C) {
	ip := nova.FloatingIP{Id: 1}
	s.createIP(c, ip)
	s.deleteIP(c, ip)
	err := s.service.removeFloatingIP(ip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
}

func (s *NovaSuite) TestAllFloatingIPs(c *C) {
	fips := s.service.allFloatingIPs()
	c.Assert(fips, HasLen, 0)
	fips = []nova.FloatingIP{
		nova.FloatingIP{Id: 1},
		nova.FloatingIP{Id: 2},
	}
	s.createIP(c, fips[0])
	defer s.deleteIP(c, fips[0])
	s.createIP(c, fips[1])
	defer s.deleteIP(c, fips[1])
	ips := s.service.allFloatingIPs()
	c.Assert(ips, HasLen, len(fips))
	if ips[0].Id != fips[0].Id {
		ips[0], ips[1] = ips[1], ips[0]
	}
	c.Assert(ips, DeepEquals, fips)
}

func (s *NovaSuite) TestGetFloatingIP(c *C) {
	fip := nova.FloatingIP{
		Id:         1,
		IP:         "1.2.3.4",
		Pool:       "pool",
		InstanceId: "sr1",
		FixedIP:    "4.3.2.1",
	}
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	ip, _ := s.service.floatingIP(fip.Id)
	c.Assert(*ip, DeepEquals, fip)
}

func (s *NovaSuite) TestGetFloatingIPByAddr(c *C) {
	fip := nova.FloatingIP{Id: 1, IP: "1.2.3.4"}
	s.ensureNoIP(c, fip)
	ip, err := s.service.floatingIPByAddr(fip.IP)
	c.Assert(err, NotNil)
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	ip, err = s.service.floatingIPByAddr(fip.IP)
	c.Assert(err, IsNil)
	c.Assert(*ip, DeepEquals, fip)
	_, err = s.service.floatingIPByAddr("invalid")
	c.Assert(err, ErrorMatches, `no such floating IP with address "invalid"`)
}

func (s *NovaSuite) TestAddHasRemoveServerFloatingIP(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1, IP: "1.2.3.4"}
	s.ensureNoServer(c, server)
	s.ensureNoIP(c, fip)
	ok := s.service.hasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	ok = s.service.hasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	ok = s.service.hasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	err := s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	ok = s.service.hasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, true)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	ok = s.service.hasServerFloatingIP(server.Id, fip.IP)
	c.Assert(ok, Equals, false)
}

func (s *NovaSuite) TestAddServerFloatingIPWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.ensureNoServer(c, server)
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	err := s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
}

func (s *NovaSuite) TestAddServerFloatingIPWithInvalidIPFails(c *C) {
	fip := nova.FloatingIP{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.ensureNoIP(c, fip)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	err := s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
}

func (s *NovaSuite) TestAddServerFloatingIPTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	err := s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	err = s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `server "sr1" already has floating IP 1`)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) TestRemoveServerFloatingIPWithInvalidServerFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.createServer(c, server)
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	err := s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	s.deleteServer(c, server)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `no such server "sr1"`)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) TestRemoveServerFloatingIPWithInvalidIPFails(c *C) {
	fip := nova.FloatingIP{Id: 1}
	server := nova.ServerDetail{Id: "sr1"}
	s.createIP(c, fip)
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	err := s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	s.deleteIP(c, fip)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaSuite) TestRemoveServerFloatingIPTwiceFails(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	fip := nova.FloatingIP{Id: 1}
	s.createServer(c, server)
	defer s.deleteServer(c, server)
	s.createIP(c, fip)
	defer s.deleteIP(c, fip)
	err := s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `server "sr1" does not have floating IP 1`)
}
