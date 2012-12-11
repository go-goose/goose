// Nova double testing service - HTTP API tests

package novaservice

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
)

type NovaHTTPSuite struct {
	httpsuite.HTTPSuite
	service *Nova
}

var _ = Suite(&NovaHTTPSuite{})

func (s *NovaHTTPSuite) SetUpSuite(c *C) {
	s.HTTPSuite.SetUpSuite(c)
	s.service = New(s.Server.URL, baseURL, token)
}

func (s *NovaHTTPSuite) TearDownSuite(c *C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *NovaHTTPSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.Mux.Handle(baseURL, s.service)
}

func (s *NovaHTTPSuite) TearDownTest(c *C) {
	s.HTTPSuite.TearDownTest(c)
}
