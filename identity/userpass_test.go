package identity

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
)

type UserPassTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&UserPassTestSuite{})

func (s *UserPassTestSuite) TestAuthAgainstServer(c *C) {
	service := identityservice.NewUserPass()
	s.Mux.Handle("/", service)
	service.AddUser("joe-user", "secrets", "active-token")
	var l Authenticator = &UserPass{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL, Secrets: "secrets"}
	auth, err := l.Auth(creds)
	c.Assert(err, IsNil)
	c.Assert(auth.Token, Equals, "active-token")
	// c.Assert(auth.ServiceURLs, DeepEquals, map[string]string{"compute": "http://management/url"})
}
