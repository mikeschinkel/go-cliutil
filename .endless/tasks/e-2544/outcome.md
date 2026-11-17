# E-2544 synthesis: what a go-cliutil handler receives

Mike settled the design below in this brainstorm (2026-11-16/17), building on his
words in gomion's E-2534 session and accepted decision ED-1647 (a command takes at
most one positional argument, scalar or variadic). Backward compatibility is not a
concern: go-cliutil is pre-1.0. The goal Mike set: make go-cliutil broadly
applicable for anyone building CLI tools in Go, while keeping its typed,
plug-in-by-file `RegisterCommand()` model and moving app bootstrap boilerplate into
go-cliutil.

## 1. Handler signature and input

`Handle()` changes once to take a single struct, so later additions never break the
signature. `HandleArgs` carries everything a run needs injected, so tests can supply
it and one command can call another.

```go
type CommandHandler interface {
	Command
	Handle(HandleArgs) error
}

type HandleArgs struct {
	Context     context.Context
	IO          dt.StdIO
	Writer      Writer
	Logger      *slog.Logger
	Config      Config
	Options     Options
	Runner      *CmdRunner // lets one command run another
	CommandPath []string   // ["repo","setup"]
	CommandLine []string   // ["repo","setup","--check","."]  (program name excluded)
	Parameters  []string   // ["--check","."]                 (everything after the command words)
	Flags       Flags      // every flag that applies to this run, global and command
	Operands    Operands   // ["."]
}
```

Naming is the POSIX set (CommandLine / Parameters / Operands), chosen over
Argv/Params/Positionals, RawArgs/Parameters/Positionals, and a bare `Args`. Every
field gets a doc comment naming exactly which words it holds; the gomion bug that
started this came from an undocumented `Args` field.

`Flags` and `Operands` are types, not slices:

```go
type Flag struct {
	Name   string
	Value  any        // parsed value, never absent: default or config fills it
	Source FlagSource // DefaultSource, ConfigSource, CommandLineSource
	Global bool
	Words  []string   // as typed: ["--format","json"]; nil when not typed
	Def    *FlagDef
}

func (f Flags) Get(name string) (Flag, bool)
func (f Flags) All() []Flag    // definition order
func (f Flags) Passed() []Flag // typed by the user, in typed order

type Operand struct {
	Value    string
	Position int     // index within Parameters
	Def      *ArgDef // nil under Passthrough
}
type Operands []Operand
func (o Operands) Strings() []string
```

Typed access uses a Go 1.27 generic method (verified compiling on go1.27.3), with no
`ok` result. An undefined name or a wrong type is a programming error, so it panics
with the flag name, command path and types:

```go
input := args.Flag[string]("input")
```

The existing `Flags` marker interface and `ArgsParser` go away (see section 5), which
frees the name.

## 2. Help

Help declares one variadic operand and reads it; it no longer needs the raw command
line, and `RunCmd`'s special case that strips `"help"` goes away.

```go
// gomion help repo setup  →  c.topic == ["repo","setup"], args.CommandPath == ["help"]
ArgDefs: []*cliutil.ArgDef{{Name: "command", Strings: &c.topic}}
```

## 3. Positionals

- **Variadic (plan Q2: A).** `ArgDef` gains a `Strings *[]string` target. Under
  ED-1647 a command has at most one ArgDef, so registration rejects a second one.
  Optional `MinCount` / `MaxCount` limit the variadic list; `Required` means
  `MinCount: 1`; registration rejects count fields on a single-value ArgDef. Mike:
  nice to have, skip if too hard.
- **Extra operands (plan Q3: A).** An operand beyond what the command declares is an
  error shown with the command's usage. Never silently dropped.
- **Defaults (plan Q4: B).** Typed default fields per target type, chosen over
  `Default any` and over "the constructor's value is the default" because they are
  self-documenting: `DefaultString`, `DefaultStrings` on ArgDef; `DefaultString`,
  `DefaultBool`, `DefaultInt`, `DefaultInt64` on FlagDef. `Default any` is removed.
  go-cliutil writes the default into the target when the value is absent; help shows
  it.

## 4. Flag placement and parsing

- **Global flags** are accepted anywhere: before, between, or after the command
  words. Users rarely know which flags are global, so they must behave alike.
  (kubectl, cargo and aws do this; docker, terraform and click allow globals only
  before the command.)
- **Command flags** are accepted only after their command word, anywhere among the
  operands: `endless task show E-2544 --all-fields -p` and
  `endless task show --all-fields -p E-2544` are valid;
  `endless --all-fields task show E-2544` and `endless task --all-fields show E-2544`
  are errors.
- Registration rejects a command flag whose name matches a global flag.
- `--` ends flags; every word after it is an operand.
- A boolean flag never consumes the following word (today `--check .` loses `.`);
  `--flag=false` still works.
- **Passthrough** is an opt-in: `CmdArgs{Passthrough: true}` puts every word after
  the command words into `Parameters` unparsed, global flags included, so global
  flags for such a command must come before its command words.

## 5. Global flags: one level, on the Registry

Today there are three layers: go-cliutil's built-in global FlagSet (`gFlagSets`), a
second go-cliutil global FlagSet (`flagSet`, fed by both `AddCLIOption` and
`AddCLIFlag`), and each app's own global flags parsed by hand with Go's `flag`
package through `ArgsParser` (gomion's `gomcfg.Flags`). The third layer cannot honour
"global flags anywhere", because `flag` stops at the first non-flag.

They collapse into one set of global FlagDefs on the Registry. go-cliutil pre-loads
the flags almost every app wants; an app adds, replaces or removes by name with one
line each. Mike's three requirements: minimal app boilerplate, go-cliutil supplies
the common flags, apps override what they want changed.

```go
func NewRegistry() *Registry // starts with verbosity, quiet, timeout, dry-run, force
func (r *Registry) AddGlobalFlag(fd FlagDef) error
func (r *Registry) ReplaceGlobalFlag(fd FlagDef) error
func (r *Registry) RemoveGlobalFlag(name string) error

func init() {
	r := cliutil.DefaultRegistry()
	r.AddGlobalFlag(cliutil.FlagDef{Name: "input", Type: cliutil.StringFlag, Usage: "..."})
	r.ReplaceGlobalFlag(cliutil.FlagDef{Name: "timeout", Type: cliutil.IntFlag, DefaultInt: 30, Usage: "..."})
	r.RemoveGlobalFlag("dry-run")
}
```

Global flags have no targets (a shared target would leak between runs). `FlagDef`
gains `Type` for a target-less flag; registration checks the typed default field
matches it. Values reach code through `args.Flag[T]` and through `Options`, which the
app builds once per run:

```go
type OptionsFunc func(Flags, Config) (Options, error)
```

Removed: the `Flags` marker interface, `ArgsParser`, the parallel `GlobalFlags` /
`GlobalOptions` structs, `AddCLIFlag` vs `AddCLIOption`, and app-side code like
gomion's `ParseOptions` "validation dance".

## 6. Config

go-cliutil imports no config library, so it can be used with go-cfgstore, Viper or
anything else. The app supplies a loader; a Config that implements `FlagValuer`
supplies values for flags not typed. Precedence: command line, then config, then the
flag's default. Mike: the flag wins.

```go
type ConfigFunc func(ConfigArgs) (Config, error)
type ConfigArgs struct {
	AppInfo appinfo.AppInfo
	Flags   Flags // already parsed, so a --config flag can pick the file
	IO      dt.StdIO
}
type FlagValuer interface {
	FlagValue(name string) (value any, ok bool)
}
// Per run: parse flags → ConfigFunc(flags) → fill untyped flags from FlagValuer
//          → OptionsFunc(flags, config) → Handle
```

go-cfgstore supplies an adapter (it already imports go-cliutil, so the dependency
direction holds). A Viper user implements `FlagValue` with `v.Get` / `v.IsSet`.

## 7. Registry and command values (plan Q6)

Commands keep registering from their own file in `init()`, but register a
constructor, `CommandFunc` (not "factory"), instead of an instance. Each run builds
fresh command values, and targets are fields of the command struct, so no value
leaks from one run to the next (today `repo setup --check .` followed by
`repo setup .` in one process still sees `check == true`, because `FlagSet.Build`
resets only targets whose FlagDef has a Default).

```go
type CommandFunc func() Command
func RegisterCommand(fn CommandFunc, parents ...Command) // into DefaultRegistry()
func DefaultRegistry() *Registry
func NewRegistry() *Registry

type CmdRunnerArgs struct {
	Registry *Registry   // nil means DefaultRegistry()
	Config   ConfigFunc
	Options  OptionsFunc
	...
}

func init() { cliutil.RegisterCommand(NewRepoSetupCmd, &RepoCmd{}) }

func NewRepoSetupCmd() cliutil.Command {
	c := &RepoSetupCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:     "setup",
		ArgDefs:  []*cliutil.ArgDef{{Name: "dir", DefaultString: ".", String: &c.dir}},
		FlagSets: []*cliutil.FlagSet{{FlagDefs: []cliutil.FlagDef{{Name: "check", Bool: &c.check}}}},
	})
	return c
}
```

No `SetRegistry`: tests pass a registry through `CmdRunnerArgs` instead of swapping a
global, so they can run in parallel. App tests use the default registry their
`init()` filled; go-cliutil's own tests build a registry and register test commands.

## 8. Test helper (plan Q7: A)

An exported package `cliutiltest` in go-cliutil's root module runs a command line
in-process and returns output and exit code, so each consumer writes the table, not
the harness. An app binary never imports it, so it is never compiled into one. The
harness fails a test when a handler asks for an undefined flag or the wrong type.

## 9. Rollout (plan Q8: A)

One go-cliutil implementation task, then one task per consumer project, each blocked
by it. Endless does not allow a task to change more than one project.

## Follow-ups

Decisions: ED-1666 (handler input, section 1), ED-1667 (flag placement, 4), ED-1668
(typed defaults, 3), ED-1669 (global flags on the Registry, 5), ED-1670 (config, 6),
ED-1671 (CommandFunc registration, 7), ED-1672 (cliutiltest, 8).

Tasks:
- E-2648 (go-cliutil): implement this design, including the three defect fixes.
- E-2649 (go-cliutil brainstorm): the rest of app bootstrap in go-cliutil.
- E-2650 (go-cfgstore): ConfigFunc adapter, blocked by E-2648.
- E-2651 (gomion): move gomion to the new design, blocked by E-2648 and E-2650.

Other consumers (doctool, gmover, macmail-ext, homelab, go-stash, gru and other
go-cli tools, go-logutil, go-testutil; Endless after its Go port) get their own
tasks when Mike chooses which to file.
