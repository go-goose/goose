package client_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/openstackservice"
	"net/url"
)

func registerLocalTests(authModes []identity.AuthMode) {
	for _, authMode := range authModes {
		Suite(&localLiveSuite{
			LiveTests: LiveTests{
				authMode: authMode,
			},
		})
	}
}

// localLiveSuite runs tests from LiveTests using a fake
// identity server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	// The following attributes are for using testing doubles.
	httpsuite.HTTPSuite
	service testservices.HttpService
}

func (s *localLiveSuite) SetUpSuite(c *C) {
	c.Logf("Using identity service test double")
	s.HTTPSuite.SetUpSuite(c)
	s.cred = &identity.Credentials{
		URL:        s.Server.URL,
		User:       "fred",
		Secrets:    "secret",
		Region:     "zone1.some region",
		TenantName: "tenant",
	}
	switch s.authMode {
	default:
		panic("Invalid authentication method")
	case identity.AuthUserPass:
		// The openstack test service sets up userpass authentication.
		s.service = openstackservice.New(s.cred)
		// Add an additional endpoint so region filtering can be properly tested.
		serviceDef := identityservice.Service{"nova", "compute", []identityservice.Endpoint{
			identityservice.Endpoint{PublicURL: "http://nova2", Region: "zone2.RegionOne"},
		}}
		s.service.(*openstackservice.Openstack).Identity.(*identityservice.UserPass).AddService(serviceDef)

	case identity.AuthLegacy:
		legacy := identityservice.NewLegacy()
		legacy.AddUser(s.cred.User, s.cred.Secrets, s.cred.TenantName)
		legacy.SetManagementURL("http://management.test.invalid/url")
		s.service = legacy
	}
	s.LiveTests.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *C) {
	s.LiveTests.TearDownSuite(c)
	s.HTTPSuite.TearDownSuite(c)
}

func (s *localLiveSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.service.SetupHTTP(s.Mux)
	s.LiveTests.SetUpTest(c)
}

func (s *localLiveSuite) TearDownTest(c *C) {
	s.LiveTests.TearDownTest(c)
	s.HTTPSuite.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.

func (s *localLiveSuite) TestInvalidRegion(c *C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("legacy authentication doesn't use regions")
	}
	creds := &identity.Credentials{
		User:    "fred",
		URL:     s.Server.URL,
		Secrets: "secret",
		Region:  "invalid",
	}
	cl := client.NewClient(creds, s.authMode, nil)
	err := cl.Authenticate()
	c.Assert(err, IsNil)
	_, err = cl.MakeServiceURL("object-store", []string{})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*invalid region.*")
}

// Test service lookup with inexact region matching.
func (s *localLiveSuite) TestInexactRegionMatch(c *C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("legacy authentication doesn't use regions")
	}
	cl := client.NewClient(s.cred, s.authMode, nil)
	err := cl.Authenticate()
	serviceURL, err := cl.MakeServiceURL("compute", []string{})
	c.Assert(err, IsNil)
	_, err = url.Parse(serviceURL)
	c.Assert(err, IsNil)
	serviceURL, err = cl.MakeServiceURL("object-store", []string{})
	c.Assert(err, IsNil)
	_, err = url.Parse(serviceURL)
	c.Assert(err, IsNil)
}
