package cliutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// FlagType represents the type of a command flag
type FlagType int

const (
	UnknownFlagType FlagType = iota
	StringFlag
	BoolFlag
	IntFlag
	Int64Flag
)

var _ Command = (*CmdBase)(nil)

// CmdBase provides common functionality for all commands
// It implements the cliutil.Cmd interface
type CmdBase struct {
	cliName      string
	name         string
	usage        string
	description  string
	flagsDefs    []FlagDef  // Legacy flag definitions (will be deprecated)
	flagSets     []*FlagSet // New FlagSet-based approach
	argDefs      []*ArgDef  // Positional argument definitions
	delegateTo   Command
	parentTypes  []reflect.Type
	subCommands  []Command
	examples     []Example // Custom examples
	noExamples   bool      // Do not display any examples
	autoExamples bool      // Display auto-generated examples even if custom are provided
	order        int       // Display order in help (0=last, 1+=ordered)
	flagName     string    // Flag name that triggers this command (e.g., "setup" for --setup)
	hide         bool      // Hide from help output
	CmdRunnerArgs
}

type CmdArgs struct {
	Name         string
	Usage        string
	Description  string
	DelegateTo   Command
	FlagDefs     []FlagDef  // Legacy flag definitions (will be deprecated)
	FlagSets     []*FlagSet // New FlagSet-based approach
	ArgDefs      []*ArgDef  // Positional argument definitions
	Examples     []Example  // Custom examples
	NoExamples   bool       // Do not display any examples
	AutoExamples bool       // Display auto-generated examples even if custom are provided
	Order        int        // Display order in help (0=last, 1+=ordered)
	FlagName     string     // Flag name that triggers this command (e.g., "setup" for --setup)
	Hide         bool       // Hide from help output
}

// NewCmdBase creates a new command base
func NewCmdBase(args CmdArgs) *CmdBase {
	return &CmdBase{
		cliName:      filepath.Base(os.Args[0]),
		name:         args.Name,
		usage:        args.Usage,
		description:  args.Description,
		flagsDefs:    args.FlagDefs,
		flagSets:     args.FlagSets, // Static FlagSets (legacy)
		argDefs:      args.ArgDefs,  // Positional argument definitions
		delegateTo:   args.DelegateTo,
		examples:     args.Examples,
		noExamples:   args.NoExamples,
		autoExamples: args.AutoExamples,
		order:        args.Order,
		flagName:     args.FlagName,
		hide:         args.Hide,
		parentTypes:  make([]reflect.Type, 0),
		subCommands:  make([]Command, 0),
	}
}

// Name returns the command name
func (c *CmdBase) Name() string {
	return c.name
}

// CLIName returns the name of the CLI app
func (c *CmdBase) CLIName() string {
	return c.cliName
}

// FullNames returns the command names prefixed with any parent names
func (c *CmdBase) FullNames() (names []string) {
	names = make([]string, len(c.parentTypes))
	for i, t := range c.parentTypes {
		parent := commandsTypeMap[t]
		for _, pn := range parent.FullNames() {
			names[i] = fmt.Sprintf("%s.%s", pn, c.name)
		}
	}
	if len(names) == 0 {
		names = []string{c.name}
	}
	return names
}

// Usage returns the command usage string
// Flags are now rendered by templates via FlagSets, not in the usage string
func (c *CmdBase) Usage() string {
	return c.usage
}

// Description returns the command description
// Flags are now rendered by templates via FlagSets, not in the description
func (c *CmdBase) Description() string {
	return c.description
}

// AddSubCommand returns the subcommands map
func (c *CmdBase) AddSubCommand(cmd Command) {
	c.subCommands = append(c.subCommands, cmd)
}

// DelegateTo returns the command to delegate to, if any
func (c *CmdBase) DelegateTo() Command {
	return c.delegateTo
}

// SetDelegateTo sets the command to delegate to
func (c *CmdBase) SetDelegateTo(cmd Command) {
	c.delegateTo = cmd
}

// ParseFlagSets parses flags using the new FlagSet-based approach
func (c *CmdBase) ParseFlagSets(args []string) (remainingArgs []string, err error) {
	var errs []error
	nonFSArgs := args

	// Parse each FlagSet in sequence
	for _, flagSet := range c.flagSets {
		nonFSArgs, err = flagSet.Parse(nonFSArgs)
		errs = append(errs, err)
	}

	err = errors.Join(errs...)
	return nonFSArgs, err
}

//// validateFlags ensures all required flags are provided
//func (c *CmdBase) validateFlags(values map[string]any) (err error) {
//	var errs []error
//	for _, fd := range c.flagsDefs {
//		if !fd.Required {
//			continue
//		}
//		value := values[fd.Name]
//		switch fd.Type() {
//		case BoolFlag:
//			// Nothing to do
//		case StringFlag:
//			if value.(string) == "" {
//				errs = AppendErr(errs, fmt.Errorf("%s is required (use --%s flag)", fd.Usage, fd.Name))
//				goto end
//			}
//		case IntFlag:
//			if value.(int) == 0 && fd.Default == nil {
//				errs = AppendErr(errs, fmt.Errorf("%s is required (use --%s flag)", fd.Usage, fd.Name))
//				goto end
//			}
//		case Int64Flag:
//			if value.(int64) == 0 && fd.Default == nil {
//				errs = AppendErr(errs, fmt.Errorf("%s is required (use --%s flag)", fd.Usage, fd.Name))
//				goto end
//			}
//		case UnknownFlagType:
//			errs = AppendErr(errs, fmt.Errorf("flag type not set for '%s'", fd.Name))
//		}
//	}
//
//end:
//	return CombineErrs(errs)
//}

// AssignArgs assigns positional arguments to their defined config fields
func (c *CmdBase) AssignArgs(args []string) (err error) {
	var errs []error

	// Check if we have enough arguments for required ones
	requiredCount := 0
	for _, argDef := range c.argDefs {
		if argDef.Required {
			requiredCount++
		}
	}

	if len(args) < requiredCount {
		err = fmt.Errorf("expected at least %d arguments, got %d", requiredCount, len(args))
		goto end
	}

	// Assign available arguments
	for i, argDef := range c.argDefs {
		if i >= len(args) {
			if argDef.Required {
				errs = append(errs, fmt.Errorf("required argument '%s' missing", argDef.Name))
			}
			continue
		}

		if argDef.String != nil {
			*argDef.String = args[i]
		}
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
	}

end:
	return err
}

func (c *CmdBase) Examples() []Example {
	return c.examples
}

func (c *CmdBase) NoExamples() bool {
	return c.noExamples
}

func (c *CmdBase) AutoExamples() bool {
	return c.autoExamples
}

func (c *CmdBase) ArgDefs() []*ArgDef {
	return c.argDefs
}

func (c *CmdBase) Order() int {
	return c.order
}

func (c *CmdBase) FlagSets() []*FlagSet {
	return c.flagSets
}

func (c *CmdBase) ParentTypes() []reflect.Type {
	return c.parentTypes
}

func (c *CmdBase) AddParent(r reflect.Type) {
	c.parentTypes = append(c.parentTypes, r)
}

//	func (c *CmdBase) Logger() *slog.Logger {
//		return c.logger
//	}
//
//	func (c *CmdBase) Writer() Writer {
//		return c.writer
//	}
func (c *CmdBase) SetCommandRunnerArgs(args CmdRunnerArgs) {
	c.CmdRunnerArgs = args
}

func (c *CmdBase) FlagName() string {
	return c.flagName
}

func (c *CmdBase) IsHidden() bool {
	return c.hide
}
