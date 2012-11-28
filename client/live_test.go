package client_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
)

func registerOpenStackTests(cred *identity.Credentials, authMethod identity.AuthMethod) {
	Suite(&LiveTests{
		cred:       cred,
		authMethod: authMethod,
	})
}

type LiveTests struct {
	cred       *identity.Credentials
	authMethod identity.AuthMethod
}

func (s *LiveTests) SetUpSuite(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownSuite(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) SetUpTest(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownTest(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TestAuthenticateFail(c *C) {
	cred := *s.cred
	cred.User = "fred"
	cred.Secrets = "broken"
	cred.Region = ""
	var osclient *client.OpenStackClient = client.NewOpenStackClient(&cred, s.authMethod)
	c.Assert(osclient.IsAuthenticated(), Equals, false)
	err := osclient.Authenticate()
	c.Assert(err, ErrorMatches, "authentication failed.*")
}

func (s *LiveTests) TestAuthenticate(c *C) {
	client := client.NewOpenStackClient(s.cred, s.authMethod)
	err := client.Authenticate()
	c.Assert(err, IsNil)
	c.Assert(client.IsAuthenticated(), Equals, true)

	// Check service endpoints are discovered
	c.Assert(client.ServiceURLs["compute"], NotNil)
	c.Assert(client.ServiceURLs["swift"], NotNil)
}
