package identitystub

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&HTTPSuite{})

type HTTPSuite struct {
	server     *httptest.Server
	mux        *http.ServeMux
	oldHandler http.Handler
}

type HelloHandler struct{}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write([]byte("Hello World\n"))
}

func (s *HTTPSuite) SetUpSuite(c *C) {
	// fmt.Printf("Starting New Server\n")
	s.server = httptest.NewServer(nil)
}

func (s *HTTPSuite) SetUpTest(c *C) {
	s.oldHandler = s.server.Config.Handler
	s.mux = http.NewServeMux()
	s.server.Config.Handler = s.mux
}

func (s *HTTPSuite) TearDownTest(c *C) {
	s.mux = nil
	s.server.Config.Handler = s.oldHandler
}

func (s *HTTPSuite) TearDownSuite(c *C) {
	if s.server != nil {
		// fmt.Printf("Stopping Server\n")
		s.server.Close()
	}
}

func (s *HTTPSuite) TestHelloWorld(c *C) {
	s.mux.Handle("/", &HelloHandler{})
	// fmt.Printf("Running HelloWorld\n")
	response, err := http.Get(s.server.URL)
	c.Check(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(err, IsNil)
	c.Check(string(content), Equals, "Hello World\n")
}
