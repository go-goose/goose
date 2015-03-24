// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package cinder

import (
	"flag"
	"testing"

	gc "gopkg.in/check.v1"
)

var (
	live = flag.Bool("live", false, "Include live OpenStack tests")
)

func init() {
	flag.Parse()
}

func Test(t *testing.T) {
	gc.TestingT(t)
}
