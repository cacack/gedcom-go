# Contributing to go-gedcom

Thank you for your interest in contributing to go-gedcom! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Submitting Changes](#submitting-changes)
- [Reporting Issues](#reporting-issues)

## Code of Conduct

This project adheres to a simple code of conduct:

- Be respectful and inclusive
- Provide constructive feedback
- Focus on what is best for the community
- Show empathy towards other community members

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git
- Familiarity with the GEDCOM specification (helpful but not required)

### Setup Development Environment

1. **Fork and clone the repository**
   ```bash
   git fork https://github.com/cacack/gedcom-go
   cd gedcom-go
   ```

2. **Install dependencies**
   ```bash
   go mod download
   # Or use the Makefile
   make download
   ```

3. **Verify everything works**
   ```bash
   go test ./...
   # Or use the Makefile
   make test
   ```

4. **Install recommended development tools** (optional but recommended)
   ```bash
   # Using the Makefile (recommended)
   make install-tools

   # Or install manually
   go install golang.org/x/tools/cmd/gopls@latest
   go install honnef.co/go/tools/cmd/staticcheck@latest
   go install golang.org/x/tools/cmd/goimports@latest
   ```

For detailed setup instructions, see [CLAUDE.md](CLAUDE.md).

## Development Workflow

### 1. Create a Branch

Create a descriptive branch name:

```bash
git checkout -b feature/add-date-parsing
git checkout -b fix/invalid-utf8-handling
git checkout -b docs/improve-examples
```

### 2. Make Your Changes

- Write clear, idiomatic Go code
- Follow the project's code style (see [Code Standards](#code-standards))
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

**Using Makefile (recommended):**

```bash
# Run all tests
make test

# Check test coverage (requires 85%+)
make test-coverage

# Run tests with race detector
make test-verbose

# Run benchmarks
make bench
```

**Using Go commands directly:**

```bash
# Run all tests
go test ./...

# Check test coverage
go test -cover ./...

# Run specific package tests
go test ./parser -v

# Run with race detector
go test -race ./...
```

### 4. Format and Lint

**Using Makefile (recommended):**

```bash
# Format code
make fmt

# Run vet
make vet

# Run staticcheck linter
make lint

# Run all checks (fmt, vet, test)
make check
```

**Using Go commands directly:**

```bash
# Format code
go fmt ./...

# Run vet
go vet ./...

# Run staticcheck (if installed)
staticcheck ./...
```

### 5. Commit Your Changes

Write clear, descriptive commit messages:

```bash
git add .
git commit -m "feat: add support for GEDCOM 7.0 dates"
git commit -m "fix: handle invalid UTF-8 sequences correctly"
git commit -m "docs: add example for parsing large files"
```

Commit message guidelines:
- Use present tense ("add feature" not "added feature")
- Use imperative mood ("move cursor to..." not "moves cursor to...")
- Limit first line to 72 characters
- Reference issues and pull requests when relevant

### 6. Push and Create Pull Request

```bash
git push origin feature/your-branch-name
```

Then create a pull request on GitHub.

## Code Standards

### Go Style Guide

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go) guidelines.

### Project-Specific Guidelines

1. **Package Organization**
   - Keep packages focused and cohesive
   - Avoid circular dependencies
   - Use `internal/` for private implementation details

2. **Naming Conventions**
   - Use clear, descriptive names
   - Follow Go naming conventions (MixedCaps for exported, mixedCaps for unexported)
   - Avoid abbreviations unless well-known (e.g., `ID`, `URL`, `UTF8`)

3. **Documentation**
   - All exported types, functions, and constants must have godoc comments
   - Package-level documentation should explain the package's purpose
   - Include usage examples in godoc format where helpful

4. **Error Handling**
   - Return errors, don't panic (except for truly exceptional situations)
   - Provide context in error messages
   - Use custom error types when appropriate
   - Include line numbers for parsing/validation errors

5. **Code Organization**
   ```go
   // Good: Clear structure
   package parser

   import (
       "fmt"
       "io"
   )

   // Line represents a GEDCOM line with level, tag, and value.
   type Line struct {
       Level int
       Tag   string
       Value string
   }

   // ParseLine parses a single GEDCOM line.
   func ParseLine(s string) (*Line, error) {
       // Implementation
   }
   ```

## Testing Requirements

### Coverage Requirements

- **Minimum**: 85% coverage for all packages
- **Target**: 90%+ coverage for core packages (parser, decoder, validator)
- **Examples**: 0% is acceptable (examples are for demonstration)

Check coverage:
```bash
go test -cover ./...
```

### Test Guidelines

1. **Table-Driven Tests**
   - Use table-driven tests for multiple test cases
   - Include edge cases and error conditions

   ```go
   func TestParseLine(t *testing.T) {
       tests := []struct {
           name    string
           input   string
           want    *Line
           wantErr bool
       }{
           {"valid line", "0 HEAD", &Line{Level: 0, Tag: "HEAD"}, false},
           {"invalid level", "X HEAD", nil, true},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got, err := ParseLine(tt.input)
               if (err != nil) != tt.wantErr {
                   t.Errorf("ParseLine() error = %v, wantErr %v", err, tt.wantErr)
                   return
               }
               // Compare got with tt.want
           })
       }
   }
   ```

2. **Test Organization**
   - One test file per source file (`foo.go` → `foo_test.go`)
   - Use descriptive test names
   - Test both success and failure cases
   - Include edge cases (empty input, nil values, boundary conditions)

3. **Benchmark Tests**
   - Add benchmarks for performance-critical code
   - Name benchmarks with `Benchmark` prefix

   ```go
   func BenchmarkParseLine(b *testing.B) {
       input := "0 @I1@ INDI"
       for i := 0; i < b.N; i++ {
           ParseLine(input)
       }
   }
   ```

4. **Integration Tests**
   - Place in `*_test.go` files with `// +build integration` tag
   - Test with real GEDCOM files from `testdata/`

## Submitting Changes

### Pull Request Process

1. **Before Submitting**
   - Ensure all tests pass: `go test ./...`
   - Check code coverage meets requirements
   - Format code: `go fmt ./...`
   - Run linters: `go vet ./...`
   - Update documentation if needed
   - Add/update tests for your changes

2. **Pull Request Description**
   - Clearly describe what changes you made and why
   - Reference any related issues
   - Include screenshots for UI changes (if applicable)
   - List any breaking changes

3. **Example PR Template**
   ```markdown
   ## Description
   Adds support for parsing GEDCOM 7.0 date formats including age calculations.

   ## Changes
   - Added `ParseDate7()` function in `parser/date.go`
   - Added comprehensive tests with 95% coverage
   - Updated documentation with examples

   ## Related Issues
   Fixes #42

   ## Breaking Changes
   None

   ## Checklist
   - [x] Tests pass locally
   - [x] Code coverage ≥85%
   - [x] Code formatted with `go fmt`
   - [x] Documentation updated
   - [x] No linter warnings
   ```

4. **Review Process**
   - Maintainers will review your PR
   - Address any feedback or requested changes
   - Once approved, a maintainer will merge your PR

### What to Expect

- Initial response within 3-5 business days
- Constructive feedback to help improve your contribution
- Possible requests for changes or clarifications
- Merged PRs will be included in the next release

## Reporting Issues

### Before Creating an Issue

1. **Search existing issues** - Your issue may already be reported
2. **Check documentation** - The answer might be in the docs
3. **Try the latest version** - The issue may already be fixed

### Creating a Good Issue

Include the following information:

1. **Bug Reports**
   ```markdown
   **Description**
   Clear description of the bug

   **To Reproduce**
   Steps to reproduce the behavior:
   1. Parse file 'example.ged'
   2. Call GetIndividual("@I1@")
   3. See error

   **Expected Behavior**
   What you expected to happen

   **Actual Behavior**
   What actually happened

   **Environment**
   - Go version: 1.21.0
   - OS: macOS 14.0
   - go-gedcom version: v0.1.0

   **Sample GEDCOM**
   Attach a minimal GEDCOM file that reproduces the issue
   ```

2. **Feature Requests**
   ```markdown
   **Problem**
   Describe the problem you're trying to solve

   **Proposed Solution**
   Describe how you'd like it to work

   **Alternatives Considered**
   Other approaches you've thought about

   **Additional Context**
   Any other relevant information
   ```

3. **Questions**
   - Use GitHub Discussions for general questions
   - Use Issues for bug reports and feature requests

## Additional Resources

- [GEDCOM 5.5 Specification](https://www.gedcom.org/gedcom.html)
- [GEDCOM 5.5.1 Specification](https://www.familysearch.org/developers/docs/gedcom/)
- [GEDCOM 7.0 Specification](https://gedcom.io/specifications/FamilySearchGEDCOMv7.html)
- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)

## Questions?

If you have questions about contributing:

- Open a GitHub Discussion
- Check existing documentation
- Review closed issues and PRs for similar questions

Thank you for contributing to go-gedcom! 🎉
