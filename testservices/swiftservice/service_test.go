// Swift double testing service - internal direct API tests

package swiftservice

import (
	"fmt"

	gc "gopkg.in/check.v1"
)

type SwiftServiceSuite struct {
	service *Swift
}

var region = "region"             // not really used here
var hostname = "http://localhost" // not really used here
var versionPath = "v2"            // not really used here
var tenantId = "tenant"           // not really used here

var _ = gc.Suite(&SwiftServiceSuite{})

func (s *SwiftServiceSuite) SetUpSuite(c *gc.C) {
	s.service = New(hostname, versionPath, tenantId, region, nil)
}

func (s *SwiftServiceSuite) TestAddHasRemoveContainer(c *gc.C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, false)
	err := s.service.AddContainer("test")
	c.Assert(err, gc.IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, true)
	err = s.service.RemoveContainer("test")
	c.Assert(err, gc.IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, false)
}

func (s *SwiftServiceSuite) TestAddGetRemoveObject(c *gc.C) {
	_, err := s.service.GetObject("test", "obj")
	c.Assert(err, gc.Not(gc.IsNil))
	err = s.service.AddContainer("test")
	c.Assert(err, gc.IsNil)
	ok := s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, true)
	data := []byte("test data")
	err = s.service.AddObject("test", "obj", data)
	c.Assert(err, gc.IsNil)
	objdata, err := s.service.GetObject("test", "obj")
	c.Assert(err, gc.IsNil)
	c.Assert(objdata, gc.DeepEquals, data)
	err = s.service.RemoveObject("test", "obj")
	c.Assert(err, gc.IsNil)
	_, err = s.service.GetObject("test", "obj")
	c.Assert(err, gc.Not(gc.IsNil))
	err = s.service.RemoveContainer("test")
	c.Assert(err, gc.IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, false)
}

func (s *SwiftServiceSuite) TestRemoveContainerWithObjects(c *gc.C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, false)
	err := s.service.AddObject("test", "obj", []byte("test data"))
	c.Assert(err, gc.IsNil)
	err = s.service.RemoveContainer("test")
	c.Assert(err, gc.IsNil)
	_, err = s.service.GetObject("test", "obj")
	c.Assert(err, gc.Not(gc.IsNil))
}

func (s *SwiftServiceSuite) TestGetURL(c *gc.C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, false)
	err := s.service.AddContainer("test")
	c.Assert(err, gc.IsNil)
	data := []byte("test data")
	err = s.service.AddObject("test", "obj", data)
	c.Assert(err, gc.IsNil)
	url, err := s.service.GetURL("test", "obj")
	c.Assert(err, gc.IsNil)
	c.Assert(url, gc.Equals, fmt.Sprintf("%s/%s/%s/test/obj", hostname, versionPath, tenantId))
	err = s.service.RemoveContainer("test")
	c.Assert(err, gc.IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, false)
}

func (s *SwiftServiceSuite) TestListContainer(c *gc.C) {
	err := s.service.AddContainer("test")
	c.Assert(err, gc.IsNil)
	data := []byte("test data")
	err = s.service.AddObject("test", "obj", data)
	c.Assert(err, gc.IsNil)
	containerData, err := s.service.ListContainer("test", nil)
	c.Assert(err, gc.IsNil)
	c.Assert(len(containerData), gc.Equals, 1)
	c.Assert(containerData[0].Name, gc.Equals, "obj")
	err = s.service.RemoveContainer("test")
	c.Assert(err, gc.IsNil)
}

func (s *SwiftServiceSuite) TestListContainerWithPrefix(c *gc.C) {
	err := s.service.AddContainer("test")
	c.Assert(err, gc.IsNil)
	data := []byte("test data")
	err = s.service.AddObject("test", "foo", data)
	c.Assert(err, gc.IsNil)
	err = s.service.AddObject("test", "foobar", data)
	c.Assert(err, gc.IsNil)
	containerData, err := s.service.ListContainer("test", map[string]string{"prefix": "foob"})
	c.Assert(err, gc.IsNil)
	c.Assert(len(containerData), gc.Equals, 1)
	c.Assert(containerData[0].Name, gc.Equals, "foobar")
	err = s.service.RemoveContainer("test")
	c.Assert(err, gc.IsNil)
}
