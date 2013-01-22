// Swift double testing service - HTTP API tests

package swiftservice

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/swift"
	"launchpad.net/goose/testing/httpsuite"
	"net/http"
)

type SwiftHTTPSuite struct {
	httpsuite.HTTPSuite
	service *Swift
}

var _ = Suite(&SwiftHTTPSuite{})

func (s *SwiftHTTPSuite) SetUpSuite(c *C) {
	s.HTTPSuite.SetUpSuite(c)
	s.service = New(s.Server.URL, token, region)
}

func (s *SwiftHTTPSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
}

func (s *SwiftHTTPSuite) TearDownTest(c *C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *SwiftHTTPSuite) TearDownSuite(c *C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *SwiftHTTPSuite) sendRequest(c *C, method, path string, body []byte,
	expectedStatusCode int) (resp *http.Response) {
	var req *http.Request
	var err error
	url := s.Server.URL + baseURL + "/" + path
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	c.Assert(err, IsNil)
	if token != "" {
		req.Header.Add("X-Auth-Token", token)
	}
	client := &http.Client{}
	resp, err = client.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, expectedStatusCode)
	return resp
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

	s.sendRequest(c, "PUT", "test", nil, http.StatusCreated)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTContainerExistsAccepted(c *C) {
	s.ensureContainer("test", c)

	s.sendRequest(c, "PUT", "test", nil, http.StatusAccepted)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "GET", "test", nil, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETContainerExistsOK(c *C) {
	s.ensureContainer("test", c)
	data := []byte("test data")
	s.ensureObject("test", "obj", data, c)

	resp := s.sendRequest(c, "GET", "test", nil, http.StatusOK)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)
	var containerData []swift.ContainerContents
	err = json.Unmarshal(body, &containerData)
	c.Assert(err, IsNil)
	c.Assert(len(containerData), Equals, 1)
	c.Assert(containerData[0].Name, Equals, "obj")

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "DELETE", "test", nil, http.StatusNotFound)
}

func (s *SwiftHTTPSuite) TestDELETEContainerExistsNoContent(c *C) {
	s.ensureContainer("test", c)

	s.sendRequest(c, "DELETE", "test", nil, http.StatusNoContent)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectMissingCreated(c *C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	data := []byte("test data")
	s.sendRequest(c, "PUT", "test/obj", data, http.StatusCreated)

	s.ensureObjectData("test", "obj", data, c)
	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectExistsCreated(c *C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	newdata := []byte("new test data")
	s.sendRequest(c, "PUT", "test/obj", newdata, http.StatusCreated)

	s.ensureObjectData("test", "obj", newdata, c)
	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	data := []byte("test data")
	s.sendRequest(c, "PUT", "test/obj", data, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectMissingNotFound(c *C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	s.sendRequest(c, "GET", "test/obj", nil, http.StatusNotFound)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "GET", "test/obj", nil, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectExistsOK(c *C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	resp := s.sendRequest(c, "GET", "test/obj", nil, http.StatusOK)

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

	s.sendRequest(c, "DELETE", "test/obj", nil, http.StatusNotFound)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectContainerMissingNotFound(c *C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "DELETE", "test/obj", nil, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectExistsNoContent(c *C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	s.sendRequest(c, "DELETE", "test/obj", nil, http.StatusNoContent)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestUnauthorizedFails(c *C) {
	oldtoken := token
	defer func() {
		token = oldtoken
	}()
	// TODO(wallyworld) - until ACLs are supported, empty tokens are assumed to be used when
	// we need to access a public container.
	// token = ""
	// s.sendRequest(c, "GET", "test", nil, http.StatusUnauthorized)

	token = "invalid"
	s.sendRequest(c, "PUT", "test", nil, http.StatusUnauthorized)

	s.sendRequest(c, "DELETE", "test", nil, http.StatusUnauthorized)
}
