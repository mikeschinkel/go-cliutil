# Plan: Move RegisterCommand's validation into Initialize

## Decisions (Mike, 2026-10-17)
- Q1: `RegisterCommand(cmd Command, parents ...Command)` returns nothing. It only appends the command, records its type in `commandsTypeMap`, and records parents. init() registration cannot fail.
- Q2: The flag-command step is a new exported function, called from `Initialize` before `ValidateCommands`. Add a comment saying we plan to unexport it, `ValidateCommands` and `BuildCommandTree` later. Only Mike's apps use them.
- Q3: Settled by moving the step into `Initialize`. The new step reports one error per conflict, not two. A FlagName that conflicts with a global flag is a developer bug and cannot be reconciled automatically, so `Initialize` fails with an error carrying `command_name` and `flag_name`. Because every init() has finished by the time `Initialize` runs, the old order dependence is gone.
- Q4: Change go-cliutil only. Endless and its dependencies don't call `RegisterCommand`. The callers are left broken, and each gets a task in its own project explaining the fix.
- Q5: Work in the E-2289 worktree. Mike's uncommitted go-cliutil changes are handled by a separate cleanup task (park on a temp branch, analyze, file tasks). None of them touch `commands.go`.

## Steps
1. `commands.go`: strip `RegisterCommand` down to append, type-map and parents.
2. Add `RegisterFlagCommands() error`, with the unexport-later comment. For each command with a `FlagName`, call `AddCLIOption` (a Bool flag with usage "Run <name> command"). Wrap each error with `ErrCommandRegistrationFailed` and `command_name`, and combine them all.
3. `Initialize`: SetWriter, then `RegisterFlagCommands`, then `ValidateCommands`, then `BuildCommandTree`. Add the same unexport-later comment to the other two.
4. Tests in `test/`:
   - FlagName equal to a built-in global flag (`quiet`) gives exactly one error, naming the command and the flag.
   - Two commands with the same FlagName give one error, for the second command.
   - A command FlagName plus an app `AddCLIOption` of the same name, added in either order before `Initialize`, gives one error from `Initialize` that names the command.
   - A FlagName command shows up in global help and routes as before.
5. Fix the in-package callers (examples/, test/) so go-cliutil builds and its tests pass.
6. Verify that Endless still builds against the worktree using a scratch GOWORK: `go build ./...` in the endless repo.
7. File one task per caller project. Each says: `RegisterCommand` no longer returns an error, so drop the error check around every `cliutil.RegisterCommand(` call. Each is blocked by E-2289 where cross-project blocking works. The projects are gomion, xmlui-cli, doctool, go-stash, macmail-ext, gmover and scout-mcp. gearboxworks/homelab waits on Mike's choice of how to register it.

## Out of scope
- A generic error-reconciliation hook (Mike's post-process de-duplication idea). File it separately if wanted.
