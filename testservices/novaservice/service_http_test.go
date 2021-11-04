// Nova double testing service - HTTP API tests

package novaservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	gc "gopkg.in/check.v1"

	"github.com/go-goose/goose/v4/nova"
	"github.com/go-goose/goose/v4/testing/httpsuite"
	"github.com/go-goose/goose/v4/testservices/hook"
	"github.com/go-goose/goose/v4/testservices/identityservice"
	"github.com/go-goose/goose/v4/testservices/neutronmodel"
)

type NovaHTTPSuite struct {
	httpsuite.HTTPSuite
	service              *Nova
	token                string
	useNeutronNetworking bool
}

var _ = gc.Suite(&NovaHTTPSuite{useNeutronNetworking: false})

type NovaHTTPSSuite struct {
	httpsuite.HTTPSuite
	service              *Nova
	token                string
	useNeutronNetworking bool
}

var _ = gc.Suite(&NovaHTTPSSuite{HTTPSuite: httpsuite.HTTPSuite{UseTLS: true}, useNeutronNetworking: true})

func (s *NovaHTTPSuite) SetUpSuite(c *gc.C) {
	s.HTTPSuite.SetUpSuite(c)
	identityDouble := identityservice.NewUserPass()
	userInfo := identityDouble.AddUser("fred", "secret", "tenant", "default")
	s.token = userInfo.Token
	s.service = New(s.Server.URL, versionPath, userInfo.TenantId, region, identityDouble, nil)
	if s.useNeutronNetworking {
		c.Logf("Nova Service using Neutron Networking")
		s.service.AddNeutronModel(neutronmodel.New())
	} else {
		c.Logf("Nova Service using Nova Networking")
	}
}

func (s *NovaHTTPSuite) TearDownSuite(c *gc.C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *NovaHTTPSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
	// this is otherwise handled not directly by nova test service
	// but by openstack that tries for / before.
	s.Mux.Handle("/", s.service.handler((*Nova).handleRoot))
}

func (s *NovaHTTPSuite) TearDownTest(c *gc.C) {
	s.HTTPSuite.TearDownTest(c)
}

// assertJSON asserts the passed http.Response's body can be
// unmarshalled into the given expected object, populating it with the
// successfully parsed data.
func assertJSON(c *gc.C, resp *http.Response, expected interface{}) {
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, gc.IsNil)
	err = json.Unmarshal(body, &expected)
	c.Assert(err, gc.IsNil)
}

// assertBody asserts the passed http.Response's body matches the
// expected response, replacing any variables in the expected body.
func assertBody(c *gc.C, resp *http.Response, expected *errorResponse) {
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, gc.IsNil)
	expBody := expected.requestBody(resp.Request)
	// cast to string for easier asserts debugging
	c.Assert(string(body), gc.Equals, string(expBody))
}

// sendRequest constructs an HTTP request from the parameters and
// sends it, returning the response or an error.
func (s *NovaHTTPSuite) sendRequest(method, url string, body []byte, headers http.Header) (*http.Response, error) {
	if !strings.HasPrefix(url, "http") {
		url = "http://" + s.service.Hostname + strings.TrimLeft(url, "/")
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
	headers.Set(authToken, s.token)
	url := s.service.endpointURL(true, path)
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

// setHeader creates http.Header map, sets the given header, and
// returns the map.
func setHeader(header, value string) http.Header {
	h := make(http.Header)
	h.Set(header, value)
	return h
}

// SimpleTest defines a simple request without a body and expected response.
type SimpleTest struct {
	unauth  bool
	method  string
	url     string
	headers http.Header
	expect  *errorResponse
}

func (s *NovaHTTPSuite) simpleTests() []SimpleTest {
	var simpleTests = []SimpleTest{
		{
			unauth:  true,
			method:  "GET",
			url:     "/any",
			headers: make(http.Header),
			expect:  errUnauthorized,
		},
		{
			unauth:  true,
			method:  "POST",
			url:     "/any",
			headers: setHeader(authToken, "phony"),
			expect:  errUnauthorized,
		},
		{
			unauth:  true,
			method:  "GET",
			url:     "/any",
			headers: setHeader(authToken, s.token),
			expect:  errMultipleChoices,
		},
		{
			unauth:  true,
			method:  "POST",
			url:     "/any/unknown/one",
			headers: setHeader(authToken, s.token),
			expect:  errMultipleChoices,
		},
		{
			method: "POST",
			url:    "/any/unknown/one",
			expect: errNotFound,
		},
		{
			unauth:  true,
			method:  "GET",
			url:     versionPath + "/phony_token",
			headers: setHeader(authToken, s.token),
			expect:  errBadRequest,
		},
		{
			method: "GET",
			url:    "/flavors/",
			expect: errNotFound,
		},
		{
			method: "GET",
			url:    "/flavors/invalid",
			expect: errNotFound,
		},
		{
			method: "POST",
			url:    "/flavors",
			expect: errBadRequest2,
		},
		{
			method: "POST",
			url:    "/flavors/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/flavors",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/flavors/invalid",
			expect: errNotFoundJSON,
		},
		{
			method: "DELETE",
			url:    "/flavors",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    "/flavors/invalid",
			expect: errForbidden,
		},
		{
			method: "GET",
			url:    "/flavors/detail/invalid",
			expect: errNotFound,
		},
		{
			method: "POST",
			url:    "/flavors/detail",
			expect: errNotFound,
		},
		{
			method: "POST",
			url:    "/flavors/detail/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/flavors/detail",
			expect: errNotFoundJSON,
		},
		{
			method: "PUT",
			url:    "/flavors/detail/invalid",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    "/flavors/detail",
			expect: errForbidden,
		},
		{
			method: "DELETE",
			url:    "/flavors/detail/invalid",
			expect: errNotFound,
		},
		{
			method: "GET",
			url:    "/servers/invalid",
			expect: &errorResponse{code: 404, body: "{\"itemNotFound\":{\"message\":\"No such server \\\"invalid\\\"\", \"code\":404}}"},
		},
		{
			method: "POST",
			url:    "/servers",
			expect: errBadRequest2,
		},
		{
			method: "POST",
			url:    "/servers/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/servers",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/servers/invalid",
			expect: errBadRequest2,
		},
		{
			method: "DELETE",
			url:    "/servers",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    "/servers/invalid",
			expect: errNotFoundJSON,
		},
		{
			method: "GET",
			url:    "/servers/detail/invalid",
			expect: errNotFound,
		},
		{
			method: "POST",
			url:    "/servers/detail",
			expect: errNotFound,
		},
		{
			method: "POST",
			url:    "/servers/detail/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/servers/detail",
			expect: errBadRequest2,
		},
		{
			method: "PUT",
			url:    "/servers/detail/invalid",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    "/servers/detail",
			expect: errNotFoundJSON,
		},
		{
			method: "DELETE",
			url:    "/servers/detail/invalid",
			expect: errNotFound,
		},
	}
	return simpleTests
}

func (s *NovaHTTPSuite) simpleNovaNetworkingTests() []SimpleTest {
	var simpleTests = []SimpleTest{
		{
			method: "GET",
			url:    "/os-security-groups/42",
			expect: errNotFoundJSONSG,
		},
		{
			method: "POST",
			url:    "/os-security-groups",
			expect: errBadRequest2,
		},
		{
			method: "POST",
			url:    "/os-security-groups/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/os-security-groups",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/os-security-groups/invalid",
			expect: errNotFoundJSONSG,
		},
		{
			method: "DELETE",
			url:    "/os-security-groups",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    "/os-security-groups/42",
			expect: errNotFoundJSONSG,
		},
		{
			method: "GET",
			url:    "/os-security-group-rules",
			expect: errNotFoundJSON,
		},
		{
			method: "GET",
			url:    "/os-security-group-rules/invalid",
			expect: errNotFoundJSON,
		},
		{
			method: "GET",
			url:    "/os-security-group-rules/42",
			expect: errNotFoundJSON,
		},
		{
			method: "POST",
			url:    "/os-security-group-rules",
			expect: errBadRequest2,
		},
		{
			method: "POST",
			url:    "/os-security-group-rules/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/os-security-group-rules",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/os-security-group-rules/invalid",
			expect: errNotFoundJSON,
		},
		{
			method: "DELETE",
			url:    "/os-security-group-rules",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    "/os-security-group-rules/42",
			expect: errNotFoundJSONSGR,
		},
		{
			method: "GET",
			url:    "/os-floating-ips/42",
			expect: errNotFoundJSON,
		},
		{
			method: "POST",
			url:    "/os-floating-ips/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/os-floating-ips",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    "/os-floating-ips/invalid",
			expect: errNotFoundJSON,
		},
		{
			method: "DELETE",
			url:    "/os-floating-ips",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    "/os-floating-ips/invalid",
			expect: errNotFoundJSON,
		},
	}
	return simpleTests
}

func (s *NovaHTTPSuite) TestSimpleRequestTests(c *gc.C) {
	s.runSimpleTests(c, s.simpleTests())
	if !s.useNeutronNetworking {
		s.runSimpleTests(c, s.simpleNovaNetworkingTests())
	}
}

func (s *NovaHTTPSuite) runSimpleTests(c *gc.C, simpleTests []SimpleTest) {
	for i, t := range simpleTests {
		c.Logf("#%d. %s %s -> %d", i, t.method, t.url, t.expect.code)
		if t.headers == nil {
			t.headers = make(http.Header)
			t.headers.Set(authToken, s.token)
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
		c.Assert(err, gc.IsNil)
		c.Assert(resp.StatusCode, gc.Equals, t.expect.code)
		assertBody(c, resp, t.expect)
	}
	fmt.Printf("total: %d\n", len(simpleTests))
}

func (s *NovaHTTPSuite) TestGetFlavors(c *gc.C) {
	// The test service has 3 default flavours.
	var expected struct {
		Flavors []nova.Entity
	}
	resp, err := s.authRequest("GET", "/flavors", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Flavors, gc.HasLen, 3)
	entities := s.service.allFlavorsAsEntities()
	c.Assert(entities, gc.HasLen, 3)
	sort.Sort(nova.EntitySortBy{Attr: "Id", Entities: expected.Flavors})
	sort.Sort(nova.EntitySortBy{Attr: "Id", Entities: entities})
	c.Assert(expected.Flavors, gc.DeepEquals, entities)
	var expectedFlavor struct {
		Flavor nova.FlavorDetail
	}
	resp, err = s.authRequest("GET", "/flavors/1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedFlavor)
	c.Assert(expectedFlavor.Flavor.Name, gc.Equals, "m1.tiny")
}

func (s *NovaHTTPSuite) TestGetFlavorsDetail(c *gc.C) {
	// The test service has 3 default flavours.
	flavors := s.service.allFlavors()
	c.Assert(flavors, gc.HasLen, 3)
	var expected struct {
		Flavors []nova.FlavorDetail
	}
	resp, err := s.authRequest("GET", "/flavors/detail", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Flavors, gc.HasLen, 3)
	sort.Sort(nova.FlavorDetailSortBy{Attr: "Id", FlavorDetails: expected.Flavors})
	sort.Sort(nova.FlavorDetailSortBy{Attr: "Id", FlavorDetails: flavors})
	c.Assert(expected.Flavors, gc.DeepEquals, flavors)
	resp, err = s.authRequest("GET", "/flavors/detail/1", nil, nil)
	c.Assert(err, gc.IsNil)
	assertBody(c, resp, errNotFound)
}

func (s *NovaHTTPSuite) TestGetServers(c *gc.C) {
	entities, err := s.service.allServersAsEntities(nil)
	c.Assert(err, gc.IsNil)
	c.Assert(entities, gc.HasLen, 0)
	var expected struct {
		Servers []nova.Entity
	}
	resp, err := s.authRequest("GET", "/servers", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 0)
	servers := []nova.ServerDetail{
		{Id: "sr1", Name: "server 1"},
		{Id: "sr2", Name: "server 2"},
	}
	for i, server := range servers {
		s.service.buildServerLinks(&server)
		servers[i] = server
		err := s.service.addServer(server)
		c.Assert(err, gc.IsNil)
		defer s.service.removeServer(server.Id)
	}
	entities, err = s.service.allServersAsEntities(nil)
	c.Assert(err, gc.IsNil)
	resp, err = s.authRequest("GET", "/servers", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 2)
	if expected.Servers[0].Id != entities[0].Id {
		expected.Servers[0], expected.Servers[1] = expected.Servers[1], expected.Servers[0]
	}
	c.Assert(expected.Servers, gc.DeepEquals, entities)
	var expectedServer struct {
		Server nova.ServerDetail
	}
	resp, err = s.authRequest("GET", "/servers/sr1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedServer)
	servers[0].Status = nova.StatusActive
	c.Assert(expectedServer.Server, gc.DeepEquals, servers[0])
}

func (s *NovaHTTPSuite) TestGetServersWithFilters(c *gc.C) {
	entities, err := s.service.allServersAsEntities(nil)
	c.Assert(err, gc.IsNil)
	c.Assert(entities, gc.HasLen, 0)
	var expected struct {
		Servers []nova.Entity
	}
	url := "/servers?status=RESCUE&status=BUILD&name=srv2&name=srv1"
	resp, err := s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 0)
	servers := []nova.ServerDetail{
		{Id: "sr1", Name: "srv1", Status: nova.StatusBuild},
		{Id: "sr2", Name: "srv2", Status: nova.StatusRescue},
		{Id: "sr3", Name: "srv3", Status: nova.StatusActive},
	}
	for i, server := range servers {
		s.service.buildServerLinks(&server)
		servers[i] = server
		err := s.service.addServer(server)
		c.Assert(err, gc.IsNil)
		defer s.service.removeServer(server.Id)
	}
	resp, err = s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 1)
	c.Assert(expected.Servers[0].Id, gc.Equals, servers[0].Id)
	c.Assert(expected.Servers[0].Name, gc.Equals, servers[0].Name)
}

func (s *NovaHTTPSuite) TestGetServersWithBadFilter(c *gc.C) {
	url := "/servers?name=(server"
	resp, err := s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusInternalServerError)
	type novaError struct {
		Code    int
		Message string
	}
	var expected struct {
		novaError `json:"computeFault"`
	}
	assertJSON(c, resp, &expected)
	c.Check(expected.Code, gc.Equals, 500)
	c.Check(expected.Message, gc.Matches, `error parsing.*\(server.*`)
}

func (s *NovaHTTPSuite) TestGetServersPatchMatch(c *gc.C) {
	cleanup := s.service.RegisterControlPoint(
		"matchServers",
		func(sc hook.ServiceControl, args ...interface{}) error {
			return fmt.Errorf("Unexpected error")
		},
	)
	defer cleanup()
	resp, err := s.authRequest("GET", "/servers", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusInternalServerError)
	type novaError struct {
		Code    int
		Message string
	}
	var expected struct {
		novaError `json:"computeFault"`
	}
	assertJSON(c, resp, &expected)
	c.Check(expected.Code, gc.Equals, 500)
	c.Check(expected.Message, gc.Equals, "Unexpected error")
}

func (s *NovaHTTPSuite) TestNewUUID(c *gc.C) {
	uuid, err := newUUID()
	c.Assert(err, gc.IsNil)
	var p1, p2, p3, p4, p5 string
	num, err := fmt.Sscanf(uuid, "%8x-%4x-%4x-%4x-%12x", &p1, &p2, &p3, &p4, &p5)
	c.Assert(err, gc.IsNil)
	c.Assert(num, gc.Equals, 5)
	uuid2, err := newUUID()
	c.Assert(err, gc.IsNil)
	c.Assert(uuid2, gc.Not(gc.Equals), uuid)
}

func (s *NovaHTTPSuite) assertAddresses(c *gc.C, serverId string) {
	server, err := s.service.server(serverId)
	c.Assert(err, gc.IsNil)
	c.Assert(server.Addresses, gc.HasLen, 1)
	c.Assert(server.Addresses["public"], gc.HasLen, 2)
	for network, addresses := range server.Addresses {
		for _, addr := range addresses {
			if addr.Version == 4 && network == "public" {
				c.Assert(addr.Address, gc.Matches, `127\.10\.0\.\d{1,3}`)
			}
		}

	}
}

func (s *NovaHTTPSuite) TestRunServer(c *gc.C) {
	entities, err := s.service.allServersAsEntities(nil)
	c.Assert(err, gc.IsNil)
	c.Assert(entities, gc.HasLen, 0)
	var req struct {
		Server struct {
			FlavorRef      string              `json:"flavorRef"`
			ImageRef       string              `json:"imageRef"`
			Name           string              `json:"name"`
			SecurityGroups []map[string]string `json:"security_groups"`
		} `json:"server"`
	}
	resp, err := s.jsonRequest("POST", "/servers", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusBadRequest)
	assertBody(c, resp, errBadRequestSrvName)
	req.Server.Name = "srv1"
	resp, err = s.jsonRequest("POST", "/servers", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusBadRequest)
	assertBody(c, resp, errBadRequestSrvImage)
	req.Server.ImageRef = "image"
	resp, err = s.jsonRequest("POST", "/servers", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusBadRequest)
	assertBody(c, resp, errBadRequestSrvFlavor)
	req.Server.FlavorRef = "flavor"
	var expected struct {
		Server struct {
			SecurityGroups []map[string]string `json:"security_groups"`
			Id             string
			Links          []nova.Link
			AdminPass      string
		}
	}
	resp, err = s.jsonRequest("POST", "/servers", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Server.SecurityGroups, gc.HasLen, 1)
	c.Assert(expected.Server.SecurityGroups[0]["name"], gc.Equals, "default")
	c.Assert(expected.Server.Id, gc.Not(gc.Equals), "")
	c.Assert(expected.Server.Links, gc.HasLen, 2)
	c.Assert(expected.Server.AdminPass, gc.Not(gc.Equals), "")
	s.assertAddresses(c, expected.Server.Id)
	srv, err := s.service.server(expected.Server.Id)
	c.Assert(err, gc.IsNil)
	c.Assert(srv.Links, gc.DeepEquals, expected.Server.Links)
	s.service.removeServer(srv.Id)
	req.Server.Name = "test2"
	req.Server.SecurityGroups = []map[string]string{
		{"name": "default"},
		{"name": "group1"},
		{"name": "group2"},
	}
	err = s.service.addSecurityGroup(nova.SecurityGroup{Id: "1", Name: "group1"})
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup("1")
	err = s.service.addSecurityGroup(nova.SecurityGroup{Id: "2", Name: "group2"})
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup("2")
	resp, err = s.jsonRequest("POST", "/servers", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Server.SecurityGroups, gc.DeepEquals, req.Server.SecurityGroups)
	srv, err = s.service.server(expected.Server.Id)
	c.Assert(err, gc.IsNil)
	ok := s.service.hasServerSecurityGroup(srv.Id, "1")
	c.Assert(ok, gc.Equals, true)
	ok = s.service.hasServerSecurityGroup(srv.Id, "2")
	c.Assert(ok, gc.Equals, true)
	ok = s.service.hasServerSecurityGroup(srv.Id, "999")
	c.Assert(ok, gc.Equals, true)
	s.service.removeServerSecurityGroup(srv.Id, "1")
	s.service.removeServerSecurityGroup(srv.Id, "2")
	s.service.removeServerSecurityGroup(srv.Id, "999")
	s.service.removeServer(srv.Id)
}

func (s *NovaHTTPSuite) TestDeleteServer(c *gc.C) {
	server := nova.ServerDetail{Id: "sr1"}
	_, err := s.service.server(server.Id)
	c.Assert(err, gc.NotNil)
	err = s.service.addServer(server)
	c.Assert(err, gc.IsNil)
	defer s.service.removeServer(server.Id)
	resp, err := s.authRequest("DELETE", "/servers/sr1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNoContent)
	_, err = s.service.server(server.Id)
	c.Assert(err, gc.NotNil)
}

func (s *NovaHTTPSuite) TestGetServersDetail(c *gc.C) {
	servers, err := s.service.allServers(nil)
	c.Assert(err, gc.IsNil)
	c.Assert(servers, gc.HasLen, 0)
	var expected struct {
		Servers []nova.ServerDetail `json:"servers"`
	}
	resp, err := s.authRequest("GET", "/servers/detail", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 0)
	servers = []nova.ServerDetail{
		{Id: "sr1", Name: "server 1"},
		{Id: "sr2", Name: "server 2"},
	}
	for i, server := range servers {
		s.service.buildServerLinks(&server)
		servers[i] = server
		err := s.service.addServer(server)
		c.Assert(err, gc.IsNil)
		defer s.service.removeServer(server.Id)
	}
	resp, err = s.authRequest("GET", "/servers/detail", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 2)
	if expected.Servers[0].Id != servers[0].Id {
		expected.Servers[0], expected.Servers[1] = expected.Servers[1], expected.Servers[0]
	}
	c.Assert(expected.Servers, gc.DeepEquals, servers)
	resp, err = s.authRequest("GET", "/servers/detail/sr1", nil, nil)
	c.Assert(err, gc.IsNil)
	assertBody(c, resp, errNotFound)
}

func (s *NovaHTTPSuite) TestGetServersDetailWithFilters(c *gc.C) {
	servers, err := s.service.allServers(nil)
	c.Assert(err, gc.IsNil)
	c.Assert(servers, gc.HasLen, 0)
	var expected struct {
		Servers []nova.ServerDetail `json:"servers"`
	}
	url := "/servers/detail?status=RESCUE&status=BUILD&name=srv2&name=srv1"
	resp, err := s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 0)
	servers = []nova.ServerDetail{
		{Id: "sr1", Name: "srv1", Status: nova.StatusBuild},
		{Id: "sr2", Name: "srv2", Status: nova.StatusRescue},
		{Id: "sr3", Name: "srv3", Status: nova.StatusActive},
	}
	for i, server := range servers {
		s.service.buildServerLinks(&server)
		servers[i] = server
		err := s.service.addServer(server)
		c.Assert(err, gc.IsNil)
		defer s.service.removeServer(server.Id)
	}
	resp, err = s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, gc.HasLen, 1)
	c.Assert(expected.Servers[0], gc.DeepEquals, servers[0])
}

func (s *NovaHTTPSuite) TestGetSecurityGroups(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	// There is always a default security group.
	groups := s.service.allSecurityGroups()
	c.Assert(groups, gc.HasLen, 1)
	var expected struct {
		Groups []nova.SecurityGroup `json:"security_groups"`
	}
	resp, err := s.authRequest("GET", "/os-security-groups", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Groups, gc.HasLen, 1)
	groups = []nova.SecurityGroup{
		{
			Id:       "1",
			Name:     "group 1",
			TenantId: s.service.TenantId,
			Rules:    []nova.SecurityGroupRule{},
		},
		{
			Id:       "2",
			Name:     "group 2",
			TenantId: s.service.TenantId,
			Rules:    []nova.SecurityGroupRule{},
		},
	}
	for _, group := range groups {
		err := s.service.addSecurityGroup(group)
		c.Assert(err, gc.IsNil)
		defer s.service.removeSecurityGroup(group.Id)
	}
	resp, err = s.authRequest("GET", "/os-security-groups", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Groups, gc.HasLen, len(groups)+1)
	checkGroupsInList(c, groups, expected.Groups)
	var expectedGroup struct {
		Group nova.SecurityGroup `json:"security_group"`
	}
	resp, err = s.authRequest("GET", "/os-security-groups/1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedGroup)
	c.Assert(expectedGroup.Group, gc.DeepEquals, groups[0])
}

func (s *NovaHTTPSuite) TestAddSecurityGroup(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	group := nova.SecurityGroup{
		Id:          "1",
		Name:        "group 1",
		Description: "desc",
		TenantId:    s.service.TenantId,
		Rules:       []nova.SecurityGroupRule{},
	}
	_, err := s.service.securityGroup(group.Id)
	c.Assert(err, gc.NotNil)
	var req struct {
		Group struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"security_group"`
	}
	req.Group.Name = group.Name
	req.Group.Description = group.Description
	var expected struct {
		Group nova.SecurityGroup `json:"security_group"`
	}
	resp, err := s.jsonRequest("POST", "/os-security-groups", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Group, gc.DeepEquals, group)
	err = s.service.removeSecurityGroup(group.Id)
	c.Assert(err, gc.IsNil)
}

func (s *NovaHTTPSuite) TestDeleteSecurityGroup(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	group := nova.SecurityGroup{Id: "1", Name: "group 1"}
	_, err := s.service.securityGroup(group.Id)
	c.Assert(err, gc.NotNil)
	err = s.service.addSecurityGroup(group)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group.Id)
	resp, err := s.authRequest("DELETE", "/os-security-groups/1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	_, err = s.service.securityGroup(group.Id)
	c.Assert(err, gc.NotNil)
}

func (s *NovaHTTPSuite) TestAddSecurityGroupRule(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	group1 := nova.SecurityGroup{Id: "1", Name: "src"}
	group2 := nova.SecurityGroup{Id: "2", Name: "tgt"}
	err := s.service.addSecurityGroup(group1)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group1.Id)
	err = s.service.addSecurityGroup(group2)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group2.Id)
	riIngress := nova.RuleInfo{
		ParentGroupId: "1",
		FromPort:      1234,
		ToPort:        4321,
		IPProtocol:    "tcp",
		Cidr:          "1.2.3.4/5",
	}
	riGroup := nova.RuleInfo{
		ParentGroupId: group2.Id,
		GroupId:       &group1.Id,
	}
	iprange := make(map[string]string)
	iprange["cidr"] = riIngress.Cidr
	rule1 := nova.SecurityGroupRule{
		Id:            "1",
		ParentGroupId: group1.Id,
		FromPort:      &riIngress.FromPort,
		ToPort:        &riIngress.ToPort,
		IPProtocol:    &riIngress.IPProtocol,
		IPRange:       iprange,
	}
	rule2 := nova.SecurityGroupRule{
		Id:            "2",
		ParentGroupId: group2.Id,
		Group: nova.SecurityGroupRef{
			Name:     group1.Name,
			TenantId: s.service.TenantId,
		},
	}
	ok := s.service.hasSecurityGroupRule(group1.Id, rule1.Id)
	c.Assert(ok, gc.Equals, false)
	ok = s.service.hasSecurityGroupRule(group2.Id, rule2.Id)
	c.Assert(ok, gc.Equals, false)
	var req struct {
		Rule nova.RuleInfo `json:"security_group_rule"`
	}
	req.Rule = riIngress
	var expected struct {
		Rule nova.SecurityGroupRule `json:"security_group_rule"`
	}
	resp, err := s.jsonRequest("POST", "/os-security-group-rules", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, gc.Equals, rule1.Id)
	c.Assert(expected.Rule.ParentGroupId, gc.Equals, rule1.ParentGroupId)
	c.Assert(expected.Rule.Group, gc.Equals, nova.SecurityGroupRef{})
	c.Assert(*expected.Rule.FromPort, gc.Equals, *rule1.FromPort)
	c.Assert(*expected.Rule.ToPort, gc.Equals, *rule1.ToPort)
	c.Assert(*expected.Rule.IPProtocol, gc.Equals, *rule1.IPProtocol)
	c.Assert(expected.Rule.IPRange, gc.DeepEquals, rule1.IPRange)
	defer s.service.removeSecurityGroupRule(rule1.Id)
	req.Rule = riGroup
	resp, err = s.jsonRequest("POST", "/os-security-group-rules", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, gc.Equals, rule2.Id)
	c.Assert(expected.Rule.ParentGroupId, gc.Equals, rule2.ParentGroupId)
	c.Assert(expected.Rule.Group, gc.DeepEquals, rule2.Group)
	err = s.service.removeSecurityGroupRule(rule2.Id)
	c.Assert(err, gc.IsNil)
}

func (s *NovaHTTPSuite) TestDeleteSecurityGroupRule(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	group1 := nova.SecurityGroup{Id: "1", Name: "src"}
	group2 := nova.SecurityGroup{Id: "2", Name: "tgt"}
	err := s.service.addSecurityGroup(group1)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group1.Id)
	err = s.service.addSecurityGroup(group2)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group2.Id)
	riGroup := nova.RuleInfo{
		ParentGroupId: group2.Id,
		GroupId:       &group1.Id,
	}
	rule := nova.SecurityGroupRule{
		Id:            "1",
		ParentGroupId: group2.Id,
		Group: nova.SecurityGroupRef{
			Name:     group1.Name,
			TenantId: group1.TenantId,
		},
	}
	err = s.service.addSecurityGroupRule(rule.Id, riGroup)
	c.Assert(err, gc.IsNil)
	resp, err := s.authRequest("DELETE", "/os-security-group-rules/1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	ok := s.service.hasSecurityGroupRule(group2.Id, rule.Id)
	c.Assert(ok, gc.Equals, false)
}

func (s *NovaHTTPSuite) TestAddServerSecurityGroup(c *gc.C) {
	group := nova.SecurityGroup{Id: "1", Name: "group"}
	err := s.service.addSecurityGroup(group)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group.Id)
	server := nova.ServerDetail{Id: "sr1"}
	err = s.service.addServer(server)
	c.Assert(err, gc.IsNil)
	defer s.service.removeServer(server.Id)
	ok := s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, gc.Equals, false)
	var req struct {
		Group struct {
			Name string `json:"name"`
		} `json:"addSecurityGroup"`
	}
	req.Group.Name = group.Name
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, gc.Equals, true)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, gc.IsNil)
}

func (s *NovaHTTPSuite) TestGetServerSecurityGroups(c *gc.C) {
	server := nova.ServerDetail{Id: "sr1"}
	groups := []nova.SecurityGroup{
		{
			Id:       "1",
			Name:     "group1",
			TenantId: s.service.TenantId,
			Rules:    []nova.SecurityGroupRule{},
		},
		{
			Id:       "2",
			Name:     "group2",
			TenantId: s.service.TenantId,
			Rules:    []nova.SecurityGroupRule{},
		},
	}
	srvGroups := s.service.allServerSecurityGroups(server.Id)
	c.Assert(srvGroups, gc.HasLen, 0)
	err := s.service.addServer(server)
	c.Assert(err, gc.IsNil)
	defer s.service.removeServer(server.Id)
	for _, group := range groups {
		err = s.service.addSecurityGroup(group)
		c.Assert(err, gc.IsNil)
		defer s.service.removeSecurityGroup(group.Id)
		err = s.service.addServerSecurityGroup(server.Id, group.Id)
		c.Assert(err, gc.IsNil)
		defer s.service.removeServerSecurityGroup(server.Id, group.Id)
	}
	var expected struct {
		Groups []nova.SecurityGroup `json:"security_groups"`
	}
	resp, err := s.authRequest("GET", "/servers/"+server.Id+"/os-security-groups", nil, nil)
	c.Assert(err, gc.IsNil)
	assertJSON(c, resp, &expected)
	// nova networking doesn't know about neutron egress direction rules,
	// created by default with a new security group
	if s.service.useNeutronNetworking {
		expected.Groups[0].Rules = []nova.SecurityGroupRule{}
		expected.Groups[1].Rules = []nova.SecurityGroupRule{}
	}
	c.Assert(expected.Groups, gc.DeepEquals, groups)
}

func (s *NovaHTTPSuite) TestDeleteServerSecurityGroup(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	group := nova.SecurityGroup{Id: "1", Name: "group"}
	err := s.service.addSecurityGroup(group)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group.Id)
	server := nova.ServerDetail{Id: "sr1"}
	err = s.service.addServer(server)
	c.Assert(err, gc.IsNil)
	defer s.service.removeServer(server.Id)
	ok := s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, gc.Equals, false)
	err = s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, gc.IsNil)
	var req struct {
		Group struct {
			Name string `json:"name"`
		} `json:"removeSecurityGroup"`
	}
	req.Group.Name = group.Name
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, gc.Equals, false)
}

func (s *NovaHTTPSuite) TestPostFloatingIP(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	fip := nova.FloatingIP{Id: "1", IP: "10.0.0.1", Pool: "nova"}
	c.Assert(s.service.allFloatingIPs(), gc.HasLen, 0)
	var expected struct {
		IP nova.FloatingIP `json:"floating_ip"`
	}
	resp, err := s.authRequest("POST", "/os-floating-ips", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.IP, gc.DeepEquals, fip)
	err = s.service.removeFloatingIP(fip.Id)
	c.Assert(err, gc.IsNil)
}

func (s *NovaHTTPSuite) TestGetFloatingIPs(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	c.Assert(s.service.allFloatingIPs(), gc.HasLen, 0)
	var expected struct {
		IPs []nova.FloatingIP `json:"floating_ips"`
	}
	resp, err := s.authRequest("GET", "/os-floating-ips", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.IPs, gc.HasLen, 0)
	fips := []nova.FloatingIP{
		{Id: "1", IP: "1.2.3.4", Pool: "nova"},
		{Id: "2", IP: "4.3.2.1", Pool: "nova"},
	}
	for _, fip := range fips {
		err := s.service.addFloatingIP(fip)
		defer s.service.removeFloatingIP(fip.Id)
		c.Assert(err, gc.IsNil)
	}
	resp, err = s.authRequest("GET", "/os-floating-ips", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.IPs[0].Id != fips[0].Id {
		expected.IPs[0], expected.IPs[1] = expected.IPs[1], expected.IPs[0]
	}
	c.Assert(expected.IPs, gc.DeepEquals, fips)
	var expectedIP struct {
		IP nova.FloatingIP `json:"floating_ip"`
	}
	resp, err = s.authRequest("GET", "/os-floating-ips/1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedIP)
	c.Assert(expectedIP.IP, gc.DeepEquals, fips[0])
}

func (s *NovaHTTPSuite) TestDeleteFloatingIP(c *gc.C) {
	if s.service.useNeutronNetworking {
		c.Skip("skipped in novaservice when using Neutron Model")
	}
	fip := nova.FloatingIP{Id: "1", IP: "10.0.0.1", Pool: "nova"}
	err := s.service.addFloatingIP(fip)
	c.Assert(err, gc.IsNil)
	defer s.service.removeFloatingIP(fip.Id)
	resp, err := s.authRequest("DELETE", "/os-floating-ips/1", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	_, err = s.service.floatingIP(fip.Id)
	c.Assert(err, gc.NotNil)
}

func (s *NovaHTTPSuite) TestAddServerFloatingIP(c *gc.C) {
	fip := nova.FloatingIP{Id: "1", IP: "1.2.3.4"}
	server := nova.ServerDetail{
		Id:        "sr1",
		Addresses: map[string][]nova.IPAddress{"private": {}},
	}
	err := s.service.addFloatingIP(fip)
	c.Assert(err, gc.IsNil)
	defer s.service.removeFloatingIP(fip.Id)
	err = s.service.addServer(server)
	c.Assert(err, gc.IsNil)
	defer s.service.removeServer(server.Id)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), gc.Equals, false)
	var req struct {
		AddFloatingIP struct {
			Address string `json:"address"`
		} `json:"addFloatingIp"`
	}
	req.AddFloatingIP.Address = fip.IP
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), gc.Equals, true)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, gc.IsNil)
}

func (s *NovaHTTPSuite) TestRemoveServerFloatingIP(c *gc.C) {
	fip := nova.FloatingIP{Id: "1", IP: "1.2.3.4"}
	server := nova.ServerDetail{
		Id:        "sr1",
		Addresses: map[string][]nova.IPAddress{"private": {}},
	}
	err := s.service.addFloatingIP(fip)
	c.Assert(err, gc.IsNil)
	defer s.service.removeFloatingIP(fip.Id)
	err = s.service.addServer(server)
	c.Assert(err, gc.IsNil)
	defer s.service.removeServer(server.Id)
	err = s.service.addServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, gc.IsNil)
	defer s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), gc.Equals, true)
	var req struct {
		RemoveFloatingIP struct {
			Address string `json:"address"`
		} `json:"removeFloatingIp"`
	}
	req.RemoveFloatingIP.Address = fip.IP
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusAccepted)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), gc.Equals, false)
}

func (s *NovaHTTPSuite) TestListAvailabilityZones(c *gc.C) {
	resp, err := s.jsonRequest("GET", "/os-availability-zone", nil, nil)
	c.Assert(err, gc.IsNil)
	assertBody(c, resp, errNotFoundJSON)

	zones := []nova.AvailabilityZone{
		{Name: "az1"},
		{
			Name: "az2", State: nova.AvailabilityZoneState{Available: true},
		},
	}
	s.service.SetAvailabilityZones(zones...)
	resp, err = s.jsonRequest("GET", "/os-availability-zone", nil, nil)
	c.Assert(err, gc.IsNil)
	var expected struct {
		Zones []nova.AvailabilityZone `json:"availabilityZoneInfo"`
	}
	assertJSON(c, resp, &expected)
	c.Assert(expected.Zones, gc.DeepEquals, zones)
}

func (s *NovaHTTPSuite) TestAddServerOSInterface(c *gc.C) {
	osInterface := nova.OSInterface{
		FixedIPs: []nova.PortFixedIP{
			{IPAddress: "10.0.0.1", SubnetID: "sub-net-id"},
		},
		IPAddress: "10.0.0.1",
	}
	server := nova.ServerDetail{
		Id:        "sr1",
		Addresses: map[string][]nova.IPAddress{"private": {}},
	}
	s.service.AddOSInterface(server.Id, osInterface)
	c.Assert(s.service.hasServerOSInterface(server.Id, osInterface.IPAddress), gc.Equals, true)

	defer s.service.RemoveOSInterface(server.Id, osInterface.IPAddress)
	s.service.RemoveOSInterface(server.Id, osInterface.IPAddress)

	defer s.service.removeServer(server.Id)
	c.Assert(s.service.hasServerOSInterface(server.Id, osInterface.IPAddress), gc.Equals, false)

	s.service.AddOSInterface(server.Id, osInterface)

	resp, err := s.jsonRequest("GET", "/servers/"+server.Id+"/os-interface", nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
}

func (s *NovaHTTPSSuite) SetUpSuite(c *gc.C) {
	s.HTTPSuite.SetUpSuite(c)
	identityDouble := identityservice.NewUserPass()
	userInfo := identityDouble.AddUser("fred", "secret", "tenant", "default")
	s.token = userInfo.Token
	c.Assert(s.Server.URL[:8], gc.Equals, "https://")
	s.service = New(s.Server.URL, versionPath, userInfo.TenantId, region, identityDouble, nil)
	if s.useNeutronNetworking {
		c.Logf("Nova Service using Neutron Networking")
		s.service.AddNeutronModel(neutronmodel.New())
	} else {
		c.Logf("Nova Service using Nova Networking")
	}
}

func (s *NovaHTTPSSuite) TearDownSuite(c *gc.C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *NovaHTTPSSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
}

func (s *NovaHTTPSSuite) TearDownTest(c *gc.C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *NovaHTTPSSuite) TestHasHTTPSServiceURL(c *gc.C) {
	endpoints := s.service.Endpoints()
	c.Assert(endpoints[0].PublicURL[:8], gc.Equals, "https://")
}

func (s *NovaHTTPSuite) TestSetServerMetadata(c *gc.C) {
	const serverId = "sr1"

	err := s.service.addServer(nova.ServerDetail{Id: serverId})
	c.Assert(err, gc.IsNil)
	defer s.service.removeServer(serverId)
	var req struct {
		Metadata map[string]string `json:"metadata"`
	}
	req.Metadata = map[string]string{
		"k1": "v1",
		"k2": "v2",
	}
	resp, err := s.jsonRequest("POST", "/servers/"+serverId+"/metadata", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)

	server, err := s.service.server(serverId)
	c.Assert(err, gc.IsNil)
	c.Assert(server.Metadata, gc.DeepEquals, req.Metadata)
}

func (s *NovaHTTPSuite) TestAttachVolumeBlankDeviceName(c *gc.C) {
	var req struct {
		VolumeAttachment struct {
			Device string `json:"device"`
		} `json:"volumeAttachment"`
	}
	resp, err := s.jsonRequest("POST", "/servers/123/os-volume_attachments", req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusBadRequest)

	// Passing an empty string in the "device" attribute
	// is invalid. It should be omitted instead.
	message := "Invalid input for field/attribute device. Value: '' does not match '(^/dev/x{0,1}[a-z]{0,1}d{0,1})([a-z]+)[0-9]*$'"
	assertBody(c, resp, &errorResponse{
		http.StatusBadRequest,
		fmt.Sprintf(`{"badRequest": {"message": "%s", "code": 400}}`, message),
		"application/json; charset=UTF-8",
		message,
		nil,
		nil,
	})

}
