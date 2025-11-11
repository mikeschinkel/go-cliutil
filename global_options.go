package cliutil

import (
	"strconv"
	"time"

	"github.com/mikeschinkel/go-dt"
)

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

//goland:noinspection GoUnusedExportedFunction
func GetGlobalOptions() *GlobalOptions {
	return options
}

type GlobalOptions struct {
	timeout   *int
	quiet     *bool
	verbosity *int
	dryRun    *bool
	force     *bool
	//Strings   stringSliceFlag
}

func (o *GlobalOptions) Options() {}

type GlobalOptionsArgs struct {
	Quiet     *bool
	Verbosity *int
	Timeout   *int
	DryRun    *bool
	Force     *bool
}

// NewGlobsalOptions creates a new GlobalOptions instance from raw values.
// This is useful when loading options from configuration files or other sources.
// Any nil values will use the corresponding defaults.
func NewGlobsalOptions(args GlobalOptionsArgs) (*GlobalOptions, error) {
	verbosity := valueOrDefault(args.Verbosity, DefaultVerbosity)
	v, err := ParseVerbosity(verbosity)
	if err != nil {
		return nil, err
	}

	return &GlobalOptions{
		quiet:     ptr(valueOrDefault(args.Quiet, DefaultQuiet)),
		verbosity: ptr(int(v)),
		timeout:   ptr(valueOrDefault(args.Timeout, DefaultTimeout)),
		dryRun:    ptr(valueOrDefault(args.DryRun, DefaultDryRun)),
		force:     ptr(valueOrDefault(args.Force, DefaultForce)),
	}, nil
}

func (o *GlobalOptions) Timeout() time.Duration {
	return time.Duration(*o.timeout) * time.Second
}
func (o *GlobalOptions) Quiet() bool {
	return *o.quiet
}
func (o *GlobalOptions) Verbosity() Verbosity {
	return Verbosity(*o.verbosity)
}
func (o *GlobalOptions) DryRun() bool {
	return *o.dryRun
}
func (o *GlobalOptions) Force() bool {
	return *o.force
}

//goland:noinspection GoUnusedExportedFunction
func GetFlagSet() *FlagSet {
	return flagset
}

var flagset = &FlagSet{
	Name: "global",
	FlagDefs: []FlagDef{
		{
			Name:     "verbosity",
			Shortcut: 'v',
			Default:  DefaultVerbosity,
			Usage:    "Verbosity of most command line output (1 to 3, default 1)",
			Int:      options.verbosity,
		},
		{
			Name:     "quiet",
			Shortcut: 'q',
			Default:  DefaultQuiet,
			Usage:    "Disable display of most command line output",
			Bool:     options.quiet,
		},
		{
			Name:     "timeout",
			Shortcut: 't',
			Default:  DefaultTimeout,
			Usage:    "timeout(in seconds) (TODO explain what this controls)",
			Int:      options.timeout,
		},
		{
			Name:    "dry-run",
			Default: DefaultDryRun,
			Usage:   "Show what command results will be if command is run",
			Bool:    options.dryRun,
		},
		{
			Name:     "force",
			Shortcut: 'f',
			Default:  DefaultForce,
			Usage:    "Force the action even if warnings",
			Bool:     options.force,
		},
	},
}

// ParseOptions converts raw options from cfgldr.Options into
// validated common.GlobalOptions. This method performs validation and type conversion
// for all XMLUI Test Server options.
//
// Expects os.Args as input. Strips program name and defaults to ["help"] if no args.
func ParseOptions(osArgs []string) (_ *GlobalOptions, _ []string, err error) {
	var errs []error
	var timeout time.Duration
	var verbosity Verbosity
	var args []string

	// Strip program name from os.Args
	if len(osArgs) > 0 {
		args = osArgs[1:]
	}

	// Default to help command if no args provided
	if len(args) == 0 {
		args = []string{"help"}
	}

	args, err = flagset.Parse(args)

	timeout, err = dt.ParseTimeDurationEx(strconv.Itoa(*options.timeout))
	errs = AppendErr(errs, err)
	if err == nil {
		*options.timeout = int(timeout.Seconds())
	}

	verbosity, err = ParseVerbosity(*options.verbosity)
	errs = AppendErr(errs, err)
	if err == nil {
		*options.verbosity = int(verbosity)
	}

	return options, args, CombineErrs(errs)
}
