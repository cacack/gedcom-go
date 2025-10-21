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

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/cacack/gedcom-go/decoder"
)

func main() {
    // Open GEDCOM file
    f, err := os.Open("family.ged")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    // Parse the file
    doc, err := decoder.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    // Access parsed data
    fmt.Printf("GEDCOM Version: %s\n", doc.Header.Version)
    fmt.Printf("Total Records: %d\n", len(doc.Records))
}
```

## Documentation

Full documentation and examples will be available at:
- Package documentation: [pkg.go.dev](https://pkg.go.dev/github.com/cacack/gedcom-go)
- Examples: See the `examples/` directory

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
