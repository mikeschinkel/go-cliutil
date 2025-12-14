# Go CLI Architecture with cliutil

**READ THIS FIRST before any work on projects using cliutil and go-cliutil.**

This document defines the non-negotiable architectural principles for CLI applications built with `go-cliutil`. These principles apply to ANY project using this framework (xmlui, squire, etc.), not just one specific project.

---

## Core Principle: Thin Commands, Thick Packages

**Commands are thin shims. Business logic is in reusable packages.**

Commands (handlers in `cliutil.CommandHandler`) should do nothing except:
1. Parse and validate command-line arguments and flags
2. Load minimal configuration
3. Call a reusable package function
4. Return the result

All business logic lives in dedicated, testable packages that can be reused by other tools, libraries, and contexts.

### Command Handler Pattern (ABSOLUTE REQUIREMENT)

Every `Handle()` method must follow this structure:

```go
func (c *SomeCmd) Handle() error {
    // 1. Minimal setup (type assertion, config loading if needed)
    // 2. SINGLE call to a reusable package function
    // 3. Optionally: orchestration decisions (e.g., "should I call X after Y?")
    // 4. Return the error
}
```

**What Handle() MUST NOT do:**
- ❌ Contain filtering logic
- ❌ Contain selection/matching logic
- ❌ Contain conditional business decisions
- ❌ Contain output formatting logic
- ❌ Contain validation logic beyond basic setup
- ❌ Handle multiple business concerns
- ❌ Duplicate logic that should be in a package

**Example of CORRECT pattern:**
```go
func (c *DemoCmd) Handle() error {
    cfg, err := dtx.AssertType[*cli.Config](c.Config)
    if err != nil {
        return err
    }

    result, err := mypackage.RunDemo(&mypackage.RunDemoArgs{
        SourceArg:      *demoOpts.Repo,
        BranchArg:      *demoOpts.Branch,
        Reinstall:      *demoOpts.Reinstall,
        DryRun:         cfg.Options.DryRun(),
        Writer:         c.Writer,
    })
    if err != nil {
        return err
    }

    if !*demoOpts.NoStart {
        err = SomeOrchestratorFunc(/* ... */)
    }
    return err
}
```

**Example of WRONG pattern (don't do this):**
```go
func (c *DemoInspectCmd) Handle() error {
    // ❌ WRONG: Filtering logic in Handle()
    switch {
    case demoName != "":
        // selection logic here
    case *inspectOpts.WithErrors:
        // filtering logic here
    }

    // ❌ WRONG: Output formatting in Handle()
    if *inspectOpts.JSON {
        // output logic here
    }

    // ❌ WRONG: Validation logic in Handle()
    if len(items) == 0 {
        c.Writer.Printf("No items\n")
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

**Should this logic live in a Handle() method or a package?**

Ask these questions in order:

1. **Could another package/tool need this?** → YES: Move to a reusable package
2. **Might another package/tool need this in the future?** → YES: Move to a reusable package
3. **Is this purely CLI orchestration (decide what to call)?** → YES: Can stay in Handle()
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

### Examples of Logic That Can Stay in Handle()

- Deciding whether to start a server (orchestration)
- Type-asserting c.Config (minimal setup)
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
    Logger         slog.Logger     // ← Injected
    // ... other args
}
```

### Testing Output

Tests can capture output by injecting mock Writers and Loggers:

```go
func TestProcessItem(t *testing.T) {
    var buf bytes.Buffer
    mockWriter := &testutil.MockWriter{Output: &buf}

    result, err := mypackage.ProcessItem(&mypackage.ProcessItemArgs{
        Item:   testItem,
        Writer: mockWriter,
        Logger: testutil.NoOpLogger(),
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Assert on buf contents
    if !strings.Contains(buf.String(), "expected output") {
        t.Errorf("output missing expected text")
    }
}
```

### Important

- `Handle()` methods themselves do NOT need unit tests if they're just orchestrators (they're integration points)
- ALL business logic in packages MUST have unit tests
- Unit tests should test the function with various inputs, mocked dependencies, and edge cases
- Tests should NOT depend on CLI context, file systems, or external tools

---

## Output Formatting is Business Logic

**Output formatting belongs in packages, not Handle().**

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

Then call from Handle():

```go
func (c *ShowCmd) Handle() error {
    formatter := mypackage.NewItemFormatter(c.Writer)

    item, err := mypackage.GetItem(*itemOpts.ID)
    if err != nil {
        return err
    }

    return formatter.FormatItem(item, "text")
}
```

### Why

1. **Testable** - You can test formatting in isolation with a mock Writer
2. **Reusable** - Other tools can use the same formatter
3. **Clean Handle()** - No output logic cluttering the command
4. **Mockable** - Tests can inject a mock Writer and verify exact output
5. **Flexible** - Easy to add new formats without changing Handle()

---

## Error Handling and Context

**Provide granular error context FROM WITHIN package functions, not in Handle().**

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

Handle() receives rich error information:

```go
func (c *ShowCmd) Handle() error {
    item, err := mypackage.FindItem(*itemOpts.ID)
    if err != nil {
        return err  // mypackage already provided context
    }
    // ...
}
```

**DO NOT:**
```go
// ❌ WRONG: Wrapping errors in Handle()
item, err := mypackage.FindItem(*itemOpts.ID)
if err != nil {
    err = NewErr(cliutil.ErrOmitUserNotify, ErrItemNotFound, err)  // ← Should be in mypackage
    return err
}
```

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

// Then Handle() can type-assert if needed for specific handling
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

A reader should be able to understand WHAT a Handle() does in 30 seconds without reading implementation details:

```go
func (c *DemoCmd) Handle() error {
    cfg, err := dtx.AssertType[*cli.Config](c.Config)
    if err != nil {
        return err
    }

    result, err := minion.RunDemo(&minion.RunDemoArgs{
        SourceArg:      *demoOpts.Repo,
        BranchArg:      *demoOpts.Branch,
        Reinstall:      *demoOpts.Reinstall,
        DryRun:         cfg.Options.DryRun(),
        Writer:         c.Writer,
    })
    if err != nil {
        return err
    }

    if !*demoOpts.NoStart && !cfg.Options.DryRun() {
        err = ServeWeb(ServeWebArgs{
            Config:   cfg,
            SiteName: result.SiteName,
        })
    }

    return err
}
```

**What you understand immediately:**
- We get config
- We run the demo
- If it succeeded and `--no-start` wasn't set, we serve the web
- We return any error

**You don't need to see HOW it works.** You can read minion.RunDemo() if you need those details.

### WRONG "Readability" (Seeing All Details in One Function)

```go
func (c *DemoInspectCmd) Handle() error {
    configDir, err := cfgstore.CLIConfigDir(cli.ConfigSlug)
    if err != nil {
        goto end
    }

    demos, err := mypackage.FindDemos(&mypackage.FindDemosArgs{
        ConfigDir: configDir,
        Logger:    c.Logger,
    })
    if err != nil {
        goto end
    }

    if len(demos) == 0 {
        c.Writer.Printf("No demos installed\n")
        goto end
    }

    demoName := *inspectOpts.DemoName

    switch {
    case demoName != "":
        var matchResult *mypackage.MatchDemosResult
        matchResult, err = mypackage.MatchDemos(&mypackage.MatchDemosArgs{
            Selector:  demoName,
            ConfigDir: configDir,
            Logger:    c.Logger,
        })
        // ... 20 more lines of error handling and logic
    case *inspectOpts.WithErrors:
        // ... filtering logic
    default:
        // ... more logic
    }

    if *inspectOpts.JSON {
        c.Writer.Printf("%s\n", selectedDemos.JSON())
    } else {
        for _, demo := range selectedDemos {
            printDemoInspection(c.Writer, demo)
        }
    }

    goto end
}
```

**Problems:**
- You can't tell what this command does without reading all 90 lines
- Testing individual concerns is nearly impossible
- Other tools can't reuse the filtering, selection, or formatting logic
- Adding new modes requires modifying Handle()
- The complexity is overwhelming

### The Truth About Understanding Code

**You understand code better with abstraction, not less.**

When I write:
```go
result, err := mypackage.InspectDemos(args)
```

You immediately understand what happened. You know to look in `mypackage.InspectDemos` if you need details.

When I write 90 lines of filtering and formatting logic in Handle(), you have to read all 90 lines to understand what's happening. That's WORSE readability.

---

## Anti-Patterns (DO NOT DO THESE)

### ❌ Anti-Pattern 1: Filtering/Selection Logic in Handle()

```go
// WRONG
switch {
case demoName != "":
    matchResult, err := mypackage.MatchDemos(...)
    selectedDemos = matchResult.Matches
case *inspectOpts.WithErrors:
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

// In Handle()
selectedDemos, err := mypackage.SelectDemos(&mypackage.SelectDemosArgs{
    AllDemos: demos,
    Mode:     determineMode(*inspectOpts),
    Selector: *inspectOpts.DemoName,
})
```

**Benefits:**
- Can be unit tested independently
- Can be reused by other commands or tools
- Easy to add new selection modes
- Handle() is readable at a glance

### ❌ Anti-Pattern 2: Output Formatting in Handle()

```go
// WRONG
if *inspectOpts.JSON {
    c.Writer.Printf("%s\n", selectedDemos.JSON())
} else {
    for _, demo := range selectedDemos {
        c.Writer.Printf("Demo: %s\n", demo.FullName())
        if demo.Valid() {
            c.Writer.Printf("Status: ✓ Valid\n")
        } else {
            c.Writer.Printf("Status: ✗ Invalid\n")
            for _, errMsg := range demo.ValidationErrors() {
                c.Writer.Printf("  - %s\n", errMsg)
            }
        }
        c.Writer.Printf("Install Path: %s\n", demo.InstallPath)
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

// In Handle()
formatter := mypackage.NewDemoFormatter(c.Writer)
err := formatter.Format(selectedDemos, format)
```

**Benefits:**
- Output formatting is testable in isolation
- Easy to add new formats (CSV, XML, etc.)
- Other tools can use the same formatter
- Handle() is clean and short

### ❌ Anti-Pattern 3: Validation Logic in Handle()

```go
// WRONG
if len(demos) == 0 {
    c.Writer.Printf("No demos installed\n")
    goto end
}

if matchResult.MatchType == NoMatch {
    c.Writer.Errorf("No demo found matching: %s\n", demoName)
    err = NewErr(cliutil.ErrOmitUserNotify, ErrDemoNotFound)
    goto end
}

if matchResult.MatchType == AmbiguousMatch {
    c.Writer.Errorf("Ambiguous demo name '%s'. Matches:\n", demoName)
    for _, d := range matchResult.Matches {
        c.Writer.Errorf("  - %s\n", d.FullName())
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

// In Handle()
result, err := mypackage.InspectDemos(&mypackage.InspectDemosArgs{
    ConfigDir: configDir,
    Mode:      mode,
    Selector:  *inspectOpts.DemoName,
    Writer:    c.Writer,
    Logger:    c.Logger,
})
```

**Benefits:**
- All validation logic is in one place (the package)
- Easy to test error conditions
- Error messages are consistent
- Handle() is simple and testable

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
    Logger slog.Logger
}

// Handle() passes them
err := mypackage.ProcessItem(&mypackage.ProcessItemArgs{
    Item:   item,
    Writer: c.Writer,
    Logger: c.Logger,
})
```

**Benefits:**
- Output is capturable in tests
- You can use mock Writers/Loggers
- No hidden dependencies
- Code is more testable and composable

### ❌ Anti-Pattern 5: Duplicating Logic Across Commands

```go
// WRONG: Same filtering logic in multiple commands
// demoCmd.go
func (c *DemoCmd) Handle() error {
    demos, err := findAllDemos()
    for _, demo := range demos {
        if !demo.Valid() {
            selectedDemos = append(selectedDemos, demo)
        }
    }
}

// listCmd.go
func (c *ListCmd) Handle() error {
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

- [ ] Is Handle() a thin shim? (Usually ≤20 lines)
- [ ] Does Handle() call ONE business function (plus optional orchestration)?
- [ ] Is all filtering/selection logic in a package? ❌ NOT in Handle()
- [ ] Is all output formatting in a package? ❌ NOT in Handle()
- [ ] Is all validation/error context in a package? ❌ NOT in Handle()
- [ ] Are dependencies injected (Writer, Logger, etc.)? ✅ NOT hard-coded
- [ ] Could this logic be reused by other code? ✅ YES (it's in a package)
- [ ] Can this function be unit tested without CLI context? ✅ YES
- [ ] Is Handle() easy to understand at a glance? ✅ YES (function names tell the story)
- [ ] Am I duplicating logic across multiple commands? ❌ NO (it's in a package)

---

## Questions This Answers

**Q: Won't the package get huge?**
A: No. Packages contain subdivisions (separate functions for each concern). You test and understand them independently. That's the entire point.

**Q: What about "command-specific" behavior?**
A: "Command-specific" means: "Should I start the server after running the demo?" That's orchestration, stays in Handle(). Everything else—filtering, validation, formatting—is business logic, belongs in a package.

**Q: Why does this matter if other tools aren't using it yet?**
A: Because when they do, the code is already reusable. And the code is easier to test and understand now. Plus, libraries and other packages will want access to this functionality.

**Q: Isn't Handle() harder to understand if I have to read package functions?**
A: No. Handle() is easier to understand because it's SHORT and function names tell you what's happening. You only read package functions if you need to know HOW something works. You understand WHAT it does from the function name and signature.

**Q: Can Handle() be tested?**
A: Yes, but it's an integration test, not a unit test. Unit tests should test the package functions in isolation. Handle() is mostly glue code that doesn't need unit testing if it's thin enough.

**Q: What if I need user interaction (prompts, etc.)?**
A: Inject an interface. Define a Prompter interface in the package, implement it for CLI (using c.Reader), implement it for tests (returning fixed values). Same dependency injection pattern.
