package identitystub

import (
	. "launchpad.net/gocheck"
	"net/http"
	"net/http/httptest"
	"io/ioutil"
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&HTTPSuite{})

type HTTPSuite struct {
	server *httptest.Server
}

type HelloHandler struct {}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write([]byte("Hello World\n"))
}

func (s *HTTPSuite) SetUpSuite(c *C) {
	h := HelloHandler{}
	fmt.Printf("Starting New Server\n")
	s.server = httptest.NewServer(&h)
}

func (s *HTTPSuite) TearDownSuite(c *C) {
	if s.server != nil {
		fmt.Printf("Stopping Server\n")
		s.server.Close()
	}
}

func (s *HTTPSuite) TestHelloWorld(c *C) {
	fmt.Printf("Running HelloWorld\n")
	response, err := http.Get(s.server.URL)
	c.Check(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(err, IsNil)
	c.Check(string(content), Equals, "Hello World\n")
}
