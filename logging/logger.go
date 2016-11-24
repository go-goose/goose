// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package logging

import "github.com/juju/loggo"

// CompatLogger is a minimal logging interface that may be provided
// when constructing a goose Client to log requests and responses,
// retaining compatibility with the old *log.Logger that was
// previously dependend upon directly.
//
// TODO(axw) in goose.v2, drop this and use loggo.Logger directly.
type CompatLogger interface {
	// Printf prints a log message. Arguments are handled
	// in the/ manner of fmt.Printf.
	Printf(format string, v ...interface{})
}

// Logger is a logger that may be provided when constructing
// a goose Client to log requests and responses. Users must
// provide a CompatLogger, which will be upgraded to Logger
// if provided.
type Logger struct {
	loggo.Logger
}

// Printf is part of the CompatLogger interface.
func (l Logger) Printf(format string, v ...interface{}) {
	l.Debugf(format, v...)
}
