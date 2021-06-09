package client_test

import (
	"encoding/json"
	"net/http"

	"github.com/golang/mock/gomock"
	gc "gopkg.in/check.v1"

	"github.com/go-goose/goose/v3/client"
	"github.com/go-goose/goose/v3/client/mocks"
	"github.com/go-goose/goose/v3/errors"
	goosehttp "github.com/go-goose/goose/v3/http"
	"github.com/go-goose/goose/v3/identity"
)

type localMockSuite struct {
	creds       *identity.Credentials
	authDetails *identity.AuthDetails

	authenticator   *mocks.MockAuthenticator
	gooseHttpClient *mocks.MockHttpClient
	logger          *mocks.MockCompatLogger
}

var _ = gc.Suite(&localMockSuite{})

func (s *localMockSuite) SetUpTest(c *gc.C) {
	s.creds = &identity.Credentials{
		URL:           "http://localhost/v3",
		User:          "fred",
		Secrets:       "secret",
		Region:        "RegionOne",
		TenantName:    "tenant",
		ProjectDomain: "default",
	}

	URLs := make(map[string]identity.ServiceURLs)
	endpoints := make(map[string]string)
	endpoints["compute"] = "http://localhost/compute/v2.1/project_uuid"
	endpoints["object-store"] = "http://localhost/swift/v1"
	endpoints["network"] = "http://localhost/network"
	URLs[s.creds.Region] = endpoints

	s.authDetails = &identity.AuthDetails{
		Token:             "token",
		TenantId:          "tenant",
		UserId:            "1",
		RegionServiceURLs: URLs,
	}
}

func (s *localMockSuite) TestSendRequest(c *gc.C) {
	defer s.setup(c).Finish()

	s.expectAuthentication()
	s.expectJsonRequestComputeAPIVersionDiscoverySuccess(c)
	s.expectBinaryRequestCompute()

	cl := s.newClient()
	client.SetAuthenticator(cl, s.authenticator)

	err := cl.SendRequest(client.POST, "compute", "v2", "flavor/detail", &goosehttp.RequestData{})
	c.Assert(err, gc.IsNil)
}

func (s *localMockSuite) TestSendRequestHandleMultipleChoicesError(c *gc.C) {
	defer s.setup(c).Finish()

	s.expectAuthentication()
	s.expectJsonRequestComputeAPIVersionDiscoveryMultipleChoice(c)
	s.expectBinaryRequestComputeMultipleChoicesError()

	cl := s.newClient()
	client.SetAuthenticator(cl, s.authenticator)

	err := cl.SendRequest(client.POST, "compute", "v2", "flavor/detail", &goosehttp.RequestData{})
	c.Assert(err, gc.IsNil)
}

func (s *localMockSuite) TestSendRequestHandleOneServiceFallback(c *gc.C) {
	defer s.setup(c).Finish()

	s.expectAuthentication()
	s.expectJsonRequestComputeAPIVersionDiscoveryMultipleChoice(c)
	s.expectJsonRequestNetworkAPIVersionDiscoverySuccess(c)
	s.expectBinaryRequestComputeMultipleChoicesError()
	s.expectBinaryRequestNetwork()

	cl := s.newClient()
	client.SetAuthenticator(cl, s.authenticator)

	err := cl.SendRequest(client.POST, "compute", "v2", "flavor/detail", &goosehttp.RequestData{})
	c.Assert(err, gc.IsNil)

	// Now ensure that the network service didn't fall back to the
	// service catalogue url too.
	err = cl.SendRequest(client.POST, "network", "v2", "networks", &goosehttp.RequestData{})
	c.Assert(err, gc.IsNil)
}

func (s *localMockSuite) setup(c *gc.C) *gomock.Controller {
	ctrl := gomock.NewController(c)

	s.authenticator = mocks.NewMockAuthenticator(ctrl)
	s.gooseHttpClient = mocks.NewMockHttpClient(ctrl)

	s.logger = mocks.NewMockCompatLogger(ctrl)
	printIt := func(message string, args ...interface{}) { c.Logf(message, args) }
	s.logger.EXPECT().Printf(gomock.Any(), gomock.Any()).AnyTimes().Do(printIt).AnyTimes()

	return ctrl
}

func (s *localMockSuite) newClient() client.AuthenticatingClient {
	return client.NewClientForTest(s.creds, identity.AuthUserPass, s.gooseHttpClient, s.logger)
}

func (s *localMockSuite) expectAuthentication() {
	s.authenticator.EXPECT().Auth(gomock.Any()).Return(s.authDetails, nil)
}

func (s *localMockSuite) expectBinaryRequestCompute() {
	gExp := s.gooseHttpClient.EXPECT()
	gExp.BinaryRequest(client.POST, "http://localhost/compute/v2.1/project_uuid/flavor/detail", "token", gomock.Any(), gomock.Any())
}

func (s *localMockSuite) expectBinaryRequestNetwork() {
	gExp := s.gooseHttpClient.EXPECT()
	gExp.BinaryRequest(client.POST, "http://localhost/network/v2.0/networks", "token", gomock.Any(), gomock.Any())
}

func (s *localMockSuite) expectBinaryRequestComputeMultipleChoicesError() {
	retErr := errors.NewMultipleChoicesf(&goosehttp.HttpError{StatusCode: http.StatusMultipleChoices}, "", "")
	gExp := s.gooseHttpClient.EXPECT()
	one := gExp.BinaryRequest(client.POST, "http://localhost/compute/compute/v2.1/project_uuid/flavor/detail", "token", gomock.Any(), gomock.Any()).Return(retErr)
	gExp.BinaryRequest(client.POST, "http://localhost/compute/v2.1/project_uuid/flavor/detail", "token", gomock.Any(), gomock.Any()).After(one)
}

type testApiVersionInfo struct {
	Id     string                  `json:"id"`
	Links  []client.ApiVersionLink `json:"links"`
	Status string                  `json:"status"`
}

type valuesObject struct {
	Values []testApiVersionInfo `json:"values"`
}

func (s *localMockSuite) expectJsonRequestNetworkAPIVersionDiscoverySuccess(c *gc.C) {
	versions := []testApiVersionInfo{
		{
			Status: "supported",
			Id:     "v2.0",
			Links: []client.ApiVersionLink{
				{Href: "http://localhost/network/v2.0/", Rel: "self"},
			},
		},
	}
	s.expectJsonRequestComputeAPIVersionDiscovery("http://localhost/network/", versions, c)
}

func (s *localMockSuite) expectJsonRequestComputeAPIVersionDiscoverySuccess(c *gc.C) {
	versions := []testApiVersionInfo{
		{
			Status: "supported",
			Id:     "v2.0",
			Links: []client.ApiVersionLink{
				{Href: "http://localhost/compute/v2.0/", Rel: "self"},
			},
		},
		{
			Status: "current",
			Id:     "v2.1",
			Links: []client.ApiVersionLink{
				{Href: "http://localhost/compute/v2.1/", Rel: "self"},
			},
		},
	}
	s.expectJsonRequestComputeAPIVersionDiscovery("http://localhost/compute/", versions, c)
}

func (s *localMockSuite) expectJsonRequestComputeAPIVersionDiscoveryMultipleChoice(c *gc.C) {
	versions := []testApiVersionInfo{
		{
			Status: "supported",
			Id:     "v2.0",
			Links: []client.ApiVersionLink{
				{Href: "http://localhost/compute/compute/v2.0/", Rel: "self"},
			},
		},
		{
			Status: "current",
			Id:     "v2.1",
			Links: []client.ApiVersionLink{
				{Href: "http://localhost/compute/compute/v2.1/", Rel: "self"},
			},
		},
	}
	s.expectJsonRequestComputeAPIVersionDiscovery("http://localhost/compute/", versions, c)
}

func (s *localMockSuite) expectJsonRequestComputeAPIVersionDiscovery(url string, versions []testApiVersionInfo, c *gc.C) {
	do := func(_, _, _ string, reqData *goosehttp.RequestData, _ interface{}) {
		raw, ok := reqData.RespValue.(*struct {
			Versions json.RawMessage "json:\"versions\""
		})
		c.Assert(ok, gc.Equals, true)

		object := valuesObject{Values: versions}
		js, err := json.Marshal(object)
		c.Assert(err, gc.IsNil)
		raw.Versions = js
	}
	gExp := s.gooseHttpClient.EXPECT()
	gExp.JsonRequest(client.GET, url, "token", gomock.Any(), gomock.Any()).Return(nil).Do(do)
}
