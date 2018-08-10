// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package gooseio

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

// maxBufSize holds the maximum amount of data
// that may be allocated as a buffer when sending
// a request when the body is not seekable.
const maxBufSize = 1024 * 1024 * 1024

// MakeGetReqReader creates a safe mechanism to get a new request body
// reader for retrying requests. It returns a suitable replacement reader
// along with a function that should be used to get subsequent readers
// when retrying.
func MakeGetReqReader(r io.Reader, size int64) (io.ReadCloser, func() (io.ReadCloser, error)) {
	if rs, ok := r.(io.ReadSeeker); ok {
		return makeSeekingGetReqReader(rs)
	}
	return makeBufferingGetReqReader(r, size)
}

func makeSeekingGetReqReader(rs io.ReadSeeker) (io.ReadCloser, func() (io.ReadCloser, error)) {
	var mu sync.Mutex
	mu.Lock()
	rc := &unlockingReadCloser{
		mu:     &mu,
		Reader: rs,
	}
	f := func() (io.ReadCloser, error) {
		mu.Lock()
		_, err := rs.Seek(0, 0)
		if err != nil {
			return nil, err
		}
		return &unlockingReadCloser{
			mu:     &mu,
			Reader: rs,
		}, nil
	}
	return rc, f
}

func makeBufferingGetReqReader(r io.Reader, size int64) (io.ReadCloser, func() (io.ReadCloser, error)) {
	var mu sync.Mutex
	mu.Lock()
	var buf bytes.Buffer
	rc := &unlockingReadCloser{
		mu:     &mu,
		Reader: io.TeeReader(io.LimitReader(r, maxBufSize), &buf),
	}
	f := func() (io.ReadCloser, error) {
		mu.Lock()
		if buf.Len() > maxBufSize {
			return nil, errors.New("read past maximum buffer size")
		}
		return &unlockingReadCloser{
			mu: &mu,
			Reader: io.MultiReader(
				bytes.NewReader(buf.Bytes()),
				io.TeeReader(io.LimitReader(r, maxBufSize-int64(buf.Len())), &buf),
			),
		}, nil
	}
	return rc, f
}

type unlockingReadCloser struct {
	mu *sync.Mutex
	io.Reader
}

func (c *unlockingReadCloser) Close() error {
	if c.mu != nil {
		c.mu.Unlock()
		c.mu = nil
	}
	return nil
}
