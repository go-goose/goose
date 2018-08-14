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
var maxBufSize = int64(1024 * 1024 * 1024)

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
	return &unlockingReadCloser{
			mu:     &mu,
			Reader: rs,
		}, func() (io.ReadCloser, error) {
			mu.Lock()
			_, err := rs.Seek(0, 0)
			if err != nil {
				mu.Unlock()
				return nil, err
			}
			return &unlockingReadCloser{
				mu:     &mu,
				Reader: rs,
			}, nil
		}
}

func makeBufferingGetReqReader(r io.Reader, size int64) (io.ReadCloser, func() (io.ReadCloser, error)) {
	// mu guards r and buf.
	var mu sync.Mutex
	var buf bytes.Buffer

	// getReader returns a reader that buffers up to maxBufSize bytes in memory.
	getReader := func() io.ReadCloser {
		// Return a reader that first reads any bytes that have been buffered already,
		// then from a reader that continues to fill up the buffer up to maxBufSize
		// bytes, then directly from r.
		return &unlockingReadCloser{
			mu: &mu,
			Reader: io.MultiReader(
				// First read any bytes already buffered.
				bytes.NewReader(buf.Bytes()),

				// Then read while filling up the buffer to its maximum size.
				io.TeeReader(io.LimitReader(r, maxBufSize-int64(buf.Len())), &buf),

				// Finally, when the buffer has filled up, read directly from r.
				// If this happens and the request is repeated, we'll trigger the
				// "read past maximum buffer size" error.
				r,
			),
		}
	}
	// Because net/http doesn't guarantee not to start using the
	// second reader before the first one has been closed,
	// we guard it with a mutex, so the second reader cannot
	// be acquired until the first is closed.
	mu.Lock()
	return getReader(), func() (io.ReadCloser, error) {
		mu.Lock()
		if int64(buf.Len()) >= maxBufSize {
			mu.Unlock()
			return nil, errors.New("read past maximum buffer size")
		}
		return getReader(), nil
	}
}

// unlockingReadCloser implements io.ReadCloser by unlocking the
// mutex when it's closed.
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
