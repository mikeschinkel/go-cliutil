package cliutil

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mikeschinkel/go-dt"
)

//goland:noinspection GoUnusedExportedFunction
func GetCLIOptions() *CLIOptions {
	return options
}

var _ Options = (*CLIOptions)(nil)

type CLIOptions struct {
	timeout       *int
	quiet         *bool
	verbosity     *int
	dryRun        *bool
	force         *bool
	originalFlags []string // Flags from original command line for validation
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

var (
	flagNameRegex = regexp.MustCompile(`^[a-z0-9-]+$`)
)

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

func AddCLIOption(flagDef FlagDef) (err error) {
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
	for _, existing = range flagset.FlagDefs {
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
	flagset.FlagDefs = append(flagset.FlagDefs, flagDef)
end:
	if err != nil {
		err = WithErr(err, dt.ErrFlagValidationFailed, "flag_name", flagDef.Name)
	}
	return err
}

var ErrFlagTypeNotDiscoverable = errors.New("flag type is not discoverable")

// ParseCLIOptions converts raw options into CLIOptions.
//
// Expects os.Args as input. Strips program name and defaults to ["help"] if no args.
func ParseCLIOptions(osArgs []string) (_ *CLIOptions, _ []string, err error) {
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

	// Check for --help and handle it first
	helpRequested, args = containsHelpFlag(args)
	if helpRequested {
		args = append([]string{"help"}, args...)
	}

	// Extract and save original flags for later validation (after --help is removed)
	options.originalFlags = extractFlags(args)

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

// extractFlags returns all args that start with '-' (flags only, not values)
func extractFlags(args []string) (flags []string) {
	var arg string

	for _, arg = range args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		}
	}

	return flags
}

// transformFlagCommands checks if first arg is a flag command (e.g., --test-hidden)
// and transforms it to a command name (e.g., test-hidden) BEFORE flagset.Parse() consumes it
func transformFlagCommands(args []string) (transformed []string) {
	var firstArg string
	var flagName string
	var cmd Command
	var globalFS *FlagSet
	var fd FlagDef
	var found bool

	transformed = args
	if len(args) == 0 {
		goto end
	}

	firstArg = args[0]
	if !strings.HasPrefix(firstArg, "--") {
		goto end
	}

	// Extract flag name (remove -- prefix)
	flagName = strings.TrimPrefix(firstArg, "--")

	// Check if any registered command has this FlagName
	for _, cmd = range RegisteredCommands() {
		if cmd.FlagName() != flagName {
			continue
		}

		// Verify this flag exists in global flagset
		globalFS = GetFlagSet()
		if globalFS == nil {
			goto end
		}

		found = false
		for _, fd = range globalFS.FlagDefs {
			if fd.Name == flagName {
				found = true
				break
			}
		}

		if !found {
			// Flag command registered but flag not found - error will be caught later
			goto end
		}

		// Transform: replace --flagname with command name
		transformed = append([]string{cmd.Name()}, args[1:]...)
		goto end
	}

end:
	return transformed
}

// containsHelpFlag checks if --help is in args and removes it
func containsHelpFlag(args []string) (helpRequested bool, filteredArgs []string) {
	var i int
	var arg string

	filteredArgs = args

	for i, arg = range args {
		if strings.HasPrefix(arg, "--help") {
			filteredArgs = append(args[:i], args[i+1:]...)
			helpRequested = true
			goto end
		}
	}

end:
	return helpRequested, filteredArgs
}
