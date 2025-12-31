package cliutil

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mikeschinkel/go-dt"
)

var gFlags = &GlobalFlags{
	timeout:   new(int),
	quiet:     new(bool),
	verbosity: new(int),
	dryRun:    new(bool),
	force:     new(bool),
}

//goland:noinspection GoUnusedExportedFunction
func GetGlobalFlags() *GlobalFlags {
	return gFlags
}

type GlobalFlags struct {
	timeout       *int
	quiet         *bool
	verbosity     *int
	dryRun        *bool
	force         *bool
	originalFlags []string // Flags from original command line for validation
	//Strings   stringSliceFlag
}

func (o *GlobalFlags) Flags() {}

type GlobalFlagsArgs struct {
	Quiet     *bool
	Verbosity *int
	Timeout   *int
	DryRun    *bool
	Force     *bool
}

// NewGlobalFlags creates a new GlobalFlags instance from raw values.
// This is useful when loading flags from configuration files or other sources.
// Any nil values will use the corresponding defaults.
func NewGlobalFlags(args GlobalFlagsArgs) (*GlobalFlags, error) {
	verbosity := valueOrDefault(args.Verbosity, DefaultVerbosity)
	v, err := ParseVerbosity(verbosity)
	if err != nil {
		return nil, err
	}

	return &GlobalFlags{
		quiet:     ptr(valueOrDefault(args.Quiet, DefaultQuiet)),
		verbosity: ptr(int(v)),
		timeout:   ptr(valueOrDefault(args.Timeout, DefaultTimeout)),
		dryRun:    ptr(valueOrDefault(args.DryRun, DefaultDryRun)),
		force:     ptr(valueOrDefault(args.Force, DefaultForce)),
	}, nil
}

func (o *GlobalFlags) Timeout() time.Duration {
	return time.Duration(*o.timeout) * time.Second
}
func (o *GlobalFlags) Quiet() bool {
	return *o.quiet
}
func (o *GlobalFlags) Verbosity() Verbosity {
	return Verbosity(*o.verbosity)
}
func (o *GlobalFlags) DryRun() bool {
	return *o.dryRun
}
func (o *GlobalFlags) Force() bool {
	return *o.force
}

//goland:noinspection GoUnusedExportedFunction
func GetGlobalFlagsSet() *FlagSet {
	return gFlagSets
}

//var (
//	flagNameRegex = regexp.MustCompile(`^[a-z0-9-]+$`)
//)

var gFlagSets = &FlagSet{
	Name: "global",
	FlagDefs: []FlagDef{
		{
			Name:     "verbosity",
			Shortcut: 'v',
			Default:  DefaultVerbosity,
			Usage:    "Verbosity of most command line output (1 to 3, default 1)",
			Int:      gFlags.verbosity,
		},
		{
			Name:     "quiet",
			Shortcut: 'q',
			Default:  DefaultQuiet,
			Usage:    "Disable display of most command line output",
			Bool:     gFlags.quiet,
		},
		{
			Name:     "timeout",
			Shortcut: 't',
			Default:  DefaultTimeout,
			Usage:    "timeout(in seconds) (TODO explain what this controls)",
			Int:      gFlags.timeout,
		},
		{
			Name:    "dry-run",
			Default: DefaultDryRun,
			Usage:   "Show what command results will be if command is run",
			Bool:    gFlags.dryRun,
		},
		{
			Name:     "force",
			Shortcut: 'f',
			Default:  DefaultForce,
			Usage:    "Force the action even if warnings",
			Bool:     gFlags.force,
		},
	},
}

func AddCLIFlag(flagDef FlagDef) (err error) {
	var errs []error
	var types []string
	var existing FlagDef

	// Validate Name: lowercase alphanumeric + dashes only
	if flagDef.Name == "" {
		errs = append(errs, NewErr(dt.ErrEmpty, "empty_property", "Name"))
	} else if !flagNameRegex.MatchString(flagDef.Name) {
		errs = append(errs, NewErr(dt.ErrInvalidFlagName, "rule", "may contain only lowercase letters, numbers, and dashes"))
	}

	// Validate no duplicate flag names
	for _, existing = range flagSet.FlagDefs {
		if existing.Name == flagDef.Name {
			errs = append(errs, NewErr(dt.ErrInvalidDuplicateFlag, "where", "global flags"))
			break
		}
	}

	// Validate exactly one type is set
	if flagDef.String != nil {
		types = append(types, "string")
	}
	if flagDef.Bool != nil {
		types = append(types, "bool")
	}
	if flagDef.Int != nil {
		types = append(types, "int")
	}
	if flagDef.Int64 != nil {
		types = append(types, "int64")
	}
	rule := "exactly one property of .String, .Bool, .Int, or .Int64 must be non-nil"
	switch len(types) {
	case 0:
		errs = append(errs,
			NewErr(ErrFlagTypeNotDiscoverable, "rule", rule),
		)
	case 1:
		// Success - exactly one type is set
	default:
		errs = append(errs,
			NewErr(ErrFlagTypeNotDiscoverable, "rule", rule, "duplicates", strings.Join(types, ", ")),
		)
	}

	// Validate Usage is not empty
	if strings.TrimSpace(flagDef.Usage) == "" {
		errs = append(errs, NewErr(dt.ErrEmpty, "empty_property", "Usage"))
	}

	err = CombineErrs(errs)
	if err != nil {
		goto end
	}
	flagSet.FlagDefs = append(flagSet.FlagDefs, flagDef)
end:
	if err != nil {
		err = WithErr(err, dt.ErrFlagValidationFailed, "flag_name", flagDef.Name)
	}
	return err
}

//var ErrFlagTypeNotDiscoverable = errors.New("flag type is not discoverable")

// ParseGlobalFlags converts raw flags into GlobalFlags.
//
// Uses os.Args for as input. Not designed to be testable
func ParseGlobalFlags(mode CLIUtilMode) (_ *GlobalFlags, _ []string, err error) {
	return ParseGlobalFlagsFromArgs(os.Args, mode)
}

//var ErrFlagTypeNotDiscoverable = errors.New("flag type is not discoverable")

type CLIUtilMode int

const (
	UnspecifiedCLIUtilMode CLIUtilMode = iota
	CommandMode
	FlagsOnlyMode
)

// ParseGlobalFlagsFromArgs converts raw flags into GlobalFlags.
//
// Expects os.Args or equivalent as input. Designed to be testable
func ParseGlobalFlagsFromArgs(osArgs []string, mode CLIUtilMode) (_ *GlobalFlags, _ []string, err error) {
	var errs []error
	var timeout time.Duration
	var verbosity Verbosity
	var args []string
	var helpRequested bool

	// Strip program name from os.Args
	if len(osArgs) > 0 {
		args = osArgs[1:]
	}

	// Transform flag commands (e.g., --test-hidden -> test-hidden) BEFORE flag parsing
	args = transformFlagCommands(args)

	if mode == CommandMode {
		// Check for --help and handle it first
		helpRequested, args = containsHelpFlag(args)
		if helpRequested {
			args = append([]string{"help"}, args...)
		}
	}
	// Extract and save original flags for later validation (after --help is removed)
	gFlags.originalFlags = extractFlags(args)

	// Default to help command if no args provided
	if mode == CommandMode {
		if len(args) == 0 {
			args = []string{"help"}
		}
	}

	args, err = gFlagSets.Parse(args)
	if err != nil {
		goto end
	}

	timeout, err = dt.ParseTimeDurationEx(strconv.Itoa(*gFlags.timeout))
	errs = AppendErr(errs, err)
	if err == nil {
		*gFlags.timeout = int(timeout.Seconds())
	}

	verbosity, err = ParseVerbosity(*gFlags.verbosity)
	errs = AppendErr(errs, err)
	if err == nil {
		*gFlags.verbosity = int(verbosity)
	}

	err = CombineErrs(errs)

	if err != nil && gFlags == nil {
		// This should never happen as ParseGlobalOptions returns an error if it fails
		panic("cliutil.ParseGlobalFlags() would have return nil without error (this is a programming error.)")
	}

end:
	return gFlags, args, err
}
