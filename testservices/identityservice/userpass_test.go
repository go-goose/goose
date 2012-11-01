package identityservice

import (
	"encoding/json"
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

func UserPassAuthRequest(URL, user, key string) (*http.Response, error) {
	client := &http.Client{}
	req := UserPassRequest{}
	req.Auth.PasswordCredentials.Username = user
	req.Auth.PasswordCredentials.Password = key
	req.Auth.TenantName = "tenant-something-or-other"
	as_json, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body := strings.NewReader(string(as_json))
	request, err := http.NewRequest("POST", URL, body)
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}

func (s *UserPassSuite) TestInvalidRequest(c *C) {
	token := s.setupUserPass("user", "secret")
	c.Assert(token, NotNil)
	res, err := UserPassAuthRequest(s.Server.URL, "user", "bad-secret")
	c.Assert(err, IsNil)
	c.Check(res.StatusCode, Equals, http.StatusUnauthorized)
}

func (s *UserPassSuite) TestValidAuthorization(c *C) {
	token := s.setupUserPass("user", "secret")
	c.Assert(token, NotNil)
	res, err := UserPassAuthRequest(s.Server.URL, "user", "secret")
	c.Assert(err, IsNil)
	c.Check(res.StatusCode, Equals, http.StatusOK)
}
