package goose

import (
	. "launchpad.net/gocheck"
)

type VersionTestSuite struct {
}

var _ = Suite(&VersionTestSuite{})

func (s *VersionTestSuite) TestStringMatches(c *C) {
	c.Assert(Version, Equals, VersionNumber.String())
}
