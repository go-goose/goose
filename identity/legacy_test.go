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
	userInfo := service.AddUser("joe-user", "secrets")
	service.SetManagementURL("http://management.test.invalid/url")
	var l Authenticator = &Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, IsNil)
	c.Assert(auth.Token, Equals, userInfo.Token)
	c.Assert(
		auth.ServiceURLs, DeepEquals,
		map[string]string{"compute": "http://management.test.invalid/url/compute", "object-store": "http://management.test.invalid/url/object-store"})
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
