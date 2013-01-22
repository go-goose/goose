package swift_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/swiftservice"
)

func registerLocalTests() {
	Suite(&localLiveSuite{})
}

// localLiveSuite runs tests from LiveTests using a fake
// swift server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	LiveTestsPublicContainer
	// The following attributes are for using testing doubles.
	httpsuite.HTTPSuite
	identityDouble *identityservice.UserPass
	swiftDouble    *swiftservice.Swift
}

func (s *localLiveSuite) SetUpSuite(c *C) {
	c.Logf("Using identity and swift service test doubles")
	s.HTTPSuite.SetUpSuite(c)
	s.LiveTests.cred = &identity.Credentials{
		URL:     s.Server.URL,
		User:    "fred",
		Secrets: "secret",
		Region:  "some region"}
	s.LiveTestsPublicContainer.cred = s.LiveTests.cred
	// Create an identity service and register a Swift endpoint.
	s.identityDouble = identityservice.NewUserPass()
	token := s.identityDouble.AddUser(s.LiveTests.cred.User, s.LiveTests.cred.Secrets)
	s.swiftDouble = swiftservice.New(s.Server.URL, token, s.LiveTests.cred.Region)
	s.identityDouble.RegisterService("swift", "object-store", s.swiftDouble)

	s.LiveTests.SetUpSuite(c)
	s.LiveTestsPublicContainer.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *C) {
	s.LiveTests.TearDownSuite(c)
	s.LiveTestsPublicContainer.TearDownSuite(c)
	s.HTTPSuite.TearDownSuite(c)
}

func (s *localLiveSuite) SetUpTest(c *C) {
	s.HTTPSuite.SetUpTest(c)
	s.swiftDouble.SetupHTTP(s.Mux)
	s.identityDouble.SetupHTTP(s.Mux)
	s.LiveTests.SetUpTest(c)
	s.LiveTestsPublicContainer.SetUpTest(c)
}

func (s *localLiveSuite) TearDownTest(c *C) {
	s.LiveTests.TearDownTest(c)
	s.LiveTestsPublicContainer.TearDownTest(c)
	s.HTTPSuite.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.
