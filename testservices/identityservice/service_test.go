package identityservice

import (
	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/testing/httpsuite"
)

// All tests in the IdentityServiceSuite run against each IdentityService
// implementation.

type IdentityServiceSuite struct {
	httpsuite.HTTPSuite
	service IdentityService
}

var _ = gc.Suite(&IdentityServiceSuite{service: NewUserPass()})
var _ = gc.Suite(&IdentityServiceSuite{service: NewLegacy()})

func (s *IdentityServiceSuite) TestAddUserGivesNewToken(c *gc.C) {
	userInfo1 := s.service.AddUser("user-1", "password-1", "tenant")
	userInfo2 := s.service.AddUser("user-2", "password-2", "tenant")
	c.Assert(userInfo1.Token, gc.Not(gc.Equals), userInfo2.Token)
}
