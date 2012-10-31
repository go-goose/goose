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

func (s *HTTPSuite) TestLegacyIdentityServiceFailedAuth(c *C) {
	s.mux.Handle("/", NewLegacyIdentityService(""))
	// No headers set for Authentication
	response, err := http.Get(s.server.URL)
	c.Assert(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), Equals, "")
	c.Check(string(content), Equals, "")
	c.Check(response.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *HTTPSuite) TestIdentityServiceLegacyFailedOnlyUser(c *C) {
	s.mux.Handle("/", NewLegacyIdentityService(""))
	// No headers set for Authentication
	client := &http.Client{}
	request, err := http.NewRequest("GET", s.server.URL, nil)
	c.Assert(err, IsNil)
	request.Header.Set("X-Auth-User", "user")
	response, err := client.Do(request)
	c.Assert(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), Equals, "")
	c.Check(response.Header.Get("X-Server-Management-Url"), Equals, "")
	c.Check(string(content), Equals, "")
	c.Check(response.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *HTTPSuite) TestIdentityServiceLegacyNoSuchUser(c *C) {
	s.mux.Handle("/", NewLegacyIdentityService(""))
	// No headers set for Authentication
	client := &http.Client{}
	request, err := http.NewRequest("GET", s.server.URL, nil)
	c.Assert(err, IsNil)
	request.Header.Set("X-Auth-User", "nouser")
	request.Header.Set("X-Auth-Key", "key")
	response, err := client.Do(request)
	c.Assert(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), Equals, "")
	c.Check(response.Header.Get("X-Server-Management-Url"), Equals, "")
	c.Check(string(content), Equals, "")
	c.Check(response.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *HTTPSuite) TestIdentityServiceLegacyInvalidAuth(c *C) {
	serverURL := "http://test/url"
	identity := NewLegacyIdentityService(serverURL)
	s.mux.Handle("/", identity)
	identity.AddUser("user", "secret-key", "spec-token")
	// No headers set for Authentication
	client := &http.Client{}
	request, err := http.NewRequest("GET", s.server.URL, nil)
	c.Assert(err, IsNil)
	request.Header.Set("X-Auth-User", "user")
	request.Header.Set("X-Auth-Key", "bad-key")
	response, err := client.Do(request)
	c.Assert(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), Equals, "")
	c.Check(response.Header.Get("X-Server-Management-Url"), Equals, "")
	c.Check(string(content), Equals, "")
	c.Check(response.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *HTTPSuite) TestIdentityServiceLegacyAuth(c *C) {
	serverURL := "http://test/url"
	identity := NewLegacyIdentityService(serverURL)
	s.mux.Handle("/", identity)
	identity.AddUser("user", "secret-key", "spec-token")
	// No headers set for Authentication
	client := &http.Client{}
	request, err := http.NewRequest("GET", s.server.URL, nil)
	c.Assert(err, IsNil)
	request.Header.Set("X-Auth-User", "user")
	request.Header.Set("X-Auth-Key", "secret-key")
	response, err := client.Do(request)
	c.Assert(err, IsNil)
	content, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	c.Check(response.Header.Get("X-Auth-Token"), Equals, "spec-token")
	c.Check(response.Header.Get("X-Server-Management-Url"), Equals, serverURL)
	c.Check(string(content), Equals, "")
	c.Check(response.StatusCode, Equals, http.StatusNoContent)
}
