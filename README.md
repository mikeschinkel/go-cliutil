# CLI Framework for Go — Opinionated and Productive

A comprehensive CLI framework for Go applications that provides command registration, flag parsing, argument handling, and structured output management.

## What Makes cliutil Different

`cliutil` is **opinionated and architectural** - it enforces a specific project structure that scales from simple tools to complex multi-command applications:

- **Enforced architecture** - `cmd/main.go` → `<apppkg>.RunCLI()` → `<apppkg>.Run()` pattern separates concerns
- **Auto-registration via `init()`** - Commands register themselves; blank import `_ "myapp/myappcmds"` loads all commands
- **Type-safe options** - Two-tier pattern: parse raw CLI args, then convert to validated domain types
- **Integrated logging** - `WriterLogger` combines `*slog.Logger` with verbosity-aware console output
- **Rich error context** - `doterr` pattern attaches metadata to errors for better debugging
- **Production-focused** - Designed for maintainable apps, not quick scripts

## Comparison with Other Frameworks

| Feature | **cliutil** | Cobra | urfave/cli |
|---------|------------|-------|------------|
| **Architecture** | Enforced (opinionated) | Flexible | Flexible |
| **Command Registration** | Auto via `init()` | Manual in `main()` | Manual in `main()` |
| **Project Structure** | Required pattern | Suggested | None |
| **Options Pattern** | Two-tier (raw → typed) | Single-tier | Single-tier |
| **Logging Integration** | Built-in WriterLogger | Manual | Manual |
| **Error Handling** | doterr with metadata | Standard Go errors | Standard Go errors |
| **Verbosity Levels** | 4 levels (0-3) + quiet | Manual | Manual |
| **Type Safety** | Domain types (go-dt) | Basic types | Basic types |
| **Best For** | Production apps | All use cases | Simple CLIs |

## Standard Features

Like other CLI frameworks, `cliutil` also provides:
- Command hierarchies and subcommands
- Flag parsing with shortcuts (e.g., `--verbose`, `-v`)
- Positional argument handling
- Automatic help generation
- Standard exit codes

## Installation

```bash
go get github.com/mikeschinkel/go-cliutil
```

## Quick Start

### Simple CLI Application

This is an example of a CLI application that only uses global flags. 

In other word, **this is not a useful applicaiton** at all, but it shows the initial boilerplate required.

```go
package main

import (
    "fmt"
    "os"
    "github.com/mikeschinkel/go-cliutil"
)

func main() {
    // Parse global options
    opts, args, err := cliutil.ParseGlobalOptions(os.Args)
    if err != nil {
        cliutil.Stderrf("Error: %v\n", err)
        os.Exit(1)
    }

    // Create writer with verbosity settings
    writer := cliutil.NewWriter(&cliutil.WriterArgs{
        Quiet:     opts.Quiet(),
        Verbosity: opts.Verbosity(),
    })

    // Use the writer
    writer.Printf("Hello, World!\n")
    writer.V2().Printf("Verbose output\n")
    writer.V3().Printf("Very verbose output\n")
}
```

### Command-Based Application

The `cliutil` package follows a specific architectural pattern for command-based applications. Here's the proper structure:

#### Project Structure

**Single Executable:**

```
myapp/
├── cmd/
│   └── main.go                    # Entry point - calls myapppkg.RunCLI()
├── myapppkg/
│   ├── init.go                    # Blank import: _ "myapp/myapppkg/myappcmds"
│   ├── run_cli.go                 # RunCLI() - parses options, sets up logging
│   ├── run.go                     # Run() - creates runner, executes command
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
│   ├── run.go
│   └── app1cmds/
│       └── ...
├── app2pkg/
│   ├── init.go                    # _ "myproject/app2pkg/app2cmds"
│   ├── run_cli.go
│   ├── run.go
│   └── app2cmds/
│       └── ...
└── go.mod
```

#### 1. Entry Point: `cmd/main.go`

```go
package main

import "github.com/yourorg/myapp/myapppkg"

func main() {
    myapppkg.RunCLI()
}
```

#### 2. CLI Runner: `myapppkg/run_cli.go`

```go
package myapppkg

import (
    "context"
    "os"
    "github.com/mikeschinkel/go-cliutil"
    "github.com/mikeschinkel/go-cfgstore"
    "github.com/yourorg/myapp/myapppkg/myapp"
)

func RunCLI() {
    var err error
    var globalOptions *cliutil.GlobalOptions
    var args []string
    var options *myapp.Options
    var config *myapp.Config
    var wl cliutil.WriterLogger

    // Parse global options
    globalOptions, args, err = cliutil.ParseGlobalOptions(os.Args)
    if err != nil {
        cliutil.Stderrf("Invalid option(s): %v\n", err)
        os.Exit(cliutil.ExitOptionsParseError)
    }

    // Create options
    options, err = myapp.NewOptions(myapp.OptionsArgs{
        GlobalOptions: globalOptions,
    })
    if err != nil {
        cliutil.Stderrf("Failed to create options: %v\n", err)
        os.Exit(cliutil.ExitOptionsParseError)
    }

    // Create logger and writer
    wl, err = cfgstore.CreateWriterLogger(&cfgstore.WriterLoggerArgs{
        Quiet:      options.Quiet(),
        Verbosity:  options.Verbosity(),
        ConfigSlug: "myapp",
        LogFile:    "myapp.log",
    })
    if err != nil {
        cliutil.Stderrf("Failed to setup logger: %v\n", err)
        os.Exit(cliutil.ExitLoggerSetupError)
    }

    // Create config
    config = myapp.NewConfig(myapp.ConfigArgs{
        Options: options,
        Logger:  wl.Logger,
        Writer:  wl.Writer,
    })

    // Run the CLI
    ctx := context.Background()
    err = Run(ctx, &RunArgs{
        CLIArgs: args,
        Config:  config,
        Options: options,
    })

    if err != nil {
        wl.Errorf("Error: %v\n", err)
        os.Exit(cliutil.ExitUnknownRuntimeError)
    }
}
```

#### 3. Command Execution: `myapppkg/run.go`

```go
package myapppkg

import (
    "context"
    "errors"
    "github.com/mikeschinkel/go-cliutil"
    "github.com/yourorg/myapp/myapppkg/myapp"
)

type RunArgs struct {
    CLIArgs []string
    Config  *myapp.Config
    Options *myapp.Options
}

func Run(ctx context.Context, args *RunArgs) error {
    var err error

    // Initialize cliutil
    err = cliutil.Initialize(args.Config.Writer)
    if err != nil {
        return err
    }

    // Create command runner
    runner := cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
        Context: ctx,
        Args:    args.CLIArgs,
        Logger:  args.Config.Logger,
        Writer:  args.Config.Writer,
        Config:  args.Config,
        Options: args.Options,
    })

    // Parse command
    cmd, err := runner.ParseCmd(args.CLIArgs)
    if err != nil {
        if errors.Is(err, cliutil.ErrShowUsage) {
            cliutil.ShowMainHelp(cliutil.UsageArgs{
                Writer: args.Config.Writer,
            })
        }
        return err
    }

    // Execute command
    return runner.RunCmd(cmd)
}
```

#### 4. Command Package Import: `myapppkg/init.go`

```go
package myapppkg

import (
    // Blank import to trigger command registration
    _ "github.com/yourorg/myapp/myapppkg/myappcmds"
)
```

#### 5. Command Implementation: `myapppkg/myappcmds/greet_cmd.go`

```go
package myappcmds

import (
    "strings"
    "github.com/mikeschinkel/go-cliutil"
)

var _ cliutil.CommandHandler = (*GreetCmd)(nil)

type GreetCmd struct {
    *cliutil.CmdBase
    name string
    loud bool
}

func init() {
    cmd := &GreetCmd{}

    cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "greet",
        Usage:       "greet [--loud] <name>",
        Description: "Greet someone by name",
        Order:       1,
        ArgDefs: []*cliutil.ArgDef{
            {
                Name:     "name",
                Usage:    "Name of person to greet",
                Required: true,
                String:   &cmd.name,
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
                        Bool:     &cmd.loud,
                        Default:  false,
                    },
                },
            },
        },
    })

    if err := cliutil.RegisterCommand(cmd); err != nil {
        panic(err)
    }
}

func (c *GreetCmd) Handle() error {
    greeting := "Hello, " + c.name + "!"
    if c.loud {
        greeting = strings.ToUpper(greeting)
    }
    c.Writer.Printf("%s\n", greeting)
    return nil
}
```

#### 6. Config and Options: `myapppkg/myapp/options.go` and `myapppkg/myapp/config.go`

```go
// options.go
package myapp

import "github.com/mikeschinkel/go-cliutil"

var _ cliutil.Options = (*Options)(nil)
var _ cliutil.GlobalOptionsGetter = (*Options)(nil)

type Options struct {
    *cliutil.GlobalOptions
    // Add application-specific options here
}

type OptionsArgs struct {
    GlobalOptions *cliutil.GlobalOptions
}

func NewOptions(args OptionsArgs) (*Options, error) {
    return &Options{
        GlobalOptions: args.GlobalOptions,
    }, nil
}

func (o *Options) Options() {}

func (o *Options) GlobalOptions() *cliutil.GlobalOptions {
    return o.GlobalOptions
}
```

```go
// config.go
package myapp

import (
    "log/slog"
    "github.com/mikeschinkel/go-cliutil"
)

var _ cliutil.Config = (*Config)(nil)

type Config struct {
    Options *Options
    Logger  *slog.Logger
    Writer  cliutil.Writer
}

type ConfigArgs struct {
    Options *Options
    Logger  *slog.Logger
    Writer  cliutil.Writer
}

func NewConfig(args ConfigArgs) *Config {
    return &Config{
        Options: args.Options,
        Logger:  args.Logger,
        Writer:  args.Writer,
    }
}
```

#### Usage

```bash
# Run with default verbosity
myapp greet Alice

# Run with loud mode
myapp greet --loud Bob
myapp greet -l Bob

# Run with verbosity
myapp --verbosity 2 greet Alice
myapp -v 3 greet Alice

# Run in quiet mode
myapp --quiet greet Alice
myapp -q greet Alice

# Show help
myapp help
myapp greet --help
```

## Core Concepts

### Commands

Commands are the building blocks of your CLI. Each command:
- Embeds `*cliutil.CmdBase` for common functionality
- Implements the `CommandHandler` interface with a `Handle()` method
- Is registered during package initialization via `init()` functions
- Can have flags, arguments, and subcommands

```go
type MyCmd struct {
    *cliutil.CmdBase
    // Command-specific fields
}

func (c *MyCmd) Handle() error {
    // Command implementation
    return nil
}
```

### Flags

Flags are optional parameters specified with `--name` or `-n` syntax:

```go
FlagSets: []*cliutil.FlagSet{
    {
        Name: "mycommand",
        FlagDefs: []cliutil.FlagDef{
            {
                Name:     "output",
                Shortcut: 'o',
                Usage:    "Output file path",
                String:   &cmd.outputFile,
                Required: false,
                Default:  "output.txt",
            },
            {
                Name:     "verbose",
                Shortcut: 'v',
                Usage:    "Enable verbose output",
                Bool:     &cmd.verbose,
                Default:  false,
            },
            {
                Name:    "count",
                Usage:   "Number of iterations",
                Int:     &cmd.count,
                Default: 10,
            },
        },
    },
}
```

**Supported flag types:**
- `String` - `*string`
- `Bool` - `*bool`
- `Int` - `*int`
- `Int64` - `*int64`

**Flag features:**
- Shortcut support (single character)
- Default values
- Required validation
- Regex validation
- Custom validation functions

### Arguments

Arguments are positional parameters:

```go
ArgDefs: []*cliutil.ArgDef{
    {
        Name:     "source",
        Usage:    "Source directory",
        Required: true,
        String:   &cmd.source,
        Example:  "/path/to/source",
    },
    {
        Name:     "destination",
        Usage:    "Destination directory",
        Required: false,
        Default:  ".",
        String:   &cmd.dest,
        Example:  "/path/to/dest",
    },
}
```

### Global Options

Standard CLI options available to all commands:

```go
type GlobalOptions struct {
    timeout   *int      // Timeout in seconds
    quiet     *bool     // Suppress output
    verbosity *int      // Verbosity level (0-3)
    dryRun    *bool     // Dry run mode
    force     *bool     // Force operation
}

// Accessor methods
opts.Quiet() bool
opts.Verbosity() Verbosity
opts.Timeout() time.Duration
opts.DryRun() bool
opts.Force() bool
```

**Usage:**
```bash
myapp --quiet command           # Suppress output
myapp --verbosity 3 command     # Maximum verbosity
myapp -v 2 command              # Medium verbosity (shorthand)
myapp --timeout 30 command      # 30 second timeout
myapp --dry-run command         # Preview mode
myapp --force command           # Force operation
```

### Writer Interface

The `Writer` interface provides verbosity-aware output:

```go
type Writer interface {
    Printf(string, ...any)           // Normal output
    Errorf(string, ...any)           // Error output
    Loud() Writer                    // Ignore quiet mode
    V2() Writer                      // Verbosity level 2
    V3() Writer                      // Verbosity level 3
    Writer() io.Writer               // Underlying stdout
    ErrWriter() io.Writer            // Underlying stderr
}
```

**Verbosity levels:**
- `NoVerbosity (0)` - No output
- `LowVerbosity (1)` - Normal output (default)
- `MediumVerbosity (2)` - Detailed output
- `HighVerbosity (3)` - Debug output

**Example:**
```go
writer.Printf("Always shown\n")
writer.V2().Printf("Shown with --verbosity 2 or higher\n")
writer.V3().Printf("Shown with --verbosity 3\n")
writer.Loud().Printf("Shown even with --quiet\n")
writer.Errorf("Error: %v\n", err)
```

### WriterLogger

Combines `Writer` and `*slog.Logger` for unified output:

```go
type WriterLogger struct {
    Writer cliutil.Writer
    Logger *slog.Logger
}

wl := cliutil.NewWriterLogger(writer, logger)

// Console and log output
wl.InfoPrint("message", "key", value)

// Console and log, ignore quiet mode
wl.InfoLoud("message", "key", value)

// Error to console and log, return error
err := wl.ErrorError("failed", "key", value)

// Warning to console and log
wl.WarnError("warning", "key", value)
```

### Exit Codes

Standard exit codes for consistent error handling:

```go
cliutil.ExitSuccess               // 0
cliutil.ExitOptionsParseError     // 1
cliutil.ExitConfigLoadError       // 2
cliutil.ExitConfigParseError      // 3
cliutil.ExitKnownRuntimeError     // 4
cliutil.ExitUnknownRuntimeError   // 5
cliutil.ExitLoggerSetupError      // 6
```

## Advanced Usage

### Subcommands (Parent-Child Commands)

Commands can be organized in hierarchies (e.g., `myapp db migrate` where `db` is the parent and `migrate` is the child). This section provides comprehensive guidance on implementing parent-child command structures.

#### Overview

Parent-child commands create command trees:
- `myapp db` - Parent command
- `myapp db migrate` - Child command
- `myapp db migrate up` - Grandchild command (three levels)

The help output automatically shows the hierarchy:
```
COMMANDS:
    db [migrate, backup]    Database management commands
```

#### Why Package-Level Initialization Matters

Go's initialization order guarantees:
1. **Package-level variables** are initialized first (in declaration order within a file)
2. **init() functions** run after all package variables are initialized
3. This happens **regardless of file order** in the package

By initializing parent commands as package variables (not inside init()), you ensure they're available when child commands' init() functions run. This is dependency-based initialization, not file-name-based.

```go
// Go guarantees this order:
var parentCmd = &ParentCmd{...}  // 1. Variable initialized
var childCmd = &ChildCmd{...}    // 2. Variable initialized

func init() {                     // 3. All init() run after variables
    cliutil.RegisterCommand(parentCmd)
    cliutil.RegisterCommand(childCmd, parentCmd)  // Safe - parentCmd exists
}
```

**Why not init()**: If you initialize in init(), the order depends on file compilation order, which is undefined and fragile. The compiler cannot warn you about this.

#### Key Concepts

**Command Names**:
- Parent: Full command name (e.g., `"db"`)
- Child: **Just the node name** (e.g., `"migrate"`, NOT `"db migrate"`)
- This is critical - using the full path will break registration

**Command Usage**:
- Parent: `"db <subcommand>"` or `"db [options]"`
- Child: `"migrate [options]"` (just the child name, framework builds full path)

**Registration Order**:
- Parent must be registered before children
- Ensure initialization through package-level variable initialization
- Package variables are initialized before any init() functions run

**File Organization**:
- **One command per file** (recommended practice)
- File naming: `parent_cmd.go`, `parent_child_cmd.go`
- Clear naming aids code organization and navigation

#### Implementation Pattern

Each command gets its own file. Parent commands are initialized as package variables (not inside init()), ensuring they're available when child commands' init() functions run:

```go
// db_cmd.go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

var _ cliutil.CommandHandler = (*DbCmd)(nil)

// DbCmd is the parent command for database operations
type DbCmd struct {
    *cliutil.CmdBase
}

// dbCmd is initialized at package initialization time (before init() runs).
// This guarantees child commands can safely reference it in their init() functions.
var dbCmd = &DbCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "db",
        Usage:       "db <subcommand>",
        Description: "Database management commands",
        Order:       10,
    }),
}

func init() {
    // Only register the already-initialized command
    if err := cliutil.RegisterCommand(dbCmd); err != nil {
        panic(err)
    }
}

func (c *DbCmd) Handle() error {
    c.Writer.Printf("Database commands:\n")
    c.Writer.Printf("  migrate  - Run database migrations\n")
    c.Writer.Printf("  backup   - Backup database\n")
    c.Writer.Printf("  restore  - Restore database\n")
    return nil
}
```

```go
// db_migrate_cmd.go
package myappcmds

import "github.com/mikeschinkel/go-cliutil"

var _ cliutil.CommandHandler = (*MigrateCmd)(nil)

// MigrateCmd handles database migrations
type MigrateCmd struct {
    *cliutil.CmdBase
    up   bool
    down bool
}

func init() {
    cmd := &MigrateCmd{}

    cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "migrate",  // Just the node name
        Usage:       "migrate [--up|--down]",
        Description: "Run database migrations",
        Order:       1,
        FlagSets: []*cliutil.FlagSet{
            {
                Name: "migrate",
                FlagDefs: []cliutil.FlagDef{
                    {
                        Name:    "up",
                        Usage:   "Run up migrations",
                        Bool:    &cmd.up,
                        Default: true,
                    },
                    {
                        Name:    "down",
                        Usage:   "Run down migrations",
                        Bool:    &cmd.down,
                        Default: false,
                    },
                },
            },
        },
    })

    // Register with parent using package variable
    if err := cliutil.RegisterCommand(cmd, dbCmd); err != nil {
        panic(err)
    }
}

func (c *MigrateCmd) Handle() error {
    if c.down {
        c.Writer.Printf("Rolling back migrations...\n")
        return c.runDownMigrations()
    }
    c.Writer.Printf("Running up migrations...\n")
    return c.runUpMigrations()
}

func (c *MigrateCmd) runUpMigrations() error {
    // Implementation
    return nil
}

func (c *MigrateCmd) runDownMigrations() error {
    // Implementation
    return nil
}
```

**File Naming Convention**:
- Parent: `parent_cmd.go` (e.g., `db_cmd.go`)
- Children: `parent_child_cmd.go` (e.g., `db_migrate_cmd.go`, `db_backup_cmd.go`)
- Consistent naming aids code navigation and maintainability

#### Multi-Level Nesting

Commands can be nested multiple levels deep. Each level initializes its package variable at declaration time:

```go
// db_cmd.go - Level 1
var dbCmd = &DbCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "db",
        Usage:       "db <subcommand>",
        Description: "Database commands",
    }),
}

func init() {
    cliutil.RegisterCommand(dbCmd)
}

// db_migrate_cmd.go - Level 2
// Initialized at package level so grandchildren can reference it
var migrateCmd = &MigrateCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "migrate",
        Usage:       "migrate <subcommand>",
        Description: "Migration commands",
    }),
}

func init() {
    cliutil.RegisterCommand(migrateCmd, dbCmd)  // Child of db
}

// db_migrate_up_cmd.go - Level 3
var upCmd = &UpCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "up",
        Usage:       "up [options]",
        Description: "Run up migrations",
    }),
}

func init() {
    cliutil.RegisterCommand(upCmd, migrateCmd)  // Child of migrate
}
```

**Usage**: `myapp db migrate up`

**File naming for deep nesting**:
- Level 1: `db_cmd.go`
- Level 2: `db_migrate_cmd.go`
- Level 3: `db_migrate_up_cmd.go`

#### Multiple Children

A parent can have multiple children. Each child references the same parent package variable:

```go
// db_cmd.go
var dbCmd = &DbCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "db",
        Usage:       "db <subcommand>",
        Description: "Database commands",
    }),
}

func init() {
    cliutil.RegisterCommand(dbCmd)
}

// db_migrate_cmd.go
var migrateCmd = &MigrateCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "migrate",
        Description: "Run migrations",
    }),
}

func init() {
    cliutil.RegisterCommand(migrateCmd, dbCmd)
}

// db_backup_cmd.go
var backupCmd = &BackupCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "backup",
        Description: "Backup database",
    }),
}

func init() {
    cliutil.RegisterCommand(backupCmd, dbCmd)
}

// db_restore_cmd.go
var restoreCmd = &RestoreCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "restore",
        Description: "Restore database",
    }),
}

func init() {
    cliutil.RegisterCommand(restoreCmd, dbCmd)
}
```

**Help output**:
```
COMMANDS:
    db [migrate, backup, restore]    Database management commands
```

#### Best Practices

**1. Parent Command Behavior**

Parents should either:
- **Show help/list subcommands** (recommended):
  ```go
  func (c *DbCmd) Handle() error {
      c.Writer.Printf("Available subcommands:\n")
      c.Writer.Printf("  migrate - Run migrations\n")
      c.Writer.Printf("  backup  - Backup database\n")
      return nil
  }
  ```

- **Delegate to a default child** (see Command Delegation section):
  ```go
  cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
      Name:       "db",
      DelegateTo: &MigrateCmd{},  // Default to migrate
  })
  ```

- **Provide standalone functionality** (less common):
  ```go
  func (c *DbCmd) Handle() error {
      // Show database status if no subcommand
      return c.showStatus()
  }
  ```

**2. Consistent Naming**

- Command type names: `ParentChildCmd` (e.g., `DbMigrateCmd`, `DbBackupCmd`)
- Package variables: `parentChildCmd` (e.g., `dbMigrateCmd`)
- File names: `parent_child_cmd.go` (e.g., `db_migrate_cmd.go`)

**3. Initialization Order**

Ensure proper initialization through dependency-based initialization:
- **Initialize parent commands as package variables** (at declaration, not in init())
- Package variables are initialized before any init() functions run
- This guarantees child commands can safely reference parents in their init()
- **Do not rely on file naming** - this is fragile and the compiler cannot warn you

```go
// CORRECT: Parent initialized at package level
var dbCmd = &DbCmd{...}  // Initialized before init() runs

func init() {
    cliutil.RegisterCommand(dbCmd)  // Just register, don't initialize
}
```

**4. Package Variables**

Use package-level variables for parent commands, initialized at declaration:
```go
// Initialize at declaration (NOT inside init())
var dbCmd = &DbCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name: "db",
        ...
    }),
}

// Grandparent commands may need exporting
var migrateCmd = &MigrateCmd{...}  // Exported if grandchildren need it
```

**5. Error Handling**

Always check registration errors:
```go
if err := cliutil.RegisterCommand(cmd); err != nil {
    panic(err)  // Registration errors are programming errors, panic is appropriate
}
```

#### Common Pitfalls

**❌ Wrong: Using full path in child Name**
```go
// WRONG - This will break!
child := &MigrateCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name: "db migrate",  // ❌ Don't include parent name
    }),
}
```

**✅ Correct: Using just the node name**
```go
// CORRECT
child := &MigrateCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name: "migrate",  // ✅ Just the child name
    }),
}
```

**❌ Wrong: Child registered before parent**
```go
// WRONG - Order matters!
cliutil.RegisterCommand(child, parent)   // ❌ parent not registered yet
cliutil.RegisterCommand(parent)
```

**✅ Correct: Parent registered first**
```go
// CORRECT
cliutil.RegisterCommand(parent)          // ✅ Register parent first
cliutil.RegisterCommand(child, parent)
```

**❌ Wrong: Initializing parent in init() function**
```go
// db_cmd.go
var dbCmd *DbCmd  // ❌ Declared but not initialized

func init() {
    // ❌ Initialized in init() - child commands might run first!
    dbCmd = &DbCmd{
        CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
            Name: "db",
            ...
        }),
    }
    cliutil.RegisterCommand(dbCmd)
}
```

**✅ Correct: Initialize parent at package level**
```go
// db_cmd.go
// ✅ Initialized at declaration time (before any init() runs)
var dbCmd = &DbCmd{
    CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
        Name: "db",
        ...
    }),
}

func init() {
    // ✅ Just register the already-initialized command
    cliutil.RegisterCommand(dbCmd)
}

// db_migrate_cmd.go
func init() {
    // ✅ Safe - dbCmd was initialized at package level before this runs
    cliutil.RegisterCommand(migrateCmd, dbCmd)
}
```

#### Help System Integration

The help system automatically understands parent-child relationships:

```bash
# Show all commands
$ myapp help
COMMANDS:
    db [migrate, backup]    Database management commands

# Show parent help
$ myapp db --help
USAGE:
    db <subcommand>

Database management commands

SUBCOMMANDS:
    migrate    Run database migrations
    backup     Backup database

# Show child help
$ myapp db migrate --help
USAGE:
    migrate [--up|--down]

Run database migrations

OPTIONS:
    --up      Run up migrations (default: true)
    --down    Run down migrations (default: false)
```

#### Testing Parent-Child Commands

```go
func TestDbMigrateCmd(t *testing.T) {
    // Setup
    writer := cliutil.NewWriter(&cliutil.WriterArgs{
        Quiet:     false,
        Verbosity: cliutil.LowVerbosity,
    })

    // Create parent
    dbCmd := &DbCmd{
        CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
            Name: "db",
        }),
    }

    // Create and register child
    migrateCmd := &MigrateCmd{
        CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
            Name: "migrate",
        }),
    }

    // Set up command context
    migrateCmd.SetCommandRunnerArgs(cliutil.CmdRunnerArgs{
        Writer: writer,
    })

    // Test
    err := migrateCmd.Handle()
    if err != nil {
        t.Fatalf("Command failed: %v", err)
    }
}
```

#### Complete Working Example

See the [squire](https://github.com/mikeschinkel/squire) project for a real-world example:
- Parent: `requires_cmd.go` - Parent command for dependency analysis
- Child: `requires_tree_cmd.go` - Tree visualization subcommand
- Demonstrates multi-file registration with package variables
- Usage: `squire requires tree`

### Command Delegation

Delegate to a default subcommand:

```go
cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
    Name:       "parent",
    DelegateTo: &DefaultChildCmd{},
    ...
})
```

### Flag Commands

Commands triggered by flags (e.g., `--version` instead of `version`):

```go
cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
    Name:     "version",
    FlagName: "version",  // Enables --version flag
    ...
})
```

### Custom Validation

Add custom validation to flags:

```go
FlagDef{
    Name:  "email",
    Usage: "Email address",
    String: &cmd.email,
    Regex: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
    ValidationFunc: func(value any) error {
        email := value.(string)
        if !strings.Contains(email, "@") {
            return fmt.Errorf("invalid email format")
        }
        return nil
    },
}
```

### Examples in Help

Add custom examples to commands:

```go
cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
    Name: "deploy",
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
    ...
})
```

### Context Support

Commands receive context for cancellation:

```go
func (c *MyCmd) Handle() error {
    ctx := c.Context

    select {
    case <-ctx.Done():
        return ctx.Err()
    case result := <-doWork(ctx):
        c.Writer.Printf("Result: %v\n", result)
    }

    return nil
}
```

## Architecture Patterns

### Two-Tier Options Pattern

Separate raw CLI parsing from typed domain options:

```go
// Tier 1: Raw CLI options (basic types)
type RawOptions struct {
    Port      int
    Host      string
    Timeout   int
    Verbosity int
    Quiet     bool
}

// Tier 2: Typed domain options
type AppOptions struct {
    *cliutil.GlobalOptions
    Port    ServerPort    // Custom validated type
    Host    Hostname      // Custom validated type
    Timeout time.Duration // Parsed duration
}

// Convert raw to typed with validation
func ParseOptions(raw *RawOptions) (*AppOptions, error) {
    globalOpts, err := cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
        Quiet:     &raw.Quiet,
        Verbosity: &raw.Verbosity,
    })
    if err != nil {
        return nil, err
    }

    port, err := ParseServerPort(raw.Port)
    if err != nil {
        return nil, err
    }

    host, err := ParseHostname(raw.Host)
    if err != nil {
        return nil, err
    }

    timeout := time.Duration(raw.Timeout) * time.Second

    return &AppOptions{
        GlobalOptions: globalOpts,
        Port:         port,
        Host:         host,
        Timeout:      timeout,
    }, nil
}
```

### Initializer Pattern

Register initialization functions to be called during setup:

```go
func init() {
    cliutil.RegisterInitializerFunc(func(args cliutil.InitializerArgs) error {
        // Initialize package-level state
        SetWriter(args.Writer)
        return nil
    })
}
```

## Real-World Examples

### Example 1: Server Application

See [`examples/basic_usage/main.go`](examples/basic_usage/main.go) for a complete example.

### Example 2: Multi-Command CLI

The [xmlui-cli](https://github.com/xmlui-org/cli) project demonstrates:
- Multiple commands (init, demo, run, mcp, version, help)
- Complex flag and argument handling
- Configuration file integration
- Subcommand hierarchies

### Example 3: Simple Server

The [xmlui-localsvr](https://github.com/xmlui-org/xmlui-localsvr) project shows:
- Simplified options-based pattern without full command registration
- Integration with standard `flag` package
- Two-tier options parsing

### Example 4: Repository Manager

The [squire](https://github.com/mikeschinkel/squire) project demonstrates:
- Production-quality command structure
- Type-safe configuration
- Structured error handling with doterr pattern
- Complex validation logic

## Integration with Other Packages

### go-dt (Domain Types)

Use domain types for type safety:

```go
import "github.com/mikeschinkel/go-dt"

type MyCmd struct {
    *cliutil.CmdBase
    inputFile  string
    outputFile string
}

func (c *MyCmd) Handle() error {
    // Convert strings to domain types
    input, err := dt.ParseFilepath(c.inputFile)
    if err != nil {
        return err
    }

    output, err := dt.ParseFilepath(c.outputFile)
    if err != nil {
        return err
    }

    // Use typed values
    exists, _ := input.Exists()
    if !exists {
        return fmt.Errorf("input file not found: %s", input)
    }

    return processFiles(input, output)
}
```

### go-cfgstore (Configuration Storage)

Create WriterLogger with file logging:

```go
import "github.com/mikeschinkel/go-cfgstore"

wl, err := cfgstore.CreateWriterLogger(&cfgstore.WriterLoggerArgs{
    Quiet:      opts.Quiet(),
    Verbosity:  opts.Verbosity(),
    ConfigSlug: "myapp",
    LogFile:    "app.log",
})
```

### go-logutil (Logging Utilities)

Initialize loggers across packages:

```go
import "github.com/mikeschinkel/go-logutil"

err := logutil.CallInitializerFuncs(logutil.InitializerArgs{
    AppInfo: appInfo,
    Logger:  logger,
})
```

## Best Practices

### 1. Use init() for Command Registration

```go
func init() {
    cmd := &MyCmd{}
    cmd.CmdBase = cliutil.NewCmdBase(...)

    if err := cliutil.RegisterCommand(cmd); err != nil {
        panic(err)
    }
}
```

### 2. Direct Field Binding

Use field pointers for automatic value binding:

```go
type MyCmd struct {
    *cliutil.CmdBase
    verbose bool
    output  string
}

FlagDefs: []cliutil.FlagDef{
    {
        Name: "verbose",
        Bool: &cmd.verbose,  // Direct binding
    },
    {
        Name:   "output",
        String: &cmd.output,  // Direct binding
    },
}
```

### 3. Structured Error Handling

Use the doterr pattern for rich error context:

```go
func (c *MyCmd) Handle() (err error) {
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

end:
    return err
}
```

### 4. Verbosity-Aware Output

Use appropriate verbosity levels:

```go
c.Writer.Printf("Operation complete\n")              // Always shown
c.Writer.V2().Printf("Processed %d items\n", count)  // Detailed
c.Writer.V3().Printf("Item details: %+v\n", item)    // Debug
```

### 5. Type-Safe Configuration

Type-assert configuration in commands:

```go
func (c *MyCmd) Handle() error {
    cfg, err := dtx.AssertType[*MyConfig](c.Config)
    if err != nil {
        return err
    }

    // Use typed config
    cfg.DoSomething()
    return nil
}
```

### 6. Context Awareness

Respect context cancellation:

```go
func (c *MyCmd) Handle() error {
    if err := c.Context.Err(); err != nil {
        return err
    }

    // Long-running operation
    return doWork(c.Context)
}
```

### 7. Consistent Exit Codes

Use standard exit codes:

```go
if err != nil {
    c.Writer.Errorf("Error: %v\n", err)
    switch {
    case errors.Is(err, ErrConfigNotFound):
        os.Exit(cliutil.ExitConfigLoadError)
    case errors.Is(err, ErrInvalidConfig):
        os.Exit(cliutil.ExitConfigParseError)
    default:
        os.Exit(cliutil.ExitUnknownRuntimeError)
    }
}
```

## Best Practices for Writing Commands

Command handlers should follow these principles for maintainability and testability:

### 1. Thin Handlers, Thick Domain Logic

Command `Handle()` functions should be **thin orchestrators** that delegate to domain packages:

**Good** (following established patterns):
- Config setup (10-20 lines)
- Single domain function call with args struct (5-10 lines)
- Output formatting (10-30 lines)
- Total: ~60-80 lines

**Bad**:
- Business logic in Handle()
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
- Flag/argument parsing
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

This keeps formatting logic with the data, not in Handle().

## User Notifications

When errors occur that require user-facing messages (not technical stack traces), use the `ErrOmitUserNotify` pattern.

### Pattern

1. **Print user-friendly message** via `Writer`
2. **Return error wrapped** with `ErrOmitUserNotify`

### Example: File Already Exists

```go
func (c *InitCmd) promptForOverwrite(conflicts []dt.RelFilepath) error {
    // 1. User-friendly explanation
    c.Writer.Printf("\nCannot initialize app files because the following files already exist:\n")
    for _, file := range conflicts {
        c.Writer.Printf("  - %s\n", file)
    }
    c.Writer.Printf("\nTo overwrite these files, use the --overwrite flag.\n")
    c.Writer.Printf("Alternatively, move or remove these files before running 'xmlui init --app'.\n")

    // 2. Return error with ErrOmitUserNotify
    return NewErr(
        cliutil.ErrOmitUserNotify,
        ErrCmd, ErrInitCommand, ErrFilesAlreadyExist,
        "files", conflicts,
    )
}
```

### How It Works

The CLI runner checks for `ErrOmitUserNotify`:

```go
err = runner.RunCmd(cmd)
if err != nil {
    if !errors.Is(err, cliutil.ErrOmitUserNotify) {
        cliutil.Printf("Command failed: %v", err)  // Only print if NOT omit
    }
    logger.Error("Run aborted", "error", err)  // Always log
    os.Exit(cliutil.ExitUnknownRuntimeError)
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
func PrepareDemo(args *PrepareDemoArgs) (*Result, error) {
    if alreadyInstalled && !args.Reinstall {
        // Print user message
        args.Writer.Errorf("Demo already installed. Use --reinstall to re-download.\n")

        // Return valid result + ErrOmitUserNotify
        return &Result{...}, NewErr(cliutil.ErrOmitUserNotify, "demo already installed")
    }
    // ...
}
```

The CLI handles this transparently - no special code needed in Handle().

## Testing

### Testing Commands

```go
func TestMyCmd(t *testing.T) {
    // Create test writer
    writer := cliutil.NewWriter(&cliutil.WriterArgs{
        Quiet:     false,
        Verbosity: cliutil.LowVerbosity,
    })

    // Create command
    cmd := &MyCmd{}
    cmd.CmdBase = cliutil.NewCmdBase(...)
    cmd.SetCommandRunnerArgs(cliutil.CmdRunnerArgs{
        Writer: writer,
    })

    // Execute command
    err := cmd.Handle()
    if err != nil {
        t.Fatalf("Command failed: %v", err)
    }
}
```

### Testing with Buffered Writer

```go
import "github.com/mikeschinkel/go-testutil"

func TestOutput(t *testing.T) {
    buf := testutil.NewBufferedWriter()
    writer := cliutil.NewWriter(&cliutil.WriterArgs{
        Quiet:     false,
        Verbosity: cliutil.LowVerbosity,
    })

    // Execute command
    cmd.Writer = writer
    cmd.Handle()

    // Check output
    output := buf.String()
    if !strings.Contains(output, "expected text") {
        t.Errorf("Expected output not found: %s", output)
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
