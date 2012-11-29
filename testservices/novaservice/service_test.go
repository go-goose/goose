// Nova double testing service - internal direct API tests

package novaservice

import (
	. "launchpad.net/gocheck"
)

type NovaServiceSuite struct {
	service NovaService
}

var baseURL = "/v2/"
var token = "token"
var hostname = "localhost" // not really used here

var _ = Suite(&NovaServiceSuite{})

func (s *NovaServiceSuite) SetUpSuite(c *C) {
	s.service = New(hostname, baseURL, token)
}
