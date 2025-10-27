package cliutil

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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
	name        string
	usage       string
	description string
	flagsDefs   []FlagDef  // Legacy flag definitions (will be deprecated)
	flagSets    []*FlagSet // New FlagSet-based approach
	argDefs     []*ArgDef  // Positional argument definitions
	delegateTo  Command
	parentTypes []reflect.Type
	subCommands []Command
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

type CmdArgs struct {
	Name        string
	Usage       string
	Description string
	DelegateTo  Command
	FlagDefs    []FlagDef  // Legacy flag definitions (will be deprecated)
	FlagSets    []*FlagSet // New FlagSet-based approach
	ArgDefs     []*ArgDef  // Positional argument definitions
}

// NewCmdBase creates a new command base
func NewCmdBase(args CmdArgs) *CmdBase {
	return &CmdBase{
		name:        args.Name,
		usage:       args.Usage,
		description: args.Description,
		flagsDefs:   args.FlagDefs,
		flagSets:    args.FlagSets, // Static FlagSets (legacy)
		argDefs:     args.ArgDefs,  // Positional argument definitions
		delegateTo:  args.DelegateTo,
		parentTypes: make([]reflect.Type, 0),
		subCommands: make([]Command, 0),
	}
}

// Name returns the command name
func (c *CmdBase) Name() string {
	return c.name
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

// Usage returns the command usage string with flags
func (c *CmdBase) Usage() string {
	if len(c.flagsDefs) == 0 {
		return c.usage
	}

	var flagUsage []string
	for _, flagDef := range c.flagsDefs {
		var flagStr string
		if flagDef.Required {
			flagStr = fmt.Sprintf("--%s=VALUE", flagDef.Name)
		} else {
			flagStr = fmt.Sprintf("[--%s=VALUE]", flagDef.Name)
		}
		flagUsage = append(flagUsage, flagStr)
	}

	return fmt.Sprintf("%s %s", c.usage, strings.Join(flagUsage, " "))
}

// Description returns the command description with flag details
func (c *CmdBase) Description() string {
	var desc strings.Builder

	if len(c.flagsDefs) == 0 {
		return c.description
	}

	desc.WriteString(c.description)
	desc.WriteString("\n\nFlags:")

	for _, flagDef := range c.flagsDefs {
		required := ""
		if flagDef.Required {
			required = " (required)"
		}
		desc.WriteString(fmt.Sprintf("\n    --%s: %s%s", flagDef.Name, flagDef.Usage, required))
	}

	return desc.String()
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

// validateFlags ensures all required flags are provided
func (c *CmdBase) validateFlags(values map[string]any) (err error) {
	var errs []error
	for _, fd := range c.flagsDefs {
		if !fd.Required {
			continue
		}
		value := values[fd.Name]
		switch fd.Type() {
		case BoolFlag:
			// Nothing to do
		case StringFlag:
			if value.(string) == "" {
				err = fmt.Errorf("%s is required (use --%s flag)", fd.Usage, fd.Name)
				goto end
			}
		case IntFlag:
			if value.(int) == 0 && fd.Default == nil {
				err = fmt.Errorf("%s is required (use --%s flag)", fd.Usage, fd.Name)
				goto end
			}
		case Int64Flag:
			if value.(int64) == 0 && fd.Default == nil {
				err = fmt.Errorf("%s is required (use --%s flag)", fd.Usage, fd.Name)
				goto end
			}
		case UnknownFlagType:
			errs = append(errs, fmt.Errorf("flag type not set for '%s'", fd.Name))
		}
	}

end:
	return errors.Join(errs...)
}

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
