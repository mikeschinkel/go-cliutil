package cliutil

import (
	"errors"
	"os"
	"syscall"
)

// IsTerminalError checks if an error is related to terminal/input operations
// These errors should abort the entire operation rather than continue
func IsTerminalError(err error) (isTermErr bool) {
	var pathErr *os.PathError
	var errno syscall.Errno
	var ok bool

	if err == nil {
		goto end
	}

	switch {
	case errors.As(err, &pathErr):
		// Check for syscall errors related to terminal operations
		errno, ok = pathErr.Err.(syscall.Errno)
		if ok && errno == syscall.ENOTTY { // ENOTTY = "inappropriate ioctl for device"
			isTermErr = true
			goto end
		}
	case errors.As(err, &errno):
		// Check for direct syscall errors
		if errno == syscall.ENOTTY { // ENOTTY = "inappropriate ioctl for device"
			isTermErr = true
			goto end
		}
	}

end:
	return isTermErr
}
