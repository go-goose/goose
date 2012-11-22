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
		err = errors.New(fmt.Sprintf(format, args...) + ": " + err.Error())
	}
	return err
}
