// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package gooseio_test

import (
	"io"
	"io/ioutil"
	"strings"
	"sync"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v2/internal/gooseio"
)

type getReqReaderSuite struct{}

var _ = gc.Suite(&getReqReaderSuite{})

func (s *getReqReaderSuite) TestConcurrentBuffered(c *gc.C) {
	body := "test body"
	var r0 struct{ io.Reader }
	r0.Reader = strings.NewReader(body)
	r, getBody := gooseio.MakeGetReqReader(r0, int64(len(body)))
	s.testConcurrent(c, r, getBody, body)
}

func (s *getReqReaderSuite) TestConcurrentSeekable(c *gc.C) {
	body := "test body"
	r, getBody := gooseio.MakeGetReqReader(strings.NewReader(body), int64(len(body)))
	s.testConcurrent(c, r, getBody, body)
}

func (s *getReqReaderSuite) testConcurrent(c *gc.C, r io.ReadCloser, getBody func() (io.ReadCloser, error), body string) {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := getBody()
			if err != nil {
				c.Error(err)
				return
			}
			defer r.Close()
			received, err := ioutil.ReadAll(r)
			c.Check(err, gc.Equals, nil)
			c.Check(string(received), gc.Equals, body)
		}()
	}
	r.Close()
	wg.Wait()
}

func (s *getReqReaderSuite) TestBufferedEarlyClose(c *gc.C) {
	body := "test body"
	var r0 struct{ io.Reader }
	r0.Reader = strings.NewReader(body)
	r, getBody := gooseio.MakeGetReqReader(r0, int64(len(body)))

	var buf [5]byte
	_, err := io.ReadFull(r, buf[:])
	c.Assert(err, gc.Equals, nil)
	r.Close()
	r, err = getBody()
	c.Assert(err, gc.Equals, nil)
	buf2, err := ioutil.ReadAll(r)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(buf2), gc.Equals, body)
}
