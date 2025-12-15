# GEDCOM Feature Gaps

**Last Updated**: 2025-12-15

## Overview

This document tracks remaining GEDCOM features not yet implemented.

## Remaining Gaps (Low Priority)

### 1. TRAN (Transliteration) for Names

- Non-Latin script transliterations in PERSONAL_NAME_STRUCTURE
- Impact: Low - primarily for non-Latin genealogy
- Complexity: Medium

### 2. GEDCOM 7.0 Specific Features

- Enhanced ASSO with shared events structure
- PHRASE subordinate for date phrases

### 3. Advanced Validation

- Custom schema rules
- Cross-reference integrity checking beyond basic validation

## Character Encoding (Planned)

See `ENCODING_IMPLEMENTATION_PLAN.md` for details.

- **UTF-16 LE/BE** - 2 test files ready
- **ANSEL** - 4 torture test files ready

## Date Parsing (Planned)

Currently, dates are stored as raw strings. A comprehensive date parsing system is planned.

See `TODO.md` section 7 for phased implementation plan.
See `GEDCOM_DATE_FORMATS_RESEARCH.md` for comprehensive research.

## References

- **GEDCOM 5.5.1 Specification**: https://gedcom.io/specifications/ged551.pdf
- **GEDCOM 7.0 Specification**: https://gedcom.io/specifications/FamilySearchGEDCOMv7.pdf
- **Test Files**: testdata/gedcom-5.5/, testdata/gedcom-5.5.1/, testdata/gedcom-7.0/
