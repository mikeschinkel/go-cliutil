package cliutil

import (
	"time"
)

type Options interface {
	Options()
	Timeout() time.Duration
	Quiet() bool
	Verbosity() Verbosity
	DryRun() bool
	Force() bool
	Parse(Flags) error
}

const (
	DefaultTimeout   = 3
	DefaultQuiet     = false
	DefaultDryRun    = false
	DefaultForce     = false
	DefaultVerbosity = int(LowVerbosity)
)

var options = &GlobalOptions{
	timeout:   new(int),
	quiet:     new(bool),
	verbosity: new(int),
	dryRun:    new(bool),
	force:     new(bool),
}
