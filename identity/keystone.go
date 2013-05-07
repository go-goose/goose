package identity

import (
	"fmt"
	goosehttp "launchpad.net/goose/http"
)

// Authenticate to OpenStack cloud using keystone v2 authentication.
//
// Uses `client` to submit HTTP requests to `URL`
// and posts `auth_data` as JSON.
func keystoneAuth(client *goosehttp.Client, auth_data interface{}, URL string) (*AuthDetails, error) {

	var accessWrapper accessWrapper
	requestData := goosehttp.RequestData{ReqValue: auth_data, RespValue: &accessWrapper}
	err := client.JsonRequest("POST", URL, "", &requestData, nil)
	if err != nil {
		return nil, err
	}

	details := &AuthDetails{}
	access := accessWrapper.Access
	respToken := access.Token
	if respToken.Id == "" {
		return nil, fmt.Errorf("authentication failed")
	}
	details.Token = respToken.Id
	details.TenantId = respToken.Tenant.Id
	details.UserId = access.User.Id
	details.RegionServiceURLs = make(map[string]ServiceURLs, len(access.ServiceCatalog))
	for _, service := range access.ServiceCatalog {
		for i, e := range service.Endpoints {
			endpointURLs, ok := details.RegionServiceURLs[e.Region]
			if !ok {
				endpointURLs = make(ServiceURLs)
				details.RegionServiceURLs[e.Region] = endpointURLs
			}
			endpointURLs[service.Type] = service.Endpoints[i].PublicURL
		}
	}
	return details, nil
}
