package http

import "net/http"

func SendHTTPRequest(c *Client, method, url string, headers *http.Header) (resp *http.Response, err error) {
	return c.sendRateLimitedRequest(method, url, *headers, []byte{}, nil)
}
