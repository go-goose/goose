package identity_test

import (
	"net/url"
	"strings"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v2/identity"
	"gopkg.in/goose.v2/testservices/openstackservice"
)

func registerLocalTests(authMode identity.AuthMode) {
	lt := LiveTests{authMode: authMode}
	gc.Suite(&localLiveSuite{LiveTests: lt})
}

// localLiveSuite runs tests from LiveTests using a fake
// nova server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	openstack *openstackservice.Openstack
}

func (s *localLiveSuite) SetUpSuite(c *gc.C) {
	c.Logf("Using identity and nova service test doubles")

	// Set up an Openstack service.
	s.cred = &identity.Credentials{
		User:    "fred",
		Secrets: "secret",
		Region:  "zone1.some region",
	}
	var logMsg []string
	s.openstack, logMsg = openstackservice.New(s.cred, s.authMode, false)
	for _, msg := range logMsg {
		c.Logf(msg)
	}
	s.openstack.SetupHTTP(nil)

	if s.authMode == identity.AuthUserPassV3 {
		s.cred.URL = s.cred.URL + "/v3"
	}

	s.openstack.Identity.AddUser("fred", "secret", "tenant", "default")
	s.LiveTests.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *gc.C) {
	s.LiveTests.TearDownSuite(c)
	s.openstack.Stop()
}

func (s *localLiveSuite) SetUpTest(c *gc.C) {
	s.LiveTests.SetUpTest(c)
}

func (s *localLiveSuite) TearDownTest(c *gc.C) {
	s.LiveTests.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.

func (s *localLiveSuite) TestProductStreamsEndpoint(c *gc.C) {
	err := s.client.Authenticate()
	c.Assert(err, gc.IsNil)

	serviceURL, err := s.client.MakeServiceURL("product-streams", "", nil)
	c.Assert(err, gc.IsNil)

	_, err = url.Parse(serviceURL)
	c.Assert(err, gc.IsNil)
	c.Assert(strings.HasSuffix(serviceURL, "/imagemetadata"), gc.Equals, true)
}

func (s *localLiveSuite) TestJujuToolsEndpoint(c *gc.C) {
	err := s.client.Authenticate()
	c.Assert(err, gc.IsNil)

	serviceURL, err := s.client.MakeServiceURL("juju-tools", "", nil)
	c.Assert(err, gc.IsNil)

	_, err = url.Parse(serviceURL)
	c.Assert(err, gc.IsNil)
}
