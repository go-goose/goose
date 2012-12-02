package nova_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/nova"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type NovaSuite struct {
	nova         *nova.Client
	testServerId string
	userId       string
	tenantId     string
}

func (s *NovaSuite) SetUpSuite(c *C) {
	if !*live {
		c.Skip("-live not provided")
	}

	cred, err := identity.CompleteCredentialsFromEnv()
	c.Assert(err, IsNil)
	client := client.NewClient(cred, identity.AuthUserPass)
	c.Assert(err, IsNil)
	err = client.Authenticate()
	c.Assert(err, IsNil)
	c.Logf("client authenticated")
	s.nova = nova.New(client)
	s.userId = client.UserId
	s.tenantId = client.TenantId
}

var suite = Suite(&NovaSuite{})

func (n *NovaSuite) TestListFlavors(c *C) {
	flavors, err := n.nova.ListFlavors()
	c.Assert(err, IsNil)
	if len(flavors) < 1 {
		c.Fatalf("no flavors to list")
	}
	for _, f := range flavors {
		c.Assert(f.Id, Not(Equals), "")
		c.Assert(f.Name, Not(Equals), "")
		for _, l := range f.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Matches, "self|bookmark")
		}
	}
}

func (n *NovaSuite) TestListFlavorsDetail(c *C) {
	flavors, err := n.nova.ListFlavorsDetail()
	c.Assert(err, IsNil)
	if len(flavors) < 1 {
		c.Fatalf("no flavors (details) to list")
	}
	for _, f := range flavors {
		c.Assert(f.Name, Not(Equals), "")
		c.Assert(f.Id, Not(Equals), "")
		if f.RAM < 0 || f.VCPUs < 0 || f.Disk < 0 {
			c.Fatalf("invalid flavor found: %#v", f)
		}
	}
}

func (n *NovaSuite) TestListServers(c *C) {
	servers, err := n.nova.ListServers()
	c.Assert(err, IsNil)
	foundTest := false
	for _, sr := range servers {
		c.Assert(sr.Id, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		if sr.Id == n.testServerId {
			c.Assert(sr.Name, Equals, "test_server1")
			foundTest = true
		}
		for _, l := range sr.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Matches, "self|bookmark")
		}
	}
	if !foundTest {
		c.Fatalf("test server (%s) not found in server list", n.testServerId)
	}
}

func (n *NovaSuite) TestListServersDetail(c *C) {
	servers, err := n.nova.ListServersDetail()
	c.Assert(err, IsNil)
	if len(servers) < 1 {
		c.Fatalf("no servers to list (expected at least 1)")
	}
	foundTest := false
	for _, sr := range servers {
		c.Assert(sr.Created, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(sr.Updated, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(sr.Id, Not(Equals), "")
		c.Assert(sr.HostId, Not(Equals), "")
		c.Assert(sr.TenantId, Equals, n.tenantId)
		c.Assert(sr.UserId, Equals, n.userId)
		c.Assert(sr.Status, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		if sr.Id == n.testServerId {
			c.Assert(sr.Name, Equals, "test_server1")
			c.Assert(sr.Flavor.Id, Equals, "1")
			c.Assert(sr.Image.Id, Equals, "3fc0ef0b-82a9-4f44-a797-a43f0f73b20e")
			foundTest = true
		}
		for _, l := range sr.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Matches, "self|bookmark")
		}
		c.Assert(sr.Flavor.Id, Not(Equals), "")
		for _, f := range sr.Flavor.Links {
			c.Assert(f.Href, Matches, "https?://.*")
			c.Assert(f.Rel, Matches, "self|bookmark")
		}
		c.Assert(sr.Image.Id, Not(Equals), "")
		for _, i := range sr.Image.Links {
			c.Assert(i.Href, Matches, "https?://.*")
			c.Assert(i.Rel, Matches, "self|bookmark")
		}
	}
	if !foundTest {
		c.Fatalf("test server (%s) not found in server list (details)", n.testServerId)
	}
}

func (n *NovaSuite) TestListSecurityGroups(c *C) {
	groups, err := n.nova.ListSecurityGroups()
	c.Assert(err, IsNil)
	if len(groups) < 1 {
		c.Fatalf("no security groups found (expected at least 1)")
	}
	for _, g := range groups {
		c.Assert(g.TenantId, Equals, n.tenantId)
		c.Assert(g.Name, Not(Equals), "")
		c.Assert(g.Description, Not(Equals), "")
		c.Assert(g.Rules, NotNil)
	}
}

func (n *NovaSuite) TestCreateAndDeleteSecurityGroup(c *C) {
	group, err := n.nova.CreateSecurityGroup("test_secgroup", "test_desc")
	c.Check(err, IsNil)
	c.Check(group.Name, Equals, "test_secgroup")
	c.Check(group.Description, Equals, "test_desc")

	groups, err := n.nova.ListSecurityGroups()
	found := false
	for _, g := range groups {
		if g.Id == group.Id {
			found = true
			break
		}
	}
	if found {
		err = n.nova.DeleteSecurityGroup(group.Id)
		c.Check(err, IsNil)
	} else {
		c.Fatalf("test security group (%d) not found", group.Id)
	}
}

func (n *NovaSuite) TestCreateAndDeleteSecurityGroupRules(c *C) {
	group1, err := n.nova.CreateSecurityGroup("test_secgroup1", "test_desc")
	c.Check(err, IsNil)
	group2, err := n.nova.CreateSecurityGroup("test_secgroup2", "test_desc")
	c.Check(err, IsNil)

	// First type of rule - port range + protocol
	ri := nova.RuleInfo{
		IPProtocol:    "tcp",
		FromPort:      1234,
		ToPort:        4321,
		Cidr:          "10.0.0.0/8",
		ParentGroupId: group1.Id,
	}
	rule, err := n.nova.CreateSecurityGroupRule(ri)
	c.Check(err, IsNil)
	c.Check(*rule.FromPort, Equals, 1234)
	c.Check(*rule.ToPort, Equals, 4321)
	c.Check(rule.ParentGroupId, Equals, group1.Id)
	c.Check(*rule.IPProtocol, Equals, "tcp")
	c.Check(rule.Group, IsNil)
	err = n.nova.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, IsNil)

	// Second type of rule - inherited from another group
	ri = nova.RuleInfo{
		GroupId:       &group2.Id,
		ParentGroupId: group1.Id,
	}
	rule, err = n.nova.CreateSecurityGroupRule(ri)
	c.Check(err, IsNil)
	c.Check(rule.ParentGroupId, Equals, group1.Id)
	c.Check(rule.Group, Not(IsNil))
	c.Check(rule.Group.TenantId, Equals, n.tenantId)
	c.Check(rule.Group.Name, Equals, "test_secgroup2")
	err = n.nova.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, IsNil)

	err = n.nova.DeleteSecurityGroup(group1.Id)
	c.Check(err, IsNil)
	err = n.nova.DeleteSecurityGroup(group2.Id)
	c.Check(err, IsNil)
}

func (n *NovaSuite) TestGetServer(c *C) {
	server, err := n.nova.GetServer(n.testServerId)
	c.Assert(err, IsNil)
	c.Assert(server.Id, Equals, n.testServerId)
	c.Assert(server.Name, Equals, "test_server1")
	c.Assert(server.Flavor.Id, Equals, "1")
	c.Assert(server.Image.Id, Equals, "3fc0ef0b-82a9-4f44-a797-a43f0f73b20e")
}

func (n *NovaSuite) waitTestServerToStart(c *C) {
	// Wait until the test server is actually running
	c.Logf("waiting the test server %s to start...", n.testServerId)
	for {
		server, err := n.nova.GetServer(n.testServerId)
		c.Check(err, IsNil)
		if server.Status == "ACTIVE" {
			break
		}
		// There's a rate limit of max 10 POSTs per minute!
		time.Sleep(10 * time.Second)
	}
	c.Logf("started")
}

func (n *NovaSuite) TestServerAddGetRemoveSecurityGroup(c *C) {
	group, err := n.nova.CreateSecurityGroup("test_server_secgroup", "test desc")
	c.Assert(err, IsNil)

	n.waitTestServerToStart(c)
	err = n.nova.AddServerSecurityGroup(n.testServerId, group.Name)
	c.Check(err, IsNil)
	groups, err := n.nova.GetServerSecurityGroups(n.testServerId)
	c.Check(err, IsNil)
	found := false
	for _, g := range groups {
		if g.Id == group.Id || g.Name == group.Name {
			found = true
			break
		}
	}
	err = n.nova.RemoveServerSecurityGroup(n.testServerId, group.Name)
	c.Check(err, IsNil)

	err = n.nova.DeleteSecurityGroup(group.Id)
	c.Assert(err, IsNil)

	if !found {
		c.Fail()
	}
}

func (n *NovaSuite) TestFloatingIPs(c *C) {
	ip, err := n.nova.AllocateFloatingIP()
	c.Assert(err, IsNil)
	c.Check(ip.IP, Not(Equals), "")
	c.Check(ip.Pool, Not(Equals), "")
	c.Check(ip.FixedIP, IsNil)
	c.Check(ip.InstanceId, IsNil)

	ips, err := n.nova.ListFloatingIPs()
	c.Check(err, IsNil)
	if len(ips) < 1 {
		c.Errorf("no floating IPs found (expected at least 1)")
	} else {
		found := false
		for _, i := range ips {
			c.Check(i.IP, Not(Equals), "")
			c.Check(i.Pool, Not(Equals), "")
			if i.Id == ip.Id {
				c.Check(i.IP, Equals, ip.IP)
				c.Check(i.Pool, Equals, ip.Pool)
				found = true
			}
		}
		if !found {
			c.Errorf("expected to find added floating IP: %#v", ip)
		}

		fip, err := n.nova.GetFloatingIP(ip.Id)
		c.Check(err, IsNil)
		c.Check(fip.Id, Equals, ip.Id)
		c.Check(fip.IP, Equals, ip.IP)
		c.Check(fip.Pool, Equals, ip.Pool)
	}
	err = n.nova.DeleteFloatingIP(ip.Id)
	c.Check(err, IsNil)
}

func (n *NovaSuite) TestServerFloatingIPs(c *C) {
	ip, err := n.nova.AllocateFloatingIP()
	c.Assert(err, IsNil)
	c.Check(ip.IP, Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

	n.waitTestServerToStart(c)
	err = n.nova.AddServerFloatingIP(n.testServerId, ip.IP)
	c.Check(err, IsNil)

	fip, err := n.nova.GetFloatingIP(ip.Id)
	c.Check(err, IsNil)
	c.Check(fip.FixedIP, Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	c.Check(fip.InstanceId, Equals, n.testServerId)

	err = n.nova.RemoveServerFloatingIP(n.testServerId, ip.IP)
	c.Check(err, IsNil)
	fip, err = n.nova.GetFloatingIP(ip.Id)
	c.Check(err, IsNil)
	c.Check(fip.FixedIP, IsNil)
	c.Check(fip.InstanceId, IsNil)

	err = n.nova.DeleteFloatingIP(ip.Id)
	c.Check(err, IsNil)
}
