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
	emptyHeaders := http.Header{}
	headers := createHeaders(emptyHeaders, "content-type")
	contentTypes := []string{"content-type"}
	c.Assert(headers, DeepEquals,
		http.Header{"Content-Type": contentTypes, "Accept": contentTypes, "User-Agent": []string{gooseAgent()}})
	c.Assert(emptyHeaders, DeepEquals, http.Header{})
}

func (s *HTTPClientTestSuite) TestCreateHeadersCopiesSupplied(c *C) {
	initialHeaders := make(http.Header)
	initialHeaders["Foo"] = []string{"Bar"}
	contentType := contentTypeJSON
	contentTypes := []string{contentType}
	headers := createHeaders(initialHeaders, contentType)
	// it should not change the headers passed in
	c.Assert(initialHeaders, DeepEquals, http.Header{"Foo": []string{"Bar"}})
	// The initial headers should be in the output
	c.Assert(headers, DeepEquals,
		http.Header{"Foo": []string{"Bar"}, "Content-Type": contentTypes, "Accept": contentTypes, "User-Agent": []string{gooseAgent()}})
}
