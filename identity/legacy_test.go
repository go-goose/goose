package identity

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
)

type LegacyTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&LegacyTestSuite{})

func (s *LegacyTestSuite) TestAuthAgainstServer(c *C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	token := service.AddUser("joe-user", "secrets")
	service.SetManagementURL("http://management/url")
	var l Authenticator = &Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, IsNil)
	c.Assert(auth.Token, Equals, token)
	c.Assert(
		auth.ServiceURLs, DeepEquals,
		map[string]string{"compute":"http://management/url/compute", "object-store":"http://management/url/object-store"})
}

func (s *LegacyTestSuite) TestBadAuth(c *C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	_ = service.AddUser("joe-user", "secrets")
	var l Authenticator = &Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "bad-secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, NotNil)
	c.Assert(auth, IsNil)
}
