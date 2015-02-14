// Copyright 2013 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package goose

import (
	gc "gopkg.in/check.v1"
)

type VersionTestSuite struct {
}

var _ = gc.Suite(&VersionTestSuite{})

func (s *VersionTestSuite) TestStringMatches(c *gc.C) {
	c.Assert(Version, gc.Equals, VersionNumber.String())
}
