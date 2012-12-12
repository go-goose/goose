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
	s.service = New(s.Server.URL, versionPath, token, tenantId)
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
func assertJSON(c *C, resp *http.Response, expected interface{}) {
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	err = json.Unmarshal(body, &expected)
	c.Assert(err, IsNil)
	// TODO(dimitern) Validate expected's contents (possibly "laxer" DeepEquals)
}

// assertBody asserts the passed http.Response's body matches the
// expected response, replacing any variables in the expected body.
func assertBody(c *C, resp *http.Response, expected response) {
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
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for header, values := range headers {
		for _, value := range values {
			req.Header.Add(header, value)
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

// makeFlavors takes any number of nova.FlavorDetail objects and
// returns them as a list.
func makeFlavors(flavor ...nova.FlavorDetail) []nova.FlavorDetail {
	return append([]nova.FlavorDetail{}, flavor...)
}

// setHeader creates http.Header map, sets the given header, and
// returns the map.
func setHeader(header, value string) http.Header {
	h := make(http.Header)
	h.Set(header, value)
	return h
}

var simpleTests = []struct {
	unauth  bool
	method  string
	url     string
	headers http.Header
	expect  response
	flavors []nova.FlavorDetail
}{
	{
		unauth:  true,
		method:  "GET",
		url:     "/any",
		headers: make(http.Header),
		expect:  unauthorizedResponse,
	},
	{
		unauth:  true,
		method:  "POST",
		url:     "/any",
		headers: setHeader(authToken, "phony"),
		expect:  unauthorizedResponse,
	},
	{
		unauth:  true,
		method:  "GET",
		url:     "/",
		headers: setHeader(authToken, token),
		expect:  noVersionResponse,
	},
	{
		unauth:  true,
		method:  "GET",
		url:     "/any",
		headers: setHeader(authToken, token),
		expect:  multipleChoicesResponse,
	},
	{
		unauth:  true,
		method:  "POST",
		url:     "/any/unknown/one",
		headers: setHeader(authToken, token),
		expect:  multipleChoicesResponse,
	},
	{
		method: "POST",
		url:    "/any/unknown/one",
		expect: notFoundResponse,
	},
	{
		unauth:  true,
		method:  "GET",
		url:     versionPath + "/phony_token",
		headers: setHeader(authToken, token),
		expect:  badRequestResponse,
	},
	{
		method: "GET",
		url:    "/flavors",
		expect: noContentResponse,
	},
	{
		method: "GET",
		url:    "/flavors/",
		expect: notFoundResponse,
	},
	{
		method: "GET",
		url:    "/flavors/invalid",
		expect: notFoundResponse,
	},
	{
		method:  "GET",
		url:     "/flavors/invalid",
		expect:  notFoundResponse,
		flavors: makeFlavors(nova.FlavorDetail{Id: "fl1"}),
	},
	{
		method: "POST",
		url:    "/flavors/invalid",
		expect: notFoundResponse,
	},
	{
		method: "POST",
		url:    "/flavors",
		expect: badRequest2Response,
	},
	{
		method: "PUT",
		url:    "/flavors",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/flavors",
		expect: notFoundResponse,
	},
	{
		method: "POST",
		url:    "/flavors/detail",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/flavors/detail",
		expect: notFoundJSONResponse,
	},
	{
		method: "DELETE",
		url:    "/flavors/detail",
		expect: forbiddenResponse,
	},
}

func (s *NovaHTTPSuite) TestSimpleRequestTests(c *C) {
	for i, t := range simpleTests {
		c.Logf("#%d. %s %s -> %d\n", i+1, t.method, t.url, t.expect.code)
		for _, flavor := range t.flavors {
			s.service.buildFlavorLinks(&flavor)
			err := s.service.addFlavor(flavor)
			defer s.service.removeFlavor(flavor.Id)
			c.Assert(err, IsNil)
		}
		if t.headers == nil {
			t.headers = make(http.Header)
			t.headers.Set(authToken, s.service.token)
		}
		var (
			resp *http.Response
			err  error
		)
		if t.unauth {
			resp, err = s.sendRequest(t.method, t.url, nil, t.headers)
		} else {
			resp, err = s.authRequest(t.method, t.url, nil, t.headers)
		}
		c.Assert(err, IsNil)
		c.Assert(resp.StatusCode, Equals, t.expect.code)
		assertBody(c, resp, t.expect)
	}
}

func (s *NovaHTTPSuite) TestGetFlavors(c *C) {
	entities := s.service.allFlavorsAsEntities()
	c.Assert(entities, HasLen, 0)
	flavors := []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1", Name: "flavor 1"},
		nova.FlavorDetail{Id: "fl2", Name: "flavor 2"},
	}
	for _, flavor := range flavors {
		s.service.buildFlavorLinks(&flavor)
		err := s.service.addFlavor(flavor)
		defer s.service.removeFlavor(flavor.Id)
		c.Assert(err, IsNil)
	}
	entities = s.service.allFlavorsAsEntities()
	resp, err := s.authRequest("GET", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	var expected struct {
		Flavors []nova.Entity
	}
	assertJSON(c, resp, expected)
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
	assertBody(c, resp, createdResponse)
	_, err = s.service.flavor("fl1")
	c.Assert(err, IsNil)
	s.service.removeFlavor("fl1")
}

func (s *NovaHTTPSuite) TestGetFlavorsDetail(c *C) {
	flavors := s.service.allFlavors()
	c.Assert(flavors, HasLen, 0)
	flavors = []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1", Name: "flavor 1"},
		nova.FlavorDetail{Id: "fl2", Name: "flavor 2"},
	}
	for _, flavor := range flavors {
		s.service.buildFlavorLinks(&flavor)
		err := s.service.addFlavor(flavor)
		defer s.service.removeFlavor(flavor.Id)
		c.Assert(err, IsNil)
	}
	resp, err := s.authRequest("GET", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	var expected struct {
		Flavors []nova.FlavorDetail
	}
	assertJSON(c, resp, expected)
}
