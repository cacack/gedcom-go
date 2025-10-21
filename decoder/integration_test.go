package decoder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// T058: Parse all sample files in testdata/ and verify success
func TestParseRealGEDCOMFiles(t *testing.T) {
	testFiles := []string{
		"../testdata/gedcom-5.5/minimal.ged",
		"../testdata/gedcom-5.5.1/minimal.ged",
		"../testdata/gedcom-7.0/minimal.ged",
	}

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			f, err := os.Open(testFile)
			if err != nil {
				t.Skipf("Test file not found: %s", testFile)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for file %s", err, testFile)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			t.Logf("Successfully parsed %s: %d records", testFile, len(doc.Records))
		})
	}
}

// Test GEDCOM 5.5 Torture Test Suite - comprehensive validation
// NOTE: Currently skipped because these files use ISO-8859/ANSEL encoding
// which requires character set conversion support not yet implemented.
// TODO: Implement ANSEL-to-UTF-8 conversion in charset package
func TestTortureTestSuite(t *testing.T) {
	t.Skip("Torture test files use ANSEL encoding which is not yet supported. " +
		"These files use ISO-8859 character encoding with ANSEL special characters. " +
		"Future work: implement charset conversion in the charset package.")

	testFiles := []struct {
		path        string
		description string
		minRecords  int
	}{
		{
			path:        "../testdata/gedcom-5.5/torture-test/TGC551.ged",
			description: "Full torture test, CR line endings, single NAME structure",
			minRecords:  10, // Expect at least 10 records (individuals, families, sources, etc.)
		},
		{
			path:        "../testdata/gedcom-5.5/torture-test/TGC551LF.ged",
			description: "Full torture test, CRLF line endings, single NAME structure",
			minRecords:  10,
		},
		{
			path:        "../testdata/gedcom-5.5/torture-test/TGC55C.ged",
			description: "Full torture test, CR line endings, multiple NAME structures",
			minRecords:  10,
		},
		{
			path:        "../testdata/gedcom-5.5/torture-test/TGC55CLF.ged",
			description: "Full torture test, CRLF line endings, multiple NAME structures",
			minRecords:  10,
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for %s", err, tt.description)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			if len(doc.Records) < tt.minRecords {
				t.Errorf("Expected at least %d records, got %d", tt.minRecords, len(doc.Records))
			}

			// Verify it's GEDCOM 5.5
			if doc.Header.Version != gedcom.Version55 {
				t.Errorf("Expected GEDCOM 5.5, got %v", doc.Header.Version)
			}

			// Verify ANSEL encoding is detected
			if doc.Header.Encoding != gedcom.EncodingANSEL {
				t.Logf("Warning: Expected ANSEL encoding, got %v", doc.Header.Encoding)
			}

			t.Logf("Successfully parsed %s: %d records, %d XRefs",
				tt.description, len(doc.Records), len(doc.XRefMap))
		})
	}
}

// Test GEDCOM 7.0 FamilySearch Examples - edge cases and features
func TestFamilySearchGEDCOM70Examples(t *testing.T) {
	testFiles := []struct {
		path        string
		description string
	}{
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/age.ged",
			description: "AGE payload format variations",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/escapes.ged",
			description: "@ character escaping rules",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/extension-record.ged",
			description: "Custom _LOC record extensions",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/extensions.ged",
			description: "Various extension formats",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/lang.ged",
			description: "LANG payload examples",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/filename-1.ged",
			description: "FILE payload format variations",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/long-url.ged",
			description: "Very long line parsing",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/notes-1.ged",
			description: "NOTE and SNOTE usage patterns",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/obje-1.ged",
			description: "OBJE record variations",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/voidptr.ged",
			description: "@VOID@ reference handling",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/xref.ged",
			description: "Cross-reference identifier formats",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/remarriage2.ged",
			description: "Divorce/remarriage scenario",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/same-sex-marriage.ged",
			description: "Same-sex marriage example",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/maximal70-tree1.ged",
			description: "Individuals, families, sources, events",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/maximal70-tree2.ged",
			description: "Extended attributes and events",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/maximal70-lds.ged",
			description: "LDS ordinance structures",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/maximal70-memories1.ged",
			description: "Multimedia object records",
		},
		{
			path:        "../testdata/gedcom-7.0/familysearch-examples/maximal70-memories2.ged",
			description: "Family and event multimedia",
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for %s", err, tt.description)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			// Verify it's GEDCOM 7.0
			if doc.Header.Version != gedcom.Version70 {
				t.Errorf("Expected GEDCOM 7.0, got %v", doc.Header.Version)
			}

			t.Logf("Successfully parsed %s: %d records", tt.description, len(doc.Records))
		})
	}
}

// Test large real-world GEDCOM files for performance and scalability
func TestLargeRealWorldFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file tests in short mode")
	}

	testFiles := []struct {
		path           string
		description    string
		minIndividuals int
	}{
		{
			path:           "../testdata/gedcom-5.5/pres2020.ged",
			description:    "US Presidents and families (2,322 individuals)",
			minIndividuals: 2000,
		},
		{
			path:           "../testdata/gedcom-5.5/royal92.ged",
			description:    "European royal families (3,010 individuals)",
			minIndividuals: 3000,
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for %s", err, tt.description)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			// Count individual records
			individualCount := 0
			for _, rec := range doc.Records {
				if rec.Type == gedcom.RecordTypeIndividual {
					individualCount++
				}
			}

			if individualCount < tt.minIndividuals {
				t.Errorf("Expected at least %d individuals, got %d", tt.minIndividuals, individualCount)
			}

			t.Logf("Successfully parsed %s: %d total records, %d individuals, %d XRefs",
				tt.description, len(doc.Records), individualCount, len(doc.XRefMap))
		})
	}
}

// Test all other GEDCOM samples
func TestOtherGEDCOMSamples(t *testing.T) {
	testFiles := []struct {
		path        string
		description string
	}{
		{
			path:        "../testdata/gedcom-7.0/minimal70.ged",
			description: "Smallest legal FamilySearch GEDCOM 7.0",
		},
		{
			path:        "../testdata/gedcom-7.0/maximal70.ged",
			description: "Exercises all standard GEDCOM 7.0 tags",
		},
		{
			path:        "../testdata/gedcom-7.0/remarriage1.ged",
			description: "Divorce/remarriage scenario",
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for %s", err, tt.description)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			t.Logf("Successfully parsed %s: %d records", tt.description, len(doc.Records))
		})
	}
}

// Test that malformed files are properly rejected or handled
func TestMalformedFilesIntegration(t *testing.T) {
	testFiles := []struct {
		path        string
		description string
		shouldError bool
	}{
		{
			path:        "../testdata/malformed/invalid-level.ged",
			description: "File with level 99 (unusually deep nesting)",
			shouldError: false, // Parser accepts any level < 100
		},
		{
			path:        "../testdata/malformed/invalid-xref.ged",
			description: "File with malformed cross-reference",
			shouldError: false, // Decoder may accept, validator should catch
		},
		{
			path:        "../testdata/malformed/missing-header.ged",
			description: "File missing required HEAD record",
			shouldError: false, // Parser accepts, decoder may validate
		},
		{
			path:        "../testdata/malformed/missing-xref.ged",
			description: "File with reference to non-existent record",
			shouldError: false, // Decoder accepts, validator should catch broken XRef
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Logf("Got error (may be acceptable): %v", err)
				}
				if doc != nil {
					t.Logf("Parsed %s: %d records", tt.description, len(doc.Records))
				}
			}
		})
	}
}

// Test GEDCOM 5.5.1 comprehensive features (EMAIL, FAX, WWW tags)
func TestGEDCOM551Comprehensive(t *testing.T) {
	t.Run("comprehensive.ged", func(t *testing.T) {
		f, err := os.Open("../testdata/gedcom-5.5.1/comprehensive.ged")
		if err != nil {
			t.Skipf("Test file not found: %s", "../testdata/gedcom-5.5.1/comprehensive.ged")
			return
		}
		defer f.Close()

		doc, err := Decode(f)
		if err != nil {
			t.Fatalf("Decode() error = %v for comprehensive GEDCOM 5.5.1 file", err)
		}

		if doc == nil {
			t.Fatal("Decode() returned nil document")
		}

		// Verify it's GEDCOM 5.5.1
		if doc.Header.Version != gedcom.Version551 {
			t.Errorf("Expected GEDCOM 5.5.1, got %v", doc.Header.Version)
		}

		// Should have at least 10 records (individuals, families, sources, etc.)
		if len(doc.Records) < 10 {
			t.Errorf("Expected at least 10 records, got %d", len(doc.Records))
		}

		t.Logf("Successfully parsed comprehensive GEDCOM 5.5.1: %d records, %d XRefs",
			len(doc.Records), len(doc.XRefMap))
	})
}

// Test various character encodings
func TestCharacterEncodings(t *testing.T) {
	testFiles := []struct {
		path        string
		description string
		encoding    gedcom.Encoding
		skipReason  string
	}{
		{
			path:        "../testdata/encoding/utf8-bom.ged",
			description: "UTF-8 with BOM (GEDCOM 5.5.5)",
			encoding:    gedcom.EncodingUTF8,
		},
		{
			path:        "../testdata/encoding/utf8-unicode.ged",
			description: "UTF-8 with extensive Unicode characters",
			encoding:    gedcom.EncodingUTF8,
		},
		{
			path:        "../testdata/encoding/utf16le.ged",
			description: "UTF-16 Little Endian with BOM",
			encoding:    gedcom.EncodingUNICODE,
			skipReason:  "UTF-16 decoding not yet implemented",
		},
		{
			path:        "../testdata/encoding/utf16be.ged",
			description: "UTF-16 Big Endian with BOM",
			encoding:    gedcom.EncodingUNICODE,
			skipReason:  "UTF-16 decoding not yet implemented",
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for %s", err, tt.description)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			// Verify encoding if not UTF-16 (which we skip)
			if tt.encoding == gedcom.EncodingUTF8 {
				if doc.Header.Encoding != gedcom.EncodingUTF8 {
					t.Errorf("Expected UTF-8 encoding, got %v", doc.Header.Encoding)
				}
			}

			t.Logf("Successfully parsed %s: %d records", tt.description, len(doc.Records))
		})
	}
}

// Test edge cases (CONT/CONC line continuation)
func TestEdgeCases(t *testing.T) {
	testFiles := []struct {
		path        string
		description string
	}{
		{
			path:        "../testdata/edge-cases/cont-conc.ged",
			description: "CONT/CONC line continuation tests",
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for %s", err, tt.description)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			t.Logf("Successfully parsed %s: %d records", tt.description, len(doc.Records))
		})
	}
}

// Test additional malformed file scenarios
func TestAdditionalMalformedFiles(t *testing.T) {
	testFiles := []struct {
		path        string
		description string
		shouldError bool
	}{
		{
			path:        "../testdata/malformed/circular-reference.ged",
			description: "Circular family relationships",
			shouldError: false, // Parser/decoder accepts, validator should catch
		},
		{
			path:        "../testdata/malformed/duplicate-xref.ged",
			description: "Duplicate cross-reference identifiers",
			shouldError: false, // Decoder accepts (last wins), may warn
		},
	}

	for _, tt := range testFiles {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("Test file not found: %s", tt.path)
				return
			}
			defer f.Close()

			doc, err := Decode(f)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Logf("Got error (may be acceptable): %v", err)
				}
				if doc != nil {
					t.Logf("Parsed %s: %d records, %d XRefs", tt.description, len(doc.Records), len(doc.XRefMap))
				}
			}
		})
	}
}
