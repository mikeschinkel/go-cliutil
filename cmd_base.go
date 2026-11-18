package cliutil

import (
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
	StringsFlag
)

var _ Command = (*CmdBase)(nil)

// CmdBase provides common functionality for all commands
// It implements the cliutil.Cmd interface
type CmdBase struct {
	cliName      string
	name         string
	usage        string
	description  string
	flagSets     []*FlagSet // Flags the command accepts
	argDefs      []*ArgDef  // Positional argument definition; at most one
	delegateTo   Command
	parentTypes  []reflect.Type
	subCommands  []Command
	examples     []Example // Custom examples
	noExamples   bool      // Do not display any examples
	autoExamples bool      // Display auto-generated examples even if custom are provided
	order        int       // Display order in help (0=last, 1+=ordered)
	flagName     string    // Flag name that triggers this command (e.g., "setup" for --setup)
	hide         bool      // Hide from help output
	passthrough  bool      // Receive every word after the command words unparsed
	fullNames    []string  // Dot-separated paths, set when a run builds its commands
}

type CmdArgs struct {
	Name         string
	Usage        string
	Description  string
	DelegateTo   Command
	FlagSets     []*FlagSet // Flags the command accepts
	ArgDefs      []*ArgDef  // Positional argument definition; at most one
	Examples     []Example  // Custom examples
	NoExamples   bool       // Do not display any examples
	AutoExamples bool       // Display auto-generated examples even if custom are provided
	Order        int        // Display order in help (0=last, 1+=ordered)
	FlagName     string     // Flag name that triggers this command (e.g., "setup" for --setup)
	Hide         bool       // Hide from help output

	// Passthrough puts every word after the command words into
	// HandleArgs.Parameters unparsed, global flags included. Global flags for such
	// a command must come before its command words.
	Passthrough bool
}

// NewCmdBase creates a new command base
func NewCmdBase(args CmdArgs) *CmdBase {
	return &CmdBase{
		cliName:      filepath.Base(os.Args[0]),
		name:         args.Name,
		usage:        args.Usage,
		description:  args.Description,
		flagSets:     args.FlagSets,
		argDefs:      args.ArgDefs,
		delegateTo:   args.DelegateTo,
		examples:     args.Examples,
		noExamples:   args.NoExamples,
		autoExamples: args.AutoExamples,
		order:        args.Order,
		flagName:     args.FlagName,
		hide:         args.Hide,
		passthrough:  args.Passthrough,
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

// FullNames returns the command's dot-separated paths, one per parent path. A
// run sets them when it builds its commands; before that it returns the name.
func (c *CmdBase) FullNames() []string {
	if len(c.fullNames) == 0 {
		return []string{c.name}
	}
	return c.fullNames
}

// SetFullNames records the command's dot-separated paths for this run.
func (c *CmdBase) SetFullNames(names []string) {
	c.fullNames = names
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

func (c *CmdBase) FlagName() string {
	return c.flagName
}

func (c *CmdBase) IsHidden() bool {
	return c.hide
}

// Passthrough reports whether the command receives its words unparsed.
func (c *CmdBase) Passthrough() bool {
	return c.passthrough
}
