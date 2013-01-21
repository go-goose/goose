package main_test

import (
	"bytes"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/novaservice"
	tool "launchpad.net/goose/tools/secgroup-delete-all"
	"os"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

const (
	username = "auser"
	password = "apass"
	region   = "aregion"
)

type ToolSuite struct {
	httpsuite.HTTPSuite
	Nova *novaservice.Nova
}

var _ = Suite(&ToolSuite{})

// GZ 2013-01-21: Should require EnvSuite for this, but clashes with HTTPSuite
func prepareEnv(auth_url string) {
	os.Setenv("OS_AUTH_URL", auth_url)
	os.Setenv("OS_USERNAME", username)
	os.Setenv("OS_PASSWORD", password)
	os.Setenv("OS_REGION_NAME", region)
}

func (s *ToolSuite) makeServices(c *C) {
	ident := identityservice.NewUserPass()
	token := ident.AddUser(username, password)
	// GZ 2013-01-21: Current novaservice double requires magic url like so
	computeurl := s.Server.URL + "/v2.0/1"
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
	prepareEnv(s.Server.URL)
	s.Mux.Handle("/tokens", ident)
	comp := novaservice.New("unused.invalid", "v2.0", token, "1")
	comp.SetupHTTP(s.Mux)
	s.Nova = comp
}

func (s *ToolSuite) TestNoGroups(c *C) {
	s.makeServices(c)
	var buf bytes.Buffer
	err := tool.DeleteAll(&buf, identity.AuthUserPass)
	c.Assert(err, IsNil)
	c.Assert(string(buf.Bytes()), Equals, "No security groups to delete.\n")
}

func (s *ToolSuite) TestTwoGroups(c *C) {
	s.makeServices(c)
	s.Nova.MakeSecurityGroup("group-a", "A group")
	s.Nova.MakeSecurityGroup("group-b", "Another group")
	var buf bytes.Buffer
	err := tool.DeleteAll(&buf, identity.AuthUserPass)
	c.Assert(err, IsNil)
	c.Assert(string(buf.Bytes()), Equals, "2 security groups deleted.\n")
}

// GZ 2013-01-21: Should also test undeleteable groups, but can't induce
//                novaservice errors currently.
