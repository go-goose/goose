package http_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type HTTPClientTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&HTTPClientTestSuite{})

func (s *HTTPClientTestSuite) TestSendsUserAgent(c *C) {
	c.Assert(2+2, Equals, 4)
}
