package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	goosehttp "gopkg.in/goose.v1/http"
)

type apiVersion struct {
	major int
	minor int
}

type apiVersionInfo struct {
	Version apiVersion       `json:"id"`
	Links   []apiVersionLink `json:"links"`
	Status  string           `json:"status"`
}

type apiVersionLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type apiURLVersion struct {
	rootURL          url.URL
	serviceURLSuffix string
	versions         []apiVersionInfo
}

// getAPIVersionURL returns a full formed serviceURL based on the API version requested,
// the rootURL and the serviceURLSuffix.  If there is no match to the requested API
// version an error is returned.  If only the major number is defined for the requested
// version, the first match found is returned.
func getAPIVersionURL(apiURLVersionInfo *apiURLVersion, requested apiVersion) (string, error) {
	var match string
	for _, v := range apiURLVersionInfo.versions {
		if v.Version.major != requested.major {
			continue
		}
		if requested.minor != -1 && v.Version.minor != requested.minor {
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
		if requested.minor != -1 {
			break
		}
	}
	if match == "" {
		return "", fmt.Errorf("could not find matching URL")
	}
	versionURL := apiURLVersionInfo.rootURL
	versionURL.Path = path.Join(versionURL.Path, match, apiURLVersionInfo.serviceURLSuffix)
	return versionURL.String(), nil
}

func (v *apiVersion) UnmarshalJSON(b []byte) error {
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

// parseVersion takes a version string into the major and minor ints for an apiVersion
// structure. The string part of the data is returned by a request to List API versions
// send to an OpenStack service.  It is in the format "v<major>.<minor>". If apiVersion
// is empty, return {-1, -1}, to differentiate with "v0".
func parseVersion(s string) (apiVersion, error) {
	if s == "" {
		return apiVersion{-1, -1}, nil
	}
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 0 || len(parts) > 2 {
		return apiVersion{}, fmt.Errorf("invalid API version %q", s)
	}
	var minor int = -1
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return apiVersion{}, err
	}
	if len(parts) == 2 {
		var err error
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return apiVersion{}, err
		}
	}
	return apiVersion{major, minor}, nil
}

func unmarshallVersion(Versions json.RawMessage) ([]apiVersionInfo, error) {
	// Some services respond with {"versions":[...]}, and
	// some respond with {"versions":{"values":[...]}}.
	var object interface{}
	var versions []apiVersionInfo
	if err := json.Unmarshal(Versions, &object); err != nil {
		return versions, err
	}
	if _, ok := object.(map[string]interface{}); ok {
		var valuesObject struct {
			Values []apiVersionInfo `json:"values"`
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

// getAPIVersions returns data on the API versions supported by the specified service endpoint.
func (c *authenticatingClient) getAPIVersions(serviceCatalogURL string) (*apiURLVersion, error) {
	url, err := url.Parse(serviceCatalogURL)
	if err != nil {
		return nil, err
	}

	// Identify the version in the URL, if there is one, and record
	// everything proceeding it. We will need to append this to the
	// API version-specific base URL.
	var pathParts, origPathParts []string
	if url.Path != "/" {
		// The rackspace object-store endpoint triggers the version removal here,
		// so keep the original parts for object-store and container endpoints.
		// e.g. https://storage101.dfw1.clouddrive.com/v1/MossoCloudFS_1019383
		origPathParts = strings.Split(strings.Trim(url.Path, "/"), "/")
		pathParts = origPathParts
		if _, err := parseVersion(pathParts[0]); err == nil {
			pathParts = pathParts[1:]
		}
		url.Path = "/"
	}

	baseURL := url.String()

	// If this is an object-store serviceType, or an object-store container endpoint,
	// there is no list version API call to make. Return a apiURLVersion which will
	// satisfy a requested api version of "", "v1" or "v1.0"
	if c.serviceURLs["object-store"] != "" && strings.Contains(serviceCatalogURL, c.serviceURLs["object-store"]) {
		objectStoreLink := apiVersionLink{Href: baseURL, Rel: "self"}
		objectStoreApiVersionInfo := []apiVersionInfo{
			{
				Version: apiVersion{major: 1, minor: 0},
				Links:   []apiVersionLink{objectStoreLink},
				Status:  "stable",
			},
			{
				Version: apiVersion{major: -1, minor: -1},
				Links:   []apiVersionLink{objectStoreLink},
				Status:  "stable",
			},
		}
		return &apiURLVersion{*url, strings.Join(origPathParts, "/"), objectStoreApiVersionInfo}, nil
	}

	// make sure we haven't already received the version info
	c.apiVersionMu.Lock()
	defer c.apiVersionMu.Unlock()
	if apiInfo, ok := c.apiURLVersions[baseURL]; ok {
		return apiInfo, nil
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
	if err := c.sendRequest("GET", baseURL, c.Token(), requestData); err != nil {
		return nil, err
	}

	versions, err := unmarshallVersion(raw.Versions)
	if err != nil {
		return nil, err
	}

	// save this info, so we don't have to get it again
	apiURLVersionInfo := &apiURLVersion{*url, strings.Join(pathParts, "/"), versions}
	c.apiURLVersions[baseURL] = apiURLVersionInfo

	return apiURLVersionInfo, nil
}
