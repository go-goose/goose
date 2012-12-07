// This package provides an Error implementation which knows about types of error, and which has support
// for nested errors.

package errors

import "fmt"

type Code string

const (
	unspecifiedError = Code("Unspecified")
	// Public available error types.
	// These errors are provided because they are specifically required by business logic in the callers.
	NotFoundError       = Code("NotFound")
	DuplicateValueError = Code("DuplicateValue")
)

// Error instances store an error code so that the type can be inferred, as well as an optional error cause.
type Error interface {
	error
	Code() Code
	Context() interface{}
	CausedBy(code Code) bool
}

type gooseError struct {
	error
	context interface{}
	code    Code
	cause   error
}

// Code returns the error code.
func (err *gooseError) Code() Code {
	return err.code
}

// Context returns any context associated with the error.
func (err *gooseError) Context() interface{} {
	return err.context
}

// CausedBy returns true if this error or any of its nested errors are of the specified error code.
func (err *gooseError) CausedBy(code Code) bool {
	if err.code == code {
		return true
	}
	if cause, ok := err.cause.(Error); ok {
		return cause.CausedBy(code)
	}
	return false
}

// Error fulfills the error interface, taking account of any nested errors.
func (err *gooseError) Error() string {
	result := err.error.Error()
	if cause, ok := err.cause.(Error); ok {
		return fmt.Sprintf("%s, caused by: %s", result, cause.Error())
	}
	return result
}

func IsNotFound(err error) bool {
	if e, ok := err.(Error); ok {
		return e.CausedBy(NotFoundError)
	}
	return false
}

func IsDuplicateValue(err error) bool {
	if e, ok := err.(Error); ok {
		return e.CausedBy(DuplicateValueError)
	}
	return false
}

// New creates a new Error instance, using the specified context or cause.
// If an error is provided, the new Error instance becomes a nested error, otherwise
// the context is recorded.
func Newf(contextOrCause interface{}, format string, args ...interface{}) Error {
	var context interface{}
	var cause error
	var ok bool
	if cause, ok = contextOrCause.(error); !ok {
		context = contextOrCause
	}
	return makeErrorf(unspecifiedError, context, cause, format, args...)
}

// NewNotFound creates a NotFound error with the specified (optional) cause and error text.
func NewNotFoundf(context interface{}, cause error, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Not found: %s", context)
	}
	return makeErrorf(NotFoundError, context, cause, format, args...)
}

// NewDuplicateValue creates a DuplicateValue error with the specified (optional) cause and error text.
func NewDuplicateValuef(context interface{}, cause error, format string, args ...interface{}) Error {
	if format == "" {
		format = fmt.Sprintf("Duplicate: %s", context)
	}
	return makeErrorf(DuplicateValueError, context, cause, format, args...)
}

// makeError is a private method for creating Error instances.
func makeErrorf(code Code, context interface{}, cause error, format string, args ...interface{}) Error {
	return &gooseError{
		context: context,
		code:    code,
		error:   fmt.Errorf(format, args...),
		cause:   cause,
	}
}
