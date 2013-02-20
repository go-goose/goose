package client_test

import (
	"fmt"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/errors"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices"
	"launchpad.net/goose/testservices/identityservice"
	"launchpad.net/goose/testservices/openstackservice"
	"net/url"
	"runtime"
	"sync"
	"time"
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

type fakeAuthenticator struct {
	mu        sync.Mutex
	nrCallers int
}

// authStart is used as a gate to signal the fake authenticator that it can start.
var authStart chan struct{}

func (auth *fakeAuthenticator) Auth(creds *identity.Credentials) (*identity.AuthDetails, error) {
	auth.mu.Lock()
	auth.nrCallers++
	auth.mu.Unlock()
	// Wait till the test says the authenticator can proceed.
	<-authStart
	runtime.Gosched()
	defer func() {
		auth.mu.Lock()
		auth.nrCallers--
		auth.mu.Unlock()
	}()
	if auth.nrCallers > 1 {
		return nil, fmt.Errorf("Too many callers of Auth function")
	}
	URLs := make(map[string]identity.ServiceURLs)
	endpoints := make(map[string]string)
	endpoints["compute"] = "http://localhost"
	URLs[creds.Region] = endpoints
	return &identity.AuthDetails{
		Token:             "token",
		TenantId:          "tenant",
		UserId:            "1",
		RegionServiceURLs: URLs,
	}, nil
}

func (s *localLiveSuite) TestAuthenticationTimeout(c *C) {
	cl := client.NewClient(s.cred, s.authMode, nil)
	defer client.SetAuthenticationTimeout(time.Duration(1) * time.Millisecond)()
	client.SetAuthenticator(cl, &fakeAuthenticator{})
	authStart = make(chan struct{})

	var err error
	err = cl.Authenticate()
	// Wake up the authenticator after we have timed out.
	authStart <- struct{}{}
	c.Assert(errors.IsTimeout(err), Equals, true)
}

func (s *localLiveSuite) TestAuthenticationSuccess(c *C) {
	cl := client.NewClient(s.cred, s.authMode, nil)
	defer client.SetAuthenticationTimeout(time.Duration(1) * time.Millisecond)()
	client.SetAuthenticator(cl, &fakeAuthenticator{})

	// Signal that the authenticator can proceed immediately.
	authStart = make(chan struct{}, 1)
	authStart <- struct{}{}
	err := cl.Authenticate()
	c.Assert(err, IsNil)
	// It completed with no error but check it also ran correctly.
	c.Assert(cl.IsAuthenticated(), Equals, true)
	URL, err := cl.MakeServiceURL("compute", nil)
	c.Assert(err, IsNil)
	c.Assert(URL, Equals, "http://localhost/")
}

func checkAuthentication(cl client.AuthenticatingClient) error {
	err := cl.Authenticate()
	if err != nil {
		return err
	}
	URL, err := cl.MakeServiceURL("compute", nil)
	if err != nil {
		return err
	}
	if URL != "http://localhost/" {
		return fmt.Errorf("Unexpected URL: %s", URL)
	}
	return nil
}

func (s *localLiveSuite) TestAuthenticationForbidsMultipleCallers(c *C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("legacy authentication")
	}
	cl := client.NewClient(s.cred, s.authMode, nil)
	client.SetAuthenticator(cl, &fakeAuthenticator{})

	// Signal that the authenticator can proceed immediately.
	authStart = make(chan struct{}, 2)
	authStart <- struct{}{}
	authStart <- struct{}{}
	allDone := make(chan struct{})
	// Record the error outside the go routine since Assert failures inside the func makes
	// the test hang.
	var err1, err2 error
	go func() {
		err1 = checkAuthentication(cl)
		allDone <- struct{}{}
	}()
	go func() {
		err2 = checkAuthentication(cl)
		allDone <- struct{}{}
	}()
	<-allDone
	<-allDone
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)
}
