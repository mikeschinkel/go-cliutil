package cliutil

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// HandleArgs is everything a command receives for one run.
type HandleArgs struct {
	// Context is the run's context; it is never nil.
	Context context.Context

	// IO holds the run's input, output and error streams.
	IO dt.StdIO

	// Writer writes user-facing output to the run's streams.
	Writer Writer

	// Logger is the run's logger; it is never nil.
	Logger *slog.Logger

	// Config is the value the app's ConfigFunc returned, or nil without one.
	Config Config

	// Options holds the run's global flags; read it with OptionsAs.
	Options Options

	// Runner lets this command run another command line.
	Runner *CmdRunner

	// CommandPath holds the resolved command's names, such as ["repo","setup"]. A
	// default subcommand's name appears here even though the user did not type it.
	CommandPath []string

	// CommandLine holds every word after the program name, exactly as typed, such
	// as ["repo","setup","--check","."].
	CommandLine []string

	// Parameters holds every word after the last typed command word, flags
	// included, such as ["--check","."].
	Parameters []string

	// Flags holds every flag that applies to this run, global and command.
	Flags Flags

	// Operands holds the positional words only, such as ["."]. Under Passthrough
	// it holds every word in Parameters.
	Operands Operands
}

// Flag returns the named flag's value as T. It panics, naming the flag, command
// path and types, when the flag is undefined or T is wrong, since either is a
// programming error.
func (a HandleArgs) Flag[T any](name string) T {
	command := strings.Join(a.CommandPath, " ")
	flag, ok := a.Flags.Get(name)
	if !ok {
		panic(fmt.Sprintf("cliutil.HandleArgs.Flag: command %q has no flag %q", command, name))
	}
	value, ok := flag.Value.(T)
	if !ok {
		panic(fmt.Sprintf("cliutil.HandleArgs.Flag: command %q flag %q is %T, not %s",
			command, name, flag.Value, reflect.TypeFor[T]()))
	}
	return value
}

// OptionsAs returns the run's Options as T. It panics naming both types when the
// Options is not a T, since that is a programming error.
func (a HandleArgs) OptionsAs[T any]() T {
	value, ok := any(a.Options).(T)
	if !ok {
		panic(fmt.Sprintf("cliutil.HandleArgs.OptionsAs: options are %T, not %s", a.Options, reflect.TypeFor[T]()))
	}
	return value
}

// UsageArgs returns the input ShowMainHelp and ShowCmdHelp need for this run.
func (a HandleArgs) UsageArgs() UsageArgs {
	return UsageArgs{
		AppInfo:  a.Runner.Args.AppInfo,
		Writer:   a.Writer,
		Commands: a.Runner.Commands(),
	}
}
