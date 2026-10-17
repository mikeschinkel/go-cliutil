# Plan: Move RegisterCommand's validation into Initialize

## Decisions (2026-10-17)
- Q1: `RegisterCommand(cmd Command, parents ...Command)` returns nothing. It only appends the command, records its type in `commandsTypeMap`, and records parents. init() registration cannot fail.
- Q2: The flag-command step is a new exported function, called from `Initialize` before `ValidateCommands`. Add a comment saying we plan to unexport it, `ValidateCommands` and `BuildCommandTree` later. No known caller uses them directly.
- Q3: Settled by moving the step into `Initialize`. The new step reports one error per conflict, not two. A FlagName that conflicts with a global flag is a developer bug and cannot be reconciled automatically, so `Initialize` fails with an error carrying `command_name` and `flag_name`. Because every init() has finished by the time `Initialize` runs, the old order dependence is gone.
- Q4: Change go-cliutil only. Downstream repos that call `RegisterCommand` are fixed separately, each through a task in its own project that this task blocks.
- Q5: Work in this task's worktree. The uncommitted changes in the main checkout are handled by a separate cleanup task. None of them touch `commands.go`.

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
6. Verify that downstream repos which don't call `RegisterCommand` still build against the worktree, using a scratch GOWORK.
7. Downstream caller tasks are already filed and linked as blocked by this task.

## Out of scope
- A generic error-reconciliation hook. File it separately if needed.
