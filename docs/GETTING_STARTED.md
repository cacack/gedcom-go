# Getting Started with gedcom-go

This guide will help you get started with using the gedcom-go library to parse, validate, and create GEDCOM genealogy files in your Go applications.

## Table of Contents

1. [Installation](#installation)
2. [Basic Concepts](#basic-concepts)
3. [Parsing GEDCOM Files](#parsing-gedcom-files)
4. [Working with Parsed Data](#working-with-parsed-data)
5. [Validating GEDCOM Files](#validating-gedcom-files)
6. [Creating GEDCOM Files](#creating-gedcom-files)
7. [Error Handling](#error-handling)
8. [Performance Considerations](#performance-considerations)

## Installation

```bash
go get github.com/cacack/gedcom-go@latest
```

**Requirements**: Go 1.21 or later

## Basic Concepts

### GEDCOM Structure

GEDCOM (GEnealogical Data COMmunication) is a hierarchical file format for exchanging genealogical data. The library supports three major versions:

- **GEDCOM 5.5** - Most widely used version
- **GEDCOM 5.5.1** - Enhanced with additional tags (EMAIL, WWW, MAP, etc.)
- **GEDCOM 7.0** - Modern version with new structure and tags

### Key Data Types

- **`Document`** - The root container for all GEDCOM data
- **`Header`** - Metadata about the GEDCOM file (version, encoding, source)
- **`Record`** - Top-level records (individuals, families, sources, etc.)
- **`Individual`** - Person records with names, events, and relationships
- **`Family`** - Family units linking spouses and children
- **`Tag`** - The hierarchical data structure within records

## Parsing GEDCOM Files

### Simple Parsing

The easiest way to parse a GEDCOM file:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/cacack/gedcom-go/decoder"
)

func main() {
    f, err := os.Open("family.ged")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    doc, err := decoder.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Successfully parsed GEDCOM %s file\n", doc.Header.Version)
    fmt.Printf("Contains %d individuals\n", len(doc.Individuals()))
}
```

### Parsing with Options

For more control over the parsing process:

```go
import "github.com/cacack/gedcom-go/decoder"

options := &decoder.Options{
    MaxNestingDepth: 100,  // Maximum tag nesting depth
    StrictMode:      true, // Strict validation during parsing
}

doc, err := decoder.DecodeWithOptions(f, options)
```

### Automatic Version Detection

The decoder automatically detects the GEDCOM version from the header. You can also access version information:

```go
import "github.com/cacack/gedcom-go/version"

// Version is automatically detected
fmt.Printf("Detected version: %s\n", doc.Header.Version)

// You can also detect version before full parsing
lines, _ := parser.Parse(f)
detectedVersion, _ := version.DetectVersion(lines)
```

## Working with Parsed Data

### Accessing Individuals

```go
// Get all individuals
individuals := doc.Individuals()

for _, ind := range individuals {
    // Access names
    if len(ind.Names) > 0 {
        name := ind.Names[0]
        fmt.Printf("Name: %s %s\n", name.Given, name.Surname)
        fmt.Printf("Full: %s\n", name.Full)
    }

    // Access birth event
    for _, event := range ind.Events {
        if event.Tag == "BIRT" {
            fmt.Printf("Born: %s at %s\n", event.Date, event.Place)
        }
    }

    // Access sex
    fmt.Printf("Sex: %s\n", ind.Sex)
}
```

### Finding Specific Individuals

```go
// Find individual by XRef
individual := doc.GetIndividual("@I1@")
if individual != nil {
    fmt.Printf("Found: %s\n", individual.Names[0].Full)
}

// Search by name (manual iteration)
for _, ind := range doc.Individuals() {
    if len(ind.Names) > 0 && strings.Contains(ind.Names[0].Full, "Smith") {
        fmt.Printf("Found Smith: %s\n", ind.Names[0].Full)
    }
}
```

### Working with Families

```go
// Get all families
families := doc.Families()

for _, fam := range families {
    // Access family members
    if fam.Husband != "" {
        husband := doc.GetIndividual(fam.Husband)
        if husband != nil && len(husband.Names) > 0 {
            fmt.Printf("Husband: %s\n", husband.Names[0].Full)
        }
    }

    if fam.Wife != "" {
        wife := doc.GetIndividual(fam.Wife)
        if wife != nil && len(wife.Names) > 0 {
            fmt.Printf("Wife: %s\n", wife.Names[0].Full)
        }
    }

    // Access children
    for _, childRef := range fam.Children {
        child := doc.GetIndividual(childRef)
        if child != nil && len(child.Names) > 0 {
            fmt.Printf("Child: %s\n", child.Names[0].Full)
        }
    }

    // Access marriage event
    for _, event := range fam.Events {
        if event.Tag == "MARR" {
            fmt.Printf("Married: %s\n", event.Date)
        }
    }
}
```

### Accessing Sources

```go
sources := doc.Sources()

for _, src := range sources {
    fmt.Printf("Source: %s\n", src.Title)
    fmt.Printf("Author: %s\n", src.Author)

    if src.PublicationFacts != "" {
        fmt.Printf("Publication: %s\n", src.PublicationFacts)
    }
}
```

## Validating GEDCOM Files

The validator checks for common issues in GEDCOM files:

```go
import "github.com/cacack/gedcom-go/validator"

// Create validator
v := validator.New(doc)

// Validate the document
errors := v.Validate()

if len(errors) == 0 {
    fmt.Println("✓ GEDCOM file is valid")
} else {
    fmt.Printf("Found %d validation errors:\n", len(errors))

    for _, err := range errors {
        // Type assertion to get validation details
        if verr, ok := err.(*validator.ValidationError); ok {
            fmt.Printf("[%s] Line %d: %s\n", verr.Code, verr.Line, verr.Message)
        } else {
            fmt.Printf("Error: %s\n", err.Error())
        }
    }
}
```

Common validation errors:
- **BROKEN_XREF** - Reference to non-existent record
- **MISSING_REQUIRED_FIELD** - Required field missing (e.g., NAME in Individual)
- **INVALID_VALUE** - Invalid value for a field

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
        SourceSystem: "My App",
    },
    Records: []*gedcom.Record{
        {
            XRef: "@I1@",
            Type: gedcom.RecordTypeIndividual,
            Tags: []*gedcom.Tag{
                {Level: 1, Tag: "NAME", Value: "John /Doe/"},
                {Level: 1, Tag: "SEX", Value: "M"},
                {
                    Level: 1,
                    Tag:   "BIRT",
                    Children: []*gedcom.Tag{
                        {Level: 2, Tag: "DATE", Value: "1 JAN 1980"},
                        {Level: 2, Tag: "PLAC", Value: "New York, NY"},
                    },
                },
            },
            Entity: &gedcom.Individual{
                XRef: "@I1@",
                Names: []*gedcom.PersonalName{
                    {Full: "John /Doe/", Given: "John", Surname: "Doe"},
                },
                Sex: "M",
            },
        },
    },
}

// Write to file
f, err := os.Create("output.ged")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

if err := encoder.Encode(f, doc); err != nil {
    log.Fatal(err)
}
```

### Encoding Options

```go
options := &encoder.Options{
    LineEnding: "\r\n", // Windows line endings (CRLF)
}

encoder.EncodeWithOptions(f, doc, options)
```

## Error Handling

### Parser Errors

Parser errors include line numbers and context:

```go
doc, err := decoder.Decode(f)
if err != nil {
    // Error includes line number and context
    fmt.Printf("Parse error: %v\n", err)
    // Example output: "line 15: invalid level number (context: 'INVALID LINE')"
}
```

### Handling Invalid UTF-8

```go
import "github.com/cacack/gedcom-go/charset"

// The charset package handles UTF-8 validation automatically
// Invalid UTF-8 will return an error with line/column information
_, err := decoder.Decode(f)
if err != nil {
    if utfErr, ok := err.(*charset.ErrInvalidUTF8); ok {
        fmt.Printf("Invalid UTF-8 at line %d, column %d\n", utfErr.Line, utfErr.Column)
    }
}
```

## Performance Considerations

### Benchmarks

Performance characteristics on Apple M2 (from `decoder/benchmark_test.go`):

| File Size | Time/Operation | Memory | Allocations |
|-----------|----------------|--------|-------------|
| ~170B (minimal) | 7 µs | 7 KB | 69 |
| ~15KB (small) | 381 µs | 214 KB | 3,530 |
| ~458KB (medium) | 17 ms | 8.5 MB | 139K |
| ~1.1MB (large) | 32 ms | 14 MB | 214K |

**Throughput**: ~32 MB/s for large files

### Best Practices

1. **Stream Large Files**: The decoder uses stream-based parsing, so memory usage is proportional to file size (~1.2x)

2. **Reuse Validators**: Create one validator instance and reuse it:
   ```go
   v := validator.New(doc)
   errors := v.Validate()
   ```

3. **Check for Errors Early**: Validate incrementally if processing many files:
   ```go
   doc, err := decoder.Decode(f)
   if err != nil {
       // Handle parse errors before validation
       return err
   }
   ```

4. **Use Appropriate Buffer Sizes**: For network streams, use `bufio.Reader`:
   ```go
   import "bufio"

   reader := bufio.NewReaderSize(networkStream, 64*1024)
   doc, err := decoder.Decode(reader)
   ```

## Next Steps

- **Explore Examples**: Check out the [`examples/`](../examples/) directory for complete working examples
- **Read Package Documentation**: Visit [pkg.go.dev/github.com/cacack/gedcom-go](https://pkg.go.dev/github.com/cacack/gedcom-go) for detailed API documentation
- **Contribute**: See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on contributing to the project

## Need Help?

- **Issues**: Report bugs or request features at [GitHub Issues](https://github.com/cacack/gedcom-go/issues)
- **Discussions**: Ask questions in [GitHub Discussions](https://github.com/cacack/gedcom-go/discussions)
- **Documentation**: Full API documentation at [pkg.go.dev](https://pkg.go.dev/github.com/cacack/gedcom-go)
