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

**What it does**: Demonstrates comprehensive GEDCOM validation with error categorization and reporting. Shows how to:
- Validate GEDCOM files against specification rules
- Group validation errors by error code
- Display detailed error information with line numbers and context
- Generate validation reports

**How to run**:
```bash
cd examples/validate
go run main.go ../../testdata/gedcom-5.5/minimal.ged
```

**Example output** (valid file):
```
Validating GEDCOM File: ../../testdata/gedcom-5.5/minimal.ged
Version: 5.5
Encoding: UTF-8

✅ Validation passed!
No errors found.
```

**Example output** (invalid file):
```
Validating GEDCOM File: malformed.ged
Version: 5.5
Encoding: UTF-8

❌ Validation failed with 12 error(s):

Error Code: MISSING_REQUIRED_TAG (5 occurrence(s))
  - Required tag NAME missing in record @I1@ (line 15, XRef: @I1@)
  - Required tag SEX missing in record @I2@ (line 25, XRef: @I2@)
  - Required tag DATE missing in BIRT event (line 18, XRef: @I1@)
  ... and 2 more

Error Code: INVALID_XREF (3 occurrence(s))
  - Invalid cross-reference @I99@ (line 42)
  - Cross-reference @F5@ not found (line 55)
  ... and 1 more

=== Summary ===
Total Records: 20
Total Errors: 12
Error Types: 4
```

**Use cases**:
- Quality assurance for GEDCOM files
- Pre-import validation
- Data cleaning workflows
- Compliance checking

---

### 4. Encode - Creating GEDCOM Files

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

## Running All Examples

You can test all examples at once using the test data provided:

```bash
# From the project root
cd examples

# Run parse example
cd parse && go run main.go ../../testdata/gedcom-5.5/minimal.ged && cd ..

# Run query example
cd query && go run main.go ../../testdata/gedcom-5.5/royal92.ged && cd ..

# Run validate example
cd validate && go run main.go ../../testdata/gedcom-5.5/minimal.ged && cd ..

# Run encode example
cd encode && go run main.go /tmp/output.ged && cd ..
```

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

Always check for nil and empty slices:

```go
if len(person.Names) > 0 {
    fmt.Println(person.Names[0].Full)
}
```

## Next Steps

After exploring these examples, you can:

1. Read the [USAGE.md](../USAGE.md) guide for comprehensive documentation
2. Check the [API documentation](https://pkg.go.dev/github.com/cacack/gedcom-go)
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

Ensure you're using Go 1.24 or later:

```bash
go version  # Should show 1.24 or higher
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
