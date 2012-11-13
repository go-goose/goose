package client_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"testing"
)

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type ClientSuite struct {
	client   *client.OpenStackClient
	username string
	password string
	tenant   string
	skipAuth bool
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
}

func (s *ClientSuite) TestListFlavors(c *C) {
	flavors, err := s.client.ListFlavors()
	c.Assert(err, IsNil)
	if len(flavors) < 1 {
		c.Fail()
	}
	for _, f := range flavors {
		c.Assert(f.Id, Not(Equals), "")
		c.Assert(f.Name, Not(Equals), "")
		for _, l := range f.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Not(Equals), "")
		}
	}
}

func (s *ClientSuite) TestListFlavorsDetail(c *C) {
	flavors, err := s.client.ListFlavorsDetail()
	c.Assert(err, IsNil)
	if len(flavors) < 1 {
		c.Fail()
	}
	for _, f := range flavors {
		c.Assert(f.Name, Not(Equals), "")
		c.Assert(f.Id, Not(Equals), "")
		if f.RAM < 0 || f.VCPUs < 0 || f.Disk < 0 {
			c.Fail()
		}
	}
}

func (s *ClientSuite) TestListServers(c *C) {
	servers, err := s.client.ListServers()
	c.Assert(err, IsNil)
	for _, sr := range servers {
		c.Assert(sr.Id, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		for _, l := range sr.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Not(Equals), "")
		}
	}
}

func (s *ClientSuite) TestListServersDetail(c *C) {
	servers, err := s.client.ListServersDetail()
	c.Assert(err, IsNil)
	if len(servers) < 1 {
		c.Logf("no servers to test!")
	}
	for _, sr := range servers {
		c.Assert(sr.Created, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(sr.Updated, Matches, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*`)
		c.Assert(sr.Id, Not(Equals), "")
		c.Assert(sr.HostId, Not(Equals), "")
		c.Assert(sr.TenantId, Equals, s.client.Token.Tenant.Id)
		c.Assert(sr.UserId, Equals, s.client.User.Id)
		c.Assert(sr.Status, Not(Equals), "")
		c.Assert(sr.Name, Not(Equals), "")
		for _, l := range sr.Links {
			c.Assert(l.Href, Matches, "https?://.*")
			c.Assert(l.Rel, Not(Equals), "")
		}
		c.Assert(sr.Flavor.Id, Not(Equals), "")
		for _, f := range sr.Flavor.Links {
			c.Assert(f.Href, Matches, "https?://.*")
			c.Assert(f.Rel, Not(Equals), "")
		}
		c.Assert(sr.Image.Id, Not(Equals), "")
		for _, i := range sr.Image.Links {
			c.Assert(i.Href, Matches, "https?://.*")
			c.Assert(i.Rel, Not(Equals), "")
		}
	}
}

func (s *ClientSuite) TestListSecurityGroups(c *C) {
	groups, err := s.client.ListSecurityGroups()
	c.Assert(err, IsNil)
	if len(groups) < 1 {
		c.Fail()
	}
	for _, g := range groups {
		c.Assert(g.TenantId, Equals, s.client.Token.Tenant.Id)
		c.Assert(g.Name, Not(Equals), "")
		c.Assert(g.Description, Not(Equals), "")
		c.Assert(g.Rules, NotNil)
	}
}
