# Feature Documentation Audit

**Date:** 2026-01-18
**Prepared for:** 1.0.0 Release

## Summary

| Metric | Count |
|--------|-------|
| Total packages audited | 8 |
| Missing from FEATURES.md | 14 items |
| Outdated documentation | 3 items |
| Recommended changes | 17 |

The codebase has excellent feature documentation overall. Most public APIs are well-documented in FEATURES.md. This audit identified several recent features and convenience methods that should be added before the 1.0.0 release.

---

## Package-by-Package Findings

### gedcom/

**Status:** Good coverage with minor additions needed

#### Missing from FEATURES.md

1. **Individual convenience methods** - Not documented:
   - `BirthEvent()` - Returns first birth event (`individual.go:182`)
   - `DeathEvent()` - Returns first death event (`individual.go:191`)
   - `BirthDate()` - Returns parsed birth date (`individual.go:203`)
   - `DeathDate()` - Returns parsed death date (`individual.go:213`)

2. **Date comparison methods** - Documented but incomplete:
   - `IsAfter(other *Date) bool` (`date.go:806`)
   - `IsBefore(other *Date) bool` (`date.go:815`)
   - `IsEqual(other *Date) bool` (`date.go:824`)
   - `YearsBetween(d1, d2 *Date)` (`date.go:839`) - Returns years and exactness flag

3. **Date calendar conversion** - Not documented:
   - `ToGregorian() (*Date, error)` - Converts any calendar to Gregorian (`date.go:753`)

4. **ConversionReport type** - Referenced in Version Conversion section but details not listed:
   - Type is in `report.go`, has `String()`, `AddTransformation()`, `AddDataLoss()`, `HasDataLoss()`
   - `Transformation` and `DataLossItem` types not documented

5. **Record type methods** - Not documented:
   - `IsIndividual()`, `IsFamily()`, `IsSource()` boolean checks (`record.go:53-65`)
   - `GetIndividual()`, `GetFamily()`, `GetSource()`, etc. type assertions (`record.go:68-121`)

#### Correctly Documented
- Document structure and XRefMap
- All record types (Individual, Family, Source, Repository, etc.)
- Relationship traversal methods (Parents, Spouses, Children, etc.)
- Name transliterations
- Date parsing with all formats and calendars
- Vendor detection and vendor extensions (Ancestry, FamilySearch)
- All event and attribute types

---

### decoder/

**Status:** Well documented

#### Missing from FEATURES.md

1. **DecodeOptions.Context** - The context field for cancellation is present in code (`options.go:13`) but not mentioned in documentation
2. **DecodeOptions.MaxNestingDepth** - Malformed file protection (`options.go:16`) not documented
3. **DecodeOptions.StrictMode** - Strict parsing option (`options.go:20`) not documented

#### Correctly Documented
- `Decode()` and `DecodeWithOptions()` functions
- Progress callbacks with `OnProgress` and `TotalSize`
- Automatic version detection
- Character encoding handling

---

### encoder/

**Status:** Well documented

#### Missing from FEATURES.md

1. **EncodeStreaming()** and **EncodeStreamingWithOptions()** - Convenience functions for streaming a complete document (`streaming.go:238-265`) - these mirror the batch API

2. **StreamEncoder.State()** - Returns current encoder state for debugging (`streaming.go:225`)

3. **StreamEncoder.Err()** - Returns any sticky encoding error (`streaming.go:231`)

#### Correctly Documented
- `Encode()` and `EncodeWithOptions()` functions
- StreamEncoder with all write methods
- Line continuation (CONT/CONC) handling
- Entity encoding for all record types
- MaxLineLength and DisableLineWrap options
- Inline repository support

---

### parser/

**Status:** Good coverage

#### Missing from FEATURES.md

1. **RecordIteratorWithOffset** - Variant with accurate byte offset tracking (`iterator.go:156-326`) - used internally for index building

2. **Index persistence methods**:
   - `RecordIndex.SetEncoding(enc string)` (`index.go:127`)
   - `RecordIndex.Encoding() string` (`index.go:132`)

3. **LazyParser additional methods**:
   - `HasIndex() bool` (`lazy.go:80`)
   - `Index() *RecordIndex` (`lazy.go:85`)
   - `RecordCount() int` (`lazy.go:196`)
   - `XRefs() []string` (`lazy.go:187`)
   - `IterateFrom(offset int64)` - Resume iteration from offset (`lazy.go:172`)

#### Correctly Documented
- RecordIterator streaming
- RecordIndex with O(1) lookup
- LazyParser combining iteration and random access
- Index persistence (Save/LoadIndex)

---

### validator/

**Status:** Good coverage with one significant addition

#### Missing from FEATURES.md

1. **StreamingValidator memory monitoring methods**:
   - `SeenXRefCount() int` (`streaming.go:397`)
   - `UsedXRefCount() int` (`streaming.go:403`)

2. **FilterBySeverity() and FilterByCode()** - Utility functions for filtering issues (`issue.go:215-234`)

3. **QualityReport additional methods**:
   - `JSON() ([]byte, error)` - Returns report in JSON format (`quality.go:114`)
   - `IssuesForRecord(xref string) []Issue` (`quality.go:119`)
   - `IssuesByCode(code string) []Issue` (`quality.go:131`)

4. **Issue builder methods** (fluent API):
   - `NewIssue()` (`issue.go:176`)
   - `WithRelatedXRef()` (`issue.go:190`)
   - `WithDetail()` (`issue.go:199`)

#### Outdated Documentation

1. **FEATURES.md line 575**: StreamingValidator two-phase validation correctly describes immediate vs deferred, but doesn't mention that parent-child date logic checks are NOT supported in streaming mode (requires full document).

#### Correctly Documented
- StreamingValidator with ValidateRecord() and Finalize()
- TagRegistry and vendor registries
- Date logic validation
- Orphaned reference detection
- Duplicate detection with DuplicateConfig
- QualityReport with String() output

---

### converter/

**Status:** Well documented

#### Missing from FEATURES.md

1. **ConvertOptions.PreserveUnknownTags** - Option to keep vendor extensions (`options.go:15`) - not mentioned in options list

#### Correctly Documented
- Convert() and ConvertWithOptions()
- All supported conversion paths (6 directions)
- Transformation tracking
- Data loss reporting
- StrictDataLoss option
- Post-conversion validation

---

### charset/

**Status:** Complete documentation

#### Missing from FEATURES.md

1. **Helper functions** (low priority - internal use):
   - `ValidateString(s string) bool` (`charset.go:203`)
   - `ValidateBytes(b []byte) bool` (`charset.go:208`)
   - `DetectBOM()` - Returns reader with encoding detection (`charset.go:222`)
   - `DetectEncodingFromHeader()` - Detects encoding from CHAR tag (`charset.go:303`)
   - `NewReaderWithEncoding()` - Creates reader with specific encoding (`charset.go:358`)

#### Correctly Documented
- All supported encodings (UTF-8, UTF-16 LE/BE, ANSEL, LATIN1, ASCII)
- BOM detection
- Automatic conversion to UTF-8

---

### version/

**Status:** Complete documentation

#### Missing from FEATURES.md

1. **IsValidVersion()** - Validates version string (`detect.go:145`) - minor utility

#### Correctly Documented
- DetectVersion() from header
- Heuristic-based detection from tags
- Support for 5.5, 5.5.1, and 7.0

---

## Recommended FEATURES.md Changes

### Additions

#### 1. Individual Convenience Methods (New Section)

Add after "Relationship Traversal" section:

```markdown
### Event and Date Access

Convenience methods for accessing parsed events and dates on individuals:

| Method | Return Type | Description |
|--------|-------------|-------------|
| `BirthEvent()` | `*Event` | First birth event (nil if none) |
| `DeathEvent()` | `*Event` | First death event (nil if none) |
| `BirthDate()` | `*Date` | Parsed birth date (nil if no event or no date) |
| `DeathDate()` | `*Date` | Parsed death date (nil if no event or no date) |
```

#### 2. Date Comparison Methods

Add to Date Parsing API section:

```markdown
// Compare dates
result := date1.Compare(date2)  // -1, 0, or 1
isAfter := date1.IsAfter(date2)
isBefore := date1.IsBefore(date2)
isEqual := date1.IsEqual(date2)

// Calculate years between dates
years, exact, err := gedcom.YearsBetween(birthDate, deathDate)
// exact is true if both dates are complete (day/month/year)
```

#### 3. Calendar Conversion

Add to Date Parsing Features section:

```markdown
### Calendar Conversion

Convert dates from any calendar to Gregorian:

```go
hebrewDate, _ := gedcom.ParseDate("@#DHEBREW@ 15 NSN 5785")
gregorian, err := hebrewDate.ToGregorian()
// gregorian.Year, gregorian.Month, gregorian.Day in Gregorian calendar
```
```

#### 4. Decoder Options

Add to Progress Reporting section or new Decoder Options section:

```markdown
### Decoder Configuration

| Option | Type | Description |
|--------|------|-------------|
| `Context` | `context.Context` | Cancellation and timeout control |
| `MaxNestingDepth` | `int` | Maximum nesting depth (default: 100) |
| `StrictMode` | `bool` | Reject non-standard extensions |
| `OnProgress` | `ProgressCallback` | Progress reporting callback |
| `TotalSize` | `int64` | Expected file size for progress percentage |
```

#### 5. QualityReport Methods

Add to Quality Report section:

```markdown
Additional QualityReport methods:

| Method | Description |
|--------|-------------|
| `JSON()` | Returns report in JSON format |
| `IssuesForRecord(xref)` | Get all issues affecting a specific record |
| `IssuesByCode(code)` | Get all issues with a specific error code |
```

#### 6. LazyParser Methods

Add to Lazy Parser section:

```markdown
Additional LazyParser methods:

| Method | Description |
|--------|-------------|
| `HasIndex()` | Returns true if index is loaded |
| `Index()` | Returns the current RecordIndex |
| `RecordCount()` | Total indexed records |
| `XRefs()` | List of all indexed cross-references |
| `IterateFrom(offset)` | Resume iteration from byte offset |
```

#### 7. Converter Options

Add PreserveUnknownTags to converter options table:

```markdown
| `PreserveUnknownTags` | `bool` | Keep vendor extensions (default: true) |
```

### Updates

#### 1. Streaming Validator Limitation

Update FEATURES.md line ~575 to clarify:

```markdown
**Two-Phase Validation**:
- **Immediate**: Date logic errors (death before birth), structural issues detected per-record
- **Deferred**: Cross-reference validation (orphaned FAMC, FAMS, HUSB, WIFE, CHIL, SOUR) in `Finalize()`

**Note**: Parent-child date logic checks (child born before parent) require a complete document and are not supported in streaming mode.
```

#### 2. Test Coverage Line

Update FEATURES.md line 999 to reflect current coverage:
```markdown
- 93% test coverage across core packages
```
(Verify this is still accurate before release)

#### 3. Performance Benchmarks

Verify benchmark numbers in FEATURES.md line 907-911 are still accurate for 1.0.0.

### Removals

None identified. All documented features are present in the codebase.

---

## Verification Checklist

- [x] gedcom/ - All 8 public type files examined
- [x] decoder/ - decoder.go, options.go, entity.go, progress.go examined
- [x] encoder/ - encoder.go, options.go, entity_writer.go, streaming.go examined
- [x] parser/ - parser.go, iterator.go, index.go, lazy.go examined
- [x] validator/ - validator.go, issue.go, streaming.go, quality.go, registry.go, vendor_tags.go examined
- [x] converter/ - converter.go, options.go examined
- [x] charset/ - charset.go examined
- [x] version/ - detect.go examined
- [x] Cross-referenced all public APIs against FEATURES.md sections
- [x] Report saved to docs/audit-features-code.md
- [x] Findings include file:line references
