package cliutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
)

// CLIUtilMode says how a command line selects what to run.
type CLIUtilMode int

const (
	UnspecifiedCLIUtilMode CLIUtilMode = iota // Behaves as CommandMode.
	CommandMode                               // Command words select a command.
	FlagsOnlyMode                             // The only registered command always runs.
)

// CmdRunner runs command lines against a Registry.
type CmdRunner struct {
	Args CmdRunnerArgs
	set  *CommandSet // The current run's commands; nil outside a run.
}

// CmdRunnerArgs configures a CmdRunner.
type CmdRunnerArgs struct {
	AppInfo  appinfo.AppInfo
	Logger   *slog.Logger
	Writer   Writer // Nil means a Writer on IO honoring the run's quiet and verbosity.
	Context  context.Context
	Registry *Registry   // Nil means DefaultRegistry().
	Config   ConfigFunc  // Nil means the run has no config.
	Options  OptionsFunc // Nil means NewGlobalOptions with no arguments.
	IO       dt.StdIO    // Nil means the process streams.
	Mode     CLIUtilMode
}

// NewCmdRunner creates a CmdRunner, filling unset arguments with their defaults.
func NewCmdRunner(args CmdRunnerArgs) *CmdRunner {
	if args.Registry == nil {
		args.Registry = DefaultRegistry()
	}
	if args.IO == nil {
		args.IO = dt.NewDefaultStdIO()
	}
	if args.Context == nil {
		args.Context = context.Background()
	}
	if args.Logger == nil {
		args.Logger = slog.New(slog.DiscardHandler)
	}
	if args.Options == nil {
		args.Options = func() Options { return NewGlobalOptions(GlobalOptionsArgs{}) }
	}
	if args.AppInfo == nil {
		args.AppInfo = appinfo.New(appinfo.Args{
			Name:    filepath.Base(os.Args[0]),
			ExeName: dt.Filename(filepath.Base(os.Args[0])),
		})
	}
	return &CmdRunner{Args: args}
}

// Commands returns the current run's commands.
func (cr *CmdRunner) Commands() *CommandSet {
	return cr.set
}

// ParsedCmd is a command line resolved and ready to run.
type ParsedCmd struct {
	Command Command
	args    HandleArgs
}

// Run parses words, the command line without the program name, and runs the
// command they select. Each call builds fresh
// command and Options values, so a command can use it to run another command.
func (cr *CmdRunner) Run(words []string) (err error) {
	var parsed *ParsedCmd

	parsed, err = cr.ParseCmd(words)
	if err != nil {
		goto end
	}
	err = cr.RunCmd(parsed)

end:
	return err
}

// ParseCmd builds this run's values, parses words and resolves every flag and
// operand. A flag's value comes from the command line, else config, else its
// typed default.
func (cr *CmdRunner) ParseCmd(words []string) (parsed *ParsedCmd, err error) {
	var opts Options
	var globalDefs []FlagDef
	var set *CommandSet
	var line *parsedLine
	var config Config
	var flags Flags
	var writer Writer

	opts = cr.Args.Options()
	globalDefs, err = globalFlagDefs(opts)
	if err != nil {
		err = WithErr(err, ErrCommandRegistrationFailed)
		goto end
	}

	set, err = newCommandSet(cr.Args.Registry.snapshot(), globalDefs)
	if err != nil {
		goto end
	}

	line, err = (&lineParser{
		set:     set,
		globals: newFlagLookup(globalDefs),
		mode:    cr.Args.Mode,
	}).parse(words)
	if err != nil {
		err = WithErr(err, ErrFlagsParsingFailed, ErrShowUsage, "command", strings.Join(words, " "))
		goto end
	}

	writer = cr.Args.Writer

	err = assignOperands(line.cmd, line.operands)
	if err != nil {
		cr.showCmdUsage(writer, opts, line.cmd, set)
		err = WithErr(err, ErrFlagsParsingFailed, ErrAssigningArgsFailed)
		goto end
	}

	config, flags, err = cr.resolveFlags(opts, line, globalDefs)
	if err != nil {
		goto end
	}

	if writer == nil {
		writer = cr.newRunWriter(opts)
	}

	parsed = &ParsedCmd{
		Command: line.cmd,
		args: HandleArgs{
			Context:     cr.Args.Context,
			IO:          cr.Args.IO,
			Writer:      writer,
			Logger:      cr.Args.Logger,
			Config:      config,
			Options:     opts,
			Runner:      &CmdRunner{Args: cr.Args, set: set},
			CommandPath: line.path,
			CommandLine: words,
			Parameters:  line.parameters,
			Flags:       flags,
			Operands:    line.operands,
		},
	}

end:
	return parsed, err
}

// resolveFlags applies defaults, config and typed values, in that order, and
// returns the config and every flag's final value.
func (cr *CmdRunner) resolveFlags(opts Options, line *parsedLine, globalDefs []FlagDef) (config Config, flags Flags, err error) {
	var errs []error
	var suppliedByValuer map[*FlagDef]bool

	defs := newFlagLookup(globalDefs)
	globalCount := len(defs)
	defs = append(defs, commandFlagLookup(line.cmd)...)

	typedByDef := make(map[*FlagDef]*typedFlag, len(line.typed))
	for _, typed := range line.typed {
		typedByDef[typed.def] = typed
	}

	for _, fd := range defs {
		fd.assign(fd.defaultValue())
	}

	config, err = cr.loadConfig(defs, globalCount, typedByDef, line)
	if err != nil {
		goto end
	}

	suppliedByValuer, err = applyConfig(config, opts, line.cmd, defs, typedByDef)
	if err != nil {
		goto end
	}

	for _, typed := range line.typed {
		typed.def.assign(typed.value)
	}

	for i, fd := range defs {
		flag := Flag{Name: fd.Name, Global: i < globalCount, Def: fd, Value: fd.current()}
		typed, wasTyped := typedByDef[fd]
		switch {
		case wasTyped:
			flag.Source = CommandLineSource
			flag.Words = typed.words
		case suppliedByValuer[fd], fd.differsFromDefault():
			flag.Source = ConfigSource
		default:
			flag.Source = DefaultSource
		}
		if fd.Required && flag.Source == DefaultSource {
			errs = append(errs, NewErr(ErrFlagsParsingFailed, "flag_name", fd.Name,
				fmt.Errorf("flag --%s is required", fd.Name)))
		}
		if flag.Source != DefaultSource {
			errs = AppendErr(errs, fd.ValidateValue(flag.Value))
		}
		flags.all = append(flags.all, flag)
	}
	for _, typed := range line.typed {
		flags.typed = append(flags.typed, typed.def.Name)
	}

	err = CombineErrs(errs)
	if err != nil {
		err = WithErr(err, ErrFlagsParsingFailed)
	}

end:
	return config, flags, err
}

// loadConfig calls the app's ConfigFunc with the flags parsed so far.
func (cr *CmdRunner) loadConfig(defs flagLookup, globalCount int, typedByDef map[*FlagDef]*typedFlag, line *parsedLine) (config Config, err error) {
	var parsedFlags Flags

	if cr.Args.Config == nil {
		goto end
	}
	for i, fd := range defs {
		flag := Flag{Name: fd.Name, Global: i < globalCount, Def: fd, Value: fd.defaultValue()}
		if typed, ok := typedByDef[fd]; ok {
			flag.Value = typed.value
			flag.Source = CommandLineSource
			flag.Words = typed.words
		}
		parsedFlags.all = append(parsedFlags.all, flag)
	}
	for _, typed := range line.typed {
		parsedFlags.typed = append(parsedFlags.typed, typed.def.Name)
	}

	config, err = cr.Args.Config(ConfigArgs{
		AppInfo: cr.Args.AppInfo,
		Flags:   parsedFlags,
		IO:      cr.Args.IO,
	})
	if err != nil {
		err = WithErr(err, ErrConfigLoadFailed)
	}

end:
	return config, err
}

// applyConfig runs the ApplyConfig hooks, then fills flags the user did not type
// from a FlagValuer. It returns the flags the FlagValuer supplied.
func applyConfig(config Config, opts Options, cmd Command, defs flagLookup, typedByDef map[*FlagDef]*typedFlag) (supplied map[*FlagDef]bool, err error) {
	var valuer FlagValuer
	var ok bool

	supplied = make(map[*FlagDef]bool)
	if config == nil {
		goto end
	}

	for _, target := range []any{opts, cmd} {
		applier, isApplier := target.(ConfigApplier)
		if !isApplier {
			continue
		}
		err = applier.ApplyConfig(config)
		if err != nil {
			err = WithErr(err, ErrConfigApplyFailed)
			goto end
		}
	}

	valuer, ok = config.(FlagValuer)
	if !ok {
		goto end
	}
	for _, fd := range defs {
		if _, typed := typedByDef[fd]; typed {
			continue
		}
		value, found := valuer.FlagValue(fd.Name)
		if !found {
			continue
		}
		var converted any
		converted, err = fd.coerce(value)
		if err != nil {
			err = WithErr(err, ErrConfigApplyFailed)
			goto end
		}
		fd.assign(converted)
		supplied[fd] = true
	}

end:
	return supplied, err
}

// newRunWriter builds the run's Writer from its quiet and verbosity settings.
func (cr *CmdRunner) newRunWriter(opts Options) Writer {
	return NewWriter(&WriterArgs{
		IO:        cr.Args.IO,
		Quiet:     runQuiet(opts),
		Verbosity: max(runVerbosity(opts), LowVerbosity),
	})
}

// showCmdUsage writes the command's usage to the run's error stream.
func (cr *CmdRunner) showCmdUsage(writer Writer, opts Options, cmd Command, set *CommandSet) {
	if writer == nil {
		writer = cr.newRunWriter(opts)
	}
	dt.LogOnErr(CmdUsageTemplate.Execute(writer.ErrWriter(), BuildCmdUsage(cmd, set)))
}

// RunCmd calls the parsed command's handler.
func (cr *CmdRunner) RunCmd(parsed *ParsedCmd) (err error) {
	handler, ok := parsed.Command.(CommandHandler)
	if !ok {
		err = fmt.Errorf("command '%s' does not implement handler logic", parsed.Command.Name())
		goto end
	}
	err = handler.Handle(parsed.args)

end:
	return err
}

// ExitCode returns the process exit code for an error a CmdRunner returned.
func ExitCode(err error) (code int) {
	switch {
	case err == nil:
		code = ExitSuccess
	case errors.Is(err, ErrCommandRegistrationFailed):
		code = ExitInitializationError
	case errors.Is(err, ErrFlagsParsingFailed):
		code = ExitFlagsParseError
	case errors.Is(err, ErrConfigLoadFailed):
		code = ExitConfigLoadError
	case errors.Is(err, ErrConfigApplyFailed):
		code = ExitConfigParseError
	default:
		code = ExitKnownRuntimeError
	}
	return code
}

// ShowMainHelp displays the main help screen
func ShowMainHelp(args UsageArgs) error {
	return UsageTemplate.Execute(args.Writer.Writer(), BuildUsage(args))
}

// ShowCmdHelp displays help for a specific command
// cmdNameParts is a slice of command name parts that will be joined with "."
// For example: ["demo", "list"] becomes "demo.list"
func ShowCmdHelp(cmdNameParts []string, args UsageArgs) (err error) {
	var cmdName string
	var cmd Command

	if len(cmdNameParts) == 0 {
		err = fmt.Errorf("no command specified for help")
		goto end
	}

	// Build dot-notation path from parts
	cmdName = strings.Join(cmdNameParts, ".")

	cmd = args.Commands.GetExactCommand(cmdName)
	if cmd == nil {
		err = fmt.Errorf("unknown command: %s", cmdName)
		goto end
	}

	// Hidden commands should not show help
	if cmd.IsHidden() {
		err = fmt.Errorf("unknown command: %s", cmdName)
		goto end
	}

	err = CmdUsageTemplate.Execute(args.Writer.Writer(), BuildCmdUsage(cmd, args.Commands))

end:
	return err
}
