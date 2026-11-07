Found while verifying E-2550 in go-cliutil on 2026-11-06.

Go finds a go.work by walking up from the current directory. Every task
worktree lives at `.endless/worktrees/e-NNNN/`, beneath the main checkout,
and the main checkout carries the go.work that `just work` writes
(`use (. test examples/basic_usage ../go-dt)`). So any go command run in a
worktree without `GOWORK=off` resolves against main's module set and fails:

    pattern ./...: directory prefix . does not contain modules listed in
    go.work or their selected dependencies

The verify runner strips the environment and gives each check a temp HOME,
and nothing sets GOWORK. A manifest check in the recommended form,
`runner = "gotest"` with `paths = ["./..."]`, has the runner compose a plain
`go test -json ./...`, so there is nowhere to add GOWORK=off. The check exits
1 with no test failures and empty stderr, and the runner reports "check
exited non-zero but reported no test failures", which does not point at the
cause. The E-2430 and E-2460 suites passed in that form only because main
had no go.work when they ran. E-2550's suite works around it with the gotest
driver's literal `command = "GOWORK=off go test -json ./..."`.

Options, smallest first:

1. The verify runner exports GOWORK=off for every check. A suite proves the
   worktree's tree, never a workspace, so this is always right, and it fixes
   every Go repo at once. Fix lives in endless.
2. Endless writes a go.work per worktree (or runs `just work` there) on
   worktree creation. Keeps local builds in a worktree reaching sibling
   repos, but each repo needs its own hook and the suite still proves a
   workspace rather than what a consumer sees (ED-1639).
3. Document the hazard and leave it to suite authors. Cheapest, and the
   failure mode stays confusing.

Recommendation: option 1, and in addition make the runner's "non-zero but no
failures" message include the captured stdout tail so a build-output line
like the one above is visible.
