package swift_test

import (
	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/identity"
	"gopkg.in/goose.v1/testing/httpsuite"
	"gopkg.in/goose.v1/testservices/openstackservice"
)

func registerLocalTests() {
	gc.Suite(&localLiveSuite{})
}

// localLiveSuite runs tests from LiveTests using a fake
// swift server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	LiveTestsPublicContainer
	// The following attributes are for using testing doubles.
	httpsuite.HTTPSuite
	openstack *openstackservice.Openstack
}

func (s *localLiveSuite) SetUpSuite(c *gc.C) {
	c.Logf("Using identity and swift service test doubles")
	s.HTTPSuite.SetUpSuite(c)
	// Set up an Openstack service.
	s.LiveTests.cred = &identity.Credentials{
		URL:        s.Server.URL,
		User:       "fred",
		Secrets:    "secret",
		Region:     "some region",
		TenantName: "tenant",
	}
	s.LiveTestsPublicContainer.cred = s.LiveTests.cred
	s.openstack = openstackservice.New(s.LiveTests.cred,
		identity.AuthUserPass)

	s.LiveTests.SetUpSuite(c)
	s.LiveTestsPublicContainer.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *gc.C) {
	s.LiveTests.TearDownSuite(c)
	s.LiveTestsPublicContainer.TearDownSuite(c)
	s.HTTPSuite.TearDownSuite(c)
}

func (s *localLiveSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.openstack.SetupHTTP(s.Mux)
	s.LiveTests.SetUpTest(c)
	s.LiveTestsPublicContainer.SetUpTest(c)
}

func (s *localLiveSuite) TearDownTest(c *gc.C) {
	s.LiveTests.TearDownTest(c)
	s.LiveTestsPublicContainer.TearDownTest(c)
	s.HTTPSuite.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.
