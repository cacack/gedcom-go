# Features

This document provides a comprehensive list of all features implemented in gedcom-go.

For planned features, see [GitHub Issues](https://github.com/cacack/gedcom-go/issues).

## Multi-Version Support

| Version | Status | Notes |
|---------|--------|-------|
| GEDCOM 5.5 | Full | Legacy format support |
| GEDCOM 5.5.1 | Full | Most common format |
| GEDCOM 7.0 | Full | Latest standard |

- Automatic version detection from header
- Heuristic-based detection for malformed headers
- Version-aware validation rules

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
- Notes on associations

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

### Edge Cases

| Format | Example | Notes |
|--------|---------|-------|
| B.C. dates | `44 BC`, `753 B.C.E.` | IsBC flag set |
| Dual dating | `21 FEB 1750/51` | Both years accessible |
| Date phrases | `(unknown)` | GEDCOM 5.5 format |

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

// Convert to time.Time (complete dates only)
t, err := date.ToTime()

// Get original string
s := date.String()  // "25 DEC 2020"
```

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

### Error Reporting
- Line numbers for all errors
- Error categorization (error, warning)
- Clear error messages with context

## Encoder

- Write valid GEDCOM files
- Configurable line endings (LF, CRLF)
- GEDCOM 5.5, 5.5.1, 7.0 output
- UTF-8 output

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

## Testing

- 93% test coverage across core packages
- Multi-platform CI (Linux, macOS, Windows)
- Multi-version Go testing (1.21, 1.22, 1.23)
- Benchmark regression testing
- Real-world GEDCOM file testing
