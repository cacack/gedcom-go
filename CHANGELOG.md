# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-20

### Added

#### Core Functionality
- Complete GEDCOM parser supporting versions 5.5, 5.5.1, and 7.0
- Stream-based parsing for efficient memory usage with large files
- Comprehensive validation against GEDCOM specifications
- Character encoding support (UTF-8, ANSEL, ASCII, LATIN1, UNICODE)
- UTF-8 validation with BOM (Byte Order Mark) removal
- GEDCOM file encoding/writing capability
- Cross-reference resolution and validation

#### Packages
- `charset` - Character encoding utilities with UTF-8 validation
- `decoder` - High-level GEDCOM decoding with automatic version detection
- `encoder` - GEDCOM document writing with configurable line endings
- `gedcom` - Core data types (Document, Individual, Family, Source, etc.)
- `parser` - Low-level line parsing with detailed error reporting
- `validator` - Document validation with error categorization
- `version` - GEDCOM version detection (header and heuristic-based)

#### Documentation
- Complete godoc comments for all public APIs
- Package-level documentation with usage examples
- Comprehensive CONTRIBUTING.md with:
  - Development environment setup
  - Code standards and style guidelines
  - Testing requirements (85% minimum, 90% target)
  - Pull request and issue templates
- Project roadmap in docs/TODO.md
- Four example programs:
  - `examples/parse` - Basic GEDCOM parsing and information display
  - `examples/encode` - Programmatic GEDCOM document creation
  - `examples/query` - Navigating and querying genealogy data
  - `examples/validate` - GEDCOM file validation

#### Testing
- Comprehensive test suite with >90% coverage across all packages:
  - charset: 100.0% coverage
  - gedcom: 100.0% coverage
  - encoder: 95.7% coverage
  - validator: 94.4% coverage
  - parser: 94.3% coverage
  - decoder: 92.1% coverage
  - version: 87.5% coverage
- Table-driven tests for edge cases
- Integration tests with real GEDCOM files
- Error handling and validation tests

### Changed

- **BREAKING**: Module path changed from `github.com/elliotchance/go-gedcom` to `github.com/cacack/gedcom-go`
  - Users must update import statements when upgrading
  - Example: `import "github.com/cacack/gedcom-go/decoder"`

### Technical Details

#### Supported GEDCOM Versions
- GEDCOM 5.5 (genealogical data standard)
- GEDCOM 5.5.1 (enhanced version with additional tags)
- GEDCOM 7.0 (latest specification)

#### Character Encodings
- UTF-8 (recommended, with BOM support)
- ANSEL (legacy genealogy encoding)
- ASCII
- LATIN1
- UNICODE

#### Key Features
- Zero external dependencies (uses only Go standard library)
- Detailed error reporting with line numbers and context
- Cross-reference (XRef) lookup and validation
- Helper methods for common queries (Individuals(), Families(), Sources())
- Configurable encoding options (line endings, etc.)
- Malformed file handling with clear error messages

### Requirements

- Go 1.21 or later

### Installation

```bash
go get github.com/cacack/gedcom-go
```

### Quick Start

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/cacack/gedcom-go/decoder"
)

func main() {
    f, err := os.Open("family.ged")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    doc, err := decoder.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d individuals\n", len(doc.Individuals()))
}
```

---

## Release Process

This release represents the initial public version of gedcom-go, providing
a solid foundation for GEDCOM file processing in Go with:

- Production-ready parser and validator
- Comprehensive test coverage
- Complete documentation
- Real-world examples
- Clear contribution guidelines

Future releases will focus on:
- Performance optimization
- Enhanced query APIs
- Additional export formats (JSON, XML)
- Merge/diff capabilities

[0.1.0]: https://github.com/cacack/gedcom-go/releases/tag/v0.1.0
