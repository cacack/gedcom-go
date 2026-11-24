# Usage Guide

This guide provides comprehensive documentation for using the `gedcom-go` library to parse, query, validate, and create GEDCOM files in your Go applications.

## Table of Contents

- [Basic Concepts](#basic-concepts)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [The Facade Package](#the-facade-package)
- [Parsing GEDCOM Files](#parsing-gedcom-files)
- [Working with Documents](#working-with-documents)
- [Querying Data](#querying-data)
- [Validation](#validation)
- [Creating GEDCOM Files](#creating-gedcom-files)
- [Converting Between Versions](#converting-between-versions)
- [Combining and Merging](#combining-and-merging)
- [Comparing Documents](#comparing-documents)
- [GEDZIP Archives](#gedzip-archives)
- [Character Encoding](#character-encoding)
- [Error Handling](#error-handling)
- [Advanced Usage](#advanced-usage)
- [Performance](#performance)
- [Best Practices](#best-practices)

## Basic Concepts

### GEDCOM Structure

GEDCOM (GEnealogical Data COMmunication) is a hierarchical file format for exchanging genealogical data between different genealogy software applications. The format uses a line-based structure with hierarchical levels.

**Supported Versions:**

- **GEDCOM 5.5** - Most widely used version (1996)
- **GEDCOM 5.5.1** - Enhanced with additional tags like EMAIL, WWW, MAP (1999)
- **GEDCOM 7.0** - Modern version with new structure and tags (2021)

**Line Format:**

Each line in a GEDCOM file follows this pattern:
```
LEVEL [XREF] TAG [VALUE]
```

Example:
```
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
1 SEX M
1 BIRT
2 DATE 1 JAN 1900
2 PLAC New York, USA
```

### Key Data Types

Understanding these core types will help you work with the library effectively:

- **`Document`** - The root container for all GEDCOM data, including header, records, and cross-reference map
- **`Header`** - Metadata about the GEDCOM file (version, encoding, source system, creation date)
- **`Record`** - Top-level records representing individuals, families, sources, repositories, etc.
- **`Individual`** - Person records with names, events (birth, death, etc.), and family relationships
- **`Family`** - Family units linking spouses (husband/wife) and children
- **`Tag`** - The hierarchical data structure within records (represents each line in the GEDCOM file)
- **`Event`** - Life events like births, deaths, marriages with dates and places
- **`Source`** - Citation sources (books, documents, websites, etc.)
- **`Repository`** - Physical or digital locations where sources are stored

### Record Types

The main record types you'll encounter:

- `INDI` - Individual (person)
- `FAM` - Family
- `SOUR` - Source
- `REPO` - Repository
- `NOTE` - Note
- `OBJE` - Multimedia object
- `SUBM` - Submitter

## Installation

Install the library using `go get`:

```bash
go get github.com/cacack/gedcom-go/v3
```

Requirements:
- Go 1.25 or later
- One dependency, `golang.org/x/text`, fetched by `go get` (see [FEATURES.md](FEATURES.md#api-design))

## Quick Start

Here's a minimal example to get started:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/cacack/gedcom-go/v3/decoder"
)

func main() {
    // Open GEDCOM file
    f, err := os.Open("family.ged")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    // Decode the file
    doc, err := decoder.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    // Print basic information
    fmt.Printf("GEDCOM Version: %s\n", doc.Header.Version)
    fmt.Printf("Individuals: %d\n", len(doc.Individuals()))
    fmt.Printf("Families: %d\n", len(doc.Families()))
}
```

## The Facade Package

The module root, imported as `gedcomgo`, wraps the common operations of the
sub-packages behind one import. [README.md](README.md) leads with it; the rest
of this guide uses the sub-packages directly. The two are interchangeable:
each facade function forwards to its sub-package, and the facade's types
(`Document`, `Individual`, `Family`, `Issue`, `DecodeOptions` and so on) are
aliases of the sub-package types, not copies.

```go
import gedcomgo "github.com/cacack/gedcom-go/v3"

doc, err := gedcomgo.Decode(f)
if err != nil {
    log.Fatal(err)
}

for _, issue := range gedcomgo.ValidateAll(doc) {
    fmt.Println(issue)
}

converted, report, err := gedcomgo.Convert(doc, gedcomgo.Version70)
if err != nil {
    log.Fatal(err)
}
fmt.Println(report)

err = gedcomgo.Encode(out, converted)
```

The facade also has `DecodeWithOptions`, `DecodeWithDiagnostics`,
`EncodeWithOptions`, `Validate`, `ValidateWithOptions`,
`ValidateAllWithOptions` and `WithoutDiagnosed`; see
[FEATURES.md](FEATURES.md#convenience-api) for the full list.

Import a sub-package when you need what the facade does not carry:

- `decoder`: diagnostics with custom options, `DecodeRecord`, `DecodeRecords`
- `encoder`: the stream encoder, `EncodeRecord`, `SyncTags`
- `validator`: a reusable `Validator`, duplicate and quality reports, the streaming validator
- `converter`: conversion options such as `StrictDataLoss`
- `gedcom`: the record types and constants beyond the three the facade re-exports
- `merge`, `gedzip`, `parser`, `charset`, `version`: not in the facade at all

## Parsing GEDCOM Files

### Basic Parsing

The `decoder` package provides the high-level API for parsing GEDCOM files:

```go
import "github.com/cacack/gedcom-go/v3/decoder"

// Simple decoding
f, _ := os.Open("family.ged")
defer f.Close()

doc, err := decoder.Decode(f)
if err != nil {
    log.Fatal(err)
}
```

### Parsing with Options

For more control, use `DecodeWithOptions`:

```go
import (
    "context"
    "time"
    "github.com/cacack/gedcom-go/v3/decoder"
)

// Set a timeout for parsing large files
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

opts := &decoder.DecodeOptions{
    Context: ctx,

    // Strict mode - fail on the first parse error instead of recovering
    StrictMode: false,
}

doc, err := decoder.DecodeWithOptions(f, opts)
```

### Parsing from Different Sources

```go
// From a file
f, _ := os.Open("data.ged")
doc, _ := decoder.Decode(f)

// From a byte slice
data := []byte("0 HEAD\n1 GEDC\n2 VERS 5.5\n0 TRLR\n")
doc, _ := decoder.Decode(bytes.NewReader(data))

// From a string
gedcomString := "0 HEAD\n..."
doc, _ := decoder.Decode(strings.NewReader(gedcomString))

// From HTTP response
resp, _ := http.Get("https://example.com/family.ged")
defer resp.Body.Close()
doc, _ := decoder.Decode(resp.Body)
```

### Lenient Decoding and Diagnostics

Decoding is lenient by default: a malformed line is recovered from or skipped
rather than failing the decode. `Decode` and `DecodeWithOptions` discard the
record of what was recovered; `DecodeWithDiagnostics` returns it.

```go
result, err := decoder.DecodeWithDiagnostics(f, nil) // nil: the defaults, lenient
if result == nil {
    log.Fatal(err) // nothing could be decoded
}
if err != nil {
    log.Printf("decode stopped early: %v", err) // result.Document is partial
}
doc := result.Document

for _, d := range result.Diagnostics {
    // d.Severity is decoder.SeverityError, SeverityWarning or SeverityInfo
    fmt.Printf("line %d [%s] %s: %s\n", d.Line, d.Severity, d.Code, d.Message)
    fmt.Printf("  in %s at %s: %q\n", d.XRef, d.Path, d.Context) // e.g. @I1@, INDI.BURI.DATE
}

if result.Diagnostics.HasErrors() {
    fmt.Printf("%d errors, %d warnings\n",
        len(result.Diagnostics.Errors()), len(result.Diagnostics.Warnings()))
}
```

`Code` is one of the `decoder.Code*` constants. `XRef` is the record the line
is in and `Path` the tags from that record down to the line; both are empty
for a line outside any record. With `StrictMode` set the decode returns the
first error instead and reports no diagnostics.

The decoder and the validator both report a redefined XRef as
`DUPLICATE_XREF`. To report the two lists together without counting it twice,
pass the issues through `gedcomgo.WithoutDiagnosed` (from the
[facade package](#the-facade-package)):

```go
issues := validator.New().ValidateAll(result.Document)
issues = gedcomgo.WithoutDiagnosed(issues, result.Diagnostics)
```

The [decoding contract](docs/guides/decoding.md) lists every diagnostic code,
what each recovery keeps, and the bound on how many diagnostics are reported.

## Working with Documents

### Document Structure

A `Document` contains:

```go
type Document struct {
    Header  *Header      // File header with version, encoding, etc.
    Records []*Record    // All records in the file
    Trailer *Trailer     // File trailer
    XRefMap map[string]*Record  // Cross-reference lookup map
    // ... plus Vendor (detected source software) and Schema (GEDCOM 7.0 SCHMA)
}
```

### Accessing the Header

```go
// Get GEDCOM version
version := doc.Header.Version  // "5.5", "5.5.1", or "7.0"

// Get character encoding
encoding := doc.Header.Encoding  // "UTF-8", "ANSEL", "ASCII", etc.

// Get source system
if doc.Header.SourceSystem != "" {
    fmt.Printf("Created by: %s\n", doc.Header.SourceSystem)
}

// Get file date
if doc.Header.Date != "" {
    fmt.Printf("File date: %s\n", doc.Header.Date)
}

// Get language
if doc.Header.Language != "" {
    fmt.Printf("Language: %s\n", doc.Header.Language)
}
```

### Working with Records

Records are the core building blocks of GEDCOM files:

```go
// Iterate through all records
for _, record := range doc.Records {
    fmt.Printf("Record: %s (Type: %s)\n", record.XRef, record.Type)

    // Access tags
    for _, tag := range record.Tags {
        fmt.Printf("  Level %d: %s = %s\n", tag.Level, tag.Tag, tag.Value)
    }
}

// Get record by cross-reference
record := doc.GetRecord("@I1@")
if record != nil {
    fmt.Printf("Found: %s\n", record.XRef)
}

// Count records by type
recordCounts := make(map[gedcom.RecordType]int)
for _, record := range doc.Records {
    recordCounts[record.Type]++
}
```

## Querying Data

### Working with Individuals

```go
// Get all individuals
individuals := doc.Individuals()
fmt.Printf("Total individuals: %d\n", len(individuals))

// Iterate through individuals
for _, person := range individuals {
    // Get names
    if name := person.Name(); name != nil {
        fmt.Printf("%s: %s\n", person.XRef, name.Full)
        fmt.Printf("  Given: %s, Surname: %s\n", name.Given, name.Surname)
    }

    // Get sex
    if person.Sex != "" {
        fmt.Printf("  Sex: %s\n", person.Sex)
    }

    // Get events (birth, death, etc.)
    for _, event := range person.Events {
        fmt.Printf("  %s: %s", event.Type, event.Date)
        if place := event.PlaceName(); place != "" {
            fmt.Printf(" at %s", place)
        }
        fmt.Println()
    }
}

// Look up specific individual
person := doc.GetIndividual("@I1@")
fmt.Printf("Found: %s\n", person.FullName()) // "" if no such person or no name
```

### Working with Names

```go
for _, person := range doc.Individuals() {
    for _, name := range person.Names {
        // Full name (includes surname markers)
        fmt.Printf("Full name: %s\n", name.Full)  // "John /Doe/"

        // Parsed components
        fmt.Printf("Given: %s\n", name.Given)     // "John"
        fmt.Printf("Surname: %s\n", name.Surname) // "Doe"
        fmt.Printf("Prefix: %s\n", name.Prefix)   // "Dr."
        fmt.Printf("Suffix: %s\n", name.Suffix)   // "Jr."

        // Name type
        if name.Type != "" {
            fmt.Printf("Type: %s\n", name.Type)   // "aka", "birth", "married"
        }
    }
}
```

### Working with Events

Events include births, deaths, marriages, and other life events:

```go
for _, person := range doc.Individuals() {
    for _, event := range person.Events {
        // Event type (BIRT, DEAT, MARR, etc.)
        fmt.Printf("Event: %s\n", event.Type)

        // Date (in GEDCOM format)
        if event.Date != "" {
            fmt.Printf("  Date: %s\n", event.Date)  // "1 JAN 1900"
        }

        // Place
        if place := event.PlaceName(); place != "" {
            fmt.Printf("  Place: %s\n", place)
        }

        // Additional details
        if event.Description != "" {
            fmt.Printf("  Description: %s\n", event.Description)
        }

        // Source citations
        for _, citation := range event.SourceCitations {
            fmt.Printf("  Source: %s (%s)\n", citation.SourceXRef, citation.Page)
        }
    }
}

// Find specific event type
for _, person := range doc.Individuals() {
    if event := person.EventOf(gedcom.EventBirth); event != nil {
        fmt.Printf("%s was born on %s\n", person.FullName(), event.Date)
    }
}
```

### Working with Families

```go
// Get all families
families := doc.Families()
fmt.Printf("Total families: %d\n", len(families))

// Iterate through families
for _, family := range families {
    fmt.Printf("Family: %s\n", family.XRef)

    // Get husband
    if family.Husband != "" {
        husband := doc.GetIndividual(family.Husband)
        if husband != nil {
            fmt.Printf("  Husband: %s\n", husband.FullName())
        }
    }

    // Get wife
    if family.Wife != "" {
        wife := doc.GetIndividual(family.Wife)
        if wife != nil {
            fmt.Printf("  Wife: %s\n", wife.FullName())
        }
    }

    // Get children
    fmt.Printf("  Children: %d\n", len(family.Children))
    for _, childXRef := range family.Children {
        child := doc.GetIndividual(childXRef)
        if child != nil {
            fmt.Printf("    - %s\n", child.FullName())
        }
    }

    // Get family events (marriage, divorce, etc.)
    for _, event := range family.Events {
        fmt.Printf("  %s: %s\n", event.Type, event.Date)
    }
}

// Find individual's families
person := doc.GetIndividual("@I1@")
if person != nil {
    // Families where this person is a spouse
    for _, link := range person.SpouseInFamilies {
        family := doc.GetFamily(link.FamilyXRef)
        if family != nil { // nil for an orphaned FAMS pointer
            fmt.Printf("Spouse in family: %s (%d children)\n", family.XRef, len(family.Children))
        }
    }

    // Families where this person is a child
    for _, link := range person.ChildInFamilies {
        family := doc.GetFamily(link.FamilyXRef)
        if family != nil { // nil for an orphaned FAMC pointer
            fmt.Printf("Child in family: %s (pedigree: %q)\n", family.XRef, link.Pedigree)
        }
    }
}
```

### Working with Sources

```go
// Get all sources
sources := doc.Sources()
fmt.Printf("Total sources: %d\n", len(sources))

for _, source := range sources {
    fmt.Printf("Source: %s\n", source.XRef)
    fmt.Printf("  Title: %s\n", source.Title)

    if source.Author != "" {
        fmt.Printf("  Author: %s\n", source.Author)
    }

    if source.Publication != "" {
        fmt.Printf("  Publication: %s\n", source.Publication)
    }

    for _, link := range source.RepositoryLinks {
        if link.XRef != "" {
            fmt.Printf("  Repository: %s\n", link.XRef)
        }
        for _, caln := range link.CallNumbers {
            fmt.Printf("    Call number: %s (%s)\n", caln.Value, caln.MediaType)
        }
    }
}
```

### Working with Repositories

```go
// Get all repositories
repositories := doc.Repositories()

for _, repo := range repositories {
    fmt.Printf("Repository: %s\n", repo.XRef)
    fmt.Printf("  Name: %s\n", repo.Name)

    if repo.Address != nil {
        fmt.Printf("  Address: %s\n", repo.Address.Full)
        fmt.Printf("  City: %s\n", repo.Address.City)
        fmt.Printf("  Country: %s\n", repo.Address.Country)
    }
}
```

### Working with Notes

```go
// Get all notes
notes := doc.Notes()

for _, note := range notes {
    fmt.Printf("Note %s:\n", note.XRef)
    fmt.Printf("  %s\n", note.Text)
}
```

### Working with Multimedia

The library provides full GEDCOM 7.0 multimedia support including multiple files per object, crop regions, and MIME types.

```go
// Get all media objects
mediaObjects := doc.MediaObjects()
fmt.Printf("Total media objects: %d\n", len(mediaObjects))

for _, media := range mediaObjects {
    fmt.Printf("Media: %s\n", media.XRef)

    // Each media object can have multiple files
    for _, file := range media.Files {
        fmt.Printf("  File: %s\n", file.FileRef)
        fmt.Printf("    MIME Type: %s\n", file.Form)      // e.g., "image/jpeg"
        fmt.Printf("    Media Type: %s\n", file.MediaType) // e.g., "PHOTO", "VIDEO"

        if file.Title != "" {
            fmt.Printf("    Title: %s\n", file.Title)
        }

        // File translations (thumbnails, transcripts, alternate formats)
        for _, tran := range file.Translations {
            fmt.Printf("    Translation: %s (%s)\n", tran.FileRef, tran.Form)
        }
    }

    // Metadata
    if media.Restriction != "" {
        fmt.Printf("  Restriction: %s\n", media.Restriction) // CONFIDENTIAL, LOCKED, etc.
    }
}

// Look up specific media object
media := doc.GetMediaObject("@O1@")
if media != nil {
    fmt.Printf("Found media with %d files\n", len(media.Files))
}
```

#### Media Links with Crop Regions

When entities reference media objects, they use `MediaLink` which can include crop regions and title overrides:

```go
// Access media linked to an individual
person := doc.GetIndividual("@I1@")
if person != nil {
    for _, mediaLink := range person.Media {
        fmt.Printf("Media ref: %s\n", mediaLink.MediaXRef)

        // Optional title override (overrides the file's TITL)
        if mediaLink.Title != "" {
            fmt.Printf("  Title: %s\n", mediaLink.Title)
        }

        // Optional crop region (for showing a portion of an image)
        if mediaLink.Crop != nil {
            fmt.Printf("  Crop: top=%d, left=%d, %dx%d\n",
                mediaLink.Crop.Top,
                mediaLink.Crop.Left,
                mediaLink.Crop.Width,
                mediaLink.Crop.Height)
        }

        // Resolve to full media object
        mediaObj := doc.GetMediaObject(mediaLink.MediaXRef)
        if mediaObj != nil && len(mediaObj.Files) > 0 {
            fmt.Printf("  Actual file: %s\n", mediaObj.Files[0].FileRef)
        }
    }
}
```

## Validation

### Validation as Errors

`Validate` runs the same checks as `ValidateAll` and returns the findings as
`[]error`. At the default strictness that includes warnings as well as errors.

```go
import "github.com/cacack/gedcom-go/v3/validator"

// Create validator
v := validator.New()

// Validate document
errs := v.Validate(doc)

if len(errs) == 0 {
    fmt.Println("✓ Validation passed!")
} else {
    fmt.Printf("Found %d validation issues:\n", len(errs))
    for _, err := range errs {
        fmt.Printf("  - %v\n", err)
    }
}
```

### Working with Validation Errors

Every element `Validate` returns is a `*validator.Issue`:

```go
for _, err := range errs {
    var issue *validator.Issue
    if errors.As(err, &issue) {
        fmt.Printf("Severity: %s\n", issue.Severity)
        fmt.Printf("Code: %s\n", issue.Code) // compare against validator.Code* constants
        fmt.Printf("Message: %s\n", issue.Message)

        if issue.LineNumber > 0 {
            fmt.Printf("Line: %d\n", issue.LineNumber)
        }

        if issue.RecordXRef != "" {
            fmt.Printf("XRef: %s\n", issue.RecordXRef)
        }
    }
}
```

### Grouping Validation Errors

```go
// Group issues by code
issuesByCode := make(map[string][]*validator.Issue)
for _, err := range errs {
    var issue *validator.Issue
    if errors.As(err, &issue) {
        issuesByCode[issue.Code] = append(issuesByCode[issue.Code], issue)
    }
}

// Display summary
for code, issues := range issuesByCode {
    fmt.Printf("%s: %d occurrences\n", code, len(issues))
}
```

### ValidateAll and ValidateOptions

`ValidateAll` returns the findings as `[]validator.Issue`, so no `errors.As`
is needed. `ValidateOptions` selects which findings are reported:

```go
v := validator.NewWithOptions(&validator.ValidateOptions{
    // StrictnessRelaxed: errors only. StrictnessNormal (the default): errors
    // and warnings. StrictnessStrict: info findings as well.
    Strictness: validator.StrictnessStrict,

    // Drop findings by code.
    SkipRules: []string{validator.CodeMissingSUBM},

    // Duplicate detection thresholds; a zero field keeps its default.
    Duplicates: &validator.DuplicateConfig{MinConfidence: 0.9},
    // SkipDuplicateDetection: true, // or leave the costliest check out
})

issues := v.ValidateAll(doc)
for _, issue := range validator.FilterBySeverity(issues, validator.SeverityError) {
    fmt.Printf("%s %s (line %d): %s\n", issue.Code, issue.RecordXRef, issue.LineNumber, issue.Message)
}
for _, issue := range validator.FilterByCode(issues, validator.CodePotentialDuplicate) {
    fmt.Printf("%s and %s may be one person\n", issue.RecordXRef, issue.RelatedXRef)
}
```

Potential duplicates are info findings, so they appear only at
`StrictnessStrict`; `Validator.FindPotentialDuplicates` returns the scored
pairs directly. `LineNumber` is 0 for a finding with no single source line.
The other options (`CheckGrammar`, `CheckEntityTags`, `MaxErrors`, custom tag
registries, `Context`) and the duplicate scoring are described under
[Enhanced Data Validation](FEATURES.md#enhanced-data-validation) in FEATURES.md.

## Creating GEDCOM Files

### Creating a Simple Document

`gedcom.NewDocument` returns an empty document whose header declares the
version (`Header.Version` and `Header.DeclaredVersion` both set), and
`Document.AddRecord` appends a record and indexes it in `XRefMap`, so lookups
such as `GetIndividual` and `Parents` find it. A record built with only an
`Entity` (no `Tags`) is encoded from its typed fields. A minimal GEDCOM 7.0
file:

```go
doc := gedcom.NewDocument(gedcom.Version70)
err := doc.AddRecord(&gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
    Entity: &gedcom.Individual{XRef: "@I1@", Sex: "F",
        Names: []*gedcom.PersonalName{{Full: "Ada /Lovelace/"}}}})
if err != nil {
    log.Fatal(err) // wraps gedcom.ErrDuplicateXRef if @I1@ is already in doc
}
err = encoder.Encode(os.Stdout, doc)
```

Header fields such as `SourceSystem` and `Language` can be set on
`doc.Header` before encoding. If you edit `doc.Records` directly instead of
through `AddRecord`, call `doc.ReindexXRefs()` afterwards to rebuild
`XRefMap`.

### Replacing and Removing Records

`Document.ReplaceRecord` and `Document.RemoveRecord` keep `Records` and
`XRefMap` in step. `decoder.DecodeRecords` turns GEDCOM text holding only
records (no `HEAD`, no `TRLR`), such as a record printed and then edited, into
records read as if they were in the document: its version, identifier grammar
and encoding.

```go
records, diags, err := decoder.DecodeRecords(strings.NewReader(edited), doc, nil)
if err != nil {
    log.Fatal(err) // wraps decoder.ErrNotRecords for a HEAD, TRLR or leading subordinate line
}
for _, d := range diags {
    fmt.Println(d) // lines counted from the fragment's first line
}
if err := doc.ReplaceRecord(records[0]); err != nil {
    log.Fatal(err) // wraps gedcom.ErrUnknownXRef when doc has no such record
}

refs, err := doc.RemoveRecord("@I2@")
if err != nil {
    log.Fatal(err)
}
for _, ref := range refs {
    fmt.Println(ref.XRef, strings.Join(ref.Path, ".")) // e.g. @F1@ FAM.CHIL, now dangling
}
```

The replacement keeps the replaced record's position in `Records`.
`RemoveRecord` reports the pointers that still name the removed record and
leaves them in place; remove or repoint them before encoding. Encoded with
`PreserveLayout` (see [Preserving Layout](docs/guides/preserving-layout.md)),
every other record is written exactly as it was read.

### Adding Individuals

```go
err := doc.AddRecord(&gedcom.Record{
    XRef: "@I1@",
    Type: gedcom.RecordTypeIndividual,
    Entity: &gedcom.Individual{
        XRef: "@I1@",
        Names: []*gedcom.PersonalName{
            {Full: "John /Doe/", Given: "John", Surname: "Doe"},
        },
        Sex: "M",
        Events: []*gedcom.Event{
            {Type: "BIRT", Date: "1 JAN 1900", PlaceDetail: &gedcom.PlaceDetail{Name: "New York, USA"}},
        },
        SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
    },
})
```

### Adding Families

```go
err := doc.AddRecord(&gedcom.Record{
    XRef: "@F1@",
    Type: gedcom.RecordTypeFamily,
    Entity: &gedcom.Family{
        XRef:    "@F1@",
        Husband: "@I1@",
        Wife:    "@I2@",
        Events: []*gedcom.Event{
            {Type: "MARR", Date: "15 JUN 1925", PlaceDetail: &gedcom.PlaceDetail{Name: "Boston, Massachusetts, USA"}},
        },
    },
})
```

Set `Tags` instead of (or as well as) `Entity` only when you need to write
structures the typed model has no field for: `Tags` is authoritative when
encoding, and the encoder builds tags from `Entity` only for a record that has
none (see `Record.Tags`).

### Encoding with Options

```go
opts := &encoder.EncodeOptions{
    // Line ending style
    LineEnding: "\r\n",  // CRLF (Windows)
    // LineEnding: "\n",  // LF (the default)
}

err := encoder.EncodeWithOptions(f, doc, opts)
if err != nil {
    log.Fatal(err)
}
```

## Converting Between Versions

The `converter` package converts a document between GEDCOM 5.5, 5.5.1 and
7.0. It returns a converted copy and a report; the input is not modified.

```go
import "github.com/cacack/gedcom-go/v3/converter"

converted, report, err := converter.Convert(doc, gedcom.Version70)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("%s -> %s\n", report.SourceVersion, report.TargetVersion)
for _, t := range report.Transformations {
    fmt.Printf("%s x%d: %s\n", t.Type, t.Count, t.Description)
}
if report.HasDataLoss() {
    for _, d := range report.DataLoss {
        fmt.Printf("lost %s: %s (%d records)\n", d.Feature, d.Reason, len(d.AffectedRecords))
    }
}
for _, n := range report.AllNotes() { // Dropped, Normalized, Approximated, Preserved
    fmt.Printf("%s: %q -> %q (%s)\n", n.Path, n.Original, n.Result, n.Reason)
}

err = encoder.Encode(out, converted)
```

Data loss is reported, not fatal, unless `StrictDataLoss` is set. Start from
`DefaultOptions` to change one setting, since a literal leaves every other
option at its zero value:

```go
opts := converter.DefaultOptions()
opts.StrictDataLoss = true
converted, report, err = converter.ConvertWithOptions(doc, gedcom.Version551, opts)
if errors.Is(err, converter.ErrDataLoss) {
    // refused; nothing was converted
}
```

The [converter guide](docs/guides/converter.md) lists what each conversion
changes and what a downgrade loses.

## Combining and Merging

The `merge` package does the bookkeeping of combining documents and folding
records together. Deciding which records describe the same person stays with
the caller. Every function returns a new document and leaves its inputs alone.

### Combining Two Documents

```go
import "github.com/cacack/gedcom-go/v3/merge"

combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
    CollisionStrategy: merge.RenumberDoc2,
})
if errors.Is(err, merge.ErrIncompatibleHeader) {
    // the headers declare different versions or encodings
}
if err != nil {
    log.Fatal(err)
}

for old, renamed := range report.RemappedXRefs {
    fmt.Printf("%s -> %s\n", old, renamed)
}
for _, c := range report.HeaderConflicts {
    fmt.Printf("%s: kept %q, dropped %q\n", c.Field, c.Doc1, c.Doc2)
}
```

The collision strategy decides what happens to an XRef both documents use:

- `merge.ErrorOnCollision` (the zero value): refuse with `merge.ErrXRefCollision`
- `merge.PrefixDoc2`: prepend `CombineOptions.Prefix` to doc2's colliding XRefs
- `merge.RenumberDoc2`: give doc2's colliding records fresh XRefs
- `merge.ZipSharedXRefs`: treat a shared XRef as one record (see [below](#cutting-and-rejoining-a-branch))

### Merging Two Records

`merge.Individuals` and `merge.Families` fold one record into another that
the caller names as the survivor:

```go
// Fold @I7@ into @I3@: its substructures move, every pointer to it is repointed.
merged, report, err := merge.Individuals(doc, "@I3@", "@I7@", merge.RecordMergeOptions{})
if err != nil {
    log.Fatal(err) // wraps merge.ErrInvalidMerge for a missing or non-individual record
}
fmt.Printf("moved %d, already present %d, repointed %d\n",
    len(report.Moved), len(report.Deduplicated), len(report.Repointed))

// Merging two spouses' duplicates often leaves two families for one couple.
merged, _, err = merge.Families(merged, "@F1@", "@F4@", merge.RecordMergeOptions{})
```

No field-level policy is applied: a conflicting `SEX` is kept alongside the
survivor's for the validator to report. The merge reads the raw `Tags`, so
call `encoder.SyncTags` first if typed fields were edited.

### Cutting and Rejoining a Branch

`Document.BoundedSubset` cuts out a set of individuals with their families and
everything those reference, removing pointers that lead outside the set.
Records keep their XRefs, so `merge.ZipSharedXRefs` joins two such subsets
back together:

```go
branchA, _, err := doc.BoundedSubset([]string{"@I1@", "@I2@"})
if err != nil {
    log.Fatal(err) // wraps gedcom.ErrUnknownXRef or gedcom.ErrNotIndividual
}
branchB, droppedB, err := doc.BoundedSubset([]string{"@I3@"})
if err != nil {
    log.Fatal(err)
}
for _, p := range droppedB {
    fmt.Printf("%s: %s %s removed (restorable: %t)\n", p.XRef, p.Tag, p.Target, p.Restorable)
}

// A shared XRef is one record: identical copies collapse, split families are unioned.
rejoined, _, err := merge.Combine(branchA, branchB, merge.CombineOptions{
    CollisionStrategy: merge.ZipSharedXRefs,
    Zip:               &merge.ZipOptions{DroppedFromDoc2: droppedB},
})
if errors.Is(err, merge.ErrRecordConflict) {
    // a shared record differs in more than its individual and family pointers
}
```

Passing doc2's dropped-pointer report restores the original order of a split
family's pointer lines; it is optional. See
[Bounded Subset](FEATURES.md#bounded-subset) and
[Merge Primitives](FEATURES.md#merge-primitives) in FEATURES.md for the exact
rules, and [examples/merge](examples/merge/) for a runnable program.

## Comparing Documents

`gedcom.Diff` compares two documents record by record, matching records by
XRef and type. It reports differences in meaning only: reordering among
siblings with different tags, a value split differently across `CONT`/`CONC`
lines, and line numbers are not changes.

```go
d := gedcom.Diff(oldDoc, newDoc, &gedcom.DiffOptions{
    IgnoreTags: []string{"CHAN"}, // skipped with everything under them
})
if d.IsEmpty() {
    fmt.Println("No changes")
}
for _, r := range d.Added {
    fmt.Printf("+ %s %s\n", r.XRef, r.Type)
}
for _, r := range d.Removed {
    fmt.Printf("- %s %s\n", r.XRef, r.Type)
}
for _, m := range d.Modified {
    fmt.Printf("~ %s %s\n", m.XRef, m.Type)
    for _, c := range m.Changes {
        for _, t := range c.Removed {
            fmt.Printf("  - %d %s %s\n", t.Level, t.Tag, t.Value)
        }
        for _, t := range c.Added {
            fmt.Printf("  + %d %s %s\n", t.Level, t.Tag, t.Value)
        }
    }
}
```

A nil `*DiffOptions` compares every tag. The header is compared only with
`IncludeHeader`, since it changes on every export. Records are compared by
their raw `Tags`, so call `encoder.SyncTags` first if typed fields were
edited; see [Record Diff](FEATURES.md#record-diff) for the details.

## GEDZIP Archives

A GEDZIP file (`.gdz`) is the GEDCOM 7 packaging format: a zip archive holding the dataset in an entry named `gedcom.ged`, plus the local files its `FILE` and `FILE.TRAN` lines name. The `gedzip` package reads and writes them. The archive stays separate from the `Document`: decoding gives an ordinary document, and each media file is read from the archive only when asked for.

### Reading an Archive

```go
import "github.com/cacack/gedcom-go/v3/gedzip"

zr, err := gedzip.OpenReader("family.gdz", nil) // nil: default limits
if err != nil {
    log.Fatal(err)
}
defer zr.Close()

doc, err := zr.Decode(nil) // nil: the decoder's defaults
if err != nil {
    log.Fatal(err)
}

for _, media := range doc.MediaObjects() {
    for _, file := range media.Files {
        // Open takes the FILE payload exactly as the document has it.
        rc, err := zr.Open(file.FileRef)
        if errors.Is(err, gedzip.ErrNotLocal) {
            continue // a web URL (or a file: URL, or an invalid path) has no entry
        }
        if err != nil {
            log.Fatal(err)
        }
        n, err := io.Copy(io.Discard, rc)
        rc.Close()
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("%s: %d bytes\n", file.FileRef, n)
    }
}
```

`gedzip.NewReader(r, size, opts)` does the same over any `io.ReaderAt`, and `Reader.DecodeWithDiagnostics` returns the decoder's diagnostics as well.

A local `FILE` payload names its entry by percent-decoding, and the name is matched byte for byte: `media/CharlotteBront%C3%AB.jpg` opens the entry `media/CharlotteBrontë.jpg`. There is no case folding and no fallback.

### Checking References

`CheckRefs` lists the references an archive does not satisfy, without reading any entry:

```go
for _, p := range zr.CheckRefs(doc) {
    switch {
    case errors.Is(p.Err, gedzip.ErrEntryNotFound):
        fmt.Printf("%s: %s is not in the archive\n", p.XRef, p.FileRef)
    case errors.Is(p.Err, gedzip.ErrUnusableEntry):
        fmt.Printf("%s: %s cannot be read safely: %v\n", p.XRef, p.FileRef, p.Err)
    case errors.Is(p.Err, gedzip.ErrReservedName):
        fmt.Printf("%s: %s names the archive's own gedcom.ged\n", p.XRef, p.FileRef)
    default: // gedzip.ErrNotLocal: a file: URL or an invalid path
        fmt.Printf("%s: %s is %s\n", p.XRef, p.FileRef, p.Kind)
    }
}
```

### Writing an Archive

`Write` takes a GEDCOM 7 document and an `fs.FS` holding the files its local references name:

```go
out, err := os.Create("family.gdz")
if err != nil {
    log.Fatal(err)
}
defer out.Close()

// A FILE payload of media/photo.jpg is read from family-media/media/photo.jpg.
if err := gedzip.Write(out, doc, os.DirFS("family-media"), nil); err != nil {
    log.Fatal(err)
}
```

`Reader.FS()` is such a source too, so an archive can be read, changed and written back:

```go
err = gedzip.Write(out, doc, zr.FS(), nil)
```

`Write` is strict. It refuses a document that would not be written as GEDCOM 7 (`gedzip.ErrNotGEDCOM7`; convert it with the `converter` package first), a `file:` URL or invalid path (`gedzip.ErrNotLocal`), a reference to `gedcom.ged` itself (`gedzip.ErrReservedName`) and a local reference with no file in the source (`gedzip.ErrMissingMedia`), all before writing anything.

### Archives from Untrusted Sources

Reads are bounded by `gedzip.ReadOptions`, and the defaults apply when it is nil. Lower them for uploads:

```go
zr, err := gedzip.NewReader(upload, size, &gedzip.ReadOptions{
    MaxDocumentBytes: 16 << 20, // gedcom.ged, uncompressed
    MaxEntryBytes:    64 << 20, // each media file
    MaxTotalBytes:    512 << 20,
})
if errors.Is(err, gedzip.ErrLimitExceeded) {
    // refused before anything was inflated
}
```

See the [GEDZIP guide](docs/guides/gedzip.md) for every limit, which entries are unreachable, and what is out of scope.

## Character Encoding

### Supported Encodings

The library supports multiple character encodings:

- UTF-8 (recommended for new files; `UTF8` is accepted as a spelling)
- ANSEL (legacy GEDCOM standard)
- ASCII
- LATIN1, ANSI, ISO-8859-1, WINDOWS-1252, CP1252 or WIN-1252 (decoded as Windows-1252)
- Windows-1250 (Central European), for `CHAR ANSI` files written over it; no `CHAR` value declares it, so name it with `DecodeOptions.ANSIEncoding`
- IBMPC, IBM PC, CP437 or IBM-437 (IBM code page 437)
- CP850 or IBM-850 (IBM code page 850)
- MACINTOSH, MAC or MACROMAN (Mac OS Roman)
- UTF-16 LE/BE, with or without a BOM (often declared `UNICODE`; `UTF-16`, `UTF-16LE` and `UTF-16BE` are accepted too)

`CHAR` values are matched without regard to case. To read a `CHAR ANSI` file
as Windows-1250:

```go
opts := &decoder.DecodeOptions{ANSIEncoding: charset.EncodingWindows1250}
doc, err := decoder.DecodeWithOptions(f, opts)
```

Left unset, `ANSI` is read as Windows-1252, and a lenient decode reports one
`AMBIGUOUS_CODEPAGE` warning if the file holds a byte the two code pages read
differently. See [Character Encoding](FEATURES.md#character-encoding) in
FEATURES.md.

### Automatic Encoding Detection

The decoder automatically detects and handles character encoding:

```go
// The decoder reads the CHAR tag from the header
// and automatically converts to UTF-8 internally
doc, err := decoder.Decode(f)

// All strings in the document are UTF-8
fmt.Println(doc.Header.Encoding)  // Original encoding
```

### Working with ANSEL

ANSEL is a legacy character encoding used in older GEDCOM files:

```go
import (
    "bytes"
    "fmt"
    "io"

    "github.com/cacack/gedcom-go/v3/charset"
)

// The charset package handles ANSEL decoding automatically
// when used through the decoder

// For manual ANSEL handling:
anselText := []byte{0xE1, 0x41}  // ANSEL for "À" (combining grave precedes the base letter)
r := charset.NewReaderWithEncoding(bytes.NewReader(anselText), charset.EncodingANSEL)
utf8Text, _ := io.ReadAll(r)
fmt.Println(string(utf8Text))  // "À", decomposed as "A\u0300" (no NFC), so it is != "\u00C0"
```

### Validating UTF-8

```go
import "github.com/cacack/gedcom-go/v3/charset"

text := "Hello, world!"
if !charset.ValidateString(text) {
    fmt.Println("Invalid UTF-8")
}
```

## Error Handling

### Decoder Errors

```go
doc, err := decoder.Decode(f)
if err != nil {
    // Check for specific error types
    var parseErr *parser.ParseError
    switch {
    case errors.As(err, &parseErr): // strict mode, or no parseable line at all
        fmt.Printf("Syntax error on line %d: %v\n", parseErr.Line, parseErr)
    case errors.Is(err, context.DeadlineExceeded):
        fmt.Println("Parsing timeout")
    default:
        fmt.Printf("Decode error: %v\n", err)
    }
}
```

### Parser Errors

For more detailed error information, use the parser directly:

```go
import (
    "errors"
    "fmt"

    "github.com/cacack/gedcom-go/v3/parser"
)

p := parser.NewParser()
lines, err := p.Parse(f)
if err != nil {
    var parseErr *parser.ParseError
    if errors.As(err, &parseErr) {
        fmt.Printf("Parse error at line %d: %s (%q)\n",
            parseErr.Line, parseErr.Message, parseErr.Context)
    }
    return
}
fmt.Printf("Parsed %d lines\n", len(lines))
```

### Validation Errors

See [Validation](#validation) section above for handling validation errors.

## Advanced Usage

### Using Context for Cancellation

```go
import "context"

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Start parsing in a goroutine
resultChan := make(chan *gedcom.Document, 1)
errorChan := make(chan error, 1)

go func() {
    opts := &decoder.DecodeOptions{Context: ctx}
    doc, err := decoder.DecodeWithOptions(f, opts)
    if err != nil {
        errorChan <- err
        return
    }
    resultChan <- doc
}()

// Cancel if it takes too long
select {
case doc := <-resultChan:
    fmt.Printf("Parsing complete: %d records\n", len(doc.Records))
case err := <-errorChan:
    fmt.Printf("Error: %v\n", err)
case <-time.After(10 * time.Second):
    cancel()
    fmt.Println("Parsing cancelled due to timeout")
}
```

### Streaming Large Files

For very large GEDCOM files, use the lower-level parser to avoid loading everything into memory:

```go
import (
    "github.com/cacack/gedcom-go/v3/charset"
    "github.com/cacack/gedcom-go/v3/parser"
)

f, _ := os.Open("large.ged")
defer f.Close()

individualCount := 0

// Records yields one level-0 record (with its subordinate lines) at a time
for record, err := range parser.Records(charset.NewReader(f)) {
    if err != nil {
        log.Fatal(err)
    }
    if record.Type == "INDI" {
        individualCount++
    }
}

fmt.Printf("Found %d individuals\n", individualCount)
```

### Indexed Access with LazyParser

`parser.LazyParser` indexes each record's byte offset in one pass, then parses
only the records asked for. `decoder.DecodeRecord` turns a raw record into the
typed record `Decode` would build.

```go
lp := parser.NewLazyParser(f) // f must be an io.ReadSeeker
if err := lp.BuildIndex(); err != nil {
    log.Fatal(err)
}
fmt.Printf("%d records indexed\n", lp.RecordCount())

// The header declares the version DecodeRecord reads records by.
head, err := lp.FindRecordByType("HEAD")
if err != nil {
    log.Fatal(err)
}
ver := version.DeclaredVersion(head.Lines)

raw, err := lp.FindRecord("@I1@") // parses this record's lines only
if errors.Is(err, parser.ErrRecordNotFound) {
    return
}
if err != nil {
    log.Fatal(err)
}
rec, err := decoder.DecodeRecord(raw, ver, nil)
if err != nil {
    log.Fatal(err)
}
if person, ok := rec.GetIndividual(); ok {
    fmt.Println(person.FullName())
}
```

`SaveIndex` and `LoadIndex` keep the index between runs. `LazyParser` reads
the file's bytes as they are, with no charset conversion, so it suits UTF-8
files; decode an ANSEL or UTF-16 file with the `decoder` package instead.

### Streaming Encoding

`encoder.StreamEncoder` writes a file one record at a time, without a
`Document`. The calls must come in order: `WriteHeader`, `WriteRecord` any
number of times, `WriteTrailer`, `Close`.

```go
func writeStreamed(w io.Writer, header *gedcom.Header, records []*gedcom.Record) (err error) {
    enc := encoder.NewStreamEncoder(w)
    defer func() {
        // Close flushes, and fails if the trailer was never written.
        if cerr := enc.Close(); cerr != nil && err == nil {
            err = cerr
        }
    }()

    if err := enc.WriteHeader(header); err != nil {
        return err
    }
    for _, rec := range records {
        if err := enc.WriteRecord(rec); err != nil {
            return err
        }
    }
    return enc.WriteTrailer()
}
```

A bare `defer enc.Close()` would discard that error.
`encoder.NewStreamEncoderWithOptions` takes the same `EncodeOptions` as
`EncodeWithOptions`.

### Streaming Validation

`validator.StreamingValidator` validates records as they are read and defers
cross-reference checks to `Finalize`. It needs typed records, so decode each
raw record first:

```go
var sv *validator.StreamingValidator
var ver gedcom.Version
var issues []validator.Issue

for raw, err := range parser.Records(charset.NewReader(f)) {
    if err != nil {
        log.Fatal(err)
    }
    if raw.Type == "HEAD" { // comes first and declares the version
        ver = version.DeclaredVersion(raw.Lines)
    }
    if sv == nil {
        sv = validator.NewStreamingValidator(validator.StreamingOptions{Version: ver})
    }
    record, err := decoder.DecodeRecord(raw, ver, nil)
    if err != nil {
        log.Fatal(err)
    }
    issues = append(issues, sv.ValidateRecord(record)...)
}
if sv != nil {
    issues = append(issues, sv.Finalize()...) // pointers to records never declared
}
```

Only individuals and families are checked in full, and checks that need the
whole document (duplicate detection, parent-child date logic) do not run; use
`ValidateAll` when the document fits in memory. See
[Streaming Validator](FEATURES.md#streaming-validator) in FEATURES.md and
[When to Use Streaming APIs](docs/guides/performance.md#when-to-use-streaming-apis).

### Custom Record Processing

```go
// Process only specific record types
for _, record := range doc.Records {
    switch record.Type {
    case gedcom.RecordTypeIndividual:
        processIndividual(record)
    case gedcom.RecordTypeFamily:
        processFamily(record)
    case gedcom.RecordTypeSource:
        processSource(record)
    }
}
```

### Building a Family Tree

```go
// Build a map of parent-child relationships
type Person struct {
    Individual *gedcom.Individual
    Parents    []*Person
    Children   []*Person
}

people := make(map[string]*Person)

// First pass: create all people
for _, ind := range doc.Individuals() {
    people[ind.XRef] = &Person{
        Individual: ind,
        Parents:    []*Person{},
        Children:   []*Person{},
    }
}

// Second pass: build relationships
for _, family := range doc.Families() {
    var father, mother *Person

    if family.Husband != "" {
        father = people[family.Husband]
    }
    if family.Wife != "" {
        mother = people[family.Wife]
    }

    for _, childXRef := range family.Children {
        child := people[childXRef]
        if child != nil {
            if father != nil {
                child.Parents = append(child.Parents, father)
                father.Children = append(father.Children, child)
            }
            if mother != nil {
                child.Parents = append(child.Parents, mother)
                mother.Children = append(mother.Children, child)
            }
        }
    }
}

// Now you can traverse the tree
person := people["@I1@"]
fmt.Printf("%s has %d children\n",
    person.Individual.FullName(),
    len(person.Children))
```

## Performance

Benchmark figures, memory budgets and tuning advice live in one place,
[docs/guides/performance.md](docs/guides/performance.md), measured on a stated
environment; run `make bench` to reproduce them on your own hardware.

`decoder.Decode` builds the whole `Document` in memory. For files too large for
that, use the record-at-a-time APIs shown in
[Streaming Large Files](#streaming-large-files), and see
[When to Use Streaming APIs](docs/guides/performance.md#when-to-use-streaming-apis).

## Best Practices

### 1. Always Close Files

```go
f, err := os.Open("family.ged")
if err != nil {
    log.Fatal(err)
}
defer f.Close()  // Always use defer to ensure cleanup
```

### 2. Prefer the nil-safe Accessors

```go
person := doc.GetIndividual("@I1@")
fmt.Println(person.FullName()) // nil-safe: "" for a missing person or name
```

### 3. Validate After Creation

When creating GEDCOM files programmatically, always validate:

```go
doc := createDocument()

v := validator.New()
for _, issue := range v.Validate(doc) {
    // Validate also returns warnings (e.g. MISSING_SUBM); fail on errors only.
    if issue.Severity == validator.SeverityError {
        log.Fatalf("Created invalid GEDCOM: %v", issue)
    }
}

encoder.Encode(f, doc)
```

### 4. Use Appropriate Line Endings

```go
// LF is the default; set CRLF only when a consumer requires it
opts := &encoder.EncodeOptions{
    LineEnding: "\r\n",
}
encoder.EncodeWithOptions(f, doc, opts)
```

### 5. Handle Multiple GEDCOM Versions

```go
// Check version before processing
switch doc.Header.Version {
case "5.5":
    // Handle GEDCOM 5.5 specifics
case "5.5.1":
    // Handle GEDCOM 5.5.1 specifics
case "7.0":
    // Handle GEDCOM 7.0 specifics
default:
    log.Printf("Unknown GEDCOM version: %s", doc.Header.Version)
}
```

### 6. Use Encoding Detection

```go
// Let the decoder handle encoding automatically
doc, err := decoder.Decode(f)

// All strings in doc are now UTF-8, regardless of source encoding
```

### 7. Process Errors Gracefully

```go
// Collect errors instead of failing fast
var validationErrors []error
for _, individual := range doc.Individuals() {
    if len(individual.Names) == 0 {
        validationErrors = append(validationErrors,
            fmt.Errorf("individual %s has no name", individual.XRef))
    }
}

if len(validationErrors) > 0 {
    // Handle all errors together
    for _, err := range validationErrors {
        log.Println(err)
    }
}
```

### 8. Use the XRefMap for Lookups

```go
// Fast lookup by cross-reference
record := doc.XRefMap["@I1@"]

// Or use the convenience methods
individual := doc.GetIndividual("@I1@")
family := doc.GetFamily("@F1@")
```

## See Also

- [Examples](examples/) - Working code examples
- [FEATURES.md](FEATURES.md) - Every implemented feature
- [Decoding contract](docs/guides/decoding.md) - Strict and lenient modes, diagnostic codes, round-trip expectations
- [Converter guide](docs/guides/converter.md) - What each version conversion changes
- [GEDZIP guide](docs/guides/gedzip.md) - Reading and writing `.gdz` archives
- [API Documentation](https://pkg.go.dev/github.com/cacack/gedcom-go/v3) - Complete package documentation
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute to the project
- [README.md](README.md) - Project overview
