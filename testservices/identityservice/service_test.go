package identityservice

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
)

// All tests in the IdentityServiceSuite run against each IdentityService
// implementation.

type IdentityServiceSuite struct {
	httpsuite.HTTPSuite
	service IdentityService
}

var _ = Suite(&IdentityServiceSuite{service: NewUserPass()})
var _ = Suite(&IdentityServiceSuite{service: NewLegacy()})

func (s *IdentityServiceSuite) TestAddUserGivesNewToken(c *C) {
	token1 := s.service.AddUser("user-1", "password-1")
	token2 := s.service.AddUser("user-2", "password-2")
	c.Assert(token1, Not(Equals), token2)
}
