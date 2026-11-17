# Plan: rebuild go-cliutil handler input, parsing and registry

Binding decisions: ED-1647 (one positional, scalar or variadic), ED-1666 (HandleArgs),
ED-1667 (flag placement), ED-1668 (typed defaults), ED-1669 (app Options embeds
GlobalOptions), ED-1670 (config through ConfigFunc and FlagValuer), ED-1671
(CommandFunc registration), ED-1672 (cliutiltest). E-2544's outcome holds the code
shapes these came from. Backward compatibility is not a concern; this is a breaking
release.

## Approach

### Registry and command values
- New `registry.go`: `Registry` type, `NewRegistry`, `DefaultRegistry`,
  `Registry.Register(fn CommandFunc, parents ...Command)`, and package-level
  `RegisterCommand` forwarding to the default registry. Parents and `DelegateTo`
  stay prototypes identified by type.
- Each run calls every `CommandFunc`, then builds the command tree, path map, type
  map and flag-command map on a per-run value owned by the `CmdRunner`. The
  package-level `commands`, `commandsTypeMap`, `commandsPathMap` and
  `flagCommandMap` go. `Initialize`, `RegisterFlagCommands`, `ValidateCommands` and
  `BuildCommandTree` become registry or per-run steps; registration-time checks
  (one ArgDef, count fields only on a variadic, typed default matches target,
  command flag not named like a global flag, no duplicate names) run when a
  registry is validated.
- `GetExactCommand`, `GetDefaultCommand`, `GetTopLevelCmds`, `GetSubCmds`,
  `ShowMainHelp`, `ShowCmdHelp`, `BuildUsage` and `BuildCmdUsage` take the per-run
  command set instead of package maps.

### Handler input
- `commands.go`: `CommandHandler.Handle(HandleArgs) error`. `Command` loses
  `SetCommandRunnerArgs`, `ParseFlagSets` and `AssignArgs`; parsing moves out of
  commands.
- New `handle_args.go`: `HandleArgs` with Context, IO, Writer, Logger, Config,
  Options, Runner, CommandPath, CommandLine, Parameters, Flags, Operands; generic
  methods `Flag[T](name) T` and `OptionsAs[T]() T`, both panicking with a message
  naming the flag or type and the command path. Every field has a doc comment
  stating which words it holds.
- New `flags.go` content: `Flag`, `FlagSource`, `Flags` (Get, All, Passed). The old
  `Flags` marker interface and `ArgsParser` go. New `operands.go`: `Operand`,
  `Operands`, `Strings()`.
- `cmd_base.go`: `CmdBase` stops embedding `CmdRunnerArgs`; `CmdArgs` gains
  `Passthrough`; legacy `CmdArgs.FlagDefs` goes.

### Definitions
- `arg_def.go`: `Strings *[]string` target, `MinCount`, `MaxCount`,
  `DefaultString`, `DefaultStrings`; `Default any` goes.
- `flag_def.go`: `Strings *[]string` target, `Split bool`, `DefaultString`,
  `DefaultStrings`, `DefaultBool`, `DefaultInt`, `DefaultInt64`; `Default any` goes.

### Parser
- New `parser.go` replaces `FlagSet.Parse`, `classifyFlagArgs`, `Build`, `Assign`,
  `syncFlagValues`, `findBestCmdMatch`'s flag handling and `validateFlags`. It
  tokenizes with GNU rules without bundling: `--name`, `--name=value`,
  `--name value`, `-x`, `-x value`, `-x=value`; a multi-letter name after one dash
  is an unknown flag. Global flags are recognised anywhere; command words are
  matched after removing global flags and their values; command flags only after
  the matched command's last word; `--` ends flags; a boolean flag never consumes
  the next word; an unknown flag is an error; under `Passthrough` every word after
  the command words goes to Parameters unparsed. Values convert with `strconv`.
  `--help` routing and flag-routed commands (`FlagName`) keep their behaviour.
- Value resolution per flag: command line, then config (`FlagValuer`), then the
  typed default. `FlagSet` keeps its role as a named group of FlagDefs for help.

### Options and global flags
- `global_options.go`, `global_flags.go`, `options.go`: one `GlobalOptions`
  (embeddable, implements `DefaultOptions`), `GlobalOptionsArgs{Omit, Overrides}`,
  `NewGlobalOptions`, `GlobalOptions.FlagDefs()`; `Options` becomes a marker;
  `OptionsFunc func() Options`. `GlobalFlags`, `GlobalFlagsArgs`, `NewGlobalFlags`,
  `gFlags`, `gFlagSets`, `flagSet`, `options`, `AddCLIFlag`, `AddCLIOption`,
  `ParseOptions`, `ParseGlobalFlags`, `ParseGlobalFlagsFromArgs` go.
  `FlagsOnlyMode` stays for apps without subcommands.
- go-cliutil reads verbosity and quiet through `DefaultOptions` when the app's
  Options implements it, and its own defaults otherwise.

### Config
- New `config.go` content: `ConfigFunc`, `ConfigArgs`, `FlagValuer`; `Config`
  stays a marker. `CmdRunnerArgs` gains `Registry`, `Config ConfigFunc`,
  `Options OptionsFunc`, `IO`. Per run: build Options and command values, parse,
  call ConfigFunc, fill flags not typed from FlagValuer, validate, call Handle.

### Writer
- `writer.go`, `terminal.go`: `SetWriter`, `GetWriter` and the package-level
  writer go; all go-cliutil output goes through the run's Writer.

### cliutiltest
- New package `cliutiltest` in the root module: `Run(t, Args) Result`, with
  `Args{Registry, CommandLine, Config, Options, Stdin}` and
  `Result{Stdout, Stderr, ExitCode, Err}`; exit codes come from the existing
  exit-code constants. A handler asking for an undefined flag or wrong type fails
  the test with the panic message.

### Tests
- `test/` module: parser table (every placement rule, `--`, bool, passthrough,
  shortcuts, splitting and `\,`), registry validation errors, typed defaults,
  variadic counts, extra-operand error, config precedence, Omit and Overrides
  panics, parallel runs with no leaked values. Fuzz and corpus tests are rewritten
  against the new parser and options.

### Examples and docs
- `examples/basic_usage` is renamed `examples/basic` and gains commands.
- New `examples/comprehensive` shows every feature: nested subcommands, a default
  subcommand, flag routing, a hidden command, a variadic operand with counts,
  typed defaults on ArgDef and FlagDef, a split `Strings` flag, app global flags,
  Omit and Overrides, `OptionsAs`, `Flag[T]`, a `FlagValuer` config, passthrough,
  `--`, a command calling another through `Runner`, and a `cliutiltest` table
  covering its commands.
- README and `docs/BEST-PRACTICES.md` describe the new API. The request-scoped ADR
  file's status line says it is superseded by ED-1666. ROADMAP items 1 and 2 go.

### Review and release
- The session commits on the task branch. Mike reviews the code and both examples
  in the worktree before the task leaves `unverified`. After approval the task
  proposes v0.6.0 with a release description; it does not tag.
