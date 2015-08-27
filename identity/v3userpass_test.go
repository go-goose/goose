package identity

import (
	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/testing/httpsuite"
	"gopkg.in/goose.v1/testservices/identityservice"
)

type V3UserPassTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = gc.Suite(&V3UserPassTestSuite{})

func (s *V3UserPassTestSuite) TestAuthAgainstServer(c *gc.C) {
	service := identityservice.NewV3UserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant")
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
	userInfo := service.AddUser("joe-user", "secrets", "tenant")
	var l Authenticator = &V3UserPass{}
	creds := Credentials{
		User:       "joe-user",
		URL:        s.Server.URL + "/v3/auth/tokens",
		Secrets:    "secrets",
		TenantName: "tenant",
	}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(auth.TenantId, gc.Equals, userInfo.TenantId)
}

func (s *V3UserPassTestSuite) TestAuthWithCatalog(c *gc.C) {
	service := identityservice.NewV3UserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant")
	serviceDef := identityservice.Service{
		Name: "swift",
		Type: "object-store",
		Endpoints: []identityservice.Endpoint{
			{PublicURL: "http://swift", Region: "RegionOne"},
		}}
	service.AddService(serviceDef)
	serviceDef = identityservice.Service{
		Name: "nova",
		Type: "compute",
		Endpoints: []identityservice.Endpoint{
			{PublicURL: "http://nova", Region: "zone1.RegionOne"},
		}}
	service.AddService(serviceDef)
	serviceDef = identityservice.Service{
		Name: "nova",
		Type: "compute",
		Endpoints: []identityservice.Endpoint{
			{PublicURL: "http://nova2", InternalURL: "http://int.nova2", Region: "zone2.RegionOne"},
		}}
	service.AddService(serviceDef)

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
