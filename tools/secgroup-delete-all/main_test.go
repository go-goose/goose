package main_test

import (
	"bytes"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/nova"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/novaservice"
	tool "launchpad.net/goose/tools/secgroup-delete-all"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

const (
	username = "auser"
	password = "apass"
	region   = "aregion"
	tenant   = "1"
)

type ToolSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&ToolSuite{})

// GZ 2013-01-21: Should require EnvSuite for this, but clashes with HTTPSuite
func createNovaClient(auth_url string) *nova.Client {
	creds := identity.Credentials{
		URL:        auth_url,
		User:       username,
		Secrets:    password,
		Region:     region,
		TenantName: tenant,
	}
	osc := client.NewClient(&creds, identity.AuthUserPass, nil)
	return nova.New(osc)
}

func (s *ToolSuite) makeServices(c *C) *nova.Client {
	ident := identityservice.NewUserPass()
	token := ident.AddUser(username, password)
	// GZ 2013-01-21: Current novaservice double requires magic url like so
	computeurl := s.Server.URL + "/v2.0/" + tenant
	ident.AddService(identityservice.Service{
		"nova",
		"compute",
		[]identityservice.Endpoint{
			{
				AdminURL:    computeurl,
				InternalURL: computeurl,
				PublicURL:   computeurl,
				Region:      region,
			},
		}})
	s.Mux.Handle("/tokens", ident)
	comp := novaservice.New("unused.invalid", "v2.0", token, tenant)
	comp.SetupHTTP(s.Mux)
	return createNovaClient(s.Server.URL)
}

func (s *ToolSuite) TestNoGroups(c *C) {
	nova := s.makeServices(c)
	var buf bytes.Buffer
	err := tool.DeleteAll(&buf, nova)
	c.Assert(err, IsNil)
	c.Assert(string(buf.Bytes()), Equals, "No security groups to delete.\n")
}

func (s *ToolSuite) TestTwoGroups(c *C) {
	nova := s.makeServices(c)
	nova.CreateSecurityGroup("group-a", "A group")
	nova.CreateSecurityGroup("group-b", "Another group")
	var buf bytes.Buffer
	err := tool.DeleteAll(&buf, nova)
	c.Assert(err, IsNil)
	c.Assert(string(buf.Bytes()), Equals, "2 security groups deleted.\n")
}

// GZ 2013-01-21: Should also test undeleteable groups, but can't induce
//                novaservice errors currently.
