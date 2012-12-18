// This package provides an Error implementation which knows about types of error, and which has support
// for error causes.

package errors

import "fmt"

type Code string

const (
	// Public available error types.
	// These errors are provided because they are specifically required by business logic in the callers.
	UnspecifiedError    = Code("Unspecified")
	NotFoundError       = Code("NotFound")
	DuplicateValueError = Code("DuplicateValue")
)

// Error instances store an optional error cause.
type Error interface {
	error
	Context() interface{}
	Cause() error
}

type gooseError struct {
	error
	context interface{}
	errcode Code
	cause   error
}

// Type checks.
var _ Error = (*gooseError)(nil)

// Code returns the error code.
func (err *gooseError) code() Code {
	if err.errcode != UnspecifiedError {
		return err.errcode
	}
	if e, ok := err.cause.(*gooseError); ok {
		return e.code()
	}
	return UnspecifiedError
}

// Cause returns the error cause.
func (err *gooseError) Cause() error {
	return err.cause
}

// Context returns any context associated with the error.
// If the top level error has no context, return the context from the root cause error (if any).
func (err *gooseError) Context() interface{} {
	if err.context != nil {
		return err.context
	}
	if e, ok := err.cause.(*gooseError); ok {
		return e.context
	}
	return nil
}

// CausedBy returns true if this error or its cause are of the specified error code.
func (err *gooseError) causedBy(code Code) bool {
	if err.code() == code {
		return true
	}
	if cause, ok := err.cause.(*gooseError); ok {
		return cause.code() == code
	}
	return false
}

// Error fulfills the error interface, taking account of any caused by error.
func (err *gooseError) Error() string {
	result := err.error.Error()
	if err.cause != nil {
		return fmt.Sprintf("%s, caused by: %s", result, err.cause.Error())
	}
	return result
}

func IsNotFound(err error) bool {
	if e, ok := err.(*gooseError); ok {
		return e.causedBy(NotFoundError)
	}
	return false
}

func IsDuplicateValue(err error) bool {
	if e, ok := err.(*gooseError); ok {
		return e.causedBy(DuplicateValueError)
	}
	return false
}

// New creates a new Error instance with the specified cause.
func Newf(code Code, cause error, context interface{}, format string, args ...interface{}) Error {
	if format == "" {
		switch code {
		case NotFoundError:
			format = fmt.Sprintf("Not found: %s", context)
		case DuplicateValueError:
			format = fmt.Sprintf("Duplicate: %s", context)
		}
	}
	return &gooseError{
		context: context,
		errcode: code,
		error:   fmt.Errorf(format, args...),
		cause:   cause,
	}
}
