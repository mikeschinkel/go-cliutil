#!/usr/bin/env bash
#
# E-2374's suite, run by `endless task verify E-2374` through verify.toml.
# Emits TAP on stdout; diagnostics go to stderr.
#
# It proves the tree this suite sits in: every module builds and passes, the
# root and examples build from published go-dt tags, the folded fixes hold,
# Endless and the modules in its dependency repos build against it, and
# wip/E-2374 holds the snapshot of main's uncommitted work.

set -u

: "${ENDLESS_VERIFY_DIR:?run this through: endless task verify E-2374}"

WT="$(cd "${ENDLESS_VERIFY_DIR}/../../.." && pwd)"
MAIN="$(cd "$(git -C "${WT}" rev-parse --git-common-dir)/.." && pwd)"
PKGS="$(dirname "${MAIN}")"
PROJECTS="$(dirname "${PKGS}")"
SCRATCH="$(mktemp -d)"
trap 'rm -rf "${SCRATCH}"' EXIT

export GOEXPERIMENT=jsonv2

N=0
FAILED=0
ok() { N=$((N + 1)); echo "ok ${N} - $1"; }
not_ok() {
    N=$((N + 1)); FAILED=$((FAILED + 1))
    echo "not ok ${N} - $1"
    [[ -n "${2:-}" ]] && printf '%s\n' "$2" | head -20 | sed 's/^/# /'
}
# check <description> <command...>: passes when the command exits 0.
check() {
    local desc="$1" out
    shift
    if out="$("$@" 2>&1)"; then ok "${desc}"; else not_ok "${desc}" "${out}"; fi
}
# refute <description> <command...>: passes when the command exits non-zero.
refute() {
    local desc="$1" out
    shift
    if out="$("$@" 2>&1)"; then not_ok "${desc}" "${out}"; else ok "${desc}"; fi
}

# The root and examples/basic_usage modules build from published tags alone.
# test/ still reaches go-testutil through a sibling-checkout replace until
# go-testutil v0.3.0 is published (E-2430), and from a worktree that relative
# path resolves to nothing, so test/ runs through a go.work that points it at
# the real checkout.
cat > "${SCRATCH}/test.work" <<EOF
go 1.25.3
use ${WT}/test
replace github.com/mikeschinkel/go-testutil => ${PKGS}/go-testutil
EOF

in_module() {
    local m="$1"
    shift
    if [[ "${m}" == test ]]; then
        (cd "${WT}/test" && GOWORK="${SCRATCH}/test.work" "$@")
    else
        (cd "${WT}/${m}" && GOWORK=off "$@")
    fi
}

# --- Fail fast: the modules' own tests -------------------------------------
for m in . test examples/basic_usage; do
    check "go test -race passes in module ${m}" in_module "${m}" go test -race ./...
done
if (( FAILED > 0 )); then
    echo "1..${N}"
    exit 1
fi

for m in . test examples/basic_usage; do
    check "go vet is clean in module ${m}" in_module "${m}" go vet ./...
done
check "golangci-lint v2.15.0 reports no issues in the root module" \
    in_module . go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...
check "the Makefile and test.yml pin the same golangci-lint, v2.15.0" bash -c "
    grep -q 'golangci-lint@v2.15.0\"' '${WT}/Makefile' &&
    grep -q 'version: v2.15.0' '${WT}/.github/workflows/test.yml'"
check "gofmt reports no unformatted files" test -z "$(gofmt -l "${WT}" 2>/dev/null | grep -v '/\.endless/')"

# --- Built from published tags ---------------------------------------------
refute "the root module has no replace directive" grep -q '^replace' "${WT}/go.mod"
check "examples/basic_usage replaces only go-cliutil, with ../.." \
    test "$(grep '^replace' "${WT}/examples/basic_usage/go.mod")" = 'replace github.com/mikeschinkel/go-cliutil => ../..'
refute "test/go.mod no longer replaces go-dt or go-dt/dtx" \
    grep -Eq '^replace github.com/mikeschinkel/go-dt' "${WT}/test/go.mod"
for m in . test examples/basic_usage; do
    check "module ${m} requires go-dt v0.6.1" \
        grep -Eq 'github.com/mikeschinkel/go-dt v0\.6\.1' "${WT}/${m}/go.mod"
    check "module ${m} requires dtx v0.2.2" \
        grep -Eq 'github.com/mikeschinkel/go-dt/dtx v0\.2\.2' "${WT}/${m}/go.mod"
    check "module ${m} requires appinfo v0.3.0" \
        grep -Eq 'github.com/mikeschinkel/go-dt/appinfo v0\.3\.0' "${WT}/${m}/go.mod"
done

# --- go-testutil: only test/ uses it ----------------------------------------
refute "the root module does not require go-testutil" grep -q 'go-testutil' "${WT}/go.mod"
check "terminal_test.go lives in the test module" test -f "${WT}/test/terminal_test.go"
refute "no terminal_test.go remains in the root module" test -e "${WT}/terminal_test.go"
# corpus_step_skipped <workflow file>: the corpus step's next line disables it.
corpus_step_skipped() {
    grep -A1 'name: Run fuzz corpus regression tests' "$1" | grep -Fq 'if: ${{ false }}'
}
for wf in test release; do
    check "${wf}.yml skips the test/ corpus step until go-testutil v0.3.0 exists" \
        corpus_step_skipped "${WT}/.github/workflows/${wf}.yml"
done

# --- G1: LogOnErr and the go-testutil renames, with folded fixes -----------
refute "no code calls the removed dt.LogOnError" \
    grep -rn --include='*.go' --exclude-dir=.endless 'dt\.LogOnError' "${WT}"
check "test/go.mod replaces go-cliutil with .." \
    grep -Eq '^replace github.com/mikeschinkel/go-cliutil => \.\.$' "${WT}/test/go.mod"
check "the concurrent-access stderr check counts error lines" \
    grep -q 'CountErrorLines() == 0' "${WT}/test/buffered_writer_test.go"

# --- Exit code spelling, fixed before v0.4.0 ------------------------------
refute "no code uses the misspelled ExitInitializeationError" \
    grep -rn --include='*.go' --exclude-dir=.endless 'ExitInitializeationError' "${WT}"
check "exit_codes.go defines ExitInitializationError" \
    grep -Eq '^\s*ExitInitializationError\s*=' "${WT}/exit_codes.go"

# --- G3: NewGlobalOptions returns no error ---------------------------------
check "NewGlobalOptions returns *GlobalOptions only" \
    grep -q '^func NewGlobalOptions(args GlobalOptionsArgs) \*GlobalOptions {' "${WT}/global_options.go"

# --- G4: climenu and modal_demo are gone -----------------------------------
refute "climenu/ is removed" test -e "${WT}/climenu"
refute "examples/modal_demo/ is removed" test -e "${WT}/examples/modal_demo"
refute "nothing in the repo still names climenu or modal_demo" \
    grep -rln --exclude-dir=.endless --exclude-dir=.git -e climenu -e modal_demo "${WT}"

# --- P5: dependents build against this branch ------------------------------
for dep in "${PROJECTS}/endless" "${PKGS}/go-testutil" "${PKGS}/go-testutil/test" \
           "${PKGS}/go-doterr/cmd/sync-doterr"; do
    printf 'go 1.26.2\nuse %s\nreplace github.com/mikeschinkel/go-cliutil => %s\n' \
        "${dep}" "${WT}" > "${SCRATCH}/dep.work"
    check "${dep#"${PROJECTS}/"} builds against this branch" \
        bash -c "cd '${dep}' && GOWORK='${SCRATCH}/dep.work' GOFLAGS= go build ./..."
done

# --- P1: wip/E-2374 holds the snapshot main's cleanup relies on ---------------
# Main's own go.mod edits made after the snapshot are already in this branch,
# so the snapshot is not compared to main's current working tree.
check "wip/E-2374 exists and holds the WIP snapshot commit" \
    bash -c "git -C '${MAIN}' log -1 --format=%s wip/E-2374 | grep -qx 'WIP snapshot of main working tree'"

echo "1..${N}"
(( FAILED == 0 ))
