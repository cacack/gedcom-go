# Testing Guide

This document defines testing requirements and critical paths that must have test coverage.

## Coverage Requirements

| Level | Threshold | Enforcement |
|-------|-----------|-------------|
| Per-package | ≥85% | Pre-commit hook, CI |
| Total | ≥85% | CI |
| Critical paths | 100% | Code review |

## Critical Paths (Must Have Tests)

These code paths handle core functionality where bugs would cause data loss or corruption. They require 100% test coverage and explicit edge case testing.

### Parser (`parser/`)

| Function | Why Critical | Required Tests |
|----------|--------------|----------------|
| `ParseLine()` | Entry point for all GEDCOM data | Valid lines, malformed lines, edge cases |
| `ParseLevel()` | Hierarchy determines structure | 0-99, invalid, overflow |
| `ParseXRef()` | Cross-references link records | Valid `@ID@`, malformed, empty |
| `ParseTag()` | Tags determine semantics | All standard tags, custom tags, case sensitivity |

### Decoder (`decoder/`)

| Function | Why Critical | Required Tests |
|----------|--------------|----------------|
| `Decode()` | Main entry point | Valid files, malformed, large files |
| `DecodeIndividual()` | Most common record type | All INDI substructures |
| `DecodeFamily()` | Relationship data | FAM links, children, events |
| `resolveReferences()` | Data integrity | Valid refs, broken refs, circular refs |

### Encoder (`encoder/`)

| Function | Why Critical | Required Tests |
|----------|--------------|----------------|
| `Encode()` | Output generation | Round-trip fidelity |
| `EncodeRecord()` | Record serialization | All record types |
| `escapeValue()` | Prevents injection | Special characters, newlines, @ symbols |

### Date Parsing (`gedcom/date.go`)

| Function | Why Critical | Required Tests |
|----------|--------------|----------------|
| `ParseDate()` | Date interpretation | All formats from GEDCOM spec |
| `Validate()` | Data integrity | Invalid dates, leap years, edge cases |
| `Compare()` | Sorting/ordering | Partial dates, BC dates, ranges |
| Dual dating | Historical accuracy | `1750/51` format |
| BC dates | Chronological ordering | `44 BC`, comparison |
| Date phrases | Lossless representation | `(unknown)`, `(about 1850)` |

### Validator (`validator/`)

| Function | Why Critical | Required Tests |
|----------|--------------|----------------|
| `Validate()` | Error detection | All error types |
| `ValidateStructure()` | Hierarchy correctness | Level violations |
| `ValidateReferences()` | Link integrity | Missing refs, duplicates |

### Character Encoding (`charset/`)

| Function | Why Critical | Required Tests |
|----------|--------------|----------------|
| `Detect()` | Encoding detection | UTF-8, UTF-16, BOM detection |
| `Decode()` | Character conversion | All supported encodings |

## Test Patterns

### Table-Driven Tests (Required)

All parsing functions must use table-driven tests to explicitly enumerate edge cases:

```go
func TestParseDate(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Date
        wantErr bool
    }{
        // Happy paths
        {"exact date", "25 DEC 2020", &Date{Day: 25, Month: 12, Year: 2020}, false},
        {"year only", "1850", &Date{Year: 1850}, false},

        // Edge cases
        {"leap year valid", "29 FEB 2000", &Date{Day: 29, Month: 2, Year: 2000}, false},
        {"leap year invalid", "29 FEB 1900", nil, true},

        // Error cases
        {"empty", "", nil, true},
        {"invalid month", "25 XXX 2020", nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseDate(tt.input)
            // assertions...
        })
    }
}
```

### Round-Trip Tests (Required for Encoder)

Any data that can be encoded must survive a decode→encode→decode cycle:

```go
func TestRoundTrip(t *testing.T) {
    original := loadTestFile("testdata/sample.ged")

    decoded, _ := decoder.Decode(original)
    encoded := encoder.Encode(decoded)
    redecoded, _ := decoder.Decode(encoded)

    assertEqual(t, decoded, redecoded)
}
```

### Error Message Tests (Required)

Error messages must be tested to ensure they contain actionable information:

```go
func TestValidateErrorMessages(t *testing.T) {
    _, err := ParseDate("30 FEB 2020")

    require.Error(t, err)
    assert.Contains(t, err.Error(), "February")
    assert.Contains(t, err.Error(), "28") // or "29" context
}
```

## Adding New Features

When adding new functionality:

1. **Write tests first** (TDD approach per constitution)
2. **Cover the critical path** - What's the main success scenario?
3. **Cover edge cases** - What inputs are unusual but valid?
4. **Cover error cases** - What should fail and how?
5. **Check coverage** - Run `go test -cover ./package`

### Checklist for New Features

- [ ] Unit tests for all exported functions
- [ ] Table-driven tests for parsing logic
- [ ] Error case tests with message validation
- [ ] Integration test with real GEDCOM data (if applicable)
- [ ] Benchmark test (if performance-sensitive)
- [ ] Coverage ≥85% for the package

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Check specific package coverage
go test -cover ./gedcom

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./...
```

## CI Coverage Enforcement

The CI pipeline enforces coverage via:

1. **go-test-coverage action** - Fails if any package < 85%
2. **Codecov integration** - Historical tracking and PR comments
3. **Per-package report** - Shows exactly which packages need attention

If CI fails due to coverage:

1. Check the coverage report in the CI output
2. Identify uncovered lines with `go tool cover -html=coverage.out`
3. Add tests for critical paths first
4. Focus on branches (if/else) not just lines
