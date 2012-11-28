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
	if token != "" {
		req.Header.Add("X-Auth-Token", token)
	}
	client := &http.Client{}
	return client.Do(req)
}

func (s *SwiftHTTPSuite) ensureNotContainer(name string, c *C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, false)
}

func (s *SwiftHTTPSuite) ensureContainer(name string, c *C) {
	s.ensureNotContainer(name, c)
	err := s.service.AddContainer("test")
	c.Assert(err, IsNil)
}

func (s *SwiftHTTPSuite) removeContainer(name string, c *C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, Equals, true)
	err := s.service.RemoveContainer("test")
	c.Assert(err, IsNil)
}

func (s *SwiftHTTPSuite) ensureNotObject(container, object string, c *C) {
	_, err := s.service.GetObject(container, object)
	c.Assert(err, Not(IsNil))
}

func (s *SwiftHTTPSuite) ensureObject(container, object string, data []byte, c *C) {
	s.ensureNotObject(container, object, c)
	err := s.service.AddObject(container, object, data)
	c.Assert(err, IsNil)
}

func (s *SwiftHTTPSuite) ensureObjectData(container, object string, data []byte, c *C) {
	objdata, err := s.service.GetObject(container, object)
	c.Assert(err, IsNil)
	c.Assert(objdata, DeepEquals, data)
}

func (s *SwiftHTTPSuite) removeObject(container, object string, c *C) {
	err := s.service.RemoveObject(container, object)
	c.Assert(err, IsNil)
	s.ensureNotObject(container, object, c)
}

func (s *SwiftHTTPSuite) TestPUTContainerMissingCreated(c *C) {
	s.ensureNotContainer("test", c)

	resp, err := s.sendRequest("PUT", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTContainerExistsAccepted(c *C) {
	s.ensureContainer("test", c)

	resp, err := s.sendRequest("PUT", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusAccepted)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	resp, err := s.sendRequest("GET", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETContainerExistsOK(c *C) {
	s.ensureContainer("test", c)

	resp, err := s.sendRequest("GET", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	resp, err := s.sendRequest("DELETE", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
}

func (s *SwiftHTTPSuite) TestDELETEContainerExistsNoContent(c *C) {
	s.ensureContainer("test", c)

	resp, err := s.sendRequest("DELETE", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectMissingCreated(c *C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	data := []byte("test data")
	resp, err := s.sendRequest("PUT", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	s.ensureObjectData("test", "obj", data, c)
	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectExistsCreated(c *C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	newdata := []byte("new test data")
	resp, err := s.sendRequest("PUT", "test/obj", newdata)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	s.ensureObjectData("test", "obj", newdata, c)
	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	data := []byte("test data")
	resp, err := s.sendRequest("PUT", "test/obj", data)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectMissingNotFound(c *C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	resp, err := s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	resp, err := s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectExistsOK(c *C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	resp, err := s.sendRequest("GET", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	s.ensureObjectData("test", "obj", data, c)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)
	c.Assert(body, DeepEquals, data)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectMissingNotFound(c *C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	resp, err := s.sendRequest("DELETE", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	resp, err := s.sendRequest("DELETE", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectExistsNoContent(c *C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	resp, err := s.sendRequest("DELETE", "test/obj", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestUnauthorizedFails(c *C) {
	oldtoken := token
	defer func() {
		token = oldtoken
	}()
	token = ""
	resp, err := s.sendRequest("GET", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusUnauthorized)

	token = "invalid"
	resp, err = s.sendRequest("PUT", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusUnauthorized)

	resp, err = s.sendRequest("DELETE", "test", nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusUnauthorized)
}
