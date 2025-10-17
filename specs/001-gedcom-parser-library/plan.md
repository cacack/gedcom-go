# Implementation Plan: GEDCOM Parser Library

**Branch**: `001-gedcom-parser-library` | **Date**: 2025-10-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-gedcom-parser-library/spec.md`

## Summary

Build a pure Go library for parsing and validating GEDCOM genealogy files across all official versions (5.5, 5.5.1, 7.0). The library provides stream-based parsing with simple resource limits, comprehensive error reporting with line-level context, and a clean, minimal API. Core capabilities include auto-detection of GEDCOM versions, UTF-8 character encoding (with ANSEL/Latin-1 support planned for later phases), graceful error recovery for malformed files, and semantic validation beyond syntax checking. The implementation uses only Go standard library to maximize compatibility and minimize dependencies.

**Scope Note**: This plan focuses on core parsing and validation capabilities. Advanced features like version conversion, progress callbacks, and extended character encoding support are documented as future improvements to be added after the core library is stable.

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
- Initial implementation: UTF-8 encoding only (ANSEL/Latin-1 in future phases)
- Simple resource limits: 100 nesting levels max, context-based timeout (configurable)

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
├── charset/             # Character encoding handling (UTF-8 initially)
│   ├── charset.go       # Encoding detection/conversion
│   └── charset_test.go
├── examples/            # Example usage code
│   ├── parse/           # Basic parsing example
│   └── validate/        # Validation example
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

**Structure Decision**: Single library project structure with clear package separation by functionality (parser, decoder, encoder, validator). Each package is independently testable. The `gedcom` package contains core types shared across all packages. This structure aligns with Go best practices and supports the library-first design principle.

**Simplified from original**: Removed `converter/` package (deferred to future), `internal/limits/` package (using inline checks instead), `charset/ansel.go` (deferred to Phase 4), and `examples/convert/` (no converter yet).

## Complexity Tracking

### Simplifications Applied (2025-10-16)

The following components were identified as over-engineered and have been simplified or deferred:

1. **Version Converter Package** - Removed from initial scope
   - Original: Full bidirectional converter (5.5 ↔ 5.5.1 ↔ 7.0) with transformation reports
   - Reason: P4 priority, not constitutionally required, adds 4-6 days effort
   - Status: Moved to Future Improvements (may be separate library)

2. **`internal/limits/` Package** - Simplified to inline checks
   - Original: Dedicated package for resource tracking (depth, count, timeout)
   - Reason: Over-abstraction, simple counters and context.Context suffice
   - Status: Using inline depth counter + context with timeout

3. **Progress Callbacks** - Deferred to v2.0
   - Original: Callback infrastructure reporting every 1% or 100 records
   - Reason: Not constitutionally required, adds API complexity
   - Status: Moved to Future Improvements

4. **Multi-Encoding Support** - Phased approach
   - Original: Full ANSEL, UTF-8, Latin-1, ASCII support in Phase 1
   - Reason: ANSEL codec is complex (2-3 days), most modern files use UTF-8
   - Status: UTF-8 only in MVP, ANSEL/Latin-1 in Phase 4

**Impact**: Reduces initial implementation from 25-39 days to 14-22 days (41% reduction) while maintaining full constitutional compliance.

## Implementation Reference Map

This section provides cross-references to detailed implementation guidance across planning documents.

### Core Features Implementation Guide

| Feature | Primary Reference | Key Details | Related Requirements |
|---------|------------------|-------------|---------------------|
| **UTF-8 Encoding** | research.md:62-84 | UTF-8 validation and handling (initial phase) | FR-007 (partial) |
| **Date Parsing** | research.md:574-748 | All calendar types (Gregorian, Hebrew, French, Julian), parsing strategy (602-614) | FR-001 |
| **Version Detection** | research.md:232-254 | Header parsing, tag-based fallback heuristics | FR-002 |
| **Error Handling Pattern** | research.md:115-147 | Structured errors with line numbers and context | FR-006, FR-015, FR-017 |
| **Streaming Mode** | research.md:149-176 | io.Reader-based parsing, memory management | FR-016, FR-020 |
| **Two-Tier Parsing** | research.md:87-114 | Lexer/Parser → Decoder separation, clean interfaces | FR-003, FR-008 |
| **Testing Strategy** | research.md:256-290 | Table-driven tests, integration tests, benchmarks | Constitution III |

**Deferred to Future Phases**:
- **ANSEL Encoding**: research.md:319-387 (Phase 4 / v2.0)
- **Character Encoding Fallback**: research.md:62-84 (Phase 4 / v2.0)
- **Resource Limits Package**: research.md:179-202 (using inline checks instead)
- **Progress Callbacks**: research.md:206-231 (v2.0)

### Data Structure Definitions

| Component | Reference | Description |
|-----------|-----------|-------------|
| Document | data-model.md:22-58 | Complete GEDCOM file with header, records, trailer, XRefMap |
| Record | data-model.md:61-108 | Base entity for all record types (INDI, FAM, SOUR, etc.) |
| Individual | data-model.md:111-158 | Person with names, events, attributes, family relationships |
| Family | data-model.md:161-196 | Family unit with spouses, children, events |
| Source | data-model.md:199-236 | Source of genealogical information |
| Repository | data-model.md:239-265 | Physical/digital location where sources stored |
| Event | data-model.md:268-327 | Life events with dates, places, sources |
| Note | data-model.md:330-352 | Extended textual information |
| Media Object | data-model.md:355-389 | Multimedia files (photos, documents, etc.) |
| Cross-Reference | data-model.md:392-414 | Link between records using XRef IDs |
| Tag-Value Pair | data-model.md:417-453 | Fundamental hierarchical GEDCOM structure |

### API Contracts

| Interface | Reference | Purpose |
|-----------|-----------|---------|
| Parser | contracts/interfaces.go:14-23 | Low-level line tokenization |
| Decoder | contracts/interfaces.go:34-47 | High-level document building |
| Encoder | contracts/interfaces.go:161-168 | Write GEDCOM to output stream |
| Validator | contracts/interfaces.go:189-197 | Check spec compliance |
| VersionDetector | contracts/interfaces.go:315-320 | Auto-detect GEDCOM version |

**Note**: Converter interface removed from initial scope (see Future Improvements)

### Specification Details by Version

| Specification | Reference | Key Differences |
|---------------|-----------|-----------------|
| GEDCOM 5.5 | research.md:320-403 | ANSEL default, CONC/CONT for continuation, relaxed XRef format |
| GEDCOM 5.5.1 | research.md:406-439 | UTF-8 added, new tags (EMAIL, WWW, MAP, LATI, LONG), no BLOBs |
| GEDCOM 7.0 | research.md:442-484 | UTF-8 mandatory, CONC removed, strict XRef format, new tags (SCHMA, EXID, NO) |

### Go 1.21 Standard Library Features

| Feature | Reference | Use Case |
|---------|-----------|----------|
| slices Package | research.md:491-495 | Fast membership checks, filtering, sorting |
| maps Package | research.md:497-500 | XRefMap operations, cloning |
| bytes.Buffer Improvements | research.md:502-505 | Efficient encoding without allocations |
| context Package | research.md:507-510 | Timeout handling with clear error messages |
| errors.Join | research.md:512-514 | Aggregate validation errors |
| Profile-Guided Optimization | research.md:516-519 | 10-20% performance improvement |

## Package Dependency Order

Implementation must proceed in this order to satisfy package dependencies:

### Layer 1: Foundation (No Dependencies)
These packages can be implemented in parallel as they have no internal dependencies:

- **`charset/`** (research.md:62-84)
  - Purpose: Character encoding handling (UTF-8 initially)
  - No dependencies (stdlib only)
  - Estimated effort: 0.5-1 day (UTF-8 only; ANSEL deferred)
  - Files: `charset.go`, `charset_test.go`

- **`gedcom/`** (types only) (data-model.md)
  - Purpose: Core type definitions (Document, Record, Tag, etc.)
  - No dependencies
  - Estimated effort: 1-2 days
  - Files: `document.go`, `record.go`, `individual.go`, `family.go`, `source.go`, `event.go`, `tag.go`, `types_test.go`

### Layer 2: Core Parsing
These packages depend on Layer 1:

- **`parser/`** (research.md:87-114)
  - Purpose: Line-level tokenization and parsing
  - Dependencies: `charset/`
  - Estimated effort: 3-4 days
  - Files: `parser.go`, `lexer.go`, `line.go`, `errors.go`, `parser_test.go`
  - Note: Includes inline depth checking (no separate limits package)

- **`version/`** (research.md:232-254)
  - Purpose: GEDCOM version detection
  - Dependencies: `parser/` (for header parsing)
  - Estimated effort: 1-2 days
  - Files: `detect.go`, `v55.go`, `v551.go`, `v70.go`, `version_test.go`

### Layer 3: High-Level Decoding
These packages depend on Layers 1 and 2:

- **`decoder/`** (research.md:149-176)
  - Purpose: Build structured Document from line stream
  - Dependencies: `parser/`, `gedcom/`, `version/`
  - Estimated effort: 4-6 days
  - Files: `decoder.go`, `decoder_test.go`, `options.go`
  - Note: Uses context.Context for timeout, no separate limits package

### Layer 4: Validation and Encoding
These packages depend on the core being functional (Layers 1-3):

- **`validator/`** (research.md:115-147, contracts/interfaces.go:189-257)
  - Purpose: Validate GEDCOM against specification rules
  - Dependencies: `gedcom/`, `version/`
  - Estimated effort: 3-5 days
  - Files: `validator.go`, `rules.go`, `validator_test.go`, `errors.go`

- **`encoder/`** (contracts/interfaces.go:161-187)
  - Purpose: Write GEDCOM data to output stream
  - Dependencies: `gedcom/`, `charset/`
  - Estimated effort: 2-3 days
  - Files: `encoder.go`, `encoder_test.go`, `options.go`

### Layer 5: Examples and Documentation
These depend on all core functionality being complete:

- **`examples/`** (quickstart.md)
  - Purpose: Demonstrate library usage
  - Dependencies: All packages above
  - Estimated effort: 1-2 days
  - Directories: `examples/parse/`, `examples/validate/`

### Dependency Graph Visualization

```
Foundation Layer (parallel):
├── charset/ (UTF-8 only)
└── gedcom/ (types only)

Core Parsing Layer:
├── parser/ ──depends on──> charset/
└── version/ ──depends on──> parser/

High-Level Decoding Layer:
└── decoder/ ──depends on──> parser/, gedcom/, version/

Validation & Encoding Layer (can be parallel after decoder works):
├── validator/ ──depends on──> gedcom/, version/
└── encoder/ ──depends on──> gedcom/, charset/

Examples Layer:
└── examples/ ──depends on──> ALL
```

### Critical Path

The critical path for minimum viable functionality (parsing GEDCOM files):

1. `charset/` (0.5-1 day) - UTF-8 only
2. `gedcom/` types (1-2 days) - can overlap with charset
3. `parser/` (3-4 days) - **blocked by charset/**
4. `version/` (1-2 days) - **blocked by parser/**
5. `decoder/` (4-6 days) - **blocked by parser/, gedcom/, version/**

**Minimum viable product**: ~9-15 days (can parse valid GEDCOM files)

After MVP, validator and encoder can be added in parallel (~5-8 days combined).

## Phase Success Criteria

Each implementation phase has specific, measurable criteria that must be met before proceeding to the next phase.

### Phase 0: Research & Planning ✅ COMPLETE

**Criteria**:
- [x] All GEDCOM specifications researched (5.5, 5.5.1, 7.0)
- [x] Character encoding strategy decided (including ANSEL)
- [x] Architecture patterns chosen (two-tier parsing)
- [x] Data model defined for all entity types
- [x] API contracts documented
- [x] Go version selected (1.21+) with rationale
- [x] Testing strategy defined
- [x] Constitution compliance verified

**Deliverables**:
- [x] research.md completed
- [x] data-model.md completed
- [x] contracts/interfaces.go created
- [x] quickstart.md drafted
- [x] All clarifications from spec.md resolved

**Validation**: All research documents reviewed and approved ✅

---

### Phase 1: Foundation Layer (charset UTF-8, types)

**Criteria**:
- [ ] UTF-8 validation works correctly
- [ ] UTF-8 encoding/decoding functional
- [ ] All core types (Document, Record, Tag, etc.) defined per data-model.md
- [ ] Test coverage ≥85% for all foundation packages
- [ ] Zero external dependencies (stdlib only)
- [ ] No panics in any code path

**Test Requirements**:
- UTF-8 validation: Correctly identify valid/invalid UTF-8 sequences
- UTF-8 roundtrip: Read UTF-8 file → parse → write → preserves content
- Type safety: All enums and constants properly defined
- Edge cases: BOM handling, multibyte characters

**Performance**:
- UTF-8 validation: <1ms for typical header
- Encoding detection: <5ms for typical files

**Deliverables**:
- [ ] `charset/charset.go` with UTF-8 handling
- [ ] `charset/charset_test.go` with comprehensive tests
- [ ] `gedcom/` package with all type definitions
- [ ] `gedcom/types_test.go`

**Validation Method**:
- Run: `go test ./charset ./gedcom -cover`
- Coverage must be ≥85%
- All tests must pass

**Note**: ANSEL and Latin-1 support deferred to Phase 4

---

### Phase 2: Core Parsing (parser, version)

**Criteria**:
- [ ] Parser correctly tokenizes all valid GEDCOM lines per research.md:320-329
- [ ] All line endings handled (CRLF, LF, CR) per FR-018
- [ ] Line numbers preserved accurately per FR-017
- [ ] Version detection works for all three versions (5.5, 5.5.1, 7.0) per FR-002
- [ ] Errors include line number and content context per FR-006
- [ ] Parser handles malformed input gracefully (no panics) per FR-015
- [ ] Test coverage ≥85%
- [ ] Integration with charset package successful

**Test Requirements**:
- Table-driven tests for line formats: `LEVEL [XREF] TAG [VALUE]`
- Edge cases: empty lines, long lines (>1000 chars), invalid levels
- All GEDCOM versions detected from testdata files
- Error messages include line numbers and helpful context
- Cross-reference format validation: `@[A-Za-z0-9_-]+@` (FR-022)

**Performance**:
- Parse 10,000 lines/second minimum (SC-002)
- Line parsing overhead <10% of total decode time

**Deliverables**:
- [ ] `parser/line.go` with Line struct
- [ ] `parser/lexer.go` with tokenization logic
- [ ] `parser/parser.go` implementing Parser interface
- [ ] `parser/errors.go` with ParseError type
- [ ] `parser/parser_test.go` with table-driven tests
- [ ] `version/detect.go` with detection logic
- [ ] `version/v55.go`, `version/v551.go`, `version/v70.go` with tag lists
- [ ] `version/version_test.go`

**Validation Method**:
- Run: `go test ./parser ./version -cover -v`
- Test with real GEDCOM files from `testdata/`
- Verify error messages are actionable
- Check version detection accuracy on all testdata files

---

### Phase 3: High-Level Decoding (decoder)

**Criteria**:
- [ ] Successfully parses complete GEDCOM files into Document structs per FR-003
- [ ] XRefMap correctly resolves all cross-references per FR-005
- [ ] Simple resource limits: max depth tracking, context timeout
- [ ] Streaming mode works for large files per FR-016
- [ ] Empty/header-only files handled per FR-019
- [ ] Memory usage <200MB for 100MB files per SC-003
- [ ] Parse 10MB file in <2 seconds per SC-008
- [ ] Test coverage ≥85%

**Test Requirements**:
- Parse all files in `testdata/gedcom-5.5/`, `testdata/gedcom-5.5.1/`, `testdata/gedcom-7.0/`
- Validate XRefMap: every XRef in document is in map, all references resolve
- Resource limits: trigger depth limit and timeout, verify error
- Streaming: process 100K records without buffering in memory
- Edge cases: empty file, header-only, circular references (parse successfully)

**Performance Benchmarks**:
- `BenchmarkDecode10MB`: <2 seconds per SC-008
- `BenchmarkDecode100MB`: <200MB peak memory per SC-003
- `BenchmarkDecodeStream`: constant memory regardless of file size

**Deliverables**:
- [ ] `decoder/decoder.go` implementing Decoder interface
- [ ] `decoder/options.go` with DecodeOptions struct (context.Context for timeout)
- [ ] `decoder/decoder_test.go` with integration tests
- [ ] Integration tests using real testdata files

**Validation Method**:
- Run: `go test ./decoder -cover -v -timeout=10m`
- Run: `go test ./decoder -bench=. -benchmem`
- Verify SC-001: Parse file in <5 lines of code (check quickstart.md examples)
- Verify SC-002: ≥10,000 records/second
- Verify SC-003: <200MB for 100MB files
- Verify SC-008: 10MB file in <2 seconds

**Note**: Progress callbacks deferred to v2.0; entity count limit removed (depth + timeout sufficient)

---

### Phase 4: Validation (validator)

**Criteria**:
- [ ] Catches 100% of required field violations per SC-005
- [ ] Catches 100% of invalid cross-references per SC-005
- [ ] Detects circular family relationships per FR-010
- [ ] Validates date formats per FR-010 (catches "32 JAN 2020")
- [ ] Version-specific validation (different rules for 5.5, 5.5.1, 7.0)
- [ ] Warnings for non-standard XRef formats per FR-022
- [ ] All errors include line numbers and context per SC-007
- [ ] Test coverage ≥85%

**Test Requirements**:
- Missing required fields: Individual without NAME, Family without spouses/children
- Invalid dates: "32 JAN 2020", "13 ABC 2020", malformed ranges
- Broken XRefs: reference to non-existent @I999@
- Circular references: person as own ancestor
- Non-standard formats: `@I-001@` triggers warning but parses
- Version-specific: 7.0 file using 5.5 deprecated tags

**Validation Logic Coverage**:
- 100% of FR-009 requirements tested
- 100% of FR-010 requirements tested
- All validation codes from contracts/interfaces.go:243-250 exercised

**Deliverables**:
- [ ] `validator/validator.go` implementing Validator interface
- [ ] `validator/rules.go` with version-specific rules
- [ ] `validator/errors.go` with ValidationError type
- [ ] `validator/validator_test.go`

**Validation Method**:
- Run: `go test ./validator -cover -v`
- Create malformed test files for each error type
- Verify SC-004: 95% of common malformations detected
- Verify SC-005: 100% of required field/XRef violations caught

---

### Phase 5: Encoding (encoder)

**Criteria**:
- [ ] Generates valid GEDCOM output per FR-014
- [ ] Output passes validation for target version per FR-014
- [ ] Supports all character encodings (UTF-8, ANSEL, etc.) per FR-007
- [ ] Handles all line ending formats per FR-018
- [ ] Roundtrip fidelity: parse → encode → parse preserves data
- [ ] Test coverage ≥85%

**Test Requirements**:
- Roundtrip tests: parse file → encode → parse again → compare Documents
- Character encoding: encode with UTF-8, ANSEL, verify readable
- Line endings: CRLF on Windows, LF on Unix, verify consistent
- All record types: INDI, FAM, SOUR, REPO, NOTE, OBJE, SUBM
- Generated output validates without errors

**Roundtrip Validation**:
- 100% data preservation for version-compatible fields per SC-006
- Every tag/value/XRef matches after roundtrip

**Deliverables**:
- [ ] `encoder/encoder.go` implementing Encoder interface
- [ ] `encoder/options.go` with EncodeOptions
- [ ] `encoder/encoder_test.go` with roundtrip tests

**Validation Method**:
- Run: `go test ./encoder -cover -v`
- Roundtrip all testdata files
- Verify SC-006: 100% data preservation
- Validate generated output with validator package

---

### Phase 6: Examples & Documentation

**Criteria**:
- [ ] All examples from quickstart.md implemented and tested
- [ ] Examples run without errors
- [ ] README.md created with installation and basic usage
- [ ] All public APIs have godoc comments
- [ ] CHANGELOG.md started for version tracking
- [ ] Developer can integrate library in <30 minutes per SC-009

**Example Programs**:
- Parse GEDCOM file (quickstart.md:16-55)
- Stream large files (quickstart.md:107-149)
- Validate data (quickstart.md:152-203)
- Write GEDCOM (quickstart.md:267-320)

**Documentation Requirements**:
- Every exported type has godoc comment
- Every exported function has godoc comment with example
- README covers: installation, quick start, features, examples link
- CHANGELOG follows semantic versioning

**Deliverables**:
- [ ] `examples/parse/main.go`
- [ ] `examples/validate/main.go`
- [ ] `examples/convert/main.go`
- [ ] `README.md`
- [ ] `CHANGELOG.md`
- [ ] Package-level godoc for all packages

**Validation Method**:
- Run: `go run examples/parse/main.go`
- Run: `go run examples/validate/main.go`
- Run: `go run examples/convert/main.go`
- Check: `go doc -all` shows comprehensive documentation
- User test: Give library to developer, time integration (should be <30 min per SC-009)

---

### Final Acceptance Criteria

Before considering the implementation complete, ALL of these must pass:

**Functional**:
- [ ] All functional requirements (FR-001 through FR-025) implemented and tested
- [ ] All success criteria (SC-001 through SC-012) verified
- [ ] All user stories (US1-US4) have passing acceptance tests
- [ ] Constitution compliance verified (Library-First, API Clarity, Test Coverage ≥85%, Version Support, Error Transparency)

**Quality**:
- [ ] Overall test coverage ≥85% (run: `go test ./... -cover`)
- [ ] All tests pass (run: `go test ./...`)
- [ ] No panics in any code path
- [ ] All benchmarks meet performance targets (SC-002, SC-003, SC-008)
- [ ] Code passes `go fmt ./...` (no formatting issues)
- [ ] Code passes `go vet ./...` (no suspicious constructs)
- [ ] Code passes `staticcheck ./...` if installed (no linter warnings)

**Documentation**:
- [ ] All public APIs documented with godoc
- [ ] README.md complete and accurate
- [ ] Examples run successfully
- [ ] CHANGELOG.md started

**Integration**:
- [ ] Library can be imported with `go get`
- [ ] No external dependencies (only stdlib)
- [ ] Works on Linux, macOS, Windows
- [ ] Integration test: parse official GEDCOM test files (SC-010)

---

## Future Improvements

The following features were deferred from the initial implementation to reduce complexity and accelerate MVP delivery. These may be added in future versions once the core library is stable and proven.

### Version 2.0 Candidates

#### Progress Callbacks (originally FR-023)
**Description**: Optional callbacks that report parsing progress (percentage complete, records processed).

**Original Requirements**:
- Report every 1% or 100 records (whichever comes first)
- No overhead when callbacks disabled
- Pass to decoder via `DecodeOptions`

**Why Deferred**:
- Not required by constitution
- Adds API complexity
- Most users won't need progress tracking for typical files (<50MB)
- Can be added without breaking changes by enhancing `DecodeOptions`

**Estimated Effort**: 1-2 days

**Implementation Notes**:
- Add `ProgressFunc func(percent int, recordCount int)` to `DecodeOptions`
- Call at regular intervals during decoding
- Use atomic counters to avoid lock overhead

---

#### ANSEL Character Encoding Support (FR-007 partial)
**Description**: Support for ANSEL (ANSI Z39.47) character encoding used in legacy GEDCOM 5.5 files.

**Original Requirements**:
- Bidirectional ANSEL ↔ UTF-8 conversion
- Handle combining characters with order reversal
- Roundtrip fidelity

**Why Deferred**:
- Complex implementation (2-3 days for lookup tables and combining character logic)
- GEDCOM 7.0 mandates UTF-8 only
- Most modern genealogy software exports UTF-8
- Research already completed in research.md:319-387

**Estimated Effort**: 2-3 days

**Implementation Notes**:
- Create `charset/ansel.go` with lookup tables from research.md:346-378
- Implement combining character order reversal (research.md:379-383)
- Add ANSEL detection to charset package
- Comprehensive test suite for all ANSEL code points

---

#### Latin-1 Fallback (FR-021)
**Description**: Attempt UTF-8 → Latin-1 (ISO-8859-1) fallback for files without declared encoding.

**Original Requirements**:
- Try UTF-8 first, fall back to Latin-1 if invalid
- Return error if both fail
- Auto-detection based on byte patterns

**Why Deferred**:
- Most files declare encoding in header
- UTF-8 is backward compatible with ASCII (covers majority of cases)
- Can be added as enhancement to charset package

**Estimated Effort**: 0.5-1 day

**Implementation Notes**:
- Enhance `charset.DetectEncoding()` with Latin-1 fallback
- Use UTF-8 validation to trigger fallback
- Add tests with mixed-encoding files

---

### Separate Library Candidates

#### GEDCOM Version Converter (originally FR-011, FR-012, User Story 4)
**Description**: Convert GEDCOM files between versions (5.5 ↔ 5.5.1 ↔ 7.0) with transformation reporting.

**Original Requirements**:
- Bidirectional conversion support
- Document all transformations in `ConversionReport`
- Report data loss when features don't translate
- Validate converted output against target version

**Why Deferred/Separated**:
- Priority P4 (lowest user story priority)
- Complex implementation (4-6 days effort)
- Depends on parser, validator, and encoder being stable
- Could be a separate library built on top of `go-gedcom`
- Not required by constitution

**Estimated Effort**: 4-6 days (if added to this library) or separate project

**Implementation Notes** (if added later):
- Create `converter/` package
- Implement version-specific mapping files:
  - `v55_to_v551.go`: Tag additions (EMAIL, WWW, MAP, LATI, LONG)
  - `v551_to_v70.go`: UTF-8 mandatory, CONC removal, strict XRef format
  - `v70_to_v55.go`: Downgrade with graceful feature loss
- Return `ConversionReport` struct documenting all changes
- Validation pass after conversion
- 100% data preservation for compatible fields (SC-006)

**Alternative**: Create `go-gedcom-converter` as separate repository that imports `go-gedcom` for parsing/validation, providing focused conversion tooling.

---

### Enhancement Ideas (No Specific Timeline)

#### Streaming Validation
**Description**: Validate GEDCOM structure while parsing (single-pass) rather than parse-then-validate.

**Benefits**: Lower memory usage, faster feedback for invalid files

**Estimated Effort**: 2-3 days (refactor validator to work incrementally)

---

#### Custom Tag Registry
**Description**: Allow users to register custom/proprietary tag definitions for validation.

**Benefits**: Support for genealogy software extensions (e.g., Ancestry.com, FamilySearch custom tags)

**Estimated Effort**: 1-2 days

---

#### Performance Profiling Tools
**Description**: Built-in profiling hooks to identify bottlenecks in user applications.

**Benefits**: Help users optimize their GEDCOM processing pipelines

**Estimated Effort**: 1 day

---

#### Binary GEDCOM Support (GEDCOM-X)
**Description**: Support for GEDCOM-X (JSON-based format) as alternative to line-based GEDCOM.

**Benefits**: Modernize data exchange format

**Estimated Effort**: 5-7 days (new parser, but reuse data model)

**Note**: GEDCOM-X is not yet widely adopted; assess demand before implementing.

---

## Revision History

- **2025-10-16**: Initial plan created with full feature set
- **2025-10-16**: Simplified to remove over-engineering; moved converter, progress callbacks, and extended encoding to future improvements (41% effort reduction)
