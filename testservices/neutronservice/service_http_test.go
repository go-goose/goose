// Neutron double testing service - HTTP API tests

package neutronservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v2/neutron"
	"gopkg.in/goose.v2/testing/httpsuite"
	"gopkg.in/goose.v2/testservices/identityservice"
	"gopkg.in/goose.v2/testservices/neutronmodel"
)

type NeutronHTTPSuite struct {
	httpsuite.HTTPSuite
	service *Neutron
	token   string
}

var _ = gc.Suite(&NeutronHTTPSuite{})

type NeutronHTTPSSuite struct {
	httpsuite.HTTPSuite
	service *Neutron
	token   string
}

var _ = gc.Suite(&NeutronHTTPSSuite{HTTPSuite: httpsuite.HTTPSuite{UseTLS: true}})

func (s *NeutronHTTPSuite) SetUpSuite(c *gc.C) {
	s.HTTPSuite.SetUpSuite(c)
	identityDouble := identityservice.NewUserPass()
	userInfo := identityDouble.AddUser("fred", "secret", "tenant", "default")
	s.token = userInfo.Token
	s.service = New(s.Server.URL, versionPath, userInfo.TenantId, region, identityDouble, nil)
	s.service.AddNeutronModel(neutronmodel.New())
}

func (s *NeutronHTTPSuite) TearDownSuite(c *gc.C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *NeutronHTTPSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
	// this is otherwise handled not directly by neutron test service
	// but by openstack that tries for / before.
	s.Mux.Handle("/", s.service.handler((*Neutron).handleRoot))
}

func (s *NeutronHTTPSuite) TearDownTest(c *gc.C) {
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
func (s *NeutronHTTPSuite) sendRequest(method, url string, body []byte, headers http.Header) (*http.Response, error) {
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
func (s *NeutronHTTPSuite) authRequest(method, path string, body []byte, headers http.Header) (*http.Response, error) {
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set(authToken, s.token)
	url := s.service.endpointURL(true, path)
	return s.sendRequest(method, url, body, headers)
}

// jsonRequest serializes the passed body object to JSON and sends a
// the request with authRequest().
func (s *NeutronHTTPSuite) jsonRequest(method, path string, body interface{}, headers http.Header) (*http.Response, error) {
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

func (s *NeutronHTTPSuite) simpleTests() []SimpleTest {
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
			unauth:  true,
			method:  "GET",
			url:     versionPath + "/phony_token",
			headers: setHeader(authToken, s.token),
			expect:  errBadRequestMalformedURL,
		},

		{
			method: "GET",
			url:    neutron.ApiSecurityGroupsV2 + "/42",
			expect: errNotFoundJSONSG,
		},
		{
			method: "POST",
			url:    neutron.ApiSecurityGroupsV2,
			expect: errBadRequestIncorrect,
		},
		{
			method: "POST",
			url:    neutron.ApiSecurityGroupsV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiSecurityGroupsV2,
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiSecurityGroupsV2,
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiSecurityGroupsV2 + "/42",
			expect: errNotFoundJSONSG,
		},

		{
			method: "GET",
			url:    neutron.ApiSecurityGroupRulesV2,
			expect: errNotFoundJSON,
		},
		{
			method: "GET",
			url:    neutron.ApiSecurityGroupRulesV2 + "/invalid",
			expect: errNotFoundJSON,
		},
		{
			method: "GET",
			url:    neutron.ApiSecurityGroupRulesV2 + "/42",
			expect: errNotFoundJSON,
		},
		{
			method: "POST",
			url:    neutron.ApiSecurityGroupRulesV2,
			expect: errBadRequestIncorrect,
		},
		{
			method: "POST",
			url:    neutron.ApiSecurityGroupRulesV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiSecurityGroupRulesV2,
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiSecurityGroupRulesV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiSecurityGroupRulesV2,
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiSecurityGroupRulesV2 + "/42",
			expect: errNotFoundJSONSGR,
		},

		{
			method: "GET",
			url:    neutron.ApiFloatingIPsV2 + "/42",
			expect: errNotFoundJSON,
		},
		{
			method: "POST",
			url:    neutron.ApiFloatingIPsV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiFloatingIPsV2,
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiFloatingIPsV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiFloatingIPsV2,
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiFloatingIPsV2 + "/invalid",
			expect: errNotFoundJSON,
		},
		{
			method: "GET",
			url:    neutron.ApiNetworksV2 + "/42",
			expect: errNotFoundJSON,
		},
		{
			method: "POST",
			url:    neutron.ApiNetworksV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiNetworksV2,
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiNetworksV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiNetworksV2,
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiNetworksV2 + "/invalid",
			expect: errNotFound,
		},

		{
			method: "GET",
			url:    neutron.ApiSubnetsV2 + "/42",
			expect: errNotFoundJSON,
		},
		{
			method: "POST",
			url:    neutron.ApiSubnetsV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiSubnetsV2,
			expect: errNotFound,
		},
		{
			method: "PUT",
			url:    neutron.ApiSubnetsV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiSubnetsV2,
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiSubnetsV2 + "/invalid",
			expect: errNotFound,
		},

		{
			method: "GET",
			url:    neutron.ApiPortsV2 + "/42",
			expect: errNotFoundJSONP,
		},
		{
			method: "POST",
			url:    neutron.ApiPortsV2,
			expect: errBadRequestIncorrect,
		},
		{
			method: "POST",
			url:    neutron.ApiPortsV2 + "/invalid",
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiPortsV2,
			expect: errNotFound,
		},
		{
			method: "DELETE",
			url:    neutron.ApiPortsV2 + "/42",
			expect: errNotFoundJSONP,
		},
	}
	return simpleTests
}

func (s *NeutronHTTPSuite) TestSimpleRequestTests(c *gc.C) {
	simpleTests := s.simpleTests()
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

func (s *NeutronHTTPSuite) TestNewUUID(c *gc.C) {
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

func (s *NeutronHTTPSuite) TestGetSecurityGroups(c *gc.C) {
	// There is always a default security group.
	groups := s.service.allSecurityGroups()
	c.Assert(groups, gc.HasLen, 1)
	var expected struct {
		Groups []neutron.SecurityGroupV2 `json:"security_groups"`
	}
	resp, err := s.authRequest("GET", neutron.ApiSecurityGroupsV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Groups, gc.HasLen, 1)
	groups = []neutron.SecurityGroupV2{
		{
			Id:       "1",
			Name:     "group 1",
			TenantId: s.service.TenantId,
			Rules:    []neutron.SecurityGroupRuleV2{},
		},
		{
			Id:       "2",
			Name:     "group 2",
			TenantId: s.service.TenantId,
			Rules:    []neutron.SecurityGroupRuleV2{},
		},
	}
	for _, group := range groups {
		err := s.service.addSecurityGroup(group)
		c.Assert(err, gc.IsNil)
		defer s.service.removeSecurityGroup(group.Id)
	}
	groups[0].Rules = defaultSecurityGroupRules(groups[0].Id, groups[0].TenantId)
	groups[1].Rules = defaultSecurityGroupRules(groups[1].Id, groups[1].TenantId)
	resp, err = s.authRequest("GET", neutron.ApiSecurityGroupsV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Groups, gc.HasLen, len(groups)+1)
	checkGroupsInList(c, groups, expected.Groups)
	var expectedGroup struct {
		Group neutron.SecurityGroupV2 `json:"security_group"`
	}
	url := fmt.Sprintf("%s/%s", neutron.ApiSecurityGroupsV2, "1")
	resp, err = s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedGroup)
	c.Assert(expectedGroup.Group, gc.DeepEquals, groups[0])
}

func defaultSecurityGroupRules(groupId, tenantId string) []neutron.SecurityGroupRuleV2 {
	id, _ := strconv.Atoi(groupId)
	id1 := id * 999
	id2 := id * 998
	return []neutron.SecurityGroupRuleV2{
		{
			Direction:     "egress",
			EthernetType:  "IPv4",
			Id:            strconv.Itoa(id1),
			TenantId:      tenantId,
			ParentGroupId: groupId,
		},
		{
			Direction:     "egress",
			EthernetType:  "IPv6",
			Id:            strconv.Itoa(id2),
			TenantId:      tenantId,
			ParentGroupId: groupId,
		},
	}
}

func (s *NeutronHTTPSuite) TestAddSecurityGroup(c *gc.C) {
	group := neutron.SecurityGroupV2{
		Id:          "1",
		Name:        "group 1",
		Description: "desc",
		TenantId:    s.service.TenantId,
		Rules:       []neutron.SecurityGroupRuleV2{},
	}
	_, err := s.service.securityGroup(group.Id)
	c.Assert(err, gc.NotNil)
	group.Rules = defaultSecurityGroupRules(group.Id, group.TenantId)
	var req struct {
		Group struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"security_group"`
	}
	req.Group.Name = group.Name
	req.Group.Description = group.Description
	var expected struct {
		Group neutron.SecurityGroupV2 `json:"security_group"`
	}
	resp, err := s.jsonRequest("POST", neutron.ApiSecurityGroupsV2, req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusCreated)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Group, gc.DeepEquals, group)
	err = s.service.removeSecurityGroup(group.Id)
	c.Assert(err, gc.IsNil)
}

func (s *NeutronHTTPSuite) TestDeleteSecurityGroup(c *gc.C) {
	group := neutron.SecurityGroupV2{Id: "1", Name: "group 1", TenantId: s.service.TenantId}
	_, err := s.service.securityGroup(group.Id)
	c.Assert(err, gc.NotNil)
	err = s.service.addSecurityGroup(group)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group.Id)
	url := fmt.Sprintf("%s/%s", neutron.ApiSecurityGroupsV2, "1")
	resp, err := s.authRequest("DELETE", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNoContent)
	_, err = s.service.securityGroup(group.Id)
	c.Assert(err, gc.NotNil)
}

func (s *NeutronHTTPSuite) TestAddSecurityGroupRule(c *gc.C) {
	group1 := neutron.SecurityGroupV2{Id: "1", Name: "src", TenantId: s.service.TenantId}
	group2 := neutron.SecurityGroupV2{Id: "2", Name: "tgt", TenantId: s.service.TenantId}
	err := s.service.addSecurityGroup(group1)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group1.Id)
	err = s.service.addSecurityGroup(group2)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group2.Id)
	riIngress := neutron.RuleInfoV2{
		ParentGroupId:  "1",
		Direction:      "ingress",
		PortRangeMax:   22,
		PortRangeMin:   22,
		IPProtocol:     "tcp",
		RemoteIPPrefix: "1.2.3.4/5",
	}
	riIngress2 := neutron.RuleInfoV2{
		ParentGroupId:  "1",
		Direction:      "ingress",
		PortRangeMax:   22,
		PortRangeMin:   22,
		IPProtocol:     "tcp",
		RemoteIPPrefix: "2.3.4.5/6",
	}
	riEgress := neutron.RuleInfoV2{
		ParentGroupId:  group2.Id,
		Direction:      "egress",
		PortRangeMax:   22,
		PortRangeMin:   22,
		IPProtocol:     "tcp",
		RemoteIPPrefix: "5.4.3.2/1",
	}
	riIngress6 := neutron.RuleInfoV2{
		ParentGroupId:  "1",
		Direction:      "ingress",
		PortRangeMax:   22,
		PortRangeMin:   22,
		IPProtocol:     "tcp",
		RemoteIPPrefix: "2001:db8:42::/64",
		EthernetType:   "IPv6",
	}
	rule1 := neutron.SecurityGroupRuleV2{
		Id:             "1",
		ParentGroupId:  group1.Id,
		Direction:      riIngress.Direction,
		PortRangeMax:   &riIngress.PortRangeMax,
		PortRangeMin:   &riIngress.PortRangeMin,
		IPProtocol:     &riIngress.IPProtocol,
		RemoteIPPrefix: riIngress.RemoteIPPrefix,
	}
	rule2 := neutron.SecurityGroupRuleV2{
		Id:             "2",
		ParentGroupId:  group1.Id,
		Direction:      riIngress2.Direction,
		PortRangeMax:   &riIngress2.PortRangeMax,
		PortRangeMin:   &riIngress2.PortRangeMin,
		IPProtocol:     &riIngress2.IPProtocol,
		RemoteIPPrefix: riIngress2.RemoteIPPrefix,
	}
	rule3 := neutron.SecurityGroupRuleV2{
		Id:            "3",
		ParentGroupId: group2.Id,
		Direction:     riEgress.Direction,
		PortRangeMax:  &riEgress.PortRangeMax,
		PortRangeMin:  &riEgress.PortRangeMin,
		IPProtocol:    &riEgress.IPProtocol,
	}
	rule6 := neutron.SecurityGroupRuleV2{
		Id:             "5",
		ParentGroupId:  group1.Id,
		Direction:      riIngress6.Direction,
		PortRangeMax:   &riIngress6.PortRangeMax,
		PortRangeMin:   &riIngress6.PortRangeMin,
		IPProtocol:     &riIngress6.IPProtocol,
		RemoteIPPrefix: riIngress6.RemoteIPPrefix,
	}
	ok := s.service.hasSecurityGroupRule(group1.Id, rule1.Id)
	c.Assert(ok, gc.Equals, false)
	ok = s.service.hasSecurityGroupRule(group2.Id, rule2.Id)
	c.Assert(ok, gc.Equals, false)
	var req struct {
		Rule neutron.RuleInfoV2 `json:"security_group_rule"`
	}
	req.Rule = riIngress
	var expected struct {
		Rule neutron.SecurityGroupRuleV2 `json:"security_group_rule"`
	}
	resp, err := s.jsonRequest("POST", neutron.ApiSecurityGroupRulesV2, req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusCreated)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, gc.Equals, rule1.Id)
	c.Assert(expected.Rule.ParentGroupId, gc.Equals, rule1.ParentGroupId)
	c.Assert(*expected.Rule.PortRangeMax, gc.Equals, *rule1.PortRangeMax)
	c.Assert(*expected.Rule.PortRangeMin, gc.Equals, *rule1.PortRangeMin)
	c.Assert(*expected.Rule.IPProtocol, gc.Equals, *rule1.IPProtocol)
	c.Assert(expected.Rule.Direction, gc.Equals, rule1.Direction)
	c.Assert(expected.Rule.RemoteIPPrefix, gc.Equals, rule1.RemoteIPPrefix)
	// Attempt to create duplicate rule should fail
	resp, err = s.jsonRequest("POST", neutron.ApiSecurityGroupRulesV2, req, nil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusBadRequest)
	err = s.service.removeSecurityGroupRule(rule1.Id)
	c.Assert(err, gc.IsNil)
	// Attempt to create rule with all fields but RemoteIPPrefix identical should pass
	req.Rule = riIngress2
	resp, err = s.jsonRequest("POST", neutron.ApiSecurityGroupRulesV2, req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusCreated)
	err = s.service.removeSecurityGroupRule(rule2.Id)
	c.Assert(err, gc.IsNil)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, gc.Equals, rule2.Id)
	c.Assert(expected.Rule.ParentGroupId, gc.Equals, rule2.ParentGroupId)
	c.Assert(*expected.Rule.PortRangeMax, gc.Equals, *rule2.PortRangeMax)
	c.Assert(*expected.Rule.PortRangeMin, gc.Equals, *rule2.PortRangeMin)
	c.Assert(*expected.Rule.IPProtocol, gc.Equals, *rule2.IPProtocol)
	c.Assert(expected.Rule.Direction, gc.Equals, rule2.Direction)
	c.Assert(expected.Rule.RemoteIPPrefix, gc.Equals, rule2.RemoteIPPrefix)
	req.Rule = riEgress
	resp, err = s.jsonRequest("POST", neutron.ApiSecurityGroupRulesV2, req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusCreated)
	err = s.service.removeSecurityGroupRule(rule3.Id)
	c.Assert(err, gc.IsNil)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, gc.Equals, rule3.Id)
	c.Assert(expected.Rule.ParentGroupId, gc.Equals, rule3.ParentGroupId)
	// Attempt to create rule with IPv6 RemoteIPPrefix without specifying EthernetType, should fail
	req.Rule = riIngress6
	req.Rule.EthernetType = ""
	resp, err = s.jsonRequest("POST", neutron.ApiSecurityGroupRulesV2, req, nil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusBadRequest)
	// Attempt to create rule with IPv6 RemoteIPPrefix with correct EthernetType, should pass
	req.Rule = riIngress6
	resp, err = s.jsonRequest("POST", neutron.ApiSecurityGroupRulesV2, req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusCreated)
	err = s.service.removeSecurityGroupRule(rule6.Id)
	c.Assert(err, gc.IsNil)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Rule.Id, gc.Equals, rule6.Id)
	c.Assert(expected.Rule.ParentGroupId, gc.Equals, rule6.ParentGroupId)
}

func (s *NeutronHTTPSuite) TestDeleteSecurityGroupRule(c *gc.C) {
	group1 := neutron.SecurityGroupV2{Id: "1", Name: "src", TenantId: s.service.TenantId}
	group2 := neutron.SecurityGroupV2{Id: "2", Name: "tgt", TenantId: s.service.TenantId}
	err := s.service.addSecurityGroup(group1)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group1.Id)
	err = s.service.addSecurityGroup(group2)
	c.Assert(err, gc.IsNil)
	defer s.service.removeSecurityGroup(group2.Id)
	riGroup := neutron.RuleInfoV2{
		ParentGroupId: group2.Id,
		Direction:     "egress",
	}
	rule := neutron.SecurityGroupRuleV2{
		Id:            "1",
		ParentGroupId: group2.Id,
		Direction:     "egress",
	}
	err = s.service.addSecurityGroupRule(rule.Id, riGroup)
	c.Assert(err, gc.IsNil)
	url := fmt.Sprintf("%s/%s", neutron.ApiSecurityGroupRulesV2, "1")
	resp, err := s.authRequest("DELETE", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNoContent)
	ok := s.service.hasSecurityGroupRule(group2.Id, rule.Id)
	c.Assert(ok, gc.Equals, false)
}

func (s *NeutronHTTPSuite) TestPostFloatingIPV2(c *gc.C) {
	// network 998 has External = true
	fip := neutron.FloatingIPV2{Id: "1", IP: "10.0.0.1", FloatingNetworkId: "998"}
	c.Assert(s.service.allFloatingIPs(nil), gc.HasLen, 0)
	var req struct {
		IP neutron.FloatingIPV2 `json:"floatingip"`
	}
	req.IP = fip
	resp, err := s.jsonRequest("POST", neutron.ApiFloatingIPsV2, req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusCreated)
	var expected struct {
		IP neutron.FloatingIPV2 `json:"floatingip"`
	}
	assertJSON(c, resp, &expected)
	c.Assert(expected.IP, gc.DeepEquals, fip)
	err = s.service.removeFloatingIP(fip.Id)
	c.Assert(err, gc.IsNil)
	// network 999 has External = false
	req.IP.FloatingNetworkId = "999"
	resp, err = s.jsonRequest("POST", neutron.ApiFloatingIPsV2, req, nil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNotFound)
}

func (s *NeutronHTTPSuite) TestGetFloatingIPs(c *gc.C) {
	c.Assert(s.service.allFloatingIPs(nil), gc.HasLen, 0)
	var expected struct {
		IPs []neutron.FloatingIPV2 `json:"floatingips"`
	}
	resp, err := s.authRequest("GET", neutron.ApiFloatingIPsV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.IPs, gc.HasLen, 0)
	fips := []neutron.FloatingIPV2{
		{Id: "1", IP: "1.2.3.4"},
		{Id: "2", IP: "4.3.2.1"},
	}
	for _, fip := range fips {
		err := s.service.addFloatingIP(fip)
		defer s.service.removeFloatingIP(fip.Id)
		c.Assert(err, gc.IsNil)
	}
	resp, err = s.authRequest("GET", neutron.ApiFloatingIPsV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	if expected.IPs[0].Id != fips[0].Id {
		expected.IPs[0], expected.IPs[1] = expected.IPs[1], expected.IPs[0]
	}
	c.Assert(expected.IPs, gc.DeepEquals, fips)
	var expectedIP struct {
		IP neutron.FloatingIPV2 `json:"floatingip"`
	}
	url := fmt.Sprintf("%s/%s", neutron.ApiFloatingIPsV2, "1")
	resp, err = s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedIP)
	c.Assert(expectedIP.IP, gc.DeepEquals, fips[0])
}

func (s *NeutronHTTPSuite) TestDeleteFloatingIP(c *gc.C) {
	fip := neutron.FloatingIPV2{Id: "1", IP: "10.0.0.1"}
	err := s.service.addFloatingIP(fip)
	c.Assert(err, gc.IsNil)
	defer s.service.removeFloatingIP(fip.Id)
	url := fmt.Sprintf("%s/%s", neutron.ApiFloatingIPsV2, "1")
	resp, err := s.authRequest("DELETE", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNoContent)
	_, err = s.service.floatingIP(fip.Id)
	c.Assert(err, gc.NotNil)
}

func (s *NeutronHTTPSuite) TestGetNetworks(c *gc.C) {
	// There are always 4 networks
	networks, err := s.service.allNetworks(nil)
	c.Assert(err, gc.IsNil)
	c.Assert(networks, gc.HasLen, 5)
	var expected struct {
		Networks []neutron.NetworkV2 `json:"networks"`
	}
	resp, err := s.authRequest("GET", neutron.ApiNetworksV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Networks, gc.HasLen, len(networks))
	var expectedNetwork struct {
		Network neutron.NetworkV2 `json:"network"`
	}
	url := fmt.Sprintf("%s/%s", neutron.ApiNetworksV2, networks[0].Id)
	resp, err = s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedNetwork)
	c.Assert(expectedNetwork.Network, gc.DeepEquals, networks[0])
}

func (s *NeutronHTTPSuite) TestGetSubnets(c *gc.C) {
	// There are always 3 subnets
	subnets := s.service.allSubnets()
	c.Assert(subnets, gc.HasLen, 3)
	var expected struct {
		Subnets []neutron.SubnetV2 `json:"subnets"`
	}
	resp, err := s.authRequest("GET", neutron.ApiSubnetsV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Subnets, gc.HasLen, 3)
	var expectedSubnet struct {
		Subnet neutron.SubnetV2 `json:"subnet"`
	}
	url := fmt.Sprintf("%s/%s", neutron.ApiSubnetsV2, subnets[0].Id)
	resp, err = s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedSubnet)
	c.Assert(expectedSubnet.Subnet, gc.DeepEquals, subnets[0])
}

func (s *NeutronHTTPSuite) TestGetPorts(c *gc.C) {
	// There is always a default port.
	ports := s.service.allPorts()
	c.Assert(ports, gc.HasLen, 0)
	var expected struct {
		Ports []neutron.PortV2 `json:"ports"`
	}
	resp, err := s.authRequest("GET", neutron.ApiPortsV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Ports, gc.HasLen, 0)

	ports = []neutron.PortV2{
		{
			Id:        "1",
			Name:      "group 1",
			TenantId:  s.service.TenantId,
			NetworkId: "a87cc70a-3e15-4acf-8205-9b711a3531b7",
		},
		{
			Id:        "2",
			Name:      "group 2",
			TenantId:  s.service.TenantId,
			NetworkId: "a87cc70a-3e15-4acf-8205-9b711a3531xx",
		},
	}

	for _, group := range ports {
		err := s.service.addPort(group)
		c.Assert(err, gc.IsNil)
		defer s.service.removePort(group.Id)
	}

	resp, err = s.authRequest("GET", neutron.ApiPortsV2, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Ports, gc.HasLen, len(ports))

	checkPortsInList(c, ports, expected.Ports)

	var expectedPort struct {
		Port neutron.PortV2 `json:"port"`
	}
	url := fmt.Sprintf("%s/%s", neutron.ApiPortsV2, "1")
	resp, err = s.authRequest("GET", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	assertJSON(c, resp, &expectedPort)
	c.Assert(expectedPort.Port, gc.DeepEquals, ports[0])
}

func (s *NeutronHTTPSuite) TestAddPort(c *gc.C) {
	port := neutron.PortV2{
		Id:          "1",
		Name:        "port 1",
		Description: "desc",
		TenantId:    s.service.TenantId,
		NetworkId:   "a87cc70a-3e15-4acf-8205-9b711a3531b7",
	}
	_, err := s.service.port(port.Id)
	c.Assert(err, gc.NotNil)

	var req struct {
		Port struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			NetworkId   string `json:"network_id"`
		} `json:"port"`
	}
	req.Port.Name = port.Name
	req.Port.Description = port.Description
	req.Port.NetworkId = port.NetworkId

	var expected struct {
		Port neutron.PortV2 `json:"port"`
	}
	resp, err := s.jsonRequest("POST", neutron.ApiPortsV2, req, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusCreated)
	assertJSON(c, resp, &expected)
	c.Assert(expected.Port, gc.DeepEquals, port)

	err = s.service.removePort(port.Id)
	c.Assert(err, gc.IsNil)
}

func (s *NeutronHTTPSuite) TestDeletePort(c *gc.C) {
	port := neutron.PortV2{Id: "1", Name: "port 1", TenantId: s.service.TenantId}
	_, err := s.service.port(port.Id)
	c.Assert(err, gc.NotNil)

	err = s.service.addPort(port)
	c.Assert(err, gc.IsNil)
	defer s.service.removePort(port.Id)

	url := fmt.Sprintf("%s/%s", neutron.ApiPortsV2, "1")
	resp, err := s.authRequest("DELETE", url, nil, nil)
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNoContent)

	_, err = s.service.port(port.Id)
	c.Assert(err, gc.NotNil)
}

func (s *NeutronHTTPSSuite) SetUpSuite(c *gc.C) {
	s.HTTPSuite.SetUpSuite(c)
	identityDouble := identityservice.NewUserPass()
	userInfo := identityDouble.AddUser("fred", "secret", "tenant", "default")
	s.token = userInfo.Token
	c.Assert(s.Server.URL[:8], gc.Equals, "https://")
	s.service = New(s.Server.URL, versionPath, userInfo.TenantId, region, identityDouble, nil)
	s.service.AddNeutronModel(neutronmodel.New())
}

func (s *NeutronHTTPSSuite) TearDownSuite(c *gc.C) {
	s.HTTPSuite.TearDownSuite(c)
}

func (s *NeutronHTTPSSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
}

func (s *NeutronHTTPSSuite) TearDownTest(c *gc.C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *NeutronHTTPSSuite) TestHasHTTPSServiceURL(c *gc.C) {
	endpoints := s.service.Endpoints()
	c.Assert(endpoints[0].PublicURL[:8], gc.Equals, "https://")
}
