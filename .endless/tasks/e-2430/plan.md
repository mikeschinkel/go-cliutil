# Point go-cliutil/test at go-testutil v0.3.0

Releases run module by module: go-cliutil's root module was tagged v0.4.0 first (v0.3.0 was
released 2025-12-05; NewGlobalOptions no longer returning an error made the next version
breaking), then go-testutil v0.3.0 was tagged requiring go-cliutil v0.4.0 (2026-11-01), and
only now can go-cliutil's test/ module require a go-testutil tag that has the BufferedWriter
API its tests use (GetPrintOutput, CountErrorLines, ...). Until then test/go.mod required
go-testutil v0.3.0 before it existed, and resolved it through a replace pointing at the
go-testutil checkout.

## Steps

1. In test/go.mod, require the published go-testutil v0.3.0 and go-cliutil v0.4.0, and
   drop the go-testutil replace. (Its go-dt and go-dt/dtx replaces were already dropped
   in E-2374.)
2. Keep `replace github.com/mikeschinkel/go-cliutil => ..`, so the test module tests the
   go-cliutil code beside it rather than a published tag.
3. Run `go mod tidy` and the tests with GOWORK=off.
4. Re-enable the fuzz corpus step that .github/workflows/test.yml and release.yml skip
   with `if: ${{ false }}`.
5. Do not release anything. When the work is ready, tell Mike it is ready and give him,
   for each module whose contents changed since its last tag, a proposed version with the
   reason for the bump and a release description he can paste as-is: active voice, with
   backticks around every code identifier, file name and command. Mike runs the Release
   workflow and makes the tags. (test/ itself has never been tagged and is a test module,
   which the generated Release workflow never offers, so the proposal covers root only if
   root changed.)
