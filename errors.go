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
)
