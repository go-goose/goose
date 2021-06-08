package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"

	gc "gopkg.in/check.v1"
)

type ConnReuseSuite struct{}

var _ = gc.Suite(&ConnReuseSuite{})

func (*ConnReuseSuite) TestConnectionsAreReused(c *gc.C) {
	connCount := int32(0)
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, net, addr string) (net.Conn, error) {
				atomic.AddInt32(&connCount, 1)
				return http.DefaultTransport.(*http.Transport).DialContext(ctx, net, addr)
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/with-body-chunked":
			w.Header().Set("Test", "0")
			w.Write([]byte("chunked"))
		case "/with-body-non-chunked":
			body := []byte("non-chunked")
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			w.Header().Set("Test", "1")
			w.Write(body)
		case "/no-body":
			w.Header().Set("Test", "2")
		case "/large-body":
			w.Header().Set("Test", "3")
			w.Write([]byte(strings.Repeat("a", 1025)))
		default:
			c.Errorf("unexpected path %s", req.URL.Path)
		}
	}))
	client := New(WithHTTPClient(httpClient))

	assertReq := func(path string, expectTestHeader string) {
		var req RequestData
		err := client.BinaryRequest("GET", srv.URL+path, "", &req, nil)
		c.Assert(err, gc.Equals, nil)
		c.Assert(req.RespHeaders.Get("Test"), gc.Equals, expectTestHeader)
	}
	for i, path := range []string{
		"/with-body-chunked",
		"/with-body-non-chunked",
		"/no-body",
	} {
		assertReq(path, fmt.Sprint(i))
	}
	c.Assert(connCount, gc.Equals, int32(1))
	// When the body is too large, another connection
	// will be made.
	assertReq("/large-body", "3")
	assertReq("/no-body", "2")
	c.Assert(connCount, gc.Equals, int32(2))
}
