// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package gooseio_test

import (
	"errors"
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
	r, getBody := gooseio.MakeGetReqReader(
		struct{ io.Reader }{strings.NewReader(body)},
		int64(len(body)),
	)
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
	r, getBody := gooseio.MakeGetReqReader(
		struct{ io.Reader }{strings.NewReader(body)},
		int64(len(body)),
	)

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

func (s *getReqReaderSuite) TestReadBeyondBufferLimit(c *gc.C) {
	oldBufSize := *gooseio.MaxBufSize
	*gooseio.MaxBufSize = 20
	defer func() {
		*gooseio.MaxBufSize = oldBufSize
	}()

	body := "123456789 abcdefghijklmnopqrstuvqxyz"
	r, getBody := gooseio.MakeGetReqReader(
		struct{ io.Reader }{strings.NewReader(body)},
		int64(len(body)),
	)
	data, err := ioutil.ReadAll(r)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(data), gc.Equals, body)

	r.Close()

	// Trying again should fail because we've exceeded the buffer limit.

	r1, err := getBody()
	c.Assert(err, gc.ErrorMatches, `read past maximum buffer size`)
	c.Assert(r1, gc.Equals, nil)

	// Try again to make sure the lock hasn't been retained.
	r1, err = getBody()
	c.Assert(err, gc.ErrorMatches, `read past maximum buffer size`)
	c.Assert(r1, gc.Equals, nil)
}

func (s *getReqReaderSuite) TestRestartWithinBufferLimit(c *gc.C) {
	oldBufSize := *gooseio.MaxBufSize
	*gooseio.MaxBufSize = 20
	defer func() {
		*gooseio.MaxBufSize = oldBufSize
	}()

	body := "123456789 abcdefghijklmnopqrstuvqxyz"
	r, getBody := gooseio.MakeGetReqReader(
		struct{ io.Reader }{strings.NewReader(body)},
		int64(len(body)),
	)

	// Read a small amount of the reader; it should be buffered.
	buf := make([]byte, 10)
	_, err := io.ReadFull(r, buf)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(buf), gc.Equals, body[0:10])
	r.Close()

	// Read a bit more of the reader, but still within the limit.
	r, err = getBody()
	c.Assert(err, gc.Equals, nil)
	buf = make([]byte, 19)
	_, err = io.ReadFull(r, buf)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(buf), gc.Equals, body[0:19])
	r.Close()

	// Read beyond the end of the buffer, which should still
	// work but we won't be able to repeat it.
	r, err = getBody()
	c.Assert(err, gc.Equals, nil)
	buf = make([]byte, 25)
	_, err = io.ReadFull(r, buf)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(buf), gc.Equals, body[0:25])
	r.Close()

	r, err = getBody()
	c.Assert(err, gc.ErrorMatches, `read past maximum buffer size`)
	c.Assert(r, gc.Equals, nil)
}

func (s *getReqReaderSuite) TestSeekError(c *gc.C) {
	body := "test body"
	r, getBody := gooseio.MakeGetReqReader(readSeekerWithError{strings.NewReader(body)}, int64(len(body)))
	data, err := ioutil.ReadAll(r)
	c.Assert(err, gc.Equals, nil)
	c.Assert(string(data), gc.Equals, body)

	r.Close()

	r, err = getBody()
	c.Assert(err, gc.ErrorMatches, "some seek error")
	c.Assert(r, gc.Equals, nil)

	// Try again to make sure we haven't retained the lock.
	r, err = getBody()
	c.Assert(err, gc.ErrorMatches, "some seek error")
	c.Assert(r, gc.Equals, nil)
}

type readSeekerWithError struct {
	io.Reader
}

func (r readSeekerWithError) Seek(off int64, whence int) (int64, error) {
	return 0, errors.New("some seek error")
}

var _ io.ReadSeeker = readSeekerWithError{}
