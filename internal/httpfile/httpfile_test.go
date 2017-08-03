package httpfile_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing/iotest"
	"time"

	gc "gopkg.in/check.v1"
	"gopkg.in/goose.v2/internal/httpfile"
)

type httpFileSuite struct{}

var _ = gc.Suite(&httpFileSuite{})

func (*httpFileSuite) TestNoReadAhead(c *gc.C) {
	content := newContent(10 * 1024)
	requestc := make(chan readRequest, 1000)
	srv := httptest.NewServer(&fakeReadServer{
		content: content,
		request: requestc,
	})
	defer srv.Close()

	// Open the file. Because we've requested zero readahead,
	// it should just issue a HEAD request.
	f, h, err := httpfile.Open(newClient(c, srv.URL+"/file"), 0)
	c.Assert(err, gc.Equals, nil)
	defer f.Close()
	c.Check(f.Size(), gc.Equals, int64(len(content)))
	c.Check(h.Get("Etag"), gc.Equals, fmt.Sprintf(`"%x"`, md5.Sum(content)))

	req := getRequest(c, requestc)
	c.Check(req.Request.URL.Path, gc.Equals, "/file")
	c.Check(req.doneRead, gc.Equals, false, gc.Commentf("[%d %d]", req.p0, req.p1))
	c.Check(req.Method, gc.Equals, "HEAD")
	assertNoRequest(c, requestc)

	off, err := f.Seek(30, io.SeekStart)
	c.Assert(err, gc.Equals, nil)
	c.Assert(off, gc.Equals, int64(30))
	assertNoRequest(c, requestc)

	// Issuing a read request causes a single
	// request to be made to the server.
	assertReadBytes(c, f, content[30:30+100])

	req = getRequest(c, requestc)
	c.Check(req.Request.URL.Path, gc.Equals, "/file")
	c.Check(req.doneRead, gc.Equals, true)
	c.Check(req.Method, gc.Equals, "GET")
	c.Check(req.p0, gc.Equals, int64(30))
	c.Check(req.p1, gc.Equals, int64(130))
	assertNoRequest(c, requestc)

	// Check that we can seek relative to the end.
	buf := make([]byte, 200)
	off, err = f.Seek(-200, io.SeekEnd)
	c.Assert(err, gc.Equals, nil)
	c.Assert(off, gc.Equals, int64(len(content)-200))
	assertReadBytes(c, f, content[len(content)-200:])

	req = getRequest(c, requestc)
	c.Check(req.Request.URL.Path, gc.Equals, "/file")
	c.Check(req.doneRead, gc.Equals, true)
	c.Check(req.Method, gc.Equals, "GET")
	c.Check(req.p0, gc.Equals, int64(len(content)-200))
	c.Check(req.p1, gc.Equals, int64(len(content)))
	assertNoRequest(c, requestc)

	// At the end, we should see EOF with no further request
	// issued.
	n, err := f.Read(buf)
	c.Assert(n, gc.Equals, 0)
	c.Assert(err, gc.Equals, io.EOF)
	assertNoRequest(c, requestc)
}

func (*httpFileSuite) TestInfiniteReadAhead(c *gc.C) {
	content := newContent(10 * 1024)
	requestc := make(chan readRequest, 1000)
	srv := httptest.NewServer(&fakeReadServer{
		content: content,
		request: requestc,
	})
	defer srv.Close()

	// Open the file. Because we've requested arbitrary
	// readahead, it should issue a request to get the
	// whole file.
	f, _, err := httpfile.Open(newClient(c, srv.URL+"/file"), -1)
	c.Assert(err, gc.Equals, nil)
	defer f.Close()

	// Assume the loopback TCP buffer is big enough
	// to hold all of the content - if it's not, we'd
	// deadlock here.
	req := getRequest(c, requestc)
	c.Check(req.doneRead, gc.Equals, true)
	c.Check(req.Method, gc.Equals, "GET")
	c.Check(req.p0, gc.Equals, int64(0))
	c.Check(req.p1, gc.Equals, int64(len(content)))
	assertNoRequest(c, requestc)

	data, err := ioutil.ReadAll(f)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(data), gc.Equals, string(content))
	assertNoRequest(c, requestc)

	// Check that we get a single read when seeking back
	// elsewhere into the content.
	off, err := f.Seek(30, io.SeekStart)
	c.Assert(err, gc.Equals, nil)
	c.Check(off, gc.Equals, int64(30))

	assertReadBytes(c, f, content[30:40])
	req = getRequest(c, requestc)
	c.Check(req.doneRead, gc.Equals, true)
	c.Check(req.Method, gc.Equals, "GET")
	c.Check(req.p0, gc.Equals, int64(30))
	c.Check(req.p1, gc.Equals, int64(len(content)))
	assertNoRequest(c, requestc)

	// Seek ahead a bit and check that it still reuses
	// the same GET request.
	off, err = f.Seek(30, io.SeekCurrent)
	c.Assert(err, gc.Equals, nil)
	c.Check(off, gc.Equals, int64(70))

	data, err = ioutil.ReadAll(f)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(data), gc.Equals, string(content[70:]))

	assertNoRequest(c, requestc)
}

func (*httpFileSuite) TestLimitedReadAhead(c *gc.C) {
	content := newContent(10 * 1024)
	requestc := make(chan readRequest, 1000)
	srv := httptest.NewServer(&fakeReadServer{
		content: content,
		request: requestc,
	})
	defer srv.Close()

	// Open the file. Because we've requested arbitrary
	// readahead, it should issue a request to get the
	// whole file.
	client := newClient(c, srv.URL+"/file")
	f, _, err := httpfile.Open(client, 200)
	c.Assert(err, gc.Equals, nil)
	defer f.Close()

	req := getRequest(c, requestc)
	c.Check(req.doneRead, gc.Equals, true)
	c.Check(req.Method, gc.Equals, "GET")
	c.Check(req.p0, gc.Equals, int64(0))
	c.Check(req.p1, gc.Equals, int64(200))
	assertNoRequest(c, requestc)

	// Reading half the readahead amount doesn't
	// trigger a new request
	assertReadBytes(c, f, content[0:100])
	assertNoRequest(c, requestc)

	// Reading past half the readahead amount triggers
	// a new request.
	assertReadBytes(c, f, content[100:110])

	req = getRequest(c, requestc)
	c.Check(req.doneRead, gc.Equals, true)
	c.Check(req.Method, gc.Equals, "GET")
	c.Check(req.p0, gc.Equals, int64(200))
	c.Check(req.p1, gc.Equals, int64(400))
	assertNoRequest(c, requestc)

	// Reading past the read ahead buffer
	// results in a partial read ending at that
	// buffer.
	buf := make([]byte, 300)
	n, err := f.Read(buf)
	c.Assert(err, gc.Equals, nil)
	c.Assert(n, gc.Equals, 90)
	c.Assert(string(buf[0:90]), gc.Equals, string(content[110:200]))

	// Check we can read all the rest of the content.
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Expect one request per readahead block (note
		// it's rounded up) less the two requests we've already
		// seen.
		expectRequestCount := (len(content)+199)/200 - 2
		for i := 0; i < expectRequestCount; i++ {
			getRequest(c, requestc)
		}
		assertNoRequest(c, requestc)
	}()
	// Read the rest of the buffer in small increments so
	// that we use many reads. Wrap the buffer so that it
	// doesn't implement ReadFrom and bybass our buffer
	// size choice.
	var rbuf bytes.Buffer
	_, err = io.CopyBuffer(struct{ io.Writer }{&rbuf}, f, make([]byte, 20))
	c.Assert(err, gc.Equals, nil)
	c.Assert(rbuf.String(), gc.Equals, string(content[200:]))
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Fatalf("timed out waiting for expected requests")
	}

	// One connection for each concurrent read.
	c.Assert(client.connectionCount(), gc.Equals, 2)
}

func (s *httpFileSuite) TestConnectionReuseWhenStreaming(c *gc.C) {
	const size = 1024 * 1024
	content := newContent(size)
	srv := httptest.NewServer(&fakeReadServer{
		content: content,
	})
	defer srv.Close()

	client := newClient(c, srv.URL+"/file")
	f, _, err := httpfile.Open(client, 8192)
	c.Assert(err, gc.Equals, nil)
	defer f.Close()

	// Use OneByteReader so that the client takes enough time reading
	// that both connections actually get made.
	n, err := io.CopyBuffer(ioutil.Discard, iotest.OneByteReader(f), make([]byte, 512))
	c.Assert(err, gc.Equals, nil)
	c.Assert(n, gc.Equals, int64(size))

	f.Close()
	if got, want := client.connectionCount(), 2; got > want {
		c.Errorf("more client connections than expected; got %d want <= %d", got, want)
	}
}

func (s *httpFileSuite) TestCloseSkipsDataIfNotTooLong(c *gc.C) {
	// When there's less than NoSkip bytes of data
	// in the pool, the Close logic reads it so that
	// the connection can be reused.
	s.testDataSkip(c, httpfile.NoSkip, 1)
}

func (s *httpFileSuite) TestCloseDoesNotSkipDataWhenTooLong(c *gc.C) {
	// When there's at least NoSkip bytes of data
	// in the pool, the Close logic throws away
	// the connection.
	s.testDataSkip(c, httpfile.NoSkip-1, 2)
}

func (s *httpFileSuite) testDataSkip(c *gc.C, remain int64, expectConnections int) {
	size := httpfile.NoSkip * 3
	content := newContent(size)
	srv := httptest.NewServer(&fakeReadServer{
		content: content,
	})
	defer srv.Close()

	client := newClient(c, srv.URL+"/file")
	f, _, err := httpfile.Open(client, httpfile.NoSkip*2)
	c.Assert(err, gc.Equals, nil)
	defer f.Close()

	buf := make([]byte, httpfile.NoSkip*2-remain)
	_, err = io.ReadFull(f, buf)
	c.Assert(err, gc.Equals, nil)

	f.Close()

	// Allow time for the asynchronous discard to happen
	// (or not).
	time.Sleep(100 * time.Millisecond)

	// Open the file again and check the number of connections
	f, _, err = httpfile.Open(client, httpfile.NoSkip*2)
	c.Assert(err, gc.Equals, nil)
	defer f.Close()

	_, err = io.ReadFull(f, buf)
	c.Assert(err, gc.Equals, nil)

	c.Assert(client.connectionCount(), gc.Equals, expectConnections)
}

var badContentRangeResponseTests = []struct {
	about              string
	readAhead          int64
	status             int
	contentRange       string
	secondContentRange string
	expectError        string
}{{
	about:        "content range where none expected",
	readAhead:    -1,
	status:       http.StatusOK,
	contentRange: "bytes 0-300/300",
	expectError:  `received unexpected Content-Range "bytes 0-300/300" in response`,
}, {
	about:       "no content length",
	readAhead:   -1,
	status:      http.StatusOK,
	expectError: `unknown file length in response`,
}, {
	about:       "no content range",
	readAhead:   200,
	status:      http.StatusPartialContent,
	expectError: `missing Content-Range in response`,
}, {
	about:        "content range not starting with bytes",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "100-300",
	expectError:  `bad Content-Range header "100-300"`,
}, {
	about:        "bad start of range",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes foo",
	expectError:  `bad Content-Range header "bytes foo"`,
}, {
	about:        "no hyphen in range",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes 0z",
	expectError:  `bad Content-Range header "bytes 0z"`,
}, {
	about:        "bad end of range",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes 0-z",
	expectError:  `bad Content-Range header "bytes 0-z"`,
}, {
	about:        "no slash after range",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes 0-20z",
	expectError:  `bad Content-Range header "bytes 0-20z"`,
}, {
	about:        "bad content length",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes 0-20/z",
	expectError:  `bad Content-Range header "bytes 0-20/z"`,
}, {
	about:        "extra bytes after content length",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes 0-20/30z",
	expectError:  `bad Content-Range header "bytes 0-20/30z"`,
}, {
	about:        "out of order range",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes 20-19/30",
	expectError:  `bad Content-Range header "bytes 20-19/30"`,
}, {
	about:        "start after requested range",
	readAhead:    200,
	status:       http.StatusPartialContent,
	contentRange: "bytes 1-200/200",
	expectError:  `response range \[1, 201\] out of range of requested range starting at 0`,
}, {
	about:              "wrong length",
	readAhead:          200,
	status:             http.StatusPartialContent,
	contentRange:       "bytes 0-199/4096",
	secondContentRange: "bytes 200-399/4097",
	expectError:        `response range has unexpected length; got 4097 want 4096`,
}}

func (*httpFileSuite) TestBadContentRangeResponse(c *gc.C) {
	var contentRange string
	var status int
	content := newContent(4 * 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if contentRange != "" {
			w.Header().Set("Content-Range", contentRange)
		}
		w.WriteHeader(status)
		// Ensure we get chunked encoding when there's
		// no Content-Length header.
		w.Write(content)
	}))
	defer srv.Close()

	for i, test := range badContentRangeResponseTests {
		c.Logf("test %d: %v", i, test.about)
		contentRange = test.contentRange
		status = test.status
		f, _, err := httpfile.Open(newClient(c, srv.URL), test.readAhead)
		if test.secondContentRange == "" {
			c.Assert(err, gc.ErrorMatches, test.expectError)
			continue
		}
		c.Assert(err, gc.Equals, nil)
		contentRange = test.secondContentRange
		// Seek beyond the read-ahead buffer
		// so that we'll make another read request.
		f.Seek(test.readAhead, io.SeekStart)
		buf := make([]byte, 200)
		_, err = f.Read(buf)
		c.Assert(err, gc.ErrorMatches, test.expectError)
		f.Close()
	}
}

func (*httpFileSuite) TestFileNotFound(c *gc.C) {
	srv := httptest.NewServer(http.HandlerFunc(http.NotFound))
	defer srv.Close()
	_, _, err := httpfile.Open(newClient(c, srv.URL), -1)
	c.Assert(err, gc.Equals, httpfile.ErrNotFound)
}

func (*httpFileSuite) TestNotFoundFromPreconditionFailed(c *gc.C) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusPreconditionFailed)
	}))
	defer srv.Close()
	_, _, err := httpfile.Open(newClient(c, srv.URL), -1)
	c.Assert(err, gc.Equals, httpfile.ErrNotFound)
}

func (s *httpFileSuite) TestFileChangedUnderfoot(c *gc.C) {
	content := newContent(4096)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Etag", fmt.Sprintf(`"%x"`, md5.Sum(content)))
		http.ServeContent(w, req, "foo.gif", time.Now(), bytes.NewReader(content))
	}))
	f, _, err := httpfile.Open(newClient(c, srv.URL), 200)
	c.Assert(err, gc.Equals, nil)

	_, err = f.Seek(600, io.SeekStart)
	c.Assert(err, gc.Equals, nil)

	content = newContent(3000)
	buf := make([]byte, 10)
	n, err := f.Read(buf)
	c.Check(err, gc.ErrorMatches, `file has changed since it was opened`)
	c.Check(n, gc.Equals, 0)
}

func assertReadBytes(c *gc.C, f io.Reader, expect []byte) {
	buf := make([]byte, len(expect))
	n, err := f.Read(buf)
	c.Assert(err, gc.Equals, nil)
	c.Assert(n, gc.Equals, len(expect))
	c.Assert(string(buf), gc.Equals, string(expect))
}

func newContent(size int) []byte {
	var content []byte
	for i := 0; len(content) < size; i++ {
		content = append(content, fmt.Sprint(i, " ")...)
	}
	return content[0:size]
}

type client struct {
	c          *gc.C
	url        string
	transport  *http.Transport
	httpClient *http.Client
	connCount  int32
}

func newClient(c *gc.C, url string) *client {
	client := &client{
		c:          c,
		url:        url,
		httpClient: new(http.Client),
	}
	// Use an HTTP client that allows us to keep track of the total
	// number of live connections.
	dialer := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext
	client.transport = &http.Transport{
		DialContext: func(ctx context.Context, net, addr string) (net.Conn, error) {
			netConn, err := dialer(ctx, net, addr)
			if err != nil {
				return nil, err
			}
			atomic.AddInt32(&client.connCount, 1)
			return conn{netConn, client}, nil
		},
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     5 * time.Second,
	}
	client.httpClient.Transport = client.transport
	return client
}

func (c *client) connectionCount() int {
	return int(atomic.LoadInt32(&c.connCount))
}

type conn struct {
	net.Conn
	client *client
}

func (c conn) Close() error {
	return c.Conn.Close()
}

func (c *client) Do(req *httpfile.Request) (*httpfile.Response, error) {
	hreq, _ := http.NewRequest(req.Method, c.url, nil)
	for key, val := range req.Header {
		hreq.Header[key] = val
	}
	hresp, err := c.httpClient.Do(hreq)
	if err != nil {
		return nil, err
	}
	//	c.c.Logf("sent request %v %q", req.Method, req.Header)
	//	c.c.Logf("-> %v [%d] %q", hresp.StatusCode, hresp.ContentLength, hresp.Header)
	return &httpfile.Response{
		StatusCode:    hresp.StatusCode,
		Header:        hresp.Header,
		ContentLength: hresp.ContentLength,
		Body:          hresp.Body,
	}, nil
}

func getRequest(c *gc.C, requestc chan readRequest) readRequest {
	select {
	case req := <-requestc:
		return req
	case <-time.After(time.Second):
		c.Fatalf("timed out waiting for request")
		panic("unreachable")
	}
}

func assertNoRequest(c *gc.C, requestc chan readRequest) {
	select {
	case <-requestc:
		c.Fatalf("got request when none was expected")
	case <-time.After(100 * time.Millisecond):
	}
}

// readRequest holds details of a request made to fakeReadServer.
type readRequest struct {
	// Request holds the HTTP request used.
	*http.Request

	// doneRead holds whether any actual data has been requested.
	doneRead bool

	// [p0, p1) is the byte range requested with the read request.
	p0, p1 int64
}

// fakeReadServer is an HTTP server that serves some
// content with http.ServeContent.
type fakeReadServer struct {
	// content holds the content to be served.
	content []byte

	// If request is non-nil, it is sent on when a request is completed.
	request chan readRequest
}

func (srv *fakeReadServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	time.Sleep(time.Millisecond)
	// Prevent ServeContent from sniffing into the content
	// and confusing our rangeRecorder.
	w.Header().Set("Content-Type", "application/binary")

	rec := newRangeRecorder(bytes.NewReader(srv.content))
	w.Header().Set("Etag", fmt.Sprintf(`"%x"`, md5.Sum(srv.content)))

	// Use a SectionReader to make a ReadSeeker out of our
	// rangeRecord (which is a ReaderAt).
	r := io.NewSectionReader(rec, 0, int64(len(srv.content)))

	http.ServeContent(w, req, "x", time.Now(), r)
	if srv.request == nil {
		return
	}
	select {
	case srv.request <- readRequest{
		Request:  req,
		doneRead: rec.doneRead,
		p0:       rec.p0,
		p1:       rec.p1,
	}:
	default:
		panic("cannot send read request")
	}
}

type rangeRecorder struct {
	r        io.ReaderAt
	p0, p1   int64
	doneRead bool
}

func newRangeRecorder(r io.ReaderAt) *rangeRecorder {
	return &rangeRecorder{r: r}
}

func (r *rangeRecorder) ReadAt(buf []byte, p0 int64) (int, error) {
	p1 := p0 + int64(len(buf))
	if p0 < r.p0 || !r.doneRead {
		r.p0 = p0
	}
	if p1 > r.p1 || !r.doneRead {
		r.p1 = p1
	}
	r.doneRead = true
	return r.r.ReadAt(buf, p0)
}
