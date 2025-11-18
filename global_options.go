package cliutil

import (
	"strconv"
	"time"

	"github.com/mikeschinkel/go-dt"
)

//goland:noinspection GoUnusedExportedFunction
func GetCLIOptions() *CLIOptions {
	return options
}

var _ Options = (*CLIOptions)(nil)

type CLIOptions struct {
	timeout   *int
	quiet     *bool
	verbosity *int
	dryRun    *bool
	force     *bool
	//Strings   stringSliceFlag
}

func (o *CLIOptions) Options() {}

type CLIOptionsArgs struct {
	Quiet     *bool
	Verbosity *int
	Timeout   *int
	DryRun    *bool
	Force     *bool
}

// NewCLIOptions creates a new GlobalOptions instance from raw values.
// This is useful when loading options from configuration files or other sources.
// Any nil values will use the corresponding defaults.
func NewCLIOptions(args CLIOptionsArgs) (*CLIOptions, error) {
	verbosity := valueOrDefault(args.Verbosity, DefaultVerbosity)
	v, err := ParseVerbosity(verbosity)
	if err != nil {
		return nil, err
	}

	return &CLIOptions{
		quiet:     ptr(valueOrDefault(args.Quiet, DefaultQuiet)),
		verbosity: ptr(int(v)),
		timeout:   ptr(valueOrDefault(args.Timeout, DefaultTimeout)),
		dryRun:    ptr(valueOrDefault(args.DryRun, DefaultDryRun)),
		force:     ptr(valueOrDefault(args.Force, DefaultForce)),
	}, nil
}

func (o *CLIOptions) Timeout() time.Duration {
	return time.Duration(*o.timeout) * time.Second
}
func (o *CLIOptions) Quiet() bool {
	return *o.quiet
}
func (o *CLIOptions) Verbosity() Verbosity {
	return Verbosity(*o.verbosity)
}
func (o *CLIOptions) DryRun() bool {
	return *o.dryRun
}
func (o *CLIOptions) Force() bool {
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
// validated common.CLIOptions. This method performs validation and type conversion
// for all XMLUI Test Server options.
//
// Expects os.Args as input. Strips program name and defaults to ["help"] if no args.
func ParseOptions(osArgs []string) (_ *CLIOptions, _ []string, err error) {
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
	if err != nil {
		goto end
	}

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

	err = CombineErrs(errs)
end:
	return options, args, err
}
