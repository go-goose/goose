package main_test

import (
	"bytes"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/nova"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/openstack"
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
	creds *identity.Credentials
}

var _ = Suite(&ToolSuite{})

// GZ 2013-01-21: Should require EnvSuite for this, but clashes with HTTPSuite
func createNovaClient(creds *identity.Credentials) *nova.Client {
	osc := client.NewClient(creds, identity.AuthUserPass, nil)
	return nova.New(osc)
}

func (s *ToolSuite) makeServices(c *C) *nova.Client {
	creds := &identity.Credentials{
		URL:        s.Server.URL,
		User:       username,
		Secrets:    password,
		Region:     region,
		TenantName: tenant,
	}
	openstack := openstack.New(creds)
	openstack.SetupHTTP(s.Mux)
	return createNovaClient(creds)
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
