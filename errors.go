package cliutil

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrShowUsage           = fmt.Errorf("run '%s help' for usage", os.Args[0])
	ErrUnknownCommand      = errors.New("unknown command")
	ErrCommandNotFound     = errors.New("command not found")
	ErrFlagsParsingFailed  = errors.New("flags parsing failed")
	ErrAssigningArgsFailed = errors.New("assigning args failed")

	// ErrOmitUserNotify signals that the error has already been displayed to the user
	// in a user-friendly format, and the technical error message should be omitted
	// from user output (but can still be logged).
	ErrOmitUserNotify = errors.New("omit user notification")
)
