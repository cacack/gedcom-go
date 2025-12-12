# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`go-gedcom` is a pure Go library for processing GEDCOM files. GEDCOM (GEnealogical Data COMmunication) is a standard file format for exchanging genealogical data between different genealogy software.

## Development Environment Setup

### Go Installation

**Required Version**: Go 1.21 or later

**Check current version**:
```bash
go version                       # Should show go1.21.x or higher
```

**Installation methods**:

1. **Official installer** (recommended for simplicity):
   - Download from https://go.dev/dl/
   - Follow platform-specific instructions
   - Verify with `go version`

2. **Version managers** (recommended for managing multiple Go versions):
   - **asdf**: `asdf plugin add golang && asdf install golang 1.21.0`
   - **gvm**: `gvm install go1.21.0 && gvm use go1.21.0`
   - **Homebrew (macOS)**: `brew install go@1.21`

3. **Verify Go environment**:
   ```bash
   go env GOPATH                 # Should show Go workspace path
   go env GOROOT                 # Should show Go installation path
   ```

### Project Initialization

**First-time setup**:
```bash
# Clone repository (if not already done)
git clone <repo-url>
cd go-gedcom

# Download dependencies (will read go.mod)
go mod download

# Verify everything works
go test ./...
```

**Module management**:
```bash
go mod init github.com/yourorg/go-gedcom  # Already done, don't re-run
go mod tidy                      # Clean up dependencies
go mod verify                    # Verify dependency checksums
go mod vendor                    # (Optional) vendor dependencies locally
```

### Development Tools

**Essential tools**:

1. **gopls** (Go language server):
   ```bash
   go install golang.org/x/tools/gopls@latest
   ```
   - Provides IDE features (autocomplete, go-to-definition, etc.)
   - Automatically used by VS Code, vim-go, etc.

2. **staticcheck** (advanced linter):
   ```bash
   go install honnef.co/go/tools/cmd/staticcheck@latest
   staticcheck ./...
   ```

3. **golangci-lint** (meta-linter, optional but recommended):
   ```bash
   # Install via script (Linux/macOS)
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

   # Or via Homebrew (macOS)
   brew install golangci-lint

   # Run all linters
   golangci-lint run ./...
   ```

**Useful tools**:
```bash
go install golang.org/x/tools/cmd/goimports@latest  # Auto-format imports
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Editor/IDE Setup

**Visual Studio Code** (recommended):
1. Install Go extension: `ms-vscode.go`
2. Extension will prompt to install tools (gopls, dlv, etc.) - accept
3. Recommended settings (`.vscode/settings.json`):
   ```json
   {
     "go.useLanguageServer": true,
     "go.lintTool": "golangci-lint",
     "go.lintOnSave": "workspace",
     "go.formatTool": "goimports",
     "go.testOnSave": false,
     "go.coverOnSave": false,
     "editor.formatOnSave": true,
     "[go]": {
       "editor.codeActionsOnSave": {
         "source.organizeImports": true
       }
     }
   }
   ```

**Vim/Neovim**:
- Install `vim-go` plugin: https://github.com/fatih/vim-go
- Run `:GoInstallBinaries` to install tools
- Ensure gopls is configured as LSP

**GoLand/IntelliJ IDEA**:
- Native Go support built-in
- Enable "Go Modules" in settings
- Configure file watchers for `go fmt` on save

### Testing Setup

**Test data organization**:
```bash
testdata/
├── gedcom-5.5/           # GEDCOM 5.5 sample files
│   ├── minimal.ged       # Minimal valid file
│   ├── royal92.ged       # Complex real-world example
│   └── ...
├── gedcom-5.5.1/         # GEDCOM 5.5.1 samples
│   └── ...
├── gedcom-7.0/           # GEDCOM 7.0 samples
│   └── ...
└── malformed/            # Invalid files for error testing
    ├── invalid-xref.ged
    ├── missing-header.ged
    └── ...
```

**Running tests**:
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -run TestParseLine ./parser

# Run tests in short mode (skip slow integration tests)
go test -short ./...

# Run benchmarks
go test -bench=. ./...
go test -bench=BenchmarkParseLarge -benchmem ./parser
```

**Coverage requirements**:
- Minimum: 85% coverage (enforced by constitution)
- Check with: `go test -cover ./... | grep coverage`
- Target: >90% for critical packages (parser, decoder, validator)

### Pre-commit Hooks (Optional)

**Setup git hooks** to ensure quality before commit:

Create `.git/hooks/pre-commit`:
```bash
#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Format code
echo "→ Running go fmt..."
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
  echo "Error: The following files are not formatted:"
  echo "$UNFORMATTED"
  exit 1
fi

# Run vet
echo "→ Running go vet..."
go vet ./...

# Run tests
echo "→ Running tests..."
go test ./...

# Check coverage
echo "→ Checking test coverage..."
COVERAGE=$(go test -cover ./... | grep coverage | awk '{print $5}' | tr -d '%')
if (( $(echo "$COVERAGE < 85.0" | bc -l) )); then
  echo "Error: Test coverage ($COVERAGE%) is below 85%"
  exit 1
fi

echo "✓ Pre-commit checks passed"
```

Make executable:
```bash
chmod +x .git/hooks/pre-commit
```

### Performance Profiling

**CPU profiling**:
```bash
# Generate CPU profile during benchmark
go test -bench=BenchmarkParseLarge -cpuprofile=cpu.prof ./parser

# Analyze profile interactively
go tool pprof cpu.prof
# Commands: top, list, web

# Generate flame graph (requires graphviz)
go tool pprof -http=:8080 cpu.prof
```

**Memory profiling**:
```bash
# Generate memory profile
go test -bench=BenchmarkParseLarge -memprofile=mem.prof ./parser

# Analyze allocations
go tool pprof -alloc_space mem.prof
```

**Profile-Guided Optimization** (PGO, Go 1.21+):
```bash
# 1. Collect profile during benchmark
go test -bench=. -cpuprofile=default.pgo ./...

# 2. Build with PGO
go build -pgo=auto

# Performance improvement: typically 10-20% for hot paths
```

### Debugging

**Delve debugger**:
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a test
dlv test ./parser -- -test.run TestParseLine

# Debug with breakpoint
# (Add `runtime.Breakpoint()` in code, or use `break` command in dlv)
```

**VS Code debugging**:
- Use built-in debugger with Go extension
- Set breakpoints by clicking line numbers
- Press F5 to start debugging tests

### Troubleshooting

**Common issues**:

1. **"go: cannot find main module"**:
   - Ensure you're in project root (where `go.mod` exists)
   - Run `go mod init` if `go.mod` is missing

2. **"undefined: slices.Contains"**:
   - Go version too old, need Go 1.21+
   - Check with `go version`

3. **"gopls not found"**:
   - Install with `go install golang.org/x/tools/gopls@latest`
   - Ensure `$(go env GOPATH)/bin` is in PATH

4. **Tests failing with "file not found"**:
   - Tests expect to be run from package directory
   - Use `go test ./...` from project root

5. **Import cycle errors**:
   - Check for circular dependencies between packages
   - Use `internal/` packages to break cycles if needed

## Development Commands

### Testing
```bash
go test ./...                    # Run all tests
go test -v ./...                 # Run tests with verbose output
go test -run TestName ./...      # Run specific test
go test -cover ./...             # Run tests with coverage
go test -bench=. ./...           # Run benchmarks
```

### Building
```bash
go build ./...                   # Build all packages
go mod tidy                      # Clean up dependencies
go mod download                  # Download dependencies
```

### Code Quality
```bash
go fmt ./...                     # Format code
go vet ./...                     # Run static analysis
golint ./...                     # Run linter (if installed)
staticcheck ./...                # Run staticcheck (if installed)
```

## Architecture Guidelines

### GEDCOM Format Basics
- GEDCOM files use a line-based format with hierarchical levels (0, 1, 2, etc.)
- Each line format: `LEVEL TAG [VALUE] [XREF]`
- Main record types: Individual (INDI), Family (FAM), Source (SOUR), Repository (REPO), etc.
- Records are linked via cross-references (e.g., `@I1@`)

### Expected Code Structure
The library should be organized around:

1. **Parser/Lexer**: Low-level tokenization and parsing of GEDCOM line format
2. **Data Model**: Go structs representing GEDCOM records (Individual, Family, Event, etc.)
3. **Decoder**: High-level API to decode GEDCOM files into Go data structures
4. **Encoder**: Convert Go data structures back to GEDCOM format
5. **Validator**: Validate GEDCOM data against specification versions (5.5, 5.5.1, 7.0)

### Key Design Considerations
- Handle large GEDCOM files efficiently (streaming where possible)
- Support multiple GEDCOM versions (5.5, 5.5.1, 7.0)
- Character encoding handling (GEDCOM supports ANSEL, ASCII, UNICODE, etc.)
- Preserve original line numbers for error reporting
- Handle malformed GEDCOM files gracefully with clear error messages

### Standard Go Library Structure
Follow standard Go project layout:
- Root: library code (parser, types, decoder, encoder)
- `internal/`: private implementation details
- `examples/`: example usage code
- `testdata/`: sample GEDCOM files for testing

## Downstream Consumer

This library is used by `github.com/cacack/my-family` (at `/Users/chris/devel/home/my-family`) via a `replace` directive during development. When adding features:

1. **Driven by real usage**: Features should be added when my-family needs them, not speculatively
2. **Self-contained commits**: Each enhancement should be a complete, testable unit with its own tests
3. **Run consumer tests**: After changes, verify my-family still works: `cd /Users/chris/devel/home/my-family && go test ./...`
4. **API stability**: Consider how changes affect the public API; prefer additive changes over breaking ones
5. **Document in commit**: Note which my-family feature drove the change (e.g., "feat(decoder): add entity parsing for GEDCOM import")
