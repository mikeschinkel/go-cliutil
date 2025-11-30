package cliutil

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/go-dt/dtx"
)

// CLAUDE: I renamed to globalFlags because "Handler"  is GARBAJE.
// CLAUDE: Also, I restructured and renamed interface because gmover-specific flags should not be encoded into generic cliutil

//type GlobalFlagDefGetter interface {
//	GlobalFlagDefs() []FlagDef
//}

type CmdRunner struct {
	Args CmdRunnerArgs
}

type CmdRunnerArgs struct {
	AppInfo appinfo.AppInfo
	Logger  *slog.Logger
	Writer  Writer
	Context context.Context
	Config  Config
	Options Options
	Args    []string
}

func NewCmdRunner(args CmdRunnerArgs) *CmdRunner {
	return &CmdRunner{
		Args: args,
	}
}

func (cr CmdRunner) ParseCmd(args []string) (cmd Command, err error) {
	var path string

	if len(args) == 0 {
		args = []string{"help"}
	}
	osArgs := args

	// Validate commands first
	err = ValidateCmds()
	if err != nil {
		goto end
	}

	// Try to find the most specific command match
	path, args = findBestCmdMatch(args)
	if path == "" {
		err = NewErr(
			ErrUnknownCommand,
			"command_args", args,
		)
		goto end
	}

	cmd, _ = GetDefaultCommand(path, args)
	if cmd == nil {
		err = NewErr(
			ErrCommandNotFound,
			"command", path,
			"command_args", args,
		)
		goto end
	}

	args, err = cmd.ParseFlagSets(args)
	if err != nil {
		err = NewErr(ErrFlagsParsingFailed)
		goto end
	}

	// Validate original flags against known flags
	err = cr.validateFlags(cmd)
	if err != nil {
		goto end
	}

	err = cmd.AssignArgs(args)
	if err != nil {
		err = NewErr(ErrAssigningArgsFailed)
		goto end
	}

end:
	if err != nil {
		err = WithErr(err,
			ErrShowUsage,
			"command", strings.Join(osArgs, " "),
		)
	}
	return cmd, err
}

func (cr CmdRunner) RunCmd(cmd Command) (err error) {
	var handler CommandHandler
	var ok bool
	var args []string

	// Command resolution should ensure we only get CommandHandler implementations
	handler, ok = cmd.(CommandHandler)
	if !ok {
		err = fmt.Errorf("command '%s' does not implement handler logic", cmd.Name())
		goto end
	}

	// If the cmd is the Help command, remove "help" as the first element
	args = cr.Args.Args
	if cmd.Name() == "help" && len(args) != 0 && args[0] == "help" {
		cr.Args.Args = args[1:]
	}
	handler.SetCommandRunnerArgs(cr.Args)

	err = handler.Handle()

end:
	return err
}

type CLIOptionsGetter interface {
	CLIOptions() *CLIOptions
}

// validateFlags checks original flags against all known flags
func (cr CmdRunner) validateFlags(cmd Command) (err error) {
	var getter CLIOptionsGetter
	var originalFlags []string
	var knownFlags []string
	var globalFlagSet *FlagSet
	var cmdFlagSets []*FlagSet
	var flagSet *FlagSet
	var unknownFlags []string
	var flag string
	var flagName string
	var equalPos int
	var isKnown bool
	var known string
	var flagList string

	// Get original flags from options
	getter, err = dtx.AssertType[CLIOptionsGetter](cr.Args.Options)
	if err != nil {
		goto end
	}

	originalFlags = getter.CLIOptions().originalFlags
	if len(originalFlags) == 0 {
		goto end
	}

	// Collect all known flag names
	globalFlagSet = GetFlagSet()
	if globalFlagSet != nil {
		knownFlags = append(knownFlags, globalFlagSet.FlagNames()...)
	}

	cmdFlagSets = cmd.FlagSets()
	for _, flagSet = range cmdFlagSets {
		knownFlags = append(knownFlags, flagSet.FlagNames()...)
	}

	// Check each original flag against known flags
	for _, flag = range originalFlags {
		// Extract flag name (remove - prefix and =value suffix)
		flagName = strings.TrimPrefix(flag, "-")
		flagName = strings.TrimPrefix(flagName, "-")
		equalPos = strings.Index(flagName, "=")
		if equalPos != -1 {
			flagName = flagName[:equalPos]
		}

		// Check if flag is known
		isKnown = false
		for _, known = range knownFlags {
			if known == flagName {
				isKnown = true
				break
			}
		}

		if !isKnown {
			unknownFlags = append(unknownFlags, flag)
		}
	}

	// Report unknown flags
	if len(unknownFlags) > 0 {
		flagList = strings.Join(unknownFlags, ", ")
		err = fmt.Errorf("unknown flag(s): %s", flagList)
		goto end
	}

end:
	return err
}

// findBestCmdMatch finds the longest matching command path
func findBestCmdMatch(args []string) (path string, remainingArgs []string) {
	var cmd Command
	var tryPath string
	var n int
	tryPaths := make([]string, len(args))

	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		tryPath = fmt.Sprintf("%s.%s", tryPath, arg)
		if i == 0 {
			tryPath = strings.TrimLeft(tryPath, ".")
		}
		n++
		tryPaths[len(tryPaths)-i-1] = tryPath
	}
	if n < len(args) {
		tryPaths = tryPaths[len(tryPaths)-n:]
	}

	// Try progressively longer paths
	for _, p := range tryPaths {
		cmd, p = GetDefaultCommand(p, args)
		if cmd != nil {
			path = p
			remainingArgs = args[n:]
			break
		}
		n--
	}

	// If no match found, return empty path with original osArgs
	if path == "" {
		remainingArgs = args
	}

	return path, remainingArgs
}

// ShowMainHelp displays the main help screen
func ShowMainHelp(args UsageArgs) error {
	return UsageTemplate.Execute(args.Writer.Writer(), BuildUsage(args))
}

// ShowCmdHelp displays help for a specific command
func ShowCmdHelp(cmdName string, args UsageArgs) (err error) {
	cmd := GetExactCommand(cmdName)
	if cmd == nil {
		err = fmt.Errorf("unknown command: %s", cmdName)
		goto end
	}

	// Hidden commands should not show help
	if cmd.IsHidden() {
		err = fmt.Errorf("unknown command: %s", cmdName)
		goto end
	}

	err = CmdUsageTemplate.Execute(args.Writer.Writer(), BuildCmdUsage(cmd))

end:
	return err
}
