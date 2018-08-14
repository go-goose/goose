package client_test

import (
	gc "gopkg.in/check.v1"
	"gopkg.in/goose.v2/client"
	gooseerrors "gopkg.in/goose.v2/errors"
	"gopkg.in/goose.v2/identity"
)

func registerOpenStackTests(cred *identity.Credentials, authModes []identity.AuthMode) {
	for _, authMode := range authModes {
		gc.Suite(&LiveTests{
			cred:     cred,
			authMode: authMode,
		})
	}
}

type LiveTests struct {
	cred     *identity.Credentials
	authMode identity.AuthMode
}

func (s *LiveTests) SetUpSuite(c *gc.C) {
	c.Logf("Running tests with authentication method %v", s.authMode)
}

func (s *LiveTests) TearDownSuite(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) SetUpTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TestAuthenticateFail(c *gc.C) {
	cred := *s.cred
	cred.User = "fred"
	cred.Secrets = "broken"
	cred.Region = ""
	osclient := client.NewClient(&cred, s.authMode, nil)
	c.Assert(osclient.IsAuthenticated(), gc.Equals, false)
	err := osclient.Authenticate()
	c.Assert(err, gc.ErrorMatches, "authentication failed(\n|.)*")
}

func (s *LiveTests) TestNoUnauthorisedAuthenticateFail(c *gc.C) {
	cred := *s.cred
	cred.Region = ""
	osclient := client.NewClient(&cred, s.authMode, nil)
	c.Assert(osclient.IsAuthenticated(), gc.Equals, false)
	err := osclient.Authenticate()
	isAuthorised := gooseerrors.IsUnauthorised(err)
	c.Assert(isAuthorised, gc.Equals, false)
}

func (s *LiveTests) TestAuthenticate(c *gc.C) {
	cl := client.NewClient(s.cred, s.authMode, nil)
	err := cl.Authenticate()
	c.Assert(err, gc.IsNil)
	c.Assert(cl.IsAuthenticated(), gc.Equals, true)

	// Check service endpoints are discovered
	if s.authMode == identity.AuthLegacy {
		// AuthLegacy doesn't use the openstack test double, therefore
		// MakeServiceURL won't work correctly as the endpoint urls are used,
		// but setup as bad addresses for AuthLegacy
		c.Log("half of this test will not work with legacy auth")
	} else {
		url, err := cl.MakeServiceURL("compute", "v2", nil)
		c.Check(err, gc.IsNil)
		c.Check(url, gc.NotNil)
		url, err = cl.MakeServiceURL("object-store", "", nil)
		c.Check(err, gc.IsNil)
		c.Check(url, gc.NotNil)
	}
}

func (s *LiveTests) TestAuthDiscover(c *gc.C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("this test will not work with legacy auth")
	}
	cl := client.NewClient(s.cred, s.authMode, nil)
	options, err := cl.IdentityAuthOptions()
	c.Assert(err, gc.IsNil)
	optionsAvailable := len(options) > 0
	c.Assert(optionsAvailable, gc.Equals, true)
	for _, option := range options {
		switch option.Mode {
		case identity.AuthUserPass, identity.AuthUserPassV3:
		default:
			c.Logf("unknown identity AuthMode %v", option)
			c.FailNow()
		}
	}
}
