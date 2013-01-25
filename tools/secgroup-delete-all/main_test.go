package main_test

import (
	"bytes"
	"fmt"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/nova"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices"
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

func (s *ToolSuite) makeServices(c *C) (*openstack.Openstack, *nova.Client) {
	creds := &identity.Credentials{
		URL:        s.Server.URL,
		User:       username,
		Secrets:    password,
		Region:     region,
		TenantName: tenant,
	}
	openstack := openstack.New(creds)
	openstack.SetupHTTP(s.Mux)
	return openstack, createNovaClient(creds)
}

func (s *ToolSuite) TestNoGroups(c *C) {
	_, nova := s.makeServices(c)
	var buf bytes.Buffer
	err := tool.DeleteAll(&buf, nova)
	c.Assert(err, IsNil)
	c.Assert(string(buf.Bytes()), Equals, "No security groups to delete.\n")
}

func (s *ToolSuite) TestTwoGroups(c *C) {
	_, nova := s.makeServices(c)
	nova.CreateSecurityGroup("group-a", "A group")
	nova.CreateSecurityGroup("group-b", "Another group")
	var buf bytes.Buffer
	err := tool.DeleteAll(&buf, nova)
	c.Assert(err, IsNil)
	c.Assert(string(buf.Bytes()), Equals, "2 security groups deleted.\n")
}

// deleteGroupError hook raises an error if a group with id 2 is deleted.
func deleteGroupError(s testservices.ServiceControl, args ...interface{}) error {
	groupId := args[0]
	if groupId == 2 {
		return fmt.Errorf("cannot delete group %d", groupId)
	}
	return nil
}

func (s *ToolSuite) TestUndeleteableGroup(c *C) {
	os, nova := s.makeServices(c)
	nova.CreateSecurityGroup("group-a", "A group")
	nova.CreateSecurityGroup("group-b", "Another group")
	nova.CreateSecurityGroup("group-c", "Yet another group")
	os.Nova.RegisterControlPoint("removeSecurityGroup", deleteGroupError)
	defer os.Nova.RegisterControlPoint("removeSecurityGroup", nil)
	var buf bytes.Buffer
	err := tool.DeleteAll(&buf, nova)
	c.Assert(err, IsNil)
	c.Assert(string(buf.Bytes()), Equals, "2 security groups deleted.\n1 security groups could not be deleted.\n")
}
