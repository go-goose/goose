package swiftservice

import (
	"bytes"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"net/http"
)

type SwiftServiceSuite struct {
	httpsuite.HTTPSuite
	service SwiftService
}

var baseURL = "/v1/AUTH_tenant/"
var token = "token"

var _ = Suite(&SwiftServiceSuite{})

func (s *SwiftServiceSuite) SetUpSuite(c *C) {
	s.HTTPSuite.SetUpSuite(c)
	s.service = New(s.Server.URL, baseURL, token)
}

func (s *SwiftServiceSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.Mux.Handle(baseURL, s.service)
}

func (s *SwiftServiceSuite) TearDownTest(c *C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *SwiftServiceSuite) TearDownSuite(c *C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *SwiftServiceSuite) sendRequest(method, path string, body []byte) (*http.Response, error) {
	var req *http.Request
	var err error
	url := s.Server.URL + baseURL + path
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Auth-Token", token)
	return http.DefaultClient.Do(req)
}

func (s *SwiftServiceSuite) TestAddHasRemoveContainer(c *C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
	err := s.service.AddContainer("test")
	c.Assert(err, IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, true)
	resp, err := s.sendRequest("GET", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotImplemented)
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
	resp, err = s.sendRequest("PUT", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, true)
	resp, err = s.sendRequest("PUT", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusAccepted)
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
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
	resp, err := s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)
	c.Assert(body, DeepEquals, data)
	err = s.service.RemoveObject("test", "obj")
	c.Assert(err, IsNil)
	_, err = s.service.GetObject("test", "obj")
	c.Assert(err, Not(IsNil))
	resp, err = s.sendRequest("PUT", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	objdata, err = s.service.GetObject("test", "obj")
	c.Assert(err, IsNil)
	c.Assert(objdata, DeepEquals, data)
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
	c.Assert(url, Equals, s.Server.URL+path)
	req, err := http.NewRequest("GET", url, nil)
	c.Assert(err, IsNil)
	req.Header.Add("X-Auth-Token", token)
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)
	c.Assert(body, DeepEquals, data)
	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
}
