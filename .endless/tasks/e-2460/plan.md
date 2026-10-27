# Replace go-cliutil's Makefile with a justfile

Mike has decided to move all Makefiles in his Go package repos to justfiles. go-cliutil
still has an 85-line Makefile (help, ensure-valid, test, test-unit, test-corpus, test-all,
lint, fmt, vet, tidy, build, examples, clean, ci). go-dt already made the move (its
justfile arrived with E-2431) and is the model to follow.

## Steps

1. Port every Makefile target to a just recipe with the same name and behaviour, following
   go-dt's justfile for layout, shared variables (GOEXPERIMENT=jsonv2) and the help recipe.
2. Keep golangci-lint pinned to the same version as .github/workflows/test.yml (v2.15.0
   when this was filed); say in a comment beside each pin that the two must stay in step.
3. Delete the Makefile, and update anything that names `make` (README, docs, comments).
4. Run each recipe once and confirm it does what its make target did.
