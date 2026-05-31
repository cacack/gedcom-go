# Vendor Extensions

Genealogy platforms extend GEDCOM with proprietary, underscore-prefixed tags
(e.g. `_APID`, `_FSFTID`) that carry information the base specification has no
field for. This document is the single reference for how gedcom-go handles those
extensions: which tags are parsed into typed Go fields, which are preserved
as-is, and how to validate them.

For the broader "which vendors are supported and how" picture, see
[COMPATIBILITY.md](COMPATIBILITY.md). For the exhaustive feature list, see
[FEATURES.md](../FEATURES.md).

## Two ways extensions are handled

gedcom-go follows a **lossless** principle (one of the project's two
non-negotiable guarantees — see [ETHOS.md](ETHOS.md)): no data is ever dropped
on decode/encode. Vendor extensions are handled in one of two ways:

1. **Typed accessors** — A handful of high-value, widely-used tags are parsed
   into dedicated Go fields with helper methods (e.g. `_APID` →
   `SourceCitation.AncestryAPID`). These give you structured, convenient access.
2. **Preserved raw tags** — Every other custom tag is retained verbatim in the
   `Tags []*gedcom.Tag` slice on the entity it belongs to. Nothing is lost; the
   data is just not promoted to a typed field.

Both paths round-trip: a decode followed by an encode reproduces the vendor tags.

### The raw `Tags` escape hatch

Any tag without a typed accessor is available on the owning entity's `Tags`
field. The structure preserves level, value, and children:

```go
type Tag struct {
    Level    int
    Tag      string   // e.g. "_MILT", "_HASH"
    Value    string
    Xref     string   // e.g. "@I1@", if any
    Children []*Tag   // subordinate tags
}
```

Example — reading a tag that has no typed field (FamilySearch `_HASH`):

```go
for _, t := range indi.Tags {
    if t.Tag == "_HASH" {
        fmt.Println("change-detection checksum:", t.Value)
    }
}
```

## Vendor detection

The originating software is detected from the `HEAD.SOUR` value and exposed on
`Document.Vendor`:

```go
doc, _ := decoder.Decode(file)

switch doc.Vendor {
case gedcom.VendorAncestry:
    // handle Ancestry extensions
case gedcom.VendorFamilySearch:
    // handle FamilySearch extensions
}

fmt.Println(doc.Vendor.String())  // "ancestry", "familysearch", ... or "unknown"
fmt.Println(doc.Vendor.IsKnown()) // false for VendorUnknown
```

Detection is case-insensitive substring matching (`gedcom.DetectVendor`).
Recognized vendors: `VendorAncestry` (includes FamilyTreeMaker),
`VendorFamilySearch`, `VendorRootsMagic`, `VendorLegacy`, `VendorGramps`,
`VendorMyHeritage`. Unrecognized sources return `VendorUnknown` — never an error.

> **Note:** Some vendors do not claim authorship in `HEAD.SOUR`. FamilySearch,
> for example, preserves the *original* authoring system, so a FamilySearch
> export may report a different (or unknown) vendor. Its extensions are still
> parsed and preserved regardless of detection.

## Ancestry.com

| Tag | Location | Access |
|-----|----------|--------|
| `_APID` | source citation | **typed:** `SourceCitation.AncestryAPID` |
| `_TREE` | header (`HEAD.SOUR`) | **typed:** `Header.AncestryTreeID` |
| `_MILT` | individual | raw `Tags` (validated by registry) |
| `_DEST` | emigration/immigration event | raw `Tags` (validated by registry) |
| `_PRIM` | media object | raw `Tags` (validated by registry, Y/N) |
| `_PHOTO` | individual | raw `Tags` (validated by registry) |

### `_APID` — Ancestry Permanent ID

The `_APID` is Ancestry's stable identifier for an attached record, distinct
from the `SOUR` cross-reference (which can change between exports). It appears on
source citations and has the form `1,DATABASE::RECORD` (the `1,` prefix is a
type indicator). gedcom-go parses it into an `AncestryAPID` and can reconstruct
the original record URL:

```go
type AncestryAPID struct {
    Raw      string // original value, e.g. "1,7602::2771226"
    Database string // "7602"
    Record   string // "2771226"
}
```

```go
// Source citations live on individuals, events, attributes, and associations.
indi := doc.GetIndividual("@I1@")
for _, ev := range indi.Events {
    for _, cite := range ev.SourceCitations {
        if cite.AncestryAPID != nil {
            fmt.Println(cite.AncestryAPID.Database) // "7602"
            fmt.Println(cite.AncestryAPID.Record)   // "2771226"
            fmt.Println(cite.AncestryAPID.URL())    // https://www.ancestry.com/discoveryui-content/view/2771226:7602
        }
    }
}
```

You can also parse an APID string directly with `gedcom.ParseAPID`, which
returns `nil` for unparseable input.

> `_APID` tags only appear when records are attached to individuals in the
> source tree, not in basic exports.

### `_TREE` — tree reference

Found under `HEAD.SOUR`, identifying the originating Ancestry tree:

```go
fmt.Println(doc.Header.AncestryTreeID) // "12345678"
```

## FamilySearch

| Tag | Location | Access |
|-----|----------|--------|
| `_FSFTID` | individual | **typed:** `Individual.FamilySearchID` |
| `EXID` | individual / source | **typed:** `ExternalIDs` (GEDCOM 7.0) |
| `_FSORD` | individual | raw `Tags` (validated by registry) |
| `_FSTAG` | any | raw `Tags` (validated by registry) |
| `_HASH`, `_LHASH` | individual | raw `Tags` (change-detection checksums) |

### `_FSFTID` — FamilySearch Family Tree ID

The stable person identifier in the FamilySearch shared tree (e.g. `KWCJ-QN7`):

```go
if indi.FamilySearchID != "" {
    fmt.Println(indi.FamilySearchID)    // "KWCJ-QN7"
    fmt.Println(indi.FamilySearchURL()) // https://www.familysearch.org/tree/person/details/KWCJ-QN7
}
```

### `_HASH` / `_LHASH`

FamilySearch emits per-record change-detection checksums. These have no typed
field but are preserved in raw `Tags` (see the escape-hatch example above).

## MyHeritage

MyHeritage exports are detected (`VendorMyHeritage`) and preserved losslessly.
None of its extensions have typed accessors today; all are reachable via raw
`Tags` (records) and `Header.Tags` (header).

| Tag | Location | Notes |
|-----|----------|-------|
| `_UID`, `RIN` | individual | per-record identifiers, preserved in raw `Tags` |
| `_RTLSAVE`, `_PROJECT_GUID`, `_EXPORTED_FROM_SITE_ID` | header | preserved in `Header.Tags` |

> **Behavioral note:** MyHeritage exports omit `REPO` (repository) records, so a
> decoded MyHeritage document typically has zero repositories. This reflects the
> source file, not a gedcom-go limitation.

## RootsMagic

Detected (`VendorRootsMagic`) and preserved. Its tags are covered by the
validation registry; none have typed accessors.

| Tag | Location | Notes |
|-----|----------|-------|
| `_PRIM` | any | primary indicator (Y/N) — more permissive than Ancestry's `_PRIM` |
| `_SDATE` | any | sort date, separate from the display date |
| `_TMPLT` | source | source-template reference |

## Other vendors

Gramps, Legacy Family Tree, Heredis, Family Historian, and FamilyTreeMaker
exports are parsed and their custom tags preserved losslessly via raw `Tags`.
Gramps and Legacy are recognized by vendor detection (`VendorGramps`,
`VendorLegacy`); the others fall through to `VendorUnknown` but are still fully
preserved. See [COMPATIBILITY.md](COMPATIBILITY.md) for the per-vendor support
matrix and tested fixtures.

## Typed vs preserved-raw — quick reference

| Tag | Vendor | Typed field / method | Preserved raw |
|-----|--------|----------------------|:-------------:|
| `_APID` | Ancestry | `SourceCitation.AncestryAPID` (`.URL()`) | ✓ |
| `_TREE` | Ancestry | `Header.AncestryTreeID` | ✓ |
| `_FSFTID` | FamilySearch | `Individual.FamilySearchID` (`.FamilySearchURL()`) | ✓ |
| `EXID` | FamilySearch (7.0) | `Individual.ExternalIDs`, `Source.ExternalIDs` | ✓ |
| `_MILT`, `_DEST`, `_PRIM`, `_PHOTO` | Ancestry | — | ✓ |
| `_FSORD`, `_FSTAG`, `_HASH`, `_LHASH` | FamilySearch | — | ✓ |
| `_UID`, `RIN`, header exts | MyHeritage | — | ✓ |
| `_PRIM`, `_SDATE`, `_TMPLT` | RootsMagic | — | ✓ |
| any other `_`-prefixed tag | any | — | ✓ |

Everything is preserved; the typed column lists the convenience accessors.

## GEDCOM 7.0: `EXID` and `SCHMA`

GEDCOM 7.0 standardizes two mechanisms relevant to extensions:

- **`EXID`** (external identifier) — links a record to an external system. It is
  parsed into a typed `ExternalID`:

  ```go
  type ExternalID struct {
      Value string // the identifier, e.g. "9876543210"
      Type  string // the system URI from the TYPE subordinate
  }
  ```

  Available on `Individual.ExternalIDs` and `Source.ExternalIDs`. This is the
  spec-compliant successor to vendor-specific ID tags like `_FSFTID`.

- **`SCHMA`** — the header schema block that maps extension tags to their URI
  definitions, enabling interoperability of custom tags between applications.
  Schema definitions are preserved in the header.

## Validating vendor tags

The `validator` package ships pre-built registries describing the legal
placement (and value pattern, where applicable) of common vendor tags. A
registry turns "unknown custom tag" warnings into informed validation.

```go
import "github.com/cacack/gedcom-go/v2/validator"

// Validate against a single vendor's known tags:
registry := validator.AncestryRegistry()
tv := validator.NewTagValidator(registry, true) // strict mode
issues := tv.Validate(doc)

// Or cover all known vendors at once:
registry = validator.DefaultVendorRegistry()

// Or pick the registry matching the detected vendor:
registry = validator.RegistryForVendor(doc.Vendor)
```

Available registries: `AncestryRegistry()`, `FamilySearchRegistry()`,
`RootsMagicRegistry()`, `DefaultVendorRegistry()` (all merged), and
`RegistryForVendor(vendor)`. Combine custom registries with
`MergeRegistries(...)`.

Registry contents:

| Vendor | Registered tags |
|--------|-----------------|
| Ancestry | `_APID`, `_TREE`, `_MILT`, `_DEST`, `_PRIM`, `_PHOTO` |
| FamilySearch | `_FSFTID`, `_FSORD`, `_FSTAG` |
| RootsMagic | `_PRIM`, `_SDATE`, `_TMPLT` |

## Lossless guarantee

Vendor extensions — whether promoted to a typed field or kept in raw `Tags` —
survive a full decode → encode round-trip unchanged. Lossless representation is
a non-negotiable project principle ([ETHOS.md](ETHOS.md)); if you find a vendor
tag that is dropped or altered, that is a bug worth reporting.
