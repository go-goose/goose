package client_test

import (
	"fmt"
	"net/http"
	"strconv"

	gc "gopkg.in/check.v1"
)

type versionHandler struct {
	authBody string
	port     string
}

type makeServiceURLTest struct {
	serviceType string
	version     string
	parts       []string
	success     bool
	URL         string
	err         string
}

func (s *localLiveSuite) makeServiceURLTests() []makeServiceURLTest {
	return []makeServiceURLTest{
		{
			// As a special case, if no version is specified
			// then we use whatever URL is recoded in the
			// service catalogue verbatim.
			serviceType: "compute",
			version:     "",
			parts:       []string{},
			success:     true,
			URL:         "http://localhost:%s",
		},
		{
			serviceType: "compute",
			version:     "v2.1",
			parts:       []string{"foo", "bar/"},
			success:     true,
			URL:         "http://localhost:%s/v2.1/foo/bar/",
		},
		{
			serviceType: "compute",
			version:     "v2.0",
			parts:       []string{},
			success:     true,
			URL:         "http://localhost:%s/v2.0",
		},
		{
			serviceType: "compute",
			version:     "v2.0",
			parts:       []string{"foo", "bar/"},
			success:     true,
			URL:         "http://localhost:%s/v2.0/foo/bar/",
		},
		{
			serviceType: "compute",
			version:     "v2",
			parts:       []string{"foo", "bar/"},
			success:     true,
			URL:         "http://localhost:%s/v2.1/foo/bar/",
		},
		{
			serviceType: "object-store",
			version:     "",
			parts:       []string{"foo", "bar"},
			success:     true,
			URL:         "http://localhost:%s/swift/v1/foo/bar",
		},
		{
			serviceType: "object-store",
			version:     "q2.0",
			parts:       []string{"foo", "bar/"},
			success:     false,
			err:         "strconv.Atoi: parsing \"q2\": invalid syntax",
		},
		{
			serviceType: "object-store",
			version:     "v1.7",
			parts:       []string{"foo", "bar/"},
			success:     false,
			err:         "could not find matching URL",
		},
		{
			serviceType: "juju-container-test",
			version:     "v1",
			parts:       []string{"foo", "bar/"},
			success:     true,
			URL:         "http://localhost:%s/swift/v1/foo/bar/",
		},
		{
			serviceType: "juju-container-test",
			version:     "v0",
			parts:       []string{"foo", "bar/"},
			success:     false,
			err:         "could not find matching URL",
		},
		{
			serviceType: "juju-container-test",
			version:     "",
			parts:       []string{"foo", "bar/"},
			success:     true,
			URL:         "http://localhost:%s/swift/v1/foo/bar/",
		},
		{
			serviceType: "compute",
			version:     "v25.4",
			parts:       []string{},
			success:     false,
			err:         "could not find matching URL",
		},
		{
			serviceType: "no-such-service",
			version:     "",
			parts:       []string{},
			success:     false,
			err:         "no endpoints known for service type: no-such-service",
		},
		{
			// See https://bugs.launchpad.net/juju/+bug/1756135
			serviceType: "compute2",
			version:     "v2.1",
			parts:       []string{"servers/"},
			success:     true,
			URL:         "http://localhost:%s/v2.1/010ab46135ba414882641f663ec917b6/servers/",
		},
		{
			// See https://bugs.launchpad.net/juju/+bug/1756135
			serviceType: "compute3",
			version:     "v4.2",
			parts:       []string{"servers/"},
			success:     true,
			URL:         "http://localhost:%s/compute/v4.2/servers/",
		},
		{
			// See https://bugs.launchpad.net/juju/+bug/1756135
			serviceType: "compute3",
			version:     "v2",
			parts:       []string{"servers/"},
			success:     true,
			URL:         "http://localhost:%s/compute/v2.1/servers/",
		},
	}
}

func (s *localLiveSuite) TestMakeServiceURL(c *gc.C) {
	port := "3000"
	cl := s.assertAuthenticationSuccess(c, port)
	tests := s.makeServiceURLTests()
	testCount := len(tests)
	for i, t := range tests {
		c.Logf("#%d of %d. %s %s %v", i+1, testCount, t.serviceType, t.version, t.parts)
		URL, err := cl.MakeServiceURL(t.serviceType, t.version, t.parts)
		if t.success {
			c.Assert(err, gc.IsNil)
			c.Assert(URL, gc.Equals, fmt.Sprintf(t.URL, port))
			// Run twice to ensure the version caching is working
			URL, err = cl.MakeServiceURL(t.serviceType, t.version, t.parts)
			c.Assert(err, gc.IsNil)
			c.Assert(URL, gc.Equals, fmt.Sprintf(t.URL, port))
		} else {
			c.Assert(err, gc.ErrorMatches, t.err)
		}
	}
}

func (s *localLiveSuite) TestMakeServiceURLAPIVersionDiscoveryDisabled(c *gc.C) {
	port := "3000"
	cl := s.assertAuthenticationSuccess(c, port)
	wasEnabled := cl.SetVersionDiscoveryEnabled(false)
	c.Assert(wasEnabled, gc.Equals, true)

	url, err := cl.MakeServiceURL("compute", "v2.1", []string{"foo", "bar/"})
	c.Assert(err, gc.IsNil)
	c.Assert(url, gc.Equals, fmt.Sprintf("http://localhost:%s/foo/bar/", port))
}

func (s *localLiveSuite) TestMakeServiceURLNoAPIVersionEndpoint(c *gc.C) {
	// See https://bugs.launchpad.net/juju/+bug/1638304
	// Some OpenStacks don't support version discovery.

	port := "3005"
	cl := s.assertAuthenticationSuccess(c, port)

	url, err := cl.MakeServiceURL("compute", "v2.1", []string{"foo", "bar/"})
	c.Assert(err, gc.IsNil)
	c.Assert(url, gc.Equals, fmt.Sprintf("http://localhost:%s/foo/bar/", port))
}

func (s *localLiveSuite) TestMakeServiceURLValues(c *gc.C) {
	port := "3003"
	cl := s.assertAuthenticationSuccess(c, port)
	tests := s.makeServiceURLTests()
	testCount := len(tests)
	for i, t := range tests {
		c.Logf("#%d of %d. %s %s %v", i+1, testCount, t.serviceType, t.version, t.parts)
		URL, err := cl.MakeServiceURL(t.serviceType, t.version, t.parts)
		if t.success {
			c.Assert(err, gc.IsNil)
			c.Assert(URL, gc.Equals, fmt.Sprintf(t.URL, port))
			// Run twice to ensure the version caching is working
			URL, err = cl.MakeServiceURL(t.serviceType, t.version, t.parts)
			c.Assert(err, gc.IsNil)
			c.Assert(URL, gc.Equals, fmt.Sprintf(t.URL, port))
		} else {
			c.Assert(err, gc.ErrorMatches, t.err)
		}
	}
}

const authInformationBody = `{"versions": [` +
	`{"status": "stable", "updated": "2015-03-30T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v3+json"}], "id": "v3.4", "links": [{"href": "%s/v3/", "rel": "self"}]},` +
	`{"status": "stable", "updated": "2014-04-17T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v2.0+json"}], "id": "v2.0", "links": [{"href": "%s/v2.0/", "rel": "self"}, {"href": "http://docs.openstack.org/", "type": "text/html", "rel": "describedby"}]},` +
	`{"status": "stable", "updated": "2015-03-30T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v3+json"}], "id": "v2.1", "links": [{"href": "%s/v2.1/", "rel": "self"}]},` +
	`{"status": "stable", "updated": "2015-03-30T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v3+json"}], "id": "v4.2", "links": [{"href": "%s/compute/v4.2/", "rel": "self"}]}` +
	`]}`

const authValuesInformationBody = `{"versions": {"values": [` +
	`{"status": "stable", "updated": "2015-03-30T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v3+json"}], "id": "v3.4", "links": [{"href": "%s/v3/", "rel": "self"}]},` +
	`{"status": "stable", "updated": "2014-04-17T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v2.0+json"}], "id": "v2.0", "links": [{"href": "%s/v2.0/", "rel": "self"}, {"href": "http://docs.openstack.org/", "type": "text/html", "rel": "describedby"}]},` +
	`{"status": "stable", "updated": "2015-03-30T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v3+json"}], "id": "v2.1", "links": [{"href": "%s/v2.1/", "rel": "self"}]},` +
	`{"status": "stable", "updated": "2015-03-30T00:00:00Z", "media-types": [{"base": "application/json", "type": "application/vnd.openstack.identity-v3+json"}], "id": "v4.2", "links": [{"href": "%s/compute/v4.2/", "rel": "self"}]}` +
	`]}}`

func (vh *versionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if vh.authBody == "" {
		body := `{"message":"Api does not exist","request_id":"83A781AE-9A0C-43C7-B405-310A5A94566E"}`
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
		return
	}
	body := []byte(fmt.Sprintf(vh.authBody, "http://localhost:"+vh.port, "http://localhost:"+vh.port, "http://localhost:"+vh.port, "http://localhost:"+vh.port))
	// workaround for https://code.google.com/p/go/issues/detail?id=4454
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusMultipleChoices)
	w.Write(body)
}

func startApiVersionMux(vh versionHandler) string {
	mux := http.NewServeMux()

	mux.Handle("/", &vh)

	go http.ListenAndServe(":"+vh.port, mux)
	return fmt.Sprintf("Listening on localhost:%s...\n", vh.port)
}
