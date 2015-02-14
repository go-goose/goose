package identityservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/testing/httpsuite"
)

type KeyPairSuite struct {
	httpsuite.HTTPSuite
}

var _ = gc.Suite(&KeyPairSuite{})

func makeKeyPair(user, secret string) (identity *KeyPair) {
	identity = NewKeyPair()
	// Ensure that it conforms to the interface
	var _ IdentityService = identity
	if user != "" {
		identity.AddUser(user, secret, "tenant")
	}
	return
}

func (s *KeyPairSuite) setupKeyPair(user, secret string) {
	var identity *KeyPair
	identity = makeKeyPair(user, secret)
	identity.SetupHTTP(s.Mux)
	return
}

func (s *KeyPairSuite) setupKeyPairWithServices(user, secret string, services []Service) {
	var identity *KeyPair
	identity = makeKeyPair(user, secret)
	for _, service := range services {
		identity.AddService(service)
	}
	identity.SetupHTTP(s.Mux)
	return
}

const authKeyPairTemplate = `{
    "auth": {
        "tenantName": "tenant-something",
        "apiAccessKeyCredentials": {
            "accessKey": "%s",
            "secretKey": "%s"
        }
    }
}`

func keyPairAuthRequest(URL, access, secret string) (*http.Response, error) {
	client := &http.DefaultClient
	body := strings.NewReader(fmt.Sprintf(authKeyPairTemplate, access, secret))
	request, err := http.NewRequest("POST", URL+"/tokens", body)
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}

func (s *KeyPairSuite) TestNotJSON(c *gc.C) {
	// We do everything in keyPairAuthRequest, except set the Content-Type
	s.setupKeyPair("user", "secret")
	client := &http.DefaultClient
	body := strings.NewReader(fmt.Sprintf(authTemplate, "user", "secret"))
	request, err := http.NewRequest("POST", s.Server.URL+"/tokens", body)
	c.Assert(err, gc.IsNil)
	res, err := client.Do(request)
	defer res.Body.Close()
	c.Assert(err, gc.IsNil)
	CheckErrorResponse(c, res, http.StatusBadRequest, notJSON)
}

func (s *KeyPairSuite) TestBadJSON(c *gc.C) {
	// We do everything in keyPairAuthRequest, except set the Content-Type
	s.setupKeyPair("user", "secret")
	res, err := keyPairAuthRequest(s.Server.URL, `garbage"in`, "secret")
	defer res.Body.Close()
	c.Assert(err, gc.IsNil)
	CheckErrorResponse(c, res, http.StatusBadRequest, notJSON)
}

func (s *KeyPairSuite) TestNoSuchUser(c *gc.C) {
	s.setupKeyPair("user", "secret")
	res, err := keyPairAuthRequest(s.Server.URL, "not-user", "secret")
	defer res.Body.Close()
	c.Assert(err, gc.IsNil)
	CheckErrorResponse(c, res, http.StatusUnauthorized, notAuthorized)
}

func (s *KeyPairSuite) TestBadPassword(c *gc.C) {
	s.setupKeyPair("user", "secret")
	res, err := keyPairAuthRequest(s.Server.URL, "user", "not-secret")
	defer res.Body.Close()
	c.Assert(err, gc.IsNil)
	CheckErrorResponse(c, res, http.StatusUnauthorized, invalidUser)
}

func (s *KeyPairSuite) TestValidAuthorization(c *gc.C) {
	compute_url := "http://testing.invalid/compute"
	s.setupKeyPairWithServices("user", "secret", []Service{
		{"nova", "compute", []Endpoint{
			{PublicURL: compute_url},
		}}})
	res, err := keyPairAuthRequest(s.Server.URL, "user", "secret")
	defer res.Body.Close()
	c.Assert(err, gc.IsNil)
	c.Check(res.StatusCode, gc.Equals, http.StatusOK)
	c.Check(res.Header.Get("Content-Type"), gc.Equals, "application/json")
	content, err := ioutil.ReadAll(res.Body)
	c.Assert(err, gc.IsNil)
	var response AccessResponse
	err = json.Unmarshal(content, &response)
	c.Assert(err, gc.IsNil)
	c.Check(response.Access.Token.Id, gc.NotNil)
	novaURL := ""
	for _, service := range response.Access.ServiceCatalog {
		if service.Type == "compute" {
			novaURL = service.Endpoints[0].PublicURL
			break
		}
	}
	c.Assert(novaURL, gc.Equals, compute_url)
}
