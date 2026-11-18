# Go CLI Architecture with cliutil

**READ THIS FIRST before any work on projects using cliutil and go-cliutil.**

This document defines the non-negotiable architectural principles for CLI applications built with `go-cliutil`. These principles apply to ANY project using this framework (xmlui, gomion, etc.), not just one specific project.

---

## Core Principle: Thin Commands, Thick Packages

**Commands are thin shims. Business logic is in reusable packages.**

Commands (handlers in `cliutil.CommandHandler`) should do nothing except:
1. Declare their flags and operands
2. Read the values the run hands them through `cliutil.HandleArgs`
3. Call a reusable package function
4. Return the result

The framework parses and validates the command line and loads configuration before `Handle` runs.

All business logic lives in dedicated, testable packages that can be reused by other tools, libraries, and contexts.

### Command Handler Pattern (ABSOLUTE REQUIREMENT)

Every `Handle` method must follow this structure:

```go
func (c *SomeCmd) Handle(args cliutil.HandleArgs) (err error) {
    // 1. Minimal setup (read typed config and options from args)
    // 2. SINGLE call to a reusable package function
    // 3. Optionally: orchestration decisions (e.g., "should I call X after Y?")
    // 4. Return the error
}
```

A constructor builds the command, and each run calls that constructor, so flag and operand targets belong in fields of the value it returns:

```go
func init() {
    cliutil.RegisterCommand(NewDemoCmd)
}

type DemoCmd struct {
    *cliutil.CmdBase
    repo      string
    branch    string
    reinstall bool
    noStart   bool
}

func NewDemoCmd() cliutil.Command {
    c := &DemoCmd{}
    c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
        Name:        "demo",
        Description: "Run a demo",
        ArgDefs: []*cliutil.ArgDef{
            {Name: "repo", Usage: "Repository to run", Required: true, String: &c.repo},
        },
        FlagSets: []*cliutil.FlagSet{{
            Name: "demo",
            FlagDefs: []cliutil.FlagDef{
                {Name: "branch", Usage: "Branch to run", DefaultString: "main", String: &c.branch},
                {Name: "reinstall", Usage: "Download the demo again", Bool: &c.reinstall},
                {Name: "no-start", Usage: "Skip starting the server", Bool: &c.noStart},
            },
        }},
    })
    return c
}
```

**What `Handle` MUST NOT do:**
- ❌ Contain filtering logic
- ❌ Contain selection/matching logic
- ❌ Contain conditional business decisions
- ❌ Contain output formatting logic
- ❌ Contain validation logic beyond basic setup
- ❌ Handle multiple business concerns
- ❌ Duplicate logic that should be in a package

**Example of CORRECT pattern:**
```go
func (c *DemoCmd) Handle(args cliutil.HandleArgs) (err error) {
    var result *mypackage.RunDemoResult

    options := args.OptionsAs[*cli.Options]()

    result, err = mypackage.RunDemo(&mypackage.RunDemoArgs{
        SourceArg: c.repo,
        BranchArg: c.branch,
        Reinstall: c.reinstall,
        DryRun:    options.DryRun(),
        Writer:    args.Writer,
    })
    if err != nil {
        goto end
    }

    if !c.noStart {
        err = SomeOrchestratorFunc(result)
    }

end:
    return err
}
```

**Example of WRONG pattern (don't do this):**
```go
func (c *DemoInspectCmd) Handle(args cliutil.HandleArgs) error {
    // ❌ WRONG: Filtering logic in Handle
    switch {
    case c.demoName != "":
        // selection logic here
    case c.withErrors:
        // filtering logic here
    }

    // ❌ WRONG: Output formatting in Handle
    if c.json {
        // output logic here
    }

    // ❌ WRONG: Validation logic in Handle
    if len(items) == 0 {
        args.Writer.Printf("No items\n")
        return nil
    }

    return nil
}
```

---

## Reusability First (The Trump Card)

**If functionality might be needed by another package, another command, or another tool, it MUST be in a reusable package, not in the CLI command handler.**

### Why This Matters

1. **Avoids import cycles** - Command packages are tightly coupled to the CLI framework and hard to import elsewhere
2. **Code reuse across tools** - Logic in packages can be used by libraries, other CLIs, services, etc.
3. **Independent testing** - Package logic can be tested without CLI context
4. **Future-proofing** - When you need this logic in a new context, it's already reusable
5. **Single source of truth** - Improvements to the package benefit all consumers automatically

### Decision Tree

**Should this logic live in a `Handle` method or a package?**

Ask these questions in order:

1. **Could another package/tool need this?** → YES: Move to a reusable package
2. **Might another package/tool need this in the future?** → YES: Move to a reusable package
3. **Is this purely CLI orchestration (decide what to call)?** → YES: Can stay in `Handle`
4. **Would a colleague ever want to use this without running the CLI?** → YES: Move to a reusable package
5. **If not 1-4, does including this logic make testing harder?** → YES: Move to a reusable package

### Examples of Logic That Belongs in Packages

- Filtering, selection, matching logic
- Validation logic (beyond CLI flag parsing)
- Transformation logic
- Data processing logic
- Output formatting/rendering
- Error context and messages
- Any business logic

### Examples of Logic That Can Stay in `Handle`

- Deciding whether to start a server (orchestration)
- Reading typed config with `cliutil.ConfigAs[*cli.Config](args.Config)` (minimal setup)
- Checking a `--no-start` flag to skip a step (orchestration decision)
- Passing arguments to package functions

---

## Testability Through Dependency Injection

**Business logic must be testable in isolation without CLI context.**

### Pattern

Business logic functions accept dependencies as parameters instead of importing them:

```go
func ProcessItem(args *ProcessItemArgs) (*ProcessItemResult, error)

type ProcessItemArgs struct {
    Item           *Item
    Writer         cliutil.Writer  // ← Injected
    Logger         *slog.Logger    // ← Injected
    // ... other args
}
```

### Testing Output

Tests capture output by giving a real Writer buffers for its streams:

```go
func TestProcessItem(t *testing.T) {
    var out, errOut bytes.Buffer
    writer := cliutil.NewWriter(&cliutil.WriterArgs{
        IO:        dt.NewStdIO(&out, &errOut),
        Verbosity: cliutil.LowVerbosity,
    })

    _, err := mypackage.ProcessItem(&mypackage.ProcessItemArgs{
        Item:   testItem,
        Writer: writer,
        Logger: slog.New(slog.DiscardHandler),
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Assert on the captured output
    if !strings.Contains(out.String(), "expected output") {
        t.Errorf("output missing expected text")
    }
}
```

### Testing Commands End to End

`cliutiltest.Run` runs a whole command line in-process and returns its `Stdout`, `Stderr`, `ExitCode` and `Err`. Use it to cover what only the command line can show: flag wiring, operand counts, exit codes and help.

```go
func TestDemoCmd(t *testing.T) {
    t.Parallel()
    tests := []struct {
        name     string
        line     []string
        want     string
        exitCode int
    }{
        {name: "runs a repo", line: []string{"demo", "acme/site", "--no-start"}, want: "acme/site"},
        {name: "needs a repo", line: []string{"demo"}, exitCode: cliutil.ExitFlagsParseError},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            result := cliutiltest.Run(t, cliutiltest.Args{
                CommandLine: tt.line,
                Config:      cli.LoadConfig,
                Options:     cli.NewOptions,
            })
            if result.ExitCode != tt.exitCode {
                t.Errorf("exit code: want %d, got %d (err: %v)", tt.exitCode, result.ExitCode, result.Err)
            }
            if !strings.Contains(result.Stdout, tt.want) {
                t.Errorf("stdout: want %q in %q", tt.want, result.Stdout)
            }
        })
    }
}
```

Each call uses its own streams and command values, so parallel subtests can share a registry. A panic, such as a handler asking `args.Flag[T]` for an undefined flag, fails the test with the panic's message.

### Important

- ALL business logic in packages MUST have unit tests
- Unit tests should test the function with various inputs, injected dependencies, and edge cases
- Package tests should NOT depend on CLI context, file systems, or external tools
- A table of `cliutiltest.Run` cases covers each command's wiring; it does not replace package tests
- One test that calls `cliutil.DefaultRegistry().Validate(cli.NewOptions())` catches every command definition error

---

## Output Formatting is Business Logic

**Output formatting belongs in packages, not `Handle`.**

Output formatting includes:
- Text rendering (printf, structured output)
- JSON/XML formatting
- Table formatting
- Error messages
- Status messages

### Pattern

Create types/functions in packages for formatting:

```go
// In mypackage
type ItemFormatter struct {
    Writer cliutil.Writer
}

func (f *ItemFormatter) FormatItem(item *Item, format string) error {
    switch format {
    case "json":
        return f.formatJSON(item)
    case "text":
        return f.formatText(item)
    default:
        return fmt.Errorf("unknown format: %s", format)
    }
}

func (f *ItemFormatter) formatText(item *Item) error {
    f.Writer.Printf("Item: %s\n", item.Name)
    f.Writer.Printf("Status: %s\n", item.Status)
    return nil
}
```

Then call from `Handle`:

```go
func (c *ShowCmd) Handle(args cliutil.HandleArgs) (err error) {
    var item *mypackage.Item

    formatter := mypackage.NewItemFormatter(args.Writer)

    item, err = mypackage.GetItem(c.id)
    if err != nil {
        goto end
    }

    err = formatter.FormatItem(item, "text")

end:
    return err
}
```

### Why

1. **Testable** - You can test formatting in isolation with a Writer on buffers
2. **Reusable** - Other tools can use the same formatter
3. **Clean `Handle`** - No output logic cluttering the command
4. **Verifiable** - Tests can inject a Writer on buffers and verify exact output
5. **Flexible** - Easy to add new formats without changing `Handle`

---

## Error Handling and Context

**Provide granular error context FROM WITHIN package functions, not in `Handle`.**

Package functions should return semantic errors with context. The package author knows what went wrong better than the CLI layer.

### Pattern

Package functions should return errors with useful context:

```go
// In mypackage
func FindItem(selector string) (*Item, error) {
    items := listItems()

    matches := filterMatches(items, selector)

    switch len(matches) {
    case 0:
        return nil, fmt.Errorf("no item found matching: %q", selector)
    case 1:
        return matches[0], nil
    default:
        return nil, fmt.Errorf("ambiguous selector %q, matches:\n%s",
            selector, formatMatches(matches))
    }
}
```

`Handle` receives rich error information:

```go
func (c *ShowCmd) Handle(args cliutil.HandleArgs) (err error) {
    var item *mypackage.Item

    item, err = mypackage.FindItem(c.id)
    if err != nil {
        goto end  // mypackage already provided context
    }
    args.Writer.Printf("%s\n", item.Name)

end:
    return err
}
```

**DO NOT:**
```go
// ❌ WRONG: Wrapping errors in Handle
item, err := mypackage.FindItem(c.id)
if err != nil {
    err = NewErr(cliutil.ErrOmitUserNotify, ErrItemNotFound, err)  // ← Should be in mypackage
    return err
}
```

### User Notifications

The runner returns every error to the app's entry point without printing it. When a package has already explained a problem to the user, it wraps its error with `cliutil.ErrOmitUserNotify`, and the entry point skips the technical message:

```go
err = runner.Run(os.Args[1:])
if err != nil && !errors.Is(err, cliutil.ErrOmitUserNotify) {
    cliutil.Stderrf("Error: %v\n", err)
}
os.Exit(cliutil.ExitCode(err))
```

`cliutil.ExitCode` maps the error to the process exit code, so handlers return errors and never call `os.Exit`.

### Semantic Error Types

For complex scenarios, define error types in the package:

```go
// In mypackage
type NotFoundError struct {
    Selector string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("no item found matching: %q", e.Selector)
}

type AmbiguousError struct {
    Selector string
    Matches  []*Item
}

func (e *AmbiguousError) Error() string {
    return fmt.Sprintf("ambiguous selector %q, matches:\n%s",
        e.Selector, formatMatches(e.Matches))
}

// Then Handle can type-assert if needed for specific handling
item, err := mypackage.FindItem(selector)
if err != nil {
    if _, ok := err.(*AmbiguousError); ok {
        // Handle ambiguous case specially
    }
    return err
}
```

---

## Control Flow and Readability

**"Readability" means understanding the high-level purpose at a glance, NOT seeing all details in one function.**

### CORRECT Readability

A reader should be able to understand WHAT a `Handle` does in 30 seconds without reading implementation details:

```go
func (c *DemoCmd) Handle(args cliutil.HandleArgs) (err error) {
    var result *minion.RunDemoResult

    config := cliutil.ConfigAs[*cli.Config](args.Config)
    options := args.OptionsAs[*cli.Options]()

    result, err = minion.RunDemo(&minion.RunDemoArgs{
        SourceArg: c.repo,
        BranchArg: c.branch,
        Reinstall: c.reinstall,
        DryRun:    options.DryRun(),
        Writer:    args.Writer,
    })
    if err != nil {
        goto end
    }

    if !c.noStart && !options.DryRun() {
        err = ServeWeb(ServeWebArgs{
            Config:   config,
            SiteName: result.SiteName,
        })
    }

end:
    return err
}
```

**What you understand immediately:**
- We get config and options
- We run the demo
- If it succeeded and `--no-start` wasn't set, we serve the web
- We return any error

**You don't need to see HOW it works.** You can read minion.RunDemo() if you need those details.

### WRONG "Readability" (Seeing All Details in One Function)

```go
func (c *DemoInspectCmd) Handle(args cliutil.HandleArgs) (err error) {
    configDir, err := cfgstore.CLIConfigDir(cli.ConfigSlug)
    if err != nil {
        goto end
    }

    demos, err := mypackage.FindDemos(&mypackage.FindDemosArgs{
        ConfigDir: configDir,
        Logger:    args.Logger,
    })
    if err != nil {
        goto end
    }

    if len(demos) == 0 {
        args.Writer.Printf("No demos installed\n")
        goto end
    }

    switch {
    case c.demoName != "":
        var matchResult *mypackage.MatchDemosResult
        matchResult, err = mypackage.MatchDemos(&mypackage.MatchDemosArgs{
            Selector:  c.demoName,
            ConfigDir: configDir,
            Logger:    args.Logger,
        })
        // ... 20 more lines of error handling and logic
    case c.withErrors:
        // ... filtering logic
    default:
        // ... more logic
    }

    if c.json {
        args.Writer.Printf("%s\n", selectedDemos.JSON())
    } else {
        for _, demo := range selectedDemos {
            printDemoInspection(args.Writer, demo)
        }
    }

end:
    return err
}
```

**Problems:**
- You can't tell what this command does without reading all 90 lines
- Testing individual concerns is nearly impossible
- Other tools can't reuse the filtering, selection, or formatting logic
- Adding new modes requires modifying `Handle`
- The complexity is overwhelming

### The Truth About Understanding Code

**You understand code better with abstraction, not less.**

When I write:
```go
result, err := mypackage.InspectDemos(args)
```

You immediately understand what happened. You know to look in `mypackage.InspectDemos` if you need details.

When I write 90 lines of filtering and formatting logic in `Handle`, you have to read all 90 lines to understand what's happening. That's WORSE readability.

---

## Anti-Patterns (DO NOT DO THESE)

### ❌ Anti-Pattern 1: Filtering/Selection Logic in `Handle`

```go
// WRONG
switch {
case c.demoName != "":
    matchResult, err := mypackage.MatchDemos(...)
    selectedDemos = matchResult.Matches
case c.withErrors:
    for _, demo := range demos {
        if !demo.Valid() {
            selectedDemos = append(selectedDemos, demo)
        }
    }
}
```

**RIGHT:** Create a package function:
```go
// In mypackage
func SelectDemos(args *SelectDemosArgs) (Demos, error) {
    switch args.Mode {
    case SelectByName:
        return matchByName(args.AllDemos, args.Selector)
    case SelectWithErrors:
        return filterByErrors(args.AllDemos)
    case SelectAll:
        return args.AllDemos, nil
    }
}

// In Handle
selectedDemos, err := mypackage.SelectDemos(&mypackage.SelectDemosArgs{
    AllDemos: demos,
    Mode:     c.selectMode(),
    Selector: c.demoName,
})
```

**Benefits:**
- Can be unit tested independently
- Can be reused by other commands or tools
- Easy to add new selection modes
- `Handle` is readable at a glance

### ❌ Anti-Pattern 2: Output Formatting in `Handle`

```go
// WRONG
if c.json {
    args.Writer.Printf("%s\n", selectedDemos.JSON())
} else {
    for _, demo := range selectedDemos {
        args.Writer.Printf("Demo: %s\n", demo.FullName())
        if demo.Valid() {
            args.Writer.Printf("Status: ✓ Valid\n")
        } else {
            args.Writer.Printf("Status: ✗ Invalid\n")
            for _, errMsg := range demo.ValidationErrors() {
                args.Writer.Printf("  - %s\n", errMsg)
            }
        }
        args.Writer.Printf("Install Path: %s\n", demo.InstallPath)
    }
}
```

**RIGHT:** Create a formatter in the package:
```go
// In mypackage
type DemoFormatter struct {
    Writer cliutil.Writer
}

func (f *DemoFormatter) Format(demos Demos, format string) error {
    switch format {
    case "json":
        f.Writer.Printf("%s\n", demos.JSON())
        return nil
    case "text":
        return f.formatText(demos)
    default:
        return fmt.Errorf("unknown format: %s", format)
    }
}

func (f *DemoFormatter) formatText(demos Demos) error {
    for _, demo := range demos {
        f.formatDemo(demo)
    }
    return nil
}

func (f *DemoFormatter) formatDemo(demo *Demo) {
    f.Writer.Printf("Demo: %s\n", demo.FullName())
    if demo.Valid() {
        f.Writer.Printf("Status: ✓ Valid\n")
    } else {
        f.Writer.Printf("Status: ✗ Invalid\n")
        for _, errMsg := range demo.ValidationErrors() {
            f.Writer.Printf("  - %s\n", errMsg)
        }
    }
    f.Writer.Printf("Install Path: %s\n", demo.InstallPath)
}

// In Handle
formatter := mypackage.NewDemoFormatter(args.Writer)
err := formatter.Format(selectedDemos, format)
```

**Benefits:**
- Output formatting is testable in isolation
- Easy to add new formats (CSV, XML, etc.)
- Other tools can use the same formatter
- `Handle` is clean and short

### ❌ Anti-Pattern 3: Validation Logic in `Handle`

```go
// WRONG
if len(demos) == 0 {
    args.Writer.Printf("No demos installed\n")
    goto end
}

if matchResult.MatchType == NoMatch {
    args.Writer.Errorf("No demo found matching: %s\n", c.demoName)
    err = NewErr(cliutil.ErrOmitUserNotify, ErrDemoNotFound)
    goto end
}

if matchResult.MatchType == AmbiguousMatch {
    args.Writer.Errorf("Ambiguous demo name '%s'. Matches:\n", c.demoName)
    for _, d := range matchResult.Matches {
        args.Writer.Errorf("  - %s\n", d.FullName())
    }
    err = NewErr(cliutil.ErrOmitUserNotify, ErrAmbiguousSelector)
    goto end
}
```

**RIGHT:** Let the package handle validation:
```go
// In mypackage
func InspectDemos(args *InspectDemosArgs) (*InspectDemosResult, error) {
    demos, err := FindDemos(args.ConfigDir)
    if err != nil {
        return nil, fmt.Errorf("finding demos: %w", err)
    }

    if len(demos) == 0 {
        args.Writer.Printf("No demos installed\n")
        return &InspectDemosResult{}, nil
    }

    selectedDemos, err := SelectDemos(&SelectDemosArgs{
        AllDemos: demos,
        Mode:     args.Mode,
        Selector: args.Selector,
        Writer:   args.Writer,
    })
    if err != nil {
        return nil, err  // SelectDemos provides error context
    }

    return &InspectDemosResult{Demos: selectedDemos}, nil
}

// In Handle
result, err := mypackage.InspectDemos(&mypackage.InspectDemosArgs{
    ConfigDir: configDir,
    Mode:      mode,
    Selector:  c.demoName,
    Writer:    args.Writer,
    Logger:    args.Logger,
})
```

**Benefits:**
- All validation logic is in one place (the package)
- Easy to test error conditions
- Error messages are consistent
- `Handle` is simple and testable

### ❌ Anti-Pattern 4: Not Using Dependency Injection

```go
// WRONG: Hard-coded output/logging
fmt.Println("Processing item:", item.Name)  // Can't capture in tests
log.Printf("Error: %v", err)                // Can't test it
```

**RIGHT:** Inject dependencies
```go
// Package function signature
func ProcessItem(args *ProcessItemArgs) error {
    args.Writer.Printf("Processing item: %s\n", args.Item.Name)
    args.Logger.Info("processing item", "name", args.Item.Name)
    return nil
}

type ProcessItemArgs struct {
    Item   *Item
    Writer cliutil.Writer
    Logger *slog.Logger
}

// Handle passes them
err := mypackage.ProcessItem(&mypackage.ProcessItemArgs{
    Item:   item,
    Writer: args.Writer,
    Logger: args.Logger,
})
```

**Benefits:**
- Output is capturable in tests
- You can give tests a Writer on buffers and a discarding Logger
- No hidden dependencies
- Code is more testable and composable

### ❌ Anti-Pattern 5: Duplicating Logic Across Commands

```go
// WRONG: Same filtering logic in multiple commands
// demoCmd.go
func (c *DemoCmd) Handle(args cliutil.HandleArgs) error {
    demos, err := findAllDemos()
    for _, demo := range demos {
        if !demo.Valid() {
            selectedDemos = append(selectedDemos, demo)
        }
    }
}

// listCmd.go
func (c *ListCmd) Handle(args cliutil.HandleArgs) error {
    demos, err := findAllDemos()
    for _, demo := range demos {
        if !demo.Valid() {
            selectedDemos = append(selectedDemos, demo)
        }
    }
}
```

**RIGHT:** Extract to a package function
```go
// In mypackage
func FilterByValidity(demos Demos, valid bool) Demos {
    var result Demos
    for _, demo := range demos {
        if demo.Valid() == valid {
            result = append(result, demo)
        }
    }
    return result
}

// In both commands
selected, err := mypackage.FilterByValidity(demos, false)
```

---

## Summary Checklist

Before writing or reviewing command code, ask:

- [ ] Is `Handle` a thin shim? (Usually ≤20 lines)
- [ ] Does `Handle` call ONE business function (plus optional orchestration)?
- [ ] Is all filtering/selection logic in a package? ❌ NOT in `Handle`
- [ ] Is all output formatting in a package? ❌ NOT in `Handle`
- [ ] Is all validation/error context in a package? ❌ NOT in `Handle`
- [ ] Are dependencies injected (Writer, Logger, etc.)? ✅ NOT hard-coded
- [ ] Are flag and operand targets fields of the value the constructor returns? ✅ NOT package-level variables
- [ ] Could this logic be reused by other code? ✅ YES (it's in a package)
- [ ] Can this function be unit tested without CLI context? ✅ YES
- [ ] Is `Handle` easy to understand at a glance? ✅ YES (function names tell the story)
- [ ] Am I duplicating logic across multiple commands? ❌ NO (it's in a package)

---

## Questions This Answers

**Q: Won't the package get huge?**
A: No. Packages contain subdivisions (separate functions for each concern). You test and understand them independently. That's the entire point.

**Q: What about "command-specific" behavior?**
A: "Command-specific" means: "Should I start the server after running the demo?" That's orchestration, stays in `Handle`. Everything else—filtering, validation, formatting—is business logic, belongs in a package.

**Q: Why does this matter if other tools aren't using it yet?**
A: Because when they do, the code is already reusable. And the code is easier to test and understand now. Plus, libraries and other packages will want access to this functionality.

**Q: Isn't `Handle` harder to understand if I have to read package functions?**
A: No. `Handle` is easier to understand because it's SHORT and function names tell you what's happening. You only read package functions if you need to know HOW something works. You understand WHAT it does from the function name and signature.

**Q: Can `Handle` be tested?**
A: Yes, with `cliutiltest.Run`, which runs the whole command line in-process. That is an integration test, not a unit test. Unit tests should test the package functions in isolation, and a thin `Handle` needs only enough `cliutiltest.Run` cases to prove its wiring.

**Q: What if I need user interaction (prompts, etc.)?**
A: Inject an interface. Define a Prompter interface in the package, implement it for CLI (reading from `args.IO`), implement it for tests (returning fixed values). `cliutiltest.Args.Stdin` supplies input when a test runs the whole command. Same dependency injection pattern.
