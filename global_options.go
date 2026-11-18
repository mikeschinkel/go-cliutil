package cliutil

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"
)

// Names of the global flags GlobalOptions declares.
const (
	VerbosityFlagName = "verbosity"
	QuietFlagName     = "quiet"
	TimeoutFlagName   = "timeout"
	DryRunFlagName    = "dry-run"
	ForceFlagName     = "force"
)

var (
	flagNameRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

	// ErrFlagTypeNotDiscoverable means a FlagDef does not set exactly one target.
	ErrFlagTypeNotDiscoverable = errors.New("flag type is not discoverable")
)

var _ DefaultOptions = (*GlobalOptions)(nil)

// GlobalOptions holds the global flags go-cliutil supplies. Embed it in an app's
// Options type; each run fills a fresh value.
type GlobalOptions struct {
	timeout   int
	quiet     bool
	verbosity int
	dryRun    bool
	force     bool
	omit      []string
	overrides []FlagDef
}

// GlobalOptionsArgs configures NewGlobalOptions.
type GlobalOptionsArgs struct {
	// Omit names built-in flags to drop. An omitted flag is not parsed or shown
	// in help, and calling its GlobalOptions method panics.
	Omit []string

	// Overrides changes built-in flags, matched by Name. A field left at its zero
	// value keeps the built-in setting, and a target may not be set.
	Overrides []FlagDef
}

// NewGlobalOptions creates a GlobalOptions holding the flags' defaults. Invalid
// Omit or Overrides entries are reported when the registry is validated.
func NewGlobalOptions(args GlobalOptionsArgs) *GlobalOptions {
	o := &GlobalOptions{
		omit:      slices.Clone(args.Omit),
		overrides: slices.Clone(args.Overrides),
	}
	for _, fd := range o.FlagDefs() {
		fd.assign(fd.defaultValue())
	}
	return o
}

// Options marks GlobalOptions as an Options value.
func (o *GlobalOptions) Options() {}

// globalOptions gives go-cliutil the GlobalOptions embedded in an app's type.
func (o *GlobalOptions) globalOptions() *GlobalOptions {
	return o
}

// Timeout returns the --timeout value. It panics when the flag is omitted.
func (o *GlobalOptions) Timeout() time.Duration {
	o.mustHave(TimeoutFlagName)
	return time.Duration(o.timeout) * time.Second
}

// Quiet returns the --quiet value. It panics when the flag is omitted.
func (o *GlobalOptions) Quiet() bool {
	o.mustHave(QuietFlagName)
	return o.quiet
}

// Verbosity returns the --verbosity value. It panics when the flag is omitted.
func (o *GlobalOptions) Verbosity() Verbosity {
	o.mustHave(VerbosityFlagName)
	return Verbosity(o.verbosity)
}

// DryRun returns the --dry-run value. It panics when the flag is omitted.
func (o *GlobalOptions) DryRun() bool {
	o.mustHave(DryRunFlagName)
	return o.dryRun
}

// Force returns the --force value. It panics when the flag is omitted.
func (o *GlobalOptions) Force() bool {
	o.mustHave(ForceFlagName)
	return o.force
}

// SetTimeout sets the timeout in seconds, typically from ApplyConfig.
func (o *GlobalOptions) SetTimeout(seconds int) {
	o.mustHave(TimeoutFlagName)
	o.timeout = seconds
}

// SetQuiet sets the quiet value, typically from ApplyConfig.
func (o *GlobalOptions) SetQuiet(quiet bool) {
	o.mustHave(QuietFlagName)
	o.quiet = quiet
}

// SetVerbosity sets the verbosity level, typically from ApplyConfig.
func (o *GlobalOptions) SetVerbosity(verbosity int) {
	o.mustHave(VerbosityFlagName)
	o.verbosity = verbosity
}

// SetDryRun sets the dry-run value, typically from ApplyConfig.
func (o *GlobalOptions) SetDryRun(dryRun bool) {
	o.mustHave(DryRunFlagName)
	o.dryRun = dryRun
}

// SetForce sets the force value, typically from ApplyConfig.
func (o *GlobalOptions) SetForce(force bool) {
	o.mustHave(ForceFlagName)
	o.force = force
}

// mustHave panics when the named flag is omitted, a programming error.
func (o *GlobalOptions) mustHave(name string) {
	if o.isOmitted(name) {
		panic(fmt.Sprintf("cliutil.GlobalOptions: the %q flag is omitted, so its method cannot be called", name))
	}
}

func (o *GlobalOptions) isOmitted(name string) bool {
	return slices.Contains(o.omit, name)
}

// FlagDefs returns the built-in flags, without omitted flags and with Overrides
// applied. Targets are built on each call so they always point at this value.
func (o *GlobalOptions) FlagDefs() (defs []FlagDef) {
	for _, fd := range o.builtinDefs() {
		if o.isOmitted(fd.Name) {
			continue
		}
		for _, override := range o.overrides {
			if override.Name == fd.Name {
				fd = mergeOverride(fd, override)
			}
		}
		defs = append(defs, fd)
	}
	return defs
}

func (o *GlobalOptions) builtinDefs() []FlagDef {
	return []FlagDef{
		{
			Name:           VerbosityFlagName,
			Shortcut:       'v',
			DefaultInt:     DefaultVerbosity,
			Usage:          "Verbosity of most command line output (1 to 3, default 1)",
			Int:            &o.verbosity,
			ValidationFunc: validateVerbosity,
		},
		{
			Name:     QuietFlagName,
			Shortcut: 'q',
			Usage:    "Disable display of most command line output",
			Bool:     &o.quiet,
		},
		{
			Name:       TimeoutFlagName,
			Shortcut:   't',
			DefaultInt: DefaultTimeout,
			Usage:      "timeout(in seconds) (TODO explain what this controls)",
			Int:        &o.timeout,
		},
		{
			Name:  DryRunFlagName,
			Usage: "Show what command results will be if command is run",
			Bool:  &o.dryRun,
		},
		{
			Name:     ForceFlagName,
			Shortcut: 'f',
			Usage:    "Force the action even if warnings",
			Bool:     &o.force,
		},
	}
}

// validateArgs reports Omit and Overrides entries that name no built-in flag or
// that set a target.
func (o *GlobalOptions) validateArgs() error {
	var errs []error
	builtins := o.builtinDefs()
	isBuiltin := func(name string) bool {
		return slices.ContainsFunc(builtins, func(fd FlagDef) bool { return fd.Name == name })
	}
	for _, name := range o.omit {
		if !isBuiltin(name) {
			errs = append(errs, fmt.Errorf("GlobalOptionsArgs.Omit: no built-in flag named '%s'", name))
		}
	}
	for _, override := range o.overrides {
		if !isBuiltin(override.Name) {
			errs = append(errs, fmt.Errorf("GlobalOptionsArgs.Overrides: no built-in flag named '%s'", override.Name))
		}
		if override.Type() != UnknownFlagType {
			errs = append(errs, fmt.Errorf("GlobalOptionsArgs.Overrides: flag '%s' may not set a target", override.Name))
		}
	}
	return CombineErrs(errs)
}

// mergeOverride returns fd with override's non-zero fields applied.
func mergeOverride(fd, override FlagDef) FlagDef {
	if override.Shortcut != 0 {
		fd.Shortcut = override.Shortcut
	}
	if override.Usage != "" {
		fd.Usage = override.Usage
	}
	if override.Required {
		fd.Required = true
	}
	if override.Regex != nil {
		fd.Regex = override.Regex
	}
	if override.ValidationFunc != nil {
		fd.ValidationFunc = override.ValidationFunc
	}
	if override.Example != "" {
		fd.Example = override.Example
	}
	if override.DefaultBool {
		fd.DefaultBool = true
	}
	if override.DefaultInt != 0 {
		fd.DefaultInt = override.DefaultInt
	}
	return fd
}

func validateVerbosity(value any) error {
	verbosity, ok := value.(int)
	if !ok {
		return ErrInvalidateVerbosity
	}
	_, err := ParseVerbosity(verbosity)
	return err
}

// globalOptionsHolder finds the GlobalOptions an app's type embeds.
type globalOptionsHolder interface {
	globalOptions() *GlobalOptions
}

// globalFlagDefs returns every global flag an Options value declares: the
// embedded GlobalOptions' flags, then the app's own.
func globalFlagDefs(opts Options) (defs []FlagDef, err error) {
	var errs []error

	holder, hasGlobals := opts.(globalOptionsHolder)
	if hasGlobals {
		global := holder.globalOptions()
		if global == nil {
			err = errors.New("the Options value embeds a nil *GlobalOptions; use cliutil.NewGlobalOptions")
			goto end
		}
		errs = AppendErr(errs, global.validateArgs())
		defs = global.FlagDefs()
	}

	if definer, ok := opts.(FlagDefiner); ok {
		builtins := len(defs)
		for _, fd := range definer.FlagDefs() {
			// An app that does not declare FlagDefs gets GlobalOptions' promoted
			// method, which returns the built-ins already collected.
			if slices.ContainsFunc(defs[:builtins], func(b FlagDef) bool { return b.sharesTarget(&fd) }) {
				continue
			}
			defs = append(defs, fd)
		}
	}
	err = CombineErrs(errs)

end:
	return defs, err
}

// runQuiet reports the run's quiet setting, or false when the app has none.
func runQuiet(opts Options) bool {
	return readDefaultOption(opts, QuietFlagName, DefaultQuiet, DefaultOptions.Quiet)
}

// runVerbosity reports the run's verbosity, or the default when the app has none.
func runVerbosity(opts Options) Verbosity {
	return readDefaultOption(opts, VerbosityFlagName, LowVerbosity, DefaultOptions.Verbosity)
}

// readDefaultOption reads a DefaultOptions value, returning fallback when the
// app's Options does not implement it or omits the flag.
func readDefaultOption[T any](opts Options, name string, fallback T, read func(DefaultOptions) T) T {
	defaults, ok := opts.(DefaultOptions)
	if !ok {
		return fallback
	}
	holder, ok := opts.(globalOptionsHolder)
	if ok && holder.globalOptions().isOmitted(name) {
		return fallback
	}
	return read(defaults)
}
