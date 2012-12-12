// Nova double testing service - HTTP API tests

package novaservice

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/nova"
	"launchpad.net/goose/testing/httpsuite"
	"net/http"
	"strconv"
	"strings"
)

type NovaHTTPSuite struct {
	httpsuite.HTTPSuite
	service *Nova
}

var _ = Suite(&NovaHTTPSuite{})

func (s *NovaHTTPSuite) SetUpSuite(c *C) {
	s.HTTPSuite.SetUpSuite(c)
	s.service = New(s.Server.URL, baseURL, token, tenantId)
}

func (s *NovaHTTPSuite) TearDownSuite(c *C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *NovaHTTPSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.setupHTTP(s.Mux)
}

func (s *NovaHTTPSuite) TearDownTest(c *C) {
	s.HTTPSuite.TearDownTest(c)
}

// assertJSON asserts the passed http.Response's body can be
// unmarshalled into the passed expected object.
func (s *NovaHTTPSuite) assertJSON(c *C, resp *http.Response, expected interface{}) {
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	err = json.Unmarshal(body, &expected)
	c.Assert(err, IsNil)
}

// assertBody asserts the passed http.Response's body matches the
// expected response, replacing any variables in the expected body.
func (s *NovaHTTPSuite) assertBody(c *C, resp *http.Response, expected response) {
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	expBody := expected.replaceVars(resp.Request)
	// cast to string for easier asserts debugging
	c.Assert(string(body), Equals, string(expBody))
}

// sendRequest constructs an HTTP request from the parameters and
// sends it, returning the response or an error.
func (s *NovaHTTPSuite) sendRequest(method, url string, body []byte, headers http.Header) (*http.Response, error) {
	if !strings.HasPrefix(url, s.service.hostname) {
		url = s.service.hostname + strings.TrimLeft(url, "/")
	}
	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if headers != nil {
		for header, values := range headers {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}
	}
	// workaround for https://code.google.com/p/go/issues/detail?id=4454
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return http.DefaultClient.Do(req)
}

// authRequest is a shortcut for sending requests with pre-set token
// header and correct version prefix and tenant ID in the URL.
func (s *NovaHTTPSuite) authRequest(method, path string, body []byte, headers http.Header) (*http.Response, error) {
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set(authToken, s.service.token)
	url := s.service.endpoint(true, path)
	return s.sendRequest(method, url, body, headers)
}

// jsonRequest serializes the passed body object to JSON and sends a
// the request with authRequest().
func (s *NovaHTTPSuite) jsonRequest(method, path string, body interface{}, headers http.Header) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return s.authRequest(method, path, jsonBody, headers)
}

func (s *NovaHTTPSuite) TestUnauthorizedResponse(c *C) {
	resp, err := s.sendRequest("GET", "/any", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusUnauthorized)
	headers := make(http.Header)
	headers.Set(authToken, "phony")
	resp, err = s.sendRequest("POST", "/any", nil, headers)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusUnauthorized)
	s.assertBody(c, resp, unauthorizedResponse)
}

func (s *NovaHTTPSuite) TestNoVersionResponse(c *C) {
	headers := make(http.Header)
	headers.Set(authToken, s.service.token)
	resp, err := s.sendRequest("GET", "/", nil, headers)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	s.assertBody(c, resp, noVersionResponse)
}

func (s *NovaHTTPSuite) TestMultipleChoicesResponse(c *C) {
	headers := make(http.Header)
	headers.Set(authToken, s.service.token)
	resp, err := s.sendRequest("GET", "/any", nil, headers)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusMultipleChoices)
	s.assertBody(c, resp, multipleChoicesResponse)
	resp, err = s.sendRequest("POST", "/any/other/one", nil, headers)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusMultipleChoices)
	s.assertBody(c, resp, multipleChoicesResponse)
}

func (s *NovaHTTPSuite) TestNotFoundResponse(c *C) {
	resp, err := s.authRequest("GET", "/flavors/", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	resp, err = s.authRequest("POST", "/any/unknown/one", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	resp, err = s.authRequest("GET", "/flavors/id", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
}

func (s *NovaHTTPSuite) TestBadRequestResponse(c *C) {
	headers := make(http.Header)
	headers.Set(authToken, token)
	resp, err := s.sendRequest("GET", s.service.baseURL+"/phony_token", nil, headers)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusBadRequest)
	s.assertBody(c, resp, badRequestResponse)
}

func (s *NovaHTTPSuite) TestGetFlavors(c *C) {
	entities := s.service.allFlavorsAsEntities()
	c.Assert(entities, HasLen, 0)
	resp, err := s.authRequest("GET", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)
	flavors := []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1", Name: "flavor 1"},
		nova.FlavorDetail{Id: "fl2", Name: "flavor 2"},
	}
	for _, flavor := range flavors {
		s.service.buildFlavorLinks(&flavor)
		err = s.service.addFlavor(flavor)
		defer s.service.removeFlavor(flavor.Id)
		c.Assert(err, IsNil)
	}
	entities = s.service.allFlavorsAsEntities()
	resp, err = s.authRequest("GET", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	var expected struct {
		Flavors []nova.Entity
	}
	s.assertJSON(c, resp, expected)
}

func (s *NovaHTTPSuite) TestGetInvalidFlavorsFails(c *C) {
	flavor := nova.FlavorDetail{Id: "1"}
	err := s.service.addFlavor(flavor)
	c.Assert(err, IsNil)
	defer s.service.removeFlavor("1")
	resp, err := s.authRequest("GET", "/flavors/invalid", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	s.assertBody(c, resp, notFoundResponse)
}

func (s *NovaHTTPSuite) TestPostInvalidFlavorsFails(c *C) {
	resp, err := s.authRequest("POST", "/flavors/invalid", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	s.assertBody(c, resp, notFoundResponse)
}

func (s *NovaHTTPSuite) TestPostEmptyFlavorsFails(c *C) {
	resp, err := s.authRequest("POST", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusBadRequest)
	s.assertBody(c, resp, badRequest2Response)
}

func (s *NovaHTTPSuite) TestPostValidFlavorSucceeds(c *C) {
	_, err := s.service.flavor("fl1")
	c.Assert(err, NotNil)
	var req struct {
		Flavor nova.FlavorDetail `json:"flavor"`
	}
	req.Flavor = nova.FlavorDetail{Id: "fl1", Name: "flavor 1"}
	resp, err := s.jsonRequest("POST", "/flavors", req, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)
	s.assertBody(c, resp, createdResponse)
	_, err = s.service.flavor("fl1")
	c.Assert(err, IsNil)
	s.service.removeFlavor("fl1")
}

func (s *NovaHTTPSuite) TestPutFlavorsFails(c *C) {
	resp, err := s.authRequest("PUT", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	s.assertBody(c, resp, notFoundResponse)
}

func (s *NovaHTTPSuite) TestDeleteFlavorsFails(c *C) {
	resp, err := s.authRequest("DELETE", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	s.assertBody(c, resp, notFoundResponse)
}

func (s *NovaHTTPSuite) TestGetFlavorsDetail(c *C) {
	flavors := s.service.allFlavors()
	c.Assert(flavors, HasLen, 0)
	resp, err := s.authRequest("GET", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)
	flavors = []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1", Name: "flavor 1"},
		nova.FlavorDetail{Id: "fl2", Name: "flavor 2"},
	}
	for _, flavor := range flavors {
		s.service.buildFlavorLinks(&flavor)
		err = s.service.addFlavor(flavor)
		defer s.service.removeFlavor(flavor.Id)
		c.Assert(err, IsNil)
	}
	resp, err = s.authRequest("GET", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	var expected struct {
		Flavors []nova.FlavorDetail
	}
	s.assertJSON(c, resp, expected)
}

func (s *NovaHTTPSuite) TestPostFlavorsDetailFails(c *C) {
	resp, err := s.authRequest("POST", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	s.assertBody(c, resp, notFoundResponse)
}

func (s *NovaHTTPSuite) TestPutFlavorsDetailFails(c *C) {
	resp, err := s.authRequest("PUT", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
	s.assertBody(c, resp, notFoundJSONResponse)
}

func (s *NovaHTTPSuite) TestDeleteFlavorsDetailFails(c *C) {
	resp, err := s.authRequest("DELETE", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusForbidden)
	s.assertBody(c, resp, forbiddenResponse)
}
