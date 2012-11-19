package client_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"testing"
	"time"
)

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type ClientSuite struct {
	client       *client.OpenStackClient
	username     string
	password     string
	tenant       string
	testServerId string
	skipAuth     bool
}

func (s *ClientSuite) SetUpSuite(c *C) {
	if !*live {
		c.Skip("-live not provided")
	}

	username, password, tenant, region, auth_url := client.GetEnvVars()
	for i, p := range []string{username, password, tenant, region, auth_url} {
		if p == "" {
			c.Fatalf("required environment variable not set: %d", i)
		}
	}

	s.client = &client.OpenStackClient{IdentityEndpoint: auth_url, Region: region}
	s.username = username
	s.password = password
	s.tenant = tenant
	s.skipAuth = true // set after TestAuthenticate

}

// Create a test server needed to run some of the tests
func (s *ClientSuite) SetUpTestServer(c *C) {
	if s.testServerId != "" {
		return // Already done
	}
	s.SetUpTest(c) // Authenticate if needed
	ro := client.RunServerOpts{
		Name:     "test_server1",
		FlavorId: "1",                                    // m1.tiny
		ImageId:  "3fc0ef0b-82a9-4f44-a797-a43f0f73b20e", // smoser-cloud-images/ubuntu-precise-12.04-i386-server-20120424.manifest.xml
	}
	err := s.client.RunServer(ro)
	c.Check(err, IsNil)

	// Now find it and save its ID
	servers, err := s.client.ListServers()
	c.Check(err, IsNil)
	for _, sr := range servers {
		if sr.Name == "test_server1" {
			s.testServerId = sr.Id
			// Give it some time to initialize
			time.Sleep(8 * time.Second)
			break
		}
	}
	if s.testServerId == "" {
		c.Fatalf("cannot start test server")
	}
}

func (s *ClientSuite) TearDownSuite(c *C) {
	if s.testServerId != "" {
		// Remove the test server we created earlier
		err := s.client.DeleteServer(s.testServerId)
		c.Assert(err, IsNil)
	}
}

func (s *ClientSuite) SetUpTest(c *C) {
	if !s.skipAuth && !s.client.IsAuthenticated() {
		err := s.client.Authenticate(s.username, s.password, s.tenant)
		c.Assert(err, IsNil)
		c.Logf("authenticated")
	}
}

var suite = Suite(&ClientSuite{})

var authTests = []struct {
	summary string
	inputs  []string
	err     string
}{
	{
		summary: "empty args",
		inputs:  []string{"", "", ""},
		err:     "required arg.*missing",
	},
	{
		summary: "dummy args",
		inputs:  []string{"phony", "fake", "dummy"},
		err:     "authentication failed.*",
	},
	{
		summary: "valid args",
		inputs:  []string{"!", "", ""},
		err:     "",
	},
}

func (s *ClientSuite) TestAuthenticate(c *C) {
	c.Assert(s.client.IsAuthenticated(), Equals, false)
	for _, t := range authTests {
		c.Logf("test: %s", t.summary)
		var err error
		if t.inputs[0] == "!" {
			err = s.client.Authenticate(s.username, s.password, s.tenant)
		} else {
			err = s.client.Authenticate(t.inputs[0], t.inputs[1], t.inputs[2])
		}
		if t.err == "" {
			c.Assert(err, IsNil)
			c.Assert(s.client.IsAuthenticated(), Equals, true)
		} else {
			c.Assert(err, ErrorMatches, t.err)
		}
	}

	// Check service endpoints are discovered
	c.Assert(s.client.Services["compute"], NotNil)
	c.Assert(s.client.Services["swift"], NotNil)

	s.skipAuth = false

	// Since this is the first test, get the test server running ASAP
	s.SetUpTestServer(c)
}

func (s *ClientSuite) TestListFlavors(c *C) {
	flavors, err := s.client.ListFlavors()
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

func (s *ClientSuite) TestListFlavorsDetail(c *C) {
	flavors, err := s.client.ListFlavorsDetail()
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

func (s *ClientSuite) TestListServers(c *C) {
	servers, err := s.client.ListServers()
	c.Assert(err, IsNil)
	foundTest := false
	for _, sr := range servers {
		c.Assert(sr.Id, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		if sr.Id == s.testServerId {
			c.Assert(sr.Name, Equals, "test_server1")
			foundTest = true
		}
		for _, l := range sr.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Matches, "self|bookmark")
		}
	}
	if !foundTest {
		c.Fatalf("test server (%s) not found in server list", s.testServerId)
	}
}

func (s *ClientSuite) TestListServersDetail(c *C) {
	servers, err := s.client.ListServersDetail()
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
		c.Assert(sr.TenantId, Equals, s.client.Token.Tenant.Id)
		c.Assert(sr.UserId, Equals, s.client.User.Id)
		c.Assert(sr.Status, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		if sr.Id == s.testServerId {
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
		c.Fatalf("test server (%s) not found in server list (details)", s.testServerId)
	}
}

func (s *ClientSuite) TestListSecurityGroups(c *C) {
	groups, err := s.client.ListSecurityGroups()
	c.Assert(err, IsNil)
	if len(groups) < 1 {
		c.Fatalf("no security groups found (expected at least 1)")
	}
	for _, g := range groups {
		c.Assert(g.TenantId, Equals, s.client.Token.Tenant.Id)
		c.Assert(g.Name, Not(Equals), "")
		c.Assert(g.Description, Not(Equals), "")
		c.Assert(g.Rules, NotNil)
	}
}

func (s *ClientSuite) TestCreateAndDeleteSecurityGroup(c *C) {
	group, err := s.client.CreateSecurityGroup("test_secgroup", "test_desc")
	c.Check(err, IsNil)
	c.Check(group.Name, Equals, "test_secgroup")
	c.Check(group.Description, Equals, "test_desc")

	groups, err := s.client.ListSecurityGroups()
	found := false
	for _, g := range groups {
		if g.Id == group.Id {
			found = true
			break
		}
	}
	if found {
		err = s.client.DeleteSecurityGroup(group.Id)
		c.Check(err, IsNil)
	} else {
		c.Fatalf("test security group (%d) not found", group.Id)
	}
}

func (s *ClientSuite) TestCreateAndDeleteSecurityGroupRules(c *C) {
	group1, err := s.client.CreateSecurityGroup("test_secgroup1", "test_desc")
	c.Check(err, IsNil)
	group2, err := s.client.CreateSecurityGroup("test_secgroup2", "test_desc")
	c.Check(err, IsNil)

	// First type of rule - port range + protocol
	ri := client.RuleInfo{
		IPProtocol:    "tcp",
		FromPort:      1234,
		ToPort:        4321,
		Cidr:          "10.0.0.0/8",
		ParentGroupId: group1.Id,
	}
	rule, err := s.client.CreateSecurityGroupRule(ri)
	c.Check(err, IsNil)
	c.Check(*rule.FromPort, Equals, 1234)
	c.Check(*rule.ToPort, Equals, 4321)
	c.Check(rule.ParentGroupId, Equals, group1.Id)
	c.Check(*rule.IPProtocol, Equals, "tcp")
	c.Check(rule.Group, HasLen, 0)
	err = s.client.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, IsNil)

	// Second type of rule - inherited from another group
	ri = client.RuleInfo{
		GroupId:       &group2.Id,
		ParentGroupId: group1.Id,
	}
	rule, err = s.client.CreateSecurityGroupRule(ri)
	c.Check(err, IsNil)
	c.Check(rule.ParentGroupId, Equals, group1.Id)
	c.Check(rule.Group["tenant_id"], Equals, s.client.Token.Tenant.Id)
	c.Check(rule.Group["name"], Equals, "test_secgroup2")
	err = s.client.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, IsNil)

	err = s.client.DeleteSecurityGroup(group1.Id)
	c.Check(err, IsNil)
	err = s.client.DeleteSecurityGroup(group2.Id)
	c.Check(err, IsNil)
}

func (s *ClientSuite) TestGetServer(c *C) {
	server, err := s.client.GetServer(s.testServerId)
	c.Assert(err, IsNil)
	c.Assert(server.Id, Equals, s.testServerId)
	c.Assert(server.Name, Equals, "test_server1")
	c.Assert(server.Flavor.Id, Equals, "1")
	c.Assert(server.Image.Id, Equals, "3fc0ef0b-82a9-4f44-a797-a43f0f73b20e")
}

func (s *ClientSuite) waitTestServerToStart(c *C) {
	// Wait until the test server is actually running
	c.Logf("waiting the test server to start...")
	for {
		server, err := s.client.GetServer(s.testServerId)
		c.Check(err, IsNil)
		if server.Status == "ACTIVE" {
			break
		}
		// There's a rate limit of max 10 POSTs per minute!
		time.Sleep(10 * time.Second)
	}
	c.Logf("started")
}

func (s *ClientSuite) TestServerAddGetRemoveSecurityGroup(c *C) {
	group, err := s.client.CreateSecurityGroup("test_server_secgroup", "test desc")
	c.Assert(err, IsNil)

	s.waitTestServerToStart(c)
	err = s.client.AddServerSecurityGroup(s.testServerId, group.Name)
	c.Check(err, IsNil)
	groups, err := s.client.GetServerSecurityGroups(s.testServerId)
	c.Check(err, IsNil)
	found := false
	for _, g := range groups {
		if g.Id == group.Id || g.Name == group.Name {
			found = true
			break
		}
	}
	err = s.client.RemoveServerSecurityGroup(s.testServerId, group.Name)
	c.Check(err, IsNil)

	err = s.client.DeleteSecurityGroup(group.Id)
	c.Assert(err, IsNil)

	if !found {
		c.Fail()
	}
}

func (s *ClientSuite) TestFloatingIPs(c *C) {
	ip, err := s.client.AllocateFloatingIP()
	c.Assert(err, IsNil)
	c.Check(ip.IP, Not(Equals), "")
	c.Check(ip.Pool, Not(Equals), "")
	c.Check(ip.FixedIP, IsNil)
	c.Check(ip.InstanceId, IsNil)

	ips, err := s.client.ListFloatingIPs()
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

		fip, err := s.client.GetFloatingIP(ip.Id)
		c.Check(err, IsNil)
		c.Check(fip.Id, Equals, ip.Id)
		c.Check(fip.IP, Equals, ip.IP)
		c.Check(fip.Pool, Equals, ip.Pool)
	}
	err = s.client.DeleteFloatingIP(ip.Id)
	c.Check(err, IsNil)
}

func (s *ClientSuite) TestServerFloatingIPs(c *C) {
	ip, err := s.client.AllocateFloatingIP()
	c.Assert(err, IsNil)
	c.Check(ip.IP, Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

	s.waitTestServerToStart(c)
	err = s.client.AddServerFloatingIP(s.testServerId, ip.IP)
	c.Check(err, IsNil)

	fip, err := s.client.GetFloatingIP(ip.Id)
	c.Check(err, IsNil)
	c.Check(fip.FixedIP, Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	c.Check(fip.InstanceId, Equals, s.testServerId)

	err = s.client.RemoveServerFloatingIP(s.testServerId, ip.IP)
	c.Check(err, IsNil)
	fip, err = s.client.GetFloatingIP(ip.Id)
	c.Check(err, IsNil)
	c.Check(fip.FixedIP, IsNil)
	c.Check(fip.InstanceId, IsNil)

	err = s.client.DeleteFloatingIP(ip.Id)
	c.Check(err, IsNil)
}

func (s *ClientSuite) TestCreateAndDeleteContainer(c *C) {
	container := "test_container"
	err := s.client.CreateContainer(container)
	c.Check(err, IsNil)
	err = s.client.DeleteContainer(container)
	c.Check(err, IsNil)
}

func (s *ClientSuite) TestObjects(c *C) {

	container := "test_container"
	object := "test_obj"
	data := []byte("...some data...")
	err := s.client.CreateContainer(container)
	c.Check(err, IsNil)
	err = s.client.PutObject(container, object, data)
	c.Check(err, IsNil)
	objdata, err := s.client.GetObject(container, object)
	c.Check(err, IsNil)
	c.Check(objdata, Equals, data)
	err = s.client.DeleteObject(container, object)
	c.Check(err, IsNil)
	err = s.client.DeleteContainer(container)
	c.Check(err, IsNil)
}
