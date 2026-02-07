# gedcom-go

[![CI](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml/badge.svg)](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/cacack/gedcom-go/graph/badge.svg)](https://codecov.io/gh/cacack/gedcom-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/cacack/gedcom-go)](https://goreportcard.com/report/github.com/cacack/gedcom-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/cacack/gedcom-go.svg)](https://pkg.go.dev/github.com/cacack/gedcom-go)
[![Release](https://img.shields.io/github/v/release/cacack/gedcom-go)](https://github.com/cacack/gedcom-go/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/cacack/gedcom-go)](https://github.com/cacack/gedcom-go)

A pure Go library for parsing and validating GEDCOM (GEnealogical Data COMmunication) files.

## Features

- **Multi-version Support**: Parse and write GEDCOM 5.5, 5.5.1, and 7.0 with automatic version detection
- **Version Conversion**: Bidirectional conversion between versions with transformation tracking
- **Historical Calendar Support**: Parse dates in Julian, Hebrew, and French Republican calendars with conversion
- **Streaming APIs**: Memory-efficient parsing and encoding for very large files (1M+ records)
- **Comprehensive Validation**: Date logic, orphaned references, duplicates, and quality reports
- **Vendor Extensions**: Parse Ancestry.com and FamilySearch custom tags
- **Zero Dependencies**: Uses only the Go standard library
- **Well-tested**: 93% test coverage with multi-platform CI

See [FEATURES.md](FEATURES.md) for the complete feature list including all supported record types, events, attributes, and encoding details.

## Compatibility

Support status for common genealogy software:

| Software | Status |
|----------|--------|
| RootsMagic | ⚠️ Tested (older version) |
| Legacy Family Tree | ⚠️ Tested (older version) |
| Family Tree Maker | ⚠️ Tested (older version) |
| Gramps | 🧪 Synthetic test only |
| Ancestry | 🧪 Synthetic test only |

Full compatibility matrix: [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md)

**GEDCOM Specification**: Full support for 5.5, 5.5.1, and 7.0

## Installation

```bash
go get github.com/cacack/gedcom-go
```

## Requirements

- Go 1.24 or later

This library tracks Go's [release policy](https://go.dev/doc/devel/release#policy), supporting the two most recent major versions. When a Go version reaches end-of-life and no longer receives security patches, we bump our minimum accordingly.

## Quick Start

The library provides a simple, single-import API for common operations. Import with an alias for cleaner code:

```go
import gedcomgo "github.com/cacack/gedcom-go"
```

### Parse a GEDCOM File

```go
package main

import (
    "fmt"
    "log"
    "os"

    gedcomgo "github.com/cacack/gedcom-go"
)

func main() {
    f, err := os.Open("family.ged")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    doc, err := gedcomgo.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("GEDCOM Version: %s\n", doc.Header.Version)
    fmt.Printf("Individuals: %d\n", len(doc.Individuals()))
    fmt.Printf("Families: %d\n", len(doc.Families()))
}
```

### Validate a Document

```go
// Basic validation (returns []error)
errors := gedcomgo.Validate(doc)

// Comprehensive validation with severity levels (returns []Issue)
issues := gedcomgo.ValidateAll(doc)
for _, issue := range issues {
    fmt.Printf("[%s] %s\n", issue.Severity, issue.Message)
}
```

### Write a GEDCOM File

```go
f, _ := os.Create("output.ged")
defer f.Close()

err := gedcomgo.Encode(f, doc)
```

### Convert Between Versions

```go
// Convert to GEDCOM 7.0
converted, report, err := gedcomgo.Convert(doc, gedcomgo.Version70)
if report.HasDataLoss() {
    for _, item := range report.DataLoss {
        fmt.Printf("Lost: %s - %s\n", item.Feature, item.Reason)
    }
}
```

### Working with Records

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

// O(1) lookup by cross-reference ID
person := doc.GetIndividual("@I1@")
if person != nil {
    fmt.Printf("Found: %s\n", person.Names[0].Full)
}

// Navigate family relationships
family := doc.GetFamily("@F1@")
if family != nil {
    husband := doc.GetIndividual(family.Husband)
    wife := doc.GetIndividual(family.Wife)
}
```

### Parse with Diagnostics

Process GEDCOM files with errors while extracting as much valid data as possible:

```go
result, err := gedcomgo.DecodeWithDiagnostics(f)
if err != nil {
    log.Fatal(err) // Fatal I/O error
}

// Check for parse issues
if result.Diagnostics.HasErrors() {
    fmt.Printf("Found %d errors\n", len(result.Diagnostics.Errors()))
    for _, d := range result.Diagnostics {
        fmt.Printf("  Line %d: %s\n", d.Line, d.Message)
    }
}

// Use the partial document
doc := result.Document
fmt.Printf("Parsed %d individuals\n", len(doc.Individuals()))
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

## Advanced Usage

For advanced use cases requiring custom options, import the underlying packages directly:

```go
import (
    "github.com/cacack/gedcom-go/decoder"
    "github.com/cacack/gedcom-go/encoder"
    "github.com/cacack/gedcom-go/validator"
    "github.com/cacack/gedcom-go/converter"
)
```

### Custom Decode Options

```go
// Decode with progress reporting and timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

opts := &decoder.DecodeOptions{
    Context:   ctx,
    TotalSize: fileInfo.Size(),
    OnProgress: func(bytesRead, totalBytes int64) {
        fmt.Printf("\rProgress: %d%%", bytesRead*100/totalBytes)
    },
}
doc, err := decoder.DecodeWithOptions(reader, opts)
```

### Custom Validation Configuration

```go
// Configure validation strictness and duplicate detection
config := &validator.ValidatorConfig{
    Strictness: validator.StrictnessStrict,
    Duplicates: &validator.DuplicateConfig{
        RequireExactSurname: true,
        MinNameSimilarity:   0.8,
    },
}
v := validator.NewWithConfig(config)
issues := v.ValidateAll(doc)
```

### Custom Encoder Options

```go
// Encode with custom line endings and line length
opts := &encoder.EncodeOptions{
    LineEnding:    encoder.LineEndingLF,
    MaxLineLength: 255,
}
err := encoder.EncodeWithOptions(writer, doc, opts)
```

### Custom Conversion Options

```go
// Convert with strict data loss checking
opts := &converter.ConvertOptions{
    Validate:       true,
    StrictDataLoss: true,  // Fail on any data loss
}
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version55, opts)
```

## Packages

For fine-grained control, these packages are available:

- **`charset`** - Character encoding utilities with UTF-8 validation
- **`converter`** - Version conversion with transformation tracking
- **`decoder`** - High-level GEDCOM decoding with automatic version detection
- **`encoder`** - GEDCOM document writing with configurable line endings
- **`gedcom`** - Core data types (Document, Individual, Family, Source, etc.)
- **`parser`** - Low-level line parsing with detailed error reporting
- **`validator`** - Document validation with error categorization
- **`version`** - GEDCOM version detection (header and heuristic-based)

## API Stability

This library follows [Semantic Versioning](https://semver.org/). We do not break exported types in v1+ without a major version bump.

### Stable Packages

| Package | Key APIs |
|---------|----------|
| `gedcom` | Document, Individual, Family, Event, Date |
| `decoder` | Decode(), DecodeWithOptions() |
| `encoder` | Encode(), EncodeWithOptions() |
| `converter` | Convert(), ConvertWithOptions() |
| `parser` | Parse(), ParseLine() |
| `validator` | Validate(), ValidateAll() |
| `charset` | NewReader() |
| `version` | Detect() |

### What May Change

- **Experimental features** (streaming APIs, duplicate detection) may evolve in minor versions

### GEDCOM Spec Evolution

As GEDCOM 7.x evolves, we add support additively. New tags and structures are added without breaking existing code.

### Vendor Extensions

Vendor extensions (Ancestry, FamilySearch) are best-effort and not covered by stability guarantees.

For the complete policy including deprecation process, see [docs/API_STABILITY.md](docs/API_STABILITY.md).

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

For detailed performance metrics, profiling guides, and optimization opportunities, see [docs/PERFORMANCE.md](docs/PERFORMANCE.md).

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please ensure:
- All tests pass (`go test ./...`)
- Code coverage remains ≥85%
- Code is formatted (`go fmt ./...`)
- No linter warnings (`go vet ./...`)

See CONTRIBUTING.md for detailed guidelines.
