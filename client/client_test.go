package client_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"reflect"
	"testing"
)

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")

type ClientSuite struct {
	client *client.OpenStackClient
}

func (s *ClientSuite) SetUpSuite(c *C) {
	if !*live {
		c.Skip("-live not provided")
	}

	cred := identity.CompleteCredentialsFromEnv()
	s.client = client.NewOpenStackClient(cred, identity.AuthUserPass)
}

var suite = Suite(&ClientSuite{})

func (s *ClientSuite) TestAuthenticateFail(c *C) {
	cred := identity.CompleteCredentialsFromEnv()
	cred.User = "fred"
	cred.Secrets = "broken"
	cred.Region = ""
	var osclient *client.OpenStackClient = client.NewOpenStackClient(cred, identity.AuthUserPass)
	c.Assert(osclient.IsAuthenticated(), Equals, false)
	var err error
	err = osclient.Authenticate()
	c.Assert(err, ErrorMatches, "authentication failed.*")
}

func (s *ClientSuite) TestAuthenticate(c *C) {
	var err error
	err = s.client.Authenticate()
	c.Assert(err, IsNil)
	c.Assert(s.client.IsAuthenticated(), Equals, true)

	// Check service endpoints are discovered
	c.Assert(s.client.ServiceURLs["compute"], NotNil)
	c.Assert(s.client.ServiceURLs["swift"], NotNil)
}
