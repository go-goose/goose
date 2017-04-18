package identityservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v2/testing/httpsuite"
)

type V3UserPassSuite struct {
	httpsuite.HTTPSuite
}

var _ = gc.Suite(&V3UserPassSuite{})

func makeV3UserPass(user, secret string) (identity *V3UserPass) {
	identity = NewV3UserPass()
	// Ensure that it conforms to the interface
	var _ IdentityService = identity
	if user != "" {
		identity.AddUser(user, secret, "tenant", "default")
	}
	return
}

func (s *V3UserPassSuite) setupUserPass(user, secret string) {
	var identity *V3UserPass
	identity = makeV3UserPass(user, secret)
	identity.SetupHTTP(s.Mux)
	return
}

func (s *V3UserPassSuite) setupUserPassWithServices(user, secret string, services []Service) {
	var identity *V3UserPass
	identity = makeV3UserPass(user, secret)
	for _, service := range services {
		identity.AddService(service)
	}
	identity.SetupHTTP(s.Mux)
	return
}

var v3AuthTemplate = `{
    "auth": {
        "identity": {
          "method":["password"],
          "password": {
              "user": {
                "domain": {
                  "id": "default"
                },
                "name": "%s", 
                "password": "%s"
              }     
          }
        }
    }
}`

func v3UserPassAuthRequest(URL, user, key string) (*http.Response, error) {
	client := http.DefaultClient
	body := strings.NewReader(fmt.Sprintf(v3AuthTemplate, user, key))
	request, err := http.NewRequest("POST", URL+"/v3/auth/tokens", body)
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}

func (s *V3UserPassSuite) TestNotJSON(c *gc.C) {
	// We do everything in userPassAuthRequest, except set the Content-Type
	s.setupUserPass("user", "secret")
	client := http.DefaultClient
	body := strings.NewReader(fmt.Sprintf(authTemplate, "user", "secret"))
	request, err := http.NewRequest("POST", s.Server.URL+"/v3/auth/tokens", body)
	c.Assert(err, gc.IsNil)
	res, err := client.Do(request)
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	CheckErrorResponse(c, res, http.StatusBadRequest, notJSON)
}

func (s *V3UserPassSuite) TestBadJSON(c *gc.C) {
	// We do everything in userPassAuthRequest, except set the Content-Type
	s.setupUserPass("user", "secret")
	res, err := v3UserPassAuthRequest(s.Server.URL, "garbage\"in", "secret")
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	CheckErrorResponse(c, res, http.StatusBadRequest, notJSON)
}

func (s *V3UserPassSuite) TestNoSuchUser(c *gc.C) {
	s.setupUserPass("user", "secret")
	res, err := v3UserPassAuthRequest(s.Server.URL, "not-user", "secret")
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	CheckErrorResponse(c, res, http.StatusUnauthorized, notAuthorized)
}

func (s *V3UserPassSuite) TestBadPassword(c *gc.C) {
	s.setupUserPass("user", "secret")
	res, err := v3UserPassAuthRequest(s.Server.URL, "user", "not-secret")
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	CheckErrorResponse(c, res, http.StatusUnauthorized, invalidUser)
}

func (s *V3UserPassSuite) TestValidAuthorization(c *gc.C) {
	compute_url := "http://testing.invalid/compute"
	s.setupUserPassWithServices("user", "secret", []Service{
		{V3: V3Service{Name: "nova", Type: "compute", Endpoints: NewV3Endpoints("", "", compute_url, "")}}})
	res, err := v3UserPassAuthRequest(s.Server.URL, "user", "secret")
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	c.Check(res.StatusCode, gc.Equals, http.StatusCreated)
	c.Check(res.Header.Get("Content-Type"), gc.Equals, "application/json")
	content, err := ioutil.ReadAll(res.Body)
	c.Assert(err, gc.IsNil)
	var response struct {
		Token V3TokenResponse `json:"token"`
	}
	err = json.Unmarshal(content, &response)
	c.Assert(err, gc.IsNil)
	c.Check(res.Header.Get("X-Subject-Token"), gc.Not(gc.Equals), "")
	novaURL := ""
	for _, service := range response.Token.Catalog {
		if service.Type == "compute" {
			for _, ep := range service.Endpoints {
				if ep.Interface == "public" {
					novaURL = ep.URL
					break
				}
			}
			break
		}
	}
	c.Assert(novaURL, gc.Equals, compute_url)
}
