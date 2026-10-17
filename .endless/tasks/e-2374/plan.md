# Plan: Park uncommitted main changes and file tasks to land them

The main checkout of go-cliutil has uncommitted work, staged and unstaged, that predates this task. Other repos may build against that working tree through local `replace` directives. The goal is a clean main, with every change either landed through a task or deliberately dropped by the maintainer.

The changes at filing time (2026-10-17), excluding `.endless/`:
- Staged deletions: the whole `climenu/` package (README.md, go.mod, go.sum, menu.go, modal_menu.go, mode_manager.go) and the `examples/modal_demo/modal_demo` binary.
- Modified: app.go, doterr.go (the go-doterr lint fix, synced; go-doterr has its own task for it), global_options.go, stdio.go, terminal.go, go.mod, go.sum, examples/basic_usage/*, examples/modal_demo/{go.mod,main.go}, test/{buffered_writer_test.go,corpus_test.go,fuzz_test.go,go.mod,go.sum}.
- `commands.go` is not touched; a separate task changes it.

## Steps
1. **Snapshot without touching main.** From this task's worktree, read the main checkout's full diff against HEAD, staged and unstaged together, including deletions, binaries and untracked non-`.endless/` files. Recreate it on a branch named `wip/<task-id>` cut from main's HEAD, and commit it there as one commit with message "WIP snapshot of main working tree". Then confirm the branch's tree matches main's working tree file for file. Don't use `git stash`; the stash stack is shared across sessions.
2. **Analyze.** Go through the snapshot and work out why each change was made, from the code, nearby commits and other repos that depend on it. Group the changes into coherent units, such as "API rename following go-dt v0.5.0", "new feature X" or "dependency bumps". Flag anything that looks unfinished, accidental, or like a generated or binary artifact.
3. **Report to the user before filing.** Show the grouping, what each group is for, and anything you couldn't explain, then let them adjust it.
4. **File one task per agreed group** in this project. Each plan names the files and hunks it takes from `wip/<task-id>`.
5. **Check whether dependents rely on the WIP.** Find sibling repos whose go.mod has a `replace` pointing at this checkout. Build each one against this repo at main HEAD, without the WIP, using a scratch GOWORK that points this module at a clean checkout. Add blockers for whatever fails: each group whose removal breaks the build gets `--blocks` on the tasks that need it.
6. **Hand the user one command to clear main's working tree.** They run it themselves, only after confirming `wip/<task-id>` holds everything. This session doesn't write to the main checkout.

## Done when
Every change in the snapshot belongs to a filed task or is listed as "drop" with the user's agreement, main is clean, and any build dependency is recorded as a blocker.
