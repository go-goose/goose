// An HTTP Client which sends json and binary requests, handling data marshalling and response processing.

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"launchpad.net/goose/errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	contentTypeJson        = "application/json"
	contentTypeOctetStream = "application/octet-stream"
)

type Client struct {
	http.Client
	logger          *log.Logger
	authToken       string
	maxSendAttempts int
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
	ReqReader      io.Reader
	ReqLength      int
	RespReader     io.ReadCloser
}

const (
	// The maximum number of times to try sending a request before we give up
	// (assuming any unsuccessful attempts can be sensibly tried again).
	MaxSendAttempts = 3
)

func New(httpClient http.Client, logger *log.Logger, token string) *Client {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	return &Client{httpClient, logger, token, MaxSendAttempts}
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
	var body []byte
	if reqData.Params != nil {
		url += "?" + reqData.Params.Encode()
	}
	if reqData.ReqValue != nil {
		body, err = json.Marshal(reqData.ReqValue)
		if err != nil {
			err = errors.Newf(err, "failed marshalling the request body")
			return
		}
	}
	headers := make(http.Header)
	if reqData.ReqHeaders != nil {
		for header, values := range reqData.ReqHeaders {
			for _, value := range values {
				headers.Add(header, value)
			}
		}
	}
	headers.Add("Content-Type", contentTypeJson)
	headers.Add("Accept", contentTypeJson)
	respBody, err := c.sendRequest(method, url, bytes.NewReader(body), len(body), headers, reqData.ExpectedStatus)
	if err != nil {
		return
	}
	defer respBody.Close()
	respData, err := ioutil.ReadAll(respBody)
	if err != nil {
		err = errors.Newf(err, "failed reading the response body")
		return
	}

	if len(respData) > 0 {
		if reqData.RespValue != nil {
			err = json.Unmarshal(respData, &reqData.RespValue)
			if err != nil {
				err = errors.Newf(err, "failed unmarshaling the response body: %s", respData)
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
// ReqReader: an io.Reader providing the bytes to send.
// RespReader: assigned an io.ReadCloser instance used to read the returned data..
func (c *Client) BinaryRequest(method, url string, reqData *RequestData) (err error) {
	err = nil

	if reqData.Params != nil {
		url += "?" + reqData.Params.Encode()
	}
	headers := make(http.Header)
	if reqData.ReqHeaders != nil {
		for header, values := range reqData.ReqHeaders {
			for _, value := range values {
				headers.Add(header, value)
			}
		}
	}
	headers.Add("Content-Type", contentTypeOctetStream)
	headers.Add("Accept", contentTypeOctetStream)
	respBody, err := c.sendRequest(method, url, reqData.ReqReader, reqData.ReqLength, headers, reqData.ExpectedStatus)
	if err != nil {
		return
	}
	if reqData.RespReader != nil {
		reqData.RespReader = respBody
	}
	return
}

// Sends the specified request and checks that the HTTP response status is as expected.
// req: the request to send.
// extraHeaders: additional HTTP headers to include with the request.
// expectedStatus: a slice of allowed response status codes.
// payloadInfo: a string to include with an error message if something goes wrong.
func (c *Client) sendRequest(method, URL string, reqReader io.Reader, length int, headers http.Header, expectedStatus []int) (rc io.ReadCloser, err error) {
	if c.authToken != "" {
		headers.Add("X-Auth-Token", c.authToken)
	}
	var reqData []byte = make([]byte, length)
	if reqReader != nil {
		nrRead, err := reqReader.Read(reqData)
		if nrRead != length || err != nil {
			err = errors.Newf(err, "failed reading the request data, read %v of %v bytes", nrRead, length)
			return rc, err
		}
	}
	rawResp, err := c.sendRateLimitedRequest(method, URL, headers, reqData)
	if err != nil {
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
	if !foundStatus && len(expectedStatus) > 0 {
		err = handleError(URL, rawResp)
		rawResp.Body.Close()
		return
	}
	return rawResp.Body, err
}

func (c *Client) sendRateLimitedRequest(method, URL string, headers http.Header, reqData []byte) (resp *http.Response, err error) {
	for i := 0; i < c.maxSendAttempts; i++ {
		var reqReader io.Reader
		if reqData != nil {
			reqReader = bytes.NewReader(reqData)
		}
		req, err := http.NewRequest(method, URL, reqReader)
		if err != nil {
			err = errors.Newf(err, URL, "failed creating the request")
			return nil, err
		}
		for header, values := range headers {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}
		resp, err = c.Do(req)
		if err != nil {
			return nil, errors.Newf(err, URL, "failed executing the request")
		}
		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			return resp, nil
		}
		retryAfter, err := strconv.ParseFloat(resp.Header.Get("Retry-After"), 32)
		if err != nil {
			return nil, errors.Newf(err, URL, "Invalid Retry-After header")
		}
		if retryAfter == 0 {
			return nil, errors.Newf(err, URL, "Resource limit exeeded at URL %s.", URL)
		}
		c.logger.Printf("Too many requests, retrying in %dms.", int(retryAfter*1000))
		time.Sleep(time.Duration(retryAfter) * time.Second)
	}
	return nil, errors.Newf(err, URL, "Maximum number of attempts (%d) reached sending request to %s.", c.maxSendAttempts, URL)
}

type HttpError struct {
	StatusCode      int
	Data            map[string][]string
	url             string
	responseMessage string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("request (%s) returned unexpected status: %d; error info: %v",
		e.url,
		e.StatusCode,
		e.responseMessage,
	)
}

// The HTTP response status code was not one of those expected, so we construct an error.
// NotFound (404) codes have their own NotFound error type.
// We also make a guess at duplicate value errors.
func handleError(URL string, resp *http.Response) error {
	errBytes, _ := ioutil.ReadAll(resp.Body)
	errInfo := string(errBytes)
	// Check if we have a JSON representation of the failure, if so decode it.
	if resp.Header.Get("Content-Type") == contentTypeJson {
		var wrappedErr ErrorWrapper
		if err := json.Unmarshal(errBytes, &wrappedErr); err == nil {
			errInfo = wrappedErr.Error.Error()
		}
	}
	httpError := &HttpError{
		resp.StatusCode, map[string][]string(resp.Header), URL, errInfo,
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
