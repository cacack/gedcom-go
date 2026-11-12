# Examples

This directory contains practical examples demonstrating how to use the `gedcom-go` library. Each example is a standalone Go program that showcases specific features and common use cases.

## Available Examples

### 1. Parse - Basic Parsing and Information Display

**Location**: [`parse/main.go`](parse/main.go)

**What it does**: Demonstrates basic GEDCOM file parsing and displays summary information including record counts, version detection, and validation results.

**How to run**:
```bash
cd examples/parse
go run main.go ../../testdata/gedcom-5.5/minimal.ged
```

**Example output**:
```
GEDCOM File: ../../testdata/gedcom-5.5/minimal.ged
Version: 5.5
Encoding: UTF-8
Source System: MyFamilyTree

Total Records: 15
Cross-references: 8

Record Types:
  INDI: 5
  FAM: 2
  SOUR: 1

✓ No validation errors
```

**Use cases**:
- Quick inspection of GEDCOM files
- Verifying file integrity
- Getting summary statistics
- Learning the basic parsing API

---

### 2. Query - Navigating and Querying Genealogy Data

**Location**: [`query/main.go`](query/main.go)

**What it does**: Shows how to query and navigate GEDCOM data, including:
- Listing all individuals with their names and dates
- Looking up individuals by cross-reference
- Traversing family relationships
- Accessing sources and repositories
- Direct record access using XRef lookups

**How to run**:
```bash
cd examples/query
go run main.go ../../testdata/gedcom-5.5/royal92.ged
```

**Example output**:
```
Querying GEDCOM File: ../../testdata/gedcom-5.5/royal92.ged
Version: 5.5

=== All Individuals ===
Found 62 individuals:
@I1@: Elizabeth Alexandra Mary /Windsor/ (F) - Born: 21 APR 1926
@I2@: Philip /Mountbatten/ (M) - Born: 10 JUN 1921
@I3@: Charles Philip Arthur George /Windsor/ (M) - Born: 14 NOV 1948
...

=== Lookup by Cross-Reference ===
Found individual @I1@:
  Name: Elizabeth Alexandra Mary /Windsor/
  Sex: F
  Events:
    BIRT: 21 APR 1926 at London, England
    OCCU: Queen of England
  Spouse in families: [@F1@]

=== All Families ===
Found 25 families:
@F1@: Elizabeth Alexandra Mary /Windsor/ & Philip /Mountbatten/ (4 children)
@F2@: Charles Philip Arthur George /Windsor/ & Diana Frances /Spencer/ (2 children)
...
```

**Use cases**:
- Building genealogy applications
- Data extraction and analysis
- Generating reports
- Understanding family relationships

---

### 3. Validate - GEDCOM File Validation

**Location**: [`validate/main.go`](validate/main.go)

**What it does**: Demonstrates comprehensive GEDCOM validation with issue categorization and reporting. Shows how to:
- Validate GEDCOM files against specification rules
- Recover the `*validator.Issue` behind each returned error with `errors.As`
- Group validation issues by code
- Display severity, line numbers and context
- Generate validation reports

**How to run**:
```bash
cd examples/validate
go run main.go ../../testdata/gedcom-7.0/minimal70.ged
```

**Example output** (valid file):
```
Validating GEDCOM File: ../../testdata/gedcom-7.0/minimal70.ged
Version: 7.0
Encoding: 

✅ Validation passed!
No issues found.
```

**Example output** (file with a broken reference):
```
Validating GEDCOM File: ../../testdata/malformed/invalid-xref.ged
Version: 5.5
Encoding: 

❌ Validation found 2 issue(s):

Code: MISSING_SUBM (1 occurrence(s))
  - GEDCOM 5.5 requires SUBM reference in header (WARNING)

Code: ORPHANED_FAMC (1 occurrence(s))
  - FAMC reference to non-existent family @F999@ (ERROR, line 6, XRef: @I1@)

=== Summary ===
Total Records: 1
Total Issues: 2
Issue Codes: 2
```

**Use cases**:
- Quality assurance for GEDCOM files
- Pre-import validation
- Data cleaning workflows
- Compliance checking

---

### 4. Stream - Memory-Efficient Streaming Parse and Encode

**Location**: [`stream/main.go`](stream/main.go)

**What it does**: Demonstrates the streaming APIs for handling very large GEDCOM files with constant memory usage. Shows two streaming halves:

- **Streaming parse** with `parser.Records` — iterate level-0 records from any `io.Reader` without building a full `Document`
- **Streaming encode** with `encoder.NewStreamEncoder` — write records one at a time without holding the document in memory

**How to run**:
```bash
# Streaming parse only (skips encode demo)
cd examples/stream
go run main.go ../../testdata/gedcom-5.5/pres2020.ged

# Streaming parse + streaming encode to file
go run main.go ../../testdata/gedcom-5.5/pres2020.ged /tmp/out.ged
```

**Example output**:
```text
=== Streaming Parse ===
Input: ../../testdata/gedcom-5.5/pres2020.ged
Total lines processed: 49431
Record counts by type:
  HEAD: 1
  INDI: 2322
  FAM: 1115
  SOUR: 91
  NOTE: 141
  OBJE: 171
  REPO: 1
  SUBM: 1
  TRLR: 1
[heap after streaming parse] HeapAlloc=3053KB Sys=12978KB

=== Streaming Encode ===
Wrote 1000 individuals to /tmp/out.ged using StreamEncoder.
[heap after streaming encode] HeapAlloc=3382KB Sys=12978KB
```

**Use cases**:
- Files larger than available RAM (10k+ individuals)
- Extracting subsets without a full Document build
- Generating GEDCOM exports from streaming data sources
- Memory-constrained environments

---

### 5. Encode - Creating GEDCOM Files

**Location**: [`encode/main.go`](encode/main.go)

**What it does**: Shows how to create GEDCOM files programmatically by:
- Building a Document structure from scratch
- Creating individual records with names, dates, and events
- Creating family records with relationships
- Encoding the document to GEDCOM format
- Using encoding options (line endings, etc.)

**How to run**:
```bash
# Output to stdout
cd examples/encode
go run main.go

# Output to file
go run main.go output.ged
```

**Example output** (stdout):
```
GEDCOM output:
===============
0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
1 SOUR go-gedcom example
1 LANG English
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
1 SEX M
1 BIRT
2 DATE 1 JAN 1900
2 PLAC New York, USA
0 @I2@ INDI
1 NAME Jane /Smith/
2 GIVN Jane
2 SURN Smith
1 SEX F
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 15 JUN 1925
2 PLAC Boston, Massachusetts, USA
0 TRLR
```

**Use cases**:
- Creating GEDCOM exports from other formats
- Building genealogy data programmatically
- Testing and fixtures
- Data migration

---

### 6. Date Parsing - GEDCOM Date Values

**Location**: [`date-parsing/main.go`](date-parsing/main.go)

**What it does**: Parses a range of GEDCOM date values (exact dates, modifiers such as `ABT`, ranges and periods) and shows their parsed parts, calendar and comparison.

**How to run**:
```bash
cd examples/date-parsing
go run main.go
```

---

### 7. Convert - Converting Between GEDCOM Versions

**Location**: [`convert/main.go`](convert/main.go)

**What it does**: Converts a file to another GEDCOM version with `converter.Convert` and prints the conversion report: transformations, data loss, and the per-path notes for what was dropped, approximated, normalized or preserved. With an output path it also writes the converted document.

**How to run**:
```bash
cd examples/convert
go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged 7.0

# Also write the converted file
go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged 7.0 /tmp/converted.ged
```

**Example output** (abridged):
```text
Converted ../../testdata/gedcom-5.5.1/comprehensive.ged: 5.5.1 -> 7.0
Success: true

=== Transformations (5) ===
  NOTE_RECORD_TO_SNOTE x2: Converted NOTE records to SNOTE records
  ...
=== Dropped (1) ===
  Header > FILE: "FILE" -> "" (Structure not defined in GEDCOM 7.0)
...
=== Normalized (4) ===
  Header > LANG: "English" -> "en" (GEDCOM 7.0 identifies languages by BCP 47 tag)
  ...
```

---

### 8. Merge - Combining Two Documents

**Location**: [`merge/main.go`](merge/main.go)

**What it does**: Combines two files of the same GEDCOM version with `merge.Combine`, renumbering any XRef of the second file that collides with the first (`RenumberDoc2`), and prints the combine report: which XRefs were renumbered and which header values were dropped in favour of the first file's.

**How to run**:
```bash
cd examples/merge
go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged ../../testdata/gedcom-5.5.1/attribute-event-detail.ged

# Also write the combined file
go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged ../../testdata/gedcom-5.5.1/attribute-event-detail.ged /tmp/combined.ged
```

**Example output**:
```text
First:    15 records
Second:   2 records
Combined: 17 records (8 individuals, 3 families)

=== Renumbered XRefs from the second file (1) ===
  @I1@ -> @I8@

=== Header conflicts (4) ===
  SourceSystem: first "FamilyTreeMaker", second "gedcom-go"
  ...
```

---

### 9. Diagnostics - Seeing What Lenient Decoding Recovered

**Location**: [`diagnostics/main.go`](diagnostics/main.go)

**What it does**: Decodes a file with `decoder.DecodeWithDiagnostics` and lists every diagnostic with its line, severity, code and offending content. Lenient decoding (the default) recovers from or skips malformed lines; the diagnostics say where and how.

**How to run**:
```bash
cd examples/diagnostics
go run main.go ../../testdata/malformed/invalid-level.ged
```

**Example output**:
```text
GEDCOM File: ../../testdata/malformed/invalid-level.ged
Records decoded: 1
Diagnostics: 2 (0 error(s), 2 warning(s))

  [WARNING] line 6: BAD_LEVEL_JUMP: level jumped from 1 to 99 (skipping 97 level(s)); clamped to 2 for recovery (context: "INVALID")
  [WARNING] line 6: UNKNOWN_TAG: unknown tag: INVALID (context: "Deep nesting beyond reasonable limit")
```

---

### 10. Lazy - Indexed Random Access

**Location**: [`lazy/main.go`](lazy/main.go)

**What it does**: Builds a record index with `parser.LazyParser`, then parses only the records asked for by XRef and turns each into a typed record with `decoder.DecodeRecord`, without decoding the rest of the file. `LazyParser` reads the file's bytes without charset conversion, so use it on UTF-8 files; a file that starts with a byte-order mark is not yet supported ([#792](https://github.com/cacack/gedcom-go/issues/792)).

**How to run**:
```bash
cd examples/lazy
go run main.go ../../testdata/gedcom-5.5.1/comprehensive.ged @I1@ @F1@
```

**Example output**:
```text
Indexed ../../testdata/gedcom-5.5.1/comprehensive.ged: 17 records
Version: 5.5.1

@I1@ INDI at byte 928 (38 lines)
  Individual: William /Smith/
  Born: 15 JUN 1950

@F1@ FAM at byte 2434 (11 lines)
  Family: husband @I1@, wife @I2@, 1 child(ren)
```

**Use cases**:
- Looking up a few records in a large file
- Saving an index (`SaveIndex`) to skip the indexing pass next time

---

## Running All Examples

Run every example against the test data from the project root:

```bash
make examples
```

CI runs the same target, and `make build-examples` builds every directory under `examples/`.

## Test Data

The examples use test data from the `testdata/` directory:

- `testdata/gedcom-5.5/minimal.ged` - Minimal valid GEDCOM 5.5 file (good for testing)
- `testdata/gedcom-5.5/royal92.ged` - Complex real-world example (British Royal Family)
- `testdata/gedcom-5.5.1/` - GEDCOM 5.5.1 sample files
- `testdata/gedcom-7.0/` - GEDCOM 7.0 sample files
- `testdata/malformed/` - Invalid files for error testing

## Building the Examples

Each example can be built as a standalone executable:

```bash
# Build a specific example
cd parse
go build -o gedcom-parse

# Run the built executable
./gedcom-parse ../../testdata/gedcom-5.5/minimal.ged
```

## Modifying the Examples

These examples are designed to be educational and easy to modify. Feel free to:

1. **Add your own queries** to the query example
2. **Create custom validation rules** in the validate example
3. **Extend the data structures** in the encode example
4. **Combine examples** to create more complex applications

## Common Patterns

### Opening Files Safely

All examples follow this pattern:

```go
f, err := os.Open(filename)
if err != nil {
    log.Fatalf("Failed to open file: %v", err)
}
defer f.Close()
```

### Error Handling

Examples demonstrate proper error handling:

```go
doc, err := decoder.Decode(f)
if err != nil {
    log.Fatalf("Failed to decode GEDCOM: %v", err)
}
```

### Safe Field Access

Prefer the nil-safe accessors to indexing slices directly:

```go
fmt.Println(person.FullName())   // "" when there is no name
fmt.Println(person.BirthPlace()) // "" when there is no birth or place
```

## Next Steps

After exploring these examples, you can:

1. Read the [USAGE.md](../USAGE.md) guide for comprehensive documentation
2. Check the [API documentation](https://pkg.go.dev/github.com/cacack/gedcom-go/v3)
3. Look at the [test files](../decoder/decoder_test.go) for more advanced usage
4. Start building your own genealogy application!

## Troubleshooting

### "File not found" errors

Make sure you're running the examples from the correct directory and using relative paths to the testdata:

```bash
cd examples/parse
go run main.go ../../testdata/gedcom-5.5/minimal.ged
```

### Import errors

Ensure you're using Go 1.25 or later:

```bash
go version  # Should show 1.25 or higher
```

### Module issues

If you get module-related errors:

```bash
# From project root
go mod tidy
go mod download
```

## Contributing

Found a bug in an example? Have an idea for a new example? Please see [CONTRIBUTING.md](../CONTRIBUTING.md)!

## License

These examples are part of the `gedcom-go` project and are released under the MIT License. See [LICENSE](../LICENSE) for details.
