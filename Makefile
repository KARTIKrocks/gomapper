GOLANGCI_LINT_VERSION := v2.13.0
GOIMPORTS_VERSION := v0.49.0
GOVULNCHECK_VERSION := v1.7.0

.PHONY: all setup deps tidy tidy-check test test-v vet lint lint-fix fix vuln print-govulncheck-version print-golangci-lint-version bench build fmt cover clean ci

all: fmt vet lint test build

## Install development tools (skips if already present)
setup:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}
	@command -v goimports >/dev/null 2>&1 || { \
		echo "Installing goimports $(GOIMPORTS_VERSION)..."; \
		go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION); \
	}
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "Installing govulncheck $(GOVULNCHECK_VERSION)..."; \
		go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION); \
	}

## Download module dependencies
deps:
	go mod download

## Tidy go.mod/go.sum
tidy:
	go mod tidy

## Fail if go.mod/go.sum is not tidy, without leaving the change behind.
## Suitable for CI, where a stale go.sum should block the merge.
tidy-check:
	@status=$$(git status --porcelain -- go.mod go.sum); \
	if [ -n "$$status" ]; then \
		echo "go.mod/go.sum already modified; commit or stash before running tidy-check"; \
		exit 1; \
	fi
	@$(MAKE) --no-print-directory tidy
	@if ! git diff --quiet -- go.mod go.sum; then \
		echo "go.mod/go.sum are not tidy — run 'make tidy' and commit:"; \
		git diff --stat -- go.mod go.sum; \
		git checkout -- go.mod go.sum; \
		exit 1; \
	fi
	@echo "go.mod/go.sum tidy"

## Run all tests with race detector
test:
	go test -race -count=1 ./...

## Run tests with verbose output
test-v:
	go test -race -v -count=1 ./...

## Format code
fmt:
	gofmt -w .
	goimports -w .

## Run go vet
vet:
	go vet ./...

## Run golangci-lint
lint: setup
	golangci-lint run ./...

## Run golangci-lint with auto-fix
lint-fix: setup
	golangci-lint run --fix ./...

## Fix code formatting and linting issues
fix: fmt lint-fix

## Scan for known vulnerabilities in the module's dependencies, filtered to
## advisories this code actually reaches.
##
## Needs network access — the advisory database is fetched on every run.
##
## Note this also scans the standard library of whichever Go toolchain you have
## installed, so it can fail locally on a green branch when your Go is a patch
## release behind the one CI pins. That is a real finding about your machine,
## not a false positive.
vuln: setup
	govulncheck ./...

## Print the pinned scanner version. CI installs govulncheck with this rather
## than hardcoding a second copy of the number, so the workflow and this file
## cannot drift apart.
print-govulncheck-version:
	@echo $(GOVULNCHECK_VERSION)

## Print the pinned linter version. CI resolves golangci-lint-action's version
## input from this rather than hardcoding a second copy of the number, so the
## workflow and this file cannot drift apart.
print-golangci-lint-version:
	@echo $(GOLANGCI_LINT_VERSION)

## Run benchmarks
bench:
	go test -bench=. -benchmem ./...

## Build all packages
build:
	go build ./...

## Run tests with coverage report
cover:
	go test -race ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Remove build artifacts
clean:
	rm -f coverage.out coverage.html

## CI pipeline: tidy, vet, lint, test, vulnerability-scan
ci: tidy-check vet lint test vuln
