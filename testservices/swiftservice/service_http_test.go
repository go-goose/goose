// Swift double testing service - HTTP API tests

package swiftservice

import (
	"bytes"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"net/http"
)

type SwiftHTTPSuite struct {
	httpsuite.HTTPSuite
	service SwiftService
}

var _ = Suite(&SwiftHTTPSuite{})

func (s *SwiftHTTPSuite) SetUpSuite(c *C) {
	s.HTTPSuite.SetUpSuite(c)
	s.service = New(s.Server.URL, baseURL, token)
}

func (s *SwiftHTTPSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.Mux.Handle(baseURL, s.service)
}

func (s *SwiftHTTPSuite) TearDownTest(c *C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *SwiftHTTPSuite) TearDownSuite(c *C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *SwiftHTTPSuite) sendRequest(method, path string, body []byte) (*http.Response, error) {
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
	client := &http.Client{}
	return client.Do(req)
}

func (s *SwiftHTTPSuite) TestContainerHandling(c *C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, false)

	resp, err := s.sendRequest("PUT", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, true)

	resp, err = s.sendRequest("PUT", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusAccepted)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, true)

	resp, err = s.sendRequest("GET", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, true)

	resp, err = s.sendRequest("DELETE", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)

	resp, err = s.sendRequest("DELETE", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)

	resp, err = s.sendRequest("GET", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
}

func (s *SwiftHTTPSuite) TestObjectsHandling(c *C) {
	_, err := s.service.GetObject("test", "obj")
	c.Assert(err, Not(IsNil))

	resp, err := s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	data := []byte("test data")
	err = s.service.AddObject("test", "obj", data)
	c.Assert(err, IsNil)

	resp, err = s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)
	c.Assert(body, DeepEquals, data)

	resp, err = s.sendRequest("DELETE", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)

	_, err = s.service.GetObject("test", "obj")
	c.Assert(err, Not(IsNil))

	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, true)

	resp, err = s.sendRequest("PUT", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, false)

	resp, err = s.sendRequest("PUT", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	resp, err = s.sendRequest("GET", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	resp, err = s.sendRequest("DELETE", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	err = s.service.AddContainer("test")
	c.Assert(err, IsNil)

	ok = s.service.HasContainer("test")
	c.Assert(ok, Equals, true)

	_, err = s.service.GetObject("test", "obj")
	c.Assert(err, Not(IsNil))

	resp, err = s.sendRequest("GET", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	resp, err = s.sendRequest("DELETE", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	resp, err = s.sendRequest("PUT", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	objdata, err := s.service.GetObject("test", "obj")
	c.Assert(err, IsNil)
	c.Assert(objdata, DeepEquals, data)

	resp, err = s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	c.Assert(body, DeepEquals, data)

	newdata := []byte("new test data")
	resp, err = s.sendRequest("PUT", "test/obj", newdata)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	objdata, err = s.service.GetObject("test", "obj")
	c.Assert(err, IsNil)
	c.Assert(objdata, DeepEquals, newdata)

	resp, err = s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)
	c.Assert(body, DeepEquals, newdata)

	err = s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
}
