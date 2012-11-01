package identityservice

import (
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
