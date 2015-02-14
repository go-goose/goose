package goose

import (
	"testing"

	gc "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	gc.TestingT(t)
}

type GooseTestSuite struct {
}

var _ = gc.Suite(&GooseTestSuite{})
