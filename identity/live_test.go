package identity_test

import (
	"net/url"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/client"
	"gopkg.in/goose.v1/identity"
)

func registerOpenStackTests(cred *identity.Credentials) {
	gc.Suite(&LiveTests{
		cred: cred,
	})
}

type LiveTests struct {
	cred   *identity.Credentials
	client client.AuthenticatingClient
}

func (s *LiveTests) SetUpSuite(c *gc.C) {
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
}

func (s *LiveTests) TearDownSuite(c *gc.C) {
}

func (s *LiveTests) SetUpTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownTest(c *gc.C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TestAuth(c *gc.C) {
	err := s.client.Authenticate()
	c.Assert(err, gc.IsNil)
	serviceURL, err := s.client.MakeServiceURL("compute", []string{})
	c.Assert(err, gc.IsNil)
	_, err = url.Parse(serviceURL)
	c.Assert(err, gc.IsNil)
}
