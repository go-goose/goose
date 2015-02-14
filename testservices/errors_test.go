package testservices

import (
	"testing"

	gc "gopkg.in/check.v1"
)

func Test(t *testing.T) { gc.TestingT(t) }

type ErrorsSuite struct {
}

var _ = gc.Suite(&ErrorsSuite{})

func (s *ErrorsSuite) TestServerErrorMessage(c *gc.C) {
	err := &ServerError{
		message: "Instance could not be found",
		code:    404,
	}
	c.Assert(err, gc.ErrorMatches, "itemNotFound: Instance could not be found")
}

func (s *ErrorsSuite) TestServerUnknownErrcode(c *gc.C) {
	err := &ServerError{
		message: "Impossible http code.",
		code:    999,
	}
	c.Assert(err, gc.ErrorMatches, "computeFault: Impossible http code.")
}
