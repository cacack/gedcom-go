# gedcom-go

[![CI](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml/badge.svg)](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-93%25-brightgreen)](https://github.com/cacack/gedcom-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/cacack/gedcom-go)](https://goreportcard.com/report/github.com/cacack/gedcom-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/cacack/gedcom-go.svg)](https://pkg.go.dev/github.com/cacack/gedcom-go)
[![Release](https://img.shields.io/github/v/release/cacack/gedcom-go)](https://github.com/cacack/gedcom-go/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/cacack/gedcom-go)](https://github.com/cacack/gedcom-go)

A pure Go library for parsing and validating GEDCOM (GEnealogical Data COMmunication) files.

## Features

- **Multi-version Support**: Parse GEDCOM 5.5, 5.5.1, and 7.0 files
- **Historical Calendar Support**: Parse dates in Julian, Hebrew, and French Republican calendars
- **Read and Write**: Full decoder and encoder for round-trip processing
- **Comprehensive Validation**: Version-aware validation with clear error messages
- **Zero Dependencies**: Uses only the Go standard library
- **Well-tested**: 93% test coverage with multi-platform CI

See [FEATURES.md](FEATURES.md) for the complete feature list including all supported record types, events, attributes, and encoding details.

## Installation

```bash
go get github.com/cacack/gedcom-go
```

## Requirements

- Go 1.21 or later

## Quick Start

### Basic Parsing

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/cacack/gedcom-go/decoder"
)

func main() {
    // Open and parse GEDCOM file
    f, err := os.Open("family.ged")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    doc, err := decoder.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    // Print summary
    fmt.Printf("GEDCOM Version: %s\n", doc.Header.Version)
    fmt.Printf("Individuals: %d\n", len(doc.Individuals()))
    fmt.Printf("Families: %d\n", len(doc.Families()))
}
```

### Working with Individuals

```go
// Find and display individuals
for _, individual := range doc.Individuals() {
    if len(individual.Names) > 0 {
        fmt.Printf("Name: %s\n", individual.Names[0].Full)
    }

    // Access events
    for _, event := range individual.Events {
        fmt.Printf("  %s: %s\n", event.Tag, event.Date)
    }
}
```

### Lookup by Cross-Reference ID

```go
// O(1) lookup by cross-reference ID
person := doc.GetIndividual("@I1@")
if person != nil {
    fmt.Printf("Found: %s\n", person.Names[0].Full)
}

// Lookup works for all record types
family := doc.GetFamily("@F1@")
source := doc.GetSource("@S1@")
repo := doc.GetRepository("@R1@")

// Navigate family relationships
if family != nil {
    husband := doc.GetIndividual(family.Husband)
    wife := doc.GetIndividual(family.Wife)
}
```

### Validating GEDCOM Files

```go
import "github.com/cacack/gedcom-go/validator"

// Validate the document
v := validator.New(doc)
errors := v.Validate()

if len(errors) > 0 {
    fmt.Printf("Found %d validation errors:\n", len(errors))
    for _, err := range errors {
        fmt.Printf("  Line %d: %s\n", err.Line, err.Message)
    }
}
```

### Creating GEDCOM Files

```go
import "github.com/cacack/gedcom-go/encoder"

// Create a new document
doc := &gedcom.Document{
    Header: &gedcom.Header{
        Version:  "5.5",
        Encoding: "UTF-8",
    },
    Records: []*gedcom.Record{
        // Add your records here
    },
}

// Write to file
f, _ := os.Create("output.ged")
defer f.Close()

encoder.Encode(f, doc)
```

## Documentation

- **Usage Guide**: [USAGE.md](USAGE.md) - Comprehensive guide covering basic concepts, examples, and best practices
- **Examples**: See the [`examples/`](examples/) directory ([README](examples/README.md)):
  - [`examples/parse`](examples/parse) - Basic parsing and information display
  - [`examples/encode`](examples/encode) - Creating GEDCOM files programmatically
  - [`examples/query`](examples/query) - Navigating and querying genealogy data
  - [`examples/validate`](examples/validate) - Validating GEDCOM files
- **API Documentation**: [pkg.go.dev/github.com/cacack/gedcom-go](https://pkg.go.dev/github.com/cacack/gedcom-go)
- **Contributing**: [CONTRIBUTING.md](CONTRIBUTING.md)

## Packages

- **`charset`** - Character encoding utilities with UTF-8 validation
- **`decoder`** - High-level GEDCOM decoding with automatic version detection
- **`encoder`** - GEDCOM document writing with configurable line endings
- **`gedcom`** - Core data types (Document, Individual, Family, Source, etc.)
- **`parser`** - Low-level line parsing with detailed error reporting
- **`validator`** - Document validation with error categorization
- **`version`** - GEDCOM version detection (header and heuristic-based)

## Development

### Quick Start with Makefile

The project includes a Makefile for common development tasks:

```bash
# Show all available commands
make help

# Run all checks and build
make all

# Run tests
make test

# Run tests with coverage (93% coverage)
make test-coverage

# Generate HTML coverage report
make coverage-html

# Run benchmarks
make bench

# Format code
make fmt

# Run linters
make vet
make lint

# Run pre-commit checks
make pre-commit

# Clean build artifacts
make clean
```

### Manual Commands

You can also use Go commands directly:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Download dependencies
go mod download

# Build all packages
go build ./...

# Format code
go fmt ./...

# Run static analysis
go vet ./...
```

## Performance

The library is designed for high performance with efficient memory usage:

- **Parser**: 66ns/op for simple lines, ~700μs for 1000 individuals
- **Decoder**: 13ms for 1000 individuals with full document structure
- **Encoder**: 1.15ms for 1000 individuals
- **Validator**: 5.91μs for 1000 individuals, **zero allocations** for valid documents

### Benchmarking

```bash
# Run all benchmarks
make bench

# Run specific package benchmarks
make bench-parse
make bench-decode
make bench-encode

# Save baseline for comparison
make bench-save

# Compare current performance with baseline
make bench-compare
```

### Performance Regression Testing

Automated regression detection with 10% threshold:

```bash
# Run regression tests
make perf-regression
```

For detailed performance metrics, profiling guides, and optimization opportunities, see [PERFORMANCE.md](PERFORMANCE.md).

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please ensure:
- All tests pass (`go test ./...`)
- Code coverage remains ≥85%
- Code is formatted (`go fmt ./...`)
- No linter warnings (`go vet ./...`)

See CONTRIBUTING.md for detailed guidelines.
