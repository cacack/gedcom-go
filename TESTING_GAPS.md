# Testing Gaps Analysis

Generated: 2025-10-21
Current Coverage: 96.7% (library packages)

## Executive Summary

The codebase has excellent test coverage at 96.7%, but there are specific gaps that should be addressed for completeness and robustness.

## Critical Gaps (0% Coverage)

### 1. ParseError.Unwrap() - **PRIORITY HIGH**
**File**: `parser/errors.go:28`
**Coverage**: 0.0%

```go
func (e *ParseError) Unwrap() error {
    return e.Err
}
```

**Issue**: Error unwrapping is never tested, yet it's part of the public API.
**Impact**: Code using `errors.Is()` or `errors.As()` with wrapped parse errors is untested.
**Fix**: Add test in `parser/error_test.go`:
```go
func TestParseErrorUnwrap(t *testing.T) {
    baseErr := fmt.Errorf("base error")
    parseErr := wrapParseError(1, "wrapped", "context", baseErr)

    if !errors.Is(parseErr, baseErr) {
        t.Error("ParseError should unwrap to base error")
    }
}
```

## Important Gaps (>80% but <100%)

### 2. ValidationError.Error() Formatting
**File**: `validator/validator.go:33`
**Coverage**: 80.0%

**Missing Cases**:
- ValidationError with both XRef and Line number
- Edge case: empty Code field

**Fix**: Add comprehensive error formatting tests:
```go
func TestValidationErrorFormatting(t *testing.T) {
    tests := []struct {
        name string
        err  ValidationError
        want string
    }{
        {
            name: "with XRef only",
            err:  ValidationError{Code: "ERR1", Message: "test", XRef: "@I1@"},
            want: "[ERR1] test (XRef: @I1@)",
        },
        {
            name: "with Line only",
            err:  ValidationError{Code: "ERR2", Message: "test", Line: 42},
            want: "[ERR2] line 42: test",
        },
        {
            name: "with both XRef and Line",
            err:  ValidationError{Code: "ERR3", Message: "test", XRef: "@I1@", Line: 42},
            want: "[ERR3] test (XRef: @I1@)", // XRef takes precedence
        },
        {
            name: "minimal error",
            err:  ValidationError{Code: "ERR4", Message: "test"},
            want: "[ERR4] test",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.err.Error()
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### 3. Context Cancellation in Decoder
**File**: `decoder/decoder.go:40`
**Coverage**: 83.3%

**Missing**: Test for pre-parse context cancellation

**Fix**: Add to `decoder/decoder_test.go`:
```go
func TestDecodeWithCancelledContext(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel before starting

    opts := &DecodeOptions{Context: ctx}

    gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5\n0 TRLR\n"
    _, err := DecodeWithOptions(strings.NewReader(gedcom), opts)

    if !errors.Is(err, context.Canceled) {
        t.Errorf("expected context.Canceled, got %v", err)
    }
}
```

### 4. Encoder Edge Cases
**File**: `encoder/encoder.go:98` (writeRecord), `encoder/encoder.go:120` (writeTag)
**Coverage**: 88.9% and 83.3%

**Likely Missing**:
- Encoding records with no tags
- Encoding tags with special characters in values
- Encoding very long values (>255 chars)
- CONT/CONC line continuation

**Investigation Needed**: Run coverage HTML to see exact lines.

### 5. Validator Family Validation
**File**: `validator/validator.go:127`
**Coverage**: 85.7%

**Likely Missing**:
- Families with only children (no parents)
- Families with only one parent
- Families with duplicate children references
- Edge cases in family event validation

## Minor Gaps (>90% but <100%)

### 6. Charset UTF-8 Validation
**File**: `charset/charset.go:116` (findInvalidUTF8)
**Coverage**: 90.0%

**Likely Missing**: Edge cases in multi-byte UTF-8 sequences

### 7. Parser Edge Cases
**Files**:
- `parser/parser.go:61` (ParseLine) - 96.9%
- `parser/parser.go:134` (Parse) - 91.7%

**Likely Missing**:
- Maximum line length enforcement
- Unusual whitespace handling
- Buffer overflow protection

### 8. Decoder Header Building
**File**: `decoder/decoder.go:108` (buildHeader)
**Coverage**: 93.8%

**Likely Missing**:
- Headers with missing required fields
- Headers with malformed GEDC structure
- Headers with unusual character encoding declarations

## Testing Scenarios Not Covered

### Integration Test Gaps

1. **Large File Stress Testing**
   - Files > 10MB
   - Files with > 100,000 individuals
   - Deep nesting (> 50 levels)

2. **Concurrent Access**
   - Multiple goroutines parsing different files
   - Shared validator usage
   - Thread safety of Document methods

3. **Memory Pressure**
   - Parsing under low memory conditions
   - Memory leak detection
   - Cleanup of resources

4. **Real-World Files**
   - Files from popular genealogy software:
     - Ancestry.com exports
     - FamilySearch GEDCOM
     - MyHeritage exports
     - Legacy Family Tree
     - RootsMagic
   - Files with known quirks/bugs from other software

5. **Error Recovery**
   - Partial file parsing
   - Continuing after non-fatal errors
   - Best-effort parsing mode

6. **Character Encoding Edge Cases**
   - Mixed encoding within file (non-compliant but real)
   - Invalid ANSEL sequences
   - UTF-8 with unusual BOM variants
   - Files with encoding declaration mismatch

### Performance Test Gaps

1. **Benchmark Coverage**
   - ✓ Parser benchmarks exist
   - ✓ Decoder benchmarks exist
   - ✓ Encoder benchmarks exist
   - ✗ Validator benchmarks missing
   - ✗ Character set conversion benchmarks missing

2. **Performance Regression Testing**
   - No baseline performance metrics
   - No automated performance regression detection

### Fuzz Testing

**Currently Missing**: Fuzz tests for all parsers

Recommended fuzz targets:
```go
func FuzzParseLine(f *testing.F) {
    f.Add("0 HEAD")
    f.Add("1 NAME John /Doe/")
    f.Add("0 @I1@ INDI")

    f.Fuzz(func(t *testing.T, input string) {
        p := parser.New(strings.NewReader(input))
        _, _ = p.ParseLine()
        // Should not panic
    })
}
```

## Recommendations

### Immediate (High Priority)
1. ✅ Add ParseError.Unwrap() test
2. ✅ Add ValidationError formatting tests
3. ✅ Add context cancellation test
4. ⚠️ Add validator benchmarks

### Short Term (Medium Priority)
5. 📊 Generate HTML coverage report to identify exact missing lines
6. 📝 Add encoder edge case tests (empty records, long values, CONT/CONC)
7. 🏗️ Add family validator edge cases
8. 🌐 Add real-world GEDCOM files to test suite

### Long Term (Nice to Have)
9. 🔥 Implement fuzz testing
10. 📈 Set up performance regression testing
11. 🧵 Add concurrency tests
12. 💾 Add large file stress tests
13. 🔍 Add integration tests with popular genealogy software exports

## Coverage Target

**Current**: 96.7%
**Target**: 98.0%
**Stretch Goal**: 99.0%

## Quick Wins

These tests can be added quickly to boost coverage:

1. **5 minutes**: ParseError.Unwrap() test
2. **10 minutes**: ValidationError formatting tests
3. **10 minutes**: Context cancellation tests
4. **15 minutes**: Validator benchmarks
5. **30 minutes**: Encoder edge case tests

**Total Time**: ~70 minutes to get to ~97.5% coverage

## Notes

- Examples directory (0% coverage) is excluded from metrics - this is intentional
- Specs directory (no tests) is excluded from metrics - this is intentional
- Focus should be on testing public API surface and error paths
- Integration tests with real-world files would catch issues unit tests miss
