#!/usr/bin/env bash
#
# E-2374's suite, run by `endless task verify E-2374` through verify.toml.
# Emits TAP on stdout; diagnostics go to stderr.
#
# It proves the tree this suite sits in: the four landed groups build and pass
# in every module, the folded fixes hold, the dependents build against it, and
# wip/E-2374 still holds main's uncommitted work until main is cleared.

set -u

: "${ENDLESS_VERIFY_DIR:?run this through: endless task verify E-2374}"

WT="$(cd "${ENDLESS_VERIFY_DIR}/../../.." && pwd)"
MAIN="$(cd "$(git -C "${WT}" rev-parse --git-common-dir)/.." && pwd)"
PKGS="$(dirname "${MAIN}")"
PROJECTS="$(dirname "${PKGS}")"
SCRATCH="$(mktemp -d)"
trap 'rm -rf "${SCRATCH}"' EXIT

export GOEXPERIMENT=jsonv2
export GOFLAGS=-mod=readonly

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

# The worktree's own modules, with go-cliutil's sibling replaces pointed at
# the real checkouts beside the main checkout.
cat > "${SCRATCH}/cliutil.work" <<EOF
go 1.25.3
use ${WT}
use ${WT}/test
use ${WT}/examples/basic_usage
replace github.com/mikeschinkel/go-dt => ${PKGS}/go-dt
replace github.com/mikeschinkel/go-dt/dtx => ${PKGS}/go-dt/dtx
replace github.com/mikeschinkel/go-dt/appinfo => ${PKGS}/go-dt/appinfo
replace github.com/mikeschinkel/go-testutil => ${PKGS}/go-testutil
EOF

in_module() { (cd "${WT}/$1" && shift && GOWORK="${SCRATCH}/cliutil.work" "$@"); }

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
check "gofmt reports no unformatted files" test -z "$(gofmt -l "${WT}" 2>/dev/null | grep -v '/\.endless/')"

# --- G1: go-dt v0.6.0 and the go-testutil renames, with folded fixes --------
refute "no code calls the removed dt.LogOnError" grep -rn --include='*.go' 'dt\.LogOnError' "${WT}" --exclude-dir=.endless
for m in . test examples/basic_usage; do
    check "module ${m} requires go-dt v0.6.0, the first tag with LogOnErr" \
        grep -Eq 'github.com/mikeschinkel/go-dt v0\.6\.0' "${WT}/${m}/go.mod"
done
check "test/go.mod replaces go-cliutil with .." \
    grep -Eq '^replace github.com/mikeschinkel/go-cliutil => \.\.$' "${WT}/test/go.mod"
check "the concurrent-access stderr check counts error lines" \
    grep -q 'CountErrorLines() == 0' "${WT}/test/buffered_writer_test.go"

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

# --- P1: wip/E-2374 still holds main's uncommitted work --------------------
# Holds until main's working tree is cleared after landing.
check "wip/E-2374's tree matches main's working tree, .endless/ aside" \
    git -C "${MAIN}" diff --quiet wip/E-2374 -- . ':(exclude).endless'
check "main has no untracked files outside .endless/ that wip/E-2374 lacks" \
    test -z "$(git -C "${MAIN}" ls-files --others --exclude-standard -- . ':(exclude).endless')"

echo "1..${N}"
(( FAILED == 0 ))
