// Utility functions for reporting errors.

package errors

import (
	"errors"
	"fmt"
)

// AddErrorContext prefixes any error stored in err with text formatted
// according to the format specifier. If err does not contain an error,
// AddErrorContext does nothing.
func AddErrorContext(err *error, format string, args ...interface{}) {
	if *err != nil {
		*err = errors.New(fmt.Sprintf(format, args...) + ": " + (*err).Error())
	}
}
