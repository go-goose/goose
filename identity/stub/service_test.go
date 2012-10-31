package identitystub

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
)

type HelloHandler struct{}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write([]byte("Hello World\n"))
}

func (s *HTTPSuite) TestHelloWorld(c *C) {
	s.mux.Handle("/", &HelloHandler{})
	// fmt.Printf("Running HelloWorld\n")
	response, err := http.Get(s.server.URL)
	c.Check(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(err, IsNil)
	c.Check(string(content), Equals, "Hello World\n")
}

func (s *HTTPSuite) setupLegacyIdentityService(user, secret string) (token, managementURL string) {
	managementURL = s.server.URL
	identity := NewLegacyIdentityService(managementURL)
	s.mux.Handle("/", identity)
	token = "new-special-token"
	if user != "" {
		identity.AddUser(user, secret, token)
	}
	return
}

func DoAuthRequest(URL, user, key string) (*http.Response, error) {
	client := &http.Client{}
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

func AssertUnauthorized(c *C, response *http.Response) {
	content, err := ioutil.ReadAll(response.Body)
	c.Assert(err, IsNil)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), Equals, "")
	c.Check(response.Header.Get("X-Server-Management-Url"), Equals, "")
	c.Check(string(content), Equals, "")
	c.Check(response.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *HTTPSuite) TestLegacyIdentityServiceFailedAuth(c *C) {
	s.setupLegacyIdentityService("", "")
	// No headers set for Authentication
	response, err := DoAuthRequest(s.server.URL, "", "")
	c.Assert(err, IsNil)
	AssertUnauthorized(c, response)
}

func (s *HTTPSuite) TestIdentityServiceLegacyFailedOnlyUser(c *C) {
	s.setupLegacyIdentityService("", "")
	// Missing secret key
	response, err := DoAuthRequest(s.server.URL, "user", "")
	c.Assert(err, IsNil)
	AssertUnauthorized(c, response)
}

func (s *HTTPSuite) TestIdentityServiceLegacyNoSuchUser(c *C) {
	s.setupLegacyIdentityService("user", "key")
	// No user matching the username
	response, err := DoAuthRequest(s.server.URL, "notuser", "key")
	c.Assert(err, IsNil)
	AssertUnauthorized(c, response)
}

func (s *HTTPSuite) TestIdentityServiceLegacyInvalidAuth(c *C) {
	s.setupLegacyIdentityService("user", "secret-key")
	// Wrong key
	response, err := DoAuthRequest(s.server.URL, "user", "bad-key")
	c.Assert(err, IsNil)
	AssertUnauthorized(c, response)
}

func (s *HTTPSuite) TestIdentityServiceLegacyAuth(c *C) {
	token, serverURL := s.setupLegacyIdentityService("user", "secret-key")
	response, err := DoAuthRequest(s.server.URL, "user", "secret-key")
	c.Assert(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), Equals, token)
	c.Check(response.Header.Get("X-Server-Management-Url"), Equals, serverURL)
	c.Check(string(content), Equals, "")
	c.Check(response.StatusCode, Equals, http.StatusNoContent)
}
