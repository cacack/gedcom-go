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

### ConvertWithOptions

```go
func ConvertWithOptions(doc *gedcom.Document, targetVersion gedcom.Version, opts *ConvertOptions) (*gedcom.Document, *gedcom.ConversionReport, error)
```

Converts a document with custom options for fine-grained control over the conversion process.

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
| CONC removal | `CONC_REMOVED` | Line continuations merged into single values |
| CONT to newlines | `CONT_CONVERTED` | Continuation lines converted to embedded newlines |
| XRef normalization | `XREF_UPPERCASE` | Cross-references rewritten to the 7.0 form `@[A-Z0-9_]+@`: uppercased, other characters (such as 5.5.1 spaces) replaced by `_`, and a `_2`, `_3`, ... suffix added on collision with an existing record or pointer (never `@VOID@`); pointers follow, and each rewrite is a `Normalized` note. A pointer to an undefined XRef joins a record only when they differ by ASCII case alone, otherwise it gets its own identifier. A dangling spaced pointer (`1 NOTE @X 9@`) is rewritten only when the source header declares 5.5 or 5.5.1 in `GEDC.VERS`; without that declaration it is left as-is. `Count` includes rewritten dangling pointers |
| Media types | `MEDIA_TYPE_MAPPED` | Legacy formats (JPG) converted to IANA (image/jpeg) |
| Date calendars | `DATE_CALENDAR_CONVERTED` | Calendar escapes (`@#DJULIAN@`) converted to keywords (`JULIAN`) |
| Header update | `VERSION_UPGRADE` | Header version updated to 7.0 |

### Upgrade 5.5 to 5.5.1

| Transformation | Type | Description |
|---------------|------|-------------|
| Header update | `VERSION_UPGRADE` | Backward compatible upgrade |

### Downgrade from GEDCOM 7.0

| Transformation | Type | Description |
|---------------|------|-------------|
| Newlines to CONT | `CONT_EXPANDED` | Embedded newlines expanded to CONT tags |
| Media types | `MEDIA_TYPE_MAPPED` | IANA formats converted to legacy |
| Date calendars | `DATE_CALENDAR_CONVERTED` | Calendar keywords (`JULIAN`) converted to escapes (`@#DJULIAN@`) |
| Header update | `VERSION_DOWNGRADE` | Header version updated |

Every conversion sets the header's `GEDC.VERS` to the target version. A `GEDC`
with no `VERS` gets one added under it, keeping its other children (`FORM` is
kept for 5.5 and 5.5.1 targets but dropped for 7.0, which removed it). A header
with no `GEDC` gets one added.

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
alone. Every `DATE` and `SDATE` value in `Record.Tags` is rewritten, as is
`Date`/`SortDate` on individual and family events, attributes and LDS
ordinances (with `ParsedDate.Original` kept in step), and the `Data.Date` of
source citations on individual, family, media and shared note records and on
their events, attributes and associations. Each rewrite is a Normalized note.
The header's dates are not rewritten.

A calendar with no standard equivalent -- a 5.x escape such as `@#DROMAN@` or
`@#DUNKNOWN@`, or a 7.0 extension calendar such as `_MAYAN` -- is left
unchanged and reported as a Preserved note, so the output may not be valid in
the target version for that date. This is not data loss, so `StrictDataLoss`
does not reject it: check `report.Preserved` for these notes when the output
must conform to the target version.

## Data Loss Reference

### 7.0 -> 5.5.1 / 5.5

| Feature | Reason |
|---------|--------|
| EXID tags | External identifiers not supported |
| NO tags | Negative assertions not supported; removed from the output (the `NO` line and its subordinates, and the negated event from `Individual.Events` / `Family.Events`) |
| TRAN tags | Translation records not supported |
| PHRASE tags | Phrase annotations not supported |
| UID tags | Unique identifiers not supported in 5.x |
| CREA tags | Creation date not supported |
| SNOTE tags | Shared notes not supported |

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
Transformations: 3
  - CONC_REMOVED: Consolidated CONC continuation tags into parent values (15 instances)
  - CONT_CONVERTED: Converted CONT tags to embedded newlines (42 instances)
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
