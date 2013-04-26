package identity

import (
	"fmt"
	goosehttp "launchpad.net/goose/http"
)

type KeyPair struct {
	client *goosehttp.Client
}

type keypairCredentials struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type authKeypairRequest struct {
	KeypairCredentials keypairCredentials `json:"apiAccessKeyCredentials"`
	TenantName          string              `json:"tenantName"`
}

type authKeypairWrapper struct {
	Auth authKeypairRequest `json:"auth"`
}

func (u *KeyPair) Auth(creds *Credentials) (*AuthDetails, error) {
	if u.client == nil {
		u.client = goosehttp.New()
	}
	auth := authKeypairWrapper{Auth: authKeypairRequest{
		KeypairCredentials: keypairCredentials{
			AccessKey: creds.User,
			SecretKey: creds.Secrets,
		},
		TenantName: creds.TenantName}}

	var accessWrapper accessWrapper
	requestData := goosehttp.RequestData{ReqValue: auth, RespValue: &accessWrapper}
	err := u.client.JsonRequest("POST", creds.URL, "", &requestData, nil)
	if err != nil {
		return nil, err
	}

	details := &AuthDetails{}
	access := accessWrapper.Access
	respToken := access.Token
	if respToken.Id == "" {
		return nil, fmt.Errorf("Did not get valid Token from auth request")
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
