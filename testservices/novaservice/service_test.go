// Nova double testing service - internal direct API tests

package novaservice

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/nova"
)

type NovaServiceSuite struct {
	service NovaService
}

var baseURL = "/v2/"
var token = "token"
var hostname = "localhost" // not really used here

var _ = Suite(&NovaServiceSuite{})

func (s *NovaServiceSuite) SetUpSuite(c *C) {
	s.service = New(hostname, baseURL, token)
}

func (s *NovaServiceSuite) TestAddHasRemoveFlavor(c *C) {
	flavorId := "test"
	flavor := Flavor{}
	ok := s.service.HasFlavor(flavorId)
	c.Assert(ok, Equals, false)
	err := s.service.RemoveFlavor(flavorId)
	c.Assert(err, ErrorMatches, `no such flavor "test"`)
	err = s.service.AddFlavor(flavor)
	c.Assert(err, ErrorMatches, "refusing to add a nil flavor")
	flavor.entity = &nova.Entity{Id: flavorId}
	err = s.service.AddFlavor(flavor)
	c.Assert(err, IsNil)
	ok = s.service.HasFlavor(flavorId)
	c.Assert(ok, Equals, true)
	err = s.service.RemoveFlavor(flavorId)
	c.Assert(err, IsNil)
	flavor = Flavor{detail: &nova.FlavorDetail{Id: flavorId}}
	err = s.service.AddFlavor(flavor)
	c.Assert(err, IsNil)
	err = s.service.AddFlavor(flavor)
	c.Assert(err, ErrorMatches, `a flavor with id "test" already exists`)
	err = s.service.RemoveFlavor(flavorId)
	c.Assert(err, IsNil)
	ok = s.service.HasFlavor(flavorId)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestGetAllFlavors(c *C) {
	flavor1 := Flavor{entity: &nova.Entity{Id: "fl1"}}
	flavor2 := Flavor{detail: &nova.FlavorDetail{Id: "fl2"}}
	flavors, err := s.service.AllFlavors()
	c.Assert(flavors, IsNil)
	c.Assert(err, ErrorMatches, "no flavors to return")
	err = s.service.AddFlavor(flavor1)
	c.Assert(err, IsNil)
	flavor, err := s.service.GetFlavor(flavor1.entity.Id)
	c.Assert(err, IsNil)
	c.Assert(flavor, DeepEquals, flavor1)
	_, err = s.service.GetFlavor(flavor2.detail.Id)
	c.Assert(err, ErrorMatches, `no such flavor "fl2"`)
	err = s.service.AddFlavor(flavor2)
	c.Assert(err, IsNil)
	flavors, err = s.service.AllFlavors()
	c.Assert(err, IsNil)
	c.Assert(flavors, HasLen, 2)
	expectedFlavors := []Flavor{flavor1, flavor2}
	if flavors[0].entity == nil {
		expectedFlavors[1], expectedFlavors[0] = expectedFlavors[0], expectedFlavors[1]
	}
	c.Assert(expectedFlavors, DeepEquals, flavors)
	err = s.service.RemoveFlavor(flavor1.entity.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveFlavor(flavor2.detail.Id)
	c.Assert(err, IsNil)
	flavors, err = s.service.AllFlavors()
	c.Assert(err, ErrorMatches, "no flavors to return")
	c.Assert(flavors, IsNil)
}

func (s *NovaServiceSuite) TestAddHasRemoveServer(c *C) {
	serverId := "test"
	server := Server{}
	ok := s.service.HasServer(serverId)
	c.Assert(ok, Equals, false)
	err := s.service.RemoveServer(serverId)
	c.Assert(err, ErrorMatches, `no such server "test"`)
	err = s.service.AddServer(server)
	c.Assert(err, ErrorMatches, "refusing to add a nil server")
	server.server = &nova.Entity{Id: serverId}
	err = s.service.AddServer(server)
	c.Assert(err, IsNil)
	ok = s.service.HasServer(serverId)
	c.Assert(ok, Equals, true)
	err = s.service.RemoveServer(serverId)
	c.Assert(err, IsNil)
	server = Server{detail: &nova.ServerDetail{Id: serverId}}
	err = s.service.AddServer(server)
	c.Assert(err, IsNil)
	err = s.service.AddServer(server)
	c.Assert(err, ErrorMatches, `a server with id "test" already exists`)
	err = s.service.RemoveServer(serverId)
	c.Assert(err, IsNil)
	ok = s.service.HasServer(serverId)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestGetAllServers(c *C) {
	server1 := Server{server: &nova.Entity{Id: "srv1"}}
	server2 := Server{detail: &nova.ServerDetail{Id: "srv2"}}
	servers, err := s.service.AllServers()
	c.Assert(servers, IsNil)
	c.Assert(err, ErrorMatches, "no servers to return")
	err = s.service.AddServer(server1)
	c.Assert(err, IsNil)
	server, err := s.service.GetServer(server1.server.Id)
	c.Assert(err, IsNil)
	c.Assert(server, DeepEquals, server1)
	_, err = s.service.GetServer(server2.detail.Id)
	c.Assert(err, ErrorMatches, `no such server "srv2"`)
	err = s.service.AddServer(server2)
	c.Assert(err, IsNil)
	servers, err = s.service.AllServers()
	c.Assert(err, IsNil)
	c.Assert(servers, HasLen, 2)
	expectedServers := []Server{server1, server2}
	if servers[0].server == nil {
		expectedServers[1], expectedServers[0] = expectedServers[0], expectedServers[1]
	}
	c.Assert(expectedServers, DeepEquals, servers)
	err = s.service.RemoveServer(server1.server.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveServer(server2.detail.Id)
	c.Assert(err, IsNil)
	servers, err = s.service.AllServers()
	c.Assert(err, ErrorMatches, "no servers to return")
	c.Assert(servers, IsNil)
}

func (s *NovaServiceSuite) TestAddHasRemoveSecurityGroup(c *C) {
	group := nova.SecurityGroup{Id: 1}
	ok := s.service.HasSecurityGroup(group.Id)
	c.Assert(ok, Equals, false)
	err := s.service.RemoveSecurityGroup(group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
	err = s.service.AddSecurityGroup(group)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroup(group.Id)
	c.Assert(ok, Equals, true)
	err = s.service.AddSecurityGroup(group)
	c.Assert(err, ErrorMatches, "a security group with id 1 already exists")
	err = s.service.RemoveSecurityGroup(group.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroup(group.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestGetAllSecurityGroups(c *C) {
	group1 := nova.SecurityGroup{Id: 1, Name: "test", Description: "desc"}
	group2 := nova.SecurityGroup{Id: 2}
	groups, err := s.service.AllSecurityGroups()
	c.Assert(groups, IsNil)
	c.Assert(err, ErrorMatches, "no security groups to return")
	err = s.service.AddSecurityGroup(group1)
	c.Assert(err, IsNil)
	group, err := s.service.GetSecurityGroup(group1.Id)
	c.Assert(err, IsNil)
	c.Assert(group, DeepEquals, group1)
	_, err = s.service.GetSecurityGroup(group2.Id)
	c.Assert(err, ErrorMatches, "no such security group 2")
	err = s.service.AddSecurityGroup(group2)
	c.Assert(err, IsNil)
	groups, err = s.service.AllSecurityGroups()
	c.Assert(err, IsNil)
	c.Assert(groups, HasLen, 2)
	expectedGroups := []nova.SecurityGroup{group1, group2}
	if groups[0].Id == 2 {
		expectedGroups[1], expectedGroups[0] = expectedGroups[0], expectedGroups[1]
	}
	c.Assert(expectedGroups, DeepEquals, groups)
	err = s.service.RemoveSecurityGroup(group1.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveSecurityGroup(group2.Id)
	c.Assert(err, IsNil)
	groups, err = s.service.AllSecurityGroups()
	c.Assert(err, ErrorMatches, "no security groups to return")
	c.Assert(groups, IsNil)
}

func (s *NovaServiceSuite) TestSecurityGroupRules(c *C) {
	group1 := nova.SecurityGroup{Id: 1, Name: "source group"}
	ok := s.service.HasSecurityGroup(group1.Id)
	c.Assert(ok, Equals, false)
	err := s.service.AddSecurityGroup(group1)
	c.Assert(err, IsNil)
	defer s.service.RemoveSecurityGroup(group1.Id)
	group2 := nova.SecurityGroup{Id: 2, Name: "target group"}
	ok = s.service.HasSecurityGroup(group2.Id)
	c.Assert(ok, Equals, false)
	err = s.service.AddSecurityGroup(group2)
	c.Assert(err, IsNil)
	defer s.service.RemoveSecurityGroup(group2.Id)

	ruleId := 10
	riIngress := nova.RuleInfo{
		FromPort:      1234,
		ToPort:        4321,
		IPProtocol:    "tcp",
		Cidr:          "1.2.3.4/5",
		ParentGroupId: group1.Id,
	}
	riGroup := nova.RuleInfo{
		GroupId:       &group1.Id,
		ParentGroupId: group2.Id,
	}
	ok = s.service.HasSecurityGroupRule(group1.Id, ruleId)
	c.Assert(ok, Equals, false)
	err = s.service.RemoveSecurityGroupRule(ruleId)
	c.Assert(err, ErrorMatches, "no such security group rule 10")
	_, err = s.service.GetSecurityGroupRule(ruleId)
	c.Assert(err, ErrorMatches, "no such security group rule 10")
	err = s.service.AddSecurityGroupRule(ruleId, riIngress)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroupRule(group1.Id, ruleId)
	c.Assert(ok, Equals, true)
	err = s.service.AddSecurityGroupRule(ruleId, riIngress)
	c.Assert(err, ErrorMatches, "a security group rule with id 10 already exists")
	rule, err := s.service.GetSecurityGroupRule(ruleId)
	c.Assert(err, IsNil)
	c.Assert(rule.Id, Equals, ruleId)
	c.Assert(rule.ParentGroupId, Equals, riIngress.ParentGroupId)
	c.Assert(rule.FromPort, Not(IsNil))
	c.Assert(rule.ToPort, Not(IsNil))
	c.Assert(rule.IPProtocol, Not(IsNil))
	c.Assert(rule.IPRange, Not(IsNil))
	c.Assert(*rule.FromPort, Equals, riIngress.FromPort)
	c.Assert(*rule.ToPort, Equals, riIngress.ToPort)
	c.Assert(*rule.IPProtocol, Equals, riIngress.IPProtocol)
	c.Assert(rule.IPRange["cidr"], Equals, riIngress.Cidr)
	c.Assert(rule.Group, IsNil)
	err = s.service.RemoveSecurityGroupRule(ruleId)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroupRule(group2.Id, ruleId)
	c.Assert(ok, Equals, false)
	err = s.service.AddSecurityGroupRule(ruleId, riGroup)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroupRule(group2.Id, ruleId)
	c.Assert(ok, Equals, true)
	rule, err = s.service.GetSecurityGroupRule(ruleId)
	c.Assert(err, IsNil)
	c.Assert(rule.Id, Equals, ruleId)
	c.Assert(rule.ParentGroupId, Equals, riGroup.ParentGroupId)
	c.Assert(rule.Group, Not(IsNil))
	c.Assert(rule.Group.Name, Equals, group1.Name)
	c.Assert(rule.FromPort, IsNil)
	c.Assert(rule.ToPort, IsNil)
	c.Assert(rule.IPProtocol, IsNil)
	c.Assert(rule.IPRange, IsNil)
	err = s.service.RemoveSecurityGroupRule(ruleId)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroupRule(group2.Id, ruleId)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestServerSecurityGroups(c *C) {
	server := Server{server: &nova.Entity{Id: "server"}}
	group := nova.SecurityGroup{Id: 1, Name: "group"}
	group2 := nova.SecurityGroup{Id: 2, Name: "group2"}
	invalidGroupId := 42
	ok := s.service.HasServer(server.server.Id)
	c.Assert(ok, Equals, false)
	ok = s.service.HasServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(ok, Equals, false)
	err := s.service.RemoveServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, ErrorMatches, `no such server "server"`)
	err = s.service.AddServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, ErrorMatches, `no such server "server"`)
	err = s.service.AddServer(server)
	c.Assert(err, IsNil)
	defer s.service.RemoveServer(server.server.Id)
	err = s.service.AddServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")
	err = s.service.RemoveServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, ErrorMatches, "no such security group 1")

	ok = s.service.HasServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(ok, Equals, false)
	ok = s.service.HasSecurityGroup(group.Id)
	c.Assert(ok, Equals, false)
	err = s.service.AddSecurityGroup(group)
	c.Assert(err, IsNil)
	defer s.service.RemoveSecurityGroup(group.Id)
	ok = s.service.HasSecurityGroup(group2.Id)
	c.Assert(ok, Equals, false)
	err = s.service.AddSecurityGroup(group2)
	c.Assert(err, IsNil)
	ok = s.service.HasSecurityGroup(group2.Id)
	c.Assert(ok, Equals, true)
	defer s.service.RemoveSecurityGroup(group2.Id)
	err = s.service.RemoveServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, ErrorMatches, `server "server" does not belong to any groups`)
	ok = s.service.HasServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(ok, Equals, false)
	err = s.service.AddServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, IsNil)
	err = s.service.AddServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, ErrorMatches, `server "server" already belongs to group 1`)
	ok = s.service.HasServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(ok, Equals, true)
	err = s.service.RemoveServerSecurityGroup(server.server.Id, group2.Id)
	c.Assert(err, ErrorMatches, `server "server" does not belong to group 2`)
	err = s.service.RemoveServerSecurityGroup(server.server.Id, invalidGroupId)
	c.Assert(err, ErrorMatches, "no such security group 42")
	err = s.service.RemoveServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasServerSecurityGroup(server.server.Id, group.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestAddHasRemoveFloatingIP(c *C) {
	fip := nova.FloatingIP{Id: 1, IP: "1.2.3.4", Pool: "pool"}
	ok := s.service.HasFloatingIP(fip.IP)
	c.Assert(ok, Equals, false)
	err := s.service.RemoveFloatingIP(fip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
	err = s.service.AddFloatingIP(fip)
	c.Assert(err, IsNil)
	ok = s.service.HasFloatingIP(fip.IP)
	c.Assert(ok, Equals, true)
	err = s.service.AddFloatingIP(fip)
	c.Assert(err, ErrorMatches, "a floating IP with id 1 already exists")
	err = s.service.RemoveFloatingIP(fip.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasFloatingIP(fip.IP)
	c.Assert(ok, Equals, false)
}

func (s *NovaServiceSuite) TestGetAllFloatingIPs(c *C) {
	fip1 := nova.FloatingIP{Id: 1, IP: "1.2.3.4", Pool: "pool"}
	fip2 := nova.FloatingIP{Id: 2, IP: "4.3.2.1", Pool: "pool"}
	fips, err := s.service.AllFlotingIPs()
	c.Assert(fips, IsNil)
	c.Assert(err, ErrorMatches, "no floating IPs to return")
	err = s.service.AddFloatingIP(fip1)
	c.Assert(err, IsNil)
	fip, err := s.service.GetFloatingIP(fip1.Id)
	c.Assert(err, IsNil)
	c.Assert(fip, DeepEquals, fip1)
	_, err = s.service.GetFloatingIP(fip2.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 2")
	err = s.service.AddFloatingIP(fip2)
	c.Assert(err, IsNil)
	fips, err = s.service.AllFlotingIPs()
	c.Assert(err, IsNil)
	c.Assert(fips, HasLen, 2)
	expectedIPs := []nova.FloatingIP{fip1, fip2}
	if fips[0].Id == 2 {
		expectedIPs[1], expectedIPs[0] = expectedIPs[0], expectedIPs[1]
	}
	c.Assert(expectedIPs, DeepEquals, fips)
	err = s.service.RemoveFloatingIP(fip1.Id)
	c.Assert(err, IsNil)
	err = s.service.RemoveFloatingIP(fip2.Id)
	c.Assert(err, IsNil)
	fips, err = s.service.AllFlotingIPs()
	c.Assert(err, ErrorMatches, "no floating IPs to return")
	c.Assert(fips, IsNil)
}

func (s *NovaServiceSuite) TestServerFloatingIPs(c *C) {
	server := Server{server: &nova.Entity{Id: "server"}}
	fip := nova.FloatingIP{Id: 1, IP: "1.2.3.4", Pool: "pool"}
	fip2 := nova.FloatingIP{Id: 2, IP: "4.3.2.1", Pool: "pool"}
	invalidIPId := 42
	ok := s.service.HasServer(server.server.Id)
	c.Assert(ok, Equals, false)
	ok = s.service.HasServerFloatingIP(server.server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	err := s.service.RemoveServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `no such server "server"`)
	err = s.service.AddServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `no such server "server"`)
	err = s.service.AddServer(server)
	c.Assert(err, IsNil)
	defer s.service.RemoveServer(server.server.Id)
	err = s.service.AddServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")
	err = s.service.RemoveServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, ErrorMatches, "no such floating IP 1")

	ok = s.service.HasServerFloatingIP(server.server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	ok = s.service.HasFloatingIP(fip.IP)
	c.Assert(ok, Equals, false)
	err = s.service.AddFloatingIP(fip)
	c.Assert(err, IsNil)
	defer s.service.RemoveFloatingIP(fip.Id)
	ok = s.service.HasFloatingIP(fip2.IP)
	c.Assert(ok, Equals, false)
	err = s.service.AddFloatingIP(fip2)
	c.Assert(err, IsNil)
	ok = s.service.HasFloatingIP(fip2.IP)
	c.Assert(ok, Equals, true)
	defer s.service.RemoveFloatingIP(fip2.Id)
	err = s.service.RemoveServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `server "server" does not have any floating IPs to remove`)
	ok = s.service.HasServerFloatingIP(server.server.Id, fip.IP)
	c.Assert(ok, Equals, false)
	err = s.service.AddServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, IsNil)
	err = s.service.AddServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, ErrorMatches, `server "server" already has floating IP 1`)
	ok = s.service.HasServerFloatingIP(server.server.Id, fip.IP)
	c.Assert(ok, Equals, true)
	err = s.service.RemoveServerFloatingIP(server.server.Id, fip2.Id)
	c.Assert(err, ErrorMatches, `server "server" does not have floating IP 2`)
	err = s.service.RemoveServerFloatingIP(server.server.Id, invalidIPId)
	c.Assert(err, ErrorMatches, "no such floating IP 42")
	err = s.service.RemoveServerFloatingIP(server.server.Id, fip.Id)
	c.Assert(err, IsNil)
	ok = s.service.HasServerFloatingIP(server.server.Id, fip.IP)
	c.Assert(ok, Equals, false)
}
