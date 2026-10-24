# Release go-cliutil's dependency chain

go-cliutil's go.mod files reach go-dt, go-dt/dtx, go-dt/appinfo and go-testutil through
local `replace` directives, so nobody outside this machine can build a module that imports
go-cliutil at its current commit. With the replaces removed (tried 2026-10-24, after the
uncommitted main work landed):

- The root module fails: the latest published `dtx/v0.2.1` does not compile against go-dt
  v0.6.0 (`undefined: dt.ErrFileDoesNotExist`). dtx needs a new tag.
- test/ fails: it uses go-testutil's renamed BufferedWriter methods (GetPrintOutput,
  CountErrorLines, ...), which are committed after go-testutil v0.2.1 but untagged, and it
  requires go-testutil v0.3.0, which does not exist. go-testutil in turn requires
  go-cliutil v0.3.0, which does not exist either (latest tag is v0.2.1), so the two form a
  release cycle.

## Steps

1. Walk the chain from the leaves, by module rather than by repo: go-dt's root, dtx and
   appinfo, then go-testutil. Each module whose contents changed since its last tag gets its
   own version decision, and its tag is prefixed with its directory (`dtx/v0.2.2`).
2. Break the go-testutil ↔ go-cliutil cycle. go-cliutil's root module requires
   go-testutil only through terminal_test.go, so it may be possible to tag go-cliutil's root
   first, then go-testutil against it, then go-cliutil's test/ module. Confirm the order
   with Mike.
3. Show Mike each proposed tag and its reason before pushing it; tags on a public remote
   cannot be taken back cleanly.
4. Bump go-cliutil's requires to the new tags in the root, test/ and examples/basic_usage
   modules, drop the cross-project replaces, run `go mod tidy` and the tests, then commit,
   push and tag go-cliutil's root module (and test/ and examples if their contents changed).
5. Keep test/go.mod's in-repo `replace github.com/mikeschinkel/go-cliutil => ..`. It points
   the test module at the go-cliutil code beside it rather than at a published tag.
6. Check that Endless and the modules inside Endless's dependency repos still build.
