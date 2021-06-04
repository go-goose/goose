package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v3/testing/httpsuite"
)

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

var _ = gc.Suite(&HTTPClientTestSuite{})

func (s *HTTPClientTestSuite) assertHeaderValues(c *gc.C, token string) {
	emptyHeaders := http.Header{}
	headers := DefaultHeaders("GET", emptyHeaders, "content-type", token, true)
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
	headers := DefaultHeaders("GET", initialHeaders, contentType, "", true)
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
		ReqReader:      strings.NewReader(content),
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

func (s *HTTPClientTestSuite) TestJSONRequestNoPayload(c *gc.C) {
	headers, bodyChan, client := s.setupLoopbackRequest()
	req := &RequestData{
		ExpectedStatus: []int{http.StatusNoContent},
		ReqValue:       nil,
	}
	err := client.JsonRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	encoding := headers.Get("Transfer-Encoding")
	c.Check(encoding, gc.Equals, "")
	length := headers.Get("Content-Length")
	ctype := headers.Get("Content-Type")
	body, ok := <-bodyChan
	c.Assert(ok, gc.Equals, true)
	c.Check(body, gc.Equals, "")
	c.Check(length, gc.Equals, "0")
	c.Check(ctype, gc.Equals, "")
	c.Check(req.RespStatusCode, gc.Equals, http.StatusNoContent)
}

func (s *HTTPClientTestSuite) TestBinaryRequestSetsToken(c *gc.C) {
	headers, _, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "token", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("X-Auth-Token")
	c.Check(agent, gc.Equals, "token")
	c.Check(req.RespStatusCode, gc.Equals, http.StatusNoContent)
}

func (s *HTTPClientTestSuite) TestJSONRequestSetsToken(c *gc.C) {
	headers, _, client := s.setupLoopbackRequest()
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.JsonRequest("POST", s.Server.URL, "token", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("X-Auth-Token")
	c.Check(agent, gc.Equals, "token")
}

func (s *HTTPClientTestSuite) testRetryAfter(c *gc.C,
	retryAfter func(*time.Time, http.ResponseWriter),
	verifyWait func(time.Time) (time.Duration, bool)) {
	count := 0
	var t0 time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		data, _ := ioutil.ReadAll(req.Body)
		c.Check(string(data), gc.Equals, "request body")
		count++
		switch count {
		case 1:
			t0 = time.Now()
			retryAfter(&t0, w)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		case 2:
			if waitDuration, fail := verifyWait(t0); fail {
				c.Errorf("client did not wait long enough (expected 1s got %v)", waitDuration)
			}
			w.Write([]byte(`hello`))
		default:
			c.Errorf("handler too many times")
		}
	}))
	defer srv.Close()
	client := New()
	req := &RequestData{
		ExpectedStatus: []int{http.StatusOK},
		ReqReader:      strings.NewReader("request body"),
		ReqLength:      len("request body"),
		RespReader:     nopReadCloser{},
	}
	err := client.BinaryRequest("POST", srv.URL, "", req, nil)
	c.Assert(err, gc.Equals, nil)
	c.Assert(count, gc.Equals, 2)

	defer req.RespReader.Close()
	data, _ := ioutil.ReadAll(req.RespReader)
	c.Assert(string(data), gc.Equals, "hello")

	// Try without a seeker for the request body.
	count = 0
	req = &RequestData{
		ExpectedStatus: []int{http.StatusOK},
		ReqReader:      struct{ io.Reader }{strings.NewReader("request body")},
		ReqLength:      len("request body"),
		RespReader:     nopReadCloser{},
	}
	err = client.BinaryRequest("POST", srv.URL, "", req, nil)
	c.Assert(err, gc.Equals, nil)
	c.Assert(count, gc.Equals, 2)

	defer req.RespReader.Close()
	data, _ = ioutil.ReadAll(req.RespReader)
	c.Assert(string(data), gc.Equals, "hello")
}

func (s *HTTPClientTestSuite) TestRetryDelaySeconds(c *gc.C) {
	retryAfter := func(_ *time.Time, w http.ResponseWriter) {
		w.Header().Set("Retry-After", "1")
	}
	verifyWait := func(t time.Time) (time.Duration, bool) {
		waitDuration := time.Since(t)
		return waitDuration, waitDuration < time.Second
	}
	s.testRetryAfter(c, retryAfter, verifyWait)
}

func (s *HTTPClientTestSuite) TestRetryHttpDate(c *gc.C) {
	retryAfter := func(t *time.Time, w http.ResponseWriter) {
		*t = t.Add(time.Duration(1) * time.Second)
		w.Header().Set("Retry-After", t.Format(time.RFC1123))
	}
	verifyWait := func(t time.Time) (time.Duration, bool) {
		return time.Duration(1) * time.Second, time.Now().Unix() < t.Unix()
	}
	s.testRetryAfter(c, retryAfter, verifyWait)

}

func (s *HTTPClientTestSuite) testRetryAfterCheckLimits(c *gc.C, retryAfter, errorMatch string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	client := New()
	req := &RequestData{
		ExpectedStatus: []int{http.StatusOK},
	}
	err := client.JsonRequest("POST", srv.URL, "", req, nil)
	c.Assert(err, gc.ErrorMatches, errorMatch)
}

func (s *HTTPClientTestSuite) TestResourceLimitExceeded(c *gc.C) {
	s.testRetryAfterCheckLimits(c, "0", `Resource limit exceeded at URL http://.*`)
}

func (s *HTTPClientTestSuite) TestHttpDateTenMinutes(c *gc.C) {
	t0 := time.Now()
	t0 = t0.Add(time.Duration(11) * time.Minute)
	s.testRetryAfterCheckLimits(c,
		t0.Format(time.RFC1123),
		fmt.Sprintf(`Cloud is not accepting further requests from this account until %s`, t0.Format(time.UnixDate)))
}

type HTTPSClientTestSuite struct {
	LoopingHTTPSuite
}

var _ = gc.Suite(&HTTPSClientTestSuite{})

func (s *HTTPSClientTestSuite) SetUpSuite(c *gc.C) {
	s.UseTLS = true
	s.LoopingHTTPSuite.SetUpSuite(c)
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
	client := New(WithHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}))
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, gc.Not(gc.Equals), "")
	c.Check(agent, gc.Equals, gooseAgent())
}

func (s *HTTPSClientTestSuite) TestTSLConfigClient(c *gc.C) {
	headers, _, _ := s.setupLoopbackRequest()
	tlsConfig := s.tlsConfig()
	client := New(WithHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}))
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Assert(err, gc.IsNil)
	agent := headers.Get("User-Agent")
	c.Check(agent, gc.Not(gc.Equals), "")
	c.Check(agent, gc.Equals, gooseAgent())
}

func (s *HTTPSClientTestSuite) tlsConfig() *tls.Config {
	pool := x509.NewCertPool()
	pool.AddCert(s.Server.Certificate())
	return &tls.Config{
		RootCAs: pool,
	}
}

func (s *HTTPSClientTestSuite) TestTSLConfigClientNoCert(c *gc.C) {
	s.setupLoopbackRequest()
	client := New(WithHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
		},
	}))
	req := &RequestData{ExpectedStatus: []int{http.StatusNoContent}}
	err := client.BinaryRequest("POST", s.Server.URL, "", req, nil)
	c.Check(err, gc.ErrorMatches, "(.|\\n)*x509: certificate signed by unknown authority")
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

type nopReadCloser struct{}

func (nopReadCloser) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (nopReadCloser) Close() error {
	return nil
}
