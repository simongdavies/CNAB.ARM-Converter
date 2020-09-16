package common

import (
	"fmt"
)

// ValidateTimeout validates the timeout parameter
func ValidateTimeout(timeout int) error {
	minTimeout := 5
	maxTimeout := 120
	if timeout >= minTimeout && timeout <= maxTimeout {
		return nil
	}
	return fmt.Errorf("Value %d for param timeout is less than min value %d or greater than max value %d", timeout, minTimeout, maxTimeout)

}
