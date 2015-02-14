package identity

import (
	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/testing/httpsuite"
	"gopkg.in/goose.v1/testservices/identityservice"
)

type LegacyTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = gc.Suite(&LegacyTestSuite{})

func (s *LegacyTestSuite) TestAuthAgainstServer(c *gc.C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	userInfo := service.AddUser("joe-user", "secrets", "tenant")
	service.SetManagementURL("http://management.test.invalid/url")
	var l Authenticator = &Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(
		auth.RegionServiceURLs[""], gc.DeepEquals,
		ServiceURLs{"compute": "http://management.test.invalid/url/compute",
			"object-store": "http://management.test.invalid/url/object-store"})
}

func (s *LegacyTestSuite) TestBadAuth(c *gc.C) {
	service := identityservice.NewLegacy()
	s.Mux.Handle("/", service)
	_ = service.AddUser("joe-user", "secrets", "tenant")
	var l Authenticator = &Legacy{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "bad-secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.NotNil)
	c.Assert(auth, gc.IsNil)
}
