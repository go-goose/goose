package identityservice

import (
	. "launchpad.net/gocheck"
)

type UtilSuite struct{}

var _ = Suite(&UtilSuite{})

func (s *UtilSuite) TestRandomHexTokenHasLength(c *C) {
	val := randomHexToken()
	c.Assert(val, HasLen, 32)
}

func (s *UtilSuite) TestRandomHexTokenIsHex(c *C) {
	val := randomHexToken()
	for i, b := range val {
		switch {
		case (b >= 'a' && b <= 'f') || (b >= '0' && b <= '9'):
			continue
		default:
			c.Logf("char %d was not in the right range: '%c'",
				i, b)
			c.Fail()
		}
	}
}
