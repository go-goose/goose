// An HTTP Client which sends json and binary requests, handling data marshalling and response processing.

package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"gopkg.in/goose.v2"
	"gopkg.in/goose.v2/errors"
	"gopkg.in/goose.v2/internal/gooseio"
	"gopkg.in/goose.v2/logging"
)

const (
	contentTypeJSON        = "application/json"
	contentTypeOctetStream = "application/octet-stream"
)

type HttpClient interface {
	BinaryRequest(method, url, token string, reqData *RequestData, logger logging.CompatLogger) (err error)
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (resp *http.Response, err error)
	Head(url string) (resp *http.Response, err error)
	JsonRequest(method, url, token string, reqData *RequestData, logger logging.CompatLogger) error
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	PostForm(url string, data url.Values) (resp *http.Response, err error)
}

type Option func(*options)

type options struct {
	headersFunc HeadersFunc
}

func WithHeadersFunc(headersFunc HeadersFunc) Option {
	return func(options *options) {
		options.headersFunc = headersFunc
	}
}

func newOptions() *options {
	return &options{
		headersFunc: DefaultHeaders,
	}
}

type Client struct {
	http.Client
	headersFunc     HeadersFunc
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

func unmarshallError(jsonBytes []byte) (*ErrorResponse, error) {
	var response ErrorResponse
	var transientObject = make(map[string]*json.RawMessage)
	if err := json.Unmarshal(jsonBytes, &transientObject); err != nil {
		return nil, err
	}
	for key, value := range transientObject {
		if err := json.Unmarshal(*value, &response); err != nil {
			return nil, err
		}
		response.Title = key
		break
	}
	if response.Code != 0 && response.Message != "" {
		return &response, nil
	}
	return nil, fmt.Errorf("Unparsable json error body: %q", jsonBytes)
}

type RequestData struct {
	ReqHeaders     http.Header
	Params         *url.Values
	ExpectedStatus []int
	ReqValue       interface{}
	// ReqReader is used to read the body of the request. If the
	// request has to be retried for any reason then a replacement
	// ReqReader will be collected using GetReqReader.
	ReqReader io.Reader

	// GetReqReader is called to get a new ReqReader if a request
	// fails and will be retried. If ReqReader is specified but
	// GetReqReader is not then an appropriate GetReqReader function
	// will be generated from ReqReader.
	//
	// If ReqReader implements io.Seeker then the generated
	// GetReqReader function will use Seek to rewind the request.
	// Otherwise the entire body may be kept in memory whilst sending
	// the request.
	GetReqReader func() (io.ReadCloser, error)

	// TODO this should really be int64 not int.
	ReqLength int

	RespStatusCode int
	RespValue      interface{}
	RespLength     int64
	RespReader     io.ReadCloser
	RespHeaders    http.Header
}

const (
	// The maximum number of times to try sending a request before we give up
	// (assuming any unsuccessful attempts can be sensibly tried again).
	MaxSendAttempts = 3
)

var insecureClient *http.Client
var insecureClientMutex sync.Mutex

// New returns a new goose http *Client using the default net/http client.
func New(options ...Option) *Client {
	opts := newOptions()
	for _, option := range options {
		option(opts)
	}

	return &Client{
		Client:          *http.DefaultClient,
		headersFunc:     opts.headersFunc,
		maxSendAttempts: MaxSendAttempts,
	}
}

// NewNonSSLValidating creates a new goose http *Client skipping SSL validation.
func NewNonSSLValidating(options ...Option) *Client {
	opts := newOptions()
	for _, option := range options {
		option(opts)
	}

	insecureClientMutex.Lock()
	httpClient := insecureClient
	if httpClient == nil {
		insecureConfig := &tls.Config{InsecureSkipVerify: true}
		insecureTransport := &http.Transport{TLSClientConfig: insecureConfig}
		insecureClient = &http.Client{Transport: insecureTransport}
		httpClient = insecureClient
	}
	insecureClientMutex.Unlock()

	return &Client{
		Client:          *httpClient,
		headersFunc:     opts.headersFunc,
		maxSendAttempts: MaxSendAttempts,
	}
}

// NewWithTLSConfig creates a new goose http *Client with a TLS config.
func NewWithTLSConfig(tlsConfig *tls.Config, options ...Option) *Client {
	opts := newOptions()
	for _, option := range options {
		option(opts)
	}

	defaultClient := *http.DefaultClient
	defaultClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &Client{
		Client:          defaultClient,
		headersFunc:     opts.headersFunc,
		maxSendAttempts: MaxSendAttempts,
	}
}

// GooseAgent returns the current client goose agent version.
func GooseAgent() string {
	return fmt.Sprintf("goose (%s)", goose.Version)
}

// JsonRequest JSON encodes and sends the object in reqData.ReqValue (if any) to the specified URL.
// Optional method arguments are passed using the RequestData object.
// Relevant RequestData fields:
// ReqHeaders: additional HTTP header values to add to the request.
// ExpectedStatus: the allowed HTTP response status values, else an error is returned.
// ReqValue: the data object to send.
// RespValue: the data object to decode the result into.
func (c *Client) JsonRequest(method, url, token string, reqData *RequestData, logger logging.CompatLogger) error {
	var body io.Reader
	var length int64
	var getBody func() (io.ReadCloser, error)
	if reqData.Params != nil {
		url += "?" + reqData.Params.Encode()
	}
	if reqData.ReqValue != nil {
		data, err := json.Marshal(reqData.ReqValue)
		if err != nil {
			return errors.Newf(err, "failed marshalling the request body")
		}
		body = bytes.NewReader(data)
		getBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(data)), nil
		}
		length = int64(len(data))
	}
	headers := c.headersFunc(method, reqData.ReqHeaders, contentTypeJSON, token, reqData.ReqValue != nil)
	resp, err := c.sendRequest(
		method,
		url,
		body,
		getBody,
		length,
		headers,
		reqData.ExpectedStatus,
		logging.FromCompat(logger),
	)
	if err != nil {
		return err
	}
	reqData.RespHeaders = resp.Header
	reqData.RespStatusCode = resp.StatusCode
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Newf(err, "failed reading the response body")
	}

	if len(respData) > 0 && reqData.RespValue != nil {
		if err := json.Unmarshal(respData, &reqData.RespValue); err != nil {
			return errors.Newf(err, "failed unmarshaling the response body: %s", respData)
		}
	}
	return nil
}

// BinaryRequest sends the byte array in reqData.ReqValue (if any) to
// the specified URL.
// Optional method arguments are passed using the RequestData object.
// Relevant RequestData fields:
// ReqHeaders: additional HTTP header values to add to the request.
// ExpectedStatus: the allowed HTTP response status values, else an error is returned.
// ReqReader: an io.Reader providing the bytes to send.
// RespReader: if non-nil, is assigned an io.ReadCloser instance used to
// read the returned data.
func (c *Client) BinaryRequest(method, url, token string, reqData *RequestData, logger logging.CompatLogger) (err error) {
	err = nil

	if reqData.Params != nil {
		url += "?" + reqData.Params.Encode()
	}
	headers := c.headersFunc(method, reqData.ReqHeaders, contentTypeOctetStream, token, reqData.ReqLength != 0)
	resp, err := c.sendRequest(
		method,
		url,
		reqData.ReqReader,
		reqData.GetReqReader,
		int64(reqData.ReqLength),
		headers,
		reqData.ExpectedStatus,
		logging.FromCompat(logger),
	)
	if err != nil {
		return
	}
	reqData.RespStatusCode = resp.StatusCode
	reqData.RespLength = resp.ContentLength
	reqData.RespHeaders = resp.Header
	if reqData.RespReader != nil {
		reqData.RespReader = resp.Body
	} else {
		if method != "HEAD" && resp.ContentLength != 0 {
			// Read a small amount of data from the response
			// body so that the client connection can
			// be reused.
			size := resp.ContentLength
			if size > 1024 || size < 0 {
				size = 1024
			}
			resp.Body.Read(make([]byte, size))
		}
		resp.Body.Close()
	}
	return
}

// sendRequest sends the specified request to URL and checks that the
// HTTP response status is as expected.
// reqReader: a reader returning the data to send.
// length: the number of bytes to send.
// headers: HTTP headers to include with the request.
// expectedStatus: a slice of allowed response status codes.
func (c *Client) sendRequest(
	method, URL string,
	reqReader io.Reader,
	getReqReader func() (io.ReadCloser, error),
	length int64,
	headers http.Header,
	expectedStatus []int,
	logger logging.Logger,
) (*http.Response, error) {
	rawResp, err := c.sendRateLimitedRequest(method, URL, headers, reqReader, getReqReader, length, logger)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return rawResp, err
}

func (c *Client) sendRateLimitedRequest(
	method, URL string,
	headers http.Header,
	reqReader io.Reader,
	getReqReader func() (io.ReadCloser, error),
	length int64,
	logger logging.Logger,
) (resp *http.Response, err error) {
	if reqReader != nil && getReqReader == nil {
		reqReader, getReqReader = gooseio.MakeGetReqReader(reqReader, length)
	}
	for i := 0; i < c.maxSendAttempts; i++ {
		req, err := http.NewRequest(method, URL, reqReader)
		if err != nil {
			return nil, errors.Newf(err, "failed creating the request %s", URL)
		}
		req.GetBody = getReqReader
		for header, values := range headers {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}
		req.ContentLength = length
		resp, err = c.Do(req)
		if err != nil {
			return nil, errors.Newf(err, "failed executing the request %s", URL)
		}

		switch resp.StatusCode {
		case http.StatusRequestEntityTooLarge,
			http.StatusForbidden,
			http.StatusServiceUnavailable,
			http.StatusTooManyRequests:
			if resp.Header.Get("Retry-After") == "" {
				return resp, nil
			}
		default:
			return resp, nil
		}
		resp.Body.Close()
		respRetryAfter := resp.Header.Get("Retry-After")
		// Per: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After
		// Retry-After can be: <delay-seconds> or <http-date>
		// Try <delay-seconds> first
		if retryAfter, err := strconv.ParseFloat(respRetryAfter, 32); err == nil {
			if retryAfter == 0 {
				return nil, errors.Newf(err, "Resource limit exeeded at URL %s", URL)
			}
			logger.Debugf("Too many requests, retrying in %dms.", int(retryAfter*1000))
			time.Sleep(time.Duration(retryAfter) * time.Second)
		} else {
			// Failed on assuming <delay-seconds>, try <http-date>
			// http-date: <day-name>, <day> <month> <year> <hour>:<minute>:<second> GMT
			// time.RFC1123 = "Mon, 02 Jan 2006 15:04:05 MST"
			httpDate, err := time.Parse(time.RFC1123, respRetryAfter)
			if err != nil {
				return nil, errors.Newf(err, "Invalid Retry-After header %s", URL)
			}
			sleepDuration := time.Until(httpDate)
			if sleepDuration.Minutes() > 10 {
				logger.Debugf("Cloud is not accepting further requests from this account until %s", httpDate.Local().Format(time.UnixDate))
				logger.Debugf("It is recommended to verify your account rate limits")
				return nil, errors.Newf(err, "Cloud is not accepting further requests from this account until %s", httpDate.Local().Format(time.UnixDate))
			}
			logger.Debugf("Too many requests, retrying after %s", httpDate.Local().Format(time.UnixDate))
			time.Sleep(sleepDuration)
		}
		if reqReader != nil {
			reqReader, err = getReqReader()
			if err != nil {
				return nil, fmt.Errorf("cannot get request body reader: %v", err)
			}
		}
	}
	return nil, errors.Newf(err, "Maximum number of attempts (%d) reached sending request to %s", c.maxSendAttempts, URL)
}

type nopReader struct{}

func (nopReader) Read(buf []byte) (int, error) {
	return 0, io.EOF
}

type closeNotifier chan struct{}

func (c closeNotifier) Close() error {
	select {
	case <-c:
		return nil
	default:
	}
	close(c)
	return nil
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
	if resp.Header.Get("Content-Type") == contentTypeJSON {
		errorResponse, err := unmarshallError(errBytes)
		//TODO (hduran-8): Obtain a logger and log the error
		if err == nil {
			errInfo = errorResponse.Error()
		}
	}
	httpError := &HttpError{
		resp.StatusCode, map[string][]string(resp.Header), URL, errInfo,
	}
	switch resp.StatusCode {
	case http.StatusNotFound:
		return errors.NewNotFoundf(httpError, "", "Resource at %s not found", URL)
	case http.StatusForbidden, http.StatusUnauthorized:
		return errors.NewUnauthorisedf(httpError, "", "Unauthorised URL %s", URL)
	case http.StatusBadRequest:
		dupExp, _ := regexp.Compile(".*already exists.*")
		if dupExp.Match(errBytes) {
			return errors.NewDuplicateValuef(httpError, "", string(errBytes))
		}
	case http.StatusMultipleChoices:
		return errors.NewMultipleChoicesf(httpError, "", "")
	}
	return httpError
}
