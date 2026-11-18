# Analysis: choices made in planning (Mike, 2026-11-17)

- **Flag syntax: GNU rules without bundling.** `--name` for long names, `-x` for shortcuts, no `-qf`. Users of GNU tools expect it; bundling is ambiguous once a shortcut takes a value and can be added later without breaking anything. Rejected: Go's rules (`-verbosity` works, looks foreign); GNU with bundling.
- **Multi-value flags: `Strings` target, repeat the flag, commas split only when the FlagDef sets `Split`; `\,` is a literal comma in a split flag.** Quotes cannot mark a literal comma, because the shell removes them before go-cliutil sees the word. Rejected: every list flag splits; never split.
- **Every run calls every CommandFunc.** One code path, so help and the handler always see the same values. Rejected: build the tree once and rebuild only the matched command, which lets a constructor that varies between calls make help disagree with the run.
- **Options: an app type embedding GlobalOptions (ED-1669).** Keeps compile-time checked access to every flag value, which Mike named as a key advantage of go-cliutil over Cobra. Calling an omitted flag's method panics, since it is a programming error. Rejected: a marker Options with values read by name through `args.Flag`; five methods on every app's Options returning zero for removed flags. `OptionsAs[T]()` is a generic method rather than a type assertion at each call site, which Mike prefers for its explicit typing.
- **Package-level writer removed.** Parallel `cliutiltest` runs would otherwise send output to whichever writer was set last.
- **Parser written in go-cliutil.** Go's `flag` package stops at the first non-flag and cannot express flags after operands, a boolean that never takes the next word, or passthrough.
- **ADRs are Endless decisions.** No new file in `adrs/`.

## Implementation choices (Mike, 2026-11-18)

- **Help stays in each app for this task.** Mike wants a built-in, replaceable, template-driven and themeable help, designed separately under epic E-2653. Here the examples show an app-side help command with a variadic topic operand.
- **FlagsOnlyMode runs exactly one registered command.** Every word goes to that command's flags and operand, so it uses the same Handle path as every other app. Registering zero or several commands is an error.
- **A value-taking flag takes the next word even when it starts with a dash.** `--format -v` means `--format=-v`, following GNU getopt. Rejected: an error for a missing value.
- **Initialize() stays, for the home directory only.** dt.EnsureUserHomeDir writes an unsynchronized global, so calling it on each run would race in parallel cliutiltest runs. Rejected: looking the home directory up lazily in Errorf.
