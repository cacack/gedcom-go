package decoder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// ============================================================================
// Integration tests for lenient parsing with real malformed files
// ============================================================================

// TestDecodeWithDiagnostics_MalformedFiles tests lenient parsing with files from testdata/malformed/
func TestDecodeWithDiagnostics_MalformedFiles(t *testing.T) {
	tests := []struct {
		name                 string
		path                 string
		description          string
		expectDiagnostics    bool   // Should have any diagnostics
		expectError          bool   // Should return an error
		minRecords           int    // Minimum expected records (partial parsing)
		expectDiagnosticCode string // Expected diagnostic code (if any)
	}{
		{
			name:                 "invalid-level.ged",
			path:                 "../testdata/malformed/invalid-level.ged",
			description:          "File with a level-99 jump (level itself is < MaxNestingDepth, but the +98 jump is malformed indentation)",
			expectDiagnostics:    true,
			expectError:          false,
			minRecords:           1, // At least one INDI record
			expectDiagnosticCode: CodeBadLevelJump,
		},
		{
			name:              "invalid-xref.ged",
			path:              "../testdata/malformed/invalid-xref.ged",
			description:       "File with reference to non-existent family",
			expectDiagnostics: false, // Parser accepts, broken XRef is semantic
			expectError:       false,
			minRecords:        1,
		},
		{
			name:              "missing-header.ged",
			path:              "../testdata/malformed/missing-header.ged",
			description:       "File missing HEAD record",
			expectDiagnostics: false, // Missing header is semantic, not syntactic
			expectError:       false,
			minRecords:        1,
		},
		{
			name:              "missing-xref.ged",
			path:              "../testdata/malformed/missing-xref.ged",
			description:       "File with reference to non-existent family",
			expectDiagnostics: false, // Parser accepts, broken XRef is semantic
			expectError:       false,
			minRecords:        1,
		},
		{
			name:              "circular-reference.ged",
			path:              "../testdata/malformed/circular-reference.ged",
			description:       "File with circular family relationships",
			expectDiagnostics: false, // Circular refs are semantic, not syntactic
			expectError:       false,
			minRecords:        3, // Multiple individuals and families
		},
		{
			name:              "duplicate-xref.ged",
			path:              "../testdata/malformed/duplicate-xref.ged",
			description:       "File with duplicate XRef identifiers",
			expectDiagnostics: false, // Duplicate XRefs are handled (last wins)
			expectError:       false,
			minRecords:        1, // At least one record
		},
		{
			name:                 "level-jump-skip.ged",
			path:                 "../testdata/malformed/level-jump-skip.ged",
			description:          "File with level jump 1 -> 4 (real-world Ancestry-style export)",
			expectDiagnostics:    true,
			expectError:          false,
			minRecords:           1,
			expectDiagnosticCode: CodeBadLevelJump,
		},
		{
			name:                 "level-jump-subordinate.ged",
			path:                 "../testdata/malformed/level-jump-subordinate.ged",
			description:          "File where a subordinate skips a level (PLAC jumps from 1 to 3)",
			expectDiagnostics:    true,
			expectError:          false,
			minRecords:           1,
			expectDiagnosticCode: CodeBadLevelJump,
		},
		{
			name:                 "unterminated-xref.ged",
			path:                 "../testdata/malformed/unterminated-xref.ged",
			description:          "File whose XRef has no closing @ (0 @I1 INDI)",
			expectDiagnostics:    true,
			expectError:          false,
			minRecords:           1,
			expectDiagnosticCode: CodeInvalidXRef,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			result, err := DecodeWithDiagnostics(f, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			if result == nil {
				t.Fatal("DecodeWithDiagnostics returned nil result")
			}

			if result.Document == nil {
				t.Fatal("DecodeWithDiagnostics returned nil document")
			}

			// Check diagnostics
			if tt.expectDiagnostics && len(result.Diagnostics) == 0 {
				t.Error("Expected diagnostics but got none")
			}

			// Check specific diagnostic code if expected
			if tt.expectDiagnosticCode != "" {
				found := false
				for _, diag := range result.Diagnostics {
					if diag.Code == tt.expectDiagnosticCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected diagnostic with code %s not found", tt.expectDiagnosticCode)
				}
			}

			// Check minimum records (partial parsing)
			if len(result.Document.Records) < tt.minRecords {
				t.Errorf("Expected at least %d records, got %d",
					tt.minRecords, len(result.Document.Records))
			}

			t.Logf("Parsed %s: %d records, %d diagnostics",
				tt.description, len(result.Document.Records), len(result.Diagnostics))
		})
	}
}

// TestDecodeWithDiagnostics_SyntheticMalformedInput tests with synthetic malformed input
func TestDecodeWithDiagnostics_SyntheticMalformedInput(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		expectDiagnostics    int    // Expected number of diagnostics
		expectRecords        int    // Expected number of records
		expectError          bool   // Should return an error
		expectDiagnosticCode string // Expected diagnostic code
	}{
		{
			name: "empty lines mixed with valid",
			input: `0 HEAD
1 GEDC
2 VERS 5.5

0 @I1@ INDI
1 NAME John /Smith/

0 TRLR`,
			expectDiagnostics:    2, // Two empty lines
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeEmptyLine,
		},
		{
			name: "invalid level mixed with valid",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
XYZ INVALID LINE
0 @I1@ INDI
1 NAME Jane /Doe/
ANOTHER BAD LINE
0 TRLR`,
			expectDiagnostics:    2, // Two invalid lines
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeInvalidLevel,
		},
		{
			name: "missing tag after level",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1
0 TRLR`,
			expectDiagnostics:    1,
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeSyntaxError,
		},
		{
			name: "xref without tag",
			input: `0 HEAD
1 GEDC
2 VERS 5.5
0 @BADXREF@
0 @I1@ INDI
1 NAME Valid /Person/
0 TRLR`,
			expectDiagnostics:    1,
			expectRecords:        1,
			expectError:          false,
			expectDiagnosticCode: CodeInvalidXRef,
		},
		{
			name: "all lines invalid",
			input: `invalid1
invalid2
invalid3`,
			expectDiagnostics: 3,
			expectRecords:     0,
			expectError:       true, // Error when no valid lines
		},
		{
			name:              "valid file no errors",
			input:             "0 HEAD\n1 GEDC\n2 VERS 5.5\n0 @I1@ INDI\n1 NAME Test\n0 TRLR",
			expectDiagnostics: 0,
			expectRecords:     1,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeWithDiagnostics(strings.NewReader(tt.input), nil)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			if result == nil {
				if tt.expectError {
					// If error expected and result is nil, that's acceptable
					return
				}
				t.Fatal("DecodeWithDiagnostics returned nil result")
			}

			if len(result.Diagnostics) != tt.expectDiagnostics {
				t.Errorf("Expected %d diagnostics, got %d",
					tt.expectDiagnostics, len(result.Diagnostics))
				for _, d := range result.Diagnostics {
					t.Logf("  Diagnostic: %s", d.String())
				}
			}

			if result.Document != nil && len(result.Document.Records) != tt.expectRecords {
				t.Errorf("Expected %d records, got %d",
					tt.expectRecords, len(result.Document.Records))
			}

			if tt.expectDiagnosticCode != "" && len(result.Diagnostics) > 0 {
				found := false
				for _, diag := range result.Diagnostics {
					if diag.Code == tt.expectDiagnosticCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected diagnostic with code %s not found",
						tt.expectDiagnosticCode)
				}
			}
		})
	}
}

// TestDecodeWithDiagnostics_UnterminatedXRef pins the end-to-end behavior for
// an XRef with no closing "@" (issue #385): strict mode rejects the file,
// lenient mode keeps the record under its verbatim identifier and reports an
// INVALID_XREF diagnostic.
func TestDecodeWithDiagnostics_UnterminatedXRef(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1 INDI
1 NAME Broken /Xref/
0 @I2@ INDI
1 NAME Valid /Person/
0 TRLR`

	t.Run("strict mode rejects it", func(t *testing.T) {
		_, err := DecodeWithOptions(strings.NewReader(input), &DecodeOptions{StrictMode: true})
		if err == nil {
			t.Fatal("strict decode should reject an xref with no closing @")
		}
		if !strings.Contains(err.Error(), "xref is missing its closing @") {
			t.Errorf("strict decode error = %v, want it to name the unterminated xref", err)
		}
	})

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	doc := result.Document

	t.Run("reported once", func(t *testing.T) {
		if len(result.Diagnostics) != 1 {
			t.Fatalf("Expected 1 diagnostic, got %v", result.Diagnostics)
		}
		d := result.Diagnostics[0]
		if d.Code != CodeInvalidXRef || d.Severity != SeverityError || d.Line != 4 {
			t.Errorf("Diagnostic = %s, want %s at %s on line 4",
				d.String(), CodeInvalidXRef, SeverityError)
		}
		if !strings.Contains(d.Message, "@I1") {
			t.Errorf("Diagnostic message = %q, want it to name the identifier", d.Message)
		}
	})

	t.Run("record recovered under its verbatim identifier", func(t *testing.T) {
		// The identifier is stored exactly as written, with no closing "@"
		// invented. gedcom.IsPointerXRef needs both delimiters, so an ordinary
		// "@I1@" pointer still will not reach this record: the value here is
		// the diagnostic plus correct record typing, not pointer resolution.
		indi := doc.GetIndividual("@I1")
		if indi == nil {
			t.Fatal("individual should be stored under its verbatim identifier @I1")
		}
		if doc.GetIndividual("@I1@") != nil {
			t.Error("no closing @ was invented, so @I1@ must not resolve")
		}
		if len(indi.Names) != 1 || indi.Names[0].Full != "Broken /Xref/" {
			t.Errorf("Names = %+v, want the single NAME from the recovered record", indi.Names)
		}
		for _, rec := range doc.Records {
			if rec.Type == gedcom.RecordType("@I1") {
				t.Errorf("unterminated xref must not surface as a bogus record type: %+v", rec)
			}
		}
	})

	t.Run("subordinates are not reparented", func(t *testing.T) {
		// The regression guard for the panel blocker: dropping the level-0
		// line would attach Broken's NAME to the preceding record instead.
		if len(doc.Records) != 2 {
			t.Fatalf("Expected 2 records, got %d", len(doc.Records))
		}
		next := doc.GetIndividual("@I2@")
		if next == nil {
			t.Fatal("individual @I2@ should exist")
		}
		if len(next.Names) != 1 || next.Names[0].Full != "Valid /Person/" {
			t.Errorf("@I2@ Names = %+v, want exactly one NAME (\"Valid /Person/\")", next.Names)
		}
	})
}

// TestDecodeWithDiagnostics_UnterminatedXRefBeforeStructuralTag pins the
// HEAD/TRLR carve-out in parseUnterminatedXRef (issue #385). buildHeader and
// buildRecords both key off a bare level-0 HEAD/TRLR tag, so recovering the
// identifier out of "0 @I1 HEAD" would make a bogus second header overwrite
// the real one's typed fields, and out of "0 @I1 TRLR" would drop the line and
// everything subordinate to it. Both keep their pre-existing parse instead,
// reported but not recovered.
func TestDecodeWithDiagnostics_UnterminatedXRefBeforeStructuralTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{name: "HEAD", tag: "HEAD"},
		{name: "TRLR", tag: "TRLR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `0 HEAD
1 SOUR RealVendor
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1 ` + tt.tag + `
1 SOUR FakeVendor
1 CHAR ANSEL
0 @I2@ INDI
1 NAME Valid /Person/
0 TRLR`

			result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			doc := result.Document

			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != CodeInvalidXRef {
				t.Errorf("Diagnostics = %v, want exactly one %s", result.Diagnostics, CodeInvalidXRef)
			}

			// The real header must win: a second HEAD block must not be
			// treated as a header at all.
			if doc.Header.SourceSystem != "RealVendor" || doc.Header.Encoding != "UTF-8" {
				t.Errorf("Header = {SourceSystem: %q, Encoding: %q}, want the real header's values",
					doc.Header.SourceSystem, doc.Header.Encoding)
			}

			// The malformed line stays a record, keeping its subordinate lines
			// with it (Lossless Representation).
			if len(doc.Records) != 2 {
				t.Fatalf("Expected 2 records, got %d", len(doc.Records))
			}
			rec := doc.Records[0]
			if rec.Type != gedcom.RecordType("@I1") || rec.Value != tt.tag {
				t.Errorf("Record[0] = {Type: %q, Value: %q}, want the pre-existing parse {@I1, %s}",
					rec.Type, rec.Value, tt.tag)
			}
			if len(rec.Tags) != 2 || rec.Tags[0].Tag != "SOUR" || rec.Tags[1].Tag != "CHAR" {
				t.Errorf("Record[0].Tags = %+v, want both subordinate lines", rec.Tags)
			}

			next := doc.GetIndividual("@I2@")
			if next == nil {
				t.Fatal("individual @I2@ should exist")
			}
			if len(next.Names) != 1 || next.Names[0].Full != "Valid /Person/" {
				t.Errorf("@I2@ Names = %+v, want exactly one NAME (\"Valid /Person/\")", next.Names)
			}
		})
	}
}

// TestDecodeWithDiagnostics_XRefOnStructuralLine pins the handling of a
// well-formed XRef on a level-0 HEAD or TRLR line (issue #396). Neither takes a
// cross-reference identifier in the GEDCOM grammar, so "0 @X1@ HEAD" is not the
// document's header: treating it as one let a bogus second header overwrite the
// real header's typed fields, and treating "0 @X1@ TRLR" as the trailer dropped
// the line and everything subordinate to it. Both are now ordinary records and
// both are reported.
func TestDecodeWithDiagnostics_XRefOnStructuralLine(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{name: "HEAD", tag: "HEAD"},
		{name: "TRLR", tag: "TRLR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `0 HEAD
1 SOUR RealVendor
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @X1@ ` + tt.tag + `
1 SOUR FakeVendor
1 CHAR ANSEL
0 @I2@ INDI
1 NAME Valid /Person/
0 TRLR`

			result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			doc := result.Document

			// Reported, never silently accepted (ADR 0007).
			if len(result.Diagnostics) != 1 {
				t.Errorf("Diagnostics = %v, want exactly one", result.Diagnostics)
			} else {
				d := result.Diagnostics[0]
				if d.Code != CodeInvalidXRef || d.Severity != SeverityError || d.Line != 6 {
					t.Errorf("Diagnostic = %s, want %s at %s on line 6",
						d.String(), CodeInvalidXRef, SeverityError)
				}
				if !strings.Contains(d.Message, "@X1@") || !strings.Contains(d.Message, tt.tag) {
					t.Errorf("Diagnostic message = %q, want it to name both the xref and the tag", d.Message)
				}
			}

			// The real header must win: the second block is not a header at all.
			if doc.Header.SourceSystem != "RealVendor" || doc.Header.Encoding != "UTF-8" {
				t.Errorf("Header = {SourceSystem: %q, Encoding: %q}, want the real header's values",
					doc.Header.SourceSystem, doc.Header.Encoding)
			}

			// The line stays a record and keeps its subordinates (Lossless
			// Representation), and the following record is not swallowed.
			if len(doc.Records) != 2 {
				t.Fatalf("Expected 2 records, got %d", len(doc.Records))
			}
			rec := doc.Records[0]
			if rec.Type != gedcom.RecordType(tt.tag) || rec.XRef != "@X1@" {
				t.Errorf("Record[0] = {Type: %q, XRef: %q}, want {%s, @X1@}", rec.Type, rec.XRef, tt.tag)
			}
			if len(rec.Tags) != 2 || rec.Tags[0].Value != "FakeVendor" || rec.Tags[1].Value != "ANSEL" {
				t.Errorf("Record[0].Tags = %+v, want both subordinate lines", rec.Tags)
			}
			if doc.GetRecord("@X1@") != rec {
				t.Errorf("GetRecord(@X1@) = %v, want the %s record", doc.GetRecord("@X1@"), tt.tag)
			}

			// A structural record is not an entity: the typed accessors must
			// not pick it up.
			if rec.Entity != nil {
				t.Errorf("Record[0].Entity = %v, want nil", rec.Entity)
			}
			if len(doc.Individuals()) != 1 || len(doc.Notes()) != 0 {
				t.Errorf("Individuals = %d, Notes = %d, want 1 and 0",
					len(doc.Individuals()), len(doc.Notes()))
			}

			next := doc.GetIndividual("@I2@")
			if next == nil {
				t.Fatal("individual @I2@ should exist")
			}
			if len(next.Names) != 1 || next.Names[0].Full != "Valid /Person/" {
				t.Errorf("@I2@ Names = %+v, want exactly one NAME (\"Valid /Person/\")", next.Names)
			}

			// The line is well formed, so the parser accepts it and strict mode
			// does not reject the file. DecodeWithOptions has no diagnostics
			// channel, so the record shape is the only signal there — the
			// document must still come out the same way.
			strictDoc, err := DecodeWithOptions(strings.NewReader(input), &DecodeOptions{StrictMode: true})
			if err != nil {
				t.Fatalf("DecodeWithOptions() error = %v", err)
			}
			if strictDoc.Header.SourceSystem != "RealVendor" || len(strictDoc.Records) != 2 {
				t.Errorf("strict decode = {SourceSystem: %q, Records: %d}, want {RealVendor, 2}",
					strictDoc.Header.SourceSystem, len(strictDoc.Records))
			}
			if strictDoc.Records[0].Type != gedcom.RecordType(tt.tag) || strictDoc.Records[0].XRef != "@X1@" {
				t.Errorf("strict decode Record[0] = {Type: %q, XRef: %q}, want {%s, @X1@}",
					strictDoc.Records[0].Type, strictDoc.Records[0].XRef, tt.tag)
			}
		})
	}
}

// TestDecode_StructuralLinesWithoutXRef guards the control case for issue #396:
// a bare "0 HEAD" and "0 TRLR" stay structural, so an ordinary file is
// completely unaffected by the XRef check.
func TestDecode_StructuralLinesWithoutXRef(t *testing.T) {
	input := `0 HEAD
1 SOUR RealVendor
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Valid /Person/
0 TRLR`

	// DecodeWithOptions passes a nil collector; the diagnostic path must be
	// nil-safe on this API too.
	doc, err := DecodeWithOptions(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithOptions() error = %v", err)
	}
	if doc.Header.SourceSystem != "RealVendor" || doc.Header.Encoding != "UTF-8" {
		t.Errorf("Header = {SourceSystem: %q, Encoding: %q}, want the real header's values",
			doc.Header.SourceSystem, doc.Header.Encoding)
	}
	if len(doc.Records) != 1 || doc.Records[0].Type != gedcom.RecordTypeIndividual {
		t.Fatalf("Records = %+v, want the single INDI record", doc.Records)
	}

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("Diagnostics = %v, want none for a well-formed file", result.Diagnostics)
	}
}

// TestDecodeWithDiagnostics_RecoveredRecordsUsable verifies recovered records are fully usable
func TestDecodeWithDiagnostics_RecoveredRecordsUsable(t *testing.T) {
	// Input with errors but recoverable individual record
	input := `0 HEAD
1 GEDC
2 VERS 5.5

0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 BIRT
2 DATE 1 JAN 1950
2 PLAC New York, NY

0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	if result == nil || result.Document == nil {
		t.Fatal("Result or document is nil")
	}

	// Should have two diagnostics for empty lines (one before INDI, one before FAM)
	if len(result.Diagnostics) != 2 {
		t.Errorf("Expected 2 diagnostics, got %d", len(result.Diagnostics))
	}

	// Verify individual record is usable
	individual := result.Document.GetIndividual("@I1@")
	if individual == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}

	// Check name
	if len(individual.Names) != 1 {
		t.Errorf("Expected 1 name, got %d", len(individual.Names))
	}
	if individual.Names[0].Full != "John /Smith/" {
		t.Errorf("Name = %q, want %q", individual.Names[0].Full, "John /Smith/")
	}

	// Check sex
	if individual.Sex != "M" {
		t.Errorf("Sex = %q, want %q", individual.Sex, "M")
	}

	// Check birth event
	if len(individual.Events) == 0 {
		t.Fatal("No events found")
	}

	// Verify family record is usable
	family := result.Document.GetFamily("@F1@")
	if family == nil {
		t.Fatal("GetFamily(@F1@) returned nil")
	}

	if family.Husband != "@I1@" {
		t.Errorf("Husband = %q, want %q", family.Husband, "@I1@")
	}

	// Verify XRefMap is populated
	if result.Document.XRefMap["@I1@"] == nil {
		t.Error("XRefMap[@I1@] is nil")
	}
	if result.Document.XRefMap["@F1@"] == nil {
		t.Error("XRefMap[@F1@] is nil")
	}
}

// TestDecodeWithDiagnostics_StrictVsLenient compares strict and lenient mode
func TestDecodeWithDiagnostics_StrictVsLenient(t *testing.T) {
	// Input with an error in the middle
	input := `0 HEAD
1 GEDC
2 VERS 5.5
INVALID LINE
0 @I1@ INDI
1 NAME Test
0 TRLR`

	t.Run("strict mode fails on error", func(t *testing.T) {
		opts := &DecodeOptions{StrictMode: true}
		result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)

		if err == nil {
			t.Error("Expected error in strict mode")
		}
		if result != nil {
			t.Error("Expected nil result in strict mode on error")
		}
	})

	t.Run("lenient mode continues after error", func(t *testing.T) {
		opts := &DecodeOptions{StrictMode: false}
		result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)

		if err != nil {
			t.Fatalf("Unexpected error in lenient mode: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result in lenient mode")
		}
		if len(result.Diagnostics) == 0 {
			t.Error("Expected diagnostics in lenient mode")
		}
		if len(result.Document.Records) != 1 {
			t.Errorf("Expected 1 record in lenient mode, got %d",
				len(result.Document.Records))
		}
	})
}

// TestDecodeWithDiagnostics_AllMalformedFilesIntegration runs through all malformed test files
func TestDecodeWithDiagnostics_AllMalformedFilesIntegration(t *testing.T) {
	malformedDir := "../testdata/malformed"

	entries, err := os.ReadDir(malformedDir)
	if err != nil {
		t.Skipf("Could not read malformed directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ged") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(malformedDir, entry.Name())
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("Could not open file: %v", err)
			}
			defer f.Close()

			result, err := DecodeWithDiagnostics(f, nil)

			// In lenient mode, we should get a result even with malformed files
			// (unless the file is completely unparseable)
			if result == nil && err == nil {
				t.Error("Expected either result or error")
			}

			if result != nil {
				t.Logf("File %s: %d records, %d diagnostics, error=%v",
					entry.Name(),
					len(result.Document.Records),
					len(result.Diagnostics),
					err)

				// Log diagnostics for debugging
				for _, diag := range result.Diagnostics {
					t.Logf("  Diagnostic: %s", diag.String())
				}
			}
		})
	}
}

// TestDecodeWithDiagnostics_DiagnosticsHelpersWithRealData tests helper methods
func TestDecodeWithDiagnostics_DiagnosticsHelpersWithRealData(t *testing.T) {
	// Input with both parser errors and entity warnings
	input := `0 HEAD
1 GEDC
2 VERS 5.5

0 @I1@ INDI
1 NAME John /Smith/
1 UNKNOWNTAG custom value
INVALID LINE
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// Should have diagnostics
	if len(result.Diagnostics) == 0 {
		t.Fatal("Expected diagnostics")
	}

	// Test HasErrors - should be true (parser error is SeverityError)
	if !result.Diagnostics.HasErrors() {
		t.Error("HasErrors() should return true")
	}

	// Test Errors() - should return error-level diagnostics
	errors := result.Diagnostics.Errors()
	if len(errors) == 0 {
		t.Error("Errors() should return at least one error")
	}
	for _, e := range errors {
		if e.Severity != SeverityError {
			t.Errorf("Errors() returned non-error: %v", e.Severity)
		}
	}

	// Test Warnings() - should return warning-level diagnostics (unknown tag)
	warnings := result.Diagnostics.Warnings()
	// Unknown tags generate warnings
	for _, w := range warnings {
		if w.Severity != SeverityWarning {
			t.Errorf("Warnings() returned non-warning: %v", w.Severity)
		}
	}

	// Test String() output
	output := result.Diagnostics.String()
	if !strings.Contains(output, "diagnostic(s)") {
		t.Errorf("String() should contain 'diagnostic(s)', got: %s", output)
	}

	t.Logf("Diagnostics output:\n%s", output)
}

// TestDecodeWithDiagnostics_LevelJumpRecovery_SingleSubordinateSkip verifies
// that a single subordinate tag with a level jump (1 -> 4) is recovered: a
// CodeBadLevelJump diagnostic is emitted and the DATE value lands on the BIRT
// event rather than being silently dropped.
func TestDecodeWithDiagnostics_LevelJumpRecovery_SingleSubordinateSkip(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
4 DATE 1 JAN 1900
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	// One BAD_LEVEL_JUMP for the 4 DATE line.
	jumps := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeBadLevelJump {
			jumps++
		}
	}
	if jumps != 1 {
		t.Errorf("expected 1 BAD_LEVEL_JUMP diagnostic, got %d (all: %v)", jumps, result.Diagnostics)
	}

	indi := result.Document.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}
	if len(indi.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(indi.Events))
	}
	birt := indi.Events[0]
	if birt.Date != "1 JAN 1900" {
		t.Errorf("BIRT.Date = %q, want %q (DATE was orphaned by level jump)", birt.Date, "1 JAN 1900")
	}
}

// TestDecodeWithDiagnostics_LevelJumpRecovery_MidRecordSubordinateSkip verifies
// the trickier case where a subordinate tag (PLAC) jumps from level 1 to 3
// mid-record. The clamped PLAC must attach to the immediately preceding level-1
// event (DEAT), not to the prior event (BIRT) or be silently dropped.
func TestDecodeWithDiagnostics_LevelJumpRecovery_MidRecordSubordinateSkip(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Jane /Doe/
1 BIRT
2 DATE 1 JAN 1900
1 DEAT
3 PLAC London, England
0 TRLR`

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	jumps := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeBadLevelJump {
			jumps++
		}
	}
	if jumps != 1 {
		t.Errorf("expected 1 BAD_LEVEL_JUMP diagnostic, got %d (all: %v)", jumps, result.Diagnostics)
	}

	indi := result.Document.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}
	if len(indi.Events) != 2 {
		t.Fatalf("expected 2 events (BIRT + DEAT), got %d", len(indi.Events))
	}

	var birt, deat *gedcom.Event
	for _, e := range indi.Events {
		switch e.Type {
		case gedcom.EventBirth:
			birt = e
		case gedcom.EventDeath:
			deat = e
		}
	}
	if birt == nil || deat == nil {
		t.Fatalf("missing BIRT or DEAT event; got events: %+v", indi.Events)
	}

	// PLAC must attach to DEAT (its preceding level-1 sibling), not BIRT.
	if deat.PlaceName() != "London, England" {
		t.Errorf("DEAT.PlaceName() = %q, want %q (PLAC was attached to the wrong event after clamping)",
			deat.PlaceName(), "London, England")
	}
	if birt.PlaceName() != "" {
		t.Errorf("BIRT.PlaceName() = %q, expected empty (PLAC should not have leaked to BIRT)", birt.PlaceName())
	}
	// BIRT.Date should still parse normally.
	if birt.Date != "1 JAN 1900" {
		t.Errorf("BIRT.Date = %q, want %q", birt.Date, "1 JAN 1900")
	}
}

// TestDecodeWithDiagnostics_SubordinateSpacedXRef pins that a recovered XRef
// on a level >= 1 line survives into the built document (issue #395). The
// parser recovers "1 @I 1@ NOTE some text" into Line{XRef: "@I 1@"} and
// reports it, but buildRecords used to drop line.XRef when it built the
// subordinate gedcom.Tag, so the identifier vanished from the document
// entirely -- a Lossless Representation violation.
func TestDecodeWithDiagnostics_SubordinateSpacedXRef(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I2@ INDI
1 @I 1@ NOTE some text
1 NAME Valid /Person/
0 TRLR`

	t.Run("strict mode rejects it", func(t *testing.T) {
		_, err := DecodeWithOptions(strings.NewReader(input), &DecodeOptions{StrictMode: true})
		if err == nil {
			t.Fatal("strict decode should reject an xref containing a space")
		}
		if !strings.Contains(err.Error(), "xref contains a space") {
			t.Errorf("strict decode error = %v, want it to name the spaced xref", err)
		}
	})

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	doc := result.Document

	t.Run("reported once", func(t *testing.T) {
		if len(result.Diagnostics) != 1 {
			t.Fatalf("Expected 1 diagnostic, got %v", result.Diagnostics)
		}
		d := result.Diagnostics[0]
		if d.Code != CodeInvalidXRef || d.Severity != SeverityError || d.Line != 5 {
			t.Errorf("Diagnostic = %s, want %s at %s on line 5",
				d.String(), CodeInvalidXRef, SeverityError)
		}
	})

	t.Run("identifier survives on the raw tag", func(t *testing.T) {
		rec := doc.GetRecord("@I2@")
		if rec == nil {
			t.Fatal("record @I2@ should exist")
		}
		if len(rec.Tags) != 2 {
			t.Fatalf("Tags = %+v, want the NOTE and the NAME", rec.Tags)
		}
		note := rec.Tags[0]
		if note.Tag != "NOTE" || note.Value != "some text" {
			t.Errorf("tag = %+v, want NOTE with value %q", note, "some text")
		}
		if note.XRef != "@I 1@" {
			t.Errorf("tag XRef = %q, want %q -- the recovered identifier must not be dropped",
				note.XRef, "@I 1@")
		}
		// The malformed identifier is preserved, not promoted: it is not a
		// pointer (gedcom.IsPointerXRef rejects an interior space) and it does
		// not become an addressable record.
		if gedcom.IsPointerXRef(note.XRef) {
			t.Errorf("%q must not count as a resolvable pointer", note.XRef)
		}
		if doc.GetRecord("@I 1@") != nil {
			t.Error("a subordinate identifier must not be indexed as a record")
		}
	})

	t.Run("valid lines carry no subordinate xref", func(t *testing.T) {
		// Well-formed GEDCOM has no XRef at level >= 1, so every other tag in
		// the document is unaffected by carrying line.XRef through.
		rec := doc.GetRecord("@I2@")
		if got := rec.Tags[1]; got.Tag != "NAME" || got.XRef != "" {
			t.Errorf("NAME tag = %+v, want XRef %q", got, "")
		}
	})
}
