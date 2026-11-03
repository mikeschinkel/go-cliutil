# Plan: cliutil Writer takes its streams from a StdIO

Mike's edit, moved from the main checkout onto this branch (commit 67b95e5): `writer.go` imports go-dt, adds `IO dt.StdIO` to `WriterArgs` and `cliWriter`, and replaces every `os.Stdout` and `os.Stderr` with the StdIO streams, including the `V2()`, `V3()` and `Loud()` writers. `NewWriter(nil)` uses `dt.NewDefaultStdIO()`.

Two things on top of the moved edit:

- `NewWriter` with non-nil args and a nil `IO` dereferences nil at `args.IO.Output()`. Every existing caller passes `WriterArgs` without `IO` (go-cfgstore v0.4.0, gomion), so v0.5.1 would panic at startup for them. The fix is to default `args.IO` to `dt.NewDefaultStdIO()` when nil.
- `go.mod` requires go-dt v0.6.1, which has no StdIO; the build only works through go.work's `../go-dt`. Once go-dt v0.7.1 resolves, raise the require to v0.7.1 and update go.sum, so the Release workflow's `GOWORK=off` tests pass and ED-1639's no-replace rule holds.

## After landing (Mike)

From the main checkout, land and push main:

```
endless worktree land E-2550 && git push origin main
```

Also from the main checkout, release v0.5.1 (previous tag v0.5.0). The empty assignments make `gh` ignore the token in the environment and use the keyring PAT:

```
GITHUB_TOKEN= GH_TOKEN= gh workflow run release.yml -f version=v0.5.1 -f description='Makes `cliutil.NewWriter` write to the `dt.StdIO` given in `WriterArgs.IO` instead of `os.Stdout` and `os.Stderr`, defaulting to the process streams when none is given, so tests can capture CLI output. Raises the go-dt requirement to v0.7.1.'
```

E-2551 waits for this tag to resolve from the module proxy.
