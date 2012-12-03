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

const (
	// Known, pre-existing image details to use when creating a test server instance.
	testImageId   = "0f602ea9-c09e-440c-9e29-cfae5635afa3" // smoser-cloud-images/ubuntu-quantal-12.10-i386-server-20121017
	testFlavourId = "1"                                    // m1.tiny
	// A made up name we use for the test server instance.
	testImageName = "nova_test_server"
)

func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type NovaSuite struct {
	nova       *nova.Client
	testServer *nova.Entity
	userId     string
	tenantId   string
}

func (s *NovaSuite) SetUpSuite(c *C) {
	if !*live {
		c.Skip("-live not provided")
	}

	cred, err := identity.CompleteCredentialsFromEnv()
	c.Assert(err, IsNil)
	client := client.NewClient(cred, identity.AuthUserPass)
	s.nova = nova.New(client)
	s.testServer, err = s.createInstance(c)
	c.Assert(err, IsNil)
	s.waitTestServerToStart(c)
	// These will not be filled in until a client has authorised which will happen creating the instance above.
	s.userId = client.UserId
	s.tenantId = client.TenantId
}

func (s *NovaSuite) TearDownSuite(c *C) {
	err := s.nova.DeleteServer(s.testServer.Id)
	c.Check(err, IsNil)
}

func (s *NovaSuite) createInstance(c *C) (instance *nova.Entity, err error) {
	opts := nova.RunServerOpts{
		Name:     testImageName,
		FlavorId: testFlavourId,
		ImageId:  testImageId,
		UserData: nil,
	}
	instance, err = s.nova.RunServer(opts)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

var suite = Suite(&NovaSuite{})

// Assert that the server record matches the details of the test server image.
func (s *NovaSuite) assertServerDetails(c *C, sr *nova.ServerDetail) {
	c.Assert(sr.Id, Equals, s.testServer.Id)
	c.Assert(sr.Name, Equals, testImageName)
	c.Assert(sr.Flavor.Id, Equals, testFlavourId)
	c.Assert(sr.Image.Id, Equals, testImageId)
}

func (s *NovaSuite) TestListFlavors(c *C) {
	flavors, err := s.nova.ListFlavors()
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

func (s *NovaSuite) TestListFlavorsDetail(c *C) {
	flavors, err := s.nova.ListFlavorsDetail()
	c.Assert(err, IsNil)
	if len(*flavors) < 1 {
		c.Fatalf("no flavors (details) to list")
	}
	for _, f := range *flavors {
		c.Assert(f.Name, Not(Equals), "")
		c.Assert(f.Id, Not(Equals), "")
		if f.RAM < 0 || f.VCPUs < 0 || f.Disk < 0 {
			c.Fatalf("invalid flavor found: %#v", f)
		}
	}
}

func (s *NovaSuite) TestListServers(c *C) {
	servers, err := s.nova.ListServers()
	c.Assert(err, IsNil)
	foundTest := false
	for _, sr := range *servers {
		c.Assert(sr.Id, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		if sr.Id == s.testServer.Id {
			c.Assert(sr.Name, Equals, testImageName)
			foundTest = true
		}
		for _, l := range sr.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Matches, "self|bookmark")
		}
	}
	if !foundTest {
		c.Fatalf("test server (%s) not found in server list", s.testServer.Id)
	}
}

func (s *NovaSuite) TestListServersDetail(c *C) {
	servers, err := s.nova.ListServersDetail()
	c.Assert(err, IsNil)
	if len(*servers) < 1 {
		c.Fatalf("no servers to list (expected at least 1)")
	}
	foundTest := false
	for _, sr := range *servers {
		c.Assert(sr.Created, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(sr.Updated, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(sr.Id, Not(Equals), "")
		c.Assert(sr.HostId, Not(Equals), "")
		c.Assert(sr.TenantId, Equals, s.tenantId)
		c.Assert(sr.UserId, Equals, s.userId)
		c.Assert(sr.Status, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		if sr.Id == s.testServer.Id {
			s.assertServerDetails(c, &sr)
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
		c.Fatalf("test server (%s) not found in server list (details)", s.testServer.Id)
	}
}

func (s *NovaSuite) TestListSecurityGroups(c *C) {
	groups, err := s.nova.ListSecurityGroups()
	c.Assert(err, IsNil)
	if len(*groups) < 1 {
		c.Fatalf("no security groups found (expected at least 1)")
	}
	for _, g := range *groups {
		c.Assert(g.TenantId, Equals, s.tenantId)
		c.Assert(g.Name, Not(Equals), "")
		c.Assert(g.Description, Not(Equals), "")
		c.Assert(g.Rules, NotNil)
	}
}

func (s *NovaSuite) TestCreateAndDeleteSecurityGroup(c *C) {
	group, err := s.nova.CreateSecurityGroup("test_secgroup", "test_desc")
	c.Assert(err, IsNil)
	c.Check(group.Name, Equals, "test_secgroup")
	c.Check(group.Description, Equals, "test_desc")

	groups, err := s.nova.ListSecurityGroups()
	found := false
	for _, g := range *groups {
		if g.Id == group.Id {
			found = true
			break
		}
	}
	if found {
		err = s.nova.DeleteSecurityGroup(group.Id)
		c.Check(err, IsNil)
	} else {
		c.Fatalf("test security group (%d) not found", group.Id)
	}
}

func (s *NovaSuite) TestCreateAndDeleteSecurityGroupRules(c *C) {
	group1, err := s.nova.CreateSecurityGroup("test_secgroup1", "test_desc")
	c.Assert(err, IsNil)
	group2, err := s.nova.CreateSecurityGroup("test_secgroup2", "test_desc")
	c.Assert(err, IsNil)

	// First type of rule - port range + protocol
	ri := nova.RuleInfo{
		IPProtocol:    "tcp",
		FromPort:      1234,
		ToPort:        4321,
		Cidr:          "10.0.0.0/8",
		ParentGroupId: group1.Id,
	}
	rule, err := s.nova.CreateSecurityGroupRule(ri)
	c.Assert(err, IsNil)
	c.Check(*rule.FromPort, Equals, 1234)
	c.Check(*rule.ToPort, Equals, 4321)
	c.Check(rule.ParentGroupId, Equals, group1.Id)
	c.Check(*rule.IPProtocol, Equals, "tcp")
	c.Check(rule.Group, HasLen, 0)
	err = s.nova.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, IsNil)

	// Second type of rule - inherited from another group
	ri = nova.RuleInfo{
		GroupId:       &group2.Id,
		ParentGroupId: group1.Id,
	}
	rule, err = s.nova.CreateSecurityGroupRule(ri)
	c.Assert(err, IsNil)
	c.Check(rule.ParentGroupId, Equals, group1.Id)
	c.Check(rule.Group["tenant_id"], Equals, s.tenantId)
	c.Check(rule.Group["name"], Equals, "test_secgroup2")
	err = s.nova.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, IsNil)

	err = s.nova.DeleteSecurityGroup(group1.Id)
	c.Check(err, IsNil)
	err = s.nova.DeleteSecurityGroup(group2.Id)
	c.Check(err, IsNil)
}

func (s *NovaSuite) TestGetServer(c *C) {
	server, err := s.nova.GetServer(s.testServer.Id)
	c.Assert(err, IsNil)
	s.assertServerDetails(c, server)
}

func (s *NovaSuite) waitTestServerToStart(c *C) {
	// Wait until the test server is actually running
	c.Logf("waiting the test server %s to start...", s.testServer.Id)
	for {
		server, err := s.nova.GetServer(s.testServer.Id)
		c.Assert(err, IsNil)
		if server.Status == "ACTIVE" {
			break
		}
		// There's a rate limit of max 10 POSTs per minute!
		time.Sleep(10 * time.Second)
	}
	c.Logf("started")
}

func (s *NovaSuite) TestServerAddGetRemoveSecurityGroup(c *C) {
	group, err := s.nova.CreateSecurityGroup("test_server_secgroup", "test desc")
	c.Assert(err, IsNil)

	s.waitTestServerToStart(c)
	err = s.nova.AddServerSecurityGroup(s.testServer.Id, group.Name)
	c.Assert(err, IsNil)
	groups, err := s.nova.GetServerSecurityGroups(s.testServer.Id)
	c.Assert(err, IsNil)
	found := false
	for _, g := range *groups {
		if g.Id == group.Id || g.Name == group.Name {
			found = true
			break
		}
	}
	err = s.nova.RemoveServerSecurityGroup(s.testServer.Id, group.Name)
	c.Check(err, IsNil)

	err = s.nova.DeleteSecurityGroup(group.Id)
	c.Assert(err, IsNil)

	if !found {
		c.Fail()
	}
}

func (s *NovaSuite) TestFloatingIPs(c *C) {
	ip, err := s.nova.AllocateFloatingIP()
	c.Assert(err, IsNil)
	c.Check(ip.IP, Not(Equals), "")
	c.Check(ip.Pool, Not(Equals), "")
	c.Check(ip.FixedIP, IsNil)
	c.Check(ip.InstanceId, IsNil)

	ips, err := s.nova.ListFloatingIPs()
	c.Assert(err, IsNil)
	if len(*ips) < 1 {
		c.Errorf("no floating IPs found (expected at least 1)")
	} else {
		found := false
		for _, i := range *ips {
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

		fip, err := s.nova.GetFloatingIP(ip.Id)
		c.Assert(err, IsNil)
		c.Check(fip.Id, Equals, ip.Id)
		c.Check(fip.IP, Equals, ip.IP)
		c.Check(fip.Pool, Equals, ip.Pool)
	}
	err = s.nova.DeleteFloatingIP(ip.Id)
	c.Check(err, IsNil)
}

func (s *NovaSuite) TestServerFloatingIPs(c *C) {
	ip, err := s.nova.AllocateFloatingIP()
	c.Assert(err, IsNil)
	c.Check(ip.IP, Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

	s.waitTestServerToStart(c)
	err = s.nova.AddServerFloatingIP(s.testServer.Id, ip.IP)
	c.Assert(err, IsNil)

	fip, err := s.nova.GetFloatingIP(ip.Id)
	c.Assert(err, IsNil)
	c.Check(fip.FixedIP, Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	c.Check(fip.InstanceId, Equals, s.testServer.Id)

	err = s.nova.RemoveServerFloatingIP(s.testServer.Id, ip.IP)
	c.Check(err, IsNil)
	fip, err = s.nova.GetFloatingIP(ip.Id)
	c.Assert(err, IsNil)
	c.Check(fip.FixedIP, IsNil)
	c.Check(fip.InstanceId, IsNil)

	err = s.nova.DeleteFloatingIP(ip.Id)
	c.Check(err, IsNil)
}
