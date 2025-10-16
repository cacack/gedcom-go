# Quick Start Guide: go-gedcom

**Feature**: 001-gedcom-parser-library
**Date**: 2025-10-16

## Installation

```bash
go get github.com/yourorg/go-gedcom
```

**Requirements**: Go 1.21 or later

## Basic Usage

### Parse a GEDCOM File

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yourorg/go-gedcom/decoder"
)

func main() {
	// Open GEDCOM file
	file, err := os.Open("family.ged")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create decoder and parse file
	dec := decoder.New()
	doc, err := dec.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	// Access parsed data
	fmt.Printf("GEDCOM Version: %v\n", doc.Version)
	fmt.Printf("Total Records: %d\n", len(doc.Records))

	// Find individuals
	for _, record := range doc.Records {
		if record.Type == gedcom.RecordTypeIndividual {
			fmt.Printf("Individual: %s\n", record.XRef)
		}
	}
}
```

**Output**:
```
GEDCOM Version: 5.5
Total Records: 1523
Individual: @I1@
Individual: @I2@
...
```

---

### Parse with Progress Reporting

```go
package main

import (
	"fmt"
	"os"

	"github.com/yourorg/go-gedcom/decoder"
)

func main() {
	file, err := os.Open("large-family.ged")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Configure options with progress callback
	opts := decoder.DecodeOptions{
		OnProgress: func(bytesRead, totalBytes, recordCount int64) {
			pct := (bytesRead * 100) / totalBytes
			fmt.Printf("\rProgress: %d%% (%d records)", pct, recordCount)
		},
	}

	// Decode with options
	dec := decoder.New()
	doc, err := dec.DecodeWithOptions(file, opts)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nParsing complete! %d records loaded.\n", len(doc.Records))
}
```

---

### Stream Large Files

For very large files that don't fit in memory:

```go
package main

import (
	"fmt"
	"os"

	"github.com/yourorg/go-gedcom/decoder"
	"github.com/yourorg/go-gedcom/gedcom"
)

func main() {
	file, err := os.Open("huge-family.ged")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	dec := decoder.New()

	// Process records one at a time
	individualsCount := 0
	err = dec.DecodeStream(file, func(record *gedcom.Record) error {
		if record.Type == gedcom.RecordTypeIndividual {
			individualsCount++
			// Process individual immediately, don't store it
		}
		return nil // Return error to stop processing
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("Processed %d individuals\n", individualsCount)
}
```

---

### Validate GEDCOM Data

```go
package main

import (
	"fmt"
	"os"

	"github.com/yourorg/go-gedcom/decoder"
	"github.com/yourorg/go-gedcom/validator"
)

func main() {
	file, err := os.Open("family.ged")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Parse file
	dec := decoder.New()
	doc, err := dec.Decode(file)
	if err != nil {
		panic(err)
	}

	// Validate against GEDCOM spec
	v := validator.New()
	errors := v.Validate(doc)

	if len(errors) == 0 {
		fmt.Println("✓ File is valid!")
	} else {
		fmt.Printf("Found %d validation issues:\n", len(errors))
		for _, verr := range errors {
			fmt.Printf("[%s] Line %d: %s\n",
				verr.Severity, verr.Location.LineNumber, verr.Message)
		}
	}
}
```

**Output**:
```
Found 3 validation issues:
[ERROR] Line 45: Individual @I15@ missing required NAME tag
[WARNING] Line 102: Non-standard cross-reference format @I-001@
[WARNING] Line 234: Deprecated tag REFN in GEDCOM 7.0
```

---

### Convert Between GEDCOM Versions

```go
package main

import (
	"fmt"
	"os"

	"github.com/yourorg/go-gedcom/converter"
	"github.com/yourorg/go-gedcom/decoder"
	"github.com/yourorg/go-gedcom/encoder"
	"github.com/yourorg/go-gedcom/gedcom"
)

func main() {
	// Parse GEDCOM 5.5 file
	file, err := os.Open("family-v5.5.ged")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	dec := decoder.New()
	doc, err := dec.Decode(file)
	if err != nil {
		panic(err)
	}

	// Convert to GEDCOM 7.0
	conv := converter.New()
	newDoc, report, err := conv.Convert(doc, gedcom.Version70)
	if err != nil {
		panic(err)
	}

	// Review conversion report
	fmt.Printf("Conversion complete!\n")
	fmt.Printf("Tags converted: %d\n", len(report.TagsConverted))
	fmt.Printf("Data lost: %d items\n", len(report.DataLost))

	for _, loss := range report.DataLost {
		fmt.Printf("  Line %d: %s - %s\n", loss.LineNumber, loss.Tag, loss.Reason)
	}

	// Write converted file
	outFile, err := os.Create("family-v7.0.ged")
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	enc := encoder.New()
	if err := enc.Encode(outFile, newDoc); err != nil {
		panic(err)
	}

	fmt.Println("Converted file written successfully!")
}
```

---

### Write GEDCOM Files

```go
package main

import (
	"os"

	"github.com/yourorg/go-gedcom/encoder"
	"github.com/yourorg/go-gedcom/gedcom"
)

func main() {
	// Create a simple GEDCOM document
	doc := &gedcom.Document{
		Header: gedcom.Header{
			Version:      gedcom.Version55,
			Encoding:     gedcom.EncodingUTF8,
			SourceSystem: "MyApp v1.0",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
					{Level: 1, Tag: "SEX", Value: "M"},
					{Level: 1, Tag: "BIRT", SubTags: []gedcom.Tag{
						{Level: 2, Tag: "DATE", Value: "1 JAN 1950"},
						{Level: 2, Tag: "PLAC", Value: "New York, NY, USA"},
					}},
				},
			},
		},
		Trailer: gedcom.Trailer{},
		Version: gedcom.Version55,
	}

	// Write to file
	file, err := os.Create("output.ged")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	enc := encoder.New()
	if err := enc.Encode(file, doc); err != nil {
		panic(err)
	}

	fmt.Println("GEDCOM file created successfully!")
}
```

---

## Advanced Usage

### Configure Resource Limits

Protect against malicious or corrupted files:

```go
opts := decoder.DecodeOptions{
	MaxNestingDepth: 100,          // Limit tag hierarchy depth
	MaxRecordCount:  1_000_000,    // Limit total records
	Timeout:         5 * time.Minute, // Limit parsing time
}

doc, err := dec.DecodeWithOptions(file, opts)
```

### Handle Encoding Issues

For files with undeclared or mixed encodings:

```go
// The library automatically:
// 1. Tries UTF-8 first
// 2. Falls back to Latin-1
// 3. Returns error if both fail

doc, err := dec.Decode(file)
if err != nil {
	// Check for encoding errors
	if errors.Is(err, charset.ErrInvalidEncoding) {
		fmt.Println("File has encoding issues")
	}
}
```

### Detect GEDCOM Version

```go
import "github.com/yourorg/go-gedcom/version"

file, _ := os.Open("unknown-version.ged")
defer file.Close()

detector := version.NewDetector()
ver, err := detector.Detect(file)
if err != nil {
	panic(err)
}

fmt.Printf("Detected GEDCOM version: %v\n", ver)

// Rewind file for parsing
file.Seek(0, 0)
```

---

## Error Handling

All errors include context for debugging:

```go
doc, err := dec.Decode(file)
if err != nil {
	// Check error type
	var parseErr *parser.ParseError
	if errors.As(err, &parseErr) {
		fmt.Printf("Parse error at line %d: %s\n",
			parseErr.Line, parseErr.Message)
		fmt.Printf("Content: %s\n", parseErr.Content)
	}
}
```

Common error types:
- `parser.ParseError`: Syntax errors in GEDCOM format
- `charset.EncodingError`: Character encoding problems
- `validator.ValidationError`: Spec compliance issues
- `decoder.ResourceLimitError`: Resource limits exceeded

---

## Performance Tips

1. **Use streaming for large files** (>50MB)
2. **Enable progress callbacks** for user feedback on slow operations
3. **Adjust resource limits** based on expected file sizes
4. **Close files** with `defer file.Close()` to avoid resource leaks
5. **Validate after parsing** (validation is separate from parsing)

---

## Common Patterns

### Find Individual by Name

```go
for _, record := range doc.Records {
	if record.Type == gedcom.RecordTypeIndividual {
		for _, tag := range record.Tags {
			if tag.Tag == "NAME" && strings.Contains(tag.Value, "Smith") {
				fmt.Printf("Found: %s (%s)\n", tag.Value, record.XRef)
			}
		}
	}
}
```

### Follow Cross-References

```go
// Get individual
individual := doc.XRefMap["@I1@"]

// Find their families
for _, tag := range individual.Tags {
	if tag.Tag == "FAMS" {
		familyXRef := tag.Value
		family := doc.XRefMap[familyXRef]
		fmt.Printf("Spouse in family %s\n", family.XRef)
	}
}
```

### Export to JSON

```go
import "encoding/json"

doc, _ := dec.Decode(file)
jsonData, _ := json.MarshalIndent(doc, "", "  ")
os.WriteFile("output.json", jsonData, 0644)
```

---

## Testing Your Code

```go
import "testing"

func TestParseGEDCOM(t *testing.T) {
	testData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR`

	dec := decoder.New()
	doc, err := dec.Decode(strings.NewReader(testData))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(doc.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(doc.Records))
	}
}
```

---

## Next Steps

- Read the [full API documentation](https://pkg.go.dev/github.com/yourorg/go-gedcom)
- Explore [examples/](../examples/) for complete programs
- Check [GEDCOM specifications](https://gedcom.io/) for format details
- Report issues on [GitHub](https://github.com/yourorg/go-gedcom/issues)

## Success Criteria Met

This quickstart guide enables developers to:
- ✅ Parse any GEDCOM file in under 5 lines of code (SC-001)
- ✅ Integrate the library in under 30 minutes (SC-009)
- ✅ Handle all three GEDCOM versions (FR-001)
- ✅ Use progress callbacks (FR-023)
- ✅ Validate data (FR-009)
- ✅ Convert between versions (FR-011)
