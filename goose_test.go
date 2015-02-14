package goose

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type GooseTestSuite struct {
}

var _ = Suite(&GooseTestSuite{})
