// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see COPYING and COPYING.LESSER file for details.

package http

import "gopkg.in/goose.v1/logging"

type logger interface {
	Debugf(string, ...interface{})
}

func internalLogger(in logging.CompatLogger) logger {
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
