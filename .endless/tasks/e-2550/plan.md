# Plan: cliutil Writer takes its streams from a StdIO

Mike's edit, moved from the main checkout onto this branch (commit 67b95e5): `writer.go` imports go-dt, adds `IO dt.StdIO` to `WriterArgs` and `cliWriter`, and replaces every `os.Stdout` and `os.Stderr` with the StdIO streams, including the `V2()`, `V3()` and `Loud()` writers. `NewWriter(nil)` uses `dt.NewDefaultStdIO()`.

Two things complete the moved edit:

- Mike handed the writer edit over incomplete. What remains in `writer.go`: `NewWriter` must accept non-nil args with a nil `IO` (every existing caller passes `WriterArgs` without `IO`), and the `cliWriter.io` field must be set by `NewWriter` and carried into the `V2()`, `V3()` and `Loud()` literals, which read `w.io`. The exact change, against the branch at commit 67b95e5, is the diff below; hunk headers give the line numbers.

```diff
@@ -49,6 +49,7 @@
 		goto end
 	}
 	w.v2 = &cliWriter{
+		io:        w.io,
 		writer:    w.io.Output(),
 		errWriter: w.io.ErrOut(),
 		verbosity: w.verbosity,
@@ -63,6 +64,7 @@
 		goto end
 	}
 	w.v3 = &cliWriter{
+		io:        w.io,
 		writer:    w.io.Output(),
 		errWriter: w.io.ErrOut(),
 		verbosity: w.verbosity,
@@ -77,6 +79,7 @@
 		goto end
 	}
 	w.loud = &cliWriter{
+		io:        w.io,
 		writer:    w.io.Output(),
 		errWriter: w.io.ErrOut(),
 		quiet:     false,
@@ -96,15 +99,16 @@
 //goland:noinspection GoUnusedExportedFunction
 func NewWriter(args *WriterArgs) Writer {
 	if args == nil {
-		args = &WriterArgs{
-			IO:        dt.NewDefaultStdIO(),
-			Verbosity: 1,
-		}
+		args = &WriterArgs{Verbosity: 1}
 	}
+	if args.IO == nil {
+		args.IO = dt.NewDefaultStdIO()
+	}
 	if args.Verbosity < 1 || 3 < args.Verbosity {
 		panic(fmt.Sprintf("Invalid verbosity for cliutil.Writer.SetVerbosity(); must be between 1-3; got %d", args.Verbosity))
 	}
 	return &cliWriter{
+		io:        args.IO,
 		writer:    args.IO.Output(),
 		errWriter: args.IO.ErrOut(),
 		quiet:     args.Quiet,
```

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
