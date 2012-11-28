package client_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
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

	cred, err := identity.CompleteCredentialsFromEnv()
	if err != nil {
		c.Fatalf("Error setting up test suite: %s", err.Error())
	}
	s.client = client.NewOpenStackClient(cred, identity.AuthUserPass)
}

var suite = Suite(&ClientSuite{})

func (s *ClientSuite) TestAuthenticateFail(c *C) {
	cred, err := identity.CompleteCredentialsFromEnv()
	if err != nil {
		c.Fatalf(err.Error())
	}
	cred.User = "fred"
	cred.Secrets = "broken"
	cred.Region = ""
	var osclient *client.OpenStackClient = client.NewOpenStackClient(cred, identity.AuthUserPass)
	c.Assert(osclient.IsAuthenticated(), Equals, false)
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
