package cliutil

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// ErrCommandRegistrationFailed means a registered command's definition is invalid.
var ErrCommandRegistrationFailed = errors.New("command registration failed")

// Command interface for basic command metadata and delegation
type Command interface {
	CLIName() string
	Name() string
	FullNames() []string
	SetFullNames([]string)
	Usage() string
	Description() string
	AddSubCommand(Command)
	DelegateTo() Command
	AddParent(reflect.Type)
	ParentTypes() []reflect.Type
	FlagSets() []*FlagSet
	Examples() []Example
	NoExamples() bool
	AutoExamples() bool
	ArgDefs() []*ArgDef
	Order() int
	FlagName() string
	IsHidden() bool
	Passthrough() bool
}

// CommandHandler interface for commands that actually execute logic
type CommandHandler interface {
	Command
	Handle(HandleArgs) error
}

// Initialize records the user's home directory so Writer.Errorf can shorten
// paths to ~. Call it once per process, before any run.
func Initialize() (err error) {
	err = dt.EnsureUserHomeDir()
	if err != nil {
		goto end
	}
	setHomeDir(dt.GetUserHomeDir())

end:
	return err
}

// NULL is the empty value of a set.
type NULL = struct{}

// CommandSet holds the command values built for one run.
type CommandSet struct {
	commands    []Command
	typeMap     map[reflect.Type]Command
	pathMap     map[string]Command
	flagCommand map[string]Command
	globalDefs  []FlagDef
}

// commandType identifies a command by its struct type.
func commandType(cmd Command) reflect.Type {
	t := reflect.TypeOf(cmd)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// newCommandSet calls every registered CommandFunc, builds the command tree from
// the values they return, and validates the result.
func newCommandSet(registrations []registration, globalDefs []FlagDef) (set *CommandSet, err error) {
	var errs []error

	set = &CommandSet{
		typeMap:     make(map[reflect.Type]Command),
		pathMap:     make(map[string]Command),
		flagCommand: make(map[string]Command),
		globalDefs:  globalDefs,
	}

	for _, reg := range registrations {
		var cmd Command
		if reg.fn != nil {
			cmd = reg.fn()
		}
		if cmd == nil || reflect.ValueOf(cmd).Kind() == reflect.Pointer && reflect.ValueOf(cmd).IsNil() {
			errs = append(errs, NewErr(ErrInvalidRegistration, errors.New("a CommandFunc returned no command")))
			continue
		}
		t := commandType(cmd)
		if _, exists := set.typeMap[t]; exists {
			errs = append(errs, NewErr(ErrInvalidRegistration, fmt.Errorf("command type %s is registered twice", t)))
			continue
		}
		for _, parent := range reg.parents {
			cmd.AddParent(parent)
		}
		set.typeMap[t] = cmd
		set.commands = append(set.commands, cmd)
	}

	errs = AppendErr(errs, set.buildTree())
	errs = AppendErr(errs, set.validate())
	err = CombineErrs(errs)
	if err != nil {
		err = WithErr(err, ErrCommandRegistrationFailed)
	}
	return set, err
}

// buildTree resolves each command's paths and links parents to children.
func (s *CommandSet) buildTree() error {
	var errs []error
	resolved := make(map[Command][]string)

	var resolve func(cmd Command, visiting []Command) ([]string, error)
	resolve = func(cmd Command, visiting []Command) (names []string, err error) {
		if existing, ok := resolved[cmd]; ok {
			return existing, nil
		}
		if slices.Contains(visiting, cmd) {
			return nil, fmt.Errorf("command '%s' is its own ancestor", cmd.Name())
		}
		for _, parentType := range cmd.ParentTypes() {
			parent, ok := s.typeMap[parentType]
			if !ok {
				return nil, fmt.Errorf("parent command type %s not found for command %s", parentType.Name(), cmd.Name())
			}
			parentNames, err := resolve(parent, append(visiting, cmd))
			if err != nil {
				return nil, err
			}
			for _, parentName := range parentNames {
				names = append(names, parentName+"."+cmd.Name())
			}
		}
		if len(cmd.ParentTypes()) == 0 {
			names = []string{cmd.Name()}
		}
		resolved[cmd] = names
		return names, nil
	}

	for _, cmd := range s.commands {
		names, err := resolve(cmd, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		cmd.SetFullNames(names)
		for _, name := range names {
			if _, exists := s.pathMap[name]; exists {
				errs = append(errs, fmt.Errorf("two commands share the path '%s'", name))
				continue
			}
			s.pathMap[name] = cmd
		}
		for _, parentType := range cmd.ParentTypes() {
			if parent, ok := s.typeMap[parentType]; ok {
				parent.AddSubCommand(cmd)
			}
		}
	}
	return CombineErrs(errs)
}

// validate reports every definition error across the registered commands.
func (s *CommandSet) validate() error {
	var errs []error

	globalNames := make(map[string]NULL)
	globalShortcuts := make(map[byte]NULL)
	for i := range s.globalDefs {
		fd := &s.globalDefs[i]
		errs = AppendErr(errs, fd.validateDef())
		if _, dup := globalNames[fd.Name]; dup {
			errs = append(errs, NewErr(dt.ErrInvalidDuplicateFlag, "flag_name", fd.Name,
				fmt.Errorf("duplicate global flag '%s'", fd.Name)))
		}
		globalNames[fd.Name] = NULL{}
		if fd.Shortcut == 0 {
			continue
		}
		if _, dup := globalShortcuts[fd.Shortcut]; dup {
			errs = append(errs, NewErr(dt.ErrInvalidDuplicateFlag, "flag_name", fd.Name,
				fmt.Errorf("duplicate global flag shortcut '%c'", fd.Shortcut)))
		}
		globalShortcuts[fd.Shortcut] = NULL{}
	}

	for _, cmd := range s.commands {
		errs = AppendErr(errs, s.validateFlagRoute(cmd, globalNames))
		errs = AppendErr(errs, validateCommandFlags(cmd, globalNames, globalShortcuts))
		errs = AppendErr(errs, validateCommandArgs(cmd))
	}
	return CombineErrs(errs)
}

// validateFlagRoute checks a command's FlagName and records its route.
func (s *CommandSet) validateFlagRoute(cmd Command, globalNames map[string]NULL) (err error) {
	flagName := cmd.FlagName()
	if flagName == "" {
		goto end
	}
	if len(cmd.ParentTypes()) > 0 {
		err = fmt.Errorf("command '%s': subcommands cannot have FlagName (only top-level commands can use flag routing)", cmd.Name())
		goto end
	}
	if _, conflict := globalNames[flagName]; conflict {
		err = NewErr(ErrCommandRegistrationFailed, dt.ErrInvalidDuplicateFlag,
			"command_name", cmd.Name(), "flag_name", flagName,
			fmt.Errorf("command '%s': FlagName '%s' conflicts with a global flag", cmd.Name(), flagName))
		goto end
	}
	if _, conflict := s.flagCommand[flagName]; conflict {
		err = NewErr(ErrCommandRegistrationFailed, dt.ErrInvalidDuplicateFlag,
			"command_name", cmd.Name(), "flag_name", flagName,
			fmt.Errorf("command '%s': FlagName '%s' is used by another command", cmd.Name(), flagName))
		goto end
	}
	s.flagCommand[flagName] = cmd

end:
	return err
}

// validateCommandFlags checks the flags a command declares.
func validateCommandFlags(cmd Command, globalNames map[string]NULL, globalShortcuts map[byte]NULL) error {
	var errs []error
	names := make(map[string]NULL)
	shortcuts := make(map[byte]NULL)

	if cmd.Passthrough() && len(cmd.FlagSets()) > 0 {
		errs = append(errs, fmt.Errorf("command '%s': a Passthrough command cannot declare flags", cmd.Name()))
	}

	for _, fs := range cmd.FlagSets() {
		for i := range fs.FlagDefs {
			fd := &fs.FlagDefs[i]
			err := fd.validateDef()
			if err != nil {
				errs = append(errs, WithErr(err, "command_name", cmd.Name()))
			}
			if _, dup := names[fd.Name]; dup {
				errs = append(errs, fmt.Errorf("command '%s': duplicate flag '%s'", cmd.Name(), fd.Name))
			}
			names[fd.Name] = NULL{}
			if _, conflict := globalNames[fd.Name]; conflict {
				errs = append(errs, fmt.Errorf("command '%s': flag '%s' is named like a global flag", cmd.Name(), fd.Name))
			}
			if fd.Shortcut == 0 {
				continue
			}
			if _, dup := shortcuts[fd.Shortcut]; dup {
				errs = append(errs, fmt.Errorf("command '%s': duplicate flag shortcut '%c'", cmd.Name(), fd.Shortcut))
			}
			shortcuts[fd.Shortcut] = NULL{}
			if _, conflict := globalShortcuts[fd.Shortcut]; conflict {
				errs = append(errs, fmt.Errorf("command '%s': flag '%s' shortcut '%c' is used by a global flag", cmd.Name(), fd.Name, fd.Shortcut))
			}
		}
	}
	return CombineErrs(errs)
}

// validateCommandArgs checks the positional argument a command declares.
func validateCommandArgs(cmd Command) error {
	var errs []error
	argDefs := cmd.ArgDefs()

	if len(argDefs) > 1 {
		errs = append(errs, fmt.Errorf("command '%s': a command declares at most one ArgDef", cmd.Name()))
	}
	if cmd.Passthrough() && len(argDefs) > 0 {
		errs = append(errs, fmt.Errorf("command '%s': a Passthrough command cannot declare an ArgDef", cmd.Name()))
	}
	for _, ad := range argDefs {
		if (ad.String == nil) == (ad.Strings == nil) {
			errs = append(errs, fmt.Errorf("command '%s': ArgDef '%s' must set exactly one of String or Strings", cmd.Name(), ad.Name))
		}
		if !ad.isVariadic() && (ad.MinCount != 0 || ad.MaxCount != 0) {
			errs = append(errs, fmt.Errorf("command '%s': ArgDef '%s' sets MinCount or MaxCount without a Strings target", cmd.Name(), ad.Name))
		}
		if ad.MinCount < 0 || ad.MaxCount < 0 || (ad.MaxCount != 0 && ad.MaxCount < ad.MinCount) {
			errs = append(errs, fmt.Errorf("command '%s': ArgDef '%s' has an invalid MinCount or MaxCount", cmd.Name(), ad.Name))
		}
		if ad.isVariadic() && ad.DefaultString != "" || !ad.isVariadic() && ad.DefaultStrings != nil {
			errs = append(errs, fmt.Errorf("command '%s': ArgDef '%s' default does not match its target", cmd.Name(), ad.Name))
		}
	}
	return CombineErrs(errs)
}

// All returns every command in registration order.
func (s *CommandSet) All() []Command {
	return slices.Clone(s.commands)
}

// GetExactCommand retrieves a command at any depth using dot notation
func (s *CommandSet) GetExactCommand(path string) Command {
	return s.pathMap[path]
}

// GetDefaultCommand retrieves the command at path, following DelegateTo to the
// command's default subcommand. It returns the resolved command and its path.
func (s *CommandSet) GetDefaultCommand(path string) (cmd Command, _ string) {
	cmd = s.GetExactCommand(path)
	seen := make(map[Command]NULL)
	for cmd != nil && cmd.DelegateTo() != nil {
		if _, loop := seen[cmd]; loop {
			break
		}
		seen[cmd] = NULL{}
		target, exists := s.typeMap[commandType(cmd.DelegateTo())]
		if !exists {
			break
		}
		cmd = target
		for _, name := range cmd.FullNames() {
			if strings.HasPrefix(name, path) {
				path = name
				break
			}
		}
	}
	return cmd, path
}

// GetTopLevelCmds returns all top-level commands in registration order.
func (s *CommandSet) GetTopLevelCmds() (cmds []Command) {
	for _, cmd := range s.commands {
		if len(cmd.ParentTypes()) == 0 {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

// GetSubCmds returns the direct subcommands of path in registration order.
func (s *CommandSet) GetSubCmds(path string) (cmds []Command) {
	prefix := path + "."
	for _, cmd := range s.commands {
		for _, name := range cmd.FullNames() {
			rest, found := strings.CutPrefix(name, prefix)
			if found && !strings.Contains(rest, ".") {
				cmds = append(cmds, cmd)
				break
			}
		}
	}
	return cmds
}

// flagRouted returns the command a --name flag routes to, if any.
func (s *CommandSet) flagRouted(name string) Command {
	return s.flagCommand[name]
}
