# TODO for go-cliutil

## Features

### 1. Variadic Positional Arguments
Add support for repeating/variadic positional arguments (e.g., `<dir> [<dir> ...]`).

**Current limitation**: Each `ArgDef` can only receive a single value. Commands that need to accept multiple values of the same type (like multiple directories) have no way to express this in the ArgDef system.

**Proposed solution**: Add a `Variadic bool` field to `ArgDef` that indicates the argument can receive all remaining positional arguments. The assigned value would need to be a slice type (e.g., `Strings *[]string` in addition to current `String *string`).

**Use case**: Commands like `scan <dir> [<dir> ...]` that operate on multiple directories.

### 2. Repeated Flag Support
Add support for repeated flags where each invocation merges values (e.g., `--columns=a,b --columns=c,d` becomes `[a,b,c,d]`).

**Current limitation**: Each flag can only receive a single value. Commands that accept comma-separated values cannot also support repeated flag invocations to simplify command-line usage.

**Proposed solution**: Add a `Repeated bool` field to `FlagDef` that indicates the flag can be specified multiple times. Values would be merged into a single result, supporting both comma-separated values within a single invocation AND multiple flag invocations.

**Use case**: Commands like `xmlui demo list --columns=domain,path --columns=installed,size` where users might want to compose column lists incrementally.

### 3. Flag-Specific Help Functions
Add support for flag-specific help rendering via a closure on `FlagDef`.

**Current limitation**: The help system cannot show flag-specific documentation (e.g., showing available column names for `--columns`). All help text is generated from the FlagSet description.

**Proposed solution**: Add an optional `HelpFunc func(Writer) error` closure to `FlagDef`. When `help <cmd> --<flag>` is invoked, call this function instead of standard help.

**Use case**: `xmlui help demo list --columns` shows only the column table, not full command help, allowing users to quickly see available options.

