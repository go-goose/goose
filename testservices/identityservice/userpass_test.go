package identityservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/goose/testing/httpsuite"
	"net/http"
	"strings"
)

type UserPassSuite struct {
	httpsuite.HTTPSuite
}

var _ = Suite(&UserPassSuite{})

func (s *UserPassSuite) setupUserPass(user, secret string) (token string) {
	var identity = NewUserPass()
	// Ensure that it conforms to the interface
	var _ IdentityService = identity
	s.Mux.Handle("/", identity)
	token = "new-special-token"
	if user != "" {
		identity.AddUser(user, secret, token)
	}
	return
}

var authTemplate = `{
    "auth": {
        "tenantName": "tenant-something", 
        "passwordCredentials": {
            "username": "%s", 
            "password": "%s"
        }
    }
}`

func UserPassAuthRequest(URL, user, key string) (*http.Response, error) {
	client := &http.Client{}
	body := strings.NewReader(fmt.Sprintf(authTemplate, user, key))
	request, err := http.NewRequest("POST", URL, body)
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}

func (s *UserPassSuite) TestNotJSON(c *C) {
	// We do everything in UserPassAuthRequest, except set the Content-Type
	token := s.setupUserPass("user", "secret")
	c.Assert(token, NotNil)
	client := &http.Client{}
	body := strings.NewReader(fmt.Sprintf(authTemplate, "user", "secret"))
	request, err := http.NewRequest("POST", s.Server.URL, body)
	c.Assert(err, IsNil)
	res, err := client.Do(request)
	c.Assert(err, IsNil)
	c.Check(res.StatusCode, Equals, http.StatusBadRequest)
}

func (s *UserPassSuite) TestNoSuchUser(c *C) {
	token := s.setupUserPass("user", "secret")
	c.Assert(token, NotNil)
	res, err := UserPassAuthRequest(s.Server.URL, "not-user", "secret")
	defer res.Body.Close()
	c.Assert(err, IsNil)
	c.Check(res.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *UserPassSuite) TestBadPassword(c *C) {
	token := s.setupUserPass("user", "secret")
	c.Assert(token, NotNil)
	res, err := UserPassAuthRequest(s.Server.URL, "user", "not-secret")
	defer res.Body.Close()
	c.Assert(err, IsNil)
	c.Check(res.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *UserPassSuite) TestValidAuthorization(c *C) {
	token := s.setupUserPass("user", "secret")
	c.Assert(token, NotNil)
	res, err := UserPassAuthRequest(s.Server.URL, "user", "secret")
	defer res.Body.Close()
	c.Assert(err, IsNil)
	c.Check(res.StatusCode, Equals, http.StatusOK)
	content, err := ioutil.ReadAll(res.Body)
	c.Assert(err, IsNil)
	var response AccessResponse
	err = json.Unmarshal(content, &response)
	c.Assert(err, IsNil)
	c.Check(response.Access.Token.Id, Equals, token)
	novaURL := ""
	for _, service := range response.Access.ServiceCatalog {
		if service.Type == "compute" {
			novaURL = service.Endpoints[0].PublicURL
			break
		}
	}
	c.Assert(novaURL, Not(Equals), "")
}
