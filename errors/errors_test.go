package errors_test

import (
	. "launchpad.net/gocheck"
	gooseerrors "launchpad.net/goose/errors"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type ErrorsSuite struct {
}

var _ = Suite(&ErrorsSuite{})

func (s *ErrorsSuite) TestCreateSimpleNotFoundError(c *C) {
	context := "context"
	err := gooseerrors.NotFound(context)
	c.Assert(err.IsNotFound(), Equals, true)
	c.Assert(err.Context, Equals, context)
	c.Assert(err.Error(), Equals, "Not found: context")
}

func (s *ErrorsSuite) TestCreateNotFoundError(c *C) {
	context := "context"
	err := gooseerrors.NewNotFound(context, nil, "It was not found: %s", context)
	c.Assert(err.IsNotFound(), Equals, true)
	c.Assert(err.Context, Equals, context)
	c.Assert(err.Error(), Equals, "It was not found: context")
}

func (s *ErrorsSuite) TestCreateSimpleDuplicateValueError(c *C) {
	context := "context"
	err := gooseerrors.DuplicateValue(context)
	c.Assert(err.IsDuplicateValue(), Equals, true)
	c.Assert(err.Context, Equals, context)
	c.Assert(err.Error(), Equals, "Duplicate: context")
}

func (s *ErrorsSuite) TestCreateDuplicateValueError(c *C) {
	context := "context"
	err := gooseerrors.NewDuplicateValue(context, nil, "It was duplicate: %s", context)
	c.Assert(err.IsDuplicateValue(), Equals, true)
	c.Assert(err.Context, Equals, context)
	c.Assert(err.Error(), Equals, "It was duplicate: context")
}

func (s *ErrorsSuite) TestCausedBy(c *C) {
	rootCause := gooseerrors.NotFound("some value")

	err := gooseerrors.New(rootCause, "an error occurred")
	c.Assert(err.IsDuplicateValue(), Equals, false)
	c.Assert(err.CausedBy(gooseerrors.DuplicateValueError), Equals, false)
	c.Assert(err.IsNotFound(), Equals, true)
	c.Assert(err.CausedBy(gooseerrors.NotFoundError), Equals, true)
	c.Assert(err.Error(), Equals, "an error occurred, caused by: Not found: some value")
}
