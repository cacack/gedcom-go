# Implementation Plan: GEDCOM Parser Library

**Branch**: `001-gedcom-parser-library` | **Date**: 2025-10-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-gedcom-parser-library/spec.md`

## Summary

Build a pure Go library for parsing, validating, and converting GEDCOM genealogy files across all official versions (5.5, 5.5.1, 7.0). The library provides stream-based parsing with configurable resource limits, comprehensive error reporting with line-level context, and optional progress callbacks. Core capabilities include auto-detection of GEDCOM versions, handling of various character encodings (UTF-8, Latin-1, ANSEL, ASCII), graceful error recovery for malformed files, and semantic validation beyond syntax checking. The implementation uses only Go standard library to maximize compatibility and minimize dependencies.

## Technical Context

**Language/Version**: Go 1.21+ (provides good compatibility while supporting modern features like error wrapping, context cancellation, and io.Reader/Writer interfaces)

**Primary Dependencies**: Go standard library only (`io`, `bufio`, `encoding`, `errors`, `fmt`, `strings`, `unicode/utf8`, `time`, `context`)

**Storage**: N/A (library operates on streams; no persistent storage)

**Testing**: Go's built-in testing framework (`testing` package, `go test`, table-driven tests, benchmarks via `testing.B`)

**Target Platform**: Cross-platform (Linux, macOS, Windows) - pure Go with no platform-specific dependencies

**Project Type**: Single library project (no frontend/backend split)

**Performance Goals**:
- Parse 10,000 records/second on standard hardware
- Complete 10MB file parsing in <2 seconds
- Maintain <200MB peak memory for 100MB files
- Resource limit checks add <5% overhead

**Constraints**:
- No panics in parser (all errors returned via error values)
- Test coverage ≥85%
- API must use io.Reader/Writer (no file path strings)
- Support GEDCOM 5.5, 5.5.1, and 7.0 specifications
- Progress callbacks report every 1% or 100 records
- Resource limits: 100 nesting levels, 1M entities, 5min timeout (all configurable)

**Scale/Scope**:
- Typical files: 1-50MB, 1K-100K records
- Large files: up to 100MB, 1M records
- Streaming mode for files exceeding memory limits

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Library-First Design ✅ PASS

- **Requirement**: Every feature implemented as well-defined library component
- **Status**: COMPLIANT
- **Evidence**: Pure library design with no CLI dependencies; all functionality exposed as importable Go packages; parser, validator, and converter are independent modules

### II. API Clarity ✅ PASS

- **Requirement**: APIs prioritize simplicity and discoverability
- **Status**: COMPLIANT
- **Evidence**:
  - All public APIs will have comprehensive godoc comments
  - Using standard Go idioms (`(result, error)` returns)
  - APIs accept `io.Reader`/`io.Writer` per constitution requirement
  - Breaking changes will follow semantic versioning
  - Examples will be provided in `examples/` directory

### III. Test Coverage (NON-NEGOTIABLE) ✅ PASS

- **Requirement**: ≥85% test coverage, TDD approach, all tests pass before commit
- **Status**: COMPLIANT
- **Evidence**:
  - Table-driven tests planned for parsing/validation logic
  - Integration tests will use real GEDCOM files from `testdata/`
  - TDD workflow: write failing tests → implement → refactor
  - Success criteria SC-005 requires 100% validation coverage

### IV. Version Support ✅ PASS

- **Requirement**: Support GEDCOM 5.5, 5.5.1, 7.0 with auto-detection
- **Status**: COMPLIANT
- **Evidence**:
  - FR-001 explicitly requires all three versions
  - FR-002 requires auto-detection from header
  - Version-specific validation rules will be encapsulated per version
  - Roundtrip fidelity per version (SC-006)

### V. Error Transparency ✅ PASS

- **Requirement**: Errors must include line numbers, context, and never panic
- **Status**: COMPLIANT
- **Evidence**:
  - FR-006, FR-017: Line number preservation and context
  - FR-015: No panics allowed
  - FR-025: Clear error messages for resource limits
  - SC-007: 100% of errors include line numbers and context
  - Using Go 1.13+ error wrapping

### Quality Standards Check ✅ PASS

- **Code Standards**: Will pass `go fmt`, `go vet`, godoc comments on all exports
- **Performance**: Streaming mode (FR-016), benchmarks required
- **Documentation**: README, package-level godoc, CHANGELOG for API changes

**GATE RESULT**: ✅ ALL CHECKS PASS - Proceed to Phase 0

## Project Structure

### Documentation (this feature)

```
specs/001-gedcom-parser-library/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0: Technology decisions and patterns
├── data-model.md        # Phase 1: Entity definitions and relationships
├── quickstart.md        # Phase 1: Getting started guide
├── contracts/           # Phase 1: API contracts (Go package interfaces)
└── checklists/          # Validation checklists
    └── requirements.md
```

### Source Code (repository root)

```
go-gedcom/
├── parser/              # Core GEDCOM line parser and lexer
│   ├── parser.go        # Main parser logic
│   ├── lexer.go         # Tokenization
│   ├── line.go          # Line representation
│   └── parser_test.go
├── decoder/             # High-level decoder (file → structs)
│   ├── decoder.go       # Main decoder API
│   ├── decoder_test.go
│   └── options.go       # Configuration options
├── encoder/             # Encoder (structs → file)
│   ├── encoder.go       # Main encoder API
│   ├── encoder_test.go
│   └── options.go
├── validator/           # Validation engine
│   ├── validator.go     # Validation logic
│   ├── rules.go         # Validation rules by version
│   ├── validator_test.go
│   └── errors.go        # Validation error types
├── converter/           # Version conversion
│   ├── converter.go     # Conversion engine
│   ├── v55_to_v70.go    # 5.5 → 7.0 mappings
│   ├── v70_to_v55.go    # 7.0 → 5.5 mappings
│   └── converter_test.go
├── gedcom/              # Core types and models
│   ├── document.go      # GEDCOM Document
│   ├── record.go        # Base Record type
│   ├── individual.go    # Individual (INDI)
│   ├── family.go        # Family (FAM)
│   ├── source.go        # Source (SOUR)
│   ├── event.go         # Event types
│   ├── tag.go           # Tag-value pairs
│   └── types_test.go
├── version/             # Version detection and specs
│   ├── detect.go        # Auto-detection logic
│   ├── v55.go           # GEDCOM 5.5 spec
│   ├── v551.go          # GEDCOM 5.5.1 spec
│   ├── v70.go           # GEDCOM 7.0 spec
│   └── version_test.go
├── charset/             # Character encoding handling
│   ├── charset.go       # Encoding detection/conversion
│   ├── ansel.go         # ANSEL codec
│   └── charset_test.go
├── internal/            # Internal utilities
│   └── limits/          # Resource limit enforcement
│       └── limits.go
├── examples/            # Example usage code
│   ├── parse/           # Basic parsing example
│   ├── validate/        # Validation example
│   └── convert/         # Conversion example
├── testdata/            # Test GEDCOM files
│   ├── gedcom-5.5/
│   ├── gedcom-5.5.1/
│   ├── gedcom-7.0/
│   └── malformed/
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
└── LICENSE
```

**Structure Decision**: Single library project structure with clear package separation by functionality (parser, decoder, encoder, validator, converter). Each package is independently testable. The `gedcom` package contains core types shared across all packages. The `internal` package holds implementation details not exposed to users. This structure aligns with Go best practices and supports the library-first design principle.

## Complexity Tracking

*No constitution violations identified - this section intentionally left empty.*
