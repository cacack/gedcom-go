# Features

This document provides a comprehensive list of all features implemented in gedcom-go.

For planned features, see [GitHub Issues](https://github.com/cacack/gedcom-go/issues).

## Convenience API

Single-import facade for common GEDCOM operations:

```go
import gedcomgo "github.com/cacack/gedcom-go/v3"

doc, err := gedcomgo.Decode(file)           // Parse GEDCOM
err = gedcomgo.Encode(writer, doc)          // Write GEDCOM
errs := gedcomgo.Validate(doc)              // Validation as []error (*Issue elements)
issues := gedcomgo.ValidateAll(doc)         // Comprehensive validation
converted, report, err := gedcomgo.Convert(doc, gedcomgo.Version70)  // Version conversion
```

| Function | Description |
|----------|-------------|
| `Decode(r)` | Parse GEDCOM file with default options |
| `DecodeWithDiagnostics(r)` | Parse with error collection for lenient mode |
| `Encode(w, doc)` | Write GEDCOM file with default options |
| `Validate(doc)` | `ValidateAll` as `[]error`; each element is a `*validator.Issue` |
| `ValidateAll(doc)` | Comprehensive validation with severity levels (returns `[]Issue`) |
| `Convert(doc, version)` | Convert between GEDCOM versions |

Re-exported types for single-import convenience:
- `Document`, `Individual`, `Family` - Core data types
- `Version`, `Version55`, `Version551`, `Version70` - Version constants
- `DecodeResult`, `Issue`, `ConversionReport` - Result types
- `DecodeOptions`, `EncodeOptions`, `ValidateOptions` - Options types

For advanced use cases (custom options, progress callbacks, streaming), import the underlying packages directly. See README.md for examples.

## Configurable Options

Each core operation exposes a dedicated options struct with safe defaults and an explicit `*WithOptions` entry point:

| Operation | Options type | Facade entry point | Common knobs |
|-----------|--------------|--------------------|--------------|
| Decode | `decoder.DecodeOptions` | `gedcomgo.DecodeWithOptions` | `Context`, `StrictMode`, `OnProgress`, `TotalSize` |
| Encode | `encoder.EncodeOptions` | `gedcomgo.EncodeWithOptions` | `LineEnding`, `MaxLineLength`, `DisableLineWrap`, `TargetVersion`, `DropUnknownTags`, `Context` |
| Validate | `validator.ValidateOptions` | `gedcomgo.ValidateAllWithOptions` | `Strictness`, `MaxErrors`, `SkipRules`, `DateLogic`, `Duplicates`, `TagRegistry`, `ValidateCustomTags`, `SkipEncodingValidation`, `SkipDuplicateDetection`, `Context` |

`gedcomgo.DefaultDecodeOptions()`, `DefaultEncodeOptions()`, and `DefaultValidateOptions()` return populated defaults you can tweak. `validator.ValidateOptions` is an alias for the original `validator.ValidatorConfig`; both names work interchangeably. The `[]error` validation path has its own configurable entry point, `gedcomgo.ValidateWithOptions(doc, opts)`, alongside `ValidateAllWithOptions`; both run the same checks and honour every `ValidateOptions` field, including `MaxErrors` and `SkipRules`.

## Lenient Parsing & Diagnostics

`DecodeWithDiagnostics` parses real-world GEDCOM exports that don't strictly conform to the spec, collecting issues as `Diagnostic`s rather than aborting on the first error. Recovered conditions include:

| Code | Meaning | Recovery |
|------|---------|----------|
| `EMPTY_LINE` | Blank line in the data | Skipped |
| `INVALID_ENCODING` | A byte that is not valid UTF-8 in input read as UTF-8 (declared `UTF-8` or `ASCII`, or no recognized `CHAR`), such as a stray Windows-1252 `é` | Byte replaced with U+FFFD and the rest of the line and file kept; one diagnostic per byte, with its line and column, up to 100 bytes, then one counting the rest |
| `INVALID_LEVEL` | Unparseable, negative, or over-99 level number | Line skipped |
| `INVALID_XREF` | Cross-reference identifier that is malformed (contains a space in a document whose header does not declare GEDCOM 5.5 or 5.5.1, or has no closing `@`) or that the grammar forbids where it appears (`0 @X1@ HEAD`, `0 @X1@ TRLR`) | Identifier recovered verbatim so the record and its subordinate lines survive; because nothing is invented, an unterminated `@I1` is stored as `@I1` and ordinary `@I1@` pointers elsewhere do not resolve to it. An XRef on `HEAD`/`TRLR`, well-formed or recovered (`0 @I1 HEAD`), makes the line an ordinary record (`Type` `"HEAD"`/`"TRLR"`, `Entity == nil`) instead of the document's header or trailer, so the real header is never overwritten. Reported but *not* recovered when a level-0 identifier has no usable record tag after it (`0 @I1@`, `0 @I 1@`, `0 @I1`): the line keeps its pre-existing parse, so the record survives, owning its subordinate lines, but with `Type` set to the identifier and an empty `XRef`, absent from `XRefMap` — see [compatibility.md](docs/governance/policies/compatibility.md#encoding-and-grammar-limitations). An identifier recovered on a level >= 1 line (`1 @I 1@ NOTE x`) is kept on the raw `gedcom.Tag.XRef` and re-encoded verbatim |
| `BAD_LEVEL_JUMP` | Indentation skips one or more levels (e.g., `1 BIRT` → `4 DATE`) | Level clamped to `prev + 1` so the subordinate attaches to its natural parent |
| `UNKNOWN_TAG` | Unrecognized tag in a record or the header, or a level-0 record type outside the standard set (`0 @X1@ FOO`, lower-case `0 @I1@ indi`); underscore-prefixed extension tags and records are not reported. Each tag name is reported line by line up to `MaxUnknownTagDiagnosticsPerTag` (10) times, then once more counting its remaining lines | Preserved in raw form (`Record.Tags`, `Header.Tags`); an unknown record keeps its `Type` as written, with no entity |
| `DUPLICATE_XREF` | A level-0 record redefines an XRef an earlier record defined (error) | Both records kept in `Records`; `XRefMap`, so every pointer, resolves to the first definition |
| `INVALID_VALUE` | Value doesn't match the expected format | Raw value preserved |
| `MISSING_REQUIRED` | Required subordinate tag is missing | Reported; decoding continues |
| `SKIPPED_RECORD` | A record's level-0 line was dropped, so its subordinate lines are skipped instead of attaching to the preceding record | Record dropped; decoding continues |
| `DIAGNOSTICS_TRUNCATED` | More diagnostics than `DecodeOptions.MaxDiagnostics` (default 100,000; `decoder.UnlimitedDiagnostics` removes the bound) | The first ones are kept and this one, last, counts the rest; the document is unaffected |

A diagnostic's `Message` and `Context`, and a `parser.ParseError`'s, echo at most `parser.MaxContextBytes` (256) bytes of the input, followed by `…` when clipped, so a malformed 1 MiB line does not yield a 1 MiB error string.

Strict mode (`DecodeOptions{StrictMode: true}`) disables recovery and returns the first syntax error, or the first invalid UTF-8 byte as a `*charset.InvalidUTF8Error` (matchable with `errors.Is(err, charset.ErrInvalidUTF8Sequence)`).

## Multi-Version Support

| Version | Decoding | Typed coverage | Notes |
|---------|----------|----------------|-------|
| GEDCOM 5.5 | Every readable line preserved | [884 of 998 structures](docs/reference/gedcom-5.5-coverage.md) (88.6%) | Legacy format |
| GEDCOM 5.5.1 | Every readable line preserved | [1,186 of 1,316 structures](docs/reference/gedcom-5.5-coverage.md) (90.1%) | Most common format |
| GEDCOM 7.0 | Every readable line preserved | [1,269 of 1,389 structures](docs/reference/gedcom-7-coverage.md) (91.4%) | Latest standard |

- Automatic version detection from header (GEDCOM 7 patch versions such as `VERS 7.0.14` detect as 7.0; a `0 @X1@ HEAD` record is not the header and never sets the version)
- Heuristic-based detection for malformed headers
- Version-aware validation rules

The table separates three claims that are easy to run together, because this
library is much stronger on the first than on the third.

**Decoding.** Every line the parser can read is kept, whether or not the decoder
understands it: unrecognized tags land in `Record.Tags` rather than being
dropped ([ADR 0003](docs/decisions/0003-lossless-dual-storage.md)). Lines the
parser *cannot* read are skipped and reported as diagnostics — see the lenient
parsing table above, which says which shapes those are.

**Typed coverage.** How much of a version reaches typed fields rather than raw
tags. All three versions are measured, not estimated. Every structure each
specification defines, in every context it defines it, is derived by decoding a
document built for it: 1,269 of 7.0's structures reach the typed model, 884 of
5.5's, and 1,186 of 5.5.1's.
The reports say which, and why the rest do not —
[gedcom-7-coverage.md](docs/reference/gedcom-7-coverage.md) and
[gedcom-5.5-coverage.md](docs/reference/gedcom-5.5-coverage.md).
A structure without typed access is read from `Record.Tags`, which is a
supported access path, not a workaround — see
[Reading Data the Typed Model Does Not Cover](docs/guides/decoding.md#reading-data-the-typed-model-does-not-cover).

Read each percentage against its own version, not against the others. The three
specifications define different structure sets — 7.0 defines the most, and much
of what it added is what this library has yet to type — so the shares rank the
work left per version, not the versions.

**Re-encoding.** Not byte-for-byte, and this is measured too. Of 162 corpus
fixtures, 0 do not survive decode and encode at all; of the 162 that do, 149
reproduce their header byte for byte and 136 reproduce their record body.
`byte_roundtrip_test.go` names the reason behind every exception. What remains
in the body count is not a defect but a limit of byte comparison itself: a
source that was not UTF-8 has been transcoded, so its bytes cannot match
whatever the encoder does, and eight fixtures with malformed lines are rewritten by lenient
recovery rather than preserved — a difference kept on the list so that the
decision to accept it stays explicit.

### Version-Aware Export

`Document.MinimumVersion()` reports the lowest GEDCOM version that can losslessly
represent the document — the canonical way to pick a target version for export.
`Document.RequiresGEDCOM7()` is the underlying boolean check.

```go
opts := &encoder.EncodeOptions{TargetVersion: doc.MinimumVersion()}
encoder.EncodeWithOptions(w, doc, opts)
```

`MinimumVersion` returns `Version551` by default and `Version70` when the document
uses any 7.0-only feature the library models: `SCHMA`, `SNOTE` records, `EXID`,
`CREA`, negative events (`NO`), event sort dates (`SDATE`), name transliterations
(`TRAN`), association `PHRASE` and the other typed `PHRASE` fields, media
`CROP`, media-file `TRAN`, and `SNOTE` references. INT (interpreted) dates and non-Gregorian calendars are **not** 7.0
triggers — both are valid in 5.5.1.

## Vendor Detection

Automatic detection of the originating software from `HEAD.SOUR`, then
`HEAD.SOUR.NAME`, then `HEAD.SOUR.CORP` (`gedcom.DetectVendorFromHeader`).
Recognized: Ancestry (including Family Tree Maker), FamilySearch, RootsMagic,
Legacy, Gramps, MyHeritage (including Family Tree Builder's empty-`SOUR`
exports, via `HEAD.SOUR._RTLSAVE`), PAF, Ancestris, Heredis, Family Historian,
Brother's Keeper, The Master Genealogist, Family Origins, EasyTree, GEDitCOM,
webtrees, Geneanet, My Roots, webtreeprint.com, Reunion and MacFamilyTree. The
strings each matches are in the
[vendor extensions guide](docs/guides/vendor-extensions.md#vendor-detection).

- Case-insensitive; spacing and punctuation ignored; short codes (`FTM`, `PAF`, `TMG`) match whole words only
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

Structured parsing for vendor-specific GEDCOM extensions. For the complete
reference — every supported tag, typed vs preserved-raw handling, validation,
and the lossless guarantee — see [docs/guides/vendor-extensions.md](docs/guides/vendor-extensions.md).

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

### `_UID` (MyHeritage, PAF, RootsMagic, Legacy, Family Historian)

`_UID` fills the same `UIDs` fields as GEDCOM 7.0's `UID`, on `Individual`,
`Family`, `Source`, `MediaObject`, `Event` and `Attribute`; the raw tag stays
in `Tags`. The encoder writes `UIDs` as `_UID` for 5.5/5.5.1 output and `UID`
for 7.0. See [vendor extensions](docs/guides/vendor-extensions.md#_uid--record-and-event-identifiers).

### `_FREL`/`_MREL` (Family Tree Maker, Ancestry, Legacy) and `_MARNM` (MyHeritage)

`_FREL` and `_MREL` under a family's `CHIL` fill `Family.ChildLinks` (`XRef`,
`FatherRelation`, `MotherRelation`; `Family.LinkForChild(xref)` looks one up),
leaving `Children` unchanged. `_MARNM` under `INDI.NAME` fills
`PersonalName.MarriedName`. The encoder writes both back as the vendor tags. On
an upgrade the converter adds `FAMC.PEDI` from an unambiguous non-natural
relationship, and on an upgrade to 7.0 a `TYPE MARRIED` name from `_MARNM`. See
[vendor extensions](docs/guides/vendor-extensions.md#_frel--_mrel--child-relationships).

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

Supported on: Document (as top-level records accessible via `SharedNotes()` and `GetSharedNote()`), MediaObject (as `SharedNoteXRefs`)

### Encoding Validation

GEDCOM 7.0 encoding compliance validation:

- **UTF-8 Enforcement**: Validates that GEDCOM 7.0 files use UTF-8 encoding exclusively. ANSEL and UTF-16 encodings are rejected for 7.0 files (they remain valid for 5.5/5.5.1).
- **Control Character Validation**: Detects banned C0 control characters (U+0000-U+001F) in 7.0 files, except for allowed whitespace: TAB (U+0009), LF (U+000A), and CR (U+000D).

```go
// Run encoding validation
validator := validator.New()
issues := validator.ValidateEncoding(doc)

// Or as part of full validation
allIssues := validator.ValidateAll(doc)
```

| Error Code | Severity | Description |
|------------|----------|-------------|
| INVALID_ENCODING_FOR_VERSION | Error | Non-UTF-8 encoding in GEDCOM 7.0 file |
| BANNED_CONTROL_CHARACTER | Error | Banned C0 control character found in 7.0 file |

### Negative Assertions (NO)

GEDCOM 7.0 negative assertions record that an event did NOT occur - explicit statements like "never married" or "no death record found." This is different from simply having no information about an event.

Negative assertions decode into `Individual.Events` and `Family.Events` alongside the events that happened, with `IsNegative` set. Two helpers on both types separate them, so a reader that treats events as facts cannot pick up a `NO` by accident:

```go
// Events that happened (NO assertions and nil entries left out)
for _, event := range individual.OccurredEvents() {
    fmt.Println(event.Type, event.Date)
}

// Negative assertions only
for _, event := range individual.NegativeAssertions() {
    fmt.Printf("NO %s", event.Type)  // "NO MARR"
    if event.Date != "" {
        fmt.Printf(" during %s", event.Date)
    }
}
```

The library's own readers skip negative assertions: `BirthEvent`, `DeathEvent`, `BirthDate`, `DeathDate`, `BirthPlace`, `DeathPlace`, `EventOf` and `Family.MarriageEvent`; the date-logic rules (`DEATH_BEFORE_BIRTH`, `MARRIAGE_BEFORE_BIRTH`, lifespan and parent-age checks) and the quality report's completeness counts; and duplicate detection's birth-date comparison. Converting to GEDCOM 5.5 or 5.5.1, which have no `NO` structure, drops every negative assertion (the `NO` line and its subordinates, and the negated event from the typed model) and reports it as data loss. Code that ranges over `Events` directly must check `IsNegative` or use the helpers.

| Field | Description |
|-------|-------------|
| IsNegative | True if this is a NO assertion (event did not occur) |
| Type | The event type that did not occur (MARR, DEAT, DIV, etc.) |
| Date | Optional date range when non-event was known |
| SourceCitations | Evidence supporting the negative assertion |
| NoteXRefs / InlineNotes | Notes explaining the negative assertion |

Supported on: Individual events (NO MARR, NO DEAT, NO NATU, NO EMIG, etc.) and Family events (NO DIV, NO ANUL)

### Phrases (PHRASE)

A `PHRASE` qualifying a typed value is decoded into a `<Value>Phrase` field
beside it, and written back beneath that value
([ADR 0010](docs/decisions/0010-phrase-typed-access.md)):

| PHRASE under | Field |
|--------------|-------|
| `NAME.TYPE` | `PersonalName.TypePhrase` |
| `FAMC.PEDI` | `FamilyLink.PedigreePhrase` |
| `ASSO` | `Association.Phrase` |
| `ASSO.ROLE` | `Association.RolePhrase` |
| `ADOP.FAMC.ADOP` | `Event.AdoptedByPhrase` |
| Event `AGE` | `Event.AgePhrase` |
| Event/attribute `SDATE` | `Event.SortDatePhrase`, `Attribute.SortDatePhrase` |
| `FAM.HUSB`, `FAM.WIFE` | `Family.HusbandPhrase`, `Family.WifePhrase` |
| `FILE.FORM.MEDI` | `MediaFile.MediaPhrase` |
| `CALN.MEDI` | `CallNumber.MediaPhrase` |
| `DATE` | `Date.Phrase` (an empty `DATE`, or one whose value does not parse, with a `PHRASE` is a phrase-only date). `Date.PhraseSubstructure` tells a `PHRASE` apart from a 5.5.1 `(text)` value; the encoder writes it beneath event, attribute and LDS ordinance dates for 7.0 output only |

`PHRASE` under a value with no typed field (`CHIL`, `ALIA`, `FAMC.STAT`, and
others listed in the ADR) stays in `Record.Tags`.

## Character Encoding

| Encoding | Status | Notes |
|----------|--------|-------|
| UTF-8 | Full | With BOM detection |
| ASCII | Full | Subset of UTF-8 |
| LATIN1 / ANSI / ISO-8859-1 / WINDOWS-1252 / CP1252 / WIN-1252 | Full | Decoded as Windows-1252 (curly quotes, dashes, `€`); the five bytes Windows-1252 leaves undefined keep their ISO-8859-1 code points |
| IBMPC / IBM PC / CP437 / IBM-437 | Full | IBM code page 437 (Brother's Keeper, The Master Genealogist) |
| CP850 / IBM-850 | Full | IBM code page 850 (DOS Latin-1) |
| MACINTOSH / MAC / MACROMAN | Full | Mac OS Roman |
| UTF-16 LE/BE | Full | With BOM detection; without a BOM, recognised by the first line's `0 ` (as GEDCOM 5.5 `CHAR UNICODE` files are written) |
| ANSEL | Full | With combining diacritical reordering; a mark with no base character stays on the line it came from |

A byte that is not valid UTF-8 in input read as UTF-8 fails a strict decode; lenient decoding replaces it with U+FFFD and reports `INVALID_ENCODING` (see [Lenient Parsing & Diagnostics](#lenient-parsing--diagnostics)). `charset.NewReaderWithOptions` with `ReaderOptions.OnInvalidUTF8` gives the same recovery outside the decoder.

An invalid byte is reported as a `*charset.InvalidUTF8Error` or `*charset.InvalidANSELError` (recover the byte and position with `errors.As`), which match `charset.ErrInvalidUTF8Sequence` and `charset.ErrInvalidANSELByte` with `errors.Is`. The former type names `ErrInvalidUTF8` and `ErrInvalidANSEL` remain as deprecated aliases.

`CHAR` values are matched without regard to case, and the whole value is read, so `IBM PC` is one value; `UTF8` is accepted for `UTF-8`. An unrecognised value (`IBM WINDOWS`, say) is read as UTF-8 and kept as written in `Header.Encoding`.

The charset declaration is `HEAD.CHAR` at level 1 only. Charset detection, `Header.Encoding` and the encoder's rewrite of `CHAR` to `UTF-8` all read that one line; a `CHAR` nested deeper in the header (`2 CHAR` under `SOUR`, say) is not a declaration, is kept in `Header.Tags` and written back verbatim.

## Document Operations

### Building Documents

`gedcom.NewDocument(version)` returns an empty document ready to fill: a
header with `Version` set, and `DeclaredVersion` too when the version is one
the library recognizes (so the document uses that version's pointer grammar),
an empty `Records` slice, an empty `XRefMap` and a `Trailer`.

```go
doc := gedcom.NewDocument(gedcom.Version70)
err := doc.AddRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
    Entity: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: "Ada /Lovelace/"}}}})
```

- `Document.AddRecord` appends a record and indexes it in `XRefMap`, so
  `GetIndividual`, `Parents` and the other lookups find it
- A record whose XRef is already indexed is rejected with an error wrapping
  `gedcom.ErrDuplicateXRef`, leaving the document unchanged; a nil record or
  document returns `gedcom.ErrNilRecord` / `gedcom.ErrNilDocument`
- A record with an empty XRef is appended but not indexed
- `Document.ReindexXRefs` rebuilds `XRefMap` from `Records` after direct
  edits; the first record wins when two share an XRef, as in the decoder

### Graph Traversal

Transitive walks across family links, returning XRef sets that compose into other operations (e.g., `Subset`):

```go
descendants := doc.Descendants("@I1@")  // BFS through FAMS → children
ancestors   := doc.Ancestors("@I1@")    // BFS through FAMC → parents
```

Convenience wrappers on `*Individual` that resolve XRefs to records:

```go
for _, child := range ind.Descendants(doc) { /* ... */ }
for _, parent := range ind.Ancestors(doc)   { /* ... */ }
```

- BFS ordering: closer generations precede farther ones
- Cycle-safe via visited-set termination
- Results exclude the seed individual
- Missing or non-individual XRefs return nil
- No application-level policy baked in (no "include spouses" knob,
  no generation cap) — callers compose those by unioning seed sets

### Subset Extraction

Build a self-contained sub-document from a seed set of XRefs. Performs
transitive reference closure (FAM, SOUR, NOTE, OBJE, REPO, SUBM, SNOTE)
and emits a valid document. The source is not mutated.

```go
sub, err := doc.Subset(doc.Descendants("@I1@"))
```

- Header carries Version, DeclaredVersion, Encoding, SourceSystem,
  SourceVersion, SourceName, SourceCorporation, Destination, Date, Time,
  Language, Copyright, PlaceForm, AncestryTreeID, and Schema from the source
- Spaced pointers (`@N 1@`) are followed in a document whose header
  declares GEDCOM 5.5 or 5.5.1 (`Header.DeclaredVersion`), as they are by
  `merge` and the validator's broken-pointer check
- `Header.Submitter` is always carried across: the submitter record it
  names (and anything that record references) is pulled into the
  closure, even for an empty seed set. Only a pointer that resolves to
  an actual `SUBM` record is followed — one naming a different record
  type, or nothing at all, is cleared rather than reported as an error.
  A value that is not pointer-shaped, such as 7.0's `@VOID@` null pointer,
  is carried through unchanged so the typed field and the raw tag agree
- **Privacy note:** the submitter record carries whatever `Address`,
  `Phone` and `Email` the source file recorded for whoever produced it.
  Every subset therefore includes those contact details, even when the
  seeds are unrelated. Clear or replace the submitter record before
  sharing an extract if that is not wanted
- Raw header tags with XRef fields are filtered: kept only when the
  pointer target is in the closure
- Strict mode: unknown seeds and dangling references during closure
  walking return errors wrapping `gedcom.ErrUnknownXRef`
- `@VOID@` is skipped as the null pointer (not treated as a missing ref)
  when `Header.Version` is 7.0, or when no version is declared and no
  record is named `@VOID@`; this is the rule `gedcom.Apply` uses
  (`Document.IsRecordPointer`); in a 5.5/5.5.1 document it is an ordinary
  identifier, followed like any other pointer and accepted as a seed
- XRefs are preserved exactly — callers needing fresh IDs apply an
  XRef remap as a separate step

### Deep Copy

Public `Clone()` methods on `Document`, `Header`, `Trailer`, `Record`,
`Tag`, `Individual`, `Family`, `Source`, `Repository`, `Note`,
`MediaObject`, `Submitter`, `Submission`, and `SharedNote`. All return nil for a nil
receiver and share no pointers with the original.

```go
copy := doc.Clone()  // pure function; original unchanged
```

`gedcom.CloneTags([]*Tag) []*Tag` is the slice complement to
`Tag.Clone()`.

### Merge Primitives

The `merge` package provides mechanical building blocks for combining
documents. It handles the bookkeeping (XRef remapping, header
compatibility, collision resolution) without imposing record-level
merge policy — that belongs in the consuming application.

```go
// Remap every XRef through a caller-supplied transform.
remapped, mapping, err := merge.RemapXRefs(doc, func(old string) string {
    return "@a_" + old[1:]  // prefix into a disjoint namespace
})

// Combine two documents with a chosen collision strategy.
combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
    CollisionStrategy: merge.PrefixDoc2,
    Prefix:            "b_",
})
```

| Function | Description |
|----------|-------------|
| `merge.RemapXRefs(doc, transform)` | Pure XRef rewrite preserving referential integrity; original document unchanged. Records whose XRef is not pointer-shaped (e.g. recovered from a malformed line) are left as-is |
| `merge.Combine(doc1, doc2, opts)` | Merge two documents with a configurable collision strategy (`ErrorOnCollision`, `PrefixDoc2`, `RenumberDoc2`) |

`Combine` returns a `CombineReport` describing the XRefs that were
remapped on doc2 and any header fields where doc1's value was kept
over a differing doc2 value. Both inputs are deep-copied; neither is
mutated. The typed views of header substructures (`SourceVersion`,
`SourceName`, `SourceCorporation`, `Destination`, `Time`, `PlaceForm`) follow
the raw structure that holds them, so a doc2 `SOUR`, `DEST`, `DATE` or `PLAC`
dropped in favour of doc1's is reported as a `Tags.<tag>` conflict.
Differing non-empty header versions or encodings are refused
with `ErrIncompatibleHeader`; set `CombineOptions.IgnoreEncodingMismatch`
to merge documents whose declared encodings differ (text is UTF-8 in
memory either way) — doc1's encoding is kept and the difference is
reported as a `HeaderConflict`. Argument errors wrap `merge.ErrNilDocument`
(a nil document) or `merge.ErrInvalidOptions` (an unknown strategy, a
`PrefixDoc2` with no `Prefix`, a nil transform), so `errors.Is` separates
them from the data errors `ErrIncompatibleHeader`, `ErrXRefCollision` and
`ErrInvalidRemap`.

## Record Types

### Header (HEAD)

- Version (`GEDC.VERS`), encoding (`CHAR`), transmission date and time
  (`DATE`, `DATE.TIME`), language, copyright, submitter pointer and SCHMA
- Source software (`SOUR`): system ID, `VERS` (`SourceVersion`), product
  `NAME` (`SourceName`) and `CORP` (`SourceCorporation`, with its address and
  `PHON`/`EMAIL`/`FAX`/`WWW` contact slices)
- Destination system (`DEST`) and default place hierarchy (`PLAC.FORM`,
  `PlaceForm`)
- Every typed field is a read view filled by the decoder: `Header.Tags` is
  what a decoded header is encoded from (see the `encoder` package docs)

### Individuals (INDI)

- Cross-reference ID (`@I1@`)
- Names with components (given, surname, prefix, suffix, nickname)
- Name types (birth, married, aka)
- Sex (M/F/X/U)
- Events (see Events section)
- Attributes (see Attributes section)
- Family links (FAMC, FAMS) with per-link notes; pedigree types on FAMC
- Associations (ASSO) with roles (`RELA`/`ROLE`) and the 5.5 record `TYPE` (`Association.RecordType`)
- LDS ordinances (BAPL, CONL, ENDL, SLGC), with the status date (`STAT.DATE`, `StatusDate`/`StatusTime`)
- Source citations
- Notes and multimedia references
- Change dates (CHAN)

### Families (FAM)

- Cross-reference ID (`@F1@`)
- Husband/Wife references
- Children references
- Family events (see Events section)
- Family attributes (`Family.Attributes`, the counterpart of
  `Individual.Attributes`) — see Attributes section
- LDS ordinances (SLGS)
- Source citations
- Notes

### Sources (SOUR)

- Cross-reference ID (`@S1@`)
- Title, author, publication info
- Every repository link (`RepositoryLinks`, one per `REPO`) — by XRef or inline
  by name — with its per-link notes (NOTE/SNOTE) and its call numbers
  (`CallNumbers`), each CALN keeping its own MEDI media type and 7.0 MEDI
  PHRASE
- Notes and multimedia

### Repositories (REPO)

- Cross-reference ID (`@R1@`)
- Name and address
- Contact details as ordered slices, every repeat kept (each is `{0:3}`):
  `Phone` (PHON), `Email` (EMAIL), `Fax` (FAX) and `Website` (WWW)
- Notes

### Submitters (SUBM)

- Cross-reference ID (`@U1@`)
- Name, address, language

### Submissions (SUBN, GEDCOM 5.5/5.5.1)

- Cross-reference ID, `SubmitterXRef` (SUBM)
- `FamilyFileName` (FAMF), `TempleCode` (TEMP), `AncestorGenerations` (ANCE),
  `DescendantGenerations` (DESC), `OrdinanceProcessFlag` (ORDI) and
  `AutomatedRecordID` (RIN), each stored as written
- Notes and `ChangeDate` (CHAN)

### Notes (NOTE)

- Cross-reference ID (`@N1@`)
- Text content with continuation
- Split note fields on note-bearing records (Individual, Family, Source,
  Repository, Submitter, Submission, MediaObject) and on the note-bearing substructures
  (PersonalName, Event, Attribute, SourceCitation, LDSOrdinance, ChangeDate, Association,
  SourceRepositoryLink, MediaLink, FamilyLink, PlaceDetail): `NoteXRefs` holds
  XRef pointers to shared NOTE/SNOTE records, `InlineNotes` holds note text
  written directly on the structure, with `CONT`/`CONC` folded.
- `AllNotes(doc)` helper returns inline note text plus the resolved text of
  any shared notes referenced by XRef; also available on `Event` and
  `Attribute`

### Multimedia (OBJE)

- Cross-reference ID (`@M1@`)
- File references and formats
- Titles

## Events

### Individual Events

| Tag | Event | Subordinates |
|-----|-------|--------------|
| BIRT | Birth | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE, FAMC |
| DEAT | Death | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE |
| BURI | Burial | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE |
| CREM | Cremation | DATE, PLAC, TYPE, CAUS, AGE, AGNC, ADDR, SOUR, NOTE |
| ADOP | Adoption | DATE, PLAC, FAMC with ADOP type (`Event.FamilyXRef`, `AdoptedBy`) |
| BAPM | Baptism | DATE, PLAC, TYPE, CAUS, AGE, AGNC |
| CHR | Christening | DATE, PLAC, TYPE, CAUS, AGE, AGNC, FAMC |
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

`EVEN` is the only event tag whose own payload is data rather than a
`[Y|<NULL>]` flag; that descriptor is decoded into `Event.Description` and
written back out on the event line.

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
| CENS | Family Census | DATE, PLAC, plus the full EVENT_DETAIL below |
| RESI | Family Residence | DATE, PLAC, ADDR, plus the full EVENT_DETAIL below |
| EVEN | Generic Event | DATE, PLAC, TYPE |

### Event Detail

Beyond the subordinates named per tag above, every event — individual or
family — decodes the shared `EVENT_DETAIL` structure into typed fields:

| Tag | Field |
|-----|-------|
| DATE | `Event.Date`, `Event.ParsedDate` |
| PLAC | `Event.PlaceDetail` (FORM, MAP/LATI/LONG), read via `Event.PlaceName()`, written via `Event.SetPlaceName()` |
| TYPE | `Event.EventTypeDetail` |
| CAUS | `Event.Cause` |
| AGE | `Event.Age` |
| AGNC | `Event.Agency` |
| RELI | `Event.ReligiousAffiliation` |
| ADDR | `Event.Address` |
| PHON, EMAIL, FAX, WWW | `Event.Phone`, `Email`, `Fax`, `Website` |
| RESN | `Event.Restriction` |
| UID | `Event.UIDs` (repeatable, in file order) |
| SDATE | `Event.SortDate` |
| ASSO | `Event.Associations` |
| NOTE, SNOTE | `Event.NoteXRefs`, `Event.InlineNotes`, `Event.AllNotes(doc)` |
| SOUR | `Event.SourceCitations` |
| OBJE | `Event.Media` |

`gedcom.Attribute` names the same fields, so an attribute reads the same way as
an event.

`HUSB` and `WIFE` (spouse ages, `FAMILY_EVENT_DETAIL`) are accepted under a
family's events and attributes and kept raw in `Record.Tags`; under an
individual's events or attributes, where the grammar has no such position,
they are reported as `UNKNOWN_TAG`.

## Attributes

| Tag | Attribute | Constant | Notes |
|-----|-----------|----------|-------|
| OCCU | Occupation | `AttributeOccupation` | With DATE for periods |
| RESI | Residence | `EventResidence` | Modeled as an Event (lands in `Events`, not `Attributes`); with ADDR structure |
| EDUC | Education | `AttributeEducation` | |
| RELI | Religion | `AttributeReligion` | |
| TITL | Title | `AttributeTitle` | Nobility, professional |
| NATI | Nationality | `AttributeNationality` | |
| CAST | Caste | `AttributeCaste` | |
| DSCR | Physical Description | `AttributePhysicalDescription` | |
| IDNO | ID Number | `AttributeIDNumber` | |
| SSN | Social Security Number | `AttributeSSN` | |
| NCHI | Number of Children | `AttributeNumberOfChildren` | |
| NMR | Number of Marriages | `AttributeNumberOfMarriages` | |
| PROP | Property | `AttributeProperty` | |
| FACT | Generic Fact | `AttributeFact` | INDI and FAM; its meaning is the TYPE (5.5.1+) |

Each attribute's subordinate `TYPE` lands in `Attribute.TypeDetail`, alongside
the full `EVENT_DETAIL` set listed under Events — `gedcom.Attribute` carries the
same fields as `gedcom.Event`.

At the `FAM` level, `NCHI` and `FACT` decode into `Family.Attributes` and
`CENS` and `RESI` into `Family.Events`, each with its `EVENT_DETAIL`.
`Family.Attributes` is the single store for `NCHI`; read it through
`Family.NumberOfChildren()` and write it through `SetNumberOfChildren`.

## Source Citations

- Embedded citations (within records)
- Referenced citations (via @SOUR@ xref)
- PAGE - Specific location in source
- QUAY - Quality/certainty assessment (0-3)
- DATA - Citation data with DATE and every repeated TEXT
  (`SourceCitationData.Text []*SourceText`, in document order), each with
  CONT/CONC folded into `Value` and GEDCOM 7.0 `MIME`/`LANG` typed
- Notes on citations

## Place Structure

- Place name with hierarchy (comma-separated)
- MAP coordinates (LATI, LONG)
- Place notes

### Place Name Accessors

`Event.PlaceDetail` and `Attribute.PlaceDetail` are pointers, so reading a name
off them needs a nil check and writing one needs an allocation. These accessors
do both for you:

| Accessor | Behaviour |
|----------|-----------|
| `Event.PlaceName()` | Place name, or `""` when no place is recorded |
| `Attribute.PlaceName()` | Place name, or `""` when no place is recorded |
| `Event.SetPlaceName(name)` | Sets the name, allocating `PlaceDetail` if absent |
| `Attribute.SetPlaceName(name)` | Sets the name, allocating `PlaceDetail` if absent |

```go
// Nil-safe on the receiver and on PlaceDetail.
fmt.Printf("Born at %s\n", person.BirthEvent().PlaceName())

// Allocates PlaceDetail on a freshly built event; preserves Form and
// Coordinates on one that already has them.
ev := &gedcom.Event{Type: gedcom.EventBirth}
ev.SetPlaceName("Boston, MA")
```

On events and attributes, `PlaceDetail` is the sole place carrier: both readers
return `PlaceDetail.Name`, and the encoder writes a `PLAC` line whenever
`PlaceDetail` is non-nil -- an empty `Name` still writes a valueless `PLAC`,
which is what a bare `2 PLAC` line decodes to. The corollary is the reason
`SetPlaceName` exists: a nil `PlaceDetail` means *no place*, so a writer that
leaves it nil emits no `PLAC` line at all, in memory-visible silence. Setting an
empty name on an event that has no carrier is therefore a no-op; to clear a place
that is already recorded, assign `PlaceDetail = nil`.

`LDSOrdinance.Place` is a separate scalar with no `PlaceDetail` twin and is
unaffected by any of the above. That asymmetry is deliberate; see
[Accepted Asymmetries](docs/governance/policies/api-stability.md#accepted-asymmetries)
for what it costs and why closing it later would be additive.

### Coordinate Conversion

Convert GEDCOM-format coordinates (direction-prefixed, e.g. `N42.3601`) to signed decimal degrees. A coordinate is a direction letter (`N`/`S` for latitude, `E`/`W` for longitude, case-insensitive) followed by an unsigned decimal number — digits with at most one decimal point, so `N.5` and `N5.` both parse. Surrounding whitespace is ignored.

```go
// place.Coordinates is *Coordinates and is nil when the place has no MAP tag.
// AsDecimal is nil-safe (a nil receiver yields 0, 0, nil), but guarding makes
// the intent explicit:
if !place.Coordinates.IsEmpty() {
    // Whole pair, with axis + range validation
    // (lat must use N/S ∈ [-90,90]; long must use E/W ∈ [-180,180]).
    lat, long, err := place.Coordinates.AsDecimal()
}

// Single component, axis known: direction + range enforced.
lat, err := gedcom.ParseLatitude("N42.3601")    // 42.3601
_, err = gedcom.ParseLatitude("E42.3601")       // error: latitude must use N/S
long, err := gedcom.ParseLongitude("W71.0589")  // -71.0589

// Single component, axis unknown: pure format conversion, no range/axis check.
lat, err = gedcom.ParseCoordinate("N42.3601")   // 42.3601
_, err = gedcom.ParseCoordinate("N-42.3601")    // error: value must be unsigned
```

Every coordinate error wraps `gedcom.ErrEmptyCoordinate` (an empty or blank component) or `gedcom.ErrInvalidCoordinate` (anything else), so callers branch with `errors.Is` rather than on the message.

`AsDecimal` returns `(0, 0, nil)` when the pair is empty (nil receiver or both components blank) and an error when only one is present. Valid coordinates at the origin (`N0`/`E0`) also return `(0, 0, nil)` — use `IsEmpty` to distinguish the absent case.

The sign is carried by the direction letter, so an explicitly signed value such as `N-42.3601` is rejected rather than silently inverted. Only plain decimals parse: numeric spellings that `strconv.ParseFloat` would otherwise accept — `NaN`, infinities (`inf`, `infinity`), hexadecimal float literals (`0x1p3`), exponent notation (`1e2`) and Go underscore separators (`1_000`) — are all errors. A `nil` error from `AsDecimal` therefore guarantees both values are finite and within `[-90, 90]` and `[-180, 180]`.

`ParseCoordinate` accepts any direction letter by design: a lone component carries no information about which axis it belongs to, so it cannot tell a latitude from a longitude. When the axis is known, prefer `ParseLatitude` or `ParseLongitude` — they add exactly the direction and range check that `AsDecimal` applies to a complete pair.

Decoding reports bad coordinates rather than deferring them. Wherever the decoder builds a typed `PlaceDetail` — event and attribute `PLAC` structures — a `MAP` whose `LATI` or `LONG` fails any of the three checks produces an `INVALID_VALUE` warning naming the offending line:

| Check | Example that warns |
|-------|--------------------|
| Value syntax | `4 LATI Nnan`, `4 LONG E0x1p3` |
| Axis direction | `4 LATI E42.3601` (E/W on a latitude) |
| Range | `4 LATI N95` (beyond `[-90, 90]`) |

The raw text is preserved on the record regardless (ADR 0003), so the warning is the signal, not a dropped value. Decoding never fails on one — these are warnings, and `HasErrors` counts only errors.

Before v3.0.0 `LATI` and `LONG` received no decode-time validation at all, so a document with out-of-range or axis-swapped coordinates produces warnings it did not produce before. Callers asserting on exact diagnostic counts should expect this.

`LDSOrdinance.Place` is a scalar with no `PlaceDetail` twin, so an ordinance `PLAC` carries no typed `MAP` and is not checked.

## Address Structure

- ADDR payload - The address text on the ADDR line, with its CONT/CONC lines (`Full`)
- ADR1, ADR2, ADR3 - Address lines (`Line1`-`Line3`)
- CITY - City
- STAE - State/Province
- POST - Postal code
- CTRY - Country
`gedcom.Address` holds the payload and these seven `{0:1}` subordinates, and is
comparable with `==`.

The payload and ADR1-ADR3 are independent fields: `Full` is filled only from
the ADDR line and its continuations, `Line1`-`Line3` only from ADR1-ADR3, and
the entity writer writes each tag from its own field. An exporter that writes
both (RootsMagic, GEDitCOM) round-trips both. GEDCOM 7.0 deprecates ADR1-ADR3,
so `Full` is the address text to read. The grammar gives ADR1-ADR3 no CONT, so a
line break in a hand-built `Line1`-`Line3` is written as a space.

The contact tags PHON, EMAIL, FAX and WWW are siblings of ADDR in the
grammar, not part of it, and each is `{0:3}`. They are typed on the owning
structure as `[]string` fields named `Phone`, `Email`, `Fax` and `Website`:

- `Event` and `Attribute` — all four
- `Repository` — all four
- `Submitter` — `Phone` and `Email` (FAX and WWW are kept in `Record.Tags`)

## Name Structure

- Full name with surname delimiters (`/surname/`)
- GIVN - Given name
- SURN - Surname
- NPFX - Name prefix (Dr., Rev.)
- NSFX - Name suffix (Jr., III)
- SPFX - Surname prefix (de, van, von)
- NICK - Nickname
- TYPE - Name type (birth, married, aka)
- SOUR - Source citations on the name (`PersonalName.SourceCitations`)
- NOTE, SNOTE - Notes on the name (`PersonalName.NoteXRefs`, `PersonalName.InlineNotes`)

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

### Phonetic and Romanized Variations (FONE, ROMN)

GEDCOM 5.5.1's `FONE` and `ROMN` under `NAME` decode into
`PersonalName.Phonetic` and `PersonalName.Romanized` (`[]*gedcom.NameVariation`).
They are kept apart from `Transliterations` because they name their scheme with
a required `TYPE` (e.g. `kana`, `romaji`) rather than 7.0's `LANG`:

```go
for _, v := range name.Romanized {
    fmt.Println(v.Value)   // "Yamada /Taro/"
    fmt.Println(v.Type)    // "romaji"
    fmt.Println(v.Surname) // "Yamada"
}
```

`NameVariation` carries `Value`, `Type` and the six name pieces (`Given`,
`Surname`, `Prefix`, `Suffix`, `Nickname`, `SurnamePrefix`). `NOTE` and `SOUR`
beneath a variation are kept in `Record.Tags` only. The encoder writes the
typed variations only for 5.5.1 output, and `converter.Convert` to 7.0 or 5.5
removes them and reports the loss.

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
- PHRASE - Human-readable description (GEDCOM 7.0); `RolePhrase` holds the
  PHRASE under ROLE
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
| PHRASE subordinate | `3 PHRASE Afternoon` | GEDCOM 7.0 human-readable description; an empty or unparseable `DATE` with a `PHRASE` gives a phrase-only date (`IsPhrase`, `Original` the raw value) |

### Validation

```go
// Validate complete dates using stdlib
err := date.Validate()
// Returns nil for valid dates
// Returns clear error for invalid: "invalid date: February has 28 days in 2023, got day 30"
```

- Uses `time.Date()` normalization (no reimplemented calendar math)
- Skips validation for partial dates (lossless representation)
- Julian, Hebrew and French Republican dates are checked against their own calendar (`@#DJULIAN@ 30 FEB 1700`, `@#DHEBREW@ 1 ADS 5785` in a common year, `@#DFRENCH R@ 6 COMP 5` are invalid), with the bound `Bounds` uses; `ToGregorian` returns an error for them instead of rolling them over
- Detects invalid day/month combinations (Feb 30, Jun 31)
- Handles leap years correctly (Feb 29 2000 valid, 1900 invalid)
- Checks both endpoints of ranges and periods (`BET 1900 AND 31 FEB 1901` is invalid)
- Rejects a range or period that ends before it starts (`BET 1860 AND 1850`, `FROM 1860 TO 1850`); endpoints of differing precision that overlap (`BET JAN 1850 AND 1850`) are valid
- Rejects a year more than 1,000,000 either side of the epoch, so arithmetic on a hand-built `Date` cannot overflow
- BCE dates use the proleptic Gregorian calendar with astronomical years (1 BCE = year 0, so Feb 29 1 BCE is valid and Feb 29 4 BCE is not)

`ParseDate` itself rejects a day or year written as `0` (`0 JAN 1850`, `1 JAN 0`, `0 BC`) — no calendar has one, and `0` is the "absent" value — and a signed day or year (`-1850`, `+1850`, `-5 JAN 1850`); write a B.C. year with a `BC` suffix. The decoder reports such a value as `INVALID_VALUE` and keeps it in the raw tags; the validator's `INVALID_DATE` check does not see it.

Every `ParseDate` error wraps `gedcom.ErrEmptyDate` (empty or blank input) or `gedcom.ErrInvalidDate` (anything else), so `errors.Is` tells an absent date from a malformed one; the message still says what is wrong.

### Calendar Systems

Full parsing support for historical calendars used in genealogical records:

| Calendar | 5.5.x Escape | 7.0 Keyword | Month Codes |
|----------|--------------|-------------|-------------|
| Gregorian | `@#DGREGORIAN@` (default) | `GREGORIAN` (default) | JAN, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC |
| Julian | `@#DJULIAN@` | `JULIAN` | JAN, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC |
| Hebrew | `@#DHEBREW@` | `HEBREW` | TSH, CSH, KSL, TVT, SHV, ADR, ADS, NSN, IYR, SVN, TMZ, AAV, ELL |
| French Republican | `@#DFRENCH R@` | `FRENCH_R` | VEND, BRUM, FRIM, NIVO, PLUV, VENT, GERM, FLOR, PRAI, MESS, THER, FRUC, COMP |

`ParseDate` accepts either form regardless of file version, anywhere an escape is accepted. The 7.0 keywords are case-sensitive, as the 7.0 grammar specifies. Extension calendars (`_XXX`) are not recognized: such a date fails to parse, keeps its raw value, and is reported as `INVALID_VALUE`. The converter rewrites the four standard calendars between the two forms when converting to or from 7.0 (see [Version Conversion](#version-conversion)).

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

// GEDCOM 7 writes the calendar as a bare keyword
date, _ := gedcom.ParseDate("JULIAN 1 JAN 1750")
date.Calendar  // CalendarJulian

// The escape is part of <date>, so it follows the modifier keyword
date, _ := gedcom.ParseDate("ABT @#DJULIAN@ MAR 1066")
date.Calendar  // CalendarJulian
date.Modifier  // ModifierAbout

// Each <date> in a range or period carries its own escape
date, _ := gedcom.ParseDate("BET 1700 AND @#DJULIAN@ 1750")
date.Calendar          // CalendarGregorian (no escape of its own)
date.EndDate.Calendar  // CalendarJulian
```

### Features

- Case-insensitive month parsing (`Jan`, `JAN`, `jan`) for all calendars
- Whitespace tolerance (leading, trailing, multiple spaces)
- Original string preserved for round-trip fidelity
- B.C. date comparison (100 BC > 200 BC chronologically)
- Calendar escapes compose with every modifier (`ABT`, `CAL`, `EST`, `BEF`,
  `AFT`, `BET`/`AND`, `FROM`/`TO`, `INT`), per date; a date with no escape
  inherits the preceding date's calendar, defaulting to Gregorian

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
date.IsBC           // true for B.C. dates
date.DualYear       // second year from "1750/51" format
date.Phrase         // text from "(unknown)" format
date.IsPhrase       // true for date phrases
date.EndDate        // *Date: end of a range (BET...AND) or period (FROM...TO)
date.IsInterpreted  // true for INT dates
date.InterpretedFrom // original phrase captured from an INT date

// Validate complete dates
err := date.Validate()  // nil if valid

// Compare dates for sorting (dual dates read in New Style: 11 FEB 1731/32 == 11 FEB 1732)
result := date1.Compare(date2)  // -1, 0, or 1
isAfter := date1.IsAfter(date2)
isBefore := date1.IsBefore(date2)
isEqual := date1.IsEqual(date2)

// Earliest and latest day a date can denote, as Julian Day Numbers: a partial
// date spans its period, BEF/AFT/FROM/TO are open-ended (math.MinInt/MaxInt),
// BET/FROM...TO span both endpoints, and dual dates read in New Style
earliest, latest, ok := date.Bounds()
// a is provably before b when aLatest < bEarliest

// Calculate years between dates
years, exact, err := gedcom.YearsBetween(birthDate, deathDate)
// exact is true if both dates are complete (day/month/year); dual dates count in New Style;
// dates in different calendars are converted to Gregorian first (Hebrew 1 TSH 5660 to 1 JAN 1950 is 50);
// a year-only date converts from its first day, so Hebrew 5660 counts as Gregorian 1899

// Convert to time.Time (complete, valid Gregorian dates only; 31 FEB errors; BCE uses astronomical years, 44 BCE -> -43)
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

// Ranges and periods convert as a unit, endpoint included
r, _ := gedcom.ParseDate("@#DJULIAN@ BET 1 JAN 1700 AND 1 JAN 1750")
g, err := r.ToGregorian()
// g.Modifier          // ModifierBetween (preserved)
// g.Day               // 11 (Julian 1 JAN 1700)
// g.EndDate.Day       // 12 (Julian 1 JAN 1750)
// g.EndDate.Calendar  // CalendarGregorian
```

Supports conversion from Julian, Hebrew, and French Republican calendars to
Gregorian. Only the calendar-dependent fields change: the modifier, phrase and
interpreted-date fields carry over unchanged. A dual date converts from its New
Style year and loses its dual year (`@#DJULIAN@ 11 FEB 1731/32` becomes
`22 FEB 1732`), so it orders as before and passes `Validate`. When the New
Style year is unknown (`1731/32`, `MAR 1731/32`, an invalid suffix such as
`1750/53`, or a non-Julian dual date such as `@#DHEBREW@ 1 TSH 5660/61`) it
converts from `Year` and still loses its dual year,
so validate the source to catch an invalid one. An endpoint too
incomplete to convert (year 0) fails the whole conversion rather than dropping
silently.

## Metadata

- REFN - Reference numbers with their `TYPE` subordinate, as `[]gedcom.RefNumber` on
  `Individual`, `Family`, `Source` and `MediaObject` (`RefNumbers`); every occurrence is
  kept in file order. `REFN` on `Repository`, `Note`, `SharedNote` and `Submitter`
  records is preserved in raw tags only
- UID - Unique identifiers, as `UIDs []string` on `Individual`, `Family`, `Source`,
  `MediaObject`, `Event` and `Attribute`; every occurrence is kept in file order
- CHAN - Change date with DATE and TIME
- CREA - Creation date (GEDCOM 7.0)

## Validation

### Structural Validation
- Valid line format (level, tag, value, xref)
- Proper hierarchy (levels increment by 1)
- Required tags present (HEAD, TRLR)
- Valid cross-references: every pointer in a record's raw tags must name an existing record (`BROKEN_XREF`, with the pointing line; the typed `ORPHANED_*` codes cover FAMC/FAMS/SOUR/HUSB/WIFE/CHIL). `@VOID@` is exempt as the null pointer when `Header.Version` is 7.0, or when no version is declared and no record is named `@VOID@`; in a document declaring 5.5/5.5.1 a dangling `@VOID@` is reported
- XRefs defined by more than one record (`DUPLICATE_XREF`, error, on each record after the first, with its line); also reported by `StreamingValidator`
- Data-quality heuristics, not spec rules (every GEDCOM version allows both records): individuals with no NAME (`MISSING_NAME`, warning) and families with no HUSB/WIFE/CHIL (`EMPTY_FAMILY`, warning), each with the record's line. `MISSING_REQUIRED_FIELD` is deprecated and no longer reported

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
| Death before birth | Error / Warning | Death date precedes birth date |
| Child before parent | Error / Warning | Child born before parent |
| Marriage before birth | Error / Warning | Marriage date before spouse's birth |
| Impossible age | Warning | Age exceeds configurable maximum (default: 120) |
| Unreasonable parent age | Warning | Parent age at child's birth outside normal range |

```go
v := validator.New()
issues := v.ValidateDateLogic(doc)
for _, issue := range issues {
    fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
}
```

The three ordering checks are an Error only when `Date.Bounds` proves the order on every
reading of both dates. When the dates are too imprecise to prove it — death `1503` against
birth `2 FEB 1503`, or `AFT 1078` against `ABT 1090` — the same code is a Warning that
says "possibly before". An approximate date (`ABT`, `CAL`, `EST`) on either side caps
the check at Warning, even when the periods the dates name do not overlap: death
`ABT 1849` against birth `ABT 1850` is a Warning, while `1849` against `1850` is an
Error. An interpreted date (`INT`) names an exact date and can still prove the order.

**Date Validity:**

`ValidateAll` reports every date that does not exist on its calendar — `31 FEB 1900`,
`29 FEB 1900`, or the end of `BET 1900 AND 31 FEB 1901` — as `INVALID_DATE` (warning),
using `gedcom.Date.Validate`. Every `DATE` is checked: events, attributes, LDS
ordinances, citations, `CHAN` and `HEAD.DATE`. Each issue carries the DATE line and,
in `Details`, the tag holding the date (`tag`, e.g. `BIRT`), its path below the record
(`path`, e.g. `BIRT.SOUR.DATA.DATE`) and the raw `value`. Partial dates (`ABT 1900`,
`FEB 1900`) are not checked for their day. Julian, Hebrew and French Republican dates are
checked on their own calendar, so `@#DJULIAN@ 30 FEB 1700` and `@#DHEBREW@ 30 ADR 5785` are
reported; a dual-dated Julian date is checked in its New Style year. A dual-dated (Old Style)
Gregorian value such as `29 FEB 1723/24` is checked only against the longest its month can be,
so its leap day passes but `30 FEB 1723/24` is reported. A dual year that is not the
year following the first, such as `1750/53` or `1850/1860`, is reported too, even on a
partial date. A range or period that ends before it starts, such as
`BET 1860 AND 1850`, is reported too. A one- or two-digit dual year resolves forward, so
`1699/00` is 1699 and 1700. A record built in code, with no raw tags, is checked
through its typed events, attributes and LDS ordinances; those issues carry line 0 and
a `path` of just `<TYPE>.DATE`. `ValidateAll` and `QualityReport` run this check;
`ValidateDateLogic` and the streaming validator do not.

| Error Code | Severity | Description |
|------------|----------|-------------|
| INVALID_DATE | Warning | Date (or range/period end date) names a day its month does not have, or a dual year that is not the following year, or a range or period ends before it starts |

**Orphaned Reference Detection:**

Typed detection for all GEDCOM reference types, each issue carrying the line of the pointing tag when the record was decoded:

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

**Note Pointer Validation:**

`MediaObject.NoteXRefs` (NOTE pointers) and `.SharedNoteXRefs` (SNOTE pointers)
partition a record's note pointers and never overlap. A hand-built or migrated
media object can still list the same pointer in both, which duplicates the note
in `AllNotes` and on typed-path re-encode.

Only that overlap is reported, and only for media objects. A pointer repeated
*within* one slice is not a violation — the file named the note twice, and
reproducing it is lossless representation working as intended. No other entity
splits `NOTE` from `SNOTE`, so no other entity carries this invariant.

| Error Code | Severity | Description |
|------------|----------|-------------|
| OVERLAPPING_NOTE_POINTERS | Warning | Media object lists the same note pointer in both NoteXRefs and SharedNoteXRefs |

```go
issues := v.ValidateNotePointers(doc)
```

**Nil Element Detection:**

A document built in code can hold a nil where an element belongs: a nil record in
`Document.Records`, a nil element in any slice of a record, its entity or the header
(`Tags`, `Names`, `Events`, `SourceCitations`, at any depth), or a `Record.Entity`
holding a typed nil pointer. The encoder and every other check skip such a nil
silently, so `ValidateAll` reports each one. `Details["field"]` is the path to it —
`Records[7]`, `Tags[3]`, `Entity`, `Entity.Events[2].SourceCitations[1]`,
`Header.Tags[0]` — relative to the record named by `RecordXRef`, or prefixed with
`Records[N]` for a record with no XRef or one whose XRef another record shares. An unset optional field, a nil `Header` and a
record with no entity mean "absent" and are not reported. A decoded document never
holds a nil, so this rule only ever fires on hand-built input; skip it with
`ValidateOptions.SkipRules`.

| Error Code | Severity | Description |
|------------|----------|-------------|
| NIL_ELEMENT | Error | Nil record, nil slice element or typed-nil entity in a hand-built document |

**Typed Edits the Encoder Would Not Write:**

`Record.Tags` and `Header.Tags` are authoritative on encode, so a typed field
edited on a decoded document is absent from the output until
`encoder.SyncTags` carries it into the tags. `ValidateAll` reports the header
and each record whose typed fields would encode differently from its raw tags
— exactly those `SyncTags` would change — with `RecordXRef` and `LineNumber`
of the record, and `Details["record"]` holding the record type or `HEAD`. The
check reads the raw tags back through the decoder, about doubling
`ValidateAll`'s time, so it is opt-in: `ValidateOptions.CheckEntityTags` turns
it on for `ValidateAll` and `QualityReport` (which lists the findings under
`ConformanceIssues`), and `validator.WithEntityTagsCheck()` for a
`QualityAnalyzer`. The check has no direction: `Tags` edited without the
typed fields (by hand) are reported too.

| Error Code | Severity | Description |
|------------|----------|-------------|
| ENTITY_TAGS_DIVERGED | Warning | Typed fields and raw Tags of a decoded record or header disagree; `encoder.SyncTags` rewrites Tags from the typed fields |
| ENTITY_TAGS_DIVERGED | Info | `encoder.SyncTags` fails on the document, so the comparison was not made; `Details["error"]` holds the error |

**Duplicate Detection:**

Configurable matching based on name similarity and date proximity:

```go
config := &validator.DuplicateConfig{
    MinNameSimilarity: 0.8,
    MaxBirthYearDiff:  2,
    MinConfidence:     0.7,
    MaxGroupSize:      1000, // per-surname-group ceiling
    MaxNameRunes:      64,   // given names compared on at most this many runes
}
v := validator.NewWithConfig(&validator.ValidatorConfig{Duplicates: config})
pairs := v.FindPotentialDuplicates(doc)
for _, pair := range pairs {
    fmt.Printf("Potential duplicate: %s and %s (%.0f%% confidence)\n",
        pair.Individual1.XRef, pair.Individual2.XRef, pair.Confidence*100)
}
```

Every field's zero value is its default, so a partial literal such as `&validator.DuplicateConfig{MinConfidence: 0.9}` keeps the other thresholds and name normalization in force. `MinNameSimilarity`, `MinConfidence` and `MaxBirthYearDiff` treat `0` as "use the default" (0.8, 0.7 and 2) and any negative value — spelled `validator.ZeroThreshold` — as an explicit zero. Name normalization (trim, lowercase, accent folding) is on unless `DisableNameNormalization` is set. Given-name similarity is the Levenshtein distance divided by the longer name's length, both counted in characters (runes) rather than bytes, so names in non-Latin scripts score on the same scale as ASCII ones. `DefaultDuplicateConfig()` returns a `*DuplicateConfig` with every default spelled out. Birth years are compared as astronomical years, so 44 BCE and 44 CE are 87 years apart and 1 BCE and 1 CE are one year apart; a birth date with no real year (a phrase, year 0, `0 BC`) earns no birth-year credit. Birth dates in different calendars are each placed on their first day (a dual date read in New Style, a modifier or range end ignored, a day or month the calendar does not have dropped) and compared as Gregorian years, so Hebrew 5660 and Gregorian 1899 are the same year while Hebrew 1900 and Gregorian 1900 are not; two dates in the same calendar compare `Year` as written, so a dual date counts from its Old Style year. Contradictions veto a pair outright rather than merely withholding credit: when both birth dates are exact (no `ABT`, `BEF`, `AFT`, `BET`, `FROM` or other modifier) and their years differ by more than `MaxBirthYearDiff`, or both sexes are known (`M`, `F` or `X`, compared case-insensitively) and differ, the pair is never reported, so a namesake father and son born decades apart are not flagged. An approximate or range birth date outside the window only withholds credit, since its first year need not contradict the other date. A missing birth year, or a missing or `U` sex, neither vetoes nor earns credit.

Comparison is quadratic within a surname group, and the input controls the surnames, so `MaxGroupSize` bounds it: any normalized-surname group larger than the cap is skipped whole and reported as one `DUPLICATE_DETECTION_LIMITED` warning (available as `DuplicateReport.LimitIssues` from `FindDuplicatesReport`, and included in `ValidateAll` and `QualityReport` output). Each pair's cost is quadratic in the given-name length too, so `MaxNameRunes` bounds that: a longer given name is compared on its first 64 runes and the individuals so compared are counted on the same warning. Both fields are tri-state and **zero is not unlimited** — `0` selects the default (`validator.DefaultMaxGroupSize` 1000, `validator.DefaultMaxNameRunes` 64), a negative value (`UnlimitedGroupSize`, `UnlimitedNameRunes`) disables the cap, and a positive value is used as given, so a partial config literal stays bounded. `ValidateOptions.SkipDuplicateDetection` drops duplicate detection from the `ValidateAll` sweep; `QualityReport`, `FindPotentialDuplicates` and `FindPotentialDuplicatesReport` still run it, bounded by `MaxGroupSize`. See [performance](docs/guides/performance.md#bounding-untrusted-input) for the worst-case bound and [ADR 0009](docs/decisions/0009-bounded-duplicate-detection.md) for the rationale.

Two notes for callers upgrading. `DUPLICATE_DETECTION_LIMITED` is a `Warning`, so unlike the `Info`-level `POTENTIAL_DUPLICATE` it appears at the default `StrictnessNormal` — a document with an oversized surname group gains a warning where duplicate detection previously contributed nothing at that strictness. Suppress it like any other code via `ValidateOptions.SkipRules`; a rule skipping `POTENTIAL_DUPLICATE` does **not** cover it. It also lands in `QualityReport.DuplicateIssues` (JSON `duplicate_issues`) alongside the pairs, so count pairs by filtering on `POTENTIAL_DUPLICATE` rather than taking the length of that slice.

**Quality Report:**

Comprehensive quality assessment with metrics and issue aggregation. The report runs every check `ValidateAll` runs, so its counts are never clean for a document `ValidateAll` flags. The spec-conformance checks — header, broken pointers, nil elements, date validity, XRef length, note pointers and encoding — land in `ConformanceIssues` (JSON `conformance_issues`) beside the record-quality categories; the nameless-individual (`MISSING_NAME`) and memberless-family (`EMPTY_FAMILY`) heuristics land in `CompletenessIssues`. Unlike `ValidateAll`, the report reports every severity and applies none of `Strictness`, `SkipRules`, `MaxErrors`, `SkipDuplicateDetection` or `SkipEncodingValidation`; filter its issues yourself if you need those.

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
issue := validator.NewIssue(validator.SeverityError, "CUSTOM_ERROR", "Custom error message", "@I1@").
    WithRelatedXRef("@I2@").
    WithDetail("field", "NAME")
```

**Configurable Strictness:**

Control which severity levels are reported:

| Level | Reports |
|-------|---------|
| StrictnessNormal | Errors + Warnings (the zero value) |
| StrictnessRelaxed | Errors only |
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
var v *validator.StreamingValidator
var ver gedcom.Version
for raw, err := range parser.Records(charset.NewReader(file)) {
    if err != nil {
        return err
    }
    if raw.Type == "HEAD" {
        ver = version.DeclaredVersion(raw.Lines) // the header comes first
    }
    if v == nil {
        v = validator.NewStreamingValidator(validator.StreamingOptions{Version: ver})
    }
    record, err := decoder.DecodeRecord(raw, ver, nil) // typed Entity
    if err != nil {
        return err
    }
    for _, issue := range v.ValidateRecord(record) { // Immediate issues (date logic, structure)
        fmt.Printf("[%s] %s\n", issue.Severity, issue.Message)
    }
}

// Check cross-references after all records processed
finalIssues := v.Finalize()  // Orphaned reference detection
```

`ValidateRecord` reads the record's typed `Entity`. The streaming parsers yield
raw lines only, so decode each record with `decoder.DecodeRecord` first (see
[Typed Records](#typed-records)): a record holding only raw `Tags` is
registered as declared but nothing in it is checked.

Set `StreamingOptions.Version` to the document's GEDCOM version: the streaming
validator never sees the header, and `Finalize` needs it to treat `@VOID@` as
the 7.0 null pointer (never orphaned) or as a 5.5/5.5.1 identifier that must be
declared. Left empty, `@VOID@` is taken as the null pointer.

**Two-Phase Validation**:
- **Immediate**: Date logic errors (death before birth), structural issues detected per-record
- **Deferred**: Cross-reference validation (orphaned FAMC, FAMS, HUSB, WIFE, CHIL, SOUR, ASSO, REPO, NOTE) in `Finalize()`. Note pointers are checked on individual, family and source records and on every typed structure below them that carries notes -- names, events, attributes, source citations, LDS ordinances, change and creation dates, FAMC/FAMS links, ASSO, OBJE links, PLAC and REPO links; `Details["field"]` locates each (e.g. `Events[0].PlaceDetail.NoteXRefs[0]`)

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
| `StrictMode` | `bool` | Reject non-standard extensions |
| `OnProgress` | `ProgressCallback` | Progress reporting callback |
| `TotalSize` | `int64` | Expected file size for progress percentage |

The nesting ceiling is not an option: the GEDCOM grammar's two-digit level
field fixes it at `parser.MaxNestingDepth-1` (99).

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

Cancel long-running decodes with context. The decode checks the context on
every read of the input and before building each record's entity, so it stops
shortly after the context is done, having read little more of the input:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

opts := &decoder.DecodeOptions{Context: ctx}
doc, err := decoder.DecodeWithOptions(reader, opts)
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("Decode timed out")
}
```

The later stages take the same context, so one deadline can bound the whole
pipeline. Each checks it as it goes and returns its error wrapped, so
`errors.Is` matches it as above:

| Operation | Where the context goes | Checked |
|-----------|------------------------|---------|
| `validator.Validator.ValidateAllWithError` | `ValidateOptions.Context` | Between checks, and per record in date logic, nil-element and duplicate detection. `ValidateAll` and `Validate` stop early too but cannot report it |
| `converter.ConvertWithOptions` | `ConvertOptions.Context` | Per conversion step, and per record while copying the document and validating the result |
| `encoder.EncodeWithOptions`, `EncodeStreamingWithOptions`, `StreamEncoder.WriteRecord` | `EncodeOptions.Context` | Per record |
| `merge.Combine` | `CombineOptions.Context` | Per record compared, remapped and copied |
| `parser.LazyParser.BuildIndexContext` | its `ctx` argument | Per record indexed |

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
| CONC removal | Upgrade to 7.0 | Merges continuation tags into values, at any depth and including a level-0 `NOTE` record's text |
| CONT kept | Upgrade to 7.0 | CONT lines are written unchanged (7.0 keeps CONT) and not reported as a transformation |
| XRef normalization | To 7.0 (including from 7.0) | Rewrites cross-references to valid, collision-free 7.0 identifiers (uppercase, `_` for spaces and other invalid characters); a 5.5/5.5.1 record named `@VOID@`, the 7.0 null pointer, is renamed too |
| Note records → `SNOTE` | Upgrade to 7.0 | `NOTE` records become `SNOTE` shared note records, and `NOTE` pointers to them become `SNOTE` pointers |
| Dual-year dates | Upgrade to 7.0 | Kept as written (`1731/32`, which 7.0 cannot express) and reported as DataLoss and a Preserved note per date |
| Media types | Both | Maps between legacy (JPG) and IANA (image/jpeg) |
| Date calendars | Both | Maps calendar escapes (`@#DJULIAN@`) to 7.0 keywords (`JULIAN`) and back, per date in ranges and periods; non-standard calendars are left unchanged and reported as Preserved. A non-Gregorian calendar on a header date converted to 7.0 (`HEAD.DATE`, `HEAD.SOUR.DATA.DATE`, which 7.0 types as calendar-free `DateExact`) is left unchanged and reported as Preserved, needing attention |
| Newlines to CONT | Downgrade from 7.0 | Expands embedded newlines to CONT tags |
| Long lines to CONC | Downgrade from 7.0 | Splits a value line past 5.x's 255-character limit onto CONC tags, between two non-spaces, so the encoded output has no over-length line and decodes back to the same text |
| `SNOTE` → Note records | Downgrade from 7.0 | `SNOTE` records become `NOTE` records, and `SNOTE` pointers become `NOTE` pointers; the record's `MIME` and `LANG` are removed and reported as data loss |
| 7.0-only tags removed | Downgrade from 7.0 | `EXID`, `TRAN`, `PHRASE` and `CREA` are removed from `Record.Tags` with their subordinates, and from the typed entity's record-level fields, and reported as data loss |
| `UID` ↔ `_UID` | Both | `_UID`, the extension every 5.5.x desktop producer writes for the identifier 7.0 standardised as `UID`, is renamed to `UID` on an upgrade wherever 7.0 defines `UID` (level 1 of a record, level 2 under an event or attribute) and `UID` to `_UID` at any level on a downgrade (`UID_FROM_VENDOR_TAG`, `UID_TO_VENDOR_TAG`); neither is data loss |
| `FONE`/`ROMN` removed | To 7.0 or 5.5 | 5.5.1 phonetic and romanized variations are removed from `Record.Tags` with their subordinates (under `NAME` or `PLAC`), and from `PersonalName.Phonetic`/`Romanized`, and reported as data loss; they are not mapped to 7.0 `TRAN` |
| `@VOID@` pointers removed | Downgrade from 7.0 | 5.x has no null pointer, so each structure whose pointer is `@VOID@` (`FAMC`, `CHIL`, `SOUR` citation, `SNOTE`, ...) is removed with its subordinates from `Tags` and the typed entity; `HUSB`, `WIFE`, LDS `FAMC` and header `SUBM` fields are cleared. Reported as data loss |
| `_FREL`/`_MREL` → `FAMC.PEDI` | Upgrade | A child's `FAMC` gains the `PEDI` (`adopted`, `foster`, `sealing`) that every relationship the family's parents have agrees on, when that is not birth; reported as `PEDIGREE_FROM_VENDOR_TAG` |
| `_MARNM` → `TYPE MARRIED` name | Upgrade to 7.0 | A `NAME` carrying `_MARNM` is followed by a new `NAME` with `TYPE MARRIED`; reported as `MARRIED_NAME_FROM_VENDOR_TAG` |
| FamilySearch ARK EXID → `_FSFTID` | Downgrade from 7.0 | Maps to the vendor tag instead of dropping the ID (when `MapEXIDToVendorTags`) |
| Header grammar order | Both | Level-1 `HEAD` structures, including any the conversion adds (`GEDC`, `CHAR`), are placed in the target version's grammar order (`GEDC` first for 7.0, after `COPR` for 5.x); structures the target grammar does not define go last, in source order |
| `GEDC.VERS` | All, including same-version | `VERS` set to the target under `GEDC`; added under an existing `GEDC` that has none, even when source and target versions match |
| Typed entities | All, and a same-version conversion that rewrites a copy (7.0 XRef normalization, `GEDC.VERS` repair) | Every record with `Tags` has its `Entity` re-derived from the converted `Tags` (`decoder.SyncEntities`), so the typed model reads what the output encodes; a record with no `Tags` keeps its `Entity` |
| `GEDC.FORM` | To 5.5/5.5.1 | `2 FORM LINEAGE-LINKED` added under `GEDC` when missing (`GEDC_FORM_ADDED`), as both versions require; removed on the way to 7.0 (`GEDC_FORM_REMOVED`) |
| Version-foreign structures | 7.0 ↔ 5.5.1 | Every structure the source version defines and the target does not (7.0 `SDATE`, `INIL`, `CROP`, `FAM.FACT`, event `ASSO`, `NOTE.LANG`; 5.5.1 `HEAD.FILE`, ...) is removed from `Tags` with its subordinates and reported as data loss, per a table generated from the vendored specification files (`go generate ./converter`) |
| Inline media links → `OBJE` records | Upgrade to 7.0 | A pointer-less `OBJE` link carrying its `FILE` becomes a new `OBJE` record and a pointer to it (`INLINE_OBJE_TO_RECORD`) |
| `AFN`/`RFN`/`RIN` → `EXID` | Upgrade to 7.0 | Each becomes an `EXID` with its registered `TYPE` URI (`https://gedcom.io/terms/v7/AFN`, `RIN#<HEAD.SOUR>`, `RFN#<resource>`) (`IDENTIFIER_TO_EXID`) |
| `LANG` names ↔ BCP 47 | Both | `HEAD.LANG` and `SUBM.LANG` map between 5.x language names and BCP 47 tags (`English` ↔ `en`); a lossy or unknown mapping is an `Approximated` note |
| Required 5.x header | Downgrade from 7.0 | `HEAD.SOUR` (`gedcom-go`) and `HEAD.SUBM` added when missing, with a `SUBM` record (`NAME Unknown`) when the document has none |

### API

```go
// Simple conversion with defaults
converted, report, err := converter.Convert(doc, gedcom.Version70)

// With options
opts := &converter.ConvertOptions{
    Validate:            true,
    StrictDataLoss:      true,  // Fail on any data loss
    ReportPreservedTags: true,  // Itemise preserved vendor tags in the report
    MapEXIDToVendorTags: true,  // Map a 7.0 EXID to _FSFTID on downgrade
}
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version55, opts)

// Check report
if report.HasDataLoss() {
    for _, item := range report.DataLoss {
        fmt.Printf("Lost: %s - %s\n", item.Feature, item.Reason)
    }
}
```

Failures are matchable with `errors.Is`: `converter.ErrNilDocument`, `ErrInvalidVersion` (target), `ErrUnsupportedConversion` (no path from the source version) and `ErrDataLoss` (refused under `StrictDataLoss`; the report returned with it lists the loss).

See [docs/guides/converter.md](docs/guides/converter.md) for detailed documentation.

## Encoder

- Write valid GEDCOM files
- Configurable line endings (LF, CRLF)
- GEDCOM 5.5, 5.5.1, 7.0 output; a GEDCOM 7 patch version (`TargetVersion: "7.0.14"`, or such a `Header.Version`) is written as 7.0 by every version-specific rule, by the same rule the decoder reads `VERS 7.0.N` with, while `GEDC.VERS` keeps the value as given
- Header written from the source's `Header.Tags` in source order, so a same-version re-encode is byte-identical; a `GEDC` the encoder has to add for `TargetVersion` is placed where the target grammar puts it, and a 5.5/5.5.1 `GEDC` it writes (added, or from a hand-built header) carries `FORM LINEAGE-LINKED`
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

**Note pointers**: a record passed to `WriteRecord` is written without the rest of the document, so a note pointer cannot be resolved to an `SNOTE` record and is always written as `NOTE` (see [High-Level Type Encoding](#high-level-type-encoding)). `EncodeStreaming` has the whole document and writes pointers as `Encode` does.

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
| Repository | Name, address, contact info (phone, email, fax, website), notes |
| Submitter | Name, address, contact info, languages |
| Submission | Submitter ref, FAMF, TEMP, ANCE, DESC, ORDI, RIN, notes, change date |
| Note | Text with continuation lines |
| MediaObject | Files, formats, translations, citations |

A line break in a value is written as `CONT` lines. A line break in a tag name or identifier (`Tag.Tag`, `Tag.XRef`, `Record.Type`, `Record.XRef`, `Header.Submitter`, or an entity field that becomes a tag name such as `Event.Type`) has no GEDCOM spelling, so encoding returns `encoder.ErrLineBreak` rather than writing a line that could re-decode as a forged record.

The encoder does not write what its decoder cannot read back (issue #726). A subordinate `Tag.Level` outside 1-99 (including the zero value 0, which would open a record mid-record) or at 99 with a multi-line value, an empty tag name or one containing whitespace, a `Record.XRef` that does not open with `@`, a `Tag.XRef` not closed by `@`, an identifier whose whitespace the parser would split differently (`@A@ SUBM @B@`; a 5.5 `@I 1@` is still accepted), and a `HEAD`/`TRLR` record with no XRef return `encoder.ErrMalformedLine`; a value, tag name or identifier that is not valid UTF-8 returns `encoder.ErrInvalidUTF8`. Both errors name the record (or `HEAD`). A decoded document never trips them -- the unterminated `@I1` and tagless `0 @I 1@` a lenient decode recovers are still written back verbatim. A hand-built header declares `1 CHAR UTF-8` even with `Encoding` unset, except for 7.0 output, which has no `CHAR`. A nightly fuzz target (`FuzzEncodeRoundTrip`) checks that whatever the encoder writes without error re-decodes with the same record count.

Nil values in a hand-built document are written as nothing instead of panicking: a nil element in any slice (a record, a tag, a name, an event, a citation) is skipped, a nil `Header` encodes as an empty one, and a nil `Document` returns `encoder.ErrNilDocument`. The same skip-rather-than-panic rule applies library-wide — see [ADR 0007](docs/decisions/0007-error-transparency.md#nil-values) for the full policy, and the `encoder` package documentation for what the encoder writes for each shape.

Structures whose shape differs by version are written for the output version (issue #740), when a record is encoded from its entity. An association's role is `RELA` for 5.5/5.5.1 output and `ROLE` otherwise, and 5.5 output carries the `TYPE` it requires (`Association.RecordType`, `INDI` when unset); 5.5.1 and 7.0 output drop `RecordType`. An LDS `STAT` is written with its `DATE` (and, for 7.0, `TIME`) from `StatusDate`/`StatusTime`; a `Status` with no `StatusDate` is written alone rather than dropped (5.5.1 and 7.0 require the `DATE`, so set `StatusDate` for valid output), and 5.5 output, which has no `STAT.DATE`, writes `STAT` alone. A `BIRT`, `CHR` or `ADOP` event writes its `FAMC` from `Event.FamilyXRef`, and an `ADOP` event the `ADOP` beneath it from `AdoptedBy`; neither is written on any other event or on a `NO` assertion. A `MediaLink` with no pointer and nothing else is skipped rather than written as a bare `OBJE`; one with no pointer but a title, crop or note is still written, without the `FILE` the typed model does not yet hold (#733).

Note pointers (`NoteXRefs`, and `MediaObject.SharedNoteXRefs`) are written by what they point at. When the output is GEDCOM 7.0 (`TargetVersion`, or `Header.Version` when no target is set), a pointer to a document `SNOTE` record is written as `SNOTE @X@`; any other pointer -- to a `NOTE` record or to nothing -- stays `NOTE @X@`. A 5.5/5.5.1 output has no `SNOTE`, so every pointer is `NOTE`. This applies only to records encoded from their entities; a decoded record re-encodes from `Record.Tags` and keeps the tag it was read with.

### At-Sign Escaping

A literal `@` in a line value is escaped so it is not read as a pointer, by the
rule of the output version: GEDCOM 7.0 doubles a leading `@` (`@@start`, and the
same at the start of each `CONT` line), GEDCOM 5.5/5.5.1 double every `@`
(`a@@b.com`) except the `@` of an escape such as `@#DJULIAN@`. An unknown output
version takes the 7.0 rule. The encoder escapes every value it writes from a
typed field, so `InlineNotes: ["@I1@"]` is written `NOTE @@I1@` and re-decodes as
the same text rather than as a link to `@I1@`; the decoder undoes the escaping of
`Header.Version` for every typed text field -- the declared version, or for a file
that declares none the detected one, which is also the version the encoder
writes it as. A `CONC` split is measured on the escaped value, so doubling never
pushes a line past `MaxLineLength`, and never falls inside an `@@` pair. Pointer fields (`NoteXRefs`,
`Family.Husband`, `SourceCitation.SourceXRef`, ...) and raw `Tags` are written
and read as they are, so a decoded record re-encodes byte for byte and a `Tags`
value must already be in its wire form. A one-line `NOTE`/`SNOTE` record whose
`Record.Value` still reads as its entity's `Text` keeps that value as written.

### Round-Trip Encoding

Decode → modify → encode workflow:
- Lossless by default: original tags preserved when present
- Entity conversion: generates tags from typed fields when tags are empty
- All nested structures supported: events, names, citations, addresses, coordinates
- `encoder.SyncTags(doc)` carries typed edits on a decoded document into `Record.Tags` and `Header.Tags`, which the encoder writes verbatim. It applies only the difference between what the typed fields encode to now and what the raw tags decode and encode to, structure by structure: untouched structures keep their raw lines (unmodelled tags and original CONT/CONC split included), so an unmodified document is left line-for-line unchanged; a changed value keeps its unchanged substructures; added and removed structures are inserted and removed. Afterwards the tags read back as the typed fields. Header fields carried: `SourceSystem`, `Date`, `Submitter`, `Language`, `Version`, and whether `Encoding` is set (the encoder always writes `CHAR UTF-8`; not carried for 7.0, which has no `CHAR`). The typed fields win: an edit made to `Tags` alone is reverted, so call `decoder.SyncEntities` after editing `Tags` and before making typed edits (`converter.Convert` output already agrees). See [docs/guides/decoding.md](docs/guides/decoding.md#modifying-a-decoded-document)

### Line Continuation (CONT/CONC)

Automatic handling of multiline and long text per GEDCOM specification:

- **CONT (continuation)**: Multiline text automatically split on `\n` into CONT tags
- **CONC (concatenation)**: For GEDCOM 5.5/5.5.1, long lines (>248 bytes) are split between two non-space characters, never inside a UTF-8 character, so readers that trim lines lose no space. GEDCOM 7.0 removed CONC, so 7.0 output (`TargetVersion`, else `Header.Version`) is never split

```go
// Multiline text becomes CONT continuation
note := "Line one\nLine two\nLine three"
// Encodes as:
// 1 NOTE Line one
// 2 CONT Line two
// 2 CONT Line three

// In 5.x, long lines become CONC concatenation (split between non-spaces)
longText := "Very long text exceeding 248 characters..."
// Encodes as:
// 1 NOTE Very long text exceeding... the 248 charac
// 2 CONC ter limit...
```

Configurable via `EncodeOptions`:

| Option | Default | Description |
|--------|---------|-------------|
| `MaxLineLength` | 248 | Maximum line length (bytes) before CONC split; ignored for 7.0 |
| `DisableLineWrap` | false | Disable automatic CONC splitting; implied for 7.0 |

### Inline Repository Support

Sources support both XRef references and inline repository definitions:

```go
// XRef reference to separate repository record
source.RepositoryLinks = []*gedcom.SourceRepositoryLink{{XRef: "@R1@"}}
// Encodes as: 1 REPO @R1@

// Inline repository definition (no separate record needed)
source.RepositoryLinks = []*gedcom.SourceRepositoryLink{{
    Inline: &gedcom.InlineRepository{Name: "State Archives"},
}}
// Encodes as:
// 1 REPO
// 2 NAME State Archives

// Several links, each call number paired with its own media type
source.RepositoryLinks = []*gedcom.SourceRepositoryLink{
    {XRef: "@R1@", CallNumbers: []*gedcom.CallNumber{
        {Value: "MS-1234", MediaType: "MANUSCRIPT"},
        {Value: "MS-1234", MediaType: "FILM"},
    }},
    {XRef: "@R2@"},
}
// Encodes as:
// 1 REPO @R1@
// 2 CALN MS-1234
// 3 MEDI MANUSCRIPT
// 2 CALN MS-1234
// 3 MEDI FILM
// 1 REPO @R2@
```

Useful for sources imported from GEDCOM files where repository names are stored inline rather than as separate records.

## Incremental Parsing

Memory-efficient parsing for very large files with on-demand record access.

### Go 1.23+ Range-Over-Func Iterators

Modern Go 1.23 range-over-func support for idiomatic streaming:

```go
// Range-over-func pattern (Go 1.23+)
for record, err := range parser.Records(reader) {
    if err != nil {
        return fmt.Errorf("parse error: %w", err)
    }
    fmt.Printf("Record: %s %s\n", record.Type, record.XRef)
}

// With byte offset tracking for indexing
for record, err := range parser.RecordsWithOffset(reader) {
    if err != nil {
        return err
    }
    fmt.Printf("Record %s at byte %d, length %d\n",
        record.Type, record.ByteOffset, record.ByteLength)
}

// LazyParser also supports range-over-func
lp := parser.NewLazyParser(file)
for record, err := range lp.AllRecords() {
    if err != nil {
        return err
    }
    // process record
}
```

| Function | Description |
|----------|-------------|
| `Records(r)` | Iterate records from reader using range-over-func |
| `RecordsWithOffset(r)` | Iterate with accurate byte offsets for indexing |
| `LazyParser.Records()` | Iterate from current position |
| `LazyParser.AllRecords()` | Iterate all records from beginning |
| `LazyParser.RecordsFrom(offset)` | Iterate from specific byte offset |

**Benefits**:
- Clean, idiomatic Go 1.23 syntax
- Early termination with `break` stops iteration
- Error handling via second return value
- Zero overhead compared to classic iterator

The classic iterator API (`RecordIterator.Next()/Record()/Err()`) remains available
for backward compatibility and pre-Go 1.23 codebases.

### Record Iterator (Classic API)

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
| `RawRecord.ByteOffset` | Starting byte position: absolute in the file for `LazyParser` iterators, relative to the start of the reader for `Records`/`NewRecordIterator` |
| `RawRecord.ByteLength` | Total bytes for this record |

### Typed Records

`parser.Records`, `RecordIterator` and `LazyParser` yield `*parser.RawRecord`:
raw lines, with no typed entity. `decoder.DecodeRecord` turns one into the
`*gedcom.Record` `decoder.Decode` would give it — same `Entity`, `Tags` and
fields — without building a `Document`:

```go
// ver is the version the header declares: version.DeclaredVersion(head.Lines)
raw, err := lp.FindRecord("@I1@")
if err != nil {
    return err
}
record, err := decoder.DecodeRecord(raw, ver, nil)
if err != nil {
    return err
}
person, _ := record.GetIndividual()
```

| Behaviour | Detail |
|-----------|--------|
| Version | The declared version decides the identifier grammar and `@@` unescaping; `""` reads text by the 5.5 rule |
| Options | `StrictMode` and `Context` only; lenient mode clamps level jumps as `Decode` does, on a copy |
| HEAD / TRLR | A `Record` of that `Type` with its `Tags` and no `Entity` |
| Errors | Nil or empty record, or a first line that is not level 0 |
| Diagnostics | Not reported; use `DecodeWithDiagnostics` for those |

Costs are in [the performance guide](docs/guides/performance.md#typed-streaming-decoderdecoderecord).

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
it, err := lp.IterateAll()
if err != nil {
    log.Fatal(err)
}
for it.Next() {
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
- Benchmark coverage across parser, decoder, encoder, and validator hot paths.
  Run `make bench` to reproduce figures on your own hardware — absolute numbers
  are hardware- and warmup-dependent, so they are not quoted here.

## API Design

- Clean, idiomatic Go API
- Comprehensive godoc documentation
- Example code for common use cases
- Single dependency: `golang.org/x/text` (Go's extended standard library) for encoding transforms; no third-party dependencies

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
| `GetSubmission(xref)` | `*Submission` | Submission lookup |
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
| `Submissions()` | `[]*Submission` | All submissions |
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
| `FamiliesAsChild(doc)` | `[]*Family` | Families where individual is a child |
| `FamiliesAsSpouse(doc)` | `[]*Family` | Families where individual is a spouse |

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
    fmt.Println(parent.FullName())
}
for _, spouse := range person.Spouses(doc) {
    fmt.Println(spouse.FullName())
}
for _, child := range person.Children(doc) {
    fmt.Println(child.FullName())
}

// Navigate from family to members
family := doc.GetFamily("@F1@")
husband := family.HusbandIndividual(doc)
wife := family.WifeIndividual(doc)
children := family.ChildrenIndividuals(doc)
```

### Event and Date Access

Convenience methods for accessing names, parsed events and dates on individuals.
All are safe on a nil receiver and on nil slice entries; use `Name()` or
`FullName()` instead of indexing `Names[0]`, which panics on a nameless
individual:

| Method | Return Type | Description |
|--------|-------------|-------------|
| `Name()` | `*PersonalName` | First name (nil if none) |
| `FullName()` | `string` | `Full` of the first name (`""` if none) |
| `BirthEvent()` | `*Event` | First birth event (nil if none) |
| `DeathEvent()` | `*Event` | First death event (nil if none) |
| `BirthDate()` | `*Date` | Parsed birth date (nil if no event or no date) |
| `DeathDate()` | `*Date` | Parsed death date (nil if no event or no date) |
| `BirthPlace()` | `string` | Birth place name (`""` if no event or no place) |
| `DeathPlace()` | `string` | Death place name (`""` if no event or no place) |
| `EventOf(eventType)` | `*Event` | First event of that type (nil if none) |
| `AttributeOf(attributeType)` | `*Attribute` | First attribute of that type (nil if none) |
| `OccurredEvents()` | `[]*Event` | Events that happened: `Events` without negative assertions or nil entries (also on `Family`) |
| `NegativeAssertions()` | `[]*Event` | GEDCOM 7.0 `NO <EVENT>` assertions only (also on `Family`) |
| `Family.MarriageEvent()` | `*Event` | First marriage event of a family (nil if none) |

The event accessors skip GEDCOM 7.0 negative assertions: a `NO DEAT` is not a death, so `DeathEvent()` never returns one. See [Negative Assertions (NO)](#negative-assertions-no).

```go
// Access birth and death events directly
person := doc.GetIndividual("@I1@")
if birth := person.BirthEvent(); birth != nil {
    fmt.Printf("Born: %s at %s\n", birth.Date, birth.PlaceName())
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
| `IsSharedNote()` | `bool` | True if record contains a SharedNote |
| `GetIndividual()` | `(*Individual, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetFamily()` | `(*Family, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetSource()` | `(*Source, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetRepository()` | `(*Repository, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetSubmitter()` | `(*Submitter, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetSubmission()` | `(*Submission, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetNote()` | `(*Note, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetMediaObject()` | `(*MediaObject, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |
| `GetSharedNote()` | `(*SharedNote, bool)` | Type assertion (`(nil, false)` if wrong type or a typed-nil entity) |

All twelve are nil-receiver safe: on a nil `*Record` every `Is*` returns `false` and every `Get*` returns `(nil, false)`. See [ADR 0007](docs/decisions/0007-error-transparency.md#nil-values).

```go
for _, record := range doc.Records {
    if indi, ok := record.GetIndividual(); ok {
        fmt.Printf("Individual: %s\n", indi.XRef)
    }
}
```

## Testing

- Per-package test coverage floor of ≥85% enforced in CI (currently ~96% measured; run `make test-coverage`)
- Multi-platform CI (Linux, macOS, Windows)
- Multi-version Go testing (1.25, 1.26)
- Benchmark regression testing
- Real-world GEDCOM file testing

### Round-Trip Test Helper

The `gedcom/testing` package provides helpers to verify encode/decode cycles preserve data:

```go
import gedcomtesting "github.com/cacack/gedcom-go/v3/gedcom/testing"

func TestMyGEDCOM(t *testing.T) {
    data, _ := os.ReadFile("family.ged")
    gedcomtesting.AssertRoundTrip(t, data)
}
```

For programmatic use:

```go
report, err := gedcomtesting.CheckRoundTrip(file)
if !report.Equal {
    fmt.Println(report.String())
}
```

**Fidelity Contract** - what is preserved:
- All tags, values, and hierarchical structure
- Unknown/vendor tags
- XRef identifiers
- Record and tag order

**What may change** (not compared):
- Line numbers
- Formatting (line endings, whitespace)
- CONC/CONT reorganization
