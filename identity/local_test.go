package identity_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/identity"
	"gopkg.in/goose.v1/testservices/openstackservice"
)

func registerLocalTests(authMode identity.AuthMode) {
	lt := LiveTests{authMode: authMode}
	gc.Suite(&localLiveSuite{LiveTests: lt})
}

// localLiveSuite runs tests from LiveTests using a fake
// nova server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	// The following attributes are for using testing doubles.
	Server     *httptest.Server
	Mux        *http.ServeMux
	oldHandler http.Handler
}

func (s *localLiveSuite) SetUpSuite(c *gc.C) {
	c.Logf("Using identity and nova service test doubles")

	// Set up the HTTP server.
	s.Server = httptest.NewServer(nil)
	s.oldHandler = s.Server.Config.Handler
	s.Mux = http.NewServeMux()
	s.Server.Config.Handler = s.Mux

	serverURL := s.Server.URL
	if s.authMode == identity.AuthUserPassV3 {
		serverURL = serverURL + "/v3/auth"
	}
	// Set up an Openstack service.
	s.cred = &identity.Credentials{
		URL:        serverURL,
		User:       "fred",
		Secrets:    "secret",
		Region:     "zone1.some region",
		TenantName: "tenant",
		DomainName: "default",
	}
	openstack := openstackservice.New(s.cred, s.authMode)
	openstack.SetupHTTP(s.Mux)

	openstack.Identity.AddUser("fred", "secret", "tenant")
	s.LiveTests.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *gc.C) {
	s.LiveTests.TearDownSuite(c)
	s.Mux = nil
	s.Server.Config.Handler = s.oldHandler
	s.Server.Close()
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
	serviceURL, err := s.client.MakeServiceURL("product-streams", nil)
	c.Assert(err, gc.IsNil)
	_, err = url.Parse(serviceURL)
	c.Assert(err, gc.IsNil)
	c.Assert(strings.HasSuffix(serviceURL, "/imagemetadata"), gc.Equals, true)
}

func (s *localLiveSuite) TestJujuToolsEndpoint(c *gc.C) {
	err := s.client.Authenticate()
	c.Assert(err, gc.IsNil)
	serviceURL, err := s.client.MakeServiceURL("juju-tools", nil)
	c.Assert(err, gc.IsNil)
	_, err = url.Parse(serviceURL)
	c.Assert(err, gc.IsNil)
}
