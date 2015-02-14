package identity

import (
	. "gopkg.in/check.v1"
	"gopkg.in/goose.v1/testing/httpsuite"
	"gopkg.in/goose.v1/testservices/identityservice"
)

type LegacyTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&LegacyTestSuite{})

func (s *LegacyTestSuite) TestAuthAgainstServer(c *C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	userInfo := service.AddUser("joe-user", "secrets", "tenant")
	service.SetManagementURL("http://management.test.invalid/url")
	var l Authenticator = &Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, IsNil)
	c.Assert(auth.Token, Equals, userInfo.Token)
	c.Assert(
		auth.RegionServiceURLs[""], DeepEquals,
		ServiceURLs{"compute": "http://management.test.invalid/url/compute",
			"object-store": "http://management.test.invalid/url/object-store"})
}

func (s *LegacyTestSuite) TestBadAuth(c *C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	_ = service.AddUser("joe-user", "secrets", "tenant")
	var l Authenticator = &Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "bad-secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, NotNil)
	c.Assert(auth, IsNil)
}
