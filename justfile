# Pinned so a new linter release can't fail a local run without a code change;
# keep in step with golangci-lint-action's `version` in .github/workflows/test.yml.
linter := "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0"

# Every directory holding a go.mod, found rather than listed, so a new module
# needs no edit here. Worktrees under .endless are other checkouts, not modules.
modules := `find . -path ./.endless -prune -o -name go.mod -print | sed 's|/go.mod$||' | sort | tr '\n' ' '`

# List recipes
help:
    @just --list

# Write a go.work over every module, so local builds use the code on disk
work:
    rm -f go.work go.work.sum
    go work init {{modules}}

# Tidy, test, lint, vet and build the examples
ensure-valid: tidy test lint vet examples

# Run unit tests
test: test-unit

# Run unit tests (main + test/)
test-unit:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running tests for main package..."
    go test -v -race -coverprofile=./coverage.txt -covermode=atomic ./...
    echo "Running tests for test sub-package..."
    (cd test && go test -v -race ./...)

# Run fuzz corpus regression tests
test-corpus:
    cd test && go test -v -run=TestFuzzCorpus

# Run all tests (unit + corpus)
test-all: test-unit test-corpus

# Run golangci-lint in every module
lint:
    #!/usr/bin/env bash
    set -euo pipefail
    for module in {{modules}}; do
        echo "Running linter for $module..."
        (cd "$module" && go run {{linter}} run ./... --timeout=5m)
    done

# Format code with gofmt
fmt:
    gofmt -s -w .

# Run go vet
vet:
    go vet ./...

# Run go mod tidy in every module, as CI sees it (without go.work)
tidy:
    #!/usr/bin/env bash
    set -euo pipefail
    for module in {{modules}}; do
        echo "Running go mod tidy for $module..."
        (cd "$module" && GOWORK=off go mod tidy)
    done

# Build the package
build:
    go build ./...

# Build all examples to ./bin/
examples:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p bin
    for example in examples/*/; do
        name=$(basename "$example")
        echo "Building $name..."
        (cd "$example" && go build -o "../../bin/$name" .)
    done
    echo "Examples built to ./bin/"

# Clean build artifacts
clean:
    go clean
    rm -f coverage.txt
    rm -rf bin
    cd test && go clean

# Run all CI checks locally (fmt, vet, lint, test-all)
ci: fmt vet lint test-all
    @echo "All CI checks passed!"
