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

func (s *HTTPClientTestSuite) setupLoopbackRequest() (*http.Header, *Client) {
	headers := http.Header{}
	handler := func(resp http.ResponseWriter, req *http.Request) {
		headers = req.Header
		resp.Header().Add("Content-Length", "0")
		resp.WriteHeader(http.StatusNoContent)
		resp.Write([]byte{})
	}
	s.Mux.HandleFunc("/", handler)
	client := New(*http.DefaultClient, nil, "")
	return &headers, client
}

func (s *HTTPClientTestSuite) TestBinaryRequestSetsUserAgent(c *C) {
	headers, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, req)
	c.Assert(err, IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, Not(Equals), "")
	c.Check(agent, Equals, gooseAgent())
}

func (s *HTTPClientTestSuite) TestJSONRequestSetsUserAgent(c *C) {
	headers, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.JsonRequest("POST", s.Server.URL, req)
	c.Assert(err, IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, Not(Equals), "")
	c.Check(agent, Equals, gooseAgent())
}
