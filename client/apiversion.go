package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	goosehttp "github.com/go-goose/goose/v5/http"
	"github.com/go-goose/goose/v5/logging"
)

// ApiVersion represents choices.id from the openstack
// api version  Multiple choices JSON response, broken
// into  major and minor from the string.
type ApiVersion struct {
	Major int
	Minor int
}

// ApiVersionInfo represents choices from the openstack
// api version Multiple choices JSON response.
type ApiVersionInfo struct {
	Version ApiVersion       `json:"id"`
	Links   []ApiVersionLink `json:"links"`
	Status  string           `json:"status"`
}

// ApiVersionLink represents choices.links from the openstack
// api version  Multiple choices JSON response.
type ApiVersionLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type apiURLVersion struct {
	rootURL          url.URL
	serviceURLSuffix string
	versions         []ApiVersionInfo
}

// getAPIVersionURL returns a full formed serviceURL based on the API version requested,
// the rootURL and the serviceURLSuffix.  If there is no match to the requested API
// version an error is returned.  If only the Major number is defined for the requested
// version, the first match found is returned.
func (c *authenticatingClient) getAPIVersionURL(apiURLVersionInfo *apiURLVersion, requested ApiVersion) (string, error) {
	var match string
	for _, v := range apiURLVersionInfo.versions {
		if v.Version.Major != requested.Major {
			continue
		}
		if requested.Minor != -1 && v.Version.Minor != requested.Minor {
			continue
		}
		for _, link := range v.Links {
			if link.Rel != "self" {
				continue
			}
			hrefURL, err := url.Parse(link.Href)
			if err != nil {
				return "", err
			}
			match = hrefURL.Path
		}
		if requested.Minor != -1 {
			break
		}
	}
	if match == "" {
		return "", fmt.Errorf("could not find matching URL")
	}
	versionURL := apiURLVersionInfo.rootURL

	// https://bugs.launchpad.net/juju/+bug/1756135:
	// some hrefURL.Path contain more than the version, with
	// overlap on the apiURLVersionInfo.rootURL
	if strings.HasPrefix(match, "/"+versionURL.Path) {
		logger := logging.FromCompat(c.logger)
		logger.Tracef("version href path %q overlaps with url path %q, using version href", match, versionURL.Path)
		versionURL.Path = "/"
	}

	versionURL.Path = path.Join(versionURL.Path, match, apiURLVersionInfo.serviceURLSuffix)
	return versionURL.String(), nil
}

func (v *ApiVersion) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := parseVersion(s)
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

// parseVersion takes a version string into the Major and Minor ints for an ApiVersion
// structure. The string part of the data is returned by a request to List API versions
// send to an OpenStack service.  It is in the format "v<Major>.<Minor>". If ApiVersion
// is empty, return {-1, -1}, to differentiate with "v0".
func parseVersion(s string) (ApiVersion, error) {
	if s == "" {
		return ApiVersion{-1, -1}, nil
	}
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 0 || len(parts) > 2 {
		return ApiVersion{}, fmt.Errorf("invalid API version %q", s)
	}
	var minor int = -1
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ApiVersion{}, err
	}
	if len(parts) == 2 {
		var err error
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return ApiVersion{}, err
		}
	}
	return ApiVersion{major, minor}, nil
}

func unmarshallVersion(Versions json.RawMessage) ([]ApiVersionInfo, error) {
	// Some services respond with {"versions":[...]}, and
	// some respond with {"versions":{"values":[...]}}.
	var object interface{}
	var versions []ApiVersionInfo
	if err := json.Unmarshal(Versions, &object); err != nil {
		return versions, err
	}
	if _, ok := object.(map[string]interface{}); ok {
		var valuesObject struct {
			Values []ApiVersionInfo `json:"values"`
		}
		if err := json.Unmarshal(Versions, &valuesObject); err != nil {
			return versions, err
		}
		versions = valuesObject.Values
	} else {
		if err := json.Unmarshal(Versions, &versions); err != nil {
			return versions, err
		}
	}
	return versions, nil
}

// getAPIVersions returns data on the API versions supported by the specified
// service endpoint. Some OpenStack clouds do not support the version endpoint,
// in which case this method will return an empty set of versions in the result
// structure.
func (c *authenticatingClient) getAPIVersions(serviceCatalogURL string) (*apiURLVersion, error) {
	c.apiVersionMu.Lock()
	defer c.apiVersionMu.Unlock()
	logger := logging.FromCompat(c.logger)

	// Make sure we haven't already received the version info.
	// Cache done on serviceCatalogURL, https://<url.Host> is not
	// guaranteed to be unique.
	if apiInfo, ok := c.apiURLVersions[serviceCatalogURL]; ok {
		return apiInfo, nil
	}

	url, err := url.Parse(serviceCatalogURL)
	if err != nil {
		return nil, err
	}

	// Identify the version in the URL, if there is one, and record
	// everything proceeding it. We will need to append this to the
	// API version-specific base URL.
	var pathParts, origPathParts []string
	if url.Path != "/" {
		// If a version is included in the serviceCatalogURL, the
		// part before the version will end up in url, the part after
		// the version will end up in pathParts.  origPathParts is a
		// special case for "object-store"
		// e.g. https://storage101.dfw1.clouddrive.com/v1/MossoCloudFS_1019383
		// 		becomes: https://storage101.dfw1.clouddrive.com/ and MossoCloudFS_1019383
		// https://x.x.x.x/image
		// 		becomes: https://x.x.x.x/image/
		// https://x.x.x.x/cloudformation/v1
		// 		becomes: https://x.x.x.x/cloudformation/
		// https://x.x.x.x/compute/v2/9032a0051293421eb20b64da69d46252
		// 		becomes: https://x.x.x.x/compute/ and 9032a0051293421eb20b64da69d46252
		// https://x.x.x.x/volumev1/v2
		// 		becomes: https://x.x.x.x/volumev1/
		// http://y.y.y.y:9292
		// 		becomes: http://y.y.y.y:9292/
		// http://y.y.y.y:8774/v2/010ab46135ba414882641f663ec917b6
		//		becomes: http://y.y.y.y:8774/ and 010ab46135ba414882641f663ec917b6
		origPathParts = strings.Split(strings.Trim(url.Path, "/"), "/")
		pathParts = origPathParts
		found := false
		for i, p := range pathParts {
			if _, err := parseVersion(p); err == nil {
				found = true
				if i == 0 {
					pathParts = pathParts[1:]
					url.Path = "/"
				} else {
					url.Path = pathParts[0] + "/"
					pathParts = pathParts[2:]
				}
				break
			}
		}
		if !found {
			url.Path = path.Join(pathParts...) + "/"
			pathParts = []string{}
		}
	}
	logger.Tracef("api version will be inserted between %q and %q", url.String(), path.Join(pathParts...)+"/")

	getVersionURL := url.String()

	// If this is an object-store serviceType, or an object-store container endpoint,
	// there is no list version API call to make. Return a apiURLVersion which will
	// satisfy a requested api version of "", "v1" or "v1.0"
	if c.serviceURLs["object-store"] != "" && strings.Contains(serviceCatalogURL, c.serviceURLs["object-store"]) {
		url.Path = "/"
		objectStoreLink := ApiVersionLink{Href: url.String(), Rel: "self"}
		objectStoreApiVersionInfo := []ApiVersionInfo{
			{
				Version: ApiVersion{Major: 1, Minor: 0},
				Links:   []ApiVersionLink{objectStoreLink},
				Status:  "stable",
			},
			{
				Version: ApiVersion{Major: -1, Minor: -1},
				Links:   []ApiVersionLink{objectStoreLink},
				Status:  "stable",
			},
		}
		apiURLVersionInfo := &apiURLVersion{*url, strings.Join(origPathParts, "/"), objectStoreApiVersionInfo}
		c.apiURLVersions[serviceCatalogURL] = apiURLVersionInfo
		return apiURLVersionInfo, nil
	}

	var raw struct {
		Versions json.RawMessage `json:"versions"`
	}
	requestData := &goosehttp.RequestData{
		RespValue: &raw,
		ExpectedStatus: []int{
			http.StatusOK,
			http.StatusMultipleChoices,
		},
	}
	apiURLVersionInfo := &apiURLVersion{
		rootURL:          *url,
		serviceURLSuffix: strings.Join(pathParts, "/"),
	}
	if err := c.sendRequest("GET", getVersionURL, c.Token(), requestData); err != nil {
		logger.Warningf("API version discovery failed: %v", err)
		c.apiURLVersions[serviceCatalogURL] = apiURLVersionInfo
		return apiURLVersionInfo, nil
	}

	versions, err := unmarshallVersion(raw.Versions)
	if err != nil {
		logger.Debugf("API version discovery unmarshallVersion failed: %v", err)
		c.apiURLVersions[serviceCatalogURL] = apiURLVersionInfo
		return apiURLVersionInfo, nil
	}
	apiURLVersionInfo.versions = versions
	logger.Debugf("discovered API versions: %+v", versions)

	// Cache the result.
	c.apiURLVersions[serviceCatalogURL] = apiURLVersionInfo

	return apiURLVersionInfo, nil
}
