package identity

import (
	"io/ioutil"
	"net/http"

	"fmt"
	gooseerrors "gopkg.in/goose.v2/errors"
	goosehttp "gopkg.in/goose.v2/http"
)

type Legacy struct {
	client *goosehttp.Client
}

func (l *Legacy) Auth(creds *Credentials) (*AuthDetails, error) {
	if l.client == nil {
		l.client = goosehttp.New()
	}
	request, err := http.NewRequest("GET", creds.URL, nil)
	if err != nil {
		return nil, gooseerrors.Newf(err, "", "GET request failed")
	}
	request.Header.Set("X-Auth-User", creds.User)
	request.Header.Set("X-Auth-Key", creds.Secrets)
	response, err := l.client.Do(request)
	if err != nil {
		return nil, gooseerrors.NewUnauthorisedf(err, "", "request failed (maybe client policy or network connectivity issue)")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusOK {
		content, err := ioutil.ReadAll(response.Body)
		return nil, gooseerrors.NewUnauthorisedf(err, "", "Failed to Authenticate (code %d %s): %s",
			response.StatusCode, response.Status, content)
	}
	details := &AuthDetails{}
	details.Token = response.Header.Get("X-Auth-Token")
	if details.Token == "" {
		return nil, fmt.Errorf("Did not get valid Token from auth request")
	}
	details.RegionServiceURLs = make(map[string]ServiceURLs)
	serviceURLs := make(ServiceURLs)
	// Legacy authentication doesn't require a region so use "".
	details.RegionServiceURLs[""] = serviceURLs
	nova_url := response.Header.Get("X-Server-Management-Url")
	serviceURLs["compute"] = nova_url

	swift_url := response.Header.Get("X-Storage-Url")
	if swift_url == "" {
		return nil, fmt.Errorf("Did not get valid swift management URL from auth request")
	}
	serviceURLs["object-store"] = swift_url

	return details, nil
}
