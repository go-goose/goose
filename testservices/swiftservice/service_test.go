// Swift double testing service - internal direct API tests

package swiftservice

import (
	. "launchpad.net/gocheck"
)

type SwiftServiceSuite struct {
	service SwiftService
}

var baseURL = "/v1/AUTH_tenant/"
var token = "token"
var hostname = "localhost" // not really used here

var _ = Suite(&SwiftServiceSuite{})

func (s *SwiftServiceSuite) SetUpSuite(c *C) {
	s.service = New(hostname, baseURL, token)
}

func (s *SwiftServiceSuite) TestAddHasRemoveContainer(c *C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
	err := s.service.AddContainer("test")
	c.Assert(err, IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, true)
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
}

func (s *SwiftServiceSuite) TestAddGetRemoveObject(c *C) {
	_, err := s.service.GetObject("test", "obj")
	c.Assert(err, Not(IsNil))
	err = s.service.AddContainer("test")
	c.Assert(err, IsNil)
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, true)
	data := []byte("test data")
	err = s.service.AddObject("test", "obj", data)
	c.Assert(err, IsNil)
	objdata, err := s.service.GetObject("test", "obj")
	c.Assert(err, IsNil)
	c.Assert(objdata, DeepEquals, data)
	err = s.service.RemoveObject("test", "obj")
	c.Assert(err, IsNil)
	_, err = s.service.GetObject("test", "obj")
	c.Assert(err, Not(IsNil))
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
}

func (s *SwiftServiceSuite) TestRemoveContainerWithObjects(c *C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
	err := s.service.AddObject("test", "obj", []byte("test data"))
	c.Assert(err, IsNil)
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
	_, err = s.service.GetObject("test", "obj")
	c.Assert(err, Not(IsNil))
}

func (s *SwiftServiceSuite) TestGetURL(c *C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
	err := s.service.AddContainer("test")
	c.Assert(err, IsNil)
	data := []byte("test data")
	err = s.service.AddObject("test", "obj", data)
	c.Assert(err, IsNil)
	url, err := s.service.GetURL("test", "obj")
	path := baseURL + "test/obj"
	c.Assert(err, IsNil)
	c.Assert(url, Equals, hostname+path)
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
}

func (s *SwiftServiceSuite) TestListContainer(c *C) {
	err := s.service.AddContainer("test")
	c.Assert(err, IsNil)
	data := []byte("test data")
	err = s.service.AddObject("test", "obj", data)
	c.Assert(err, IsNil)
	containerData, err := s.service.ListContainer("test")
	c.Assert(err, IsNil)
	c.Assert(len(containerData), Equals, 1)
	c.Assert(containerData[0].Name, Equals, "obj")
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
}
