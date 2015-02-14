package errors_test

import (
	"testing"

	gc "gopkg.in/check.v1"

	"gopkg.in/goose.v1/errors"
)

func Test(t *testing.T) { gc.TestingT(t) }

type ErrorsSuite struct {
}

var _ = gc.Suite(&ErrorsSuite{})

func (s *ErrorsSuite) TestCreateSimpleNotFoundError(c *gc.C) {
	context := "context"
	err := errors.NewNotFoundf(nil, context, "")
	c.Assert(errors.IsNotFound(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "Not found: context")
}

func (s *ErrorsSuite) TestCreateNotFoundError(c *gc.C) {
	context := "context"
	err := errors.NewNotFoundf(nil, context, "It was not found: %s", context)
	c.Assert(errors.IsNotFound(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "It was not found: context")
}

func (s *ErrorsSuite) TestCreateSimpleDuplicateValueError(c *gc.C) {
	context := "context"
	err := errors.NewDuplicateValuef(nil, context, "")
	c.Assert(errors.IsDuplicateValue(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "Duplicate: context")
}

func (s *ErrorsSuite) TestCreateDuplicateValueError(c *gc.C) {
	context := "context"
	err := errors.NewDuplicateValuef(nil, context, "It was duplicate: %s", context)
	c.Assert(errors.IsDuplicateValue(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "It was duplicate: context")
}

func (s *ErrorsSuite) TestCreateSimpleUnauthorisedfError(c *gc.C) {
	context := "context"
	err := errors.NewUnauthorisedf(nil, context, "")
	c.Assert(errors.IsUnauthorised(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "Unauthorised: context")
}

func (s *ErrorsSuite) TestCreateUnauthorisedfError(c *gc.C) {
	context := "context"
	err := errors.NewUnauthorisedf(nil, context, "It was unauthorised: %s", context)
	c.Assert(errors.IsUnauthorised(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "It was unauthorised: context")
}

func (s *ErrorsSuite) TestCreateSimpleNotImplementedfError(c *gc.C) {
	context := "context"
	err := errors.NewNotImplementedf(nil, context, "")
	c.Assert(errors.IsNotImplemented(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "Not implemented: context")
}

func (s *ErrorsSuite) TestCreateNotImplementedfError(c *gc.C) {
	context := "context"
	err := errors.NewNotImplementedf(nil, context, "It was not implemented: %s", context)
	c.Assert(errors.IsNotImplemented(err), gc.Equals, true)
	c.Assert(err.Error(), gc.Equals, "It was not implemented: context")
}

func (s *ErrorsSuite) TestErrorCause(c *gc.C) {
	rootCause := errors.NewNotFoundf(nil, "some value", "")
	// Construct a new error, based on a not found root cause.
	err := errors.Newf(rootCause, "an error occurred")
	c.Assert(err.Cause(), gc.Equals, rootCause)
	// Check the other error attributes.
	c.Assert(err.Error(), gc.Equals, "an error occurred\ncaused by: Not found: some value")
}

func (s *ErrorsSuite) TestErrorIsType(c *gc.C) {
	rootCause := errors.NewNotFoundf(nil, "some value", "")
	// Construct a new error, based on a not found root cause.
	err := errors.Newf(rootCause, "an error occurred")
	// Check that the error is not falsely identified as something it is not.
	c.Assert(errors.IsDuplicateValue(err), gc.Equals, false)
	// Check that the error is correctly identified as a not found error.
	c.Assert(errors.IsNotFound(err), gc.Equals, true)
}
