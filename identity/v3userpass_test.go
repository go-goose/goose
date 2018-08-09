package identity

import (
	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v2/testing/httpsuite"
	"gopkg.in/goose.v2/testservices/identityservice"
)

type V3UserPassTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = gc.Suite(&V3UserPassTestSuite{})

func (s *V3UserPassTestSuite) TestAuthAgainstServer(c *gc.C) {
	service := identityservice.NewV3UserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant", "default")
	var l Authenticator = &V3UserPass{}
	creds := Credentials{
		User:    "joe-user",
		URL:     s.Server.URL + "/v3/auth/tokens",
		Secrets: "secrets",
	}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
}

func (s *V3UserPassTestSuite) TestAuthToAProject(c *gc.C) {
	service := identityservice.NewV3UserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant", "project-domain")
	var l Authenticator = &V3UserPass{}
	creds := Credentials{
		User:          "joe-user",
		URL:           s.Server.URL + "/v3/auth/tokens",
		Secrets:       "secrets",
		TenantName:    "tenant",
		ProjectDomain: "project-domain",
	}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(auth.TenantId, gc.Equals, userInfo.TenantId)
}

func (s *V3UserPassTestSuite) TestAuthToADomain(c *gc.C) {
	service := identityservice.NewV3UserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant", "domain")
	var l Authenticator = &V3UserPass{}
	creds := Credentials{
		User:       "joe-user",
		URL:        s.Server.URL + "/v3/auth/tokens",
		Secrets:    "secrets",
		TenantName: "tenant",
		Domain:     "domain",
	}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(auth.Domain, gc.Equals, "domain")
}

func (s *V3UserPassTestSuite) TestAuthToTenantNameAndTenantID(c *gc.C) {
	service := identityservice.NewV3UserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant", "project-domain")
	var l Authenticator = &V3UserPass{}
	creds := Credentials{
		User:          "joe-user",
		URL:           s.Server.URL + "/v3/auth/tokens",
		Secrets:       "secrets",
		TenantName:    "tenant",
		TenantID:      "tenantID",
		ProjectDomain: "project-domain",
	}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(auth.TenantName, gc.Equals, userInfo.TenantName)
	c.Assert(auth.TenantId, gc.Equals, userInfo.TenantId)
}

func (s *V3UserPassTestSuite) TestAuthWithCatalog(c *gc.C) {
	service := identityservice.NewV3UserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant", "default")
	serviceDef := identityservice.V3Service{
		Name:      "swift",
		Type:      "object-store",
		Endpoints: identityservice.NewV3Endpoints("", "", "http://swift", "RegionOne"),
	}
	service.AddService(identityservice.Service{V3: serviceDef})
	serviceDef = identityservice.V3Service{
		Name:      "nova",
		Type:      "compute",
		Endpoints: identityservice.NewV3Endpoints("", "", "http://nova", "zone1.RegionOne"),
	}
	service.AddService(identityservice.Service{V3: serviceDef})
	serviceDef = identityservice.V3Service{
		Name:      "nova",
		Type:      "compute",
		Endpoints: identityservice.NewV3Endpoints("", "http://int.nova2", "http://nova2", "zone2.RegionOne"),
	}
	service.AddService(identityservice.Service{V3: serviceDef})

	creds := Credentials{
		User:       "joe-user",
		URL:        s.Server.URL + "/v3/auth/tokens",
		Secrets:    "secrets",
		TenantName: "tenant",
	}
	var l Authenticator = &V3UserPass{}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.RegionServiceURLs["RegionOne"]["object-store"], gc.Equals, "http://swift")
	c.Assert(auth.RegionServiceURLs["zone1.RegionOne"]["compute"], gc.Equals, "http://nova")
	c.Assert(auth.RegionServiceURLs["zone2.RegionOne"]["compute"], gc.Equals, "http://nova2")
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(auth.TenantId, gc.Equals, userInfo.TenantId)
}
