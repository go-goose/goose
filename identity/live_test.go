package identity_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
)

func registerOpenStackTests(cred *identity.Credentials) {
	Suite(&LiveTests{
		cred: cred,
	})
}

type LiveTests struct {
	cred   *identity.Credentials
	client client.AuthenticatingClient
}

func (s *LiveTests) SetUpSuite(c *C) {
	s.client = client.NewClient(s.cred, identity.AuthUserPass, nil)
}

func (s *LiveTests) TearDownSuite(c *C) {
}

func (s *LiveTests) SetUpTest(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TearDownTest(c *C) {
	// noop, called by local test suite.
}

func (s *LiveTests) TestAuth(c *C) {
	s.client.Authenticate()
        url, err := s.client.MakeServiceURL("compute", []string{})
        c.Assert(err, IsNil)
        c.Assert(url[:len(s.cred.URL)], Equals, s.cred.URL)
}
