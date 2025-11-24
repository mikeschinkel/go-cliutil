# ADR 2025-11-24: Hidden Commands and Flag Routing

## Status

Accepted

## Context

CLI applications sometimes need commands that are:
- Callable by users but not prominently advertised in help output
- Invokable through alternative syntax for convenience (e.g., `--setup` as shorthand for `setup`)
- Available for automation scripts while keeping the primary UI simple

The motivating use case was a download script that needed to call a `setup` command, but the application maintainers didn't want end users to see this command in the help output, as it was intended for infrastructure automation rather than direct user interaction.

Prior to this ADR, the cliutil package had no mechanism to:
- Hide commands from help output while keeping them callable
- Route flag-style invocations (`--command`) to actual commands
- Validate that flag command configurations are correct at startup

## Decision

We will support **hidden commands** with optional **flag routing** through two new fields in `CmdArgs`:

### Command Configuration

```go
type CmdArgs struct {
    // ... existing fields ...
    FlagName string  // Optional: enables --flagname routing to this command
    Hide     bool    // Hide this command from help output
}
```

### Implementation Details

1. **Command Interface Extensions**
   - Added `FlagName() string` method - returns the flag name if this command supports flag routing
   - Added `IsHidden() bool` method - returns true if command should be hidden from help

2. **Flag Routing Mechanism**
   - When a command has `FlagName: "setup"`, users can invoke it with either:
     - Direct: `app setup`
     - Flag: `app --setup`
   - The flag form is transformed to the command form in `ParseCLIOptions()` BEFORE flag parsing
   - This prevents the flag from being consumed by the global flag parser
   - Transformation uses `RegisteredCommands()` to look up commands by FlagName without requiring the full command tree to be built

3. **Auto-Registration**
   - Commands with `FlagName` are automatically registered as global CLI options (Bool flags)
   - This makes them appear in the global options section of help output
   - If `Hide: true` is also set, the command doesn't appear in commands list but the flag does appear in options

4. **Help Output Filtering**
   - Hidden commands are filtered from:
     - Top-level commands list in main help
     - Example generation for help output
     - Direct help requests (`app help hidden-command` returns "unknown command")

5. **Validation**
   - At registration time (in `RegisterCommand()`):
     - FlagName must not conflict with existing global flags
     - Only top-level commands can have FlagName (subcommands cannot)
   - At command tree build time (in `ValidateCommands()`):
     - Validates subcommands don't have FlagName set

### Code Example

```go
func init() {
    err = cliutil.RegisterCommand(&SetupCmd{
        CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
            Name:        "setup",
            Usage:       "setup",
            Description: "Setup application environment",
            FlagName:    "setup",  // Enables --setup routing
            Hide:        true,      // Hidden from help
        }),
    })
    if err != nil {
        panic(err)
    }
}
```

## Consequences

### Positive

- **Flexible UX**: Commands can be discoverable or hidden based on intended audience
- **Script-Friendly**: Automation scripts can use familiar flag syntax (`--setup`) instead of command syntax
- **Validates Early**: Configuration errors are caught during initialization, not at runtime
- **Zero Breaking Changes**: Existing commands work unchanged; new fields are optional
- **Consistent with Conventions**: `--flagname` routing feels natural for Unix CLI users

### Negative

- **Two Ways to Call**: Commands with flag routing can be invoked two ways, which adds slight conceptual complexity
- **Help Asymmetry**: Flag appears in global options but command is hidden - may confuse users who discover the flag
- **Registration Overhead**: Auto-registration of flags adds a small amount of startup overhead

### Neutral

- Commands must choose between being hidden or appearing in help - there's no "semi-hidden" state
- Flag routing only works for top-level commands, not subcommands (validated and enforced)

## Alternatives Considered

### Alternative 1: Environment Variable Activation
Hidden commands only appear in help when `DEBUG=1` or similar is set.

**Rejected because**: Requires users to know about the environment variable. The goal was to keep commands entirely hidden, not conditionally visible.

### Alternative 2: Separate "Advanced" Help Section
Add `--help-advanced` flag to show hidden commands.

**Rejected because**: Still makes hidden commands discoverable, defeating the purpose. Also adds UI complexity.

### Alternative 3: Flag Routing Without Auto-Registration
Don't register flag commands as global CLI options; only handle transformation.

**Rejected because**: Flags wouldn't appear in help at all, making them completely undiscoverable. Auto-registration provides a hint in help output while keeping the command itself hidden.

### Alternative 4: Parse Flag Commands After BuildCommandTree
Wait until command tree is built to transform flag commands.

**Rejected because**: Causes flag parser to consume the flag before transformation happens, breaking the feature. Early transformation in `ParseCLIOptions()` is necessary.

## Implementation Notes

### Order of Operations

The transformation must happen BEFORE `flagset.Parse()` to prevent the flag from being consumed:

```go
// In ParseCLIOptions()
args = osArgs[1:]                      // Strip program name
args = transformFlagCommands(args)      // Transform --setup to setup
args, _ = flagset.Parse(args)          // Parse remaining flags
```

### Why RegisteredCommands() Works

We use `RegisteredCommands()` (returns the raw commands slice) rather than waiting for `BuildCommandTree()` because:
- `RegisteredCommands()` is available immediately after init() functions run
- `BuildCommandTree()` happens later in `Initialize()`, after option parsing
- We only need command name and FlagName for transformation, not the full command tree

## References

- Similar patterns in tools like `git` (e.g., `git --version` as alternative to `git version`)
- Docker's experimental commands that are hidden behind feature flags
- Kubernetes' alpha/beta commands that have limited discoverability

## Notes

This ADR was created to support the XMLUI CLI project's need for infrastructure automation commands that shouldn't clutter the user-facing help output.

Date: 2025-11-24
