# gedcom-go

[![CI](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml/badge.svg)](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/cacack/gedcom-go/badge)](https://scorecard.dev/viewer/?uri=github.com/cacack/gedcom-go)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/12809/badge)](https://www.bestpractices.dev/projects/12809)
[![codecov](https://codecov.io/gh/cacack/gedcom-go/graph/badge.svg)](https://codecov.io/gh/cacack/gedcom-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/cacack/gedcom-go/v2)](https://goreportcard.com/report/github.com/cacack/gedcom-go/v2)
[![GoDoc](https://pkg.go.dev/badge/github.com/cacack/gedcom-go/v2.svg)](https://pkg.go.dev/github.com/cacack/gedcom-go/v2)
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
- **Well-tested**: 93-100% per-package test coverage with multi-platform CI

See [FEATURES.md](FEATURES.md) for the complete feature list including all supported record types, events, attributes, and encoding details.

## Compatibility

Support status for common genealogy software:

| Software | Status |
|----------|--------|
| Ancestry.com | ✅ Real export tested (2025) |
| FamilySearch | ✅ Real export tested (2025) |
| MyHeritage | ✅ Real export tested (2025) |
| Gramps | ✅ Real export tested (2025) |
| RootsMagic | ✅ Real export tested (2026) |
| Legacy Family Tree | ⚠️ Tested (older version) |
| Family Tree Maker | ⚠️ Tested (older version) |

Full compatibility matrix: [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md)

**GEDCOM Specification**: Full support for 5.5, 5.5.1, and 7.0

## Installation

```bash
go get github.com/cacack/gedcom-go/v2
```

## Requirements

- Go 1.25 or later

This library tracks Go's [release policy](https://go.dev/doc/devel/release#policy), supporting the two most recent major versions. When a Go version reaches end-of-life and no longer receives security patches, we bump our minimum accordingly.

## Quick Start

The library provides a simple, single-import API for common operations. Import with an alias for cleaner code:

```go
import gedcomgo "github.com/cacack/gedcom-go/v2"
```

### Parse a GEDCOM File

```go
package main

import (
    "fmt"
    "log"
    "os"

    gedcomgo "github.com/cacack/gedcom-go/v2"
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

### Streaming for Large Files

For files too large to materialize in memory (10k+ individuals, exports from major platforms), use the streaming APIs. The parser yields records one at a time; the encoder writes them one at a time. The full `Document` is never constructed, so the heap retained after the operation completes is a small constant rather than proportional to file size.

```go
import (
    "os"

    "github.com/cacack/gedcom-go/v2/charset"
    "github.com/cacack/gedcom-go/v2/encoder"
    "github.com/cacack/gedcom-go/v2/gedcom"
    "github.com/cacack/gedcom-go/v2/parser"
)

// Streaming parse — iterate level-0 records without building a Document.
func countRecords(path string) (map[string]int, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    counts := make(map[string]int)
    for rec, err := range parser.Records(charset.NewReader(f)) {
        if err != nil {
            return nil, err   // rec is nil here; always check err first
        }
        counts[rec.Type]++
    }
    return counts, nil
}

// Streaming encode — call sequence is WriteHeader → WriteRecord* → WriteTrailer → Close.
// Capture Close's error so ErrTrailerNotWritten and flush failures aren't dropped.
func writeStreamed(path string, records []*gedcom.Record) (err error) {
    out, err := os.Create(path)
    if err != nil {
        return err
    }
    defer func() {
        if cerr := out.Close(); cerr != nil && err == nil {
            err = cerr
        }
    }()

    enc := encoder.NewStreamEncoder(out)
    defer func() {
        if cerr := enc.Close(); cerr != nil && err == nil {
            err = cerr
        }
    }()

    if err := enc.WriteHeader(&gedcom.Header{Version: "5.5", Encoding: "UTF-8"}); err != nil {
        return err
    }
    for _, rec := range records {
        if err := enc.WriteRecord(rec); err != nil {
            return err
        }
    }
    return enc.WriteTrailer()
}
```

On a 1.1MB / 2,322-individual file, streaming parse holds **~17%** of the heap that batch decode retains after the call returns (and ~54% of the cumulative allocations). See [`examples/stream`](examples/stream/main.go) for the full pattern and [docs/PERFORMANCE.md](docs/PERFORMANCE.md#streaming-apis-performance) for benchmark details.

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

Process GEDCOM files with errors while extracting as much valid data as possible. Lenient mode (the default for `DecodeWithDiagnostics`) recovers from common real-world quirks — empty lines, invalid level numbers, unknown tags, and **malformed indentation jumps** (e.g., `1 BIRT` directly followed by `4 DATE`, as seen in some Ancestry/MyHeritage exports). Recovered issues are reported as `Diagnostic`s with codes like `BAD_LEVEL_JUMP`, `UNKNOWN_TAG`, and `EMPTY_LINE` rather than failing the parse.

```go
result, err := gedcomgo.DecodeWithDiagnostics(f)
if err != nil {
    log.Fatal(err) // Fatal I/O error
}

// Check for parse issues
if result.Diagnostics.HasErrors() {
    fmt.Printf("Found %d errors\n", len(result.Diagnostics.Errors()))
    for _, d := range result.Diagnostics {
        fmt.Printf("  Line %d: [%s] %s\n", d.Line, d.Code, d.Message)
    }
}

// Use the partial document
doc := result.Document
fmt.Printf("Parsed %d individuals\n", len(doc.Individuals()))
```

To opt into strict parsing (fail on the first syntax error, no diagnostics collected), pass `&decoder.DecodeOptions{StrictMode: true}` — see [Custom Decode Options](#custom-decode-options).

## Documentation

- **Usage Guide**: [USAGE.md](USAGE.md) - Comprehensive guide covering basic concepts, examples, and best practices
- **Examples**: See the [`examples/`](examples/) directory ([README](examples/README.md)):
  - [`examples/parse`](examples/parse) - Basic parsing and information display
  - [`examples/encode`](examples/encode) - Creating GEDCOM files programmatically
  - [`examples/query`](examples/query) - Navigating and querying genealogy data
  - [`examples/validate`](examples/validate) - Validating GEDCOM files
  - [`examples/stream`](examples/stream) - Streaming parse and encode for very large files
- **API Documentation**: [pkg.go.dev/github.com/cacack/gedcom-go/v2](https://pkg.go.dev/github.com/cacack/gedcom-go/v2)
- **Contributing**: [CONTRIBUTING.md](CONTRIBUTING.md)

## Advanced Usage

The `gedcomgo` facade also exposes `*WithOptions` variants for the common operations:

```go
doc, err := gedcomgo.DecodeWithOptions(r, opts)
err     = gedcomgo.EncodeWithOptions(w, doc, opts)
errs   := gedcomgo.ValidateWithOptions(doc, opts)
issues := gedcomgo.ValidateAllWithOptions(doc, opts)
```

Option types — `DecodeOptions`, `EncodeOptions`, `ValidateOptions` — are re-exported from the facade. For the full surface area (streaming, diagnostics, converter), import the underlying packages directly:

```go
import (
    "github.com/cacack/gedcom-go/v2/decoder"
    "github.com/cacack/gedcom-go/v2/encoder"
    "github.com/cacack/gedcom-go/v2/validator"
    "github.com/cacack/gedcom-go/v2/converter"
)
```

### Custom Decode Options

```go
// Decode with progress reporting and timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

opts := &gedcomgo.DecodeOptions{
    Context:   ctx,
    TotalSize: fileInfo.Size(),
    OnProgress: func(bytesRead, totalBytes int64) {
        fmt.Printf("\rProgress: %d%%", bytesRead*100/totalBytes)
    },
}
doc, err := gedcomgo.DecodeWithOptions(reader, opts)
```

### Custom Validation Options

```go
// Configure validation strictness and duplicate detection
opts := &gedcomgo.ValidateOptions{
    Strictness: validator.StrictnessStrict,
    MaxErrors:  100,
    SkipRules:  []string{"W001"},
    Duplicates: &validator.DuplicateConfig{
        RequireExactSurname: true,
        MinNameSimilarity:   0.8,
    },
}
issues := gedcomgo.ValidateAllWithOptions(doc, opts)
```

### Custom Encoder Options

```go
// Encode with custom line endings and line length
opts := &gedcomgo.EncodeOptions{
    LineEnding:    "\r\n", // CRLF
    MaxLineLength: 248,
}
err := gedcomgo.EncodeWithOptions(writer, doc, opts)
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
| `encoder` | Encode(), EncodeWithOptions(), NewStreamEncoder(), NewStreamEncoderWithOptions(), EncodeStreaming(), EncodeStreamingWithOptions() |
| `converter` | Convert(), ConvertWithOptions() |
| `parser` | Parse(), ParseLine(), NewRecordIterator(), NewRecordIteratorWithOffset(), Records(), RecordsWithOffset(), NewLazyParser() |
| `validator` | Validate(), ValidateAll(), NewStreamingValidator() |
| `charset` | NewReader() |
| `version` | Detect() |

### What May Change

- **Experimental features** (duplicate detection algorithms, quality report format) may evolve in minor versions

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

# Run tests with coverage (93-100% per-package coverage)
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
