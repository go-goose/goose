package http

import "net/http"

// SendRequest dispatches the request on the client.
func SendRequest(c *http.Client, req *http.Request) (resp *http.Response, err error) {
	// See https://code.google.com/p/go/issues/detail?id=4677
	// We need to force the connection to close each time so that we don't
	// hit the above Go bug.
	req.Close = true
	return c.Do(req)
}
