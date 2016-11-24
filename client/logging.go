// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package client

import "gopkg.in/goose.v1/logging"

type logger interface {
	Debugf(f string, v ...interface{})
	Warningf(f string, v ...interface{})
}

func internalLogger(in logging.CompatLogger) logger {
	if in == nil {
		return compatLoggerAdapter{nopLogger{}}
	}
	if l, ok := in.(logging.Logger); ok {
		return l
	}
	return compatLoggerAdapter{in}
}

type compatLoggerAdapter struct {
	logging.CompatLogger
}

// Debugf is part of the logger interface.
func (a compatLoggerAdapter) Debugf(format string, v ...interface{}) {
	a.Printf("DEBUG: "+format, v...)
}

// Warningf is part of the logger interface.
func (a compatLoggerAdapter) Warningf(format string, v ...interface{}) {
	a.Printf("WARNING: "+format, v...)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...interface{}) {
}
