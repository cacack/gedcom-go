# Features

This document provides a comprehensive list of all features implemented in gedcom-go.

For planned features, see [GitHub Issues](https://github.com/cacack/gedcom-go/issues).

## Convenience API

Single-import facade for common GEDCOM operations:

```go
import gedcomgo "github.com/cacack/gedcom-go"

doc, err := gedcomgo.Decode(file)           // Parse GEDCOM
err = gedcomgo.Encode(writer, doc)          // Write GEDCOM
errors := gedcomgo.Validate(doc)            // Basic validation
issues := gedcomgo.ValidateAll(doc)         // Comprehensive validation
converted, report, err := gedcomgo.Convert(doc, gedcomgo.Version70)  // Version conversion
```

| Function | Description |
|----------|-------------|
| `Decode(r)` | Parse GEDCOM file with default options |
| `DecodeWithDiagnostics(r)` | Parse with error collection for lenient mode |
| `Encode(w, doc)` | Write GEDCOM file with default options |
| `Validate(doc)` | Basic structural validation (returns `[]error`) |
| `ValidateAll(doc)` | Comprehensive validation with severity levels (returns `[]Issue`) |
| `Convert(doc, version)` | Convert between GEDCOM versions |

Re-exported types for single-import convenience:
- `Document`, `Individual`, `Family` - Core data types
- `Version`, `Version55`, `Version551`, `Version70` - Version constants
- `DecodeResult`, `Issue`, `ConversionReport` - Result types

For advanced use cases (custom options, progress callbacks, streaming), import the underlying packages directly. See README.md for examples.

## Multi-Version Support

| Version | Status | Notes |
|---------|--------|-------|
| GEDCOM 5.5 | Full | Legacy format support |
| GEDCOM 5.5.1 | Full | Most common format |
| GEDCOM 7.0 | Full | Latest standard |

- Automatic version detection from header
- Heuristic-based detection for malformed headers
- Version-aware validation rules

## Vendor Detection

Automatic detection of the originating software from `HEAD.SOUR`:

| Vendor | Detection Patterns |
|--------|-------------------|
| Ancestry | ancestry, familytreemaker |
| FamilySearch | familysearch |
| RootsMagic | rootsmagic |
| Legacy | legacy |
| Gramps | gramps |
| MyHeritage | myheritage |

- Case-insensitive substring matching
- Exposed via `Document.Vendor` field
- Helper methods: `Vendor.String()`, `Vendor.IsKnown()`
- Unknown sources return `VendorUnknown` (never errors)

```go
doc, _ := decoder.Decode(reader)
if doc.Vendor == gedcom.VendorAncestry {
    // Handle Ancestry-specific extensions
}
```

## Vendor Extensions

Structured parsing for vendor-specific GEDCOM extensions:

### Ancestry.com Extensions

| Tag | Location | Description |
|-----|----------|-------------|
| `_APID` | Source Citation | Ancestry Permanent ID linking to database record |
| `_TREE` | Header | Ancestry tree reference |

```go
// Access Ancestry APID on source citations
for _, cite := range individual.SourceCitations {
    if cite.AncestryAPID != nil {
        fmt.Println(cite.AncestryAPID.Database)  // "7602"
        fmt.Println(cite.AncestryAPID.Record)    // "2771226"
        fmt.Println(cite.AncestryAPID.URL())     // Full Ancestry URL
    }
}

// Access tree ID from header
fmt.Println(doc.Header.AncestryTreeID)  // "@T123@"
```

### FamilySearch Extensions

| Tag | Location | Description |
|-----|----------|-------------|
| `_FSFTID` | Individual | FamilySearch Family Tree ID |

```go
// Access FamilySearch ID on individuals
indi := doc.GetIndividual("@I1@")
if indi.FamilySearchID != "" {
    fmt.Println(indi.FamilySearchID)      // "KWCJ-QN7"
    fmt.Println(indi.FamilySearchURL())   // "https://www.familysearch.org/tree/person/details/KWCJ-QN7"
}
```

### Round-Trip Preservation

All vendor extensions are preserved during encode/decode cycles. Custom tags not explicitly parsed are retained in the raw `Tags` field on each entity.

## GEDCOM 7.0 Features

### External Identifiers (EXID)

GEDCOM 7.0 external identifiers link records to external systems like FamilySearch, Ancestry, FindAGrave, etc.

```go
// Access external identifiers on any entity
for _, exid := range individual.ExternalIDs {
    fmt.Println(exid.Value)  // "123"
    fmt.Println(exid.Type)   // "http://example.com" (URI identifying the system)
}
```

| Field | Description |
|-------|-------------|
| Value | The external identifier string |
| Type | URI identifying the external system |

Supported on: Individual, Family, Source, Repository, Submitter, Note, MediaObject

### Schema Definition (SCHMA)

GEDCOM 7.0 schema definitions map extension tags to their URI identifiers, enabling interoperability with custom tags from other applications.

```go
// Access schema mappings from header
if doc.Schema != nil {
    for tag, uri := range doc.Schema.TagMappings {
        fmt.Printf("%s -> %s\n", tag, uri)
        // _SKYPEID -> http://xmlns.com/foaf/0.1/skypeID
    }
}
```

| Field | Description |
|-------|-------------|
| TagMappings | Map of tag names to URIs (e.g., `_SKYPEID` → `http://xmlns.com/foaf/0.1/skypeID`) |

The schema is parsed from the GEDCOM 7.0 header's SCHMA structure and stored in `Document.Schema`. For GEDCOM 5.5/5.5.1 files, `Document.Schema` is nil.

### Shared Notes (SNOTE)

GEDCOM 7.0 shared notes are distinct from inline NOTE tags, supporting MIME types, language tags, and translations for multi-language note content.

```go
// Access shared notes
for _, snote := range doc.SharedNotes() {
    fmt.Println(snote.XRef)      // "@N1@"
    fmt.Println(snote.Text)      // "Note content"
    fmt.Println(snote.MIME)      // "text/html"
    fmt.Println(snote.Language)  // "en"

    // Access translations
    for _, tran := range snote.Translations {
        fmt.Println(tran.Value)     // "Translated content"
        fmt.Println(tran.Language)  // "es"
        fmt.Println(tran.MIME)      // "text/plain"
    }
}

// Lookup by XRef
snote := doc.GetSharedNote("@N1@")
```

| Field | Description |
|-------|-------------|
| XRef | Cross-reference identifier |
| Text | Note content |
| MIME | Media type (text/plain, text/html) |
| Language | BCP 47 language tag |
| Translations | Translated versions with their own MIME/Language |
| SourceCitations | Source citations on the note |
| ExternalIDs | External identifiers (EXID) |
| ChangeDate | Last modification date |

Supported on: Document (as top-level records accessible via `SharedNotes()` and `GetSharedNote()`)

## Character Encoding

| Encoding | Status | Notes |
|----------|--------|-------|
| UTF-8 | Full | With BOM detection |
| ASCII | Full | Subset of UTF-8 |
| LATIN1 (ISO-8859-1) | Full | Converted to UTF-8 |
| UTF-16 LE/BE | Full | With BOM detection |
| ANSEL | Full | With combining diacritical reordering |

## Record Types

### Individuals (INDI)

- Cross-reference ID (`@I1@`)
- Names with components (given, surname, prefix, suffix, nickname)
- Name types (birth, married, aka)
- Sex (M/F/U)
- Events (see Events section)
- Attributes (see Attributes section)
- Family links (FAMC, FAMS) with pedigree types
- Associations (ASSO) with roles
- LDS ordinances (BAPL, CONL, ENDL, SLGC)
- Source citations
- Notes and multimedia references
- Change dates (CHAN)

### Families (FAM)

- Cross-reference ID (`@F1@`)
- Husband/Wife references
- Children references
- Family events (see Events section)
- LDS ordinances (SLGS)
- Source citations
- Notes

### Sources (SOUR)

- Cross-reference ID (`@S1@`)
- Title, author, publication info
- Repository references
- Notes and multimedia

### Repositories (REPO)

- Cross-reference ID (`@R1@`)
- Name and address
- Notes

### Submitters (SUBM)

- Cross-reference ID (`@U1@`)
- Name, address, language
- Multimedia references

### Notes (NOTE)

- Cross-reference ID (`@N1@`)
- Text content with continuation

### Multimedia (OBJE)

- Cross-reference ID (`@M1@`)
- File references and formats
- Titles

## Events

### Individual Events

| Tag | Event | Subordinates |
|-----|-------|--------------|
| BIRT | Birth | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE |
| DEAT | Death | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE |
| BURI | Burial | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE |
| CREM | Cremation | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE |
| ADOP | Adoption | DATE, PLAC, FAMC with ADOP type |
| BAPM | Baptism | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| CHR | Christening | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| CHRA | Adult Christening | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| BARM | Bar Mitzvah | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| BASM | Bas Mitzvah | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| BLES | Blessing | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| CONF | Confirmation | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| FCOM | First Communion | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| ORDN | Ordination | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| GRAD | Graduation | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| RETI | Retirement | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| NATU | Naturalization | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| EMIG | Emigration | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| IMMI | Immigration | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| CENS | Census | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| PROB | Probate | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| WILL | Will | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| EVEN | Generic Event | DATE, PLAC, TYPE, CAUS, AGE, AGNC |

### Family Events

| Tag | Event | Subordinates |
|-----|-------|--------------|
| MARR | Marriage | DATE, PLAC, TYPE, HUSB.AGE, WIFE.AGE |
| MARB | Marriage Bann | DATE, PLAC |
| MARC | Marriage Contract | DATE, PLAC |
| MARL | Marriage License | DATE, PLAC |
| MARS | Marriage Settlement | DATE, PLAC |
| ENGA | Engagement | DATE, PLAC |
| DIV | Divorce | DATE, PLAC |
| DIVF | Divorce Filed | DATE, PLAC |
| ANUL | Annulment | DATE, PLAC |
| EVEN | Generic Event | DATE, PLAC, TYPE |

## Attributes

| Tag | Attribute | Notes |
|-----|-----------|-------|
| OCCU | Occupation | With DATE for periods |
| RESI | Residence | With ADDR structure |
| EDUC | Education | |
| RELI | Religion | |
| TITL | Title | Nobility, professional |
| NATI | Nationality | |
| CAST | Caste | |
| DSCR | Physical Description | |
| IDNO | ID Number | |
| SSN | Social Security Number | |
| NCHI | Number of Children | |
| NMR | Number of Marriages | |
| PROP | Property | |

## Source Citations

- Embedded citations (within records)
- Referenced citations (via @SOUR@ xref)
- PAGE - Specific location in source
- QUAY - Quality/certainty assessment (0-3)
- DATA - Citation data with DATE and TEXT
- Notes on citations

## Place Structure

- Place name with hierarchy (comma-separated)
- MAP coordinates (LATI, LONG)
- Place notes

## Address Structure

- ADR1, ADR2, ADR3 - Address lines
- CITY - City
- STAE - State/Province
- POST - Postal code
- CTRY - Country
- PHON - Phone numbers
- EMAIL - Email addresses
- FAX - Fax numbers
- WWW - Web URLs

## Name Structure

- Full name with surname delimiters (`/surname/`)
- GIVN - Given name
- SURN - Surname
- NPFX - Name prefix (Dr., Rev.)
- NSFX - Name suffix (Jr., III)
- SPFX - Surname prefix (de, van, von)
- NICK - Nickname
- TYPE - Name type (birth, married, aka)

### Transliterations (TRAN)

Support for alternative name representations in different scripts/languages (GEDCOM 7.0):

```go
// Access name transliterations
for _, name := range individual.Names {
    for _, tran := range name.Transliterations {
        fmt.Println(tran.Value)     // "Johann /Müller/"
        fmt.Println(tran.Language)  // "de"
        fmt.Println(tran.Given)     // "Johann"
        fmt.Println(tran.Surname)   // "Müller"
    }
}
```

| Field | Description |
|-------|-------------|
| Value | Full transliterated name |
| Language | BCP 47 language tag (e.g., "en-GB", "ja-Latn") |
| Given | Transliterated given name |
| Surname | Transliterated surname |
| Prefix | Transliterated name prefix |
| Suffix | Transliterated name suffix |
| Nickname | Transliterated nickname |
| SurnamePrefix | Transliterated surname prefix |

## Pedigree (PEDI) Support

- FAMC with pedigree linkage type
- Supported types: birth, adopted, foster, sealing

## LDS Ordinances

| Tag | Ordinance |
|-----|-----------|
| BAPL | Baptism (LDS) |
| CONL | Confirmation (LDS) |
| ENDL | Endowment |
| SLGC | Sealing to Parents |
| SLGS | Sealing to Spouse |

Each includes: DATE, PLAC, TEMP (temple), STAT (status)

## Associations (ASSO)

- Link individuals with roles
- Supported roles: GODP (godparent), WITN (witness), custom roles
- PHRASE - Human-readable description (GEDCOM 7.0)
- Source citations on associations (GEDCOM 7.0)
- Notes on associations

```go
// Access GEDCOM 7.0 association features
for _, assoc := range individual.Associations {
    fmt.Println(assoc.Role)    // "GODP"
    fmt.Println(assoc.Phrase)  // "Godparent at baptism"
    for _, cite := range assoc.SourceCitations {
        fmt.Println(cite.SourceXRef)  // "@S1@"
    }
}
```

## Date Parsing

Structured date parsing for GEDCOM date strings with full support for:

### Date Formats

| Format | Example | Notes |
|--------|---------|-------|
| Exact | `25 DEC 2020` | Full day, month, year |
| Month-Year | `JAN 1900` | Partial date |
| Year only | `1850` | Partial date |
| About | `ABT 1850` | Approximate |
| Calculated | `CAL 1875` | Mathematically derived |
| Estimated | `EST 1820` | Algorithm-based estimate |
| Before | `BEF 1900` | Upper bound |
| After | `AFT 1850` | Lower bound |
| Range | `BET 1850 AND 1860` | Between two dates |
| Period | `FROM 1880 TO 1920` | Duration/interval |
| Interpreted | `INT 1850 (about eighteen fifty)` | User-clarified ambiguous date |

### Edge Cases

| Format | Example | Notes |
|--------|---------|-------|
| B.C. dates | `44 BC`, `753 B.C.E.` | IsBC flag set |
| Dual dating | `21 FEB 1750/51` | Both years accessible |
| Date phrases | `(unknown)` | GEDCOM 5.5 format |
| PHRASE subordinate | `3 PHRASE Afternoon` | GEDCOM 7.0 human-readable description |

### Validation

```go
// Validate complete dates using stdlib
err := date.Validate()
// Returns nil for valid dates
// Returns clear error for invalid: "invalid date: February has 28 days in 2023"
```

- Uses `time.Date()` normalization (no reimplemented calendar math)
- Skips validation for partial dates (lossless representation)
- Detects invalid day/month combinations (Feb 30, Jun 31)
- Handles leap years correctly (Feb 29 2000 valid, 1900 invalid)

### Calendar Systems

Full parsing support for historical calendars used in genealogical records:

| Calendar | Escape Sequence | Month Codes |
|----------|-----------------|-------------|
| Gregorian | `@#DGREGORIAN@` (default) | JAN, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC |
| Julian | `@#DJULIAN@` | JAN, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC |
| Hebrew | `@#DHEBREW@` | TSH, CSH, KSL, TVT, SHV, ADR, ADS, NSN, IYR, SVN, TMZ, AAV, ELL |
| French Republican | `@#DFRENCH R@` | VEND, BRUM, FRIM, NIVO, PLUV, VENT, GERM, FLOR, PRAI, MESS, THER, FRUC, COMP |

```go
// Parse a Hebrew calendar date
date, _ := gedcom.ParseDate("@#DHEBREW@ 15 NSN 5785")
date.Calendar  // CalendarHebrew
date.Month     // 8 (Nisan)
date.Year      // 5785

// Parse a French Republican date
date, _ := gedcom.ParseDate("@#DFRENCH R@ 1 VEND 1")
date.Calendar  // CalendarFrenchRepublican
date.Month     // 1 (Vendémiaire)
date.Year      // 1 (Year I of the Republic)
```

### Features

- Case-insensitive month parsing (`Jan`, `JAN`, `jan`) for all calendars
- Whitespace tolerance (leading, trailing, multiple spaces)
- Original string preserved for round-trip fidelity
- B.C. date comparison (100 BC > 200 BC chronologically)

### API

```go
// Parse a GEDCOM date string
date, err := gedcom.ParseDate("25 DEC 2020")

// Access parsed components
date.Day      // 25
date.Month    // 12
date.Year     // 2020
date.Modifier // ModifierNone

// Edge case fields
date.IsBC     // true for B.C. dates
date.DualYear // second year from "1750/51" format
date.Phrase   // text from "(unknown)" format
date.IsPhrase // true for date phrases

// Validate complete dates
err := date.Validate()  // nil if valid

// Compare dates for sorting
result := date1.Compare(date2)  // -1, 0, or 1
isAfter := date1.IsAfter(date2)
isBefore := date1.IsBefore(date2)
isEqual := date1.IsEqual(date2)

// Calculate years between dates
years, exact, err := gedcom.YearsBetween(birthDate, deathDate)
// exact is true if both dates are complete (day/month/year)

// Convert to time.Time (complete dates only)
t, err := date.ToTime()

// Get original string
s := date.String()  // "25 DEC 2020"
```

### Calendar Conversion

Convert dates from any calendar to Gregorian:

```go
hebrewDate, _ := gedcom.ParseDate("@#DHEBREW@ 15 NSN 5785")
gregorian, err := hebrewDate.ToGregorian()
// gregorian.Year, gregorian.Month, gregorian.Day in Gregorian calendar
```

Supports conversion from Julian, Hebrew, and French Republican calendars to Gregorian.

## Metadata

- REFN - Reference numbers with TYPE
- UID - Unique identifiers
- CHAN - Change date with DATE and TIME
- CREA - Creation date (GEDCOM 7.0)

## Validation

### Structural Validation
- Valid line format (level, tag, value, xref)
- Proper hierarchy (levels increment by 1)
- Required tags present (HEAD, TRLR)
- Valid cross-references

### Version-Specific Validation
- Tag validity per GEDCOM version
- Required subordinate tags
- Deprecated tag warnings
- XRef length validation (20-char limit for GEDCOM 5.5/5.5.1, unlimited for 7.0)
- Header SUBM cardinality (required for 5.5/5.5.1, optional for 7.0)

### Error Reporting
- Line numbers for all errors
- Error categorization (error, warning)
- Clear error messages with context

### Enhanced Data Validation

Comprehensive data quality validation beyond structural correctness:

**Date Logic Validation:**

| Check | Severity | Description |
|-------|----------|-------------|
| Death before birth | Error | Death date precedes birth date |
| Child before parent | Error | Child born before parent |
| Marriage before birth | Error | Marriage date before spouse's birth |
| Impossible age | Warning | Age exceeds configurable maximum (default: 120) |
| Unreasonable parent age | Warning | Parent age at child's birth outside normal range |

```go
v := validator.New()
issues := v.ValidateDateLogic(doc)
for _, issue := range issues {
    fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
}
```

**Orphaned Reference Detection:**

Typed detection for all GEDCOM reference types:

| Error Code | Reference Type | Description |
|------------|----------------|-------------|
| ORPHANED_FAMC | FAMC | Individual references non-existent family (as child) |
| ORPHANED_FAMS | FAMS | Individual references non-existent family (as spouse) |
| ORPHANED_HUSB | HUSB | Family references non-existent husband |
| ORPHANED_WIFE | WIFE | Family references non-existent wife |
| ORPHANED_CHIL | CHIL | Family references non-existent child |
| ORPHANED_SOUR | SOUR | Citation references non-existent source |

```go
issues := v.FindOrphanedReferences(doc)
```

**Duplicate Detection:**

Configurable matching based on name similarity and date proximity:

```go
config := &validator.DuplicateConfig{
    RequireExactSurname: true,
    MinNameSimilarity:   0.8,
    MaxBirthYearDiff:    2,
    MinConfidence:       0.7,
}
v := validator.NewWithConfig(&validator.ValidatorConfig{Duplicates: config})
pairs := v.FindPotentialDuplicates(doc)
for _, pair := range pairs {
    fmt.Printf("Potential duplicate: %s and %s (%.0f%% confidence)\n",
        pair.Individual1.XRef, pair.Individual2.XRef, pair.Confidence*100)
}
```

**Quality Report:**

Comprehensive quality assessment with metrics and issue aggregation:

```go
report := v.QualityReport(doc)
fmt.Println(report.String())
// Output:
// GEDCOM Quality Report
// =====================
// Records: 150 individuals, 45 families, 12 sources
//
// Data Completeness:
// - Birth dates: 89% (134/150)
// - Sources: 45% (68/150)
//
// Issues Found: 23 total
// - Errors: 3
// - Warnings: 8
// - Info: 12
```

Additional QualityReport methods:

| Method | Return Type | Description |
|--------|-------------|-------------|
| `JSON()` | `([]byte, error)` | Returns report in JSON format |
| `IssuesForRecord(xref)` | `[]Issue` | Get all issues affecting a specific record |
| `IssuesByCode(code)` | `[]Issue` | Get all issues with a specific error code |

**Issue Filtering:**

Utility functions for filtering validation issues:

```go
// Filter by severity
errors := validator.FilterBySeverity(issues, validator.SeverityError)
warnings := validator.FilterBySeverity(issues, validator.SeverityWarning)

// Filter by error code
orphaned := validator.FilterByCode(issues, "ORPHANED_FAMC")
```

**Fluent Issue Builder:**

Create validation issues with a fluent API:

```go
issue := validator.NewIssue("CUSTOM_ERROR", "Custom error message").
    WithRelatedXRef("@I1@").
    WithDetail("field", "NAME")
```

**Configurable Strictness:**

Control which severity levels are reported:

| Level | Reports |
|-------|---------|
| StrictnessRelaxed | Errors only |
| StrictnessNormal | Errors + Warnings (default) |
| StrictnessStrict | All issues including Info |

```go
v := validator.NewWithConfig(&validator.ValidatorConfig{
    Strictness: validator.StrictnessStrict,
})
issues := v.ValidateAll(doc)  // Returns all severity levels
```

### Streaming Validator

Memory-efficient validation for very large files without loading the entire document into memory:

| Method | Description |
|--------|-------------|
| `NewStreamingValidator(opts)` | Create validator with options |
| `ValidateRecord(record)` | Validate single record, returns immediate issues |
| `Finalize()` | Check cross-references, returns deferred issues |
| `Reset()` | Reset for reuse with another file |
| `SeenXRefCount()` | Number of cross-reference declarations seen |
| `UsedXRefCount()` | Number of cross-reference usages seen |

```go
// Validate records incrementally during parsing
v := validator.NewStreamingValidator(validator.StreamingOptions{})

for _, record := range records {
    issues := v.ValidateRecord(record)  // Immediate issues (date logic, structure)
    for _, issue := range issues {
        fmt.Printf("[%s] %s\n", issue.Severity, issue.Message)
    }
}

// Check cross-references after all records processed
finalIssues := v.Finalize()  // Orphaned reference detection
```

**Two-Phase Validation**:
- **Immediate**: Date logic errors (death before birth), structural issues detected per-record
- **Deferred**: Cross-reference validation (orphaned FAMC, FAMS, HUSB, WIFE, CHIL, SOUR) in `Finalize()`

**Note**: Parent-child date logic checks (child born before parent) require a complete document and are not supported in streaming mode.

**Memory**: O(unique XRefs) not O(records) - tracks only cross-reference declarations and usages.

**Custom Tag Registry:**

Register and validate vendor-specific custom tags (underscore-prefixed):

| Function | Description |
|----------|-------------|
| `NewTagRegistry()` | Create empty registry for custom tag definitions |
| `AncestryRegistry()` | Pre-built registry for Ancestry.com tags |
| `FamilySearchRegistry()` | Pre-built registry for FamilySearch tags |
| `RootsMagicRegistry()` | Pre-built registry for RootsMagic tags |
| `MergeRegistries()` | Combine multiple registries |
| `DefaultVendorRegistry()` | Merged registry with all vendor tags |
| `RegistryForVendor()` | Get registry for detected vendor |

```go
// Register custom tags
registry := validator.NewTagRegistry()
registry.Register("_MILT", validator.TagDefinition{
    AllowedParents: []string{"INDI"},
    Description:    "Military service",
})

// Or use pre-built vendor registry
registry := validator.AncestryRegistry()

// Or use combined registry for all vendors
registry := validator.DefaultVendorRegistry()

// Configure validator
v := validator.NewWithConfig(&validator.ValidatorConfig{
    TagRegistry:        registry,
    ValidateCustomTags: true,
})
issues := v.ValidateCustomTags(doc)
```

Custom tag validation error codes:

| Error Code | Severity | Description |
|------------|----------|-------------|
| UNKNOWN_CUSTOM_TAG | Warning | Underscore-prefixed tag not in registry |
| INVALID_TAG_PARENT | Error | Custom tag used under wrong parent |
| INVALID_TAG_VALUE | Error | Custom tag value doesn't match pattern |

Pre-built vendor registry tags:

| Vendor | Tags |
|--------|------|
| Ancestry | `_APID`, `_TREE`, `_MILT`, `_DEST`, `_PRIM`, `_PHOTO` |
| FamilySearch | `_FSFTID`, `_FSORD`, `_FSTAG` |
| RootsMagic | `_PRIM`, `_SDATE`, `_TMPLT` |

## Decoder Configuration

Full control over decoding behavior with `DecodeOptions`:

| Option | Type | Description |
|--------|------|-------------|
| `Context` | `context.Context` | Cancellation and timeout control |
| `MaxNestingDepth` | `int` | Maximum nesting depth (default: 100) |
| `StrictMode` | `bool` | Reject non-standard extensions |
| `OnProgress` | `ProgressCallback` | Progress reporting callback |
| `TotalSize` | `int64` | Expected file size for progress percentage |

### Progress Reporting

Optional progress callbacks for monitoring large file processing:

```go
// Track decoding progress for large files
opts := &decoder.DecodeOptions{
    TotalSize: fileInfo.Size(),
    OnProgress: func(bytesRead, totalBytes int64) {
        if totalBytes > 0 {
            fmt.Printf("\rProgress: %d%%", bytesRead*100/totalBytes)
        }
    },
}
doc, err := decoder.DecodeWithOptions(reader, opts)
```

- Zero overhead when `OnProgress` is nil (no wrapper created)
- Reports `-1` for total size when unknown (streaming inputs)
- Callback receives cumulative bytes read on each read operation

### Context Support

Cancel long-running decodes with context:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

opts := &decoder.DecodeOptions{Context: ctx}
doc, err := decoder.DecodeWithOptions(reader, opts)
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("Decode timed out")
}
```

## Version Conversion

Bidirectional conversion between GEDCOM versions with transformation tracking.

### Supported Conversions

| From | To | Type |
|------|-----|------|
| 5.5 | 5.5.1 | Upgrade (backward compatible) |
| 5.5 | 7.0 | Upgrade |
| 5.5.1 | 5.5 | Downgrade |
| 5.5.1 | 7.0 | Upgrade |
| 7.0 | 5.5 | Downgrade |
| 7.0 | 5.5.1 | Downgrade |

### Features

- **Transformation tracking**: Every change recorded in `ConversionReport`
- **Data loss reporting**: Identifies features lost during downgrades
- **Post-conversion validation**: Optional validation of converted document
- **Non-destructive**: Original document unchanged (deep copy)
- **Strict mode**: Fail on any data loss with `StrictDataLoss` option

### Automatic Transformations

| Transformation | Direction | Description |
|---------------|-----------|-------------|
| CONC removal | Upgrade to 7.0 | Merges continuation tags into values |
| CONT to newlines | Upgrade to 7.0 | Converts CONT tags to embedded newlines |
| XRef uppercase | Upgrade to 7.0 | Normalizes cross-references |
| Media types | Both | Maps between legacy (JPG) and IANA (image/jpeg) |
| Newlines to CONT | Downgrade from 7.0 | Expands embedded newlines to CONT tags |

### API

```go
// Simple conversion with defaults
converted, report, err := converter.Convert(doc, gedcom.Version70)

// With options
opts := &converter.ConvertOptions{
    Validate:            true,
    StrictDataLoss:      true,  // Fail on any data loss
    PreserveUnknownTags: true,  // Keep vendor extensions (default: true)
}
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version55, opts)

// Check report
if report.HasDataLoss() {
    for _, item := range report.DataLoss {
        fmt.Printf("Lost: %s - %s\n", item.Feature, item.Reason)
    }
}
```

See [docs/CONVERTER.md](docs/CONVERTER.md) for detailed documentation.

## Encoder

- Write valid GEDCOM files
- Configurable line endings (LF, CRLF)
- GEDCOM 5.5, 5.5.1, 7.0 output
- UTF-8 output

### Streaming Encoder

Memory-efficient encoding for very large files (1M+ records) without loading the entire document into memory:

| Method | Description |
|--------|-------------|
| `NewStreamEncoder(w)` | Create encoder with default options |
| `NewStreamEncoderWithOptions(w, opts)` | Create encoder with custom options |
| `WriteHeader(h)` | Write GEDCOM header (must be called first) |
| `WriteRecord(r)` | Write individual record (can be called 0+ times) |
| `WriteTrailer()` | Write GEDCOM trailer (completes the document) |
| `Flush()` | Flush buffered data to writer |
| `Close()` | Ensure trailer written and flush |
| `State()` | Returns current encoder state for debugging |
| `Err()` | Returns any sticky encoding error |

```go
// Stream records to file with constant memory usage
f, _ := os.Create("large.ged")
enc := encoder.NewStreamEncoder(f)

enc.WriteHeader(header)
for _, record := range records {
    enc.WriteRecord(record)  // Memory stays constant
}
enc.WriteTrailer()
enc.Close()
```

**State Machine**: Enforces valid GEDCOM structure (HEAD → records → TRLR). Invalid transitions return descriptive errors.

**Memory**: O(1) regardless of record count - suitable for generating files with millions of records.

Convenience functions for streaming a complete document:

```go
// Stream an entire document with streaming encoder internally
err := encoder.EncodeStreaming(writer, doc)

// With options
err := encoder.EncodeStreamingWithOptions(writer, doc, opts)
```

### High-Level Type Encoding

Full support for encoding typed entities back to GEDCOM format:

| Entity Type | Supported Fields |
|-------------|------------------|
| Individual | Names, sex, events, attributes, family links, associations, LDS ordinances, citations, notes, media |
| Family | Spouse/child refs, events, LDS ordinances, citations, notes, media |
| Source | Title, author, publication, text, repository ref/inline, notes, media |
| Repository | Name, address, notes |
| Submitter | Name, address, contact info, languages |
| Note | Text with continuation lines |
| MediaObject | Files, formats, translations, citations |

### Round-Trip Encoding

Decode → modify → encode workflow:
- Lossless by default: original tags preserved when present
- Entity conversion: generates tags from typed fields when tags are empty
- All nested structures supported: events, names, citations, addresses, coordinates

### Line Continuation (CONT/CONC)

Automatic handling of multiline and long text per GEDCOM specification:

- **CONT (continuation)**: Multiline text automatically split on `\n` into CONT tags
- **CONC (concatenation)**: Long lines (>248 chars) automatically split at word boundaries

```go
// Multiline text becomes CONT continuation
note := "Line one\nLine two\nLine three"
// Encodes as:
// 1 NOTE Line one
// 2 CONT Line two
// 2 CONT Line three

// Long lines become CONC concatenation (split at word boundaries)
longText := "Very long text exceeding 248 characters..."
// Encodes as:
// 1 NOTE Very long text exceeding...
// 2 CONC the 248 character limit...
```

Configurable via `EncodeOptions`:

| Option | Default | Description |
|--------|---------|-------------|
| `MaxLineLength` | 248 | Maximum line length before CONC split |
| `DisableLineWrap` | false | Disable automatic CONC splitting |

### Inline Repository Support

Sources support both XRef references and inline repository definitions:

```go
// XRef reference to separate repository record
source.RepositoryRef = "@R1@"
// Encodes as: 1 REPO @R1@

// Inline repository definition (no separate record needed)
source.Repository = &gedcom.InlineRepository{Name: "State Archives"}
// Encodes as:
// 1 REPO
// 2 NAME State Archives
```

Useful for sources imported from GEDCOM files where repository names are stored inline rather than as separate records.

## Incremental Parsing

Memory-efficient parsing for very large files with on-demand record access:

### Record Iterator

Stream records one at a time without loading all into memory:

```go
// Iterate records with minimal memory
it := parser.NewRecordIterator(reader)
for it.Next() {
    rec := it.Record()
    fmt.Printf("Record: %s %s\n", rec.XRef, rec.Type)
    // Process rec.Lines...
}
if err := it.Err(); err != nil {
    log.Fatal(err)
}
```

| Field | Description |
|-------|-------------|
| `RawRecord.XRef` | Cross-reference ID (e.g., "@I1@") |
| `RawRecord.Type` | Record type tag (INDI, FAM, SOUR, etc.) |
| `RawRecord.Lines` | All parsed lines for this record |
| `RawRecord.ByteOffset` | Starting byte position in file |
| `RawRecord.ByteLength` | Total bytes for this record |

### Record Index

Build an index for O(1) random access by cross-reference ID:

```go
// Build index (one-time O(n) scan)
index, err := parser.BuildIndex(file)

// O(1) lookup
entry, found := index.Lookup("@I1@")
if found {
    fmt.Printf("Record at byte %d\n", entry.ByteOffset)
}

// Persist index for faster subsequent access
index.Save(indexFile)

// Load pre-built index (skip O(n) scan)
index, _ = parser.LoadIndex(indexFile)
```

### Lazy Parser

Combines iteration and indexed random access:

```go
// Create lazy parser
lp := parser.NewLazyParser(file)  // file must be io.ReadSeeker

// Build or load index
lp.BuildIndex()  // Or: lp.LoadIndex(indexReader)

// Random access by XRef - O(1) after indexing
rec, err := lp.FindRecord("@I1@")

// Access special records by type
header, _ := lp.FindRecordByType("HEAD")
trailer, _ := lp.FindRecordByType("TRLR")

// Stream all records
for it := lp.IterateAll(); it.Next(); {
    rec := it.Record()
    // Process...
}

// Save index for next time
lp.SaveIndex(indexWriter)
```

Additional LazyParser methods:

| Method | Return Type | Description |
|--------|-------------|-------------|
| `HasIndex()` | `bool` | Returns true if index is loaded |
| `Index()` | `*RecordIndex` | Returns the current RecordIndex |
| `RecordCount()` | `int` | Total indexed records |
| `XRefs()` | `[]string` | List of all indexed cross-references |
| `IterateFrom(offset)` | `*RecordIterator` | Resume iteration from byte offset |

**Use Cases**:
- Quick validation of file structure without full parse
- Finding specific individuals by ID in large files
- Extracting metadata without loading entire document
- Memory-constrained environments

**Memory**: O(index entries) for indexed access, O(1) per record during iteration.

## Performance

- Zero-allocation validator for valid documents
- Benchmarked performance:
  - Parser: 66ns/op for simple lines
  - Decoder: 13ms for 1000 individuals
  - Encoder: 1.15ms for 1000 individuals
  - Validator: 5.91μs for 1000 individuals

## API Design

- Clean, idiomatic Go API
- Comprehensive godoc documentation
- Example code for common use cases
- Zero external dependencies (standard library only)

### Record Lookup

O(1) lookup by cross-reference ID for all record types:

| Method | Return Type | Description |
|--------|-------------|-------------|
| `GetRecord(xref)` | `*Record` | Generic record lookup |
| `GetIndividual(xref)` | `*Individual` | Individual lookup |
| `GetFamily(xref)` | `*Family` | Family lookup |
| `GetSource(xref)` | `*Source` | Source lookup |
| `GetRepository(xref)` | `*Repository` | Repository lookup |
| `GetSubmitter(xref)` | `*Submitter` | Submitter lookup |
| `GetNote(xref)` | `*Note` | Note lookup |
| `GetMediaObject(xref)` | `*MediaObject` | Media object lookup |

All methods return `nil` if the record is not found (consistent with Go map behavior).

### Collection Accessors

| Method | Return Type | Description |
|--------|-------------|-------------|
| `Individuals()` | `[]*Individual` | All individuals |
| `Families()` | `[]*Family` | All families |
| `Sources()` | `[]*Source` | All sources |
| `Repositories()` | `[]*Repository` | All repositories |
| `Submitters()` | `[]*Submitter` | All submitters |
| `Notes()` | `[]*Note` | All notes |
| `MediaObjects()` | `[]*MediaObject` | All media objects |

### Relationship Traversal

Navigate family relationships with convenience methods that eliminate manual cross-reference resolution:

**Individual Methods:**

| Method | Return Type | Description |
|--------|-------------|-------------|
| `Parents(doc)` | `[]*Individual` | Parents from FAMC families |
| `Spouses(doc)` | `[]*Individual` | Spouses from FAMS families (handles remarriage) |
| `Children(doc)` | `[]*Individual` | Children from all FAMS families |
| `ParentalFamilies(doc)` | `[]*Family` | Families where individual is a child |
| `SpouseFamilies(doc)` | `[]*Family` | Families where individual is a spouse |

**Family Methods:**

| Method | Return Type | Description |
|--------|-------------|-------------|
| `HusbandIndividual(doc)` | `*Individual` | Husband of the family |
| `WifeIndividual(doc)` | `*Individual` | Wife of the family |
| `ChildrenIndividuals(doc)` | `[]*Individual` | Children in GEDCOM order |
| `AllMembers(doc)` | `[]*Individual` | Husband, wife, and children |

All methods:
- Take `*Document` for O(1) cross-reference lookup
- Return `nil` or empty slice for missing/invalid references (never error)
- Preserve order from GEDCOM file

```go
// Navigate from individual to relatives
person := doc.GetIndividual("@I1@")
for _, parent := range person.Parents(doc) {
    fmt.Println(parent.Names[0].Full)
}
for _, spouse := range person.Spouses(doc) {
    fmt.Println(spouse.Names[0].Full)
}
for _, child := range person.Children(doc) {
    fmt.Println(child.Names[0].Full)
}

// Navigate from family to members
family := doc.GetFamily("@F1@")
husband := family.HusbandIndividual(doc)
wife := family.WifeIndividual(doc)
children := family.ChildrenIndividuals(doc)
```

### Event and Date Access

Convenience methods for accessing parsed events and dates on individuals:

| Method | Return Type | Description |
|--------|-------------|-------------|
| `BirthEvent()` | `*Event` | First birth event (nil if none) |
| `DeathEvent()` | `*Event` | First death event (nil if none) |
| `BirthDate()` | `*Date` | Parsed birth date (nil if no event or no date) |
| `DeathDate()` | `*Date` | Parsed death date (nil if no event or no date) |

```go
// Access birth and death events directly
person := doc.GetIndividual("@I1@")
if birth := person.BirthEvent(); birth != nil {
    fmt.Printf("Born: %s at %s\n", birth.Date, birth.Place)
}
if death := person.DeathEvent(); death != nil {
    fmt.Printf("Died: %s\n", death.Date)
}

// Access parsed dates for calculations
if birthDate := person.BirthDate(); birthDate != nil {
    fmt.Printf("Birth year: %d\n", birthDate.Year)
}
```

### Record Type Helpers

Convenience methods on `Record` for type checking and casting:

| Method | Return Type | Description |
|--------|-------------|-------------|
| `IsIndividual()` | `bool` | True if record contains an Individual |
| `IsFamily()` | `bool` | True if record contains a Family |
| `IsSource()` | `bool` | True if record contains a Source |
| `GetIndividual()` | `*Individual` | Type assertion (nil if wrong type) |
| `GetFamily()` | `*Family` | Type assertion (nil if wrong type) |
| `GetSource()` | `*Source` | Type assertion (nil if wrong type) |

```go
for _, record := range doc.Records {
    if record.IsIndividual() {
        indi := record.GetIndividual()
        fmt.Printf("Individual: %s\n", indi.XRef)
    }
}
```

## Testing

- 93% test coverage across core packages
- Multi-platform CI (Linux, macOS, Windows)
- Multi-version Go testing (1.24, 1.25)
- Benchmark regression testing
- Real-world GEDCOM file testing
