# GEDCOM Version Specifications

This document summarizes the key differences between GEDCOM versions 5.5, 5.5.1, and 7.0. This library supports all three versions.

## Quick Reference

| Feature | GEDCOM 5.5 | GEDCOM 5.5.1 | GEDCOM 7.0 |
|---------|------------|--------------|------------|
| Default Encoding | ANSEL | ANSEL | UTF-8 (mandatory) |
| UTF-8 Support | No | Yes | Required |
| UTF-16 Support | Yes (with BOM) | Yes (with BOM) | No |
| CONC Continuation | Yes | Yes | Removed |
| CONT Continuation | Yes | Yes | Replaced with multiline |
| XRef Format | `@[A-Za-z0-9_]+@` | `@[A-Za-z0-9_]+@` | `@[A-Z0-9_]+@` (uppercase) |

## GEDCOM 5.5 Specification

### Line Format

- Format: `LEVEL [XREF] TAG [VALUE]`
- Level: 0-99 (must increment by 1 from parent)
- XREF: Optional, format `@[A-Za-z0-9_]+@` (alphanumeric + underscore)
- TAG: 3-4 uppercase characters (standard tags) or `_TAG` for custom extensions
- VALUE: Optional, max 255 characters per line
- Line terminators: CRLF (recommended), LF, or CR
- Continuation: Use CONC (concatenate) or CONT (new line) tags at level+1

### Required Tags

- Header: `HEAD`, `GEDC`, `VERS`, `CHAR`, `SOUR`, `SUBM`
- Trailer: `TRLR`
- Individual: `NAME` (at least one)
- Family: At least one `HUSB`, `WIFE`, or `CHIL`

### Character Encoding (GEDCOM 5.5)

- ANSEL (default if not declared)
- ASCII (7-bit)
- UNICODE (UTF-16 with BOM)
- ANSI (Windows-1252)

### Header Structure

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

### Cross-Reference Format

- Pattern: `@[A-Za-z0-9_]+@`
- Examples: `@I1@`, `@F123@`, `@S_001@`
- Must be unique within document
- Forward references allowed (referenced before defined)

## GEDCOM 5.5.1 Specification

### Changes from 5.5

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

### Character Encoding (GEDCOM 5.5.1)

- ANSEL (default)
- ASCII
- UTF-8 (NEW - recommended)
- UTF-16 (with BOM)

### Header Version Declaration

```
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
```

## GEDCOM 7.0 Specification

### Breaking Changes from 5.5.1

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

### Header Structure

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

### Media Types (GEDCOM 7.0)

- Image: `image/jpeg`, `image/png`, `image/gif`, `image/tiff`, `image/bmp`
- Audio: `audio/mpeg`, `audio/wav`
- Video: `video/mp4`, `video/mpeg`
- Document: `application/pdf`, `text/plain`

### Removed Features

- ANSEL encoding (UTF-8 only)
- CONC tag (use longer lines)
- CONT tag (replaced with structured multiline values)
- Some custom tags standardized or removed

### Backward Compatibility

GEDCOM 7.0 is NOT backward compatible with 5.5.1. Files must be converted.

## ANSEL Character Encoding

ANSEL (ANSI Z39.47) is a character encoding used in legacy GEDCOM 5.5 files, particularly for representing characters with diacritical marks common in genealogical records.

### ANSEL Byte Ranges

- 0x00-0x7F: Standard ASCII
- 0x80-0x9F: Control characters (unused)
- 0xA0-0xCF: Spacing characters (special chars, currency, etc.)
- 0xE0-0xFF: Combining diacritics (placed BEFORE base character)

### Key ANSEL Combining Characters

| Byte | Diacritic |
|------|-----------|
| 0xE0 | Hook above |
| 0xE1 | Grave accent |
| 0xE2 | Acute accent |
| 0xE3 | Circumflex |
| 0xE4 | Tilde |
| 0xE5 | Macron |
| 0xE6 | Breve |
| 0xE7 | Dot above |
| 0xE8 | Umlaut/diaeresis |
| 0xE9 | Caron |
| 0xEA | Ring above |
| 0xF0 | Cedilla |
| 0xF6 | Underscore |
| 0xF7 | Comma below |

### ANSEL to Unicode Conversion

ANSEL places combining characters BEFORE the base character, while Unicode places them AFTER:

- ANSEL: `0xE2 0x65` (acute + e)
- Unicode: `0x65 0x0301` (e + combining acute)

Order reversal is required when converting between ANSEL and UTF-8.

## Version Detection

This library auto-detects the GEDCOM version by:

1. Parsing the header looking for `HEAD.GEDC.VERS` path
2. Extracting version from the `VERS` tag value
3. If missing/ambiguous, analyzing tag names against version-specific lists

The detected version is stored in `Document.Header.GedcomVersion`.

## Related Documentation

- [Date Format Research](GEDCOM_DATE_FORMATS_RESEARCH.md) - Detailed date parsing specifications
- [Encoding Implementation](ENCODING_IMPLEMENTATION_PLAN.md) - Character encoding implementation details
