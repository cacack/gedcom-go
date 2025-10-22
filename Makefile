# Makefile for gedcom-go
# Go genealogy library for parsing and validating GEDCOM files

.PHONY: help test test-verbose test-coverage test-short bench bench-save bench-compare perf-regression fmt vet lint clean coverage-html install-tools build tidy check all

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Coverage parameters
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html
COVERAGE_TARGET=85

# Build parameters
PACKAGES=$(shell $(GOCMD) list ./...)
TEST_PACKAGES=$(shell $(GOCMD) list ./... | grep -v /specs/)

help: ## Display this help message
	@echo "gedcom-go Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

all: clean fmt vet test build ## Run all checks and build

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	$(GOTEST) -v -race ./...

test-short: ## Run tests in short mode (skip slow tests)
	@echo "Running tests (short mode)..."
	$(GOTEST) -short ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(shell $(GOCMD) list ./... | grep -v /examples/ | grep -v /specs/)
	@echo ""
	@echo "Coverage summary (library packages only):"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total coverage: " $$3}'
	@echo ""
	@COVERAGE=$$($(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_TARGET)" | bc -l) -eq 1 ]; then \
		echo "⚠️  Warning: Coverage ($$COVERAGE%) is below target ($(COVERAGE_TARGET)%)"; \
	else \
		echo "✓ Coverage ($$COVERAGE%) meets target ($(COVERAGE_TARGET)%)"; \
	fi

coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	@echo "Open with: open $(COVERAGE_HTML)  # macOS"
	@echo "       or: xdg-open $(COVERAGE_HTML)  # Linux"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

bench-parse: ## Run parser benchmarks only
	@echo "Running parser benchmarks..."
	$(GOTEST) -bench=. -benchmem ./parser

bench-decode: ## Run decoder benchmarks only
	@echo "Running decoder benchmarks..."
	$(GOTEST) -bench=. -benchmem ./decoder

bench-encode: ## Run encoder benchmarks only
	@echo "Running encoder benchmarks..."
	$(GOTEST) -bench=. -benchmem ./encoder

bench-save: ## Save current benchmarks as baseline
	@echo "Saving benchmark baseline..."
	$(GOTEST) -bench=. -benchmem -count=5 ./parser ./decoder ./encoder ./validator > perf-baseline.txt
	@echo "✓ Baseline saved to perf-baseline.txt"

bench-compare: ## Compare current benchmarks with baseline
	@echo "Running current benchmarks..."
	$(GOTEST) -bench=. -benchmem -count=5 ./parser ./decoder ./encoder ./validator > perf-current.txt
	@echo ""
	@echo "Comparing with baseline..."
	benchstat perf-baseline.txt perf-current.txt || echo "⚠  Install benchstat: go install golang.org/x/perf/cmd/benchstat@latest"

perf-regression: ## Run performance regression tests
	@echo "Running performance regression tests..."
	@./scripts/perf-regression-test.sh

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "✓ No issues found"

lint: install-staticcheck ## Run staticcheck linter
	@echo "Running staticcheck..."
	@which staticcheck > /dev/null || (echo "staticcheck not found. Run 'make install-tools'" && exit 1)
	staticcheck ./...
	@echo "✓ No issues found"

check: fmt vet test ## Run all checks (format, vet, test)
	@echo "✓ All checks passed"

build: ## Build all packages
	@echo "Building packages..."
	$(GOBUILD) ./...
	@echo "✓ Build successful"

tidy: ## Tidy go.mod and go.sum
	@echo "Tidying go.mod..."
	$(GOMOD) tidy
	@echo "✓ go.mod tidied"

clean: ## Clean build artifacts and coverage files
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "✓ Cleaned"

install-tools: install-staticcheck ## Install development tools
	@echo "✓ All tools installed"

install-staticcheck: ## Install staticcheck linter
	@echo "Installing staticcheck..."
	@which staticcheck > /dev/null || $(GOCMD) install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "✓ staticcheck installed"

examples: ## Run all examples
	@echo "Running parse example..."
	@cd examples/parse && $(GOCMD) run main.go ../../testdata/gedcom-5.5/minimal.ged
	@echo ""
	@echo "Running validate example..."
	@cd examples/validate && $(GOCMD) run main.go ../../testdata/gedcom-5.5/minimal.ged
	@echo ""
	@echo "Running query example..."
	@cd examples/query && $(GOCMD) run main.go ../../testdata/gedcom-5.5/minimal.ged

verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GOMOD) verify
	@echo "✓ Dependencies verified"

download: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "✓ Dependencies downloaded"

deps: tidy verify download ## Update and verify dependencies

# CI targets
ci: clean fmt vet test-coverage ## Run CI checks (format, vet, test with coverage)
	@echo "✓ CI checks passed"

pre-commit: fmt vet test ## Run pre-commit checks
	@echo "✓ Pre-commit checks passed"

# Documentation targets
docs: ## Open package documentation
	@echo "Opening package documentation..."
	@echo "Visit: https://pkg.go.dev/github.com/cacack/gedcom-go"

# Development helpers
watch-test: ## Watch for changes and run tests (requires entr)
	@which entr > /dev/null || (echo "entr not found. Install with: brew install entr" && exit 1)
	@echo "Watching for changes..."
	@find . -name "*.go" | entr -c make test

# Report generation
report: test-coverage ## Generate coverage report and display statistics
	@echo ""
	@echo "=== Test Coverage Report ==="
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "=== Package Statistics ==="
	@echo "Total packages: $$(echo '$(PACKAGES)' | wc -w | tr -d ' ')"
	@echo "Go files: $$(find . -name '*.go' -not -path './vendor/*' | wc -l | tr -d ' ')"
	@echo "Lines of code: $$(find . -name '*.go' -not -path './vendor/*' -exec wc -l {} + | tail -1 | awk '{print $$1}')"
