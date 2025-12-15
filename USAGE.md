# Usage Guide

This guide provides comprehensive documentation for using the `gedcom-go` library to parse, query, validate, and create GEDCOM files in your Go applications.

## Table of Contents

- [Basic Concepts](#basic-concepts)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Parsing GEDCOM Files](#parsing-gedcom-files)
- [Working with Documents](#working-with-documents)
- [Querying Data](#querying-data)
- [Validation](#validation)
- [Creating GEDCOM Files](#creating-gedcom-files)
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
go get github.com/cacack/gedcom-go
```

Requirements:
- Go 1.21 or later
- No external dependencies (uses only the Go standard library)

## Quick Start

Here's a minimal example to get started:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/cacack/gedcom-go/decoder"
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

## Parsing GEDCOM Files

### Basic Parsing

The `decoder` package provides the high-level API for parsing GEDCOM files:

```go
import "github.com/cacack/gedcom-go/decoder"

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
    "github.com/cacack/gedcom-go/decoder"
)

opts := &decoder.DecodeOptions{
    // Set a timeout for parsing large files
    Context: func() context.Context {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        // Don't forget to cancel when done
        defer cancel()
        return ctx
    }(),

    // Strict mode - fail on any validation errors
    Strict: false,

    // Maximum line length (default: 255 for GEDCOM 5.5/5.5.1, unlimited for 7.0)
    MaxLineLength: 255,
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

## Working with Documents

### Document Structure

A `Document` contains:

```go
type Document struct {
    Header  *Header      // File header with version, encoding, etc.
    Records []*Record    // All records in the file
    Trailer *Trailer     // File trailer
    XRefMap map[string]*Record  // Cross-reference lookup map
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
    if len(person.Names) > 0 {
        name := person.Names[0]
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
        if event.Place != "" {
            fmt.Printf(" at %s", event.Place)
        }
        fmt.Println()
    }
}

// Look up specific individual
person := doc.GetIndividual("@I1@")
if person != nil {
    fmt.Printf("Found: %s\n", person.Names[0].Full)
}
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
        if event.Place != "" {
            fmt.Printf("  Place: %s\n", event.Place)
        }

        // Additional details
        if event.Description != "" {
            fmt.Printf("  Description: %s\n", event.Description)
        }

        // Source citations
        for _, citation := range event.Citations {
            fmt.Printf("  Source: %s\n", citation)
        }
    }
}

// Find specific event type
for _, person := range doc.Individuals() {
    for _, event := range person.Events {
        if event.Type == "BIRT" {
            fmt.Printf("%s was born on %s\n", person.Names[0].Full, event.Date)
        }
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
        if husband != nil && len(husband.Names) > 0 {
            fmt.Printf("  Husband: %s\n", husband.Names[0].Full)
        }
    }

    // Get wife
    if family.Wife != "" {
        wife := doc.GetIndividual(family.Wife)
        if wife != nil && len(wife.Names) > 0 {
            fmt.Printf("  Wife: %s\n", wife.Names[0].Full)
        }
    }

    // Get children
    fmt.Printf("  Children: %d\n", len(family.Children))
    for _, childXRef := range family.Children {
        child := doc.GetIndividual(childXRef)
        if child != nil && len(child.Names) > 0 {
            fmt.Printf("    - %s\n", child.Names[0].Full)
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
    for _, famXRef := range person.SpouseInFamilies {
        family := doc.GetFamily(famXRef)
        fmt.Printf("Spouse in family: %s\n", famXRef)
    }

    // Families where this person is a child
    for _, famXRef := range person.ChildInFamilies {
        family := doc.GetFamily(famXRef)
        fmt.Printf("Child in family: %s\n", famXRef)
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

    if source.Publisher != "" {
        fmt.Printf("  Publisher: %s\n", source.Publisher)
    }

    if source.RepositoryRef != "" {
        fmt.Printf("  Repository: %s\n", source.RepositoryRef)
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
        fmt.Printf("  Address: %s\n", repo.Address.FullAddress)
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

### Basic Validation

```go
import "github.com/cacack/gedcom-go/validator"

// Create validator
v := validator.New()

// Validate document
errors := v.Validate(doc)

if len(errors) == 0 {
    fmt.Println("✓ Validation passed!")
} else {
    fmt.Printf("Found %d validation errors:\n", len(errors))
    for _, err := range errors {
        fmt.Printf("  - %v\n", err)
    }
}
```

### Working with Validation Errors

Validation errors include detailed information:

```go
for _, err := range errors {
    if verr, ok := err.(*validator.ValidationError); ok {
        fmt.Printf("Code: %s\n", verr.Code)
        fmt.Printf("Message: %s\n", verr.Message)

        if verr.Line > 0 {
            fmt.Printf("Line: %d\n", verr.Line)
        }

        if verr.XRef != "" {
            fmt.Printf("XRef: %s\n", verr.XRef)
        }
    }
}
```

### Grouping Validation Errors

```go
// Group errors by code
errorsByCode := make(map[string][]error)
for _, err := range errors {
    code := "UNKNOWN"
    if verr, ok := err.(*validator.ValidationError); ok {
        code = verr.Code
    }
    errorsByCode[code] = append(errorsByCode[code], err)
}

// Display summary
for code, errs := range errorsByCode {
    fmt.Printf("%s: %d occurrences\n", code, len(errs))
}
```

## Creating GEDCOM Files

### Creating a Simple Document

```go
import (
    "github.com/cacack/gedcom-go/encoder"
    "github.com/cacack/gedcom-go/gedcom"
)

// Create document
doc := &gedcom.Document{
    Header: &gedcom.Header{
        Version:      "5.5",
        Encoding:     "UTF-8",
        SourceSystem: "MyApp",
        Language:     "English",
    },
    Records:  []*gedcom.Record{},
    XRefMap:  make(map[string]*gedcom.Record),
}

// Add individuals, families, etc. (see below)

// Write to file
f, _ := os.Create("output.ged")
defer f.Close()
encoder.Encode(f, doc)
```

### Adding Individuals

```go
// Create individual record
individual := &gedcom.Record{
    XRef: "@I1@",
    Type: gedcom.RecordTypeIndividual,
    Tags: []*gedcom.Tag{
        {Level: 1, Tag: "NAME", Value: "John /Doe/"},
        {Level: 2, Tag: "GIVN", Value: "John"},
        {Level: 2, Tag: "SURN", Value: "Doe"},
        {Level: 1, Tag: "SEX", Value: "M"},
        {Level: 1, Tag: "BIRT"},
        {Level: 2, Tag: "DATE", Value: "1 JAN 1900"},
        {Level: 2, Tag: "PLAC", Value: "New York, USA"},
    },
    Entity: &gedcom.Individual{
        XRef: "@I1@",
        Names: []*gedcom.PersonalName{
            {Full: "John /Doe/", Given: "John", Surname: "Doe"},
        },
        Sex: "M",
        Events: []*gedcom.Event{
            {Type: "BIRT", Date: "1 JAN 1900", Place: "New York, USA"},
        },
    },
}

// Add to document
doc.Records = append(doc.Records, individual)
doc.XRefMap[individual.XRef] = individual
```

### Adding Families

```go
family := &gedcom.Record{
    XRef: "@F1@",
    Type: gedcom.RecordTypeFamily,
    Tags: []*gedcom.Tag{
        {Level: 1, Tag: "HUSB", Value: "@I1@"},
        {Level: 1, Tag: "WIFE", Value: "@I2@"},
        {Level: 1, Tag: "MARR"},
        {Level: 2, Tag: "DATE", Value: "15 JUN 1925"},
        {Level: 2, Tag: "PLAC", Value: "Boston, Massachusetts, USA"},
    },
    Entity: &gedcom.Family{
        XRef:    "@F1@",
        Husband: "@I1@",
        Wife:    "@I2@",
        Events: []*gedcom.Event{
            {Type: "MARR", Date: "15 JUN 1925", Place: "Boston, Massachusetts, USA"},
        },
    },
}

doc.Records = append(doc.Records, family)
doc.XRefMap[family.XRef] = family
```

### Encoding with Options

```go
opts := &encoder.EncodeOptions{
    // Line ending style
    LineEnding: "\r\n",  // CRLF (Windows/GEDCOM standard)
    // LineEnding: "\n",  // LF (Unix)
}

err := encoder.EncodeWithOptions(f, doc, opts)
if err != nil {
    log.Fatal(err)
}
```

## Character Encoding

### Supported Encodings

The library supports multiple character encodings:

- UTF-8 (recommended for new files)
- ANSEL (legacy GEDCOM standard)
- ASCII
- LATIN1 (ISO-8859-1)
- UNICODE (UTF-16)

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
import "github.com/cacack/gedcom-go/charset"

// The charset package handles ANSEL decoding automatically
// when used through the decoder

// For manual ANSEL handling:
anselText := []byte{0xE0, 0x41}  // ANSEL for "À"
utf8Text := charset.ANSELToUTF8(anselText)
fmt.Println(string(utf8Text))  // "À"
```

### Validating UTF-8

```go
import "github.com/cacack/gedcom-go/charset"

text := "Hello, world!"
if !charset.IsValidUTF8([]byte(text)) {
    fmt.Println("Invalid UTF-8")
}
```

## Error Handling

### Decoder Errors

```go
doc, err := decoder.Decode(f)
if err != nil {
    // Check for specific error types
    switch {
    case errors.Is(err, io.EOF):
        fmt.Println("Unexpected end of file")
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
import "github.com/cacack/gedcom-go/parser"

p := parser.New(f)
for {
    line, err := p.ParseLine()
    if err == io.EOF {
        break
    }
    if err != nil {
        if parseErr, ok := err.(*parser.ParseError); ok {
            fmt.Printf("Parse error at line %d: %s\n",
                parseErr.Line, parseErr.Message)
        }
    }
    // Process line...
}
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
resultChan := make(chan *gedcom.Document)
errorChan := make(chan error)

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
    fmt.Println("Parsing complete")
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
import "github.com/cacack/gedcom-go/parser"

f, _ := os.Open("large.ged")
defer f.Close()

p := parser.New(f)
individualCount := 0

for {
    line, err := p.ParseLine()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }

    // Process lines one at a time
    if line.Level == 0 && line.Tag == "INDI" {
        individualCount++
    }
}

fmt.Printf("Found %d individuals\n", individualCount)
```

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
    person.Individual.Names[0].Full,
    len(person.Children))
```

## Performance

### Benchmarks

Performance characteristics on Apple M2 (from actual benchmarks):

| File Size | Time/Operation | Memory | Allocations | Throughput |
|-----------|----------------|--------|-------------|------------|
| ~170B (minimal) | 7 µs | 7 KB | 69 | - |
| ~15KB (small) | 381 µs | 214 KB | 3,530 | ~40 MB/s |
| ~458KB (medium) | 17 ms | 8.5 MB | 139K | ~27 MB/s |
| ~1.1MB (large) | 32 ms | 14 MB | 214K | ~32 MB/s |

**Key Metrics:**

- **Throughput**: ~30-40 MB/s for typical files
- **Memory overhead**: ~1.2-1.3x file size (due to parsing structures)
- **Scalability**: Linear performance scaling with file size

### Optimization Tips

1. **Stream Processing**: The decoder uses stream-based parsing, so you don't need to load entire files into memory first:
   ```go
   // Good - streams directly
   f, _ := os.Open("large.ged")
   doc, _ := decoder.Decode(f)

   // Avoid - loads entire file first
   data, _ := os.ReadFile("large.ged")
   doc, _ := decoder.Decode(bytes.NewReader(data))
   ```

2. **Reuse Validators**: Create one validator instance per document:
   ```go
   v := validator.New()
   errors := v.Validate(doc)
   ```

3. **Use Buffered Readers**: For network streams or slow I/O, use buffering:
   ```go
   import "bufio"

   reader := bufio.NewReaderSize(networkStream, 64*1024)
   doc, err := decoder.Decode(reader)
   ```

4. **Selective Processing**: If you only need certain record types, filter early:
   ```go
   individualCount := 0
   for _, record := range doc.Records {
       if record.Type == gedcom.RecordTypeIndividual {
           individualCount++
           // Process only individuals
       }
   }
   ```

5. **Batch Operations**: When processing multiple files, consider concurrent processing:
   ```go
   var wg sync.WaitGroup
   for _, filename := range files {
       wg.Add(1)
       go func(fn string) {
           defer wg.Done()
           f, _ := os.Open(fn)
           defer f.Close()
           doc, _ := decoder.Decode(f)
           // Process doc...
       }(filename)
   }
   wg.Wait()
   ```

### Memory Management

The library is designed for efficient memory usage:

- **Streaming parser**: Processes line-by-line, not loading entire file
- **Incremental building**: Constructs document structure as it parses
- **No caching**: Doesn't cache intermediate results (keeps memory footprint low)
- **XRefMap**: Uses Go's efficient map implementation for O(1) lookups

For very large files (>100MB), consider:
- Processing in chunks if possible
- Using the low-level parser directly for custom streaming logic
- Increasing system resources (Go runtime handles memory automatically)

## Best Practices

### 1. Always Close Files

```go
f, err := os.Open("family.ged")
if err != nil {
    log.Fatal(err)
}
defer f.Close()  // Always use defer to ensure cleanup
```

### 2. Check for nil Before Accessing

```go
person := doc.GetIndividual("@I1@")
if person != nil && len(person.Names) > 0 {
    fmt.Println(person.Names[0].Full)
}
```

### 3. Validate After Creation

When creating GEDCOM files programmatically, always validate:

```go
doc := createDocument()

v := validator.New()
errors := v.Validate(doc)
if len(errors) > 0 {
    log.Fatalf("Created invalid GEDCOM: %v", errors)
}

encoder.Encode(f, doc)
```

### 4. Use Appropriate Line Endings

```go
// For GEDCOM files, use CRLF (Windows-style) line endings
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
- [API Documentation](https://pkg.go.dev/github.com/cacack/gedcom-go) - Complete package documentation
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute to the project
- [README.md](README.md) - Project overview
