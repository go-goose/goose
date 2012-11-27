package client_test

import (
	"flag"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/client"
	"launchpad.net/goose/identity"
	"launchpad.net/goose/testing/httpsuite"
	"launchpad.net/goose/testservices/identityservice"
	"net/http"
	"testing"
)

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

var live = flag.Bool("live", false, "Include live OpenStack (Canonistack) tests")
var authMethodName = flag.String("auth_method", "userpass", "The authentication mode to use [legacy|userpass]")

type ClientSuite struct {
	cred       *identity.Credentials
	authMethod identity.AuthMethod
	// The following attributes are for using testing doubles.
	httpsuite.HTTPSuite
	identityDouble http.Handler
}

func (s *ClientSuite) SetUpSuite(c *C) {
	s.cred = identity.CompleteCredentialsFromEnv()
	switch *authMethodName {
	default:
		c.Fatalf("Invalid auth method specified: %s", *authMethodName)
	case "":
	case "userpass":
		s.authMethod = identity.AuthUserPass
	case "legacy":
		s.authMethod = identity.AuthLegacy
	}
	// If not testing live, set up the test double.
	if !*live {
		c.Logf("Using identity service test double")
		s.HTTPSuite.SetUpSuite(c)
		s.cred.URL = s.Server.URL
		switch *authMethodName {
		case "":
		case "userpass":
			s.identityDouble = identityservice.NewUserPass()
			s.identityDouble.(*identityservice.UserPass).AddUser(s.cred.User, s.cred.Secrets)
		case "legacy":
			s.identityDouble = identityservice.NewLegacy()
			var legacy = s.identityDouble.(*identityservice.Legacy)
			legacy.AddUser(s.cred.User, s.cred.Secrets)
			legacy.SetManagementURL("http://management/url")
		}
	}
}

func (s *ClientSuite) TearDownSuite(c *C) {
	if !*live {
		s.HTTPSuite.TearDownSuite(c)
	}
}

func (s *ClientSuite) SetUpTest(c *C) {
	if !*live {
		s.HTTPSuite.SetUpTest(c)
		s.Mux.Handle("/", s.identityDouble)
	}
}

func (s *ClientSuite) TearDownTest(c *C) {
	if !*live {
		s.HTTPSuite.TearDownTest(c)
	}
}

var suite = Suite(&ClientSuite{})

func (s *ClientSuite) TestAuthenticateFail(c *C) {
	s.cred.User = "fred"
	s.cred.Secrets = "broken"
	s.cred.Region = ""
	var osclient = client.NewOpenStackClient(s.cred, s.authMethod)
	c.Assert(osclient.IsAuthenticated(), Equals, false)
	var err error
	err = osclient.Authenticate()
	c.Assert(err, ErrorMatches, "authentication failed.*")
}

func (s *ClientSuite) TestAuthenticate(c *C) {
	var err error
	var client = client.NewOpenStackClient(s.cred, s.authMethod)
	err = client.Authenticate()
	c.Assert(err, IsNil)
	c.Assert(client.IsAuthenticated(), Equals, true)

	// Check service endpoints are discovered
	c.Assert(client.ServiceURLs["compute"], NotNil)
	c.Assert(client.ServiceURLs["swift"], NotNil)
}
