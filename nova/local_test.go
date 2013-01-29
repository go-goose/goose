package nova_test

import (
	"bytes"
	"fmt"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	goosehttp "launchpad.net/goose/http"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/nova"
	"launchpad.net/goose/testservices"
	"launchpad.net/goose/testservices/openstackservice"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
)

func registerLocalTests() {
	Suite(&localLiveSuite{})
}

// localLiveSuite runs tests from LiveTests using a fake
// nova server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	// The following attributes are for using testing doubles.
	Server                *httptest.Server
	Mux                   *http.ServeMux
	oldHandler            http.Handler
	openstack             *openstackservice.Openstack
	retryErrorCount       int // The current retry error count.
	retryErrorCountToSend int // The number of retry errors to send..
}

func (s *localLiveSuite) SetUpSuite(c *C) {
	c.Logf("Using identity and nova service test doubles")

	// Set up the HTTP server.
	s.Server = httptest.NewServer(nil)
	s.oldHandler = s.Server.Config.Handler
	s.Mux = http.NewServeMux()
	s.Server.Config.Handler = s.Mux

	// Set up an Openstack service.
	s.cred = &identity.Credentials{
		URL:        s.Server.URL,
		User:       "fred",
		Secrets:    "secret",
		Region:     "some region",
		TenantName: "tenant",
	}
	s.openstack = openstackservice.New(s.cred)
	s.openstack.SetupHTTP(s.Mux)

	s.LiveTests.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *C) {
	s.LiveTests.TearDownSuite(c)
	s.Mux = nil
	s.Server.Config.Handler = s.oldHandler
	s.Server.Close()
}

func (s *localLiveSuite) SetUpTest(c *C) {
	s.retryErrorCount = 0
	s.LiveTests.SetUpTest(c)
}

func (s *localLiveSuite) TearDownTest(c *C) {
	s.LiveTests.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.

func (s *localLiveSuite) retryLimitHook(sc testservices.ServiceControl) testservices.ControlProcessor {
	return func(sc testservices.ServiceControl, args ...interface{}) error {
		sendError := s.retryErrorCount < s.retryErrorCountToSend
		if sendError {
			s.retryErrorCount++
			return &testservices.RateLimitExceededError{fmt.Errorf("retry limit exceeded")}
		}
		return nil
	}
}

func (s *localLiveSuite) setupRetryErrorTest(c *C, logger *log.Logger) (*nova.Client, *nova.SecurityGroup) {
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
	testGroup, err = nova.CreateSecurityGroup("test_group", "test")
	c.Assert(err, IsNil)
	return nova, testGroup
}

// TestRateLimitRetry checks that when we make too many requests and receive a Retry-After response, the retry
// occurs and the request ultimately succeeds.
func (s *localLiveSuite) TestRateLimitRetry(c *C) {
	// Capture the logged output so we can check for retry messages.
	var logout bytes.Buffer
	logger := log.New(&logout, "", log.LstdFlags)
	nova, testGroup := s.setupRetryErrorTest(c, logger)
	s.retryErrorCountToSend = goosehttp.MaxRetries
	s.openstack.Nova.RegisterControlPoint("removeSecurityGroup", s.retryLimitHook(s.openstack.Nova))
	defer s.openstack.Nova.RegisterControlPoint("removeSecurityGroup", nil)
	err := nova.DeleteSecurityGroup(testGroup.Id)
	c.Assert(err, IsNil)
	// Ensure we got at least one retry message.
	output := logout.String()
	c.Assert(strings.Contains(output, "Too many requests, retrying in"), Equals, true)
}

// TestRateLimitRetryExceeded checks that an error is raised if too many retry responses are received from the server.
func (s *localLiveSuite) TestRateLimitRetryExceeded(c *C) {
	nova, testGroup := s.setupRetryErrorTest(c, nil)
	s.retryErrorCountToSend = goosehttp.MaxRetries + 1
	s.openstack.Nova.RegisterControlPoint("removeSecurityGroup", s.retryLimitHook(s.openstack.Nova))
	defer s.openstack.Nova.RegisterControlPoint("removeSecurityGroup", nil)
	err := nova.DeleteSecurityGroup(testGroup.Id)
	c.Assert(err, Not(IsNil))
	c.Assert(err.Error(), Matches, ".*Maximum number of retries.*")
}
