package client_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/juju/loggo"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v2/client"
	"gopkg.in/goose.v2/errors"
	"gopkg.in/goose.v2/identity"
	"gopkg.in/goose.v2/logging"
	"gopkg.in/goose.v2/swift"
	"gopkg.in/goose.v2/testing/httpsuite"
	"gopkg.in/goose.v2/testservices"
	"gopkg.in/goose.v2/testservices/identityservice"
	"gopkg.in/goose.v2/testservices/openstackservice"
)

func registerLocalTests(authModes []identity.AuthMode) {
	for _, authMode := range authModes {
		gc.Suite(&localLiveSuite{
			LiveTests: LiveTests{
				authMode: authMode,
			},
		})
	}
	gc.Suite(&localHTTPSSuite{HTTPSuite: httpsuite.HTTPSuite{UseTLS: true}})
}

// localLiveSuite runs tests from LiveTests using a fake
// identity server that runs within the test process itself.
type localLiveSuite struct {
	LiveTests
	// The following attributes are for using testing doubles.
	httpsuite.HTTPSuite
	service         testservices.HttpService
	versionHandlers map[string]*versionHandler
}

func (s *localLiveSuite) SetUpSuite(c *gc.C) {
	c.Logf("Using identity service test double")
	s.HTTPSuite.SetUpSuite(c)
	s.cred = &identity.Credentials{
		URL:           s.Server.URL,
		User:          "fred",
		Secrets:       "secret",
		Region:        "zone1.some region",
		TenantName:    "tenant",
		ProjectDomain: "default",
	}
	var logMsg []string
	switch s.authMode {
	default:
		panic("Invalid authentication method")
	case identity.AuthKeyPair:
		// The openstack test service sets up keypair authentication.
		s.service, logMsg = openstackservice.New(s.cred, identity.AuthKeyPair, s.UseTLS)
		// Add an additional endpoint so region filtering can be properly tested.
		serviceDef := identityservice.Service{
			V2: identityservice.V2Service{
				Name: "nova",
				Type: "compute",
				Endpoints: []identityservice.Endpoint{
					{PublicURL: "http://nova2", Region: "zone2.RegionOne"},
				},
			}}
		s.service.(*openstackservice.Openstack).Identity.AddService(serviceDef)
	case identity.AuthUserPass:
		// The openstack test service sets up userpass authentication.
		s.service, logMsg = openstackservice.New(s.cred, identity.AuthUserPass, s.UseTLS)
		// Add an additional endpoint so region filtering can be properly tested.
		serviceDef := identityservice.Service{
			V2: identityservice.V2Service{
				Name: "nova",
				Type: "compute",
				Endpoints: []identityservice.Endpoint{
					{PublicURL: "http://nova2", Region: "zone2.RegionOne"},
				},
			}}
		s.service.(*openstackservice.Openstack).Identity.AddService(serviceDef)
	case identity.AuthUserPassV3:
		// The openstack test service sets up userpass authentication.
		s.service, logMsg = openstackservice.New(s.cred, identity.AuthUserPass, s.UseTLS)
		// Add an additional endpoint so region filtering can be properly tested.
		serviceDef := identityservice.Service{
			V3: identityservice.V3Service{
				Name:      "nova",
				Type:      "compute",
				Endpoints: identityservice.NewV3Endpoints("", "", "http://nova2", "zone2.RegionOne"),
			}}
		s.service.(*openstackservice.Openstack).Identity.AddService(serviceDef)

	case identity.AuthLegacy:
		legacy := identityservice.NewLegacy()
		legacy.AddUser(s.cred.User, s.cred.Secrets, s.cred.TenantName, "default")
		legacy.SetManagementURL("http://management.test.invalid/url")
		s.service = legacy
	}
	for _, msg := range logMsg {
		c.Logf(msg)
	}
	if s.authMode != identity.AuthLegacy {
		s.service.SetupHTTP(nil)
	}
	s.versionHandlers = make(map[string]*versionHandler)
	s.LiveTests.SetUpSuite(c)
}

func (s *localLiveSuite) TearDownSuite(c *gc.C) {
	s.LiveTests.TearDownSuite(c)
	s.HTTPSuite.TearDownSuite(c)
	s.service.Stop()
}

func (s *localLiveSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
	s.LiveTests.SetUpTest(c)
	if s.authMode == identity.AuthLegacy {
		s.service.SetupHTTP(s.Mux)
	}
}

func (s *localLiveSuite) TearDownTest(c *gc.C) {
	s.LiveTests.TearDownTest(c)
	s.HTTPSuite.TearDownTest(c)
}

// Additional tests to be run against the service double only go here.

func (s *localLiveSuite) TestInvalidRegion(c *gc.C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("legacy authentication doesn't use regions")
	}
	creds := &identity.Credentials{
		User:    "fred",
		URL:     s.service.(*openstackservice.Openstack).URLs["identity"],
		Secrets: "secret",
		Region:  "invalid",
	}
	cl := client.NewClient(creds, s.authMode, nil)
	err := cl.Authenticate()
	c.Assert(err.Error(), gc.Matches, "(.|\n)*invalid region(.|\n)*")
}

// Test service lookup with inexact region matching.
func (s *localLiveSuite) TestInexactRegionMatch(c *gc.C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("legacy authentication doesn't use regions")
	}
	cl := client.NewClient(s.cred, s.authMode, nil)
	err := cl.Authenticate()
	serviceURL, err := cl.MakeServiceURL("compute", "v2", []string{})
	c.Assert(err, gc.IsNil)
	_, err = url.Parse(serviceURL)
	c.Assert(err, gc.IsNil)
	serviceURL, err = cl.MakeServiceURL("object-store", "", []string{})
	c.Assert(err, gc.IsNil)
	_, err = url.Parse(serviceURL)
	c.Assert(err, gc.IsNil)
}

type fakeAuthenticator struct {
	mu        sync.Mutex
	nrCallers int
	// authStart is used as a gate to signal the fake authenticator that it can start.
	authStart chan struct{}
	port      string // for startApiVersionMux()
}

func newAuthenticator(bufsize int, port string) *fakeAuthenticator {
	return &fakeAuthenticator{
		authStart: make(chan struct{}, bufsize),
		port:      port,
	}
}

// doNewAuthenticator sets up the HTTP listener on localhost:<port> for testing
// api version functionality within MakeServiceURL if we're using a fakeAuthenticator.
func (s *localLiveSuite) doNewAuthenticator(c *gc.C, bufsize int, port string) *fakeAuthenticator {
	newAuth := newAuthenticator(bufsize, port)
	newAuth.mu.Lock()
	if _, ok := s.versionHandlers[port]; !ok {
		var vh versionHandler
		switch port {
		case "3000":
			vh.authBody = authInformationBody
		case "3003":
			vh.authBody = authValuesInformationBody
		case "3005":
			vh.authBody = ""
		}
		vh.port = port
		c.Logf(startApiVersionMux(vh))
		s.versionHandlers[port] = &vh
	}
	newAuth.mu.Unlock()
	return newAuth
}

func (auth *fakeAuthenticator) Auth(creds *identity.Credentials) (*identity.AuthDetails, error) {
	auth.mu.Lock()
	auth.nrCallers++
	auth.mu.Unlock()
	// Wait till the test says the authenticator can proceed.
	<-auth.authStart
	runtime.Gosched()
	defer func() {
		auth.mu.Lock()
		auth.nrCallers--
		auth.mu.Unlock()
	}()
	auth.mu.Lock()
	tooManyCallers := auth.nrCallers > 1
	auth.mu.Unlock()
	if tooManyCallers {
		return nil, fmt.Errorf("Too many callers of Auth function")
	}
	URLs := make(map[string]identity.ServiceURLs)
	endpoints := make(map[string]string)
	endpoints["compute"] = fmt.Sprintf("http://localhost:%s", auth.port)
	// Special case for https://bugs.launchpad.net/juju/+bug/1756135
	endpoints["compute2"] = fmt.Sprintf("http://localhost:%s/v2/010ab46135ba414882641f663ec917b6", auth.port)
	// Special case for https://bugs.launchpad.net/juju/+bug/1756135
	endpoints["compute3"] = fmt.Sprintf("http://localhost:%s/compute", auth.port)
	// Special case for https://bugs.launchpad.net/juju/+bug/1756135
	endpoints["compute4"] = fmt.Sprintf("http://localhost:%s/computev1/v2", auth.port)
	endpoints["object-store"] = fmt.Sprintf("http://localhost:%s/swift/v1", auth.port)
	endpoints["juju-container-test"] = fmt.Sprintf("http://localhost:%s/swift/v1", auth.port)
	URLs[creds.Region] = endpoints
	return &identity.AuthDetails{
		Token:             "token",
		TenantId:          "tenant",
		UserId:            "1",
		RegionServiceURLs: URLs,
	}, nil
}

func (s *localLiveSuite) TestAuthenticationTimeout(c *gc.C) {
	cl := client.NewClient(s.cred, s.authMode, nil)
	defer client.SetAuthenticationTimeout(1 * time.Millisecond)()
	auth := s.doNewAuthenticator(c, 0, "3003")
	client.SetAuthenticator(cl, auth)

	var err error
	err = cl.Authenticate()
	// Wake up the authenticator after we have timed out.
	auth.authStart <- struct{}{}
	c.Assert(errors.IsTimeout(err), gc.Equals, true)
}

func (s *localLiveSuite) assertAuthenticationSuccess(c *gc.C, port string) client.AuthenticatingClient {
	cl := client.NewClient(s.cred, s.authMode, logging.LoggoLogger{loggo.GetLogger("goose.client")})
	cl.SetRequiredServiceTypes([]string{"compute"})
	defer client.SetAuthenticationTimeout(2 * time.Millisecond)()
	auth := s.doNewAuthenticator(c, 1, port)
	client.SetAuthenticator(cl, auth)

	// Signal that the authenticator can proceed immediately.
	auth.authStart <- struct{}{}
	err := cl.Authenticate()
	c.Assert(err, gc.IsNil)
	// It completed with no error but check it also ran correctly.
	c.Assert(cl.IsAuthenticated(), gc.Equals, true)
	return cl
}

func (s *localLiveSuite) TestAuthenticationSuccess(c *gc.C) {
	cl := s.assertAuthenticationSuccess(c, "3000")
	URL, err := cl.MakeServiceURL("compute", "v2.0", nil)
	c.Assert(err, gc.IsNil)
	c.Assert(URL, gc.Equals, "http://localhost:3000/v2.0")
}

func checkAuthentication(cl client.AuthenticatingClient) error {
	err := cl.Authenticate()
	if err != nil {
		return err
	}
	URL, err := cl.MakeServiceURL("compute", "v3", nil)
	if err != nil {
		return err
	}
	if URL != "http://localhost:3000/v3" {
		return fmt.Errorf("Unexpected URL: %s", URL)
	}
	return nil
}

func (s *localLiveSuite) TestAuthenticationForbidsMultipleCallers(c *gc.C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("legacy authentication")
	}
	cl := client.NewClient(s.cred, s.authMode, nil)
	cl.SetRequiredServiceTypes([]string{"compute"})
	auth := s.doNewAuthenticator(c, 2, "3000")
	client.SetAuthenticator(cl, auth)

	// Signal that the authenticator can proceed immediately.
	auth.authStart <- struct{}{}
	auth.authStart <- struct{}{}
	var allDone sync.WaitGroup
	allDone.Add(2)
	var err1, err2 error
	go func() {
		err1 = checkAuthentication(cl)
		allDone.Done()
	}()
	go func() {
		err2 = checkAuthentication(cl)
		allDone.Done()
	}()
	allDone.Wait()
	c.Assert(err1, gc.IsNil)
	c.Assert(err2, gc.IsNil)
}

type configurableAuth struct {
	regionsURLs map[string]identity.ServiceURLs
}

func NewConfigurableAuth(regionsURLData string) *configurableAuth {
	auth := &configurableAuth{}
	err := json.Unmarshal([]byte(regionsURLData), &auth.regionsURLs)
	if err != nil {
		panic(err)
	}
	return auth
}

func (auth *configurableAuth) Auth(creds *identity.Credentials) (*identity.AuthDetails, error) {
	return &identity.AuthDetails{
		Token:             "token",
		TenantId:          "tenant",
		UserId:            "1",
		RegionServiceURLs: auth.regionsURLs,
	}, nil
}

type authRegionTest struct {
	region        string
	regionURLInfo string
	errorMsg      string
}

var missingEndpointMsgf = "(.|\n)*the configured region %q does not allow access to all required services, namely: %s(.|\n)*access to these services is missing: %s"
var missingEndpointSuggestRegionMsgf = "(.|\n)*the configured region %q does not allow access to all required services, namely: %s(.|\n)*access to these services is missing: %s(.|\n)*one of these regions may be suitable instead: %s"
var invalidRegionMsgf = "(.|\n)*invalid region %q"

var authRegionTests = []authRegionTest{
	{
		"a.region.1",
		`{"a.region.1":{"compute":"http://foo"}}`,
		fmt.Sprintf(missingEndpointMsgf, "a.region.1", "compute, object-store", "object-store"),
	},
	{
		"b.region.1",
		`{"a.region.1":{"compute":"http://foo"}}`,
		fmt.Sprintf(invalidRegionMsgf, "b.region.1"),
	},
	{
		"b.region.1",
		`{"a.region.1":{"compute":"http://foo"}, "region.1":{"object-store":"http://foobar"}}`,
		fmt.Sprintf(missingEndpointSuggestRegionMsgf, "b.region.1", "compute, object-store", "compute", "a.region.1"),
	},
	{
		"region.1",
		`{"a.region.1":{"compute":"http://foo"}, "region.1":{"object-store":"http://foobar"}}`,
		fmt.Sprintf(missingEndpointSuggestRegionMsgf, "region.1", "compute, object-store", "compute", "a.region.1"),
	},
}

func (s *localLiveSuite) TestNonAccessibleServiceType(c *gc.C) {
	if s.authMode == identity.AuthLegacy {
		c.Skip("legacy authentication")
	}
	for _, at := range authRegionTests {
		s.cred.Region = at.region
		cl := client.NewClient(s.cred, s.authMode, nil)
		auth := NewConfigurableAuth(at.regionURLInfo)
		client.SetAuthenticator(cl, auth)
		err := cl.Authenticate()
		c.Assert(err, gc.ErrorMatches, at.errorMsg)
	}
}

type localHTTPSSuite struct {
	// The following attributes are for using testing doubles.
	httpsuite.HTTPSuite
	service testservices.HttpService
	cred    *identity.Credentials
}

func (s *localHTTPSSuite) SetUpSuite(c *gc.C) {
	c.Logf("Using identity service test double")
	s.HTTPSuite.SetUpSuite(c)
	c.Assert(s.Server.URL[:8], gc.Equals, "https://")
	s.cred = &identity.Credentials{
		User:       "fred",
		Secrets:    "secret",
		Region:     "zone1.some region",
		TenantName: "tenant",
	}
	// The openstack test service sets up userpass authentication.
	var logMsg []string
	s.service, logMsg = openstackservice.New(s.cred, identity.AuthUserPass, s.UseTLS)
	for _, msg := range logMsg {
		c.Logf(msg)
	}
	// Add an additional endpoint so region filtering can be properly tested.
	serviceDef := identityservice.Service{
		V2: identityservice.V2Service{
			Name: "nova",
			Type: "compute",
			Endpoints: []identityservice.Endpoint{
				{PublicURL: "https://nova2", Region: "zone2.RegionOne"},
			},
		}}
	s.service.(*openstackservice.Openstack).Identity.AddService(serviceDef)
	s.service.SetupHTTP(nil)
}

func (s *localHTTPSSuite) TearDownSuite(c *gc.C) {
	s.HTTPSuite.TearDownSuite(c)
	s.service.Stop()
}

func (s *localHTTPSSuite) SetUpTest(c *gc.C) {
	s.HTTPSuite.SetUpTest(c)
}

func (s *localHTTPSSuite) TearDownTest(c *gc.C) {
	s.HTTPSuite.TearDownTest(c)
}

func (s *localHTTPSSuite) TestDefaultClientRefusesSelfSigned(c *gc.C) {
	cl := client.NewClient(s.cred, identity.AuthUserPass, nil)
	err := cl.Authenticate()
	c.Assert(err, gc.ErrorMatches, "(.|\n)*x509: certificate signed by unknown authority")
}

func (s *localHTTPSSuite) TestNonValidatingClientAcceptsSelfSigned(c *gc.C) {
	cl := client.NewNonValidatingClient(s.cred, identity.AuthUserPass, nil)
	err := cl.Authenticate()
	c.Assert(err, gc.IsNil)

	// Requests into this client should be https:// URLs
	swiftURL, err := cl.MakeServiceURL("object-store", "", []string{"test_container"})
	c.Assert(err, gc.IsNil)
	c.Assert(swiftURL[:8], gc.Equals, "https://")
	// We use swiftClient.CreateContainer to test a Binary request
	swiftClient := swift.New(cl)
	c.Assert(swiftClient.CreateContainer("test_container", swift.Private), gc.IsNil)

	// And we use List to test the JsonRequest
	contents, err := swiftClient.List("test_container", "", "", "", 0)
	c.Assert(err, gc.IsNil)
	c.Check(contents, gc.DeepEquals, []swift.ContainerContents{})
}

func (s *localHTTPSSuite) setupPublicContainer(c *gc.C) string {
	// First set up a container that can be read publically
	authClient := client.NewNonValidatingClient(s.cred, identity.AuthUserPass, nil)
	authSwift := swift.New(authClient)
	err := authSwift.CreateContainer("test_container", swift.PublicRead)
	c.Assert(err, gc.IsNil)

	baseURL, err := authClient.MakeServiceURL("object-store", "", nil)
	c.Assert(err, gc.IsNil)
	c.Assert(baseURL[:8], gc.Equals, "https://")
	return baseURL
}

func (s *localHTTPSSuite) TestDefaultPublicClientRefusesSelfSigned(c *gc.C) {
	baseURL := s.setupPublicContainer(c)
	swiftClient := swift.New(client.NewPublicClient(baseURL, nil))
	contents, err := swiftClient.List("test_container", "", "", "", 0)
	c.Assert(err, gc.ErrorMatches, "(.|\n)*x509: certificate signed by unknown authority")
	c.Assert(contents, gc.DeepEquals, []swift.ContainerContents(nil))
}

func (s *localHTTPSSuite) TestNonValidatingPublicClientAcceptsSelfSigned(c *gc.C) {
	baseURL := s.setupPublicContainer(c)
	swiftClient := swift.New(client.NewNonValidatingPublicClient(baseURL, nil))
	contents, err := swiftClient.List("test_container", "", "", "", 0)
	c.Assert(err, gc.IsNil)
	c.Assert(contents, gc.DeepEquals, []swift.ContainerContents{})
}

func (s *localHTTPSSuite) TestAuthDiscover(c *gc.C) {
	cl := client.NewNonValidatingClient(s.cred, identity.AuthUserPass, nil)
	options, err := cl.IdentityAuthOptions()
	c.Assert(err, gc.IsNil)
	c.Assert(options, gc.DeepEquals, identity.AuthOptions{identity.AuthOption{Mode: 3, Endpoint: s.cred.URL + "/v3/"}, identity.AuthOption{Mode: 1, Endpoint: s.cred.URL + "/v2.0/"}})
}

func (s *localHTTPSSuite) TestTLSConfigClientBadConfig(c *gc.C) {
	cl := client.NewClientTLSConfig(s.cred, identity.AuthUserPass, nil, &tls.Config{})
	err := cl.Authenticate()
	c.Assert(err, gc.ErrorMatches, "(.|\n)*x509: certificate signed by unknown authority")
}

func (s *localHTTPSSuite) TestTLSConfigClient(c *gc.C) {
	cl := client.NewClientTLSConfig(s.cred, identity.AuthUserPass, nil, s.tlsConfig())
	err := cl.Authenticate()
	c.Assert(err, gc.IsNil)
}

func (s *localHTTPSSuite) tlsConfig() *tls.Config {
	pool := x509.NewCertPool()
	pool.AddCert(s.HTTPSuite.Server.Certificate())
	return &tls.Config{
		RootCAs: pool,
	}
}
