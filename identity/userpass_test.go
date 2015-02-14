package identity

import (
	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/testing/httpsuite"
	"gopkg.in/goose.v1/testservices/identityservice"
)

type UserPassTestSuite struct {
	httpsuite.HTTPSuite
}

var _ = gc.Suite(&UserPassTestSuite{})

func (s *UserPassTestSuite) TestAuthAgainstServer(c *gc.C) {
	service := identityservice.NewUserPass()
	service.SetupHTTP(s.Mux)
	userInfo := service.AddUser("joe-user", "secrets", "tenant")
	var l Authenticator = &UserPass{}
	creds := Credentials{User: "joe-user", URL: s.Server.URL + "/tokens", Secrets: "secrets"}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(auth.TenantId, gc.Equals, userInfo.TenantId)
}

// Test that the region -> service endpoint map is correctly populated.
func (s *UserPassTestSuite) TestRegionMatch(c *gc.C) {
	service := identityservice.NewUserPass()
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
			{PublicURL: "http://nova2", Region: "zone2.RegionOne"},
		}}
	service.AddService(serviceDef)

	creds := Credentials{
		User:    "joe-user",
		URL:     s.Server.URL + "/tokens",
		Secrets: "secrets",
		Region:  "zone1.RegionOne",
	}
	var l Authenticator = &UserPass{}
	auth, err := l.Auth(&creds)
	c.Assert(err, gc.IsNil)
	c.Assert(auth.RegionServiceURLs["RegionOne"]["object-store"], gc.Equals, "http://swift")
	c.Assert(auth.RegionServiceURLs["zone1.RegionOne"]["compute"], gc.Equals, "http://nova")
	c.Assert(auth.RegionServiceURLs["zone2.RegionOne"]["compute"], gc.Equals, "http://nova2")
	c.Assert(auth.Token, gc.Equals, userInfo.Token)
	c.Assert(auth.TenantId, gc.Equals, userInfo.TenantId)
}
