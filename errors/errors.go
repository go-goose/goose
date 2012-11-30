// Utility functions for reporting errors.

package errors

import (
	"errors"
	"fmt"
)

// AddContext prefixes any error stored in err with text formatted
// according to the format specifier. If err does not contain an error,
// AddContext does nothing.
func AddContext(err error, format string, args ...interface{}) error {
	if err != nil {
		msg := fmt.Sprintf(format, args...) + ": " + err.Error()
		if IsNotFound(err) {
			err = NotFound(msg)
		} else {
			err = errors.New(msg)
		}
	}
	return err
}

// NotFoundError represents the error that something is not found.
type NotFoundError struct {
	msg string
}

func (e *NotFoundError) Error() string {
	return e.msg
}

func NotFound(format string, args ...interface{}) error {
	return &NotFoundError{fmt.Sprintf(format+" not found", args...)}
}

func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}
