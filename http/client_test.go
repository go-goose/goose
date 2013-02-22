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

func (s *HTTPClientTestSuite) assertHeaderValues(c *C, token string) {
	emptyHeaders := http.Header{}
	headers := createHeaders(emptyHeaders, "content-type", token)
	contentTypes := []string{"content-type"}
	headerData := map[string][]string{
		"Content-Type": contentTypes, "Accept": contentTypes, "User-Agent": []string{gooseAgent()}}
	if token != "" {
		headerData["X-Auth-Token"] = []string{token}
	}
	expectedHeaders := http.Header(headerData)
	c.Assert(headers, DeepEquals, expectedHeaders)
	c.Assert(emptyHeaders, DeepEquals, http.Header{})
}

func (s *HTTPClientTestSuite) TestCreateHeadersNoToken(c *C) {
	s.assertHeaderValues(c, "")
}

func (s *HTTPClientTestSuite) TestCreateHeadersWithToken(c *C) {
	s.assertHeaderValues(c, "token")
}

func (s *HTTPClientTestSuite) TestCreateHeadersCopiesSupplied(c *C) {
	initialHeaders := make(http.Header)
	initialHeaders["Foo"] = []string{"Bar"}
	contentType := contentTypeJSON
	contentTypes := []string{contentType}
	headers := createHeaders(initialHeaders, contentType, "")
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
	client := New()
	return &headers, client
}

func (s *HTTPClientTestSuite) TestBinaryRequestSetsUserAgent(c *C) {
	headers, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, Not(Equals), "")
	c.Check(agent, Equals, gooseAgent())
}

func (s *HTTPClientTestSuite) TestJSONRequestSetsUserAgent(c *C) {
	headers, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.JsonRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, Not(Equals), "")
	c.Check(agent, Equals, gooseAgent())
}

func (s *HTTPClientTestSuite) TestBinaryRequestSetsToken(c *C) {
	headers, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "token", req, nil)
	c.Assert(err, IsNil)
	agent := headers.Get("X-Auth-Token")
	c.Check(agent, Equals, "token")
}

func (s *HTTPClientTestSuite) TestJSONRequestSetsToken(c *C) {
	headers, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.JsonRequest("POST", s.Server.URL, "token", req, nil)
	c.Assert(err, IsNil)
	agent := headers.Get("X-Auth-Token")
	c.Check(agent, Equals, "token")
}
