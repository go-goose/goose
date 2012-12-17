// Nova double testing service - HTTP API tests

package novaservice

import (
	"bytes"
	"encoding/json"
	"fmt"
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
// unmarshalled into the given expected object, populating it with the
// successfully parsed data.
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

// setHeader creates http.Header map, sets the given header, and
// returns the map.
func setHeader(header, value string) http.Header {
	h := make(http.Header)
	h.Set(header, value)
	return h
}

// simpleTests defines a simple request without a body and expected response.
var simpleTests = []struct {
	unauth  bool
	method  string
	url     string
	headers http.Header
	expect  response
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
		url:    "/flavors/",
		expect: notFoundResponse,
	},
	{
		method: "GET",
		url:    "/flavors/invalid",
		expect: notFoundResponse,
	},
	{
		method: "POST",
		url:    "/flavors",
		expect: badRequest2Response,
	},
	{
		method: "POST",
		url:    "/flavors/invalid",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/flavors",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/flavors/invalid",
		expect: notFoundJSONResponse,
	},
	{
		method: "DELETE",
		url:    "/flavors",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/flavors/invalid",
		expect: forbiddenResponse,
	},
	{
		method: "GET",
		url:    "/flavors/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "POST",
		url:    "/flavors/detail",
		expect: notFoundResponse,
	},
	{
		method: "POST",
		url:    "/flavors/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/flavors/detail",
		expect: notFoundJSONResponse,
	},
	{
		method: "PUT",
		url:    "/flavors/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/flavors/detail",
		expect: forbiddenResponse,
	},
	{
		method: "DELETE",
		url:    "/flavors/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "GET",
		url:    "/servers/invalid",
		expect: notFoundJSONResponse,
	},
	{
		method: "POST",
		url:    "/servers",
		expect: badRequest2Response,
	},
	{
		method: "POST",
		url:    "/servers/invalid",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/servers",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/servers/invalid",
		expect: badRequest2Response,
	},
	{
		method: "DELETE",
		url:    "/servers",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/servers/invalid",
		expect: notFoundJSONResponse,
	},
	{
		method: "GET",
		url:    "/servers/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "POST",
		url:    "/servers/detail",
		expect: notFoundResponse,
	},
	{
		method: "POST",
		url:    "/servers/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/servers/detail",
		expect: badRequest2Response,
	},
	{
		method: "PUT",
		url:    "/servers/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/servers/detail",
		expect: notFoundJSONResponse,
	},
	{
		method: "DELETE",
		url:    "/servers/detail/invalid",
		expect: notFoundResponse,
	},
	{
		method: "GET",
		url:    "/os-security-groups/invalid",
		expect: badRequestSGResponse,
	},
	{
		method: "GET",
		url:    "/os-security-groups/42",
		expect: notFoundJSONSGResponse,
	},
	{
		method: "POST",
		url:    "/os-security-groups",
		expect: badRequest2Response,
	},
	{
		method: "POST",
		url:    "/os-security-groups/invalid",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/os-security-groups",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/os-security-groups/invalid",
		expect: notFoundJSONResponse,
	},
	{
		method: "DELETE",
		url:    "/os-security-groups",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/os-security-groups/invalid",
		expect: badRequestSGResponse,
	},
	{
		method: "DELETE",
		url:    "/os-security-groups/42",
		expect: notFoundJSONSGResponse,
	},
	{
		method: "GET",
		url:    "/os-security-group-rules",
		expect: notFoundJSONResponse,
	},
	{
		method: "GET",
		url:    "/os-security-group-rules/invalid",
		expect: notFoundJSONResponse,
	},
	{
		method: "GET",
		url:    "/os-security-group-rules/42",
		expect: notFoundJSONResponse,
	},
	{
		method: "POST",
		url:    "/os-security-group-rules",
		expect: badRequest2Response,
	},
	{
		method: "POST",
		url:    "/os-security-group-rules/invalid",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/os-security-group-rules",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/os-security-group-rules/invalid",
		expect: notFoundJSONResponse,
	},
	{
		method: "DELETE",
		url:    "/os-security-group-rules",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/os-security-group-rules/invalid",
		expect: badRequestSGResponse, // sic; should've been rule-specific
	},
	{
		method: "DELETE",
		url:    "/os-security-group-rules/42",
		expect: notFoundJSONSGRResponse,
	},
	{
		method: "GET",
		url:    "/os-floating-ips/42",
		expect: notFoundJSONResponse,
	},
	{
		method: "POST",
		url:    "/os-floating-ips/invalid",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/os-floating-ips",
		expect: notFoundResponse,
	},
	{
		method: "PUT",
		url:    "/os-floating-ips/invalid",
		expect: notFoundJSONResponse,
	},
	{
		method: "DELETE",
		url:    "/os-floating-ips",
		expect: notFoundResponse,
	},
	{
		method: "DELETE",
		url:    "/os-floating-ips/invalid",
		expect: notFoundJSONResponse,
	},
}

func (s *NovaHTTPSuite) TestSimpleRequestTests(c *C) {
	for i, t := range simpleTests {
		c.Logf("#%d. %s %s -> %d", i, t.method, t.url, t.expect.code)
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
	fmt.Printf("total: %d\n", len(simpleTests))
}

func (s *NovaHTTPSuite) TestGetFlavors(c *C) {
	entities := s.service.allFlavorsAsEntities()
	c.Assert(entities, HasLen, 0)
	var expected struct {
		Flavors []nova.Entity
	}
	resp, err := s.authRequest("GET", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Flavors, HasLen, 0)
	flavors := []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1", Name: "flavor 1"},
		nova.FlavorDetail{Id: "fl2", Name: "flavor 2"},
	}
	for i, flavor := range flavors {
		s.service.buildFlavorLinks(&flavor)
		flavors[i] = flavor
		err := s.service.addFlavor(flavor)
		defer s.service.removeFlavor(flavor.Id)
		c.Assert(err, IsNil)
	}
	entities = s.service.allFlavorsAsEntities()
	resp, err = s.authRequest("GET", "/flavors", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.Flavors[0].Id != entities[0].Id {
		expected.Flavors[0], expected.Flavors[1] = expected.Flavors[1], expected.Flavors[0]
	}
	c.Assert(expected.Flavors, DeepEquals, entities)
	var expectedFlavor struct {
		Flavor nova.FlavorDetail
	}
	resp, err = s.authRequest("GET", "/flavors/fl1", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expectedFlavor)
	c.Assert(expectedFlavor.Flavor, DeepEquals, flavors[0])
}

func (s *NovaHTTPSuite) TestGetFlavorsDetail(c *C) {
	flavors := s.service.allFlavors()
	c.Assert(flavors, HasLen, 0)
	var expected struct {
		Flavors []nova.FlavorDetail
	}
	resp, err := s.authRequest("GET", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Flavors, HasLen, 0)
	flavors = []nova.FlavorDetail{
		nova.FlavorDetail{Id: "fl1", Name: "flavor 1"},
		nova.FlavorDetail{Id: "fl2", Name: "flavor 2"},
	}
	for i, flavor := range flavors {
		s.service.buildFlavorLinks(&flavor)
		flavors[i] = flavor
		err := s.service.addFlavor(flavor)
		defer s.service.removeFlavor(flavor.Id)
		c.Assert(err, IsNil)
	}
	resp, err = s.authRequest("GET", "/flavors/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.Flavors[0].Id != flavors[0].Id {
		expected.Flavors[0], expected.Flavors[1] = expected.Flavors[1], expected.Flavors[0]
	}
	c.Assert(expected.Flavors, DeepEquals, flavors)
	resp, err = s.authRequest("GET", "/flavors/detail/fl1", nil, nil)
	c.Assert(err, IsNil)
	assertBody(c, resp, notFoundResponse)
}

func (s *NovaHTTPSuite) TestGetServers(c *C) {
	entities := s.service.allServersAsEntities()
	c.Assert(entities, HasLen, 0)
	var expected struct {
		Servers []nova.Entity
	}
	resp, err := s.authRequest("GET", "/servers", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, HasLen, 0)
	servers := []nova.ServerDetail{
		nova.ServerDetail{Id: "sr1", Name: "server 1"},
		nova.ServerDetail{Id: "sr2", Name: "server 2"},
	}
	for i, server := range servers {
		s.service.buildServerLinks(&server)
		servers[i] = server
		err := s.service.addServer(server)
		defer s.service.removeServer(server.Id)
		c.Assert(err, IsNil)
	}
	entities = s.service.allServersAsEntities()
	resp, err = s.authRequest("GET", "/servers", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.Servers[0].Id != entities[0].Id {
		expected.Servers[0], expected.Servers[1] = expected.Servers[1], expected.Servers[0]
	}
	c.Assert(expected.Servers, DeepEquals, entities)
	var expectedServer struct {
		Server nova.ServerDetail
	}
	resp, err = s.authRequest("GET", "/servers/sr1", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expectedServer)
	c.Assert(expectedServer.Server, DeepEquals, servers[0])
}

func (s *NovaHTTPSuite) TestDeleteServer(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	_, err := s.service.server(server.Id)
	c.Assert(err, NotNil)
	err = s.service.addServer(server)
	defer s.service.removeServer(server.Id)
	c.Assert(err, IsNil)
	resp, err := s.authRequest("DELETE", "/servers/sr1", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)
	_, err = s.service.server(server.Id)
	c.Assert(err, NotNil)
}

func (s *NovaHTTPSuite) TestGetServersDetail(c *C) {
	servers := s.service.allServers()
	c.Assert(servers, HasLen, 0)
	var expected struct {
		Servers []nova.ServerDetail `json:"servers"`
	}
	resp, err := s.authRequest("GET", "/servers/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Servers, HasLen, 0)
	servers = []nova.ServerDetail{
		nova.ServerDetail{Id: "sr1", Name: "server 1"},
		nova.ServerDetail{Id: "sr2", Name: "server 2"},
	}
	for i, server := range servers {
		s.service.buildServerLinks(&server)
		servers[i] = server
		err := s.service.addServer(server)
		defer s.service.removeServer(server.Id)
		c.Assert(err, IsNil)
	}
	resp, err = s.authRequest("GET", "/servers/detail", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.Servers[0].Id != servers[0].Id {
		expected.Servers[0], expected.Servers[1] = expected.Servers[1], expected.Servers[0]
	}
	c.Assert(expected.Servers, DeepEquals, servers)
	resp, err = s.authRequest("GET", "/servers/detail/sr1", nil, nil)
	c.Assert(err, IsNil)
	assertBody(c, resp, notFoundResponse)
}

func (s *NovaHTTPSuite) TestGetSecurityGroups(c *C) {
	groups := s.service.allSecurityGroups()
	c.Assert(groups, HasLen, 0)
	var expected struct {
		Groups []nova.SecurityGroup `json:"security_groups"`
	}
	resp, err := s.authRequest("GET", "/os-security-groups", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Groups, HasLen, 0)
	groups = []nova.SecurityGroup{
		nova.SecurityGroup{Id: 1, Name: "group 1"},
		nova.SecurityGroup{Id: 2, Name: "group 2"},
	}
	for _, group := range groups {
		err := s.service.addSecurityGroup(group)
		defer s.service.removeSecurityGroup(group.Id)
		c.Assert(err, IsNil)
	}
	resp, err = s.authRequest("GET", "/os-security-groups", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.Groups[0].Id != groups[0].Id {
		expected.Groups[0], expected.Groups[1] = expected.Groups[1], expected.Groups[0]
	}
	c.Assert(expected.Groups, DeepEquals, groups)
	var expectedGroup struct {
		Group nova.SecurityGroup `json:"security_group"`
	}
	resp, err = s.authRequest("GET", "/os-security-groups/1", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expectedGroup)
	c.Assert(expectedGroup.Group, DeepEquals, groups[0])
}

func (s *NovaHTTPSuite) TestAddSecurityGroup(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "group 1", Description: "desc"}
	_, err := s.service.securityGroup(group.Id)
	c.Assert(err, NotNil)
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
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Group, DeepEquals, group)
	err = s.service.removeSecurityGroup(group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaHTTPSuite) TestDeleteSecurityGroup(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "group 1"}
	_, err := s.service.securityGroup(group.Id)
	c.Assert(err, NotNil)
	err = s.service.addSecurityGroup(group)
	defer s.service.removeSecurityGroup(group.Id)
	c.Assert(err, IsNil)
	resp, err := s.authRequest("DELETE", "/os-security-groups/1", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)
	_, err = s.service.securityGroup(group.Id)
	c.Assert(err, NotNil)
}

func (s *NovaHTTPSuite) TestAddSecurityGroupRule(c *C) {
	group1 := nova.SecurityGroup{Id: 1, Name: "src", TenantId: "joe"}
	group2 := nova.SecurityGroup{Id: 2, Name: "tgt"}
	err := s.service.addSecurityGroup(group1)
	defer s.service.removeSecurityGroup(group1.Id)
	c.Assert(err, IsNil)
	err = s.service.addSecurityGroup(group2)
	defer s.service.removeSecurityGroup(group2.Id)
	c.Assert(err, IsNil)
	riIngress := nova.RuleInfo{
		ParentGroupId: 1,
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
		Id:            1,
		ParentGroupId: group1.Id,
		FromPort:      &riIngress.FromPort,
		ToPort:        &riIngress.ToPort,
		IPProtocol:    &riIngress.IPProtocol,
		IPRange:       iprange,
	}
	rule2 := nova.SecurityGroupRule{
		Id:            2,
		ParentGroupId: group2.Id,
		Group: &nova.SecurityGroupRef{
			Name:     group1.Name,
			TenantId: group1.TenantId,
		},
	}
	ok := s.service.hasSecurityGroupRule(group1.Id, rule1.Id)
	c.Assert(ok, Equals, false)
	ok = s.service.hasSecurityGroupRule(group2.Id, rule2.Id)
	c.Assert(ok, Equals, false)
	var req struct {
		Rule nova.RuleInfo `json:"security_group_rule"`
	}
	req.Rule = riIngress
	var expected struct {
		Rule nova.SecurityGroupRule `json:"security_group_rule"`
	}
	resp, err := s.jsonRequest("POST", "/os-security-group-rules", req, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, Equals, rule1.Id)
	c.Assert(expected.Rule.ParentGroupId, Equals, rule1.ParentGroupId)
	c.Assert(expected.Rule.Group, IsNil)
	c.Assert(*expected.Rule.FromPort, Equals, *rule1.FromPort)
	c.Assert(*expected.Rule.ToPort, Equals, *rule1.ToPort)
	c.Assert(*expected.Rule.IPProtocol, Equals, *rule1.IPProtocol)
	c.Assert(expected.Rule.IPRange, DeepEquals, rule1.IPRange)
	defer s.service.removeSecurityGroupRule(rule1.Id)
	c.Assert(err, IsNil)
	req.Rule = riGroup
	resp, err = s.jsonRequest("POST", "/os-security-group-rules", req, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, Equals, rule2.Id)
	c.Assert(expected.Rule.ParentGroupId, Equals, rule2.ParentGroupId)
	c.Assert(*expected.Rule.Group, DeepEquals, *rule2.Group)
	err = s.service.removeSecurityGroupRule(rule2.Id)
	c.Assert(err, IsNil)
}

func (s *NovaHTTPSuite) TestDeleteSecurityGroupRule(c *C) {
	group1 := nova.SecurityGroup{Id: 1, Name: "src", TenantId: "joe"}
	group2 := nova.SecurityGroup{Id: 2, Name: "tgt"}
	err := s.service.addSecurityGroup(group1)
	defer s.service.removeSecurityGroup(group1.Id)
	c.Assert(err, IsNil)
	err = s.service.addSecurityGroup(group2)
	defer s.service.removeSecurityGroup(group2.Id)
	c.Assert(err, IsNil)
	riGroup := nova.RuleInfo{
		ParentGroupId: group2.Id,
		GroupId:       &group1.Id,
	}
	rule := nova.SecurityGroupRule{
		Id:            1,
		ParentGroupId: group2.Id,
		Group: &nova.SecurityGroupRef{
			Name:     group1.Name,
			TenantId: group1.TenantId,
		},
	}
	err = s.service.addSecurityGroupRule(rule.Id, riGroup)
	c.Assert(err, IsNil)
	resp, err := s.authRequest("DELETE", "/os-security-group-rules/1", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)
	ok := s.service.hasSecurityGroupRule(group2.Id, rule.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaHTTPSuite) TestAddServerSecurityGroup(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "group"}
	err := s.service.addSecurityGroup(group)
	defer s.service.removeSecurityGroup(group.Id)
	c.Assert(err, IsNil)
	server := nova.ServerDetail{Id: "sr1"}
	err = s.service.addServer(server)
	defer s.service.removeServer(server.Id)
	c.Assert(err, IsNil)
	ok := s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	var req struct {
		Group struct {
			Name string `json:"name"`
		} `json:"addSecurityGroup"`
	}
	req.Group.Name = group.Name
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, IsNil)
	assertBody(c, resp, noContentResponse)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, true)
	err = s.service.removeServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
}

func (s *NovaHTTPSuite) TestGetServerSecurityGroups(c *C) {
	server := nova.ServerDetail{Id: "sr1"}
	groups := []nova.SecurityGroup{
		nova.SecurityGroup{Id: 1, Name: "group1"},
		nova.SecurityGroup{Id: 2, Name: "group2"},
	}
	srvGroups := s.service.allServerSecurityGroups(server.Id)
	c.Assert(srvGroups, HasLen, 0)
	err := s.service.addServer(server)
	defer s.service.removeServer(server.Id)
	c.Assert(err, IsNil)
	for _, group := range groups {
		err = s.service.addSecurityGroup(group)
		defer s.service.removeSecurityGroup(group.Id)
		c.Assert(err, IsNil)
		err = s.service.addServerSecurityGroup(server.Id, group.Id)
		defer s.service.removeServerSecurityGroup(server.Id, group.Id)
		c.Assert(err, IsNil)
	}
	srvGroups = s.service.allServerSecurityGroups(server.Id)
	var expected struct {
		Groups []nova.SecurityGroup `json:"security_groups"`
	}
	resp, err := s.authRequest("GET", "/servers/"+server.Id+"/os-security-groups", nil, nil)
	c.Assert(err, IsNil)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Groups, DeepEquals, groups)
}

func (s *NovaHTTPSuite) TestDeleteServerSecurityGroup(c *C) {
	group := nova.SecurityGroup{Id: 1, Name: "group"}
	err := s.service.addSecurityGroup(group)
	defer s.service.removeSecurityGroup(group.Id)
	c.Assert(err, IsNil)
	server := nova.ServerDetail{Id: "sr1"}
	err = s.service.addServer(server)
	defer s.service.removeServer(server.Id)
	c.Assert(err, IsNil)
	ok := s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
	err = s.service.addServerSecurityGroup(server.Id, group.Id)
	c.Assert(err, IsNil)
	var req struct {
		Group struct {
			Name string `json:"name"`
		} `json:"removeSecurityGroup"`
	}
	req.Group.Name = group.Name
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, IsNil)
	assertBody(c, resp, noContentResponse)
	ok = s.service.hasServerSecurityGroup(server.Id, group.Id)
	c.Assert(ok, Equals, false)
}

func (s *NovaHTTPSuite) TestPostFloatingIP(c *C) {
	fip := nova.FloatingIP{Id: 1, IP: "10.0.0.1", Pool: "nova"}
	c.Assert(s.service.allFloatingIPs(), HasLen, 0)
	var expected struct {
		IP nova.FloatingIP `json:"floating_ip"`
	}
	resp, err := s.authRequest("POST", "/os-floating-ips", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.IP, DeepEquals, fip)
	err = s.service.removeFloatingIP(fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaHTTPSuite) TestGetFloatingIPs(c *C) {
	c.Assert(s.service.allFloatingIPs(), HasLen, 0)
	var expected struct {
		IPs []nova.FloatingIP `json:"floating_ips"`
	}
	resp, err := s.authRequest("GET", "/os-floating-ips", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.IPs, HasLen, 0)
	fips := []nova.FloatingIP{
		nova.FloatingIP{Id: 1, IP: "1.2.3.4", Pool: "nova"},
		nova.FloatingIP{Id: 2, IP: "4.3.2.1", Pool: "nova"},
	}
	for _, fip := range fips {
		err := s.service.addFloatingIP(fip)
		defer s.service.removeFloatingIP(fip.Id)
		c.Assert(err, IsNil)
	}
	resp, err = s.authRequest("GET", "/os-floating-ips", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.IPs[0].Id != fips[0].Id {
		expected.IPs[0], expected.IPs[1] = expected.IPs[1], expected.IPs[0]
	}
	c.Assert(expected.IPs, DeepEquals, fips)
	var expectedIP struct {
		IP nova.FloatingIP `json:"floating_ip"`
	}
	resp, err = s.authRequest("GET", "/os-floating-ips/1", nil, nil)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	assertJSON(c, resp, &expectedIP)
	c.Assert(expectedIP.IP, DeepEquals, fips[0])
}

func (s *NovaHTTPSuite) TestDeleteFloatingIP(c *C) {
	fip := nova.FloatingIP{Id: 1, IP: "10.0.0.1", Pool: "nova"}
	err := s.service.addFloatingIP(fip)
	defer s.service.removeFloatingIP(fip.Id)
	c.Assert(err, IsNil)
	resp, err := s.authRequest("DELETE", "/os-floating-ips/1", nil, nil)
	c.Assert(err, IsNil)
	assertBody(c, resp, acceptedResponse)
	_, err = s.service.floatingIP(fip.Id)
	c.Assert(err, NotNil)
}

func (s *NovaHTTPSuite) TestAddServerFloatingIP(c *C) {
	fip := nova.FloatingIP{Id: 1, IP: "1.2.3.4"}
	server := nova.ServerDetail{Id: "sr1"}
	err := s.service.addFloatingIP(fip)
	defer s.service.removeFloatingIP(fip.Id)
	c.Assert(err, IsNil)
	err = s.service.addServer(server)
	defer s.service.removeServer(server.Id)
	c.Assert(err, IsNil)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), Equals, false)
	var req struct {
		AddFloatingIP struct {
			Address string `json:"address"`
		} `json:"addFloatingIp"`
	}
	req.AddFloatingIP.Address = fip.IP
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, IsNil)
	assertBody(c, resp, noContentResponse)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), Equals, true)
	err = s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
}

func (s *NovaHTTPSuite) TestRemoveServerFloatingIP(c *C) {
	fip := nova.FloatingIP{Id: 1, IP: "1.2.3.4"}
	server := nova.ServerDetail{Id: "sr1"}
	err := s.service.addFloatingIP(fip)
	defer s.service.removeFloatingIP(fip.Id)
	c.Assert(err, IsNil)
	err = s.service.addServer(server)
	defer s.service.removeServer(server.Id)
	c.Assert(err, IsNil)
	err = s.service.addServerFloatingIP(server.Id, fip.Id)
	defer s.service.removeServerFloatingIP(server.Id, fip.Id)
	c.Assert(err, IsNil)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), Equals, true)
	var req struct {
		RemoveFloatingIP struct {
			Address string `json:"address"`
		} `json:"removeFloatingIp"`
	}
	req.RemoveFloatingIP.Address = fip.IP
	resp, err := s.jsonRequest("POST", "/servers/"+server.Id+"/action", req, nil)
	c.Assert(err, IsNil)
	assertBody(c, resp, noContentResponse)
	c.Assert(s.service.hasServerFloatingIP(server.Id, fip.IP), Equals, false)
}
