# Replace go-cliutil's Makefile with a justfile

Mike has decided to move all Makefiles in his Go package repos to justfiles. go-cliutil
still has an 83-line Makefile (help, ensure-valid, test, test-unit, test-corpus, test-all,
lint, fmt, vet, tidy, build, examples, clean, ci). go-dt already made the move (its
justfile arrived with E-2431) and is the model to follow.

E-2430 (landed 2026-11) already removed `GOEXPERIMENT=jsonv2` from the Makefile and the
workflows and made release.yml's commit list exclude `.endless`, so neither needs doing here.

## Steps

1. Port every Makefile target to a just recipe with the same name and behaviour, following
   go-dt's justfile for layout, the `modules` variable (every go.mod found, not listed) and
   the help recipe. go-cliutil has no nested consumer modules, so it needs no `nested`
   variable; its modules are the root, `test/` and `examples/basic_usage`.
2. Make `lint` and `tidy` loop over every module as go-dt's do, with `tidy` running
   `GOWORK=off` so it sees what CI sees. The Makefile linted and vetted only the root.
3. Keep golangci-lint pinned to the same version as .github/workflows/test.yml (v2.15.0
   when this was filed); say in a comment beside each pin that the two must stay in step,
   and change test.yml's comment, which names "LINTER in the Makefile", to name the justfile.
4. Delete the Makefile, and update anything that names `make` (README, docs, comments).
5. Run each recipe once and confirm it does what its make target did.
