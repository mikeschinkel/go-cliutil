package cliutil

type Options interface {
	Options()
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
