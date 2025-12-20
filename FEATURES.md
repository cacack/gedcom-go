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
| LATIN1 (ISO-8859-1) | Recognized | Declared but not converted |
| UTF-16 LE/BE | Planned | [Issue #29](https://github.com/cacack/gedcom-go/issues/29) |
| ANSEL | Planned | [Issue #30](https://github.com/cacack/gedcom-go/issues/30) |

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

## Testing

- 93% test coverage across core packages
- Multi-platform CI (Linux, macOS, Windows)
- Multi-version Go testing (1.21, 1.22, 1.23)
- Benchmark regression testing
- Real-world GEDCOM file testing
