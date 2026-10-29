#!/usr/bin/env bash
#
# E-2289's suite, run by `endless task verify E-2289` through verify.toml.
# Emits TAP on stdout; diagnostics go to stderr.
#
# It proves the tree this suite sits in: every module builds and passes,
# RegisterCommand returns nothing, Initialize reports each FlagName conflict
# once whatever the registration order, the README matches the new signature,
# and Endless and the modules in its dependency repos build against it.

set -u

: "${ENDLESS_VERIFY_DIR:?run this through: endless task verify E-2289}"

WT="$(cd "${ENDLESS_VERIFY_DIR}/../../.." && pwd)"
MAIN="$(cd "$(git -C "${WT}" rev-parse --git-common-dir)/.." && pwd)"
PKGS="$(dirname "${MAIN}")"
PROJECTS="$(dirname "${PKGS}")"
SCRATCH="$(mktemp -d)"
trap 'rm -rf "${SCRATCH}"' EXIT

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

cat > "${SCRATCH}/test.work" <<WORK
go 1.25.3
use ${WT}/test
replace github.com/mikeschinkel/go-testutil => ${PKGS}/go-testutil
WORK

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

# --- The flag-command scenarios, each run in its own process ----------------
for s in builtin-conflict two-commands option-then-command command-then-option help-and-routing; do
    # -v and the PASS line: a -run pattern that matches nothing also exits 0.
    check "TestFlagCommands/${s} runs and passes" bash -c "
        cd '${WT}/test' &&
        GOWORK='${SCRATCH}/test.work' go test -count=1 -v -run '^TestFlagCommands\$/^${s}\$' ./... |
        grep -q -- '--- PASS: TestFlagCommands/${s} '"
done

for m in . test examples/basic_usage; do
    check "go vet is clean in module ${m}" in_module "${m}" go vet ./...
done
check "golangci-lint v2.15.0 reports no issues in the root module" \
    in_module . go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...
check "gofmt reports no unformatted files" test -z "$(gofmt -l "${WT}" 2>/dev/null | grep -v '/\.endless/')"

# --- RegisterCommand and Initialize ----------------------------------------
check "RegisterCommand returns nothing" \
    grep -q '^func RegisterCommand(cmd Command, parents \.\.\.Command) {$' "${WT}/commands.go"
check "RegisterFlagCommands is exported and returns an error" \
    grep -q '^func RegisterFlagCommands() (err error) {$' "${WT}/commands.go"
check "Initialize calls RegisterFlagCommands before ValidateCommands" bash -c "
    awk '/^func Initialize\\(/,/^}/' '${WT}/commands.go' |
    grep -nE 'RegisterFlagCommands\\(\\)|ValidateCommands\\(\\)' |
    head -1 | grep -q RegisterFlagCommands"
check "RegisterFlagCommands, ValidateCommands and BuildCommandTree carry the unexport TODO" \
    test "$(grep -c 'TODO: We plan to unexport' "${WT}/commands.go")" -eq 3
refute "no in-repo Go code checks RegisterCommand's result" \
    grep -rnE --include='*.go' --exclude-dir=.endless '(err|_) :?= (cliutil\.)?RegisterCommand\(' "${WT}"

# --- README ------------------------------------------------------------------
refute "the README no longer checks RegisterCommand's result" \
    grep -q 'err := cliutil.RegisterCommand' "${WT}/README.md"
check "the README says Initialize reports FlagName conflicts" \
    grep -q 'reported by `cliutil.Initialize`' "${WT}/README.md"

# --- Dependents build against this branch ----------------------------------
for dep in "${PROJECTS}/endless" "${PKGS}/go-cfgstore" "${PKGS}/go-logutil" \
           "${PKGS}/go-logutil/test" "${PKGS}/go-testutil" "${PKGS}/go-testutil/test"; do
    rm -f "${SCRATCH}/dep.work"
    (cd "${SCRATCH}" && GOWORK="${SCRATCH}/dep.work" go work init "${dep}" &&
        GOWORK="${SCRATCH}/dep.work" go work edit -replace "github.com/mikeschinkel/go-cliutil=${WT}")
    check "${dep#"${PROJECTS}/"} builds against this branch" \
        bash -c "cd '${dep}' && GOWORK='${SCRATCH}/dep.work' GOFLAGS= go build ./..."
done

echo "1..${N}"
(( FAILED == 0 ))
