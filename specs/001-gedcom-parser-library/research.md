# Research: GEDCOM Parser Library

**Feature**: 001-gedcom-parser-library
**Date**: 2025-10-16
**Status**: Complete

## Overview

This document captures research findings and technical decisions for implementing a pure Go GEDCOM parser library. All technical context questions from plan.md have been resolved through this research phase.

## Key Decisions

### 1. Go Version Selection

**Decision**: Go 1.21+

**Rationale**:
- Go 1.21 (released Aug 2023) provides excellent stability and compatibility
- Includes all necessary features: error wrapping (`errors.Is`, `errors.As`), context cancellation, io.Reader/Writer interfaces
- Still widely supported (not cutting-edge, avoiding compatibility issues)
- Built-in support for modern patterns (generics if needed, though likely won't be used for this library)
- Three versions behind current (1.24 as of late 2024), ensuring broad compatibility

**Alternatives Considered**:
- Go 1.18: Too old, missing some stdlib improvements
- Go 1.22+: Too new, could limit adoption for projects on older Go versions
- Go 1.20: Viable alternative, but 1.21 has minor stdlib improvements for error handling

**Impact**: Sets minimum Go version requirement in go.mod

---

### 2. Dependency Strategy

**Decision**: Zero third-party dependencies (Go standard library only)

**Rationale**:
- Aligns with user requirement: "Use of 3rd party modules should be carefully considered and only used when standard libraries won't suffice"
- Go stdlib provides all needed functionality:
  - `bufio`: Buffered I/O for efficient line reading
  - `io`: Stream interfaces (Reader/Writer)
  - `encoding`: Character encoding support (with custom ANSEL codec)
  - `errors`: Error wrapping and inspection
  - `context`: Cancellation and timeouts
  - `unicode/utf8`: UTF-8 validation
  - `time`: Timeout tracking
- Reduces supply chain risk
- Simplifies installation (`go get` with no transitive dependencies)
- No version conflicts with user code

**Alternatives Considered**:
- `golang.org/x/text` for encoding: Considered for ANSEL support, but ANSEL codec is simple enough to implement ourselves (~200 lines)
- Third-party GEDCOM parsers: Rejected because we need full control over error handling, streaming, and version support

**Impact**: All packages must use only `import "..."` for stdlib packages

---

### 3. Character Encoding Strategy

**Decision**: Implement custom ANSEL codec, use stdlib for others

**Rationale**:
- GEDCOM requires support for:
  - ASCII: stdlib (subset of UTF-8)
  - UTF-8: stdlib (`unicode/utf8`)
  - UTF-16: stdlib (`unicode/utf16`)
  - Latin-1 (ISO-8859-1): Simple 1:1 byte mapping (easy to implement)
  - ANSEL: Genealogical-specific encoding (must implement custom)
- ANSEL codec implementation:
  - ANSEL is a single-byte + diacritic combining system used in legacy genealogy software
  - Character set defined in GEDCOM 5.5 spec (fixed, won't change)
  - Implementation: ~200 lines of lookup tables + combining logic
  - Convert ANSEL → UTF-8 for internal processing
- Fallback strategy (from clarifications):
  - If no encoding declared: try UTF-8, fall back to Latin-1, error if both fail
  - UTF-8 validation via `unicode/utf8.Valid()`

**Alternatives Considered**:
- `golang.org/x/text/encoding`: Doesn't include ANSEL, would still need custom codec
- External ANSEL library: None exist with sufficient quality/maintenance

**Impact**: Create `charset/` package with `ansel.go` and `charset.go`

---

### 4. Parsing Architecture

**Decision**: Two-tier architecture: Lexer/Parser → Decoder

**Rationale**:
- **Tier 1 (Lexer/Parser)**:
  - Tokenizes GEDCOM lines into level, tag, value, xref
  - Handles line endings (CRLF, LF, CR)
  - Preserves line numbers for error reporting
  - Stream-based (no buffering entire file)
  - Returns `Line` structs with metadata
- **Tier 2 (Decoder)**:
  - Consumes `Line` stream from parser
  - Builds structured `gedcom.Document` with typed records
  - Resolves cross-references
  - Handles version-specific tag interpretations
  - Provides high-level API users interact with
- Separation allows:
  - Testing parser independently from decoder
  - Alternative decoder implementations (e.g., SAX-style streaming)
  - Easier maintenance (changes to line format don't affect record logic)

**Alternatives Considered**:
- Single-pass parser: Simpler but harder to test, couples line parsing to semantics
- Tree-building parser (DOM-style): Requires full file in memory, violates streaming requirement

**Impact**: Separate `parser/` and `decoder/` packages with clear interfaces

---

### 5. Error Handling Pattern

**Decision**: Structured errors with context wrapping

**Rationale**:
- Every error includes:
  - Line number (preserved from parser)
  - Content context (snippet of problematic line)
  - Error chain (using `fmt.Errorf("context: %w", err)`)
- Error types:
  - `ParseError`: Line-level syntax errors
  - `ValidationError`: Semantic/specification violations
  - `EncodingError`: Character encoding problems
  - `ResourceLimitError`: Resource exhaustion (depth, count, timeout)
- Pattern:
  ```go
  type ParseError struct {
      Line    int
      Content string
      Err     error
  }
  ```
- Never panic in library code (return errors instead)
- Users can inspect errors via `errors.Is()` and `errors.As()`

**Alternatives Considered**:
- String errors: Not inspectable by calling code
- Panic on errors: Violates constitution principle V

**Impact**: Create error types in `parser/errors.go`, `validator/errors.go`, etc.

---

### 6. Streaming Strategy

**Decision**: Option-based API with default buffering, explicit streaming mode

**Rationale**:
- Default behavior: Buffer records in memory for convenience
- Streaming mode: User provides callback/channel for each record
- API:
  ```go
  // Default (buffered)
  doc, err := decoder.Decode(reader)

  // Streaming
  decoder.DecodeStream(reader, func(record gedcom.Record) error {
      // Process record immediately
      return nil
  })
  ```
- Streaming prevents OOM on large files
- Buffered mode simpler for typical use cases

**Alternatives Considered**:
- Always streaming: More complex API for common case
- Always buffered: Can't handle 100MB+ files

**Impact**: `decoder.Decode()` and `decoder.DecodeStream()` methods

---

### 7. Resource Limits Implementation

**Decision**: Configurable limits with safe defaults, enforced via `internal/limits`

**Rationale**:
- From clarifications: max depth 100, max entities 1M, timeout 5min
- Implementation:
  - `DecoderOptions` struct with limit fields
  - `internal/limits` package tracks:
    - Nesting depth counter
    - Entity count
    - Start time for timeout
  - Check limits at appropriate points:
    - Depth: increment on tag nesting, decrement on pop
    - Count: increment per record parsed
    - Timeout: check periodically (every 100 records)
- Errors when exceeded: `ResourceLimitError` with guidance
- Performance: <5% overhead (simple counter increments)

**Alternatives Considered**:
- Hard-coded limits: No flexibility for legitimate large files
- No limits: Vulnerable to DoS attacks

**Impact**: Create `internal/limits/limits.go`, add fields to `decoder.Options`

---

### 8. Progress Callback Design

**Decision**: Optional callback function in DecoderOptions

**Rationale**:
- From clarifications: Report every 1% or 100 records
- API:
  ```go
  type ProgressFunc func(bytesRead, totalBytes, recordCount int64)

  opts := decoder.Options{
      OnProgress: func(read, total, count int64) {
          pct := (read * 100) / total
          fmt.Printf("Progress: %d%% (%d records)\n", pct, count)
      },
  }
  ```
- Called from decoder after each record
- No callback = no overhead (nil check is fast)

**Alternatives Considered**:
- Channel-based: More complex, requires goroutine management
- Always-on progress: Unnecessary overhead

**Impact**: Add `OnProgress` field to `decoder.Options`

---

### 9. Version Detection Strategy

**Decision**: Header parsing with tag-based fallback

**Rationale**:
- GEDCOM 5.5/5.5.1: Header contains `VERS` tag
- GEDCOM 7.0: Header contains `VERS 7.0` explicitly
- Fallback: Analyze tags used (7.0 has unique tags)
- Implementation:
  - Parse first 50 lines looking for header block
  - Extract version from `HEAD.GEDC.VERS` path
  - If missing/ambiguous, analyze tag names against version-specific lists
- Caching: Store detected version in `gedcom.Document`

**Alternatives Considered**:
- Require users to specify version: Poor UX, violates auto-detection requirement
- Full file scan: Expensive, may not be conclusive

**Impact**: Create `version/detect.go` with detection heuristics

---

### 10. Testing Strategy

**Decision**: Table-driven tests + integration tests + benchmarks

**Rationale**:
- **Table-driven tests**:
  - Perfect for parser (input line → expected tokens)
  - Validation rules (record → expected errors)
  - Easy to add edge cases
  - Pattern:
    ```go
    tests := []struct {
        name    string
        input   string
        want    Line
        wantErr bool
    }{
        {"valid", "0 HEAD", Line{Level: 0, Tag: "HEAD"}, false},
        ...
    }
    ```
- **Integration tests**:
  - Real GEDCOM files in `testdata/` (covering all versions)
  - Parse → validate → encode → parse again (roundtrip)
  - Malformed files with expected errors
- **Benchmarks**:
  - `BenchmarkParseLarge` (10MB file)
  - `BenchmarkValidate` (10K records)
  - Track memory allocations (`b.ReportAllocs()`)

**Alternatives Considered**:
- Unit tests only: Insufficient for complex parsing
- Property-based testing: Overkill for well-defined spec

**Impact**: Test files in each package, `testdata/` with sample GEDCOM files

---

## Technology Stack Summary

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go 1.21+ | Balance of stability and features |
| Dependencies | stdlib only | Zero external deps per requirements |
| Character Encoding | Custom ANSEL + stdlib | ANSEL needs custom impl, rest in stdlib |
| Parsing | Lexer → Parser → Decoder | Clean separation of concerns |
| Streaming | Optional via callback | Handles large files, simple API |
| Error Handling | Structured with wrapping | Line numbers, context, inspectable |
| Resource Limits | Configurable with defaults | Security without inflexibility |
| Progress Reporting | Optional callback | Enable UX without overhead |
| Version Detection | Header + tag analysis | Automatic per FR-002 |
| Testing | Table-driven + integration | Comprehensive coverage |

## Open Questions Resolved

All "NEEDS CLARIFICATION" items from plan.md Technical Context have been resolved through this research. No external research required—all decisions based on:
- Go standard library documentation
- GEDCOM specifications (5.5, 5.5.1, 7.0)
- Constitution requirements
- User clarifications from spec.md
- Go community best practices

## Detailed Technical Research

### GEDCOM 5.5 Specification Details

**Line Format Rules**:
- Format: `LEVEL [XREF] TAG [VALUE]`
- Level: 0-99 (must increment by 1 from parent)
- XREF: Optional, format `@[A-Za-z0-9_]+@` (alphanumeric + underscore)
- TAG: 3-4 uppercase characters (standard tags) or `_TAG` for custom extensions
- VALUE: Optional, max 255 characters per line
- Line terminators: CRLF (recommended), LF, or CR (must be consistent)
- Continuation: Use CONC (concatenate) or CONT (new line) tags at level+1

**Required Tags**:
- Header: `HEAD`, `GEDC`, `VERS`, `CHAR`, `SOUR`, `SUBM`
- Trailer: `TRLR`
- Individual: `NAME` (at least one)
- Family: At least one `HUSB`, `WIFE`, or `CHIL`

**Character Encoding** (GEDCOM 5.5):
- ANSEL (default if not declared)
- ASCII (7-bit)
- UNICODE (UTF-16 with BOM)
- ANSI (Windows-1252)

**ANSEL Character Set**:
- 0x00-0x7F: Standard ASCII
- 0x80-0x9F: Control characters (unused)
- 0xA0-0xCF: Spacing characters (special chars, currency, etc.)
- 0xE0-0xFF: Combining diacritics (placed BEFORE base character)

**Key ANSEL Combining Characters**:
- 0xE0: Hook above
- 0xE1: Grave accent
- 0xE2: Acute accent
- 0xE3: Circumflex
- 0xE4: Tilde
- 0xE5: Macron
- 0xE6: Breve
- 0xE7: Dot above
- 0xE8: Umlaut/diaeresis
- 0xE9: Caron
- 0xEA: Ring above
- 0xEB: Ligature left half
- 0xEC: Ligature right half
- 0xED: Comma above right
- 0xEE: Double acute
- 0xEF: Candrabindu
- 0xF0: Cedilla
- 0xF1: Right hook
- 0xF2: Dot below
- 0xF3: Double dot below
- 0xF4: Ring below
- 0xF5: Double underscore
- 0xF6: Underscore
- 0xF7: Comma below
- 0xF8: Left hook
- 0xF9: Right cedilla
- 0xFA: Upadhmaniya
- 0xFE: High comma centered

**ANSEL to Unicode Conversion**:
- ANSEL: `0xE2 0x65` (acute + e) → Unicode: `0x65 0x0301` (e + combining acute)
- Order reversal required: ANSEL diacritics precede base, Unicode diacritics follow base
- Implementation strategy: Read-ahead buffer, reorder on conversion

**Cross-Reference Format**:
- Pattern: `@[A-Za-z0-9_]+@`
- Examples: `@I1@`, `@F123@`, `@S_001@`
- Must be unique within document
- Referenced before defined is allowed (forward references)

**Header Structure**:
```
0 HEAD
1 SOUR <SYSTEM_ID>
2 VERS <VERSION>
1 DEST <RECEIVING_SYSTEM>
1 DATE <DATE>
2 TIME <TIME>
1 SUBM @SUBM1@
1 GEDC
2 VERS 5.5
1 CHAR <ENCODING>
```

---

### GEDCOM 5.5.1 Specification Details

**Changes from 5.5**:
1. **UTF-8 Support**: Added `CHAR UTF-8` option (previously only UTF-16)
2. **New Tags** (9 additions):
   - `EMAIL`: Email address
   - `FAX`: Fax number
   - `WWW`: Website URL
   - `_ASSO`: Association (custom tag standardized)
   - `FACT`: Generic fact/attribute
   - `MAP`: Geographic coordinates
   - `LATI`: Latitude
   - `LONG`: Longitude
   - `_MARNM`: Married name (custom tag standardized)
3. **Multimedia Changes**:
   - Removed inline BLOBs (binary data)
   - All multimedia must reference external files via `FILE` tag
   - Added `_PRIM` tag to indicate primary photo
4. **Date Format**: Added support for date phrases (text in parentheses)
5. **Backward Compatibility**: 5.5.1 parsers must accept 5.5 files

**Character Encoding** (GEDCOM 5.5.1):
- ANSEL (default)
- ASCII
- UTF-8 (NEW - recommended)
- UTF-16 (with BOM)

**Header Version Declaration**:
```
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
```

---

### GEDCOM 7.0 Specification Details

**Breaking Changes from 5.5.1**:
1. **UTF-8 Mandatory**: All files must be UTF-8 (no ANSEL, ASCII, or UTF-16)
2. **CONC Tag Removed**: No more line continuation via CONC (lines can be longer)
3. **New Required Tags**:
   - `SCHMA`: Schema extension declaration
   - `NO`: Negative assertion (e.g., "NO BIRT" = birth not occurred)
   - `EXID`: External identifier (links to external databases)
   - `TRAN`: Translation of text to another language
4. **Media Type Changes**: Use IANA media types (e.g., `image/jpeg` not `JPG`)
5. **Enumeration Values**: More strict (e.g., `SEX` only allows `M`, `F`, `U`, `X`)
6. **Date Format**: More structured, removed some date phrases
7. **Cross-Reference Format**: More strict, must match `@[A-Z0-9_]+@` (uppercase only)

**New Header Structure**:
```
0 HEAD
1 GEDC
2 VERS 7.0
1 SCHMA
2 TAG _MYEXT http://example.com/myext
1 SOUR <SYSTEM_ID>
2 VERS <VERSION>
1 LANG en-US
1 PLAC
2 FORM City, County, State, Country
```

**Media Types** (GEDCOM 7.0):
- Image: `image/jpeg`, `image/png`, `image/gif`, `image/tiff`, `image/bmp`
- Audio: `audio/mpeg`, `audio/wav`
- Video: `video/mp4`, `video/mpeg`
- Document: `application/pdf`, `text/plain`

**Removed Features**:
- ANSEL encoding (UTF-8 only)
- CONC tag (use longer lines)
- CONT tag (replaced with structured multiline values)
- Some custom tags standardized or removed

**Backward Compatibility**: 7.0 is NOT backward compatible with 5.5.1

---

### Go 1.21 Specific Features to Leverage

**Standard Library Enhancements**:

1. **slices Package** (`slices`):
   - `slices.Contains()`: Fast slice membership check (2.5x faster than loop)
   - `slices.Index()`: Find element index
   - `slices.Sort()`: Generic sorting (no reflection overhead)
   - Use for record filtering, XRef lookups

2. **maps Package** (`maps`):
   - `maps.Clone()`: Deep copy maps (useful for XRefMap)
   - `maps.Equal()`: Compare maps (testing)
   - Use for XRefMap operations

3. **bytes.Buffer Improvements**:
   - `bytes.Buffer.AvailableBuffer()`: Get writable slice without allocation
   - Use for encoder to minimize allocations during write

4. **context Package**:
   - `context.WithCancelCause()`: Cancel with reason (useful for timeout errors)
   - `context.Cause()`: Retrieve cancellation reason
   - Use for timeout handling with clear error messages

5. **errors Package**:
   - `errors.Join()`: Combine multiple errors (useful for validation)
   - Use for aggregating validation errors

6. **Profile-Guided Optimization (PGO)**:
   - Enable with `go build -pgo=auto`
   - Collect profiles during benchmarking
   - Can improve parser performance by 10-20%

**Implementation Examples**:

```go
// Use slices for fast lookups
import "slices"

func (d *Document) HasRecord(xref string) bool {
    return slices.ContainsFunc(d.Records, func(r *Record) bool {
        return r.XRef == xref
    })
}

// Use maps.Clone for safe copying
import "maps"

func (d *Document) CloneXRefMap() map[string]*Record {
    return maps.Clone(d.XRefMap)
}

// Use bytes.Buffer.AvailableBuffer for efficient encoding
func (e *Encoder) writeLine(level int, tag, value string) error {
    buf := e.buf.AvailableBuffer()
    buf = strconv.AppendInt(buf, int64(level), 10)
    buf = append(buf, ' ')
    buf = append(buf, tag...)
    // ...write to underlying writer
}

// Use context.WithCancelCause for timeouts
func (d *Decoder) DecodeWithTimeout(r io.Reader, timeout time.Duration) (*Document, error) {
    ctx, cancel := context.WithTimeoutCause(context.Background(), timeout,
        fmt.Errorf("parsing exceeded timeout of %v", timeout))
    defer cancel()

    // ... parse with context checking
    if ctx.Err() != nil {
        return nil, context.Cause(ctx)
    }
}

// Use errors.Join for validation
func (v *Validator) Validate(doc *Document) error {
    var errs []error
    for _, record := range doc.Records {
        if err := v.validateRecord(record); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)
}
```

---

### Genealogy Date Format Specifications

**Gregorian Calendar** (default):

**Standard Format**: `DAY MONTH YEAR`
- Day: 1-31
- Month: JAN, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC
- Year: 4 digits (or 2 digits for recent dates)

**Date Modifiers**:
- `ABT <date>`: About (approximate)
- `CAL <date>`: Calculated
- `EST <date>`: Estimated
- `BEF <date>`: Before
- `AFT <date>`: After
- `BET <date> AND <date>`: Between two dates
- `FROM <date>`: From (start of range)
- `TO <date>`: To (end of range)
- `FROM <date> TO <date>`: Date range

**Examples**:
- `1 JAN 1950`
- `ABT 1950`
- `BEF 1 JAN 1950`
- `BET 1 JAN 1950 AND 31 DEC 1950`
- `FROM 1950 TO 1960`

**Parsing Strategy**:
```go
type DateParser struct {
    modifierRegex *regexp.Regexp
}

func (p *DateParser) Parse(raw string) Date {
    // 1. Extract modifier (ABT, BEF, etc.)
    // 2. Extract calendar (@#DHEBREW@, etc.)
    // 3. Parse date components (day, month, year)
    // 4. Convert to time.Time if possible
    // 5. Store raw string for unparseable dates
}
```

---

**Hebrew Calendar**:

**Declaration**: `@#DHEBREW@ <date>`

**Months** (13 in leap years):
1. TSH (Tishrei)
2. CSH (Cheshvan)
3. KSL (Kislev)
4. TVT (Tevet)
5. SHV (Shevat)
6. ADR (Adar) or AAD (Adar I)
7. ADS (Adar II - leap years only)
8. NSN (Nisan)
9. IYR (Iyar)
10. SVN (Sivan)
11. TMZ (Tammuz)
12. AAV (Av)
13. ELL (Elul)

**Year**: Counts from creation (e.g., 5784 = 2023-2024 CE)

**Example**: `@#DHEBREW@ 1 TSH 5784`

**Conversion Formula** (Hebrew to Gregorian):
- Complex algorithm (Gauss formula)
- Requires lookup tables for month lengths
- Implementation: Use algorithm from "Calendrical Calculations" (Dershowitz & Reingold)

---

**French Republican Calendar**:

**Declaration**: `@#DFRENCH R@ <date>`

**Months** (12 months of 30 days + 5-6 complementary days):
1. VEND (Vendémiaire)
2. BRUM (Brumaire)
3. FRIM (Frimaire)
4. NIVO (Nivôse)
5. PLUV (Pluviôse)
6. VENT (Ventôse)
7. GERM (Germinal)
8. FLOR (Floréal)
9. PRAI (Prairial)
10. MESS (Messidor)
11. THER (Thermidor)
12. FRUC (Fructidor)
13. COMP (Jours complémentaires - 5 or 6 days)

**Year**: Counts from proclamation of republic (22 SEP 1792 = 1 VEND 1)

**Example**: `@#DFRENCH R@ 1 VEND 1`

**Conversion Formula** (French to Gregorian):
```
Day 1 of Year 1 = 22 September 1792
Each year starts on autumnal equinox
Regular years: 365 days
Leap years: 366 days (years divisible by 4, except as below)
```

---

**Julian Calendar**:

**Declaration**: `@#DJULIAN@ <date>`

**Format**: Same as Gregorian (DAY MONTH YEAR)

**Difference from Gregorian**:
- Julian: Leap year every 4 years (no 100/400 rule)
- Gregorian: Leap year every 4 years, except centuries not divisible by 400
- Drift: Julian is currently 13 days behind Gregorian

**Dual Dating** (historical dates):
- Used when year started on different day (e.g., 25 MAR in England until 1752)
- Format: `1 JAN 1749/50` (Julian 1749, Gregorian 1750)

**Conversion Formula** (Julian to Gregorian):
```go
func julianToGregorian(year, month, day int) time.Time {
    // Calculate Julian Day Number (JDN)
    a := (14 - month) / 12
    y := year + 4800 - a
    m := month + 12*a - 3

    jdn := day + (153*m+2)/5 + 365*y + y/4 - 32083

    // Convert JDN to Gregorian
    a = jdn + 32044
    b := (4*a + 3) / 146097
    c := a - (146097*b)/4
    // ... continue conversion
}
```

---

**Implementation Strategy for Date Parsing**:

1. **Parser Architecture**:
   ```go
   type DateParser struct {
       calendarParsers map[string]CalendarParser
   }

   type CalendarParser interface {
       Parse(raw string) (Date, error)
       ToGregorian(Date) (time.Time, error)
   }
   ```

2. **Parsing Steps**:
   - Step 1: Extract calendar prefix (`@#DHEBREW@`, etc.) or default to Gregorian
   - Step 2: Extract modifier (ABT, BEF, etc.)
   - Step 3: Parse date components using calendar-specific parser
   - Step 4: Attempt conversion to `time.Time` for sorting/comparison
   - Step 5: Store raw string for display and round-trip fidelity

3. **Error Handling**:
   - Invalid dates (e.g., `32 JAN 2020`) return parse error
   - Unparseable dates stored as-is with `Parsed` field as zero value
   - Validation reports errors but doesn't fail parsing

4. **Testing Strategy**:
   - Table-driven tests for each calendar
   - Conversion round-trip tests
   - Edge cases: leap years, month boundaries, BCE dates
   - Malformed dates with expected errors

---

## Next Steps

Proceed to Phase 1:
1. Generate `data-model.md` with entity definitions ✅ COMPLETED
2. Create `contracts/` with Go interface definitions ✅ COMPLETED
3. Generate `quickstart.md` with usage examples ✅ COMPLETED
4. Update agent context file with Go-specific guidance ✅ COMPLETED
5. Deep research on specifications and technical details ✅ COMPLETED
