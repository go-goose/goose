package nova_test

import (
	"bytes"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/nova"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/novaservice"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
)

func registerLocalTests() {
	Suite(&localLiveSuite{})
}

const (
	baseNovaURL = "/V1/1"
)

// localLiveSuite runs tests from LiveTests using a fake
// nova server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	// The following attributes are for using testing doubles.
	Server         *httptest.Server
	Mux            *http.ServeMux
	oldHandler     http.Handler
	identityDouble *identityservice.UserPass
	novaDouble     *novaservice.Nova
}

func (s *localLiveSuite) SetUpSuite(c *C) {
	c.Logf("Using identity and nova service test doubles")

	// Set up the HTTP server.
	s.Server = httptest.NewServer(nil)
	s.oldHandler = s.Server.Config.Handler
	s.Mux = http.NewServeMux()
	s.Server.Config.Handler = s.Mux

	s.cred = &identity.Credentials{
		URL:     s.Server.URL,
		User:    "fred",
		Secrets: "secret",
		Region:  "some region"}
	// Create an identity service and register a Nova endpoint.
	s.identityDouble = identityservice.NewUserPass()
	token := s.identityDouble.AddUser(s.cred.User, s.cred.Secrets)
	ep := identityservice.Endpoint{
		AdminURL:    s.Server.URL + baseNovaURL,
		InternalURL: s.Server.URL + baseNovaURL,
		PublicURL:   s.Server.URL + baseNovaURL,
		Region:      s.cred.Region,
	}
	s.Mux.Handle("/tokens", s.identityDouble)

	service := identityservice.Service{"nova", "compute", []identityservice.Endpoint{ep}}
	s.identityDouble.AddService(service)
	// Create a nova service at the registered endpoint.
	// TODO: identityservice.UserPass always uses tenantId="1", patch this
	//	 when that changes.
	s.novaDouble = novaservice.New("localhost", "V1", token, "1")
	s.novaDouble.SetupHTTP(s.Mux)

	s.LiveTests.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *C) {
	s.LiveTests.TearDownSuite(c)
	s.Mux = nil
	s.Server.Config.Handler = s.oldHandler
	s.Server.Close()
}

func (s *localLiveSuite) SetUpTest(c *C) {
	s.LiveTests.SetUpTest(c)
}

func (s *localLiveSuite) TearDownTest(c *C) {
	s.LiveTests.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.

// TestRateLimitRetry checks that when we make too many requests and receive a Retry-After response, the retry
// occurs and the request ultimately succeeds.
func (s *localLiveSuite) TestRateLimitRetry(c *C) {
	// Capture the logged output so we can check for retry messages.
	var logout bytes.Buffer
	logger := log.New(&logout, "", log.LstdFlags)
	client := client.NewClient(s.cred, identity.AuthUserPass, logger)
	nova := nova.New(client)
	// Delete the artifact if it already exists.
	testGroup, err := nova.SecurityGroupByName("test_group")
	if err != nil {
		c.Assert(errors.IsNotFound(err), Equals, true)
	} else {
		nova.DeleteSecurityGroup(testGroup.Id)
		c.Assert(err, IsNil)
	}
	testGroup, err = nova.CreateSecurityGroup("test_group", "test rate limit")
	c.Assert(err, IsNil)
	nova.DeleteSecurityGroup(testGroup.Id)
	c.Assert(err, IsNil)
	// Ensure we got at least one retry message.
	output := logout.String()
	c.Assert(strings.Contains(output, "Too many requests, retrying in"), Equals, true)
}
