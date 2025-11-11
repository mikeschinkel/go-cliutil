# Request-Scoped Command Pattern with Injected Context

**Date:** 2025-11-07
**Status:** Accepted
**Authors:** Mike Schinkel

## Context

The `go-cliutil` package provides a command framework where commands implement the `CommandHandler` interface:

```go
type CommandHandler interface {
    Handle() error
    SetCommandRunnerArgs(CmdRunnerArgs)
}
```

Commands need access to several runtime dependencies:
- **Logger** - for structured logging
- **Writer** - for output (stdout, stderr, or test buffers)
- **AppInfo** - application metadata (name, version, etc.)
- **Context** - for cancellation, timeouts, and request-scoped values
- **Config** - parsed configuration from config files
- **Options** - parsed command-line flags
- **Args** - command-line arguments (os.Args[1:])

The design question: How should these dependencies be provided to command implementations?

## Decision

We treat commands as **request-scoped containers** and inject **all** runtime dependencies (context, config, options, args, logger, writer, appinfo) as properties via `SetCommandRunnerArgs()` before calling `Handle()`. The `Handle()` method takes **no parameters** - all dependencies are accessed as properties. Context is stored in the command struct and accessed via a `Context()` method, following the `http.Request` pattern.

### Implementation

**CmdRunnerArgs contains all runtime dependencies:**
```go
type CmdRunnerArgs struct {
    Context context.Context
    AppInfo appinfo.AppInfo
    Logger  *slog.Logger
    Writer  Writer
    Config  cliutil.Config
    Options cliutil.Options
    Args    []string  // os.Args[1:]
}
```

**Commands expose context via method:**
```go
type CmdBase struct {
    ctx     context.Context
    Logger  *slog.Logger
    Writer  Writer
    AppInfo appinfo.AppInfo
    Config  Config
    Options Options
    Args    []string
    // ... other fields
}

func (c *CmdBase) Context() context.Context {
    return c.ctx
}

func (c *CmdBase) SetCommandRunnerArgs(args CmdRunnerArgs) {
    c.ctx = args.Context
    c.Logger = args.Logger
    c.Writer = args.Writer
    c.AppInfo = args.AppInfo
    c.Config = args.Config
    c.Options = args.Options
    c.Args = args.Args
}
```

**Handle signature simplified:**
```go
// Before: context, config, and args as parameters
Handle(ctx context.Context, config Config, args []string) error

// After: all dependencies injected via SetCommandRunnerArgs
Handle() error
```

**Caller creates and owns context:**
```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle signals gracefully
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
        <-sigChan
        cancel() // Propagates to all Handle() methods
    }()

    err := Run(ctx, &RunArgs{
        AppInfo: appInfo,
        Logger:  logger,
        Writer:  writer,
        Config:  config,
        CLIArgs: os.Args,
    })
}
```

## Rationale

### 1. Consistency in Dependency Access

All runtime dependencies are accessed the same way:
- `c.Writer.Printf(...)` - accessed as property
- `c.Logger.Info(...)` - accessed as property
- `c.AppInfo.Version` - accessed as property
- `c.Context()` - accessed via method
- `c.Config` - accessed as property
- `c.Options` - accessed as property
- `c.Args` - accessed as property

All dependencies are injected via `SetCommandRunnerArgs()`, creating a uniform access pattern.

### 2. Following http.Request Pattern

The Go standard library's `http.Request` stores context in the struct and exposes it via `.Context()`:

```go
type Request struct {
    ctx context.Context
    // ... other fields
}

func (r *Request) Context() context.Context {
    return r.ctx
}
```

This is considered correct because:
- Request is a request-scoped container
- Context lifetime matches request lifetime
- No concurrent access to the same request
- Context flows from caller (server creates it)

Our commands have identical characteristics:
- Commands are request-scoped (mutated via SetCommandRunnerArgs before each Handle call)
- Context lifetime matches command execution
- CLI processes one command at a time (no concurrency)
- Context flows from caller (main creates it)

### 3. Simplified Handle Signature

All dependencies are now injected as properties, eliminating parameter lists entirely:

```go
// Before: parameters required
func (c *VersionCmd) Handle(ctx context.Context, config Config, args []string) error {
    c.Writer.Printf("Version %s\n", c.AppInfo.Version)
    return nil
}

// After: all dependencies as properties
func (c *VersionCmd) Handle() error {
    c.Writer.Printf("Version %s\n", c.AppInfo.Version)
    return nil
}

// Access config, args, and context as properties
func (c *SlowCmd) Handle() error {
    if len(c.Args) > 0 {
        return doWork(c.Context(), c.Writer, c.Config)
    }
    return nil
}
```

### 4. Commands Already Treated as Request-Scoped

The architecture already treats commands as request-scoped containers:
- `SetCommandRunnerArgs()` mutates the command before Handle
- Each command execution gets fresh dependencies injected
- No reuse of command instances across requests

Storing context in the struct aligns with this existing pattern.

### 5. Caller-Created Context Enables Signal Handling

Having the CLI main function create the context allows:
- Graceful shutdown on SIGINT/SIGTERM
- Application-level timeout policies
- Context cancellation propagating to all commands
- Test-specific contexts (timeouts, cancellation triggers)

```go
// In tests
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()
err := runner.RunCmd(ctx, cmd, config) // Fails if command takes too long
```

## Consequences

### Positive

- **Consistency**: All runtime dependencies accessed uniformly
- **Simplicity**: Most Handle signatures don't need context parameter
- **Testability**: Callers can inject test-specific contexts
- **Signal handling**: Graceful shutdown via context cancellation
- **Clear ownership**: Caller controls context lifecycle

### Negative

- **Violates idiom**: "Context as first parameter" is standard Go practice
- **Unfamiliar pattern**: Developers expect context as parameter
- **Extra method call**: `c.Context()` to access, vs direct `ctx` parameter
- **Documentation needed**: Must explain why we diverge from idiom

### Mitigation

- This ADR documents the decision and rationale
- The http.Request precedent provides standard library justification
- Code comments reference this pattern at SetCommandRunnerArgs

## Alternatives Considered

### Alternative 1: Keep Parameters in Handle Signature (Idiomatic)

```go
Handle(ctx context.Context, config Config, args []string) error
```

**Pros:**
- More familiar to Go developers
- Explicit in signature what the method needs

**Cons:**
- Inconsistent with Writer/Logger/AppInfo access pattern
- Mixed pattern: some dependencies as properties, others as parameters
- Context/config/args noise in every Handle signature

**Rejected because:** Uniform dependency injection provides better consistency. All dependencies should flow through `SetCommandRunnerArgs()`.

### Alternative 2: Create Context in RunCmd

```go
func (cr CmdRunner) RunCmd(cmd Command, config Config) error {
    ctx := context.Background() // Create here
    handler.SetCommandRunnerArgs(CmdRunnerArgs{
        Context: ctx,
        Logger:  cr.Args.Logger,
        Writer:  cr.Args.Writer,
        AppInfo: cr.Args.AppInfo,
    })
    return handler.Handle(config, cr.osArgs)
}
```

**Pros:**
- Simpler caller interface

**Cons:**
- Loses signal handling capability
- Loses timeout control
- Loses test control over cancellation
- Breaks "context flows from above" principle

**Rejected because:** Caller needs control over context for signal handling and testing.

## References

- [Go Context Package](https://pkg.go.dev/context)
- [net/http Request.Context()](https://pkg.go.dev/net/http#Request.Context)
- [Context Best Practices](https://go.dev/blog/context)

## Migration Impact

This is a breaking change to the `CommandHandler` interface:

**Before:**
```go
func (c *MyCmd) Handle(ctx context.Context, config Config, args []string) error {
    // Use parameters
}
```

**After:**
```go
func (c *MyCmd) Handle() error {
    // Access via properties: c.Context(), c.Config, c.Args, etc.
}
```

All existing commands must update their Handle signatures to take no parameters and access all dependencies as properties. Since the package is in early development with no backward compatibility guarantees, this change is acceptable.
