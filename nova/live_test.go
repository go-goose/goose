package nova_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/client"
	"gopkg.in/goose.v1/errors"
	"gopkg.in/goose.v1/identity"
	"gopkg.in/goose.v1/nova"
)

const (
	// A made up name we use for the test server instance.
	testImageName = "nova_test_server"
)

func registerOpenStackTests(cred *identity.Credentials, testImageDetails imageDetails) {
	gc.Suite(&LiveTests{
		cred:        cred,
		testImageId: testImageDetails.imageId,
		testFlavor:  testImageDetails.flavor,
		vendor:      testImageDetails.vendor,
	})
}

type LiveTests struct {
	cred                 *identity.Credentials
	client               client.AuthenticatingClient
	nova                 *nova.Client
	testServer           *nova.Entity
	userId               string
	tenantId             string
	testImageId          string
	testFlavor           string
	testFlavorId         string
	testAvailabilityZone string
	vendor               string
	useNumericIds        bool
}

func (s *LiveTests) SetUpSuite(c *gc.C) {
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
	s.nova = nova.New(s.client)
	var err error
	s.testFlavorId, err = s.findFlavorId(s.testFlavor)
	c.Assert(err, gc.IsNil)
	s.testServer, err = s.createInstance(testImageName)
	c.Assert(err, gc.IsNil)
	s.waitTestServerToStart(c)
	// These will not be filled in until a client has authorised which will happen creating the instance above.
	s.userId = s.client.UserId()
	s.tenantId = s.client.TenantId()
}

func (s *LiveTests) findFlavorId(flavorName string) (string, error) {
	flavors, err := s.nova.ListFlavors()
	if err != nil {
		return "", err
	}
	var flavorId string
	for _, flavor := range flavors {
		if flavor.Name == flavorName {
			flavorId = flavor.Id
			break
		}
	}
	if flavorId == "" {
		return "", fmt.Errorf("No such flavor %s", flavorName)
	}
	return flavorId, nil
}

func (s *LiveTests) TearDownSuite(c *gc.C) {
	if s.testServer != nil {
		err := s.nova.DeleteServer(s.testServer.Id)
		c.Check(err, gc.IsNil)
	}
}

func (s *LiveTests) SetUpTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) createInstance(name string) (instance *nova.Entity, err error) {
	opts := nova.RunServerOpts{
		Name:             name,
		FlavorId:         s.testFlavorId,
		ImageId:          s.testImageId,
		AvailabilityZone: s.testAvailabilityZone,
		UserData:         nil,
	}
	instance, err = s.nova.RunServer(opts)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// Assert that the server record matches the details of the test server image.
func (s *LiveTests) assertServerDetails(c *gc.C, sr *nova.ServerDetail) {
	c.Check(sr.Id, gc.Equals, s.testServer.Id)
	c.Check(sr.Name, gc.Equals, testImageName)
	c.Check(sr.Flavor.Id, gc.Equals, s.testFlavorId)
	c.Check(sr.Image.Id, gc.Equals, s.testImageId)
	if s.testAvailabilityZone != "" {
		c.Check(sr.AvailabilityZone, gc.Equals, s.testAvailabilityZone)
	}
}

func (s *LiveTests) TestListFlavors(c *gc.C) {
	flavors, err := s.nova.ListFlavors()
	c.Assert(err, gc.IsNil)
	if len(flavors) < 1 {
		c.Fatalf("no flavors to list")
	}
	for _, f := range flavors {
		c.Check(f.Id, gc.Not(gc.Equals), "")
		c.Check(f.Name, gc.Not(gc.Equals), "")
		for _, l := range f.Links {
			c.Check(l.Href, gc.Matches, "https?://.*")
			c.Check(l.Rel, gc.Matches, "self|bookmark")
		}
	}
}

func (s *LiveTests) TestListFlavorsDetail(c *gc.C) {
	flavors, err := s.nova.ListFlavorsDetail()
	c.Assert(err, gc.IsNil)
	if len(flavors) < 1 {
		c.Fatalf("no flavors (details) to list")
	}
	for _, f := range flavors {
		c.Check(f.Name, gc.Not(gc.Equals), "")
		c.Check(f.Id, gc.Not(gc.Equals), "")
		if f.RAM < 0 || f.VCPUs < 0 || f.Disk < 0 {
			c.Fatalf("invalid flavor found: %#v", f)
		}
	}
}

func (s *LiveTests) TestListServers(c *gc.C) {
	servers, err := s.nova.ListServers(nil)
	c.Assert(err, gc.IsNil)
	foundTest := false
	for _, sr := range servers {
		c.Check(sr.Id, gc.Not(gc.Equals), "")
		c.Check(sr.Name, gc.Not(gc.Equals), "")
		if sr.Id == s.testServer.Id {
			c.Check(sr.Name, gc.Equals, testImageName)
			foundTest = true
		}
		for _, l := range sr.Links {
			c.Check(l.Href, gc.Matches, "https?://.*")
			c.Check(l.Rel, gc.Matches, "self|bookmark")
		}
	}
	if !foundTest {
		c.Fatalf("test server (%s) not found in server list", s.testServer.Id)
	}
}

func (s *LiveTests) TestListServersWithFilter(c *gc.C) {
	inst, err := s.createInstance("filtered_server")
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteServer(inst.Id)
	filter := nova.NewFilter()
	filter.Set(nova.FilterServer, "filtered_server")
	servers, err := s.nova.ListServers(filter)
	c.Assert(err, gc.IsNil)
	found := false
	for _, sr := range servers {
		if sr.Id == inst.Id {
			c.Assert(sr.Name, gc.Equals, "filtered_server")
			found = true
		}
	}
	if !found {
		c.Fatalf("server (%s) not found in filtered server list %v", inst.Id, servers)
	}
}

func (s *LiveTests) TestListServersDetail(c *gc.C) {
	servers, err := s.nova.ListServersDetail(nil)
	c.Assert(err, gc.IsNil)
	if len(servers) < 1 {
		c.Fatalf("no servers to list (expected at least 1)")
	}
	foundTest := false
	for _, sr := range servers {
		// not checking for Addresses, because it could be missing
		c.Check(sr.Created, gc.Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Check(sr.Updated, gc.Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Check(sr.Id, gc.Not(gc.Equals), "")
		c.Check(sr.HostId, gc.Not(gc.Equals), "")
		c.Check(sr.TenantId, gc.Equals, s.tenantId)
		c.Check(sr.UserId, gc.Equals, s.userId)
		c.Check(sr.Status, gc.Not(gc.Equals), "")
		c.Check(sr.Name, gc.Not(gc.Equals), "")
		if sr.Id == s.testServer.Id {
			s.assertServerDetails(c, &sr)
			foundTest = true
		}
		for _, l := range sr.Links {
			c.Check(l.Href, gc.Matches, "https?://.*")
			c.Check(l.Rel, gc.Matches, "self|bookmark")
		}
		c.Check(sr.Flavor.Id, gc.Not(gc.Equals), "")
		for _, f := range sr.Flavor.Links {
			c.Check(f.Href, gc.Matches, "https?://.*")
			c.Check(f.Rel, gc.Matches, "self|bookmark")
		}
		c.Check(sr.Image.Id, gc.Not(gc.Equals), "")
		for _, i := range sr.Image.Links {
			c.Check(i.Href, gc.Matches, "https?://.*")
			c.Check(i.Rel, gc.Matches, "self|bookmark")
		}
	}
	if !foundTest {
		c.Fatalf("test server (%s) not found in server list (details)", s.testServer.Id)
	}
}

func (s *LiveTests) TestListServersDetailWithFilter(c *gc.C) {
	inst, err := s.createInstance("filtered_server")
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteServer(inst.Id)
	filter := nova.NewFilter()
	filter.Set(nova.FilterServer, "filtered_server")
	servers, err := s.nova.ListServersDetail(filter)
	c.Assert(err, gc.IsNil)
	found := false
	for _, sr := range servers {
		if sr.Id == inst.Id {
			c.Assert(sr.Name, gc.Equals, "filtered_server")
			found = true
		}
	}
	if !found {
		c.Fatalf("server (%s) not found in filtered server details %v", inst.Id, servers)
	}
}

func (s *LiveTests) TestListSecurityGroups(c *gc.C) {
	groups, err := s.nova.ListSecurityGroups()
	c.Assert(err, gc.IsNil)
	if len(groups) < 1 {
		c.Fatalf("no security groups found (expected at least 1)")
	}
	for _, g := range groups {
		c.Check(g.TenantId, gc.Equals, s.tenantId)
		c.Check(g.Name, gc.Not(gc.Equals), "")
		c.Check(g.Description, gc.Not(gc.Equals), "")
		c.Check(g.Rules, gc.NotNil)
	}
}

func (s *LiveTests) TestCreateAndDeleteSecurityGroup(c *gc.C) {
	group, err := s.nova.CreateSecurityGroup("test_secgroup", "test_desc")
	c.Assert(err, gc.IsNil)
	c.Check(group.Name, gc.Equals, "test_secgroup")
	c.Check(group.Description, gc.Equals, "test_desc")

	groups, err := s.nova.ListSecurityGroups()
	found := false
	for _, g := range groups {
		if g.Id == group.Id {
			found = true
			break
		}
	}
	if found {
		err = s.nova.DeleteSecurityGroup(group.Id)
		c.Check(err, gc.IsNil)
	} else {
		c.Fatalf("test security group (%d) not found", group.Id)
	}
}

func (s *LiveTests) TestUpdateSecurityGroup(c *gc.C) {
	group, err := s.nova.CreateSecurityGroup("test_secgroup", "test_desc")
	c.Assert(err, gc.IsNil)
	c.Check(group.Name, gc.Equals, "test_secgroup")
	c.Check(group.Description, gc.Equals, "test_desc")

	groupUpdated, err := s.nova.UpdateSecurityGroup(group.Id, "test_secgroup_new", "test_desc_new")
	c.Assert(err, gc.IsNil)
	c.Check(groupUpdated.Name, gc.Equals, "test_secgroup_new")
	c.Check(groupUpdated.Description, gc.Equals, "test_desc_new")

	groups, err := s.nova.ListSecurityGroups()
	found := false
	for _, g := range groups {
		if g.Id == group.Id {
			found = true
			c.Assert(g.Name, gc.Equals, "test_secgroup_new")
			c.Assert(g.Description, gc.Equals, "test_desc_new")
			break
		}
	}
	if found {
		err = s.nova.DeleteSecurityGroup(group.Id)
		c.Check(err, gc.IsNil)
	} else {
		c.Fatalf("test security group (%d) not found", group.Id)
	}
}

func (s *LiveTests) TestDuplicateSecurityGroupError(c *gc.C) {
	group, err := s.nova.CreateSecurityGroup("test_dupgroup", "test_desc")
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteSecurityGroup(group.Id)
	group, err = s.nova.CreateSecurityGroup("test_dupgroup", "test_desc")
	c.Assert(errors.IsDuplicateValue(err), gc.Equals, true)
}

func (s *LiveTests) TestCreateAndDeleteSecurityGroupRules(c *gc.C) {
	group1, err := s.nova.CreateSecurityGroup("test_secgroup1", "test_desc")
	c.Assert(err, gc.IsNil)
	group2, err := s.nova.CreateSecurityGroup("test_secgroup2", "test_desc")
	c.Assert(err, gc.IsNil)

	// First type of rule - port range + protocol
	ri := nova.RuleInfo{
		IPProtocol:    "tcp",
		FromPort:      1234,
		ToPort:        4321,
		Cidr:          "10.0.0.0/8",
		ParentGroupId: group1.Id,
	}
	rule, err := s.nova.CreateSecurityGroupRule(ri)
	c.Assert(err, gc.IsNil)
	c.Check(*rule.FromPort, gc.Equals, 1234)
	c.Check(*rule.ToPort, gc.Equals, 4321)
	c.Check(rule.ParentGroupId, gc.Equals, group1.Id)
	c.Check(*rule.IPProtocol, gc.Equals, "tcp")
	c.Check(rule.Group, gc.Equals, nova.SecurityGroupRef{})
	err = s.nova.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, gc.IsNil)

	// Second type of rule - inherited from another group
	ri = nova.RuleInfo{
		GroupId:       &group2.Id,
		ParentGroupId: group1.Id,
	}
	rule, err = s.nova.CreateSecurityGroupRule(ri)
	c.Assert(err, gc.IsNil)
	c.Check(rule.ParentGroupId, gc.Equals, group1.Id)
	c.Check(rule.Group, gc.NotNil)
	c.Check(rule.Group.TenantId, gc.Equals, s.tenantId)
	c.Check(rule.Group.Name, gc.Equals, "test_secgroup2")
	err = s.nova.DeleteSecurityGroupRule(rule.Id)
	c.Check(err, gc.IsNil)

	err = s.nova.DeleteSecurityGroup(group1.Id)
	c.Check(err, gc.IsNil)
	err = s.nova.DeleteSecurityGroup(group2.Id)
	c.Check(err, gc.IsNil)
}

func (s *LiveTests) TestGetServer(c *gc.C) {
	server, err := s.nova.GetServer(s.testServer.Id)
	c.Assert(err, gc.IsNil)
	s.assertServerDetails(c, server)
}

func (s *LiveTests) waitTestServerToStart(c *gc.C) {
	// Wait until the test server is actually running
	c.Logf("waiting the test server %s to start...", s.testServer.Id)
	for {
		server, err := s.nova.GetServer(s.testServer.Id)
		c.Assert(err, gc.IsNil)
		if server.Status == nova.StatusActive {
			break
		}
		// We dont' want to flood the connection while polling the server waiting for it to start.
		c.Logf("server has status %s, waiting 10 seconds before polling again...", server.Status)
		time.Sleep(10 * time.Second)
	}
	c.Logf("started")
}

func (s *LiveTests) TestServerAddGetRemoveSecurityGroup(c *gc.C) {
	group, err := s.nova.CreateSecurityGroup("test_server_secgroup", "test desc")
	if err != nil {
		c.Assert(errors.IsDuplicateValue(err), gc.Equals, true)
		group, err = s.nova.SecurityGroupByName("test_server_secgroup")
		c.Assert(err, gc.IsNil)
	}

	s.waitTestServerToStart(c)
	err = s.nova.AddServerSecurityGroup(s.testServer.Id, group.Name)
	c.Assert(err, gc.IsNil)
	groups, err := s.nova.GetServerSecurityGroups(s.testServer.Id)
	c.Assert(err, gc.IsNil)
	found := false
	for _, g := range groups {
		if g.Id == group.Id || g.Name == group.Name {
			found = true
			break
		}
	}
	err = s.nova.RemoveServerSecurityGroup(s.testServer.Id, group.Name)
	c.Check(err, gc.IsNil)

	err = s.nova.DeleteSecurityGroup(group.Id)
	c.Assert(err, gc.IsNil)

	if !found {
		c.Fail()
	}
}

func (s *LiveTests) TestFloatingIPs(c *gc.C) {
	ip, err := s.nova.AllocateFloatingIP()
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteFloatingIP(ip.Id)
	c.Check(ip.IP, gc.Not(gc.Equals), "")
	c.Check(ip.FixedIP, gc.IsNil)
	c.Check(ip.InstanceId, gc.IsNil)

	ips, err := s.nova.ListFloatingIPs()
	c.Assert(err, gc.IsNil)
	if len(ips) < 1 {
		c.Errorf("no floating IPs found (expected at least 1)")
	} else {
		found := false
		for _, i := range ips {
			c.Check(i.IP, gc.Not(gc.Equals), "")
			if i.Id == ip.Id {
				c.Check(i.IP, gc.Equals, ip.IP)
				c.Check(i.Pool, gc.Equals, ip.Pool)
				found = true
			}
		}
		if !found {
			c.Errorf("expected to find added floating IP: %#v", ip)
		}

		fip, err := s.nova.GetFloatingIP(ip.Id)
		c.Assert(err, gc.IsNil)
		c.Check(fip.Id, gc.Equals, ip.Id)
		c.Check(fip.IP, gc.Equals, ip.IP)
		c.Check(fip.Pool, gc.Equals, ip.Pool)
	}
}

func (s *LiveTests) TestServerFloatingIPs(c *gc.C) {
	ip, err := s.nova.AllocateFloatingIP()
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteFloatingIP(ip.Id)
	c.Check(ip.IP, gc.Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

	s.waitTestServerToStart(c)
	err = s.nova.AddServerFloatingIP(s.testServer.Id, ip.IP)
	c.Assert(err, gc.IsNil)
	// TODO (wallyworld) - 2013-02-11 bug=1121666
	// where we are creating a real server, test that the IP address created above
	// can be used to connect to the server
	defer s.nova.RemoveServerFloatingIP(s.testServer.Id, ip.IP)

	fip, err := s.nova.GetFloatingIP(ip.Id)
	c.Assert(err, gc.IsNil)
	c.Check(*fip.FixedIP, gc.Matches, `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	c.Check(*fip.InstanceId, gc.Equals, s.testServer.Id)

	err = s.nova.RemoveServerFloatingIP(s.testServer.Id, ip.IP)
	c.Check(err, gc.IsNil)
	fip, err = s.nova.GetFloatingIP(ip.Id)
	c.Assert(err, gc.IsNil)
	c.Check(fip.FixedIP, gc.IsNil)
	c.Check(fip.InstanceId, gc.IsNil)
}

// TestRateLimitRetry checks that when we make too many requests and receive a Retry-After response, the retry
// occurs and the request ultimately succeeds.
func (s *LiveTests) TestRateLimitRetry(c *gc.C) {
	if s.vendor != "canonistack" {
		c.Skip("TestRateLimitRetry is only run for Canonistack")
	}
	// Capture the logged output so we can check for retry messages.
	var logout bytes.Buffer
	logger := log.New(&logout, "", log.LstdFlags)
	client := client.NewClient(s.cred, identity.AuthUserPass, logger)
	novaClient := nova.New(client)
	// Delete the artifact if it already exists.
	testGroup, err := novaClient.SecurityGroupByName("test_group")
	if err != nil {
		c.Assert(errors.IsNotFound(err), gc.Equals, true)
	} else {
		novaClient.DeleteSecurityGroup(testGroup.Id)
		c.Assert(err, gc.IsNil)
	}
	// Create some artifacts a number of times in succession and ensure each time is successful,
	// even with retries being required. As soon as we see a retry message, the test has passed
	// and we exit.
	for i := 0; i < 50; i++ {
		testGroup, err = novaClient.CreateSecurityGroup("test_group", "test")
		c.Assert(err, gc.IsNil)
		novaClient.DeleteSecurityGroup(testGroup.Id)
		c.Assert(err, gc.IsNil)
		output := logout.String()
		if strings.Contains(output, "Too many requests, retrying in") == true {
			return
		}
	}
	// No retry message logged so test has failed.
	c.Fail()
}

func (s *LiveTests) TestRegexpInstanceFilters(c *gc.C) {
	serverNames := []string{
		"foobar123",
		"foo123baz",
		"123barbaz",
	}
	for _, name := range serverNames {
		inst, err := s.createInstance(name)
		c.Assert(err, gc.IsNil)
		defer s.nova.DeleteServer(inst.Id)
	}
	filter := nova.NewFilter()
	filter.Set(nova.FilterServer, `foo.*baz`)
	servers, err := s.nova.ListServersDetail(filter)
	c.Assert(err, gc.IsNil)
	c.Assert(servers, gc.HasLen, 1)
	c.Assert(servers[0].Name, gc.Equals, serverNames[1])
	filter.Set(nova.FilterServer, `[0-9]+[a-z]+`)
	servers, err = s.nova.ListServersDetail(filter)
	c.Assert(err, gc.IsNil)
	c.Assert(servers, gc.HasLen, 2)
	if servers[0].Name != serverNames[1] {
		servers[0], servers[1] = servers[1], servers[0]
	}
	c.Assert(servers[0].Name, gc.Equals, serverNames[1])
	c.Assert(servers[1].Name, gc.Equals, serverNames[2])
}

func (s *LiveTests) TestListNetworks(c *gc.C) {
	networks, err := s.nova.ListNetworks()
	c.Assert(err, gc.IsNil)
	for _, network := range networks {
		c.Check(network.Id, gc.Not(gc.Equals), "")
		c.Check(network.Label, gc.Not(gc.Equals), "")
		c.Assert(network.Cidr, gc.Matches, `\d{1,3}(\.+\d{1,3}){3}\/\d+`)
	}
}

func (s *LiveTests) runServerAvailabilityZone(zone string) (*nova.Entity, error) {
	old := s.testAvailabilityZone
	defer func() { s.testAvailabilityZone = old }()
	s.testAvailabilityZone = zone
	return s.createInstance(testImageName)
}

func (s *LiveTests) TestRunServerUnknownAvailabilityZone(c *gc.C) {
	_, err := s.runServerAvailabilityZone("something_that_will_never_exist")
	c.Assert(err, gc.ErrorMatches, "(.|\n)*The requested availability zone is not available(.|\n)*")
}

func (s *LiveTests) TestUpdateServerName(c *gc.C) {
	entity, err := s.nova.RunServer(nova.RunServerOpts{
		Name:             "oldName",
		FlavorId:         s.testFlavorId,
		ImageId:          s.testImageId,
		AvailabilityZone: s.testAvailabilityZone,
		Metadata:         map[string]string{},
	})
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteServer(entity.Id)

	newEntity, err := s.nova.UpdateServerName(entity.Id, "newName")
	c.Assert(err, gc.IsNil)
	c.Assert(newEntity.Name, gc.Equals, "newName")

	server, err := s.nova.GetServer(entity.Id)
	c.Assert(err, gc.IsNil)
	c.Assert(server.Name, gc.Equals, "newName")
}

func (s *LiveTests) TestInstanceMetadata(c *gc.C) {
	metadata := map[string]string{"my-key": "my-value"}
	entity, err := s.nova.RunServer(nova.RunServerOpts{
		Name:             "inst-metadata",
		FlavorId:         s.testFlavorId,
		ImageId:          s.testImageId,
		AvailabilityZone: s.testAvailabilityZone,
		Metadata:         metadata,
	})
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteServer(entity.Id)

	server, err := s.nova.GetServer(entity.Id)
	c.Assert(err, gc.IsNil)

	// nova may have added metadata as well;
	// delete it before comparing.
	for k := range server.Metadata {
		if _, ok := metadata[k]; !ok {
			delete(server.Metadata, k)
		}
	}
	c.Assert(server.Metadata, gc.DeepEquals, metadata)
}

func (s *LiveTests) TestSetServerMetadata(c *gc.C) {
	entity, err := s.nova.RunServer(nova.RunServerOpts{
		Name:             "inst-metadata",
		FlavorId:         s.testFlavorId,
		ImageId:          s.testImageId,
		AvailabilityZone: s.testAvailabilityZone,
	})
	c.Assert(err, gc.IsNil)
	defer s.nova.DeleteServer(entity.Id)

	for _, metadata := range []map[string]string{{
		"k1": "v1",
	}, {
		"k1": "v1.replacement",
		"k2": "v2",
	}} {
		err = s.nova.SetServerMetadata(entity.Id, metadata)
		c.Assert(err, gc.IsNil)
	}

	server, err := s.nova.GetServer(entity.Id)
	c.Assert(err, gc.IsNil)
	c.Assert(server.Metadata["k1"], gc.Equals, "v1.replacement")
	c.Assert(server.Metadata["k2"], gc.Equals, "v2")
}
