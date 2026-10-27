# Point go-cliutil/test at go-testutil v0.3.0

Releases run module by module: go-cliutil's root module is tagged v0.4.0 first (v0.3.0 was
released 2025-12-05; NewGlobalOptions no longer returning an error makes the next version
breaking), then go-testutil v0.3.0 is tagged requiring go-cliutil v0.4.0, and only then can
go-cliutil's test/ module require a go-testutil tag that has the BufferedWriter API its
tests use (GetPrintOutput, CountErrorLines, ...). Until then test/go.mod requires
go-testutil v0.3.0, which does not exist yet, and resolves it through a replace pointing at
the go-testutil checkout.

## Steps

1. In test/go.mod, require the published go-testutil v0.3.0 (or whatever it was tagged)
   and go-cliutil v0.4.0, and drop the go-testutil replace. (Its go-dt and go-dt/dtx
   replaces were already dropped in E-2374.)
2. Keep `replace github.com/mikeschinkel/go-cliutil => ..`, so the test module tests the
   go-cliutil code beside it rather than a published tag.
3. Run `go mod tidy` and the tests with GOWORK=off.
4. Re-enable the fuzz corpus step that .github/workflows/test.yml and release.yml skip
   with `if: ${{ false }}`.
5. If test/'s contents changed since its last tag, tell Mike which `test/vX.Y.Z` tag to
   make and why; Mike does the tagging.
