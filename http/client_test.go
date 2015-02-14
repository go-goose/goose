package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/testing/httpsuite"
)

func Test(t *testing.T) {
	gc.TestingT(t)
}

type LoopingHTTPSuite struct {
	httpsuite.HTTPSuite
}

func (s *LoopingHTTPSuite) setupLoopbackRequest() (*http.Header, chan string, *Client) {
	var headers http.Header
	bodyChan := make(chan string, 1)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		headers = req.Header
		bodyBytes, _ := ioutil.ReadAll(req.Body)
		req.Body.Close()
		bodyChan <- string(bodyBytes)
		resp.Header().Add("Content-Length", "0")
		resp.Header().Add("Testing", "true")
		resp.WriteHeader(http.StatusNoContent)
		resp.Write([]byte{})
	}
	s.Mux.HandleFunc("/", handler)
	client := New()
	return &headers, bodyChan, client
}

type HTTPClientTestSuite struct {
	LoopingHTTPSuite
}

type HTTPSClientTestSuite struct {
	LoopingHTTPSuite
}

var _ = gc.Suite(&HTTPClientTestSuite{})
var _ = gc.Suite(&HTTPSClientTestSuite{LoopingHTTPSuite{httpsuite.HTTPSuite{UseTLS: true}}})

func (s *HTTPClientTestSuite) assertHeaderValues(c *gc.C, token string) {
	emptyHeaders := http.Header{}
	headers := createHeaders(emptyHeaders, "content-type", token)
	contentTypes := []string{"content-type"}
	headerData := map[string][]string{
		"Content-Type": contentTypes, "Accept": contentTypes, "User-Agent": {gooseAgent()}}
	if token != "" {
		headerData["X-Auth-Token"] = []string{token}
	}
	expectedHeaders := http.Header(headerData)
	c.Assert(headers, gc.DeepEquals, expectedHeaders)
	c.Assert(emptyHeaders, gc.DeepEquals, http.Header{})
}

func (s *HTTPClientTestSuite) TestCreateHeadersNoToken(c *gc.C) {
	s.assertHeaderValues(c, "")
}

func (s *HTTPClientTestSuite) TestCreateHeadersWithToken(c *gc.C) {
	s.assertHeaderValues(c, "token")
}

func (s *HTTPClientTestSuite) TestCreateHeadersCopiesSupplied(c *gc.C) {
	initialHeaders := make(http.Header)
	initialHeaders["Foo"] = []string{"Bar"}
	contentType := contentTypeJSON
	contentTypes := []string{contentType}
	headers := createHeaders(initialHeaders, contentType, "")
	// it should not change the headers passed in
	c.Assert(initialHeaders, gc.DeepEquals, http.Header{"Foo": []string{"Bar"}})
	// The initial headers should be in the output
	c.Assert(headers, gc.DeepEquals,
		http.Header{"Foo": []string{"Bar"}, "Content-Type": contentTypes, "Accept": contentTypes, "User-Agent": []string{gooseAgent()}})
}

func (s *HTTPClientTestSuite) TestBinaryRequestSetsUserAgent(c *gc.C) {
	headers, _, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, gc.Not(gc.Equals), "")
	c.Check(agent, gc.Equals, gooseAgent())
	c.Check(req.RespHeaders.Get("Testing"), gc.Equals, "true")
}

func (s *HTTPClientTestSuite) TestJSONRequestSetsUserAgent(c *gc.C) {
	headers, _, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.JsonRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, gc.Not(gc.Equals), "")
	c.Check(agent, gc.Equals, gooseAgent())
	c.Check(req.RespHeaders.Get("Testing"), gc.Equals, "true")
}

func (s *HTTPClientTestSuite) TestBinaryRequestSetsContentLength(c *gc.C) {
	headers, bodyChan, client := s.setupLoopbackRequest()
	content := "binary\ncontent\n"
	req := &RequestData{
		ExpectedStatus: []int{http.StatusNoContent},
		ReqReader:      bytes.NewBufferString(content),
		ReqLength:      len(content),
	}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	encoding := headers.Get("Transfer-Encoding")
	c.Check(encoding, gc.Equals, "")
	length := headers.Get("Content-Length")
	c.Check(length, gc.Equals, fmt.Sprintf("%d", len(content)))
	body, ok := <-bodyChan
	c.Assert(ok, gc.Equals, true)
	c.Check(body, gc.Equals, content)
}

func (s *HTTPClientTestSuite) TestJSONRequestSetsContentLength(c *gc.C) {
	headers, bodyChan, client := s.setupLoopbackRequest()
	reqMap := map[string]string{"key": "value"}
	req := &RequestData{
		ExpectedStatus: []int{http.StatusNoContent},
		ReqValue:       reqMap,
	}
	err := client.JsonRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	encoding := headers.Get("Transfer-Encoding")
	c.Check(encoding, gc.Equals, "")
	length := headers.Get("Content-Length")
	body, ok := <-bodyChan
	c.Assert(ok, gc.Equals, true)
	c.Check(body, gc.Not(gc.Equals), "")
	c.Check(length, gc.Equals, fmt.Sprintf("%d", len(body)))
}

func (s *HTTPClientTestSuite) TestBinaryRequestSetsToken(c *gc.C) {
	headers, _, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "token", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("X-Auth-Token")
	c.Check(agent, gc.Equals, "token")
}

func (s *HTTPClientTestSuite) TestJSONRequestSetsToken(c *gc.C) {
	headers, _, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.JsonRequest("POST", s.Server.URL, "token", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("X-Auth-Token")
	c.Check(agent, gc.Equals, "token")
}

func (s *HTTPClientTestSuite) TestHttpTransport(c *gc.C) {
	transport := http.DefaultTransport.(*http.Transport)
	c.Assert(transport.DisableKeepAlives, gc.Equals, true)
}

func (s *HTTPSClientTestSuite) TestDefaultClientRejectSelfSigned(c *gc.C) {
	_, _, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.NotNil)
	c.Check(err, gc.ErrorMatches, "(.|\\n)*x509: certificate signed by unknown authority")
}

func (s *HTTPSClientTestSuite) TestInsecureClientAllowsSelfSigned(c *gc.C) {
	headers, _, _ := s.setupLoopbackRequest()
	client := NewNonSSLValidating()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, gc.Not(gc.Equals), "")
	c.Check(agent, gc.Equals, gooseAgent())
}

func (s *HTTPSClientTestSuite) TestProperlyFormattedJsonUnmarshalling(c *gc.C) {
	validJSON := `{"itemNotFound": {"message": "A Meaningful error", "code": 404}}`
	unmarshalled, err := unmarshallError([]byte(validJSON))
	c.Assert(err, gc.IsNil)
	c.Check(unmarshalled.Code, gc.Equals, 404)
	c.Check(unmarshalled.Title, gc.Equals, "itemNotFound")
	c.Check(unmarshalled.Message, gc.Equals, "A Meaningful error")
}

func (s *HTTPSClientTestSuite) TestImproperlyFormattedJSONUnmarshalling(c *gc.C) {
	invalidJSON := `This string is not a valid JSON`
	unmarshalled, err := unmarshallError([]byte(invalidJSON))
	c.Assert(err, gc.NotNil)
	c.Assert(unmarshalled, gc.IsNil)
	c.Check(err, gc.ErrorMatches, "invalid character 'T' looking for beginning of value")
}

func (s *HTTPSClientTestSuite) TestJSONMissingCodeUnmarshalling(c *gc.C) {
	missingCodeJSON := `{"itemNotFound": {"message": "A Meaningful error"}}`
	unmarshalled, err := unmarshallError([]byte(missingCodeJSON))
	c.Assert(err, gc.NotNil)
	c.Assert(unmarshalled, gc.IsNil)
	c.Check(err, gc.ErrorMatches, `Unparsable json error body: "{\\"itemNotFound\\": {\\"message\\": \\"A Meaningful error\\"}}"`)
}

func (s *HTTPSClientTestSuite) TestJSONMissingMessageUnmarshalling(c *gc.C) {
	missingMessageJSON := `{"itemNotFound": {"code": 404}}`
	unmarshalled, err := unmarshallError([]byte(missingMessageJSON))
	c.Assert(err, gc.NotNil)
	c.Assert(unmarshalled, gc.IsNil)
	c.Check(err, gc.ErrorMatches, `Unparsable json error body: "{\\"itemNotFound\\": {\\"code\\": 404}}"`)
}

func (s *HTTPSClientTestSuite) TestBrokenBodyJSONUnmarshalling(c *gc.C) {
	invalidBodyJSON := `{"itemNotFound": {}}`
	unmarshalled, err := unmarshallError([]byte(invalidBodyJSON))
	c.Assert(err, gc.NotNil)
	c.Assert(unmarshalled, gc.IsNil)
	c.Check(err, gc.ErrorMatches, `Unparsable json error body: \"{\\\"itemNotFound\\\": {}}\"`)
}
