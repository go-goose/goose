// Swift double testing service - HTTP API tests

package swiftservice

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/swift"
	"gopkg.in/goose.v1/testing/httpsuite"
	"gopkg.in/goose.v1/testservices/identityservice"
)

type SwiftHTTPSuite struct {
	httpsuite.HTTPSuite
	service *Swift
	token   string
}

var _ = gc.Suite(&SwiftHTTPSuite{})

type SwiftHTTPSSuite struct {
	httpsuite.HTTPSuite
	service *Swift
	token   string
}

var _ = gc.Suite(&SwiftHTTPSSuite{HTTPSuite: httpsuite.HTTPSuite{UseTLS: true}})

func (s *SwiftHTTPSuite) SetUpSuite(c *gc.C) {
	s.HTTPSuite.SetUpSuite(c)
	identityDouble := identityservice.NewUserPass()
	s.service = New(s.Server.URL, versionPath, tenantId, region, identityDouble)
	userInfo := identityDouble.AddUser("fred", "secret", "tenant")
	s.token = userInfo.Token
}

func (s *SwiftHTTPSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
}

func (s *SwiftHTTPSuite) TearDownTest(c *gc.C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *SwiftHTTPSuite) TearDownSuite(c *gc.C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *SwiftHTTPSuite) sendRequest(c *gc.C, method, path string, body []byte,
	expectedStatusCode int) (resp *http.Response) {
	return s.sendRequestWithParams(c, method, path, nil, body, expectedStatusCode)
}

func (s *SwiftHTTPSuite) sendRequestWithParams(c *gc.C, method, path string, params map[string]string, body []byte,
	expectedStatusCode int) (resp *http.Response) {
	var req *http.Request
	var err error
	URL := s.service.endpointURL(path)
	if len(params) > 0 {
		urlParams := make(url.Values, len(params))
		for k, v := range params {
			urlParams.Set(k, v)
		}
		URL += "?" + urlParams.Encode()
	}
	if body != nil {
		req, err = http.NewRequest(method, URL, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, URL, nil)
	}
	c.Assert(err, gc.IsNil)
	if s.token != "" {
		req.Header.Add("X-Auth-Token", s.token)
	}
	client := &http.DefaultClient
	resp, err = client.Do(req)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, expectedStatusCode)
	return resp
}

func (s *SwiftHTTPSuite) ensureNotContainer(name string, c *gc.C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, false)
}

func (s *SwiftHTTPSuite) ensureContainer(name string, c *gc.C) {
	s.ensureNotContainer(name, c)
	err := s.service.AddContainer("test")
	c.Assert(err, gc.IsNil)
}

func (s *SwiftHTTPSuite) removeContainer(name string, c *gc.C) {
	ok := s.service.HasContainer("test")
	c.Assert(ok, gc.Equals, true)
	err := s.service.RemoveContainer("test")
	c.Assert(err, gc.IsNil)
}

func (s *SwiftHTTPSuite) ensureNotObject(container, object string, c *gc.C) {
	_, err := s.service.GetObject(container, object)
	c.Assert(err, gc.Not(gc.IsNil))
}

func (s *SwiftHTTPSuite) ensureObject(container, object string, data []byte, c *gc.C) {
	s.ensureNotObject(container, object, c)
	err := s.service.AddObject(container, object, data)
	c.Assert(err, gc.IsNil)
}

func (s *SwiftHTTPSuite) ensureObjectData(container, object string, data []byte, c *gc.C) {
	objdata, err := s.service.GetObject(container, object)
	c.Assert(err, gc.IsNil)
	c.Assert(objdata, gc.DeepEquals, data)
}

func (s *SwiftHTTPSuite) removeObject(container, object string, c *gc.C) {
	err := s.service.RemoveObject(container, object)
	c.Assert(err, gc.IsNil)
	s.ensureNotObject(container, object, c)
}

func (s *SwiftHTTPSuite) TestPUTContainerMissingCreated(c *gc.C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "PUT", "test", nil, http.StatusCreated)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTContainerExistsAccepted(c *gc.C) {
	s.ensureContainer("test", c)

	s.sendRequest(c, "PUT", "test", nil, http.StatusAccepted)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETContainerMissingNotFound(c *gc.C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "GET", "test", nil, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETContainerExistsOK(c *gc.C) {
	s.ensureContainer("test", c)
	data := []byte("test data")
	s.ensureObject("test", "obj", data, c)

	resp := s.sendRequest(c, "GET", "test", nil, http.StatusOK)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, gc.IsNil)
	var containerData []swift.ContainerContents
	err = json.Unmarshal(body, &containerData)
	c.Assert(err, gc.IsNil)
	c.Assert(len(containerData), gc.Equals, 1)
	c.Assert(containerData[0].Name, gc.Equals, "obj")

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETContainerWithPrefix(c *gc.C) {
	s.ensureContainer("test", c)
	data := []byte("test data")
	s.ensureObject("test", "foo", data, c)
	s.ensureObject("test", "foobar", data, c)

	resp := s.sendRequestWithParams(c, "GET", "test", map[string]string{"prefix": "foob"}, nil, http.StatusOK)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, gc.IsNil)
	var containerData []swift.ContainerContents
	err = json.Unmarshal(body, &containerData)
	c.Assert(err, gc.IsNil)
	c.Assert(len(containerData), gc.Equals, 1)
	c.Assert(containerData[0].Name, gc.Equals, "foobar")

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEContainerMissingNotFound(c *gc.C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "DELETE", "test", nil, http.StatusNotFound)
}

func (s *SwiftHTTPSuite) TestDELETEContainerExistsNoContent(c *gc.C) {
	s.ensureContainer("test", c)

	s.sendRequest(c, "DELETE", "test", nil, http.StatusNoContent)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectMissingCreated(c *gc.C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	data := []byte("test data")
	s.sendRequest(c, "PUT", "test/obj", data, http.StatusCreated)

	s.ensureObjectData("test", "obj", data, c)
	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectExistsCreated(c *gc.C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	newdata := []byte("new test data")
	s.sendRequest(c, "PUT", "test/obj", newdata, http.StatusCreated)

	s.ensureObjectData("test", "obj", newdata, c)
	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestPUTObjectContainerMissingNotFound(c *gc.C) {
	s.ensureNotContainer("test", c)

	data := []byte("test data")
	s.sendRequest(c, "PUT", "test/obj", data, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectMissingNotFound(c *gc.C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	s.sendRequest(c, "GET", "test/obj", nil, http.StatusNotFound)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectContainerMissingNotFound(c *gc.C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "GET", "test/obj", nil, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestGETObjectExistsOK(c *gc.C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	resp := s.sendRequest(c, "GET", "test/obj", nil, http.StatusOK)

	s.ensureObjectData("test", "obj", data, c)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, gc.IsNil)
	c.Assert(body, gc.DeepEquals, data)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectMissingNotFound(c *gc.C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	s.sendRequest(c, "DELETE", "test/obj", nil, http.StatusNotFound)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectContainerMissingNotFound(c *gc.C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "DELETE", "test/obj", nil, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestDELETEObjectExistsNoContent(c *gc.C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	s.sendRequest(c, "DELETE", "test/obj", nil, http.StatusNoContent)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestHEADContainerExistsOK(c *gc.C) {
	s.ensureContainer("test", c)
	data := []byte("test data")
	s.ensureObject("test", "obj", data, c)

	resp := s.sendRequest(c, "HEAD", "test", nil, http.StatusOK)
	c.Assert(resp.Header.Get("Date"), gc.Not(gc.Equals), "")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, gc.IsNil)
	c.Assert(body, gc.DeepEquals, []byte{})
	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestHEADContainerMissingNotFound(c *gc.C) {
	s.ensureNotContainer("test", c)

	s.sendRequest(c, "HEAD", "test", nil, http.StatusNotFound)

	s.ensureNotContainer("test", c)
}

func (s *SwiftHTTPSuite) TestHEADObjectExistsOK(c *gc.C) {
	data := []byte("test data")
	s.ensureContainer("test", c)
	s.ensureObject("test", "obj", data, c)

	resp := s.sendRequest(c, "HEAD", "test/obj", nil, http.StatusOK)

	s.ensureObjectData("test", "obj", data, c)
	c.Assert(resp.Header.Get("Date"), gc.Not(gc.Equals), "")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, gc.IsNil)
	c.Assert(body, gc.DeepEquals, []byte{})

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestHEADObjectMissingNotFound(c *gc.C) {
	s.ensureContainer("test", c)
	s.ensureNotObject("test", "obj", c)

	s.sendRequest(c, "HEAD", "test/obj", nil, http.StatusNotFound)

	s.removeContainer("test", c)
}

func (s *SwiftHTTPSuite) TestUnauthorizedFails(c *gc.C) {
	oldtoken := s.token
	defer func() {
		s.token = oldtoken
	}()
	// TODO(wallyworld) - 2013-02-11 bug=1121682
	// until ACLs are supported, empty tokens are assumed to be used when we need to access a public container.
	// token = ""
	// s.sendRequest(c, "GET", "test", nil, http.StatusUnauthorized)

	s.token = "invalid"
	s.sendRequest(c, "PUT", "test", nil, http.StatusUnauthorized)

	s.sendRequest(c, "DELETE", "test", nil, http.StatusUnauthorized)
}

func (s *SwiftHTTPSSuite) SetUpSuite(c *gc.C) {
	s.HTTPSuite.SetUpSuite(c)
	identityDouble := identityservice.NewUserPass()
	userInfo := identityDouble.AddUser("fred", "secret", "tenant")
	s.token = userInfo.Token
	c.Assert(s.Server.URL[:8], gc.Equals, "https://")
	s.service = New(s.Server.URL, versionPath, userInfo.TenantId, region, identityDouble)
}

func (s *SwiftHTTPSSuite) TearDownSuite(c *gc.C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *SwiftHTTPSSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
}

func (s *SwiftHTTPSSuite) TearDownTest(c *gc.C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *SwiftHTTPSSuite) TestHasHTTPSServiceURL(c *gc.C) {
	endpoints := s.service.Endpoints()
	c.Assert(endpoints[0].PublicURL[:8], gc.Equals, "https://")
}
