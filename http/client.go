// An HTTP Client which sends json and binary requests, handling data marshalling and response processing.

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"launchpad.net/goose/errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Client struct {
	http.Client
	AuthToken string
}

type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Title   string `json:"title"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("Failed: %d %s: %s", e.Code, e.Title, e.Message)
}

type ErrorWrapper struct {
	Error ErrorResponse `json:"error"`
}

type RequestData struct {
	ReqHeaders     http.Header
	Params         *url.Values
	ExpectedStatus []int
	ReqValue       interface{}
	RespValue      interface{}
	ReqData        []byte
	RespData       *[]byte
}

// JsonRequest JSON encodes and sends the supplied object (if any) to the specified URL.
// Optional method arguments are pass using the RequestData object.
// Relevant RequestData fields:
// ReqHeaders: additional HTTP header values to add to the request.
// ExpectedStatus: the allowed HTTP response status values, else an error is returned.
// ReqValue: the data object to send.
// RespValue: the data object to decode the result into.
func (c *Client) JsonRequest(method, url string, reqData *RequestData) (err error) {
	err = nil
	var (
		req  *http.Request
		body []byte
	)
	if reqData.Params != nil {
		url += "?" + reqData.Params.Encode()
	}
	if reqData.ReqValue != nil {
		body, err = json.Marshal(reqData.ReqValue)
		if err != nil {
			err = errors.Newf(err, "failed marshalling the request body")
			return
		}
		reqBody := strings.NewReader(string(body))
		req, err = http.NewRequest(method, url, reqBody)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		err = errors.Newf(err, "failed creating the request")
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	respBody, err := c.sendRequest(req, reqData.ReqHeaders, reqData.ExpectedStatus, string(body))
	if err != nil {
		return
	}

	if len(respBody) > 0 {
		if reqData.RespValue != nil {
			err = json.Unmarshal(respBody, &reqData.RespValue)
			if err != nil {
				err = errors.Newf(err, "failed unmarshaling the response body: %s", respBody)
			}
		}
	}
	return
}

// Sends the supplied byte array (if any) to the specified URL.
// Optional method arguments are pass using the RequestData object.
// Relevant RequestData fields:
// ReqHeaders: additional HTTP header values to add to the request.
// ExpectedStatus: the allowed HTTP response status values, else an error is returned.
// ReqData: the byte array to send.
// RespData: the byte array to decode the result into.
func (c *Client) BinaryRequest(method, url string, reqData *RequestData) (err error) {
	err = nil

	var req *http.Request

	if reqData.Params != nil {
		url += "?" + reqData.Params.Encode()
	}
	if reqData.ReqData != nil {
		rawReqReader := bytes.NewReader(reqData.ReqData)
		req, err = http.NewRequest(method, url, rawReqReader)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		err = errors.Newf(err, "failed creating the request")
		return
	}
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("Accept", "application/octet-stream")

	respBody, err := c.sendRequest(req, reqData.ReqHeaders, reqData.ExpectedStatus, string(reqData.ReqData))
	if err != nil {
		return
	}

	if len(respBody) > 0 {
		if reqData.RespData != nil {
			*reqData.RespData = respBody
		}
	}
	return
}

// Sends the specified request and checks that the HTTP response status is as expected.
// req: the request to send.
// extraHeaders: additional HTTP headers to include with the request.
// expectedStatus: a slice of allowed response status codes.
// payloadInfo: a string to include with an error message if something goes wrong.
func (c *Client) sendRequest(req *http.Request, extraHeaders http.Header, expectedStatus []int, payloadInfo string) (respBody []byte, err error) {
	if extraHeaders != nil {
		for header, values := range extraHeaders {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}
	}
	if c.AuthToken != "" {
		req.Header.Add("X-Auth-Token", c.AuthToken)
	}
	rawResp, err := c.Do(req)
	if err != nil {
		err = errors.Newf(err, "failed executing the request")
		return
	}
	foundStatus := false
	if len(expectedStatus) == 0 {
		expectedStatus = []int{http.StatusOK}
	}
	for _, status := range expectedStatus {
		if rawResp.StatusCode == status {
			foundStatus = true
			break
		}
	}
	defer rawResp.Body.Close()
	if !foundStatus && len(expectedStatus) > 0 {
		err = handleError(req.URL, rawResp, payloadInfo)
		return
	}

	respBody, err = ioutil.ReadAll(rawResp.Body)
	if err != nil {
		err = errors.Newf(err, "failed reading the response body")
		return
	}
	return
}

type HttpError struct {
	StatusCode      int
	Data            map[string][]string
	url             string
	responseMessage string
	requestPayload  string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("request (%s) returned unexpected status: %s; error info: %v; request body: [%s]",
		e.url,
		e.StatusCode,
		e.responseMessage,
		e.requestPayload,
	)
}

// The HTTP response status code was not one of those expected, so we construct an error.
// NotFound (404) codes have their own NotFound error type.
// We also make a guess at duplicate value errors.
func handleError(URL *url.URL, resp *http.Response, payloadInfo string) error {
	errBytes, _ := ioutil.ReadAll(resp.Body)
	errInfo := string(errBytes)
	// Check if we have a JSON representation of the failure, if so decode it.
	if resp.Header.Get("Content-Type") == "application/json" {
		var wrappedErr ErrorWrapper
		if err := json.Unmarshal(errBytes, &wrappedErr); err == nil {
			errInfo = wrappedErr.Error.Error()
		}
	}
	httpError := &HttpError{
		resp.StatusCode, map[string][]string(resp.Header), URL.String(), errInfo, payloadInfo,
	}
	switch resp.StatusCode {
	case http.StatusNotFound:
		{
			return errors.NewNotFoundf(httpError, "", "Resource at %s not found", URL)
		}
	case http.StatusBadRequest:
		{
			dupExp, _ := regexp.Compile(".*already exists.*")
			if dupExp.Match(errBytes) {
				return errors.NewDuplicateValuef(httpError, "", string(errBytes))
			}
		}
	}
	return httpError
}
