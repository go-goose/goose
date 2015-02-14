package httpsuite

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	gc "gopkg.in/check.v1"
)

type HTTPTestSuite struct {
	HTTPSuite
}

type HTTPSTestSuite struct {
	HTTPSuite
}

func Test(t *testing.T) {
	gc.TestingT(t)
}

var _ = gc.Suite(&HTTPTestSuite{})
var _ = gc.Suite(&HTTPSTestSuite{HTTPSuite{UseTLS: true}})

type HelloHandler struct{}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write([]byte("Hello World\n"))
}

func (s *HTTPTestSuite) TestHelloWorld(c *gc.C) {
	s.Mux.Handle("/", &HelloHandler{})
	// fmt.Printf("Running HelloWorld\n")
	response, err := http.Get(s.Server.URL)
	c.Check(err, gc.IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(err, gc.IsNil)
	c.Check(response.Status, gc.Equals, "200 OK")
	c.Check(response.StatusCode, gc.Equals, 200)
	c.Check(string(content), gc.Equals, "Hello World\n")
}

func (s *HTTPSTestSuite) TestHelloWorldWithTLS(c *gc.C) {
	s.Mux.Handle("/", &HelloHandler{})
	c.Check(s.Server.URL[:8], gc.Equals, "https://")
	response, err := http.Get(s.Server.URL)
	// Default http.Get fails because the cert is self-signed
	c.Assert(err, gc.NotNil)
	c.Assert(reflect.TypeOf(err.(*url.Error).Err), gc.Equals, reflect.TypeOf(x509.UnknownAuthorityError{}))
	// Connect again with a Client that doesn't validate the cert
	insecureClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	response, err = insecureClient.Get(s.Server.URL)
	c.Assert(err, gc.IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(err, gc.IsNil)
	c.Check(response.Status, gc.Equals, "200 OK")
	c.Check(response.StatusCode, gc.Equals, 200)
	c.Check(string(content), gc.Equals, "Hello World\n")
}
