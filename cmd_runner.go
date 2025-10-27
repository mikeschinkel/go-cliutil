package cliutil

import (
	"context"
	"fmt"
	"strings"
)

// CLAUDE: I renamed to globalFlags because "Handler"  is GARBAJE.
// CLAUDE: Also, I restructured and renamed interface because gmover-specific flags should not be encoded into generic cliutil

//type GlobalFlagDefGetter interface {
//	GlobalFlagDefs() []FlagDef
//}

type CmdRunner struct {
	args []string
}

func NewCmdRunner(args []string) *CmdRunner {
	return &CmdRunner{
		args: args,
	}
}

func (cr CmdRunner) ParseCmd(args []string) (cmd Command, err error) {
	var path string

	if len(args) == 0 {
		err = fmt.Errorf("no command specified")
		goto end
	}
	cr.args = args

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
		)
		goto end
	}

	cmd, path = GetDefaultCommand(path, args)
	if cmd == nil {
		err = NewErr(
			ErrCommandNotFound,
		)
		goto end
	}

	args, err = cmd.ParseFlagSets(args)
	if err != nil {
		err = NewErr(ErrFlagsParsingFailed)
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
			"command", strings.Join(cr.args, " "),
		)
	}
	return cmd, err
}

func (cr CmdRunner) RunCmd(ctx context.Context, cmd Command, config Config) (err error) {
	var handler CommandHandler
	var ok bool

	// Command resolution should ensure we only get CommandHandler implementations
	handler, ok = cmd.(CommandHandler)
	if !ok {
		err = fmt.Errorf("command '%s' does not implement handler logic", cmd.Name())
		goto end
	}

	err = handler.Handle(ctx, config, cr.args)

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

	// If no match found, return empty path with original args
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
func ShowCmdHelp(cmdName string) (err error) {
	var cmd Command
	var subCmds []Command

	cmd = GetExactCommand(cmdName)
	if cmd == nil {
		err = fmt.Errorf("unknown command: %s", cmdName)
		goto end
	}

	Printf("Usage: %s\n\n%s\n", cmd.Usage(), cmd.Description())

	subCmds = GetSubCmds(cmdName)
	if len(subCmds) > 0 {
		Printf("\nSubcommands:\n")
		for _, subCmd := range subCmds {
			Printf("    %-12s %s\n", subCmd.Name(), subCmd.Description())
		}
	}

end:
	return err
}
