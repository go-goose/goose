package http

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"net/http"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type HTTPClientTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&HTTPClientTestSuite{})

func (s *HTTPClientTestSuite) TestCreateHeaders(c *C) {
	headers := createHeaders(make(http.Header), "content-type")
	content_types := []string{"content-type"}
	c.Assert(headers, DeepEquals, http.Header{"Content-Type": content_types, "Accept": content_types})
}
