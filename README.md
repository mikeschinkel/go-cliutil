# CLI Framework for Go — Opinionated and Productive

A comprehensive CLI framework for Go applications that provides command registration, flag parsing, operand handling, layered configuration, and structured output management.

## What Makes cliutil Different

`cliutil` is **opinionated and architectural** - it enforces a specific project structure that scales from simple tools to complex multi-command applications:

- **Enforced architecture** - `cmd/myapp/main.go` → `<apppkg>.RunCLI()` → `CmdRunner.Run()` separates concerns
- **Auto-registration via `init()`** - Commands register themselves; blank import `_ "myapp/myappcmds"` loads all commands
- **A fresh command for every run** - Constructors build new values, so nothing leaks between runs and parallel tests stay safe
- **Everything arrives through `HandleArgs`** - Handlers receive their context, streams, config, options, flags and operands as one argument
- **Type-safe options and config** - Your options type embeds `*cliutil.GlobalOptions`, and `ConfigApplier` lets the compiler check every config mapping
- **Integrated logging** - `WriterLogger` combines `*slog.Logger` with verbosity-aware console output
- **Rich error context** - `doterr` pattern attaches metadata to errors for better debugging
- **In-process testing** - `cliutiltest.Run` runs a whole command line and returns its output and exit code
- **Production-focused** - Designed for maintainable apps, not quick scripts

## Comparison with Other Frameworks

| Feature | **cliutil** | Cobra | urfave/cli |
|---------|------------|-------|------------|
| **Architecture** | Enforced (opinionated) | Flexible | Flexible |
| **Command Registration** | Auto via `init()` | Manual in `main()` | Manual in `main()` |
| **Project Structure** | Required pattern | Suggested | None |
| **Options Pattern** | Typed app options embedding `GlobalOptions` | Single-tier | Single-tier |
| **Logging Integration** | Built-in WriterLogger | Manual | Manual |
| **Error Handling** | doterr with metadata | Standard Go errors | Standard Go errors |
| **Verbosity Levels** | 3 levels + quiet | Manual | Manual |
| **Type Safety** | Domain types (go-dt) | Basic types | Basic types |
| **Best For** | Production apps | All use cases | Simple CLIs |

## Standard Features

Like other CLI frameworks, `cliutil` also provides:
- Command hierarchies and subcommands
- GNU-style flag parsing with shortcuts (e.g., `--verbosity`, `-v`)
- Positional operand handling
- Generated help text
- Standard exit codes

## Installation

```bash
go get github.com/mikeschinkel/go-cliutil
```

`cliutil` requires Go 1.27 or later.

## Quick Start

### Simple CLI Application

This application has one command and no command word, so it runs in `FlagsOnlyMode`. It shows the smallest amount of boilerplate an app needs.

```go
package main

import (
	"os"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
)

func init() {
	cliutil.RegisterCommand(NewHelloCmd)
}

type HelloCmd struct {
	*cliutil.CmdBase
	name string
}

func NewHelloCmd() cliutil.Command {
	c := &HelloCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "hello",
		Description: "Say hello",
		ArgDefs: []*cliutil.ArgDef{
			{Name: "name", Usage: "Who to greet", DefaultString: "World", String: &c.name},
		},
	})
	return c
}

func (c *HelloCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("Hello, %s!\n", c.name)
	args.Writer.V2().Printf("Verbose output\n")
	args.Writer.V3().Printf("Very verbose output\n")
	return nil
}

func main() {
	runner := cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
		AppInfo: appinfo.New(appinfo.Args{
			Name:        "hello",
			Description: "Say hello",
			ExeName:     "hello",
		}),
		Mode: cliutil.FlagsOnlyMode,
	})
	err := runner.Run(os.Args[1:])
	if err != nil {
		cliutil.Stderrf("Error: %v\n", err)
	}
	os.Exit(cliutil.ExitCode(err))
}
```

```bash
hello                # Hello, World!
hello Alice -v 2     # Hello, Alice! and the verbose line
```

[`examples/basic/main.go`](examples/basic/main.go) shows the smallest app with command words, and [`examples/comprehensive`](examples/comprehensive) exercises every feature.

### Command-Based Application

The `cliutil` package follows a specific architectural pattern for command-based applications. Here's the proper structure:

#### Project Structure

**Single Executable:**

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go                # Entry point - calls myapppkg.RunCLI()
├── myapppkg/
│   ├── init.go                    # Blank import: _ "myapp/myapppkg/myappcmds"
│   ├── run_cli.go                 # RunCLI() - creates the runner, runs the command line
│   ├── myapp/
│   │   ├── config.go              # Application config
│   │   └── options.go             # Application options
│   └── myappcmds/
│       ├── greet_cmd.go           # Greet command
│       └── help_cmd.go            # Help command
└── go.mod
```

**Multiple Executables:**

```
myproject/
├── cmd/
│   ├── app1/
│   │   └── main.go                # Calls app1pkg.RunCLI()
│   └── app2/
│       └── main.go                # Calls app2pkg.RunCLI()
├── app1pkg/
│   ├── init.go                    # _ "myproject/app1pkg/app1cmds"
│   ├── run_cli.go
│   └── app1cmds/
│       └── ...
├── app2pkg/
│   ├── init.go                    # _ "myproject/app2pkg/app2cmds"
│   ├── run_cli.go
│   └── app2cmds/
│       └── ...
└── go.mod
```

#### 1. Entry Point: `cmd/myapp/main.go`

```go
package main

import (
	"os"

	"github.com/yourorg/myapp/myapppkg"
)

func main() {
	os.Exit(myapppkg.RunCLI(os.Args[1:]))
}
```

#### 2. CLI Runner: `myapppkg/run_cli.go`

`RunCLI` creates the runner, runs the command line, reports any error and returns the exit code. The runner never prints the error a run returns, so this function owns that message.

```go
package myapppkg

import (
	"errors"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/yourorg/myapp/myapppkg/myapp"
)

func RunCLI(words []string) (code int) {
	var err error
	var runner *cliutil.CmdRunner

	err = cliutil.Initialize()
	if err != nil {
		cliutil.Stderrf("Error: %v\n", err)
		code = cliutil.ExitInitializationError
		goto end
	}

	runner = cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
		AppInfo: appinfo.New(appinfo.Args{
			Name:        "myapp",
			Description: "Greets people",
			ExeName:     "myapp",
			InfoURL:     "https://github.com/yourorg/myapp",
		}),
		Config:  myapp.LoadConfig,
		Options: myapp.NewOptions,
	})

	err = runner.Run(words)
	if err != nil && !errors.Is(err, cliutil.ErrOmitUserNotify) {
		cliutil.Stderrf("Error: %v\n", err)
	}
	code = cliutil.ExitCode(err)

end:
	return code
}
```

`cliutil.Initialize()` records the user's home directory so `Writer.Errorf` can shorten paths to `~`. Call it once per process, before the first run. An app that skips it prints paths in full.

`CmdRunnerArgs` accepts these fields, and every one is optional:

| Field | Purpose | When unset |
|-------|---------|------------|
| `AppInfo` | Name, description, version and URL shown in help | A value named after the executable |
| `Logger` | The `*slog.Logger` handlers receive | A logger that discards everything |
| `Writer` | The `Writer` handlers receive | A `Writer` on `IO` that honors `--quiet` and `--verbosity` |
| `Context` | The context handlers receive | `context.Background()` |
| `Registry` | The commands the app can run | `cliutil.DefaultRegistry()` |
| `Config` | The `ConfigFunc` that loads config | The run has no config |
| `Options` | The `OptionsFunc` that builds the global flags | `NewGlobalOptions` with no arguments |
| `IO` | Input, output and error streams | The process streams |
| `Mode` | `CommandMode` or `FlagsOnlyMode` | `CommandMode` |

Leave `Writer` unset unless you need a custom implementation. A `Writer` you supply replaces the run's own, so `--quiet` and `--verbosity` do not affect it.

#### 3. Command Package Import: `myapppkg/init.go`

```go
package myapppkg

import (
	// Blank import to trigger command registration
	_ "github.com/yourorg/myapp/myapppkg/myappcmds"
)
```

#### 4. Command Implementation: `myapppkg/myappcmds/greet_cmd.go`

```go
package myappcmds

import (
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/yourorg/myapp/myapppkg/myapp"
)

var _ cliutil.CommandHandler = (*GreetCmd)(nil)

func init() {
	cliutil.RegisterCommand(NewGreetCmd)
}

type GreetCmd struct {
	*cliutil.CmdBase
	name string
	loud bool
}

func NewGreetCmd() cliutil.Command {
	c := &GreetCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "greet",
		Usage:       "greet [--loud] <name>",
		Description: "Greet someone by name",
		Order:       1,
		ArgDefs: []*cliutil.ArgDef{
			{
				Name:     "name",
				Usage:    "Name of person to greet",
				Required: true,
				String:   &c.name,
				Example:  "Alice",
			},
		},
		FlagSets: []*cliutil.FlagSet{
			{
				Name: "greet",
				FlagDefs: []cliutil.FlagDef{
					{
						Name:     "loud",
						Shortcut: 'l',
						Usage:    "Use loud greeting",
						Bool:     &c.loud,
					},
				},
			},
		},
	})
	return c
}

func (c *GreetCmd) Handle(args cliutil.HandleArgs) error {
	config := cliutil.ConfigAs[*myapp.Config](args.Config)
	options := args.OptionsAs[*myapp.Options]()

	greeting := config.Greeting + ", " + c.name + " from " + options.Team + "!"
	if c.loud {
		greeting = strings.ToUpper(greeting)
	}
	args.Writer.Printf("%s\n", greeting)
	return nil
}
```

#### 5. Help Command: `myapppkg/myappcmds/help_cmd.go`

Each app registers its own help command. Give it a variadic operand that holds the topic:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

func init() {
	cliutil.RegisterCommand(NewHelpCmd)
}

type HelpCmd struct {
	*cliutil.CmdBase
	topic []string
}

func NewHelpCmd() cliutil.Command {
	c := &HelpCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "help",
		Usage:       "help [command]",
		Description: "Show help information",
		Order:       99,
		ArgDefs: []*cliutil.ArgDef{
			{Name: "command", Usage: "Command to explain", Strings: &c.topic},
		},
	})
	return c
}

func (c *HelpCmd) Handle(args cliutil.HandleArgs) error {
	if len(c.topic) == 0 {
		return cliutil.ShowMainHelp(args.UsageArgs())
	}
	return cliutil.ShowCmdHelp(c.topic, args.UsageArgs())
}
```

The command must be named `help`, because the parser routes `--help` and an empty command line to it. See [Help](#help).

#### 6. Config and Options: `myapppkg/myapp/options.go` and `myapppkg/myapp/config.go`

```go
// options.go
package myapp

import "github.com/mikeschinkel/go-cliutil"

var _ cliutil.Options = (*Options)(nil)

type Options struct {
	*cliutil.GlobalOptions
	Team string
}

// NewOptions builds a fresh Options value for each run.
func NewOptions() cliutil.Options {
	return &Options{
		GlobalOptions: cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{}),
	}
}

// FlagDefs declares the app's own global flags, targeting this value's fields.
func (o *Options) FlagDefs() []cliutil.FlagDef {
	return []cliutil.FlagDef{
		{Name: "team", Usage: "Team to sign greetings with", DefaultString: "the team", String: &o.Team},
	}
}
```

```go
// config.go
package myapp

import "github.com/mikeschinkel/go-cliutil"

var _ cliutil.Config = (*Config)(nil)

type Config struct {
	Greeting string
}

// Config marks Config as a cliutil.Config.
func (*Config) Config() {}

// LoadConfig is the app's cliutil.ConfigFunc.
func LoadConfig(cliutil.ConfigArgs) (cliutil.Config, error) {
	return &Config{Greeting: "Hello"}, nil
}
```

#### Usage

```bash
# Run with default verbosity
myapp greet Alice

# Run with loud mode
myapp greet --loud Bob
myapp greet -l Bob

# Run with verbosity; global flags go anywhere on the line
myapp --verbosity 2 greet Alice
myapp greet Alice -v 3

# Run in quiet mode
myapp --quiet greet Alice
myapp -q greet Alice

# Show help
myapp help
myapp help greet
myapp greet --help
```

## Core Concepts

### Commands

Commands are the building blocks of your CLI. Each command:
- Embeds `*cliutil.CmdBase` for common functionality
- Implements the `CommandHandler` interface with a `Handle(cliutil.HandleArgs) error` method
- Has a `CommandFunc` constructor, `func() cliutil.Command`, that the file's `init()` registers
- Can have flags, an operand definition, and subcommands

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

func init() {
	cliutil.RegisterCommand(NewMyCmd)
}

type MyCmd struct {
	*cliutil.CmdBase
	output string // Flag and operand targets are fields of the command.
}

func NewMyCmd() cliutil.Command {
	c := &MyCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "mycmd",
		Description: "Do one thing well",
		FlagSets: []*cliutil.FlagSet{{
			Name: "mycmd",
			FlagDefs: []cliutil.FlagDef{
				{Name: "output", Usage: "Output file path", String: &c.output},
			},
		}},
	})
	return c
}

func (c *MyCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("Writing to %s\n", c.output)
	return nil
}
```

Each run calls every constructor, so each run works on fresh values. Keep flag and operand targets in the struct the constructor returns, not in package-level variables.

#### Registries

`cliutil.RegisterCommand` adds a command to the default registry, which `cliutil.DefaultRegistry()` returns. An app or test that wants its own set of commands creates a separate registry:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

func newTestRegistry() (*cliutil.Registry, error) {
	registry := cliutil.NewRegistry()
	registry.Register(NewMyCmd)
	return registry, registry.Validate(nil)
}
```

`Register` only records the constructor, so it is safe to call from `init()` and returns nothing. `Registry.Validate(opts)` builds every command and reports every definition error, checking command flags against the global flags `opts` declares. A nil `opts` means the built-in global flags. Every run performs the same validation, and a failure exits with `ExitInitializationError`.

Validation rejects:
- A command type registered twice, or two commands that share a path
- A parent type that nothing registers, or a command that is its own ancestor
- A flag that does not set exactly one target, or a default that does not match its target
- A command flag whose name or shortcut matches a global flag
- More than one `ArgDef` on a command

The built-in global flags claim the shortcuts `-v`, `-q`, `-t` and `-f`, so a command cannot use those letters unless the app omits or re-letters the matching global flag.

### HandleArgs

`Handle` receives everything a run needs in one value:

| Field | Type | Contents |
|-------|------|----------|
| `Context` | `context.Context` | The run's context; never nil |
| `IO` | `dt.StdIO` | The run's input, output and error streams |
| `Writer` | `cliutil.Writer` | User-facing output on the run's streams |
| `Logger` | `*slog.Logger` | The run's logger; never nil |
| `Config` | `cliutil.Config` | The value the `ConfigFunc` returned, or nil without one |
| `Options` | `cliutil.Options` | The run's global flags; read it with `OptionsAs` |
| `Runner` | `*cliutil.CmdRunner` | Runs another command line |
| `CommandPath` | `[]string` | The resolved command's names, such as `["repo","setup"]`; includes a default subcommand the user did not type |
| `CommandLine` | `[]string` | Every word after the program name, as typed |
| `Parameters` | `[]string` | Every word after the last typed command word, flags included |
| `Flags` | `cliutil.Flags` | Every flag that applies to this run, global and command |
| `Operands` | `cliutil.Operands` | The positional words only |

For `myapp repo setup --check .`, `CommandLine` holds `["repo","setup","--check","."]`, `Parameters` holds `["--check","."]` and `Operands` holds `.`.

Three helpers read those fields:

```go
package myappcmds

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/yourorg/myapp/myapppkg/myapp"
)

type ReportCmd struct{ *cliutil.CmdBase }

func (c *ReportCmd) Handle(args cliutil.HandleArgs) (err error) {
	// Flag returns a flag's value as T: string, bool, int, int64 or []string.
	limit := args.Flag[int]("limit")

	// OptionsAs returns the run's Options as T.
	options := args.OptionsAs[*myapp.Options]()

	// UsageArgs returns the input ShowMainHelp and ShowCmdHelp need.
	if limit == 0 {
		err = cliutil.ShowMainHelp(args.UsageArgs())
		goto end
	}

	args.Writer.Printf("%d reports for %s\n", limit, options.Team)

end:
	return err
}
```

`args.Flag[T]` panics, naming the flag, command path and types, when the flag is undefined or `T` is wrong. `args.OptionsAs[T]` panics, naming both types, when the options are not a `T`. Both conditions are programming errors, and `cliutiltest.Run` turns the panic into a test failure.

`args.Flags` describes each flag in full:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

func describeFlags(args cliutil.HandleArgs) {
	// Get finds one flag by its long name.
	format, ok := args.Flags.Get("format")
	if ok && format.Source == cliutil.ConfigSource {
		args.Writer.V2().Printf("Using the configured format %v\n", format.Value)
	}

	// All returns every flag in definition order, global flags first.
	for _, flag := range args.Flags.All() {
		args.Logger.Debug("flag", "name", flag.Name, "value", flag.Value, "source", flag.Source.String())
	}

	// Passed returns only the flags the user typed, in the order first typed.
	for _, flag := range args.Flags.Passed() {
		args.Writer.V3().Printf("Typed %v\n", flag.Words)
	}
}
```

| `Flag` field | Contents |
|--------------|----------|
| `Name` | Long name, without dashes |
| `Value` | Resolved value: `string`, `bool`, `int`, `int64` or `[]string` |
| `Source` | `CommandLineSource`, `ConfigSource` or `DefaultSource` |
| `Global` | True for a global flag, false for a command's own flag |
| `Words` | The words as typed, such as `["--format","json"]`; nil when not typed |
| `Def` | The flag's `*FlagDef` |

`Source` reports `ConfigSource` when a `FlagValuer` supplied the value, or when `ApplyConfig` left the target holding something other than the default. An `ApplyConfig` that writes the default value therefore reports `DefaultSource`.

`args.Operands` is a slice of `Operand{Value, Position, Def}`, where `Position` indexes `args.Parameters`. `args.Operands.Strings()` returns the values alone.

### Flags

Flags are named parameters. A command groups its flags into `FlagSets`:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

type ExportCmd struct {
	*cliutil.CmdBase
	output string
	pretty bool
	count  int
	tags   []string
}

func NewExportCmd() cliutil.Command {
	c := &ExportCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "export",
		Description: "Export records",
		FlagSets: []*cliutil.FlagSet{
			{
				Name: "export",
				FlagDefs: []cliutil.FlagDef{
					{
						Name:          "output",
						Shortcut:      'o',
						Usage:         "Output file path",
						String:        &c.output,
						DefaultString: "output.txt",
					},
					{
						Name:        "pretty",
						Shortcut:    'p',
						Usage:       "Indent the output",
						Bool:        &c.pretty,
						DefaultBool: true,
					},
					{
						Name:       "count",
						Usage:      "Number of iterations",
						Int:        &c.count,
						DefaultInt: 10,
					},
					{
						Name:    "tag",
						Usage:   "Tags to match; repeat the flag or separate with commas",
						Strings: &c.tags,
						Split:   true,
					},
				},
			},
		},
	})
	return c
}
```

Each flag sets exactly one target, and may set only the default that matches it:

| Target | Type | Default field |
|--------|------|---------------|
| `String` | `*string` | `DefaultString` |
| `Bool` | `*bool` | `DefaultBool` |
| `Int` | `*int` | `DefaultInt` |
| `Int64` | `*int64` | `DefaultInt64` |
| `Strings` | `*[]string` | `DefaultStrings` |

**Flag features:**
- A long `Name` of lowercase letters, digits and dashes
- A `Shortcut` of one ASCII letter or digit
- Typed defaults, shown in help when they differ from the zero value
- `Required`, which the command line or config can satisfy
- `Regex` and `ValidationFunc` validation
- `Example`, a sample value for generated examples

**Repeating flags:** A `Strings` flag collects a value each time the user repeats it, and those values replace `DefaultStrings`. Any other flag keeps the last value typed.

**Splitting on commas:** `Split` also splits each value of a `Strings` flag on commas. Write `\,` for a literal comma, because the shell removes quotes before the program sees them:

```bash
myapp export --tag a,b --tag 'c\,d'     # tags: "a", "b", "c,d"
```

### Parsing Rules

`cliutil` follows GNU conventions, without bundling of short flags:

| You type | Meaning |
|----------|---------|
| `--name` | A boolean flag set to true, or a flag whose value is the next word |
| `--name=value` | A flag with an attached value |
| `--name value` | A value-taking flag followed by its value |
| `-x` | A boolean shortcut set to true, or a shortcut whose value is the next word |
| `-x value` | A value-taking shortcut followed by its value |
| `-x=value` | A shortcut with an attached value |
| `--` | The end of flags; every later word is an operand |

The parser also applies these rules:

- **One dash means one letter.** The parser treats `-abc` as an unknown flag, not as `-a -b -c` or as `--abc`.
- **A value-taking flag takes the next word,** even when that word starts with a dash, so `--offset -5` and `--format -x` both work. A flag at the end of the line with no value is an error.
- **A boolean never consumes the next word.** Use `--flag=false` to turn off a flag whose default is true. `--flag false` passes `false` as an operand.
- **An unknown flag is an error.** So is a word such as `-5` in operand position; write `-- -5` to pass it as an operand.
- **Global flags go anywhere:** before, between or after the command words.
- **A command's flags go only after its last command word.** `myapp --steps 3 db migrate` is an error, and `myapp db migrate --steps 3` works.
- **Operands and flags can mix** after the command words, so `myapp greet Alice --loud` and `myapp greet --loud Alice` are equivalent.

A command that wraps another program can opt out of parsing with `CmdArgs.Passthrough`. See [Passthrough Commands](#passthrough-commands).

### Operands

Operands are positional parameters. A command declares at most one `ArgDef`, which takes either one value or every remaining value:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

// CopyCmd takes exactly one required operand.
type CopyCmd struct {
	*cliutil.CmdBase
	source string
}

func NewCopyCmd() cliutil.Command {
	c := &CopyCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "copy",
		Description: "Copy a directory",
		ArgDefs: []*cliutil.ArgDef{
			{
				Name:     "source",
				Usage:    "Source directory",
				Required: true,
				String:   &c.source,
				Example:  "/path/to/source",
			},
		},
	})
	return c
}

// TouchCmd takes one to three operands and collects them in a slice.
type TouchCmd struct {
	*cliutil.CmdBase
	files []string
}

func NewTouchCmd() cliutil.Command {
	c := &TouchCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "touch",
		Description: "Touch up to three files",
		ArgDefs: []*cliutil.ArgDef{
			{
				Name:     "file",
				Usage:    "Files to touch",
				Strings:  &c.files,
				MinCount: 1,
				MaxCount: 3,
			},
		},
	})
	return c
}
```

| `ArgDef` field | Purpose |
|----------------|---------|
| `String` | Receives the single value |
| `Strings` | Receives every value and makes the operand variadic |
| `Required` | Requires a value; for `Strings`, at least one |
| `MinCount`, `MaxCount` | Limit how many values a `Strings` target accepts; 0 means no limit |
| `DefaultString` | Value for a `String` target when the user gives no word |
| `DefaultStrings` | Values for a `Strings` target when the user gives no word |
| `Example` | Sample value for generated examples |

Too many or too few operands is an error. The runner writes the command's usage to the error stream and returns an error that maps to `ExitFlagsParseError`.

### Global Options

`GlobalOptions` supplies standard flags to every command:

| Flag | Shortcut | Default | Accessor |
|------|----------|---------|----------|
| `--verbosity` | `-v` | `1` | `Verbosity() Verbosity` |
| `--quiet` | `-q` | `false` | `Quiet() bool` |
| `--timeout` | `-t` | `3` | `Timeout() time.Duration` |
| `--dry-run` | | `false` | `DryRun() bool` |
| `--force` | `-f` | `false` | `Force() bool` |

`GlobalOptions` parses these values and hands them to your code. `--quiet` and `--verbosity` control the run's `Writer`. Your handlers decide what `--timeout`, `--dry-run` and `--force` mean, so the run's `Context` carries no deadline unless a handler adds one.

**Usage:**
```bash
myapp --quiet command           # Suppress output
myapp --verbosity 3 command     # Maximum verbosity
myapp -v 2 command              # Medium verbosity (shorthand)
myapp --timeout 30 command      # 30 second timeout
myapp --dry-run command         # Preview mode
myapp --force command           # Force operation
```

An app customizes these flags through `NewGlobalOptions`, and adds its own by embedding the result in its options type:

```go
package myapp

import "github.com/mikeschinkel/go-cliutil"

type ServerOptions struct {
	*cliutil.GlobalOptions
	Region string
}

// NewServerOptions is the app's cliutil.OptionsFunc. Each run calls it.
func NewServerOptions() cliutil.Options {
	return &ServerOptions{
		GlobalOptions: cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
			// Omit drops built-in flags.
			Omit: []string{cliutil.DryRunFlagName},
			// Overrides changes built-in flags, matched by Name.
			Overrides: []cliutil.FlagDef{
				{Name: cliutil.TimeoutFlagName, DefaultInt: 30, Usage: "Seconds to wait for the server"},
			},
		}),
	}
}

// FlagDefs declares the app's own global flags.
func (o *ServerOptions) FlagDefs() []cliutil.FlagDef {
	return []cliutil.FlagDef{
		{Name: "region", Shortcut: 'r', Usage: "Region to use", DefaultString: "us", String: &o.Region},
	}
}
```

- **`Omit`** removes a flag from parsing and help. Calling an omitted flag's method, such as `DryRun()` above, panics.
- **`Overrides`** merges non-zero fields into the built-in flag. A zero value keeps the built-in setting, so an override cannot reset a default to `0` or `false`. An override may not set a target.
- **`FlagDefs()`** declares global flags whose targets are fields of the same value.
- **`OptionsFunc`** builds a fresh value for each run, so `CmdRunnerArgs.Options` takes the function, not a value.

The `cliutil.DefaultOptions` interface declares the accessors in the table above, and `*GlobalOptions` implements it. An app that does not embed `*GlobalOptions` has no built-in flags. Such a type needs only the `Options()` marker method, and it can implement `DefaultOptions` to control the run's quiet and verbosity settings.

### Config

`go-cliutil` imports no config library. Your `ConfigFunc` loads config however the app stores it, and the runner calls it once per run after parsing the command line:

```go
package myapp

import (
	"github.com/mikeschinkel/go-cliutil"
)

type AppConfig struct {
	Format string
	Values map[string]any
}

// Config marks AppConfig as a cliutil.Config.
func (*AppConfig) Config() {}

// FlagValue supplies a value for any flag the user did not type.
func (c *AppConfig) FlagValue(name string) (value any, ok bool) {
	value, ok = c.Values[name]
	return value, ok
}

// LoadAppConfig is the app's cliutil.ConfigFunc.
func LoadAppConfig(args cliutil.ConfigArgs) (cliutil.Config, error) {
	// args.Flags already holds the parsed flags, so a --config flag can choose the file.
	// args.AppInfo and args.IO are also available here.
	return &AppConfig{Format: "yaml"}, nil
}

// ConfiguredCmd copies config into its own fields.
type ConfiguredCmd struct {
	*cliutil.CmdBase
	format string
}

// ApplyConfig implements cliutil.ConfigApplier.
func (c *ConfiguredCmd) ApplyConfig(cfg cliutil.Config) error {
	c.format = cliutil.ConfigAs[*AppConfig](cfg).Format
	return nil
}
```

Two routes carry config values into flags:

- **`ConfigApplier`** copies values into typed fields, so the compiler checks every mapping. Each run calls `ApplyConfig` on the options value and then on the matched command.
- **`FlagValuer`** supplies values by flag name, which suits libraries such as Viper. The `Config` value implements `FlagValue(name string) (any, bool)`, and the runner converts the result to the flag's type. It runs after `ApplyConfig`.

A flag's value comes from the **command line, then config, then its typed default**. `cliutil.ConfigAs[T]` returns the config as `T` and panics, naming both types, when it is not a `T`.

Keep these details in mind:
- The runner calls `ConfigFunc` for every run, including `help`, so a `ConfigFunc` that fails also blocks help. Treat a missing optional config file as an empty config, not as an error.
- A typed operand beats `ApplyConfig`, just as a typed flag does. `ApplyConfig` can replace an operand's default.
- The runner validates values from the command line and from config, but not typed defaults.

### Help

`ShowMainHelp` writes the app's overview, and `ShowCmdHelp` writes one command's usage. Both write to standard output and take the value `args.UsageArgs()` returns. The app registers its own `help` command, as [step 5](#5-help-command-myapppkgmyappcmdshelp_cmdgo) shows.

The parser routes these command lines to that command:

| You type | The parser runs |
|----------|-----------------|
| `myapp` | `myapp help` |
| `myapp --quiet` (only global flags) | `myapp help --quiet` |
| `myapp db migrate --help` | `myapp help db migrate` |

`--help` works anywhere before a `--` and takes priority over every other meaning of the word, including the value of a flag. `-h` has no special meaning. An app with no `help` command reports `unknown command: help` for these command lines.

### Writer Interface

The `Writer` interface provides verbosity-aware output:

```go
type Writer interface {
	Printf(string, ...any) // Normal output
	Errorf(string, ...any) // Error output
	Loud() Writer          // Ignore quiet mode
	V2() Writer            // Verbosity level 2
	V3() Writer            // Verbosity level 3
	Writer() io.Writer     // Underlying stdout
	ErrWriter() io.Writer  // Underlying stderr
}
```

**Verbosity levels:**
- `LowVerbosity (1)` - Normal output (default)
- `MediumVerbosity (2)` - Detailed output
- `HighVerbosity (3)` - Debug output

`--verbosity` also accepts `0` (`NoVerbosity`), which the run's `Writer` treats like `1`.

**Example:**
```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

func report(writer cliutil.Writer, err error) {
	writer.Printf("Shown unless --quiet is set\n")
	writer.V2().Printf("Shown with --verbosity 2 or higher\n")
	writer.V3().Printf("Shown with --verbosity 3\n")
	writer.Loud().Printf("Shown even with --quiet\n")
	writer.Errorf("Error: %v\n", err)
}
```

`--quiet` silences `Printf` on the run's `Writer`. It does not affect `Errorf` or `Loud()`, and the `V2()` and `V3()` writers check only the verbosity level, so `--quiet --verbosity 2` still prints `V2()` output.

### WriterLogger

Combines `Writer` and `*slog.Logger` for unified output:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

func logBoth(args cliutil.HandleArgs) error {
	wl := cliutil.NewWriterLogger(args.Writer, args.Logger)

	// Console and log output
	wl.InfoPrint("message", "key", "value")

	// Console and log, ignore quiet mode
	wl.InfoLoud("message", "key", "value")

	// Warning to console and log
	wl.WarnError("warning", "key", "value")

	// Error to console and log, return error
	return wl.ErrorError("failed", "key", "value")
}
```

### Exit Codes

`cliutil.ExitCode(err)` converts the error a run returned into the process exit code:

| Condition | Constant | Code |
|-----------|----------|------|
| No error | `ExitSuccess` | 0 |
| Unknown flag or command, invalid or missing flag value, wrong operand count | `ExitFlagsParseError` | 1 |
| Invalid command or flag definition | `ExitInitializationError` | 2 |
| `ConfigFunc` failed | `ExitConfigLoadError` | 2 |
| `ApplyConfig` or a `FlagValuer` value failed | `ExitConfigParseError` | 3 |
| Any other error, such as one a handler returned | `ExitKnownRuntimeError` | 4 |

The package also defines `ExitOptionsParseError` (1), `ExitExecutionError` (4), `ExitUnknownRuntimeError` (5), `ExitUnexpectedError` (5) and `ExitLoggerSetupError` (6) for code outside a run. `ExitCode` never returns 5 or 6.

## Advanced Usage

### Subcommands (Parent-Child Commands)

Commands can be organized in hierarchies (e.g., `myapp db migrate` where `db` is the parent and `migrate` is the child). This section provides comprehensive guidance on implementing parent-child command structures.

#### Overview

Parent-child commands create command trees:
- `myapp db` - Parent command
- `myapp db migrate` - Child command
- `myapp db migrate up` - Grandchild command (three levels)

The main help lists each top-level command with its first subcommand:
```
COMMANDS:
    db [migrate]        Database management commands
```

#### Key Concepts

**Registering a child**: Pass a prototype of each parent after the constructor. `cliutil.RegisterCommand(NewDbMigrateCmd, &DbCmd{})` makes `migrate` a child of `db`. The runner identifies a parent by its type and never runs the prototype, so `&DbCmd{}` needs no fields.

**Registration order does not matter**: The runner builds the tree from every registration when a run starts, so each command can register itself from its own file's `init()`.

**Command Names**:
- Parent: Full command name (e.g., `"db"`)
- Child: **Just the node name** (e.g., `"migrate"`, NOT `"db migrate"`)

**Command Usage**:
- `Usage` replaces the generated usage line in help, so write the full path: `"db migrate [--down]"`
- A command with no `Usage` shows its dotted path, such as `db.migrate [flags]`

**File Organization**:
- **One command per file** (recommended practice)
- File naming: `parent_cmd.go`, `parent_child_cmd.go`
- Clear naming aids code organization and navigation

#### Implementation Pattern

Each command gets its own file and registers itself from that file's `init()`:

```go
// db_cmd.go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

var _ cliutil.CommandHandler = (*DbCmd)(nil)

func init() {
	cliutil.RegisterCommand(NewDbCmd)
}

// DbCmd is the parent command for database operations
type DbCmd struct {
	*cliutil.CmdBase
}

func NewDbCmd() cliutil.Command {
	return &DbCmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			Name:        "db",
			Usage:       "db <subcommand>",
			Description: "Database management commands",
			Order:       10,
		}),
	}
}

func (c *DbCmd) Handle(args cliutil.HandleArgs) error {
	return cliutil.ShowCmdHelp(args.CommandPath, args.UsageArgs())
}
```

```go
// db_migrate_cmd.go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

var _ cliutil.CommandHandler = (*DbMigrateCmd)(nil)

func init() {
	cliutil.RegisterCommand(NewDbMigrateCmd, &DbCmd{})
}

// DbMigrateCmd handles database migrations
type DbMigrateCmd struct {
	*cliutil.CmdBase
	down bool
}

func NewDbMigrateCmd() cliutil.Command {
	c := &DbMigrateCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "migrate", // Just the node name
		Usage:       "db migrate [--down]",
		Description: "Run database migrations",
		Order:       1,
		FlagSets: []*cliutil.FlagSet{
			{
				Name: "migrate",
				FlagDefs: []cliutil.FlagDef{
					{Name: "down", Usage: "Roll migrations back", Bool: &c.down},
				},
			},
		},
	})
	return c
}

func (c *DbMigrateCmd) Handle(args cliutil.HandleArgs) error {
	if c.down {
		args.Writer.Printf("Rolling back migrations...\n")
		return c.runDownMigrations()
	}
	args.Writer.Printf("Running up migrations...\n")
	return c.runUpMigrations()
}

func (c *DbMigrateCmd) runUpMigrations() error {
	// Implementation
	return nil
}

func (c *DbMigrateCmd) runDownMigrations() error {
	// Implementation
	return nil
}
```

**File Naming Convention**:
- Parent: `parent_cmd.go` (e.g., `db_cmd.go`)
- Children: `parent_child_cmd.go` (e.g., `db_migrate_cmd.go`, `db_backup_cmd.go`)
- Consistent naming aids code navigation and maintainability

#### Multi-Level Nesting

Commands can be nested multiple levels deep. Each level names its immediate parent:

```go
// In each command's own file:
func init() {
	cliutil.RegisterCommand(NewDbCmd)                           // myapp db
	cliutil.RegisterCommand(NewDbMigrateCmd, &DbCmd{})          // myapp db migrate
	cliutil.RegisterCommand(NewDbMigrateUpCmd, &DbMigrateCmd{}) // myapp db migrate up
}
```

**File naming for deep nesting**:
- Level 1: `db_cmd.go`
- Level 2: `db_migrate_cmd.go`
- Level 3: `db_migrate_up_cmd.go`

#### Multiple Children

A parent can have multiple children, and each child names the same parent type:

```go
// In each command's own file:
func init() {
	cliutil.RegisterCommand(NewDbMigrateCmd, &DbCmd{})
	cliutil.RegisterCommand(NewDbBackupCmd, &DbCmd{})
	cliutil.RegisterCommand(NewDbRestoreCmd, &DbCmd{})
}
```

A command that lists several parents appears under each one.

#### Best Practices

**1. Parent Command Behavior**

A parent that has no `Handle` method returns an error when the user runs it alone, so give each parent one of these behaviors:

- **Show help for its subcommands** (recommended):
  ```go
  func (c *DbCmd) Handle(args cliutil.HandleArgs) error {
  	return cliutil.ShowCmdHelp(args.CommandPath, args.UsageArgs())
  }
  ```

- **Delegate to a default child** (see [Command Delegation](#command-delegation)):
  ```go
  func NewDbCmd() cliutil.Command {
  	return &DbCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
  		Name:       "db",
  		DelegateTo: &DbMigrateCmd{}, // Default to migrate
  	})}
  }
  ```

- **Provide standalone functionality** (less common):
  ```go
  func (c *DbCmd) Handle(args cliutil.HandleArgs) error {
  	// Show database status if no subcommand
  	return c.showStatus(args)
  }
  ```

**2. Consistent Naming**

- Command type names: `ParentChildCmd` (e.g., `DbMigrateCmd`, `DbBackupCmd`)
- Constructors: `NewParentChildCmd` (e.g., `NewDbMigrateCmd`)
- File names: `parent_child_cmd.go` (e.g., `db_migrate_cmd.go`)

**3. Error Handling**

`RegisterCommand` only records the command, so it cannot fail and returns nothing. `Registry.Validate` and every run report definition errors, such as a `FlagName` that conflicts with a global flag. Treat them as programming errors, and call `Validate` from a test so they fail the build.

#### Common Pitfalls

**Wrong: Using the full path in a child's Name**
```go
cliutil.NewCmdBase(cliutil.CmdArgs{
	Name: "db migrate", // Don't include the parent's name
})
```

**Correct: Using just the node name**
```go
cliutil.NewCmdBase(cliutil.CmdArgs{
	Name: "migrate",
})
```

**Wrong: Sharing state through package-level targets**
```go
var down bool // Every run and every parallel test shares this variable

func NewDbMigrateCmd() cliutil.Command {
	c := &DbMigrateCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name: "migrate",
		FlagSets: []*cliutil.FlagSet{{
			Name:     "migrate",
			FlagDefs: []cliutil.FlagDef{{Name: "down", Bool: &down}},
		}},
	})
	return c
}
```

**Correct: Targeting a field of the value the constructor returns**
```go
func NewDbMigrateCmd() cliutil.Command {
	c := &DbMigrateCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name: "migrate",
		FlagSets: []*cliutil.FlagSet{{
			Name:     "migrate",
			FlagDefs: []cliutil.FlagDef{{Name: "down", Bool: &c.down}},
		}},
	})
	return c
}
```

#### Help System Integration

The help system automatically understands parent-child relationships:

```bash
# Show all commands
$ myapp help
COMMANDS:
    db [migrate]        Database management commands

# Show parent help
$ myapp db --help
USAGE:

   myapp db <subcommand>

Database management commands

SUBCOMMANDS:
   migrate  Run database migrations
   backup   Backup database

# Show child help
$ myapp db migrate --help
USAGE:

   myapp db migrate [--down]

Run database migrations

OPTIONS:
   --down    Roll migrations back [optional]
```

#### Testing Parent-Child Commands

```go
package myappcmds_test

import (
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil/cliutiltest"
	_ "github.com/yourorg/myapp/myapppkg/myappcmds"
)

func TestDbMigrateCmd(t *testing.T) {
	result := cliutiltest.Run(t, cliutiltest.Args{
		CommandLine: []string{"db", "migrate", "--down"},
	})
	if result.Err != nil {
		t.Fatalf("Command failed: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "Rolling back") {
		t.Errorf("Unexpected output: %s", result.Stdout)
	}
}
```

#### Complete Working Example

[`examples/comprehensive/commands.go`](examples/comprehensive/commands.go) builds a three-level tree: `project`, `project list`, `project member` and `project member add`.

### Command Delegation

Delegate to a default subcommand. `DelegateTo` takes a prototype, which the runner matches by type:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

type ProjectCmd struct{ *cliutil.CmdBase }
type ProjectListCmd struct{ *cliutil.CmdBase }

func NewProjectCmd() cliutil.Command {
	return &ProjectCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "project",
		Description: "Work with projects",
		DelegateTo:  &ProjectListCmd{},
	})}
}
```

`myapp project` now runs `project list`, and accepts the flags and operands of `project list`. `args.CommandPath` holds `["project","list"]` even though the user did not type `list`.

### Flag Commands

Commands triggered by flags (e.g., `--version` instead of `version`):

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

type VersionCmd struct{ *cliutil.CmdBase }

func NewVersionCmd() cliutil.Command {
	return &VersionCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "version",
		Description: "Show the version",
		FlagName:    "version", // Enables the --version flag
	})}
}
```

Only a top-level command can set `FlagName`. The user must type the flag in its long form, without a value, before any command word. Help lists it with the global flags. If the name is already a global flag or another command's `FlagName`, validation returns one error carrying the `command_name` and `flag_name`, whichever order things were registered in.

### Passthrough Commands

A `Passthrough` command receives every word after its command words unparsed, which suits a command that wraps another program:

```go
package myappcmds

import (
	"strings"

	"github.com/mikeschinkel/go-cliutil"
)

type ExecCmd struct{ *cliutil.CmdBase }

func NewExecCmd() cliutil.Command {
	return &ExecCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "exec",
		Description: "Pass words to another program",
		Passthrough: true,
	})}
}

func (c *ExecCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("exec %s\n", strings.Join(args.Parameters, " "))
	return nil
}
```

`args.Parameters` and `args.Operands` both hold every later word, including `--`, `--help` and anything shaped like a global flag. Global flags for this command must come before its command words: `myapp --quiet exec ls --all`. A `Passthrough` command cannot declare flags or an `ArgDef`.

### Flags-Only Applications

`FlagsOnlyMode` suits a tool that does one thing. The app registers exactly one command, and the user types no command word:

```go
package main

import (
	"os"

	"github.com/mikeschinkel/go-cliutil"
)

func run() error {
	runner := cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
		Mode: cliutil.FlagsOnlyMode,
	})
	return runner.Run(os.Args[1:])
}
```

Every word becomes a flag or an operand of that command. A registry with any other number of commands makes the run fail. The parser does not route `--help` in this mode, so an app that wants it declares a `help` flag on its command and calls `cliutil.ShowCmdHelp` from `Handle`.

### Custom Validation

Add custom validation to flags:

```go
package myappcmds

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
)

func emailFlag(target *string) cliutil.FlagDef {
	return cliutil.FlagDef{
		Name:   "email",
		Usage:  "Email address",
		String: target,
		Regex:  regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		ValidationFunc: func(value any) error {
			email := value.(string)
			if strings.HasSuffix(email, ".invalid") {
				return fmt.Errorf("invalid email domain")
			}
			return nil
		},
	}
}
```

`Regex` checks a `String` value and each value of a `Strings` flag. `ValidationFunc` receives the flag's typed value: `string`, `bool`, `int`, `int64` or `[]string`.

### Examples in Help

Add custom examples to commands. `ShowCmdHelp` lists them:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

type DeployCmd struct{ *cliutil.CmdBase }

func NewDeployCmd() cliutil.Command {
	return &DeployCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "deploy",
		Description: "Deploy the app",
		Examples: []cliutil.Example{
			{
				Descr: "Deploy to production",
				Cmd:   "myapp deploy --env production",
			},
			{
				Descr: "Deploy to staging with dry-run",
				Cmd:   "myapp deploy --env staging --dry-run",
			},
		},
	})}
}
```

`CmdArgs` also offers `Order`, which sets the position in the main help (0 sorts last), and `Hide`, which keeps a runnable command out of help.

### Context Support

Commands receive context for cancellation. Apply `--timeout` to it when a command should honor that flag:

```go
package myappcmds

import (
	"context"

	"github.com/mikeschinkel/go-cliutil"
)

type FetchCmd struct{ *cliutil.CmdBase }

func (c *FetchCmd) Handle(args cliutil.HandleArgs) (err error) {
	var cancel context.CancelFunc
	var ctx context.Context

	options := args.OptionsAs[cliutil.DefaultOptions]()
	ctx, cancel = context.WithTimeout(args.Context, options.Timeout())
	defer cancel()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case result := <-doWork(ctx):
		args.Writer.Printf("Result: %v\n", result)
	}
	return err
}

func doWork(ctx context.Context) <-chan string {
	results := make(chan string, 1)
	results <- "done"
	return results
}
```

### Running Another Command

`args.Runner.Run` lets one command run another command line:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

type SummaryCmd struct{ *cliutil.CmdBase }

func (c *SummaryCmd) Handle(args cliutil.HandleArgs) error {
	words := []string{"project", "list", "--limit", "2"}

	// A nested run parses its own words, so forward the global flags the user typed.
	for _, flag := range args.Flags.Passed() {
		if flag.Global {
			words = append(words, flag.Words...)
		}
	}
	return args.Runner.Run(words)
}
```

Each call builds fresh command and options values. The nested run does not inherit the outer run's flags, which is why the handler above forwards them.

## Architecture Patterns

### Typed Options Pattern

Keep flag targets simple, and convert them to validated domain types where the app uses them. Your options type holds the raw flag values and offers typed accessors:

```go
package myapp

import (
	"fmt"
	"time"

	"github.com/mikeschinkel/go-cliutil"
)

type ServerPort int

// ParseServerPort converts a raw flag value to a validated port.
func ParseServerPort(raw int) (port ServerPort, err error) {
	if raw < 1 || raw > 65535 {
		err = fmt.Errorf("port %d is out of range", raw)
		goto end
	}
	port = ServerPort(raw)

end:
	return port, err
}

type AppOptions struct {
	*cliutil.GlobalOptions
	port  int
	grace int
}

func NewAppOptions() cliutil.Options {
	return &AppOptions{GlobalOptions: cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{})}
}

func (o *AppOptions) FlagDefs() []cliutil.FlagDef {
	return []cliutil.FlagDef{
		{Name: "port", Usage: "Port to listen on", DefaultInt: 8080, Int: &o.port},
		{Name: "grace", Usage: "Shutdown grace period in seconds", DefaultInt: 5, Int: &o.grace},
	}
}

// Port returns the --port value as a validated domain type.
func (o *AppOptions) Port() (ServerPort, error) {
	return ParseServerPort(o.port)
}

// Grace returns the --grace value as a duration.
func (o *AppOptions) Grace() time.Duration {
	return time.Duration(o.grace) * time.Second
}
```

### Initializer Pattern

Packages can register functions that receive the app's `Writer` during setup. The runner does not call them, so the app calls `cliutil.CallInitializerFuncs` before it starts:

```go
package mypkg

import "github.com/mikeschinkel/go-cliutil"

var pkgWriter cliutil.Writer

func init() {
	cliutil.RegisterInitializerFunc(func(args cliutil.InitializerArgs) error {
		// Initialize package-level state
		pkgWriter = args.Writer
		return nil
	})
}
```

## Real-World Examples

### Example 1: Minimal Application

[`examples/basic/main.go`](examples/basic/main.go) defines two commands and help in one file.

### Example 2: Every Feature

[`examples/comprehensive`](examples/comprehensive) demonstrates:
- Nested commands with a default subcommand
- App options with `Omit`, `Overrides` and app-defined global flags
- `ConfigApplier` and `FlagValuer` config
- Split `Strings` flags and variadic operands with count limits
- Flag commands, hidden commands and `Passthrough` commands
- One command running another through `args.Runner`
- A table of tests that run through `cliutiltest`

## Integration with Other Packages

### go-dt (Domain Types)

Use domain types for type safety:

```go
package myappcmds

import (
	"fmt"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

type ConvertCmd struct {
	*cliutil.CmdBase
	inputFile  string
	outputFile string
}

func (c *ConvertCmd) Handle(args cliutil.HandleArgs) (err error) {
	var input, output dt.Filepath
	var exists bool

	// Convert strings to domain types
	input, err = dt.ParseFilepath(c.inputFile)
	if err != nil {
		goto end
	}

	output, err = dt.ParseFilepath(c.outputFile)
	if err != nil {
		goto end
	}

	// Use typed values
	exists, err = input.Exists()
	if err != nil {
		goto end
	}
	if !exists {
		err = fmt.Errorf("input file not found: %s", input)
		goto end
	}

	err = processFiles(input, output)

end:
	return err
}

func processFiles(input, output dt.Filepath) error {
	return nil
}
```

### go-cfgstore (Configuration Storage)

`go-cliutil` imports no config library, so an app can call [go-cfgstore](https://github.com/mikeschinkel/go-cfgstore) from its `ConfigFunc` to load layered configuration.

### go-logutil (Logging Utilities)

Pass any `*slog.Logger` to the runner through `CmdRunnerArgs.Logger`, and handlers receive it as `args.Logger`. [go-logutil](https://github.com/mikeschinkel/go-logutil) provides a JSON file logger for that purpose.

## Best Practices

### 1. Use init() for Command Registration

```go
func init() {
	cliutil.RegisterCommand(NewMyCmd)
}
```

### 2. Direct Field Binding

Point flag and operand targets at fields of the value the constructor returns:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

type BoundCmd struct {
	*cliutil.CmdBase
	dryRun bool
	output string
}

func NewBoundCmd() cliutil.Command {
	c := &BoundCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name: "bound",
		FlagSets: []*cliutil.FlagSet{{
			Name: "bound",
			FlagDefs: []cliutil.FlagDef{
				{Name: "preview", Bool: &c.dryRun},  // Direct binding
				{Name: "output", String: &c.output}, // Direct binding
			},
		}},
	})
	return c
}
```

### 3. Structured Error Handling

Use the doterr pattern for rich error context:

```go
package myappcmds

import (
	"errors"

	"github.com/mikeschinkel/go-cliutil"
)

var ErrOperationFailed = errors.New("operation failed")

type Result struct{}

func doSomething() (Result, error) { return Result{}, nil }

type OperationCmd struct{ *cliutil.CmdBase }

func (c *OperationCmd) Handle(args cliutil.HandleArgs) (err error) {
	var result Result

	result, err = doSomething()
	if err != nil {
		err = cliutil.NewErr(
			ErrOperationFailed,
			"operation", "doSomething",
			"context", "additional info",
			err,
		)
		goto end
	}
	args.Writer.Printf("Result: %v\n", result)

end:
	return err
}
```

### 4. Verbosity-Aware Output

Use appropriate verbosity levels:

```go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

func progress(args cliutil.HandleArgs, count int, item any) {
	args.Writer.Printf("Operation complete\n")             // Normal
	args.Writer.V2().Printf("Processed %d items\n", count) // Detailed
	args.Writer.V3().Printf("Item details: %+v\n", item)   // Debug
}
```

### 5. Type-Safe Configuration

Read configuration through `cliutil.ConfigAs`:

```go
package myappcmds

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/yourorg/myapp/myapppkg/myapp"
)

type GreetingCmd struct{ *cliutil.CmdBase }

func (c *GreetingCmd) Handle(args cliutil.HandleArgs) error {
	config := cliutil.ConfigAs[*myapp.Config](args.Config)

	// Use typed config
	args.Writer.Printf("%s\n", config.Greeting)
	return nil
}
```

### 6. Context Awareness

Respect context cancellation:

```go
package myappcmds

import (
	"context"

	"github.com/mikeschinkel/go-cliutil"
)

type LongCmd struct{ *cliutil.CmdBase }

func longRunningWork(ctx context.Context) error { return ctx.Err() }

func (c *LongCmd) Handle(args cliutil.HandleArgs) (err error) {
	err = args.Context.Err()
	if err != nil {
		goto end
	}

	// Long-running operation
	err = longRunningWork(args.Context)

end:
	return err
}
```

### 7. Consistent Exit Codes

Let `cliutil.ExitCode` choose the exit code, and exit only from `main`:

```go
package main

import (
	"os"

	"github.com/mikeschinkel/go-cliutil"
)

func main() {
	err := cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{}).Run(os.Args[1:])
	if err != nil {
		cliutil.Stderrf("Error: %v\n", err)
	}
	os.Exit(cliutil.ExitCode(err))
}
```

## Best Practices for Writing Commands

Command handlers should follow these principles for maintainability and testability. [`docs/BEST-PRACTICES.md`](docs/BEST-PRACTICES.md) covers them in depth.

### 1. Thin Handlers, Thick Domain Logic

Command `Handle` functions should be **thin orchestrators** that delegate to domain packages:

**Good** (following established patterns):
- Config setup (10-20 lines)
- Single domain function call with args struct (5-10 lines)
- Output formatting (10-30 lines)
- Total: ~60-80 lines

**Bad**:
- Business logic in `Handle`
- Multiple sequential domain calls with conditional logic
- String-based error checking
- Manual struct construction
- Total: >100 lines

### 2. Single Domain Call Pattern

Prefer a single high-level domain function over multiple low-level calls:

```go
// Good: Single cohesive operation
result, err := domain.PrepareResource(&domain.PrepareArgs{...})

// Bad: Multiple steps with conditional logic
source := domain.ResolveSource(...)
if source.Type == X {
    source := domain.ValidateBranch(...)
}
result := domain.Install(...)
validated := domain.Validate(...)
```

### 3. Args Struct Pattern

Use args structs for functions with 3+ parameters:

```go
type PrepareArgs struct {
    Source    string
    ConfigDir dt.DirPath
    Force     bool
    Writer    cliutil.Writer
}

func Prepare(args *PrepareArgs) (*Result, error)
```

Benefits:
- Easy to add parameters
- Self-documenting
- Easy to test (named fields)

### 4. Error Handling

Use consistent error patterns:

```go
result, err := domain.Operation(...)
if err != nil {
    err = NewErr(ErrCmd, ErrContext, err)
    goto end
}
```

For user-facing messages, see [User Notifications](#user-notifications) section.

### 5. Separation of Concerns

**CLI Layer Responsibilities**:
- Flag and operand definitions
- Config resolution
- Output formatting
- Error wrapping with CLI context

**Domain Layer Responsibilities**:
- Business logic
- Validation
- File operations
- Data transformations

**Anti-patterns**:
- Business decisions in CLI (type checks, conditional paths)
- Output formatting in domain layer
- CLI-specific types in domain signatures

### 6. Return Typed Data

Domain functions should return structured data, not strings:

```go
// Good
type DemoInfo struct {
    Name      string
    Path      dt.DirPath
    Size      int64
    UpdatedAt time.Time
}

func ListDemos() ([]DemoInfo, error)

// Bad
func ListDemos() ([]string, error)  // loses type safety
```

### 7. Method Chaining for Output

Put output formatting methods on result types:

```go
type DemoInfos []DemoInfo

func (d DemoInfos) JSON() string
func (d DemoInfos) TableWriter() *tablewriter.Table
func (d DemoInfos) FullNames() []string
```

This keeps formatting logic with the data, not in `Handle`.

## User Notifications

When errors occur that require user-facing messages (not technical stack traces), use the `ErrOmitUserNotify` pattern.

### Pattern

1. **Print user-friendly message** via `args.Writer`
2. **Return error wrapped** with `ErrOmitUserNotify`

### Example: File Already Exists

```go
package myappcmds

import (
	"errors"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

var ErrFilesAlreadyExist = errors.New("files already exist")

type InitCmd struct{ *cliutil.CmdBase }

func (c *InitCmd) promptForOverwrite(args cliutil.HandleArgs, conflicts []dt.RelFilepath) error {
	// 1. User-friendly explanation
	args.Writer.Printf("\nCannot initialize app files because the following files already exist:\n")
	for _, file := range conflicts {
		args.Writer.Printf("  - %s\n", file)
	}
	args.Writer.Printf("\nTo overwrite these files, use the --overwrite flag.\n")
	args.Writer.Printf("Alternatively, move or remove these files before running 'myapp init'.\n")

	// 2. Return error with ErrOmitUserNotify
	return cliutil.NewErr(
		cliutil.ErrOmitUserNotify,
		ErrFilesAlreadyExist,
		"files", conflicts,
	)
}
```

### How It Works

The runner returns every error to its caller without printing it, so the app's entry point checks for `ErrOmitUserNotify`:

```go
package myapppkg

import (
	"errors"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
)

func runAndReport(runner *cliutil.CmdRunner, logger *slog.Logger, words []string) int {
	err := runner.Run(words)
	if err != nil {
		if !errors.Is(err, cliutil.ErrOmitUserNotify) {
			cliutil.Stderrf("Command failed: %v\n", err) // Only print if NOT omit
		}
		logger.Error("Run aborted", "error", err) // Always log
	}
	return cliutil.ExitCode(err)
}
```

### When to Use

Use `ErrOmitUserNotify` when:
- Error requires explanation for user action
- Technical error details aren't helpful
- You've already printed a clear message

Examples:
- "File already exists" → explain --overwrite flag
- "Demo already installed" → explain --reinstall flag
- "Configuration missing" → explain how to create it
- "Permission denied" → explain required permissions

### When NOT to Use

Don't use for:
- Unexpected technical errors (let them print)
- Validation errors (simple message is fine)
- Errors that should show stack trace

### Domain Layer Pattern

Domain functions can also trigger user notifications:

```go
package mydomain

import (
	"errors"

	"github.com/mikeschinkel/go-cliutil"
)

var ErrAlreadyInstalled = errors.New("demo already installed")

type Result struct{}

type PrepareDemoArgs struct {
	Installed bool
	Reinstall bool
	Writer    cliutil.Writer
}

func PrepareDemo(args *PrepareDemoArgs) (*Result, error) {
	if args.Installed && !args.Reinstall {
		// Print user message
		args.Writer.Errorf("Demo already installed. Use --reinstall to re-download.\n")

		// Return valid result + ErrOmitUserNotify
		return &Result{}, cliutil.NewErr(cliutil.ErrOmitUserNotify, ErrAlreadyInstalled)
	}
	return &Result{}, nil
}
```

The entry point handles this transparently - no special code needed in `Handle`.

## Testing

### Testing Commands

`cliutiltest.Run` runs a command line in-process and returns what it produced. Only test code imports the package, so it never reaches an app binary.

```go
package myappcmds_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/cliutiltest"
	"github.com/yourorg/myapp/myapppkg/myapp"
	_ "github.com/yourorg/myapp/myapppkg/myappcmds"
)

func TestGreet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		line     []string
		want     string
		exitCode int
		wantErr  error
	}{
		{name: "greets by name", line: []string{"greet", "Alice"}, want: "Hello, Alice"},
		{name: "loud", line: []string{"greet", "Alice", "--loud"}, want: "HELLO, ALICE"},
		{
			name:     "missing name",
			line:     []string{"greet"},
			exitCode: cliutil.ExitFlagsParseError,
			wantErr:  cliutil.ErrTooFewOperands,
		},
		{
			name:     "unknown flag",
			line:     []string{"greet", "Alice", "--nope"},
			exitCode: cliutil.ExitFlagsParseError,
			wantErr:  cliutil.ErrUnknownFlag,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cliutiltest.Run(t, cliutiltest.Args{
				CommandLine: tt.line,
				Config:      myapp.LoadConfig,
				Options:     myapp.NewOptions,
			})
			if result.ExitCode != tt.exitCode {
				t.Errorf("exit code: want %d, got %d (err: %v)", tt.exitCode, result.ExitCode, result.Err)
			}
			if tt.wantErr != nil && !errors.Is(result.Err, tt.wantErr) {
				t.Errorf("error: want %v, got %v", tt.wantErr, result.Err)
			}
			if !strings.Contains(result.Stdout, tt.want) {
				t.Errorf("stdout: want %q in %q", tt.want, result.Stdout)
			}
		})
	}
}
```

Each call uses its own streams and command values, so parallel subtests can share a registry.

| `cliutiltest.Args` field | Purpose | When unset |
|--------------------------|---------|------------|
| `Registry` | The commands to run | `cliutil.DefaultRegistry()` |
| `CommandLine` | The words after the program name | No words |
| `Config` | The `ConfigFunc` | The run has no config |
| `Options` | The `OptionsFunc` | The built-in global flags |
| `Stdin` | The run's input | Empty input |
| `AppInfo` | The app's name and description | A placeholder named `app` |
| `Mode` | `CommandMode` or `FlagsOnlyMode` | `CommandMode` |
| `Context` | The run's context | `context.Background()` |

`Result` holds `Stdout`, `Stderr`, `ExitCode` and `Err`. A panic, such as a handler asking for an undefined flag or the wrong type, fails the test with the panic's message.

### Validating Definitions

A single test can catch every definition error across the app's commands:

```go
package myappcmds_test

import (
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/yourorg/myapp/myapppkg/myapp"
	_ "github.com/yourorg/myapp/myapppkg/myappcmds"
)

func TestCommandDefinitions(t *testing.T) {
	err := cliutil.DefaultRegistry().Validate(myapp.NewOptions())
	if err != nil {
		t.Fatal(err)
	}
}
```

### Capturing Output

Domain packages that accept a `cliutil.Writer` can be tested without a command line. `WriterArgs.IO` takes a `dt.StdIO`, so a test can point the writer (and the `V2()`, `V3()` and `Loud()` writers it derives) at buffers instead of the process streams. Leave `IO` nil to write to `os.Stdout` and `os.Stderr`.

```go
package mydomain_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

func TestOutput(t *testing.T) {
	var out, errOut bytes.Buffer
	writer := cliutil.NewWriter(&cliutil.WriterArgs{
		IO:        dt.NewStdIO(&out, &errOut),
		Quiet:     false,
		Verbosity: cliutil.LowVerbosity,
	})

	// Call the code under test
	writer.Printf("expected text\n")

	// Check output
	if !strings.Contains(out.String(), "expected text") {
		t.Errorf("Expected output not found: %s", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("Unexpected error output: %s", errOut.String())
	}
}
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - Copyright (c) Mike Schinkel

## Related Packages

- [go-dt](https://github.com/mikeschinkel/go-dt) - Domain types for type safety
- [go-cfgstore](https://github.com/mikeschinkel/go-cfgstore) - Configuration storage
- [go-logutil](https://github.com/mikeschinkel/go-logutil) - Logging utilities
- [go-testutil](https://github.com/mikeschinkel/go-testutil) - Testing utilities
- [go-fsfix](https://github.com/mikeschinkel/go-fsfix) - File system test fixtures

## Support

For issues, questions, or contributions, please visit:
https://github.com/mikeschinkel/go-cliutil
