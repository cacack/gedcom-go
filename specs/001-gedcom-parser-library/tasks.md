# Tasks: GEDCOM Parser Library

**Input**: Design documents from `/specs/001-gedcom-parser-library/`
**Prerequisites**: plan.md (complete), spec.md (complete), research.md (complete), data-model.md (complete), contracts/interfaces.go (complete)

**Tests**: This project follows TDD approach with ≥85% test coverage requirement (per constitution). All test tasks must be completed BEFORE implementation tasks.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Note that User Story 4 (Version Conversion) has been deferred to future improvements.

## Format: `[ID] [P?] [Story] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions
- Go library structure: packages at repository root
- Test files: `*_test.go` alongside source files
- Examples: `examples/` directory
- Test data: `testdata/` directory with GEDCOM sample files

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Initialize Go module with `go mod init github.com/user/go-gedcom` (adjust org/repo as needed)
- [ ] T002 [P] Create directory structure: `charset/`, `gedcom/`, `parser/`, `version/`, `decoder/`, `validator/`, `encoder/`, `examples/`, `testdata/`
- [ ] T003 [P] Create README.md with project description and installation instructions
- [ ] T004 [P] Create CHANGELOG.md for tracking version changes
- [ ] T005 [P] Create LICENSE file (choose appropriate open source license)
- [ ] T006 [P] Setup `.gitignore` for Go projects (exclude binaries, coverage files, IDE files)
- [ ] T007 [P] Collect test data: Download sample GEDCOM files for `testdata/gedcom-5.5/`, `testdata/gedcom-5.5.1/`, `testdata/gedcom-7.0/`, `testdata/malformed/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and utilities that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Core Types (Layer 1)

- [ ] T008 [P] Define Version enum (V5_5, V5_5_1, V7_0) in `gedcom/version.go`
- [ ] T009 [P] Define Encoding enum (UTF8, ANSEL, ASCII, LATIN1) in `gedcom/encoding.go`
- [ ] T010 [P] Create Header struct in `gedcom/header.go` with Version, Encoding, SourceSystem, Date, Language fields
- [ ] T011 [P] Create Trailer struct in `gedcom/trailer.go`
- [ ] T012 [P] Create Tag struct in `gedcom/tag.go` with Level, Tag, Value, XRef, LineNumber fields
- [ ] T013 [P] Create Record struct in `gedcom/record.go` with XRef, Type, Tags fields
- [ ] T014 [P] Create Individual struct in `gedcom/individual.go` with Names, Events, Attributes, Families
- [ ] T015 [P] Create Family struct in `gedcom/family.go` with Spouses, Children, MarriageEvents
- [ ] T016 [P] Create Source struct in `gedcom/source.go` with Title, Author, Publication fields
- [ ] T017 [P] Create Repository struct in `gedcom/repository.go` with Name, Address fields
- [ ] T018 [P] Create Event struct in `gedcom/event.go` with Type, Date, Place, Sources
- [ ] T019 [P] Create Note struct in `gedcom/note.go` with Text, Continuation fields
- [ ] T020 [P] Create MediaObject struct in `gedcom/media.go` with FileRef, Format, Title
- [ ] T021 Create Document struct in `gedcom/document.go` with Header, Records, Trailer, Version, XRefMap
- [ ] T022 [P] Write unit tests for all gedcom types in `gedcom/types_test.go`

### Character Encoding (Layer 1)

- [ ] T023 Create charset package with UTF-8 detection/validation in `charset/charset.go`
- [ ] T024 Implement NewReader() function that wraps io.Reader with UTF-8 validation in `charset/charset.go`
- [ ] T025 [P] Write table-driven tests for UTF-8 validation (valid sequences, invalid sequences, BOM handling) in `charset/charset_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Parse Valid GEDCOM File (Priority: P1) 🎯 MVP

**Goal**: Enable developers to parse valid GEDCOM files (5.5, 5.5.1, 7.0) and access all records via a simple API

**Independent Test**: Provide valid GEDCOM file → parse → verify all records accessible with correct data

### Tests for User Story 1 (TDD Approach - Write First)

**NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T026 [P] [US1] Write table-driven tests for line parsing (valid lines, edge cases, line endings) in `parser/parser_test.go`
- [ ] T027 [P] [US1] Write tests for GEDCOM 5.5 parsing with sample file from `testdata/gedcom-5.5/` in `parser/parser_test.go`
- [ ] T028 [P] [US1] Write tests for GEDCOM 5.5.1 parsing with sample file from `testdata/gedcom-5.5.1/` in `parser/parser_test.go`
- [ ] T029 [P] [US1] Write tests for GEDCOM 7.0 parsing with sample file from `testdata/gedcom-7.0/` in `parser/parser_test.go`
- [ ] T030 [P] [US1] Write tests for version detection (header-based and tag-based fallback) in `version/version_test.go`
- [ ] T031 [P] [US1] Write integration tests for full document parsing in `decoder/decoder_test.go`
- [ ] T032 [P] [US1] Write tests for XRefMap resolution (all XRefs indexed, valid lookups) in `decoder/decoder_test.go`
- [ ] T033 [P] [US1] Write benchmark tests for 10MB file parsing (must complete <2s) in `decoder/decoder_test.go`
- [ ] T034 [P] [US1] Write benchmark tests for 100MB file memory usage (must be <200MB) in `decoder/decoder_test.go`

### Implementation for User Story 1 (Parser - Layer 2)

- [ ] T035 [US1] Create Line struct in `parser/line.go` with Level, Tag, Value, XRef, LineNumber fields
- [ ] T036 [US1] Implement Lexer in `parser/lexer.go` for tokenizing GEDCOM lines (handle CRLF, LF, CR line endings per FR-018)
- [ ] T037 [US1] Implement Parser interface in `parser/parser.go` with ParseLine() and Reset() methods
- [ ] T038 [US1] Add line number tracking to parser in `parser/parser.go` (for error reporting per FR-017)
- [ ] T039 [US1] Add inline depth checking (max 100 levels) to parser in `parser/parser.go`
- [ ] T040 [US1] Create ParseError type with line number and context in `parser/errors.go` (per FR-006, FR-015)
- [ ] T041 [US1] Implement error wrapping with context in `parser/parser.go` (use Go 1.13+ errors.Wrap)

### Implementation for User Story 1 (Version Detection - Layer 2)

- [ ] T042 [P] [US1] Define GEDCOM 5.5 tag list in `version/v55.go`
- [ ] T043 [P] [US1] Define GEDCOM 5.5.1 tag list in `version/v551.go`
- [ ] T044 [P] [US1] Define GEDCOM 7.0 tag list in `version/v70.go`
- [ ] T045 [US1] Implement version detection from header in `version/detect.go` (per FR-002)
- [ ] T046 [US1] Implement tag-based version fallback heuristics in `version/detect.go`

### Implementation for User Story 1 (Decoder - Layer 3)

- [ ] T047 [US1] Create DecodeOptions struct in `decoder/options.go` with Context, MaxNestingDepth, StrictMode
- [ ] T048 [US1] Implement Decoder interface with Decode() method in `decoder/decoder.go`
- [ ] T049 [US1] Implement DecodeWithOptions() method in `decoder/decoder.go` (uses context for timeout)
- [ ] T050 [US1] Build Document from line stream in `decoder/decoder.go` (parse lines → build records)
- [ ] T051 [US1] Build XRefMap during decoding in `decoder/decoder.go` (index all cross-references per FR-005)
- [ ] T052 [US1] Implement streaming mode for large files in `decoder/decoder.go` (use io.Reader, constant memory per FR-016)
- [ ] T053 [US1] Handle empty/header-only files in `decoder/decoder.go` (return empty Records per FR-019)
- [ ] T054 [US1] Add timeout support via context.Context in `decoder/decoder.go`

### Verification for User Story 1

- [ ] T055 [US1] Run all tests: `go test ./parser ./version ./decoder -v`
- [ ] T056 [US1] Verify test coverage ≥85%: `go test ./parser ./version ./decoder -cover`
- [ ] T057 [US1] Run benchmarks and verify performance targets: `go test ./decoder -bench=. -benchmem`
- [ ] T058 [US1] Parse all sample files in `testdata/` and verify success
- [ ] T059 [US1] Verify SC-001: Can parse file in <5 lines of code (create simple test program)
- [ ] T060 [US1] Verify SC-002: ≥10,000 records/second parsing rate
- [ ] T061 [US1] Verify SC-008: 10MB file parses in <2 seconds

**Checkpoint**: At this point, User Story 1 should be fully functional - can parse valid GEDCOM files independently

---

## Phase 4: User Story 2 - Handle Malformed GEDCOM Files (Priority: P2)

**Goal**: Provide clear, actionable error messages for malformed files with line numbers and context

**Independent Test**: Provide intentionally malformed GEDCOM file → parse → verify specific, helpful error messages with line numbers

### Tests for User Story 2 (TDD Approach - Write First)

- [ ] T062 [P] [US2] Write tests for invalid tag errors in `parser/parser_test.go` (expect "Invalid tag 'XYZ' at line 45")
- [ ] T063 [P] [US2] Write tests for hierarchy level errors in `parser/parser_test.go` (mismatched nesting)
- [ ] T064 [P] [US2] Write tests for missing cross-reference targets in `decoder/decoder_test.go` (broken XRef)
- [ ] T065 [P] [US2] Write tests for encoding errors in `charset/charset_test.go` (invalid UTF-8 sequences)
- [ ] T066 [P] [US2] Write tests for malformed files in `testdata/malformed/` directory
- [ ] T067 [P] [US2] Write tests for partial file recovery (error recovery mode) in `decoder/decoder_test.go`

### Implementation for User Story 2

- [ ] T068 [P] [US2] Enhance ParseError with context snippet in `parser/errors.go` (show surrounding lines)
- [ ] T069 [P] [US2] Add structured error types for common issues in `parser/errors.go` (InvalidTagError, LevelMismatchError, etc.)
- [ ] T070 [US2] Implement graceful error handling in parser (no panics, return errors per FR-015) in `parser/parser.go`
- [ ] T071 [US2] Add error recovery mode to decoder in `decoder/decoder.go` (continue parsing after errors, collect all errors)
- [ ] T072 [US2] Implement broken XRef detection in `decoder/decoder.go` (check XRefMap for dangling references)
- [ ] T073 [US2] Add encoding error detection and reporting in `charset/charset.go`
- [ ] T074 [US2] Enhance error messages with suggestions in `parser/errors.go` (e.g., "expected INDI, FAM, or SOUR")

### Verification for User Story 2

- [ ] T075 [US2] Run all error handling tests: `go test ./parser ./decoder ./charset -run TestError -v`
- [ ] T076 [US2] Verify SC-004: 95% of common malformations detected with helpful messages
- [ ] T077 [US2] Verify SC-007: 100% of errors include line numbers and context
- [ ] T078 [US2] Test with malformed files from `testdata/malformed/` and verify error quality
- [ ] T079 [US2] Verify no panics occur with any malformed input (fuzz testing recommended)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - parsing succeeds OR fails with clear errors

---

## Phase 5: User Story 3 - Validate GEDCOM Against Specification (Priority: P3)

**Goal**: Validate GEDCOM data against version-specific rules (required fields, formats, relationships)

**Independent Test**: Provide GEDCOM data with specific violations → validate → verify correct validation errors reported

### Tests for User Story 3 (TDD Approach - Write First)

- [ ] T080 [P] [US3] Write tests for missing required fields in `validator/validator_test.go` (Individual without NAME)
- [ ] T081 [P] [US3] Write tests for invalid date formats in `validator/validator_test.go` ("32 JAN 2020")
- [ ] T082 [P] [US3] Write tests for broken cross-references in `validator/validator_test.go`
- [ ] T083 [P] [US3] Write tests for circular family relationships in `validator/validator_test.go`
- [ ] T084 [P] [US3] Write tests for non-standard XRef format warnings in `validator/validator_test.go` (@I-001@ triggers warning)
- [ ] T085 [P] [US3] Write tests for version-specific validation in `validator/validator_test.go` (7.0 using 5.5 deprecated tags)
- [ ] T086 [P] [US3] Write tests for valid GEDCOM structures (no errors) in `validator/validator_test.go`

### Implementation for User Story 3

- [ ] T087 [P] [US3] Create ValidationError type with line numbers in `validator/errors.go`
- [ ] T088 [P] [US3] Define validation error codes in `validator/errors.go` (MISSING_REQUIRED_FIELD, INVALID_DATE, BROKEN_XREF, CIRCULAR_REFERENCE)
- [ ] T089 [US3] Create Validator interface in `validator/validator.go` with Validate(doc *Document) method
- [ ] T090 [US3] Implement required field validation in `validator/rules.go` (check NAME for Individual, etc.)
- [ ] T091 [US3] Implement date format validation in `validator/rules.go` (check day/month/year validity per FR-010)
- [ ] T092 [US3] Implement cross-reference validation in `validator/rules.go` (all XRefs resolve via XRefMap)
- [ ] T093 [US3] Implement circular relationship detection in `validator/rules.go` (person as own ancestor)
- [ ] T094 [US3] Implement non-standard XRef format warnings in `validator/rules.go` (per FR-022)
- [ ] T095 [P] [US3] Create GEDCOM 5.5 validation rules in `validator/v55_rules.go`
- [ ] T096 [P] [US3] Create GEDCOM 5.5.1 validation rules in `validator/v551_rules.go`
- [ ] T097 [P] [US3] Create GEDCOM 7.0 validation rules in `validator/v70_rules.go`
- [ ] T098 [US3] Implement version-specific validation dispatch in `validator/validator.go`

### Verification for User Story 3

- [ ] T099 [US3] Run all validation tests: `go test ./validator -v`
- [ ] T100 [US3] Verify test coverage ≥85%: `go test ./validator -cover`
- [ ] T101 [US3] Verify SC-005: 100% of required field/XRef violations caught
- [ ] T102 [US3] Create invalid GEDCOM test files and verify each error type is caught
- [ ] T103 [US3] Validate all sample files from `testdata/` and verify version-specific rules

**Checkpoint**: All three user stories should now be independently functional - parse, error handling, and validation complete

---

## Phase 6: Encoding Support (Enables Roundtrip Testing)

**Purpose**: Write GEDCOM data back to files (required for roundtrip tests, not a separate user story)

**Goal**: Enable writing parsed GEDCOM data back to output stream with correct formatting

### Tests for Encoding (TDD Approach - Write First)

- [ ] T104 [P] Write tests for encoding all record types in `encoder/encoder_test.go`
- [ ] T105 [P] Write roundtrip tests (parse → encode → parse → compare) in `encoder/encoder_test.go`
- [ ] T106 [P] Write tests for line ending formats (CRLF, LF) in `encoder/encoder_test.go`
- [ ] T107 [P] Write tests for character encoding (UTF-8) in `encoder/encoder_test.go`

### Implementation for Encoding

- [ ] T108 [P] Create EncodeOptions struct in `encoder/options.go` with LineEnding, Encoding options
- [ ] T109 Create Encoder interface in `encoder/encoder.go` with Encode(w io.Writer, doc *Document) method
- [ ] T110 Implement Encoder.Encode() in `encoder/encoder.go` (write Header, Records, Trailer)
- [ ] T111 Implement tag formatting in `encoder/encoder.go` (level, tag, value, line endings per FR-018)
- [ ] T112 Implement cross-reference formatting in `encoder/encoder.go` (@XREF@ format)
- [ ] T113 Add UTF-8 encoding support in `encoder/encoder.go`

### Verification for Encoding

- [ ] T114 Run all encoding tests: `go test ./encoder -v`
- [ ] T115 Verify test coverage ≥85%: `go test ./encoder -cover`
- [ ] T116 Verify SC-006: Roundtrip preserves 100% of data for all sample files
- [ ] T117 Verify FR-014: Generated output validates successfully

**Checkpoint**: Encoding complete - can now do full roundtrip (parse → encode → parse)

---

## Phase 7: Examples & Documentation

**Purpose**: Demonstrate library usage and complete documentation

- [ ] T118 [P] Create basic parsing example in `examples/parse/main.go` (read file, print record count)
- [ ] T119 [P] Create streaming example in `examples/stream/main.go` (process large file with callback)
- [ ] T120 [P] Create validation example in `examples/validate/main.go` (validate and report errors)
- [ ] T121 [P] Add package-level godoc to all packages (parser, decoder, encoder, validator, etc.)
- [ ] T122 [P] Add godoc comments to all exported types and functions
- [ ] T123 [P] Add godoc examples for key functions (Decode, Validate, Encode)
- [ ] T124 Update README.md with installation, quick start, and feature overview
- [ ] T125 Update README.md with links to examples and documentation
- [ ] T126 [P] Add CONTRIBUTING.md with development setup and testing instructions
- [ ] T127 Update CHANGELOG.md with initial v0.1.0 release notes

**Checkpoint**: Documentation complete - library is ready for external use

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final quality improvements and constitution compliance verification

- [ ] T128 Run `go fmt ./...` to format all code
- [ ] T129 Run `go vet ./...` and fix all issues
- [ ] T130 Run full test suite: `go test ./...` and verify all tests pass
- [ ] T131 Verify overall test coverage ≥85%: `go test ./... -cover`
- [ ] T132 Run benchmarks and verify all performance targets met: `go test ./... -bench=. -benchmem`
- [ ] T133 [P] Run staticcheck if available: `staticcheck ./...`
- [ ] T134 Verify no external dependencies: `go mod graph` (only stdlib)
- [ ] T135 Test on Linux, macOS, and Windows (cross-platform compatibility)
- [ ] T136 Verify SC-009: Developer can integrate library in <30 minutes (user test)
- [ ] T137 Verify SC-010: Library handles all official GEDCOM test files without errors
- [ ] T138 Create git tag for v0.1.0 release
- [ ] T139 Verify constitution compliance: Library-First ✅, API Clarity ✅, Test Coverage ✅, Version Support ✅, Error Transparency ✅

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational completion
- **User Story 2 (Phase 4)**: Depends on Foundational completion (can run parallel to US1 if desired, but builds on US1 parser)
- **User Story 3 (Phase 5)**: Depends on Foundational completion (can run parallel to US1/US2, but validation needs parsed documents)
- **Encoding (Phase 6)**: Depends on US1 completion (needs parser and types)
- **Examples (Phase 7)**: Depends on US1, US3, and Encoding completion
- **Polish (Phase 8)**: Depends on all previous phases completion

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
  - Required for: US2 (builds on parser), US3 (validates parsed docs), Encoding (writes parsed docs)

- **User Story 2 (P2)**: Can start after Foundational (Phase 2)
  - Enhances: US1 parser with better error handling
  - Can be implemented independently but logically extends US1

- **User Story 3 (P3)**: Can start after Foundational (Phase 2)
  - Requires: Parsed documents from US1
  - Independent validation logic, can be tested separately

### Within Each Phase

**Phase 2 (Foundational)**:
- Core types tasks (T008-T022) can all run in parallel
- Charset tasks (T023-T025) can run in parallel with core types
- All foundational tasks must complete before Phase 3

**Phase 3 (User Story 1)**:
- Tests (T026-T034) MUST be written FIRST and should FAIL
- Parser implementation (T035-T041) after tests written
- Version detection (T042-T046) can run parallel with parser
- Decoder (T047-T054) depends on Parser and Version completion
- Verification (T055-T061) runs after all implementation complete

**Phase 4 (User Story 2)**:
- Tests (T062-T067) MUST be written FIRST
- Implementation tasks (T068-T074) can mostly run in parallel
- Verification (T075-T079) runs after implementation

**Phase 5 (User Story 3)**:
- Tests (T080-T086) MUST be written FIRST
- Error types and codes (T087-T088) first
- Version-specific rules (T095-T097) can run in parallel
- Core validation (T089-T094, T098) after rules defined
- Verification (T099-T103) runs after implementation

**Phase 6 (Encoding)**:
- Tests (T104-T107) MUST be written FIRST
- Implementation tasks (T108-T113) run sequentially
- Verification (T114-T117) runs after implementation

**Phase 7 (Examples)**:
- All example and documentation tasks (T118-T127) can run in parallel

**Phase 8 (Polish)**:
- All polish tasks run sequentially (each depends on previous checks passing)

### Parallel Opportunities

**Within Setup (Phase 1)**:
- T003, T004, T005, T006, T007 can all run in parallel after T001, T002

**Within Foundational (Phase 2)**:
- All T008-T020 (type definitions) can run in parallel
- T023-T025 (charset) can run in parallel
- T022 (types test) waits for T008-T021

**Within User Story 1 (Phase 3)**:
- Tests T026-T034 can all be written in parallel
- Version files T042-T044 can be created in parallel
- Verification tasks T055-T061 can run in parallel (different aspects)

**Within User Story 2 (Phase 4)**:
- Tests T062-T067 can all be written in parallel
- Implementation T068-T069 (error types) in parallel
- T073 (charset errors) parallel with parser error work

**Within User Story 3 (Phase 5)**:
- Tests T080-T086 can all be written in parallel
- T087-T088 (error infrastructure) in parallel
- T095-T097 (version rules) in parallel

**Within Encoding (Phase 6)**:
- Tests T104-T107 can all be written in parallel
- T108 (options) parallel with T109 (interface)

**Within Examples (Phase 7)**:
- All T118-T127 can run in parallel

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all User Story 1 tests together (write them first, they should FAIL):
Task T026: "Write table-driven tests for line parsing in parser/parser_test.go"
Task T027: "Write tests for GEDCOM 5.5 parsing in parser/parser_test.go"
Task T028: "Write tests for GEDCOM 5.5.1 parsing in parser/parser_test.go"
Task T029: "Write tests for GEDCOM 7.0 parsing in parser/parser_test.go"
Task T030: "Write tests for version detection in version/version_test.go"
Task T031: "Write integration tests for full document parsing in decoder/decoder_test.go"
Task T032: "Write tests for XRefMap resolution in decoder/decoder_test.go"
Task T033: "Write benchmark tests for 10MB file parsing in decoder/decoder_test.go"
Task T034: "Write benchmark tests for 100MB file memory usage in decoder/decoder_test.go"
```

---

## Parallel Example: User Story 1 Implementation

```bash
# After tests are written and failing, launch parallel implementation:

# Version detection files can be created in parallel:
Task T042: "Define GEDCOM 5.5 tag list in version/v55.go"
Task T043: "Define GEDCOM 5.5.1 tag list in version/v551.go"
Task T044: "Define GEDCOM 7.0 tag list in version/v70.go"

# After parser is complete, these can run in parallel:
Task T047: "Create DecodeOptions struct in decoder/options.go"
Task T048: "Implement Decoder interface in decoder/decoder.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (~1 day)
2. Complete Phase 2: Foundational (~2-3 days)
3. Complete Phase 3: User Story 1 (~4-6 days)
4. **STOP and VALIDATE**: Test US1 independently with all sample files
5. **MVP READY**: Can parse valid GEDCOM files from all three versions

**Timeline**: ~7-10 days for MVP
**Deliverable**: Functional GEDCOM parser library (parsing only, no validation yet)

### Incremental Delivery

1. **Foundation** (Phases 1-2): ~3-4 days → Project structure ready
2. **MVP** (Phase 3): +4-6 days → Can parse valid files ✅
3. **Error Handling** (Phase 4): +2-3 days → Handles malformed files gracefully ✅
4. **Validation** (Phase 5): +3-5 days → Validates against spec ✅
5. **Roundtrip** (Phase 6): +2-3 days → Can write GEDCOM files ✅
6. **Examples** (Phase 7): +1-2 days → Documentation and examples complete ✅
7. **Polish** (Phase 8): +1 day → Ready for v0.1.0 release ✅

**Total Timeline**: ~16-24 days for complete library (aligns with plan.md estimate of 14-22 days)

### Parallel Team Strategy

With multiple developers:

1. **Team completes Setup + Foundational together** (Phases 1-2)
2. **Once Foundational is done, split work**:
   - Developer A: User Story 1 (Phase 3) - Core parsing
   - Developer B: User Story 2 (Phase 4) - Error handling (can start tests in parallel)
   - Developer C: User Story 3 (Phase 5) - Validation (can start defining rules early)
3. **After US1 complete**: Developer A picks up Encoding (Phase 6)
4. **Convergence**: All developers work on Examples and Polish together

---

## Notes

- **[P] tasks**: Different files, no dependencies, can run in parallel
- **[Story] label**: Maps task to specific user story for traceability
- **TDD Approach**: Constitution requires TDD - tests MUST be written first and FAIL before implementation
- **Test Coverage**: Must maintain ≥85% coverage at all times (constitution requirement)
- **Each user story**: Should be independently completable and testable
- **Commit frequently**: After each task or logical group
- **Stop at checkpoints**: Validate story independence before proceeding
- **Performance**: Verify benchmarks meet targets (SC-002, SC-003, SC-008)
- **No panics**: All error paths must return errors, never panic (FR-015)
- **User Story 4 deferred**: Version conversion moved to future improvements per plan.md simplification

---

## Task Count Summary

- **Phase 1 (Setup)**: 7 tasks
- **Phase 2 (Foundational)**: 18 tasks
- **Phase 3 (User Story 1)**: 36 tasks
- **Phase 4 (User Story 2)**: 18 tasks
- **Phase 5 (User Story 3)**: 24 tasks
- **Phase 6 (Encoding)**: 14 tasks
- **Phase 7 (Examples)**: 10 tasks
- **Phase 8 (Polish)**: 12 tasks

**Total**: 139 tasks

**Parallel opportunities**: 62 tasks marked [P] (44% can run in parallel)

**MVP (User Stories 1 only)**: 61 tasks (Phases 1-3)

**Independent test criteria**:
- US1: Parse valid GEDCOM file and access all records correctly
- US2: Parse malformed file and receive specific error with line number
- US3: Validate GEDCOM data and catch spec violations

**Suggested MVP scope**: Complete through Phase 3 (User Story 1) for functional parser
