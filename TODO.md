# TODO for go-cliutil

## Features

### 1. Variadic Positional Arguments
Add support for repeating/variadic positional arguments (e.g., `<dir> [<dir> ...]`).

**Current limitation**: Each `ArgDef` can only receive a single value. Commands that need to accept multiple values of the same type (like multiple directories) have no way to express this in the ArgDef system.

**Proposed solution**: Add a `Variadic bool` field to `ArgDef` that indicates the argument can receive all remaining positional arguments. The assigned value would need to be a slice type (e.g., `Strings *[]string` in addition to current `String *string`).

**Use case**: Commands like `scan <dir> [<dir> ...]` that operate on multiple directories.

