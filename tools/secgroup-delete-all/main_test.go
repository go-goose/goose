package main

import (
	"bytes"
	"fmt"
	"testing"

	gc "gopkg.in/check.v1"

	"github.com/go-goose/goose/v5/client"
	"github.com/go-goose/goose/v5/identity"
	"github.com/go-goose/goose/v5/nova"
	"github.com/go-goose/goose/v5/testing/httpsuite"
	"github.com/go-goose/goose/v5/testservices/hook"
	"github.com/go-goose/goose/v5/testservices/openstackservice"
)

func Test(t *testing.T) {
	gc.TestingT(t)
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

var _ = gc.Suite(&ToolSuite{})

// GZ 2013-01-21: Should require EnvSuite for this, but clashes with HTTPSuite
func createNovaClientFromCreds(creds *identity.Credentials) *nova.Client {
	osc := client.NewClient(creds, identity.AuthUserPass, nil)
	return nova.New(osc)
}

func (s *ToolSuite) makeServices(c *gc.C) (*openstackservice.Openstack, *nova.Client) {
	creds := &identity.Credentials{
		URL:        s.Server.URL,
		User:       username,
		Secrets:    password,
		Region:     region,
		TenantName: tenant,
	}
	openstack, _ := openstackservice.New(creds, identity.AuthUserPass, false)
	openstack.SetupHTTP(s.Mux)
	return openstack, createNovaClientFromCreds(creds)
}

func (s *ToolSuite) TestNoGroups(c *gc.C) {
	_, nova := s.makeServices(c)
	var buf bytes.Buffer
	err := DeleteAll(&buf, nova)
	c.Assert(err, gc.IsNil)
	c.Assert(string(buf.Bytes()), gc.Equals, "No security groups to delete.\n")
}

func (s *ToolSuite) TestTwoGroups(c *gc.C) {
	_, novaClient := s.makeServices(c)
	novaClient.CreateSecurityGroup("group-a", "A group")
	novaClient.CreateSecurityGroup("group-b", "Another group")
	var buf bytes.Buffer
	err := DeleteAll(&buf, novaClient)
	c.Assert(err, gc.IsNil)
	c.Assert(string(buf.Bytes()), gc.Equals, "2 security groups deleted.\n")
}

// This group is one for which we will simulate a deletion error in the following test.
var doNotDelete *nova.SecurityGroup

// deleteGroupError hook raises an error if a group with id 2 is deleted.
func deleteGroupError(s hook.ServiceControl, args ...interface{}) error {
	groupId := args[0].(string)
	if groupId == doNotDelete.Id {
		return fmt.Errorf("cannot delete group %s", groupId)
	}
	return nil
}

func (s *ToolSuite) TestUndeleteableGroup(c *gc.C) {
	os, novaClient := s.makeServices(c)
	novaClient.CreateSecurityGroup("group-a", "A group")
	doNotDelete, _ = novaClient.CreateSecurityGroup("group-b", "Another group")
	novaClient.CreateSecurityGroup("group-c", "Yet another group")
	cleanup := os.Nova.RegisterControlPoint("removeSecurityGroup", deleteGroupError)
	defer cleanup()
	var buf bytes.Buffer
	err := DeleteAll(&buf, novaClient)
	c.Assert(err, gc.IsNil)
	c.Assert(string(buf.Bytes()), gc.Equals, "2 security groups deleted.\n1 security groups could not be deleted.\n")
}
