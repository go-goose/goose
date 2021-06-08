package neutron

import (
	"net/http"

	goosehttp "gopkg.in/goose.v3/http"
)

// NeutronHeaders creates a set of http.Headers from the given arguments passed
// in.
// In this case it applies the headers passed in first, then sets the following:
//  - X-Auth-Token
//  - Content-Type
//  - Accept
//  - User-Agent
//
func NeutronHeaders(method string, extraHeaders http.Header, contentType, authToken string, payloadExists bool) http.Header {
	headers := goosehttp.BasicHeaders()

	if authToken != "" {
		headers.Set("X-Auth-Token", authToken)
	}

	// Officially we should also take into account the method, as we should not
	// be applying this to every request.
	if payloadExists {
		headers.Add("Content-Type", contentType)
	}

	// According to the REST specs, the following should be considered the
	// correct implementation of requests. Openstack implementation follow
	// these specs and will return errors if certain headers are pressent.
	//
	// POST allows:    [Content-Type, Accept]
	// PUT allows:     [Content-Type, Accept]
	// GET allows:     [Accept]
	// PATCH allows:   [Content-Type, Accept]
	// HEAD allows:    []
	// OPTIONS allows: []
	// DELETE allows:  []
	// COPY allows:    []

	var ignoreAccept bool
	switch method {
	case "DELETE", "HEAD", "OPTIONS", "COPY":
		ignoreAccept = true
	}

	if !ignoreAccept {
		headers.Add("Accept", contentType)
	}

	// Now apply the passed in headers to the newly created headers.
	if extraHeaders != nil {
		for header, values := range extraHeaders {
			for _, value := range values {
				headers.Add(header, value)
			}
		}
	}

	return headers
}
