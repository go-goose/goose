package identity

import (
	"fmt"
	"io/ioutil"
	"net/http"

	gooseerrors "github.com/go-goose/goose/v5/errors"
	goosehttp "github.com/go-goose/goose/v5/http"
)

type Legacy struct {
	client goosehttp.HttpClient
}

func (l *Legacy) Auth(creds *Credentials) (*AuthDetails, error) {
	if l.client == nil {
		l.client = goosehttp.New()
	}

	request, err := http.NewRequest("GET", creds.URL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Auth-User", creds.User)
	request.Header.Set("X-Auth-Key", creds.Secrets)

	response, err := l.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("Failed to Authenticate (code %d %s): %s",
			response.StatusCode, response.Status, content)
	}

	details := &AuthDetails{}
	details.Token = response.Header.Get("X-Auth-Token")
	if details.Token == "" {
		return nil, gooseerrors.NewUnauthorisedf(nil, "", "Did not get valid Token from auth request")
	}
	details.RegionServiceURLs = make(map[string]ServiceURLs)

	serviceURLs := make(ServiceURLs)

	// Legacy authentication doesn't require a region so use "".
	details.RegionServiceURLs[""] = serviceURLs
	novaURL := response.Header.Get("X-Server-Management-Url")
	serviceURLs["compute"] = novaURL

	swiftURL := response.Header.Get("X-Storage-Url")
	if swiftURL == "" {
		return nil, fmt.Errorf("Did not get valid swift management URL from auth request")
	}
	serviceURLs["object-store"] = swiftURL

	return details, nil
}
