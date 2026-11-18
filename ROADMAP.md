# TODO for go-cliutil

## Features

### 1. Flag-Specific Help Functions
Add support for flag-specific help rendering via a closure on `FlagDef`.

**Current limitation**: The help system cannot show flag-specific documentation (e.g., showing available column names for `--columns`). All help text is generated from the FlagSet description.

**Proposed solution**: Add an optional `HelpFunc func(Writer) error` closure to `FlagDef`. When `help <cmd> --<flag>` is invoked, call this function instead of standard help.

**Use case**: `xmlui help demo list --columns` shows only the column table, not full command help, allowing users to quickly see available options.

