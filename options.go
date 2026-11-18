package cliutil

import (
	"time"
)

// Options marks the value that holds an app's global flags. An app's type
// usually embeds GlobalOptions.
type Options interface {
	Options()
}

// DefaultOptions declares the global flags go-cliutil supplies.
type DefaultOptions interface {
	Options
	Timeout() time.Duration
	Quiet() bool
	Verbosity() Verbosity
	DryRun() bool
	Force() bool
}

// OptionsFunc builds a fresh Options value for one run.
type OptionsFunc func() Options

// FlagDefiner is implemented by an Options value that declares global flags.
// Each FlagDef targets a field of that same value.
type FlagDefiner interface {
	FlagDefs() []FlagDef
}

const (
	DefaultTimeout   = 3
	DefaultQuiet     = false
	DefaultDryRun    = false
	DefaultForce     = false
	DefaultVerbosity = int(LowVerbosity)
)
