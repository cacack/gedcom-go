# GEDCOM Version Converter

The converter package provides bidirectional conversion between GEDCOM versions 5.5, 5.5.1, and 7.0.

## Overview

The GEDCOM specification has evolved through multiple versions, each with different features and requirements. This converter enables:

- **Upgrading** documents to newer versions (5.5 -> 5.5.1 -> 7.0)
- **Downgrading** documents to older versions (7.0 -> 5.5.1 -> 5.5)
- **Tracking** all transformations and data loss in a detailed report

## Quick Start

```go
import (
    "github.com/cacack/gedcom-go/v3/converter"
    "github.com/cacack/gedcom-go/v3/decoder"
    "github.com/cacack/gedcom-go/v3/gedcom"
)

// Load a GEDCOM 5.5 file
doc, _ := decoder.Decode(reader)

// Convert to GEDCOM 7.0
converted, report, err := converter.Convert(doc, gedcom.Version70)
if err != nil {
    log.Fatal(err)
}

// Check what changed
fmt.Println(report)
```

## API Reference

### Convert

```go
func Convert(doc *gedcom.Document, targetVersion gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, error)
```

Converts a document to the target version using default options. The original document is not modified; a deep copy is created for conversion.

When the document is already at the target version, it is returned as-is and the report is empty. The exception is 7.0 → 7.0: XRefs the 7.0 grammar rejects (`0 @i 2@ INDI`) are normalized on a copy exactly as in [XRef normalization](#upgrade-to-gedcom-70), pointers included, and each rename is a `Normalized` note, so converting to 7.0 always yields valid 7.0 identifiers. No other transformation runs. A dangling pointer containing a space (`1 NOTE @X 9@`) is not a pointer under the 7.0 grammar a 7.0 header declares, so it is left as-is. A 7.0 → 7.0 call therefore returns the input itself when nothing needs renaming and a copy otherwise, so do not rely on the result's pointer identity.

### ConvertWithOptions

```go
func ConvertWithOptions(doc *gedcom.Document, targetVersion gedcom.Version, opts *ConvertOptions) (*gedcom.Document, *gedcom.ConversionReport, error)
```

Converts a document with custom options for fine-grained control over the conversion process.

A document already at the target version is handled as described under [Convert](#convert); the options do not apply on that path.

### ConvertOptions

| Option | `DefaultOptions` | Description |
|--------|------------------|-------------|
| `Validate` | `true` | Run validation on converted document |
| `StrictDataLoss` | `false` | Fail if any data would be lost |
| `ReportPreservedTags` | `true` | Itemise each preserved vendor/unknown tag in the report. Report-only — tags are preserved either way |
| `MapEXIDToVendorTags` | `true` | On a 7.0 downgrade, map a FamilySearch ARK EXID to `_FSFTID` instead of recording it as data loss |

Nothing the converter does drops a tag: preservation is unconditional, and
`ReportPreservedTags` only controls whether the report lists what was kept.

`nil` and a wholly zero `&converter.ConvertOptions{}` both mean "use the
defaults". A **partially** populated literal does not — each field you leave
out takes its own zero value, so start from `DefaultOptions()` to change one
setting:

```go
opts := converter.DefaultOptions()
opts.StrictDataLoss = true  // Fail on any data loss
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version70, opts)
```

## Version-Specific Transformations

### Upgrade to GEDCOM 7.0

| Transformation | Type | Description |
|---------------|------|-------------|
| CONC removal | `CONC_REMOVED` | Line continuations merged into single values, at any depth, including those continuing a level-0 line such as a 5.x `NOTE` record's text |
| XRef normalization | `XREF_UPPERCASE` | Cross-references rewritten to the 7.0 form `@[A-Z0-9_]+@`: uppercased, other characters (such as 5.5.1 spaces) replaced by `_`, and a `_2`, `_3`, ... suffix added on collision with an existing record or pointer (never `@VOID@`); pointers follow, and each rewrite is a `Normalized` note. A record named `@VOID@`, an ordinary 5.5/5.5.1 identifier but the 7.0 null pointer, is renamed the same way (`@VOID_2@`) so its pointers do not become null; a `@VOID@` pointer no record defines stays the null pointer. A pointer to an undefined XRef joins a record only when they differ by ASCII case alone, otherwise it gets its own identifier. A dangling spaced pointer (`1 NOTE @X 9@`) is rewritten only when the source header declares 5.5 or 5.5.1 in `GEDC.VERS`; without that declaration it is left as-is. `Count` includes rewritten dangling pointers |
| Note records | `NOTE_RECORD_TO_SNOTE` | `0 @N1@ NOTE` records become `0 @N1@ SNOTE` shared note records (the typed `Note` becomes a `SharedNote`); each is a `Normalized` note |
| Note pointers | `NOTE_POINTER_TO_SNOTE` | `NOTE @N1@` pointers to a converted note record become `SNOTE @N1@`, at any level; inline note text and pointers no record defines stay `NOTE` |
| Media types | `MEDIA_TYPE_MAPPED` | Legacy formats (JPG) converted to IANA (image/jpeg) |
| Date calendars | `DATE_CALENDAR_CONVERTED` | Calendar escapes (`@#DJULIAN@`) converted to keywords (`JULIAN`) |
| Header update | `VERSION_UPGRADE` | Header version updated to 7.0 |

### Upgrade 5.5 to 5.5.1

| Transformation | Type | Description |
|---------------|------|-------------|
| Header update | `VERSION_UPGRADE` | Backward compatible upgrade |
| GEDC.FORM | `GEDC_FORM_ADDED` | `2 FORM LINEAGE-LINKED` added under `GEDC` when missing |

### Downgrade from GEDCOM 7.0

| Transformation | Type | Description |
|---------------|------|-------------|
| Newlines to CONT | `CONT_EXPANDED` | Embedded newlines expanded to CONT tags |
| Long lines to CONC | `CONC_ADDED` | A value line too long for 5.x's 255-character limit split onto CONC tags, between two non-spaces, including a level-0 `SNOTE` record's text; each split value is a `Normalized` note |
| Media types | `MEDIA_TYPE_MAPPED` | IANA formats converted to legacy |
| Date calendars | `DATE_CALENDAR_CONVERTED` | Calendar keywords (`JULIAN`) converted to escapes (`@#DJULIAN@`) |
| Header update | `VERSION_DOWNGRADE` | Header version updated |
| GEDC.FORM | `GEDC_FORM_ADDED` | `2 FORM LINEAGE-LINKED` added under `GEDC` when missing (required by 5.5 and 5.5.1) |

Every conversion sets the header's `GEDC.VERS` to the target version, rewriting
any existing `VERS` under any `GEDC`. If no `GEDC` has a `VERS`, one is added
under the first `GEDC`, keeping its other children (`FORM` is kept for 5.5 and
5.5.1 targets but dropped for 7.0, which removed it). A header with no `GEDC`
gets one added. A 5.5 or 5.5.1 target's `GEDC` always carries
`FORM LINEAGE-LINKED`, after `VERS`.

The header's level-1 structures are then put in the target version's grammar
order -- `SOUR, DEST, DATE, SUBM, SUBN, FILE, COPR, GEDC, CHAR, LANG, PLAC,
NOTE` for 5.5/5.5.1, `GEDC, SCHMA, SOUR, DEST, DATE, SUBM, COPR, LANG, PLAC,
NOTE` for 7.0 -- each moving with its subordinate lines. Structures the target
grammar does not define (extensions, or `FILE` in 7.0) go last in source order.
A same-version conversion returns the document unchanged, so its header keeps
the source's order. The one header repair it makes is `VERS`: a `GEDC` with no
`VERS` (such a header is detected by tag fallback) gets `2 VERS <target>` added
as its first child, on a copy. A header with no `GEDC`, or one that already
declares `VERS`, is left as it is.

## Media Type Mappings

When converting between versions, media types are automatically mapped:

| Legacy (5.5/5.5.1) | IANA (7.0) |
|-------------------|------------|
| JPG, JPEG | image/jpeg |
| PNG | image/png |
| GIF | image/gif |
| TIFF, TIF | image/tiff |
| BMP | image/bmp |
| MP3 | audio/mpeg |
| WAV | audio/wav |
| MP4 | video/mp4 |
| MPEG, MPG | video/mpeg |
| AVI | video/x-msvideo |
| PDF | application/pdf |
| TXT, TEXT | text/plain |

## Date Calendar Mappings

GEDCOM 5.5/5.5.1 marks a date's calendar with an escape; GEDCOM 7.0 uses a
keyword. Converting to or from 7.0 rewrites each one:

| 5.5/5.5.1 escape | 7.0 keyword |
|------------------|-------------|
| `@#DGREGORIAN@` | `GREGORIAN` |
| `@#DJULIAN@` | `JULIAN` |
| `@#DHEBREW@` | `HEBREW` |
| `@#DFRENCH R@` | `FRENCH_R` |

A calendar belongs to each date, so both endpoints of a range or period are
translated (`BET @#DJULIAN@ 1700 AND @#DGREGORIAN@ 1710` becomes
`BET JULIAN 1700 AND GREGORIAN 1710`); date phrase text in parentheses is left
alone. Every `DATE` and `SDATE` value in `Record.Tags` and `Header.Tags` (such
as `HEAD.SOUR.DATA.DATE`) is rewritten, as are `Header.Date` and
`Date`/`SortDate` on individual and family events, attributes and LDS
ordinances (with `ParsedDate.Original` kept in step), and the `Data.Date` of
source citations on individual, family, media and shared note records and on
their events, attributes and associations. Each rewrite is a Normalized note.

A calendar with no standard equivalent -- a 5.x escape such as `@#DROMAN@` or
`@#DUNKNOWN@`, or a 7.0 extension calendar such as `_MAYAN` -- is left
unchanged and reported as a Preserved note, so the output may not be valid in
the target version for that date. This is not data loss, so `StrictDataLoss`
does not reject it: check `report.Preserved` for these notes when the output
must conform to the target version.

## Data Loss Reference

### 5.5 / 5.5.1 -> 7.0

| Feature | Reason |
|---------|--------|
| Dual-year dates | 7.0 has no dual-year syntax (`11 FEB 1731/32`); the date is left unchanged, so the output is not valid 7.0 for it. Each such date is also a `Preserved` note at its path, and `AffectedRecords` lists each record holding one. Dual years inside a date phrase (`(born 1799/00)`) are not flagged |

The `SNOTE` records this direction produces become `NOTE` records again when
converted back down to 5.x (see below).

### 7.0 -> 5.5.1 / 5.5

| Feature | Reason |
|---------|--------|
| NO tags | Negative assertions not supported; removed from the output (the `NO` line and its subordinates, and the negated event from `Individual.Events` / `Family.Events`) |
| EXID tags | External identifiers not supported |
| TRAN tags | Translation records not supported |
| PHRASE tags | Phrase annotations not supported |
| UID tags | Unique identifiers not supported in 5.x |
| CREA tags | Creation date not supported |
| `@VOID@` null pointers | 5.x has no null pointer; a kept `@VOID@` would dangle |

Each `EXID`, `TRAN`, `PHRASE`, `UID` and `CREA` is removed from the record's
`Record.Tags` with its subordinate lines, at any level, so the output matches
the report; a match nested inside a removed block is covered by the outer
entry. The typed entity's record-level fields for them (`CreationDate`,
`UIDs`, `ExternalIDs`, and a shared note's `Translations`) are cleared too,
because the encoder writes a record without `Tags` from its entity; a record
built in code without `Tags` is reported from those fields. Nested entity
fields, such as an event's `UIDs`, are not cleared, so a record built in code
can still write them. A FamilySearch ARK `EXID` mapped to `_FSFTID`
(`MapEXIDToVendorTags`) is not data loss.

Each `@VOID@` pointer is removed with the structure that holds it: a tag whose
value is `@VOID@` (`1 FAMC @VOID@`, `2 SOUR @VOID@`, ...) goes with its
subordinate lines, and the matching typed structure (a `FamilyLink`,
citation, media link, association, repository link or note pointer) is
removed from the entity; `HUSB`, `WIFE`, an LDS `FAMC` and the header `SUBM`
are cleared. Each is a `Dropped` note, and the conversion reports one
`@VOID@ null pointers` data loss entry, so `Subset` and the validator never
meet a dangling `@VOID@` in the output.

Shared notes are not lost: `0 @N1@ SNOTE` records become `0 @N1@ NOTE`
records (the typed `SharedNote` becomes a `Note`) and every `SNOTE` tag becomes
`NOTE`, reported as a `Normalized` note per record and the
`SNOTE_RECORD_TO_NOTE` and `SNOTE_POINTER_TO_NOTE` transformations. A shared
note's level-1 `MIME` and `LANG`, which a 5.x `NOTE` record does not define,
are removed and reported as `Shared note MIME` and `Shared note LANG` data
loss.

Only the converter's 7.0 downgrade removes negative assertions. The encoder
writes a negated event as `1 NO <EVENT>` whatever the header version, so a
document built in code with `IsNegative` events and a 5.5/5.5.1 header, or a
5.5.1 -> 5.5 conversion of a document that carries `NO`, still produces `NO`
lines. For a document built in code, set `Events` to `OccurredEvents()` on
each individual and family before encoding for a 5.x target; a decoded 5.x
document encodes from `Record.Tags`, so remove its level-1 `NO` tags there.

### 5.5.1 -> 5.5

| Feature | Reason |
|---------|--------|
| EMAIL | Tag introduced in 5.5.1 |
| FAX | Tag introduced in 5.5.1 |
| WWW | Tag introduced in 5.5.1 |
| FACT | Generic fact tag introduced in 5.5.1 |
| MAP, LATI, LONG | Geographic coordinates introduced in 5.5.1 |

Note: These tags are preserved but may not be recognized by strict 5.5 parsers.

## ConversionReport Structure

The `ConversionReport` provides detailed information about what changed during conversion:

```go
type ConversionReport struct {
    SourceVersion    Version          // Original GEDCOM version
    TargetVersion    Version          // Converted GEDCOM version
    Transformations  []Transformation // All changes made
    DataLoss         []DataLossItem   // Features lost (typically downgrades)
    ValidationIssues []string         // Problems found after conversion
    Success          bool             // Whether conversion completed successfully
}
```

### Transformation

```go
type Transformation struct {
    Type        string   // Short identifier (e.g., "XREF_UPPERCASE")
    Description string   // Human-readable explanation
    Count       int      // Number of instances transformed
    Details     []string // Optional specific information
}
```

### DataLossItem

```go
type DataLossItem struct {
    Feature         string   // Name of lost feature
    Reason          string   // Why the feature was lost
    AffectedRecords []string // XRefs of affected records
}
```

### Report Methods

| Method | Description |
|--------|-------------|
| `HasDataLoss()` | Returns true if any data was lost |
| `String()` | Human-readable summary of the conversion |

Example output from `report.String()`:

```
Conversion: 5.5 -> 7.0
Success: true
Transformations: 2
  - CONC_REMOVED: Consolidated CONC continuation tags into parent values (15 instances)
  - XREF_UPPERCASE: Normalized XRefs to uppercase letters, digits and underscore (required for GEDCOM 7.0) (8 instances)
```

## Strict Mode

When `StrictDataLoss` is enabled, the conversion fails if any data would be lost:

```go
opts := converter.DefaultOptions()
opts.StrictDataLoss = true
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version55, opts)
if err != nil {
    // Error: "conversion would result in data loss (strict mode enabled)"
    // Check report.DataLoss for details
}
```

## Related Documentation

- [GEDCOM Versions](../reference/gedcom-versions.md) - Detailed version specifications
- [Encoding Implementation](../archive/encoding-implementation-plan.md) - Character encoding details (archived)
