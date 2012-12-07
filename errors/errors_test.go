package errors_test

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goose/errors"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type ErrorsSuite struct {
}

var _ = Suite(&ErrorsSuite{})

func (s *ErrorsSuite) TestCreateSimpleNotFoundError(c *C) {
	context := "context"
	err := errors.NewNotFoundf(context, nil, "")
	c.Assert(errors.IsNotFound(err), Equals, true)
	c.Assert(err.Context(), Equals, context)
	c.Assert(err.Error(), Equals, "Not found: context")
}

func (s *ErrorsSuite) TestCreateNotFoundError(c *C) {
	context := "context"
	err := errors.NewNotFoundf(context, nil, "It was not found: %s", context)
	c.Assert(errors.IsNotFound(err), Equals, true)
	c.Assert(err.Context(), Equals, context)
	c.Assert(err.Error(), Equals, "It was not found: context")
}

func (s *ErrorsSuite) TestCreateSimpleDuplicateValueError(c *C) {
	context := "context"
	err := errors.NewDuplicateValuef(context, nil, "")
	c.Assert(errors.IsDuplicateValue(err), Equals, true)
	c.Assert(err.Context(), Equals, context)
	c.Assert(err.Error(), Equals, "Duplicate: context")
}

func (s *ErrorsSuite) TestCreateDuplicateValueError(c *C) {
	context := "context"
	err := errors.NewDuplicateValuef(context, nil, "It was duplicate: %s", context)
	c.Assert(errors.IsDuplicateValue(err), Equals, true)
	c.Assert(err.Context(), Equals, context)
	c.Assert(err.Error(), Equals, "It was duplicate: context")
}

func (s *ErrorsSuite) TestCausedBy(c *C) {
	rootCause := errors.NewNotFoundf("some value", nil, "")

	err := errors.Newf(rootCause, "an error occurred")
	c.Assert(errors.IsDuplicateValue(err), Equals, false)
	c.Assert(err.CausedBy(errors.DuplicateValueError), Equals, false)
	c.Assert(errors.IsNotFound(err), Equals, true)
	c.Assert(err.CausedBy(errors.NotFoundError), Equals, true)
	c.Assert(err.Error(), Equals, "an error occurred, caused by: Not found: some value")
}
