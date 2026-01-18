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
    "github.com/cacack/gedcom-go/converter"
    "github.com/cacack/gedcom-go/decoder"
    "github.com/cacack/gedcom-go/gedcom"
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

| Option | Default | Description |
|--------|---------|-------------|
| `Validate` | `true` | Run validation on converted document |
| `StrictDataLoss` | `false` | Fail if any data would be lost |
| `PreserveUnknownTags` | `true` | Keep vendor extensions and unknown tags |

```go
opts := &converter.ConvertOptions{
    Validate:            true,
    StrictDataLoss:      true,  // Fail on any data loss
    PreserveUnknownTags: true,
}
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version70, opts)
```

## Version-Specific Transformations

### Upgrade to GEDCOM 7.0

| Transformation | Type | Description |
|---------------|------|-------------|
| CONC removal | `CONC_REMOVED` | Line continuations merged into single values |
| CONT to newlines | `CONT_CONVERTED` | Continuation lines converted to embedded newlines |
| XRef uppercase | `XREF_UPPERCASE` | All cross-references normalized to uppercase |
| Media types | `MEDIA_TYPE_MAPPED` | Legacy formats (JPG) converted to IANA (image/jpeg) |
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
| Header update | `VERSION_DOWNGRADE` | Header version updated |

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

## Data Loss Reference

### 7.0 -> 5.5.1 / 5.5

| Feature | Reason |
|---------|--------|
| EXID tags | External identifiers not supported |
| NO tags | Negative assertions not supported |
| TRAN tags | Translation records not supported |
| PHRASE tags | Phrase annotations not supported |
| UID tags | Unique identifiers not supported in 5.x |
| CREA tags | Creation date not supported |
| SNOTE tags | Shared notes not supported |

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
  - XREF_UPPERCASE: Normalized cross-references to uppercase (8 instances)
```

## Strict Mode

When `StrictDataLoss` is enabled, the conversion fails if any data would be lost:

```go
opts := &converter.ConvertOptions{
    StrictDataLoss: true,
}
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version55, opts)
if err != nil {
    // Error: "conversion would result in data loss (strict mode enabled)"
    // Check report.DataLoss for details
}
```

## Related Documentation

- [GEDCOM Versions](GEDCOM_VERSIONS.md) - Detailed version specifications
- [Encoding Implementation](ENCODING_IMPLEMENTATION_PLAN.md) - Character encoding details
