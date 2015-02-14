package identityservice

import (
	"io/ioutil"
	"net/http"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/testing/httpsuite"
)

type LegacySuite struct {
	httpsuite.HTTPSuite
}

var _ = gc.Suite(&LegacySuite{})

func (s *LegacySuite) setupLegacy(user, secret string) (token, managementURL string) {
	managementURL = s.Server.URL
	identity := NewLegacy()
	// Ensure that it conforms to the interface
	var _ IdentityService = identity
	identity.SetManagementURL(managementURL)
	identity.SetupHTTP(s.Mux)
	if user != "" {
		userInfo := identity.AddUser(user, secret, "tenant")
		token = userInfo.Token
	}
	return
}

func LegacyAuthRequest(URL, user, key string) (*http.Response, error) {
	client := &http.DefaultClient
	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	if user != "" {
		request.Header.Set("X-Auth-User", user)
	}
	if key != "" {
		request.Header.Set("X-Auth-Key", key)
	}
	return client.Do(request)
}

func AssertUnauthorized(c *gc.C, response *http.Response) {
	content, err := ioutil.ReadAll(response.Body)
	c.Assert(err, gc.IsNil)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), gc.Equals, "")
	c.Check(response.Header.Get("X-Server-Management-Url"), gc.Equals, "")
	c.Check(string(content), gc.Equals, "")
	c.Check(response.StatusCode, gc.Equals, http.StatusUnauthorized)
}

func (s *LegacySuite) TestLegacyFailedAuth(c *gc.C) {
	s.setupLegacy("", "")
	// No headers set for Authentication
	response, err := LegacyAuthRequest(s.Server.URL, "", "")
	c.Assert(err, gc.IsNil)
	AssertUnauthorized(c, response)
}

func (s *LegacySuite) TestLegacyFailedOnlyUser(c *gc.C) {
	s.setupLegacy("", "")
	// Missing secret key
	response, err := LegacyAuthRequest(s.Server.URL, "user", "")
	c.Assert(err, gc.IsNil)
	AssertUnauthorized(c, response)
}

func (s *LegacySuite) TestLegacyNoSuchUser(c *gc.C) {
	s.setupLegacy("user", "key")
	// No user matching the username
	response, err := LegacyAuthRequest(s.Server.URL, "notuser", "key")
	c.Assert(err, gc.IsNil)
	AssertUnauthorized(c, response)
}

func (s *LegacySuite) TestLegacyInvalidAuth(c *gc.C) {
	s.setupLegacy("user", "secret-key")
	// Wrong key
	response, err := LegacyAuthRequest(s.Server.URL, "user", "bad-key")
	c.Assert(err, gc.IsNil)
	AssertUnauthorized(c, response)
}

func (s *LegacySuite) TestLegacyAuth(c *gc.C) {
	token, serverURL := s.setupLegacy("user", "secret-key")
	response, err := LegacyAuthRequest(s.Server.URL, "user", "secret-key")
	c.Assert(err, gc.IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), gc.Equals, token)
	c.Check(response.Header.Get("X-Server-Management-Url"), gc.Equals, serverURL+"/compute")
	c.Check(response.Header.Get("X-Storage-Url"), gc.Equals, serverURL+"/object-store")
	c.Check(string(content), gc.Equals, "")
	c.Check(response.StatusCode, gc.Equals, http.StatusNoContent)
}
