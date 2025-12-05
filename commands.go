package cliutil

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Final resolved command maps (built during BuildCommandTree)
var commands = make([]Command, 0)
var commandsTypeMap = make(map[reflect.Type]Command)
var commandsPathMap = make(map[string]Command)
var flagCommandMap = make(map[string]Command)

// Command interface for basic command metadata and delegation
type Command interface {
	Name() string
	FullNames() []string
	Usage() string
	Description() string
	AddSubCommand(Command)
	DelegateTo() Command
	AddParent(reflect.Type)
	ParentTypes() []reflect.Type
	FlagSets() []*FlagSet
	ParseFlagSets([]string) ([]string, error)
	AssignArgs([]string) error
	Examples() []Example
	NoExamples() bool
	AutoExamples() bool
	ArgDefs() []*ArgDef
	Order() int
	SetCommandRunnerArgs(CmdRunnerArgs)
	FlagName() string
	IsHidden() bool
}

// CommandHandler interface for commands that actually execute logic
type CommandHandler interface {
	Command
	Handle() error
}

func Initialize(w Writer) (err error) {
	SetWriter(w)

	err = ValidateCommands()
	if err != nil {
		goto end
	}

	err = BuildCommandTree()
	if err != nil {
		goto end
	}

end:
	return err
}

func RegisteredCommands() (cmds []Command) {
	return commands
}

// RegisterCommand registers a command with optional parent type declarations
// First argument is the actual command, remaining arguments are parent type prototypes
// Example: RegisterCommand(&JobRunCmd{...}, &JobCmd{})
func RegisterCommand(cmd Command, parents ...Command) (err error) {
	var errs []error
	var parent Command
	var flagName string
	var globalFS *FlagSet
	var fd FlagDef

	for _, parent = range parents {
		cmd.AddParent(reflect.TypeOf(parent).Elem())
	}
	commands = append(commands, cmd)
	commandsTypeMap[reflect.TypeOf(cmd).Elem()] = cmd

	// Auto-register flag commands as global GlobalOptions
	flagName = cmd.FlagName()
	if flagName == "" {
		goto end
	}

	// Validate: Check for conflict with existing global flags
	globalFS = GetGlobalFlagSet()
	if globalFS != nil {
		for _, fd = range globalFS.FlagDefs {
			if fd.Name == flagName {
				errs = append(errs, fmt.Errorf("FlagName '%s' conflicts with existing global flag '%s'",
					flagName, fd.Name))
			}
		}
	}

	// TODO: Add more validations here in Part 8

	// Auto-register as global CLIOption so it appears in help
	err = AddCLIOption(FlagDef{
		Name:  flagName,
		Usage: fmt.Sprintf("Run %s command", cmd.Name()),
		Bool:  new(bool),
	})
	if err != nil {
		errs = append(errs, err)
	}

	err = CombineErrs(errs)
	if err != nil {
		err = WithErr(err, ErrCommandRegistrationFailed, "command_name", cmd.Name())
		goto end
	}

end:
	return err
}

var ErrCommandRegistrationFailed = errors.New("command registration failed")

// BuildCommandTree builds the command hierarchy from registrations
// This should be called by gmover.Initialize() after all init() functions complete
func BuildCommandTree() (err error) {
	//var topLevelCmds []Command
	var parentCmd Command
	var exists bool
	var cmd Command
	var flagName string

	// Second pass: build parent-child relationships
	for _, cmd = range commands {
		pts := cmd.ParentTypes()
		if len(pts) == 0 {
			// Top-level command
			//topLevelCmds = append(topLevelCmds, cmd.cmd)
			commandsPathMap[cmd.Name()] = cmd
			continue
		}
		// Child command - add to all parents
		for _, parentType := range pts {
			parentCmd, exists = commandsTypeMap[parentType]
			if !exists {
				err = fmt.Errorf("parent command type %s not found for command %s",
					parentType.Name(), cmd.Name())
				goto end
			}

			// Add child to parent's SubCommands
			parentCmd.AddSubCommand(cmd)

			// Add to commands map with parent path prefix
			for _, fn := range cmd.FullNames() {
				commandsPathMap[fn] = cmd
			}
		}
	}

	// Build flag command map
	for _, cmd = range commands {
		flagName = cmd.FlagName()
		if flagName != "" {
			flagCommandMap[flagName] = cmd
		}
	}

end:
	return err
}

type NULL = struct{}

func ValidateCommands() (err error) {
	var errs []error
	var ok bool
	var cmd Command
	var fs *FlagSet
	var fd FlagDef

	flagSets := make(map[*FlagSet]struct{})

	// 1. Existing: Check for duplicate FlagDefs within FlagSets
	for _, cmd = range commands {
		for _, fs = range cmd.FlagSets() {
			fdNames := make(map[string]struct{})
			_, ok = flagSets[fs]
			if ok {
				// We've already processed it, don't need to process again
				continue
			}
			flagSets[fs] = NULL{}
			for _, fd = range fs.FlagDefs {
				_, ok = fdNames[fd.Name]
				if ok {
					errs = append(errs, fmt.Errorf("duplicate FlagDef '%s' in FlagSet '%s'", fd.Name, fs.Name))
					continue
				}
				fdNames[fd.Name] = NULL{}
			}
		}
	}

	// 2. New: Validate single-dash flags are only one character
	for _, cmd = range commands {
		for _, fs = range cmd.FlagSets() {
			for _, fd = range fs.FlagDefs {
				if fd.Shortcut != 0 && fd.Shortcut > 127 {
					errs = append(errs, fmt.Errorf("command '%s': flag '%s' shortcut must be a single ASCII character", cmd.Name(), fd.Name))
				}
			}
		}
	}

	// 3. New: Validate subcommands cannot have FlagName
	for _, cmd = range commands {
		if len(cmd.ParentTypes()) > 0 && cmd.FlagName() != "" {
			errs = append(errs, fmt.Errorf("command '%s': subcommands cannot have FlagName (only top-level commands can use flag routing)", cmd.Name()))
		}
	}

	return errors.Join(errs...)
}

// GetExactCommand retrieves a command at any depth using dot notation
func GetExactCommand(path string) Command {
	return commandsPathMap[path]
}

// GetDefaultCommand retrieves a command or its default at any depth using dot notation
func GetDefaultCommand(path string, args []string) (cmd Command, _ string) {
	var defaultCmd Command
	var delegateType reflect.Type
	var exists bool

	cmd = GetExactCommand(path)
	if cmd == nil {
		goto end
	}

	if cmd.DelegateTo() == nil {
		goto end
	}

	// CLAUDE: Why is this important?
	if len(args) == 0 {
		goto end
	}

	// Check if the first arg is not a flag (would be handled by the default subcommand)
	// CLAUDE: Why is this important?
	if strings.HasPrefix(args[0], "-") {
		goto end
	}

	if cmd.DelegateTo() == nil {
		goto end
	}

	// Delegate to a default subcommand
	// Look up delegate by type
	delegateType = reflect.TypeOf(cmd.DelegateTo()).Elem()
	defaultCmd, exists = commandsTypeMap[delegateType]
	if exists {
		cmd = defaultCmd
		for _, p := range cmd.FullNames() {
			if !strings.HasPrefix(p, path) {
				continue
			}
			path = p
		}
		// TOOD: Should we add any error messages to use saying if we did not find a match?
	}

end:
	return cmd, path
}

// GetTopLevelCmds returns all top-level commands sorted by name
func GetTopLevelCmds() []Command {
	var topCmds []Command
	for name, cmd := range commandsPathMap {
		if !strings.Contains(name, ".") {
			topCmds = append(topCmds, cmd)
		}
	}
	return topCmds
}

// GetSubCmds returns all subcommands for a given path
func GetSubCmds(path string) []Command {
	var subCmds []Command
	prefix := path + "."
	for name, cmd := range commandsPathMap {
		if strings.HasPrefix(name, prefix) {
			// Only include direct children, not grandchildren
			remaining := strings.TrimPrefix(name, prefix)
			if !strings.Contains(remaining, ".") {
				subCmds = append(subCmds, cmd)
			}
		}
	}
	return subCmds
}

// ValidateCmds ensures all registered commands have handlers
func ValidateCmds() (err error) {
	return validateCmdTree(commandsPathMap, "")
}

// validateCmdTree recursively validates the command tree
func validateCmdTree(cmdMap map[string]Command, path string) (err error) {
	noop(path)
	for name, cmd := range cmdMap {
		if cmd == nil {
			err = fmt.Errorf("command %s has nil handler", name)
			goto end
		}
	}
end:
	return err
}
