package http

import "net/http"

// HeadersFunc is type for aligning the creation of a series of client headers.
type HeadersFunc = func(method string, headers http.Header, contentType, authToken string, hasPayload bool) http.Header

// DefaultHeaders creates a set of http.Headers from the given arguments passed
// in.
// In this case it applies the headers passed in first, then sets the following:
//  - X-Auth-Token
//  - Content-Type
//  - Accept
//  - User-Agent
//
func DefaultHeaders(method string, extraHeaders http.Header, contentType, authToken string, payloadExists bool) http.Header {
	headers := make(http.Header)
	if extraHeaders != nil {
		for header, values := range extraHeaders {
			for _, value := range values {
				headers.Add(header, value)
			}
		}
	}
	if authToken != "" {
		headers.Set("X-Auth-Token", authToken)
	}
	if payloadExists {
		headers.Add("Content-Type", contentType)
	}
	headers.Add("Accept", contentType)
	headers.Add("User-Agent", GooseAgent())
	return headers
}
