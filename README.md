# gedcom-go

[![CI](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml/badge.svg)](https://github.com/cacack/gedcom-go/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-96.5%25-brightgreen)](https://github.com/cacack/gedcom-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/cacack/gedcom-go)](https://goreportcard.com/report/github.com/cacack/gedcom-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/cacack/gedcom-go.svg)](https://pkg.go.dev/github.com/cacack/gedcom-go)
[![Release](https://img.shields.io/github/v/release/cacack/gedcom-go)](https://github.com/cacack/gedcom-go/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/cacack/gedcom-go)](https://github.com/cacack/gedcom-go)

A pure Go library for parsing and validating GEDCOM (GEnealogical Data COMmunication) files.

## Features

- **Multi-version Support**: Parse GEDCOM 5.5, 5.5.1, and 7.0 files with automatic version detection
- **Stream-based Parsing**: Efficient memory usage for large genealogy files
- **Comprehensive Validation**: Validate GEDCOM data against specification rules
- **Character Encoding**: Support for UTF-8, ANSEL, ASCII, LATIN1, and UNICODE encodings
- **Clear Error Reporting**: All errors include line numbers and context
- **Zero Dependencies**: Uses only the Go standard library
- **Well-tested**: 96.5% test coverage across all packages
- **Production Ready**: Full CI/CD pipeline with automated testing on multiple platforms

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

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

### Building

```bash
# Download dependencies
go mod download

# Build all packages
go build ./...

# Format code
go fmt ./...

# Run static analysis
go vet ./...
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please ensure:
- All tests pass (`go test ./...`)
- Code coverage remains ≥85%
- Code is formatted (`go fmt ./...`)
- No linter warnings (`go vet ./...`)

See CONTRIBUTING.md for detailed guidelines.
