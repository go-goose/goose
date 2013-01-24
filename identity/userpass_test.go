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
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant")
	var l Authenticator = &UserPass{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL + "/tokens", Secrets: "secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, IsNil)
	c.Assert(auth.Token, Equals, userInfo.Token)
	c.Assert(auth.TenantId, Equals, userInfo.TenantId)
}
