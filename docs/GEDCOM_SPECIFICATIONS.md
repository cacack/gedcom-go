# GEDCOM Official Specifications

This document provides links to official GEDCOM specifications and key reference materials for development and compliance auditing.

## Official Specification PDFs

| Version | Release Date | PDF Link | Notes |
|---------|--------------|----------|-------|
| 5.5 | 1996-01-10 | [ged55.pdf](https://gedcom.io/specifications/ged55.pdf) | Original widely-adopted version |
| 5.5.1 | 2019-11-15 | [ged551.pdf](https://gedcom.io/specifications/ged551.pdf) | Clarifications, EMAIL/WWW/MAP tags |
| 7.0 | 2021-06-07 | [FamilySearchGEDCOMv7.pdf](https://gedcom.io/specifications/FamilySearchGEDCOMv7.pdf) | Major overhaul, UTF-8 only |

## Official Resources

- **Specifications Portal**: https://gedcom.io/specs/
- **Migration Guide (5.5.1 → 7.0)**: https://gedcom.io/migrate/
- **GitHub Repository**: https://github.com/FamilySearch/GEDCOM
- **FamilySearch Wiki**: https://www.familysearch.org/en/wiki/GEDCOM

## Community Resources

- **GEDCOM 5.5.5 Annotated** (Tamura Jones): https://webtrees.net/downloads/gedcom-555.pdf
- **GEDCOM-L Addendum**: https://genealogy.net/GEDCOM/GEDCOM551%20GEDCOM-L%20Addendum-R1.pdf

## Version Summary

### GEDCOM 5.5 (1996)
- First widely-adopted version
- ANSEL, ASCII, or ANSI encoding
- Maximum line length: 255 characters
- LDS ordinance support

### GEDCOM 5.5.1 (1999/2019)
- Clarifications to 5.5
- Added: EMAIL, FAX, WWW tags
- Added: MAP, LATI, LONG for coordinates
- UTF-8 support added
- Maximum line length: 255 characters (recommended)

### GEDCOM 7.0 (2021)
- **Breaking changes** from 5.5.1
- UTF-8 encoding only (no ANSEL)
- No maximum line length
- New: EXID (external identifiers)
- New: SCHMA (schema for extensions)
- New: SNOTE (shared notes)
- Restructured multimedia (OBJE as record)
- JSON-LD compatibility considerations
- GEDZip packaging format

## Key Breaking Changes (5.5.1 → 7.0)

1. **Encoding**: UTF-8 only (ANSEL removed)
2. **Line length**: No limit (was 255)
3. **CONC**: Removed (use CONT for all continuations)
4. **Multimedia**: OBJE restructured as top-level record
5. **Notes**: SNOTE introduced for shared notes
6. **Extensions**: Must use SCHMA for custom tags
7. **Header**: Structure changes (GEDC.FORM removed)

## Local Copies

For offline reference, specification PDFs can be downloaded to `docs/specs/`:

```bash
mkdir -p docs/specs
curl -o docs/specs/ged55.pdf https://gedcom.io/specifications/ged55.pdf
curl -o docs/specs/ged551.pdf https://gedcom.io/specifications/ged551.pdf
curl -o docs/specs/gedcom7.pdf https://gedcom.io/specifications/FamilySearchGEDCOMv7.pdf
```

Note: PDF files should be added to `.gitignore` if not intended for version control.
