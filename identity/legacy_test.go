package identity

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type LegacyTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&LegacyTestSuite{})

func (s *LegacyTestSuite) TestSomething(c *C) {
}
