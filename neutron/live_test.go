package neutron_test

import (
	"net"

	gc "gopkg.in/check.v1"

	"github.com/go-goose/goose/v5/client"
	"github.com/go-goose/goose/v5/identity"
	"github.com/go-goose/goose/v5/neutron"
)

func registerOpenStackTests(cred *identity.Credentials) {
	gc.Suite(&LiveTests{
		cred: cred,
	})
}

type LiveTests struct {
	cred     *identity.Credentials
	client   client.AuthenticatingClient
	neutron  *neutron.Client
	userId   string
	tenantId string
}

func (s *LiveTests) SetUpSuite(c *gc.C) {
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
	s.neutron = neutron.New(s.client)
}

func (s *LiveTests) TearDownSuite(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) SetUpTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TestFloatingIPsV2(c *gc.C) {
	networks, err := s.neutron.ListNetworksV2()
	c.Assert(err, gc.IsNil)
	var netId string
	for _, net := range networks {
		if net.External == true {
			netId = net.Id
			break
		}
	}
	if netId == "" {
		c.Errorf("no valid network to create floating IP")
	}
	c.Assert(netId, gc.Not(gc.Equals), "")
	ip, err := s.neutron.AllocateFloatingIPV2(netId)
	c.Assert(err, gc.IsNil)
	defer s.neutron.DeleteFloatingIPV2(ip.Id)
	c.Assert(ip, gc.Not(gc.IsNil))
	c.Check(ip.IP, gc.Not(gc.Equals), "")
	c.Check(ip.FixedIP, gc.Equals, "")
	c.Check(ip.Id, gc.Not(gc.Equals), "")
	c.Check(ip.FloatingNetworkId, gc.Not(gc.Equals), "")
	ips, err := s.neutron.ListFloatingIPsV2()
	c.Assert(err, gc.IsNil)
	if len(ips) < 1 {
		c.Errorf("no floating IPs found (expected at least 1)")
	} else {
		found := false
		for _, i := range ips {
			c.Check(i.IP, gc.Not(gc.Equals), "")
			if i.Id == ip.Id {
				c.Check(i.IP, gc.Equals, ip.IP)
				c.Check(i.FloatingNetworkId, gc.Equals, ip.FloatingNetworkId)
				found = true
			}
		}
		if !found {
			c.Errorf("expected to find added floating IP: %#v", ip)
		}
		fip, err := s.neutron.GetFloatingIPV2(ip.Id)
		c.Assert(err, gc.IsNil)
		c.Check(fip.Id, gc.Equals, ip.Id)
		c.Check(fip.IP, gc.Equals, ip.IP)
		c.Check(fip.FloatingNetworkId, gc.Equals, ip.FloatingNetworkId)
	}
	err = s.neutron.DeleteFloatingIPV2(ip.Id)
	c.Assert(err, gc.IsNil)
	_, err = s.neutron.GetFloatingIPV2(ip.Id)
	c.Assert(err, gc.Not(gc.IsNil))
}

// For the purposes of this test, project_id and tenant_id are
// interchangeable.
func (s *LiveTests) TestFloatingIPsV2WithFilter(c *gc.C) {
	filter := neutron.NewFilter()
	filter.Set(neutron.FilterRouterExternal, "true")
	networks, err := s.neutron.ListNetworksV2(filter)
	c.Assert(err, gc.IsNil)
	if len(networks) < 2 {
		c.Errorf("at least two external neutron networks are necessary for this test")
	}

	network0 := networks[0]
	c.Assert(network0, gc.Not(gc.Equals), "")
	network1 := networks[1]
	c.Assert(network1, gc.Not(gc.Equals), "")
	c.Assert(network0.TenantId, gc.Not(gc.Equals), network1.TenantId)

	fipNetworkOne, err := s.neutron.AllocateFloatingIPV2(network0.Id)
	c.Assert(err, gc.IsNil)
	defer s.neutron.DeleteFloatingIPV2(fipNetworkOne.Id)
	c.Assert(fipNetworkOne, gc.Not(gc.IsNil))
	c.Check(fipNetworkOne.FloatingNetworkId, gc.Equals, network0.Id)

	fipNetworkTwo, err := s.neutron.AllocateFloatingIPV2(network1.Id)
	c.Assert(err, gc.IsNil)
	defer s.neutron.DeleteFloatingIPV2(fipNetworkTwo.Id)
	c.Assert(fipNetworkTwo, gc.Not(gc.IsNil))
	c.Check(fipNetworkTwo.FloatingNetworkId, gc.Equals, network1.Id)

	filter = neutron.NewFilter()
	filter.Set(neutron.FilterProjectId, network1.TenantId)
	ips, err := s.neutron.ListFloatingIPsV2(filter)
	c.Assert(err, gc.IsNil)
	c.Assert(ips, gc.HasLen, 1)
	c.Assert(ips[0].FloatingNetworkId, gc.Equals, network1.Id)
}

func (s *LiveTests) TestListNetworksV2(c *gc.C) {
	networks, err := s.neutron.ListNetworksV2()
	c.Assert(err, gc.IsNil)
	if len(networks) < 1 {
		c.Errorf("at least one neutron network is necessary for this tests")
	}
	for _, network := range networks {
		c.Check(network.Id, gc.Not(gc.Equals), "")
		c.Check(network.Name, gc.Not(gc.Equals), "")
	}
	firstNetwork := networks[0]
	foundNetwork, err := s.neutron.GetNetworkV2(firstNetwork.Id)
	c.Assert(err, gc.IsNil)
	c.Check(foundNetwork.Id, gc.Equals, firstNetwork.Id)
	c.Check(foundNetwork.Name, gc.Equals, firstNetwork.Name)
}

func (s *LiveTests) TestListNetworksV2WithFilters(c *gc.C) {
	filter := neutron.NewFilter()
	filter.Set(neutron.FilterRouterExternal, "true")
	networks, err := s.neutron.ListNetworksV2(filter)
	c.Assert(err, gc.IsNil)
	if len(networks) < 1 {
		c.Errorf("at least one neutron network is necessary for this tests")
	}
	for _, network := range networks {
		c.Check(network.Id, gc.Not(gc.Equals), "")
		c.Check(network.Name, gc.Not(gc.Equals), "")
		c.Check(network.External, gc.Equals, true)
	}
	firstNetwork := networks[0]
	foundNetwork, err := s.neutron.GetNetworkV2(firstNetwork.Id)
	c.Assert(err, gc.IsNil)
	c.Check(foundNetwork.Id, gc.Equals, firstNetwork.Id)
	c.Check(foundNetwork.Name, gc.Equals, firstNetwork.Name)
}

func (s *LiveTests) TestSubnetsV2(c *gc.C) {
	subnets, err := s.neutron.ListSubnetsV2()
	c.Assert(err, gc.IsNil)
	if len(subnets) < 1 {
		c.Errorf("at least one neutron subnet is necessary for this tests")
	}
	for _, subnet := range subnets {
		c.Check(subnet.Id, gc.Not(gc.Equals), "")
		c.Check(subnet.NetworkId, gc.Not(gc.Equals), "")
		c.Check(subnet.Name, gc.Not(gc.Equals), "")
		_, _, err := net.ParseCIDR(subnet.Cidr)
		c.Assert(err, gc.IsNil)
	}
	firstSubnet := subnets[0]
	foundSubnet, err := s.neutron.GetSubnetV2(firstSubnet.Id)
	c.Assert(err, gc.IsNil)
	c.Check(foundSubnet.Id, gc.Equals, firstSubnet.Id)
	c.Check(foundSubnet.NetworkId, gc.Equals, firstSubnet.NetworkId)
	c.Check(foundSubnet.Name, gc.Equals, firstSubnet.Name)
}

func (s *LiveTests) deleteSecurityGroup(id string, c *gc.C) {
	err := s.neutron.DeleteSecurityGroupV2(id)
	c.Assert(err, gc.IsNil)
}

func (s *LiveTests) TestSecurityGroupsV2(c *gc.C) {
	newSecGrp, err := s.neutron.CreateSecurityGroupV2("SecurityGroupTest", "Testing create security group")
	c.Assert(err, gc.IsNil)
	c.Assert(newSecGrp, gc.Not(gc.IsNil))
	defer s.deleteSecurityGroup(newSecGrp.Id, c)
	secGrps, err := s.neutron.ListSecurityGroupsV2()
	c.Assert(err, gc.IsNil)
	c.Assert(secGrps, gc.Not(gc.HasLen), 0)
	var found bool
	for _, secGrp := range secGrps {
		c.Check(secGrp.Id, gc.Not(gc.Equals), "")
		c.Check(secGrp.Name, gc.Not(gc.Equals), "")
		c.Check(secGrp.Description, gc.Not(gc.Equals), "")
		c.Check(secGrp.TenantId, gc.Not(gc.Equals), "")
		// Is this the SecurityGroup we just created?
		if secGrp.Id == newSecGrp.Id {
			found = true
		}
	}
	if !found {
		c.Errorf("expected to find added security group %s", newSecGrp)
	}
	// Change the created SecurityGroup's name
	updatedSecGroup, err := s.neutron.UpdateSecurityGroupV2(newSecGrp.Id, "NameChanged", "")
	c.Assert(err, gc.IsNil)
	// Verify the name change
	foundSecGrps, err := s.neutron.SecurityGroupByNameV2(updatedSecGroup.Name)
	c.Assert(err, gc.IsNil)
	c.Assert(foundSecGrps, gc.Not(gc.HasLen), 0)
	found = false
	for _, secGrp := range foundSecGrps {
		if secGrp.Id == updatedSecGroup.Id {
			found = true
			break
		}
	}
	if !found {
		c.Errorf("expected to find added security group %s, when requested by name", updatedSecGroup.Name)
	}
	_, err = s.neutron.SecurityGroupByNameV2(newSecGrp.Name)
	c.Assert(err, gc.Not(gc.IsNil))
}

func (s *LiveTests) TestSecurityGroupsByNameV2(c *gc.C) {
	// Create and find a SecurityGroup
	newSecGrp, err := s.neutron.CreateSecurityGroupV2("SecurityGroupTest", "Testing find security group by name")
	c.Assert(err, gc.IsNil)
	defer s.deleteSecurityGroup(newSecGrp.Id, c)
	c.Assert(newSecGrp, gc.Not(gc.IsNil))
	foundSecGrps, err := s.neutron.SecurityGroupByNameV2(newSecGrp.Name)
	c.Assert(err, gc.IsNil)
	c.Assert(foundSecGrps, gc.HasLen, 1)
	if newSecGrp.Id != foundSecGrps[0].Id {
		c.Errorf("expected to find added security group %s, when requested by name", newSecGrp.Name)
	}
	// Try to find a SecurityGroup that doesn't exist
	errorSecGrps, err := s.neutron.SecurityGroupByNameV2("NonExistentGroup")
	c.Assert(err, gc.Not(gc.IsNil))
	c.Assert(errorSecGrps, gc.HasLen, 0)
	// Create and find a SecurityGroup with spaces in the name
	newSecGrp2, err := s.neutron.CreateSecurityGroupV2("Security Group Test", "Testing find security group by name")
	c.Assert(err, gc.IsNil)
	defer s.deleteSecurityGroup(newSecGrp2.Id, c)
	c.Assert(newSecGrp2, gc.Not(gc.IsNil))
	foundSecGrps2, err := s.neutron.SecurityGroupByNameV2(newSecGrp2.Name)
	c.Assert(err, gc.IsNil)
	c.Assert(foundSecGrps2, gc.HasLen, 1)
	if newSecGrp2.Id != foundSecGrps2[0].Id {
		c.Errorf("expected to find added security group %s, when requested by name", newSecGrp2.Name)
	}
	// Create a second SecurityGroup with the same name as one already created,
	// find both.
	newSecGrp3, err := s.neutron.CreateSecurityGroupV2(newSecGrp.Name, "Testing find security group by name, 2nd")
	c.Assert(err, gc.IsNil)
	defer s.deleteSecurityGroup(newSecGrp3.Id, c)
	c.Assert(newSecGrp3, gc.Not(gc.IsNil))
	foundSecGrps3, err := s.neutron.SecurityGroupByNameV2(newSecGrp.Name)
	c.Assert(err, gc.IsNil)
	c.Assert(foundSecGrps3, gc.HasLen, 2)
}

func (s *LiveTests) TestSecurityGroupsRulesV2(c *gc.C) {
	newSecGrp, err := s.neutron.CreateSecurityGroupV2("SecurityGroupTestRules", "Testing create security group")
	c.Assert(err, gc.IsNil)
	defer s.deleteSecurityGroup(newSecGrp.Id, c)
	rule := neutron.RuleInfoV2{
		ParentGroupId:  newSecGrp.Id,
		RemoteIPPrefix: "0.0.0.0/0",
		IPProtocol:     "icmp",
		Direction:      "ingress",
		EthernetType:   "IPv4",
	}
	newSecGrpRule, err := s.neutron.CreateSecurityGroupRuleV2(rule)
	c.Assert(err, gc.IsNil)
	c.Assert(newSecGrp.Id, gc.Equals, newSecGrpRule.ParentGroupId)
	c.Assert(*newSecGrpRule.IPProtocol, gc.Equals, rule.IPProtocol)
	c.Assert(newSecGrpRule.Direction, gc.Equals, rule.Direction)

	secGrps, err := s.neutron.SecurityGroupByNameV2(newSecGrp.Name)
	c.Assert(err, gc.IsNil)
	c.Assert(secGrps, gc.Not(gc.HasLen), 0)
	var found bool
	for _, secGrp := range secGrps {
		if secGrp.Id == newSecGrp.Id {
			for _, secGrpRule := range secGrp.Rules {
				if newSecGrpRule.Id == secGrpRule.Id {
					found = true
				}
			}
		}
	}
	if !found {
		c.Errorf("expected to find added security group rule %s", newSecGrpRule.Id)
	}
	err = s.neutron.DeleteSecurityGroupRuleV2(newSecGrpRule.Id)
	c.Assert(err, gc.IsNil)
}

func (s *LiveTests) deletePort(id string, c *gc.C) {
	err := s.neutron.DeletePortV2(id)
	c.Assert(err, gc.IsNil)
}

func (s *LiveTests) TestPortsV2(c *gc.C) {
	port := neutron.PortV2{
		Name:        "PortTest",
		Description: "Testing create port",
		NetworkId:   "a87cc70a-3e15-4acf-8205-9b711a3531b7",
		Tags:        []string{"tag0", "tag1"},
	}
	newPort, err := s.neutron.CreatePortV2(port)
	c.Assert(err, gc.IsNil)
	defer s.deletePort(newPort.Id, c)
	c.Assert(newPort, gc.Not(gc.IsNil))

	ports, err := s.neutron.ListPortsV2()
	c.Assert(err, gc.IsNil)
	c.Assert(ports, gc.Not(gc.HasLen), 0)

	var found bool
	for _, port := range ports {
		c.Check(port.Id, gc.Not(gc.Equals), "")
		c.Check(port.Name, gc.Not(gc.Equals), "")
		c.Check(port.Description, gc.Not(gc.Equals), "")
		c.Check(port.TenantId, gc.Not(gc.Equals), "")
		c.Check(port.NetworkId, gc.Not(gc.Equals), "")
		c.Check(port.Tags, gc.HasLen, 2)
		// Is this the Port we just created?
		if port.Id == newPort.Id {
			found = true
		}
	}
	if !found {
		c.Errorf("expected to find added port %s", newPort)
	}

	port1 := ports[0]

	filter := neutron.NewFilter()
	filter.Set(neutron.FilterProjectId, port1.TenantId)
	ports, err = s.neutron.ListPortsV2(filter)
	c.Assert(err, gc.IsNil)
	c.Assert(ports, gc.HasLen, 1)
	c.Assert(ports[0].Name, gc.Equals, port.Name)
	c.Assert(ports[0].Tags, gc.DeepEquals, port.Tags)
}

func (s *LiveTests) TestPortByIdV2(c *gc.C) {
	// Create and find a Port
	port := neutron.PortV2{
		Name:        "PortTest",
		Description: "Testing create port",
		NetworkId:   "a87cc70a-3e15-4acf-8205-9b711a3531b7",
		Tags:        []string{"tag0", "tag1"},
	}
	newPort, err := s.neutron.CreatePortV2(port)
	c.Assert(err, gc.IsNil)
	defer s.deletePort(newPort.Id, c)
	c.Assert(newPort, gc.Not(gc.IsNil))

	foundPort, err := s.neutron.PortByIdV2(newPort.Id)
	c.Assert(err, gc.IsNil)
	if newPort.Id != foundPort.Id {
		c.Errorf("expected to find added port %s, when requested by Id", newPort.Id)
	}

	// Try to find a Port that doesn't exist
	_, err = s.neutron.PortByIdV2("xunknown-port-idxx-8205-9b711a3531b7")
	c.Assert(err, gc.Not(gc.IsNil))
}

func (s *LiveTests) TestPortsDeleteV2(c *gc.C) {
	port := neutron.PortV2{
		Name:        "PortTest",
		Description: "Testing create port",
		NetworkId:   "a87cc70a-3e15-4acf-8205-9b711a3531b7",
		Tags:        []string{"tag0", "tag1"},
	}
	newPort, err := s.neutron.CreatePortV2(port)
	c.Assert(err, gc.IsNil)
	c.Assert(newPort, gc.Not(gc.IsNil))

	err = s.neutron.DeletePortV2(newPort.Id)
	c.Assert(err, gc.IsNil)

	_, err = s.neutron.PortByIdV2(newPort.Id)
	c.Assert(err, gc.Not(gc.IsNil))
}
