// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package gooseio

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/goose.v2/errors"
)

// maxBufSize holds the maximum amount of data
// that may be allocated as a buffer when sending
// a request when the body is not seekable.
const maxBufSize = 1024 * 1024 * 1024

// Seekable returns a ReadSeeker that contains the contents of the given
// Reader.
func Seekable(r io.Reader, length int64) (io.ReadSeeker, error) {
	if r == nil {
		return nil, nil
	}
	if r, ok := r.(io.ReadSeeker); ok {
		return r, nil
	}
	if length > maxBufSize {
		return nil, fmt.Errorf("body of length %d is too large to hold in memory (max %d bytes)", length, maxBufSize)
	}
	reqData := make([]byte, int(length))
	nrRead, err := io.ReadFull(r, reqData)
	if err != nil {
		return nil, errors.Newf(err, "failed reading the request data, read %v of %v bytes", nrRead, length)
	}
	return bytes.NewReader(reqData), nil
}
