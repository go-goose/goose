// This package provides an Error implementation which knows about types of error, and which has support
// for nested errors.

package errors

import "fmt"

type ErrorCode string

const (
	unspecifiedError = "Unspecified"
	// Public available error types.
	// These errors are provided because they are specifically required by business logic in the callers.
	NotFoundError       = "NotFound"
	DuplicateValueError = "DuplicateValue"
)

// New creates a new Error instance, using the specified context or cause.
// If an error is provided, the new Error instance becomes a nested error, otherwise
// the context is recorded.
func New(contextOrCause interface{}, format string, args ...interface{}) Error {
	var context interface{}
	var cause error
	if _, ok := contextOrCause.(error); ok {
		cause = contextOrCause.(error)
	} else {
		context = contextOrCause
	}
	return makeError(unspecifiedError, context, cause, format, args...)
}

// NewError creates a new Error instance, using the specified context and cause.
func NewError(context interface{}, cause error, format string, args ...interface{}) Error {
	return makeError(unspecifiedError, context, cause, format, args...)
}

// makeError is a private method for creating Error instances.
func makeError(code ErrorCode, context interface{}, cause error, format string, args ...interface{}) Error {
	return Error{
		Context: context,
		code:    code,
		error:   fmt.Errorf(format, args...),
		Cause:   cause,
	}
}

// A NestedError is caused by another error.
type NestedError interface {
	error
	CausedBy(code ErrorCode) bool
}

// Error instances store an error code so that the type can be inferred, as well as an optional error cause.
type Error struct {
	error
	Context interface{}
	code    ErrorCode
	Cause   error
}

// CausedBy returns true if this error or any of its nested errors are of the specified error code.
func (err Error) CausedBy(code ErrorCode) bool {
	if err.code == code {
		return true
	}
	if _, ok := err.Cause.(NestedError); ok {
		return err.Cause.(NestedError).CausedBy(code)
	}
	return false
}

// Error fulfills the error interface, taking account of any nested errors.
func (err Error) Error() string {
	result := err.error.Error()
	if _, ok := err.Cause.(NestedError); ok {
		return fmt.Sprintf("%s, caused by: %s", result, err.Cause.(NestedError).Error())
	}
	return result
}

// IsNotFound returns true if this error is caused by a NotFoundError.
func (err Error) IsNotFound() bool {
	return err.CausedBy(NotFoundError)
}

func IsNotFound(err error) bool {
	if e, ok := err.(Error); ok {
		return e.IsNotFound()
	}
	return false
}

// IsNotFound returns true if this error is caused by a DuplicateValueError.
func (err Error) IsDuplicateValue() bool {
	return err.CausedBy(DuplicateValueError)
}

func IsDuplicateValue(err error) bool {
	if e, ok := err.(Error); ok {
		return e.IsDuplicateValue()
	}
	return false
}

// NotFound creates the simplest possible NotFound error.
func NotFound(context interface{}) Error {
	return NewNotFound(context, nil, fmt.Sprintf("Not found: %s", context))
}

// NewNotFound creates a NotFound error with the specified (optional) cause and error text.
func NewNotFound(context interface{}, cause error, format string, args ...interface{}) Error {
	return makeError(NotFoundError, context, cause, format, args...)
}

// DuplicateValue creates the simplest possible DuplicateValue error.
func DuplicateValue(context interface{}) Error {
	return NewDuplicateValue(context, nil, fmt.Sprintf("Duplicate: %s", context))
}

// NewDuplicateValue creates a DuplicateValue error with the specified (optional) cause and error text.
func NewDuplicateValue(context interface{}, cause error, format string, args ...interface{}) Error {
	return makeError(DuplicateValueError, context, cause, format, args...)
}
