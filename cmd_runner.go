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
		err = fmt.Errorf("unknown command: %s\nRun 'xmluicli help' for usage", cr.args[0])
		goto end
	}

	cmd, path = GetDefaultCommand(path, args)
	if cmd == nil {
		err = fmt.Errorf("command not found: %s", path)
		goto end
	}

	args, err = cmd.ParseFlagSets(args)
	if err != nil {
		goto end
	}

	err = cmd.AssignArgs(args)
	if err != nil {
		goto end
	}

end:
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
func ShowMainHelp() (err error) {
	Printf(`GMover - Move emails between Gmail accounts and labels

USAGE:
    gmover <command> [subcommand] [options]

COMMANDS:
`)

	// Show all top-level commands
	topCmds := GetTopLevelCmds()
	for _, cmd := range topCmds {
		subCmds := GetSubCmds(cmd.Name())
		subCmdText := ""
		if len(subCmds) > 0 {
			subCmdText = fmt.Sprintf(" [%s]", subCmds[0].Name()) // Show first subcommand as example
		}
		Printf("    %-20s %s\n", cmd.Name()+subCmdText, cmd.Description())
	}

	Printf(`
EXAMPLES:
    # Show help for a specific command
    gmover help list
    gmover help move

    # List available labels
    gmover list --src=user@example.com

    # Move emails  
    gmover move --src=user@example.com --dst=archive@example.com --src-label="INBOX" --dst-label="archived"

    # Job operations
    gmover job define daily-archive.json --src=user@example.com --dst=archive@example.com
    gmover job run daily-archive.json --auto-confirm

For more information, visit: https://github.com/mikeschinkel/gmover
`)
	return err
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
