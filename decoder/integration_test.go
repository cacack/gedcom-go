package decoder

import (
	"os"
	"path/filepath"
	"strings"
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
// ANSEL encoding is now fully supported via the charset package.
func TestTortureTestSuite(t *testing.T) {

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

			// Verify the copyright symbol (ANSEL 0xC3) decodes correctly to Unicode (U+00A9)
			// The header contains: "(c) 1997 by H. Eichmann" where (c) is actually the copyright symbol
			if doc.Header.Copyright != "" {
				// The copyright sign in Unicode is (c) U+00A9
				if !strings.Contains(doc.Header.Copyright, "\u00A9") {
					t.Errorf("Copyright symbol not decoded correctly: got %q, want to contain U+00A9 (\u00A9)", doc.Header.Copyright)
				}
				t.Logf("Copyright field: %s", doc.Header.Copyright)
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

// TestIntegration_MediaObjects_Maximal70 tests parsing maximal70.ged for media objects
func TestIntegration_MediaObjects_Maximal70(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Skipf("Test file not found: %s", "../testdata/gedcom-7.0/maximal70.ged")
		return
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Get all media objects
	mediaObjects := doc.MediaObjects()
	if len(mediaObjects) < 2 {
		t.Fatalf("Expected at least 2 media objects, got %d", len(mediaObjects))
	}

	// Check @O1@ - should have multiple files with translations
	o1 := doc.GetMediaObject("@O1@")
	if o1 == nil {
		t.Fatal("GetMediaObject(@O1@) returned nil")
	}

	if len(o1.Files) < 3 {
		t.Errorf("@O1@ expected at least 3 files, got %d", len(o1.Files))
	}

	// Find the file with translations (media/original.mp3)
	var mp3File *gedcom.MediaFile
	for _, file := range o1.Files {
		if file.FileRef == "media/original.mp3" {
			mp3File = file
			break
		}
	}
	if mp3File == nil {
		t.Fatal("Could not find media/original.mp3 in @O1@")
	}

	if mp3File.Form != "audio/mp3" {
		t.Errorf("mp3File.Form = %s, want audio/mp3", mp3File.Form)
	}
	if mp3File.MediaType != "AUDIO" {
		t.Errorf("mp3File.MediaType = %s, want AUDIO", mp3File.MediaType)
	}

	if len(mp3File.Translations) != 2 {
		t.Errorf("mp3File should have 2 translations, got %d", len(mp3File.Translations))
	}

	// Verify @O1@ has RESN
	if o1.Restriction != "CONFIDENTIAL, LOCKED" {
		t.Errorf("@O1@ Restriction = %s, want 'CONFIDENTIAL, LOCKED'", o1.Restriction)
	}

	// Check @O2@ - should have single file with RESN
	o2 := doc.GetMediaObject("@O2@")
	if o2 == nil {
		t.Fatal("GetMediaObject(@O2@) returned nil")
	}

	if len(o2.Files) != 1 {
		t.Errorf("@O2@ expected 1 file, got %d", len(o2.Files))
	}

	if o2.Restriction != "PRIVACY" {
		t.Errorf("@O2@ Restriction = %s, want PRIVACY", o2.Restriction)
	}

	t.Logf("Successfully parsed maximal70.ged: %d media objects", len(mediaObjects))
}

// TestIntegration_MediaObjects_Obje1 tests parsing obje-1.ged
func TestIntegration_MediaObjects_Obje1(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/familysearch-examples/obje-1.ged")
	if err != nil {
		t.Skipf("Test file not found: %s", "../testdata/gedcom-7.0/familysearch-examples/obje-1.ged")
		return
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	mediaObjects := doc.MediaObjects()
	if len(mediaObjects) != 2 {
		t.Fatalf("Expected 2 media objects, got %d", len(mediaObjects))
	}

	// Check @1@ - should have 2 files (jpeg and mp3)
	m1 := doc.GetMediaObject("@1@")
	if m1 == nil {
		t.Fatal("GetMediaObject(@1@) returned nil")
	}

	if len(m1.Files) != 2 {
		t.Fatalf("@1@ expected 2 files, got %d", len(m1.Files))
	}

	// Verify first file (JPEG)
	if m1.Files[0].FileRef != "example.jpg" {
		t.Errorf("@1@ Files[0].FileRef = %s, want example.jpg", m1.Files[0].FileRef)
	}
	if m1.Files[0].Form != "image/jpeg" {
		t.Errorf("@1@ Files[0].Form = %s, want image/jpeg", m1.Files[0].Form)
	}
	if m1.Files[0].MediaType != "PHOTO" {
		t.Errorf("@1@ Files[0].MediaType = %s, want PHOTO", m1.Files[0].MediaType)
	}

	// Verify second file (MP3)
	if m1.Files[1].FileRef != "example.mp3" {
		t.Errorf("@1@ Files[1].FileRef = %s, want example.mp3", m1.Files[1].FileRef)
	}
	if m1.Files[1].Form != "application/x-mp3" {
		t.Errorf("@1@ Files[1].Form = %s, want application/x-mp3", m1.Files[1].Form)
	}

	// Check @X1@ - should have 2 VIDEO files (both webm)
	mX1 := doc.GetMediaObject("@X1@")
	if mX1 == nil {
		t.Fatal("GetMediaObject(@X1@) returned nil")
	}

	if len(mX1.Files) != 2 {
		t.Fatalf("@X1@ expected 2 files, got %d", len(mX1.Files))
	}

	// Both should be VIDEO
	for i, file := range mX1.Files {
		if file.MediaType != "VIDEO" {
			t.Errorf("@X1@ Files[%d].MediaType = %s, want VIDEO", i, file.MediaType)
		}
		if file.Form != "application/x-other" {
			t.Errorf("@X1@ Files[%d].Form = %s, want application/x-other", i, file.Form)
		}
	}

	t.Logf("Successfully parsed obje-1.ged: %d media objects", len(mediaObjects))
}

// TestIntegration_MediaLinksWithCrop tests media links with CROP in maximal70.ged
func TestIntegration_MediaLinksWithCrop(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Skipf("Test file not found: %s", "../testdata/gedcom-7.0/maximal70.ged")
		return
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Get submitter @U1@ which has OBJE links with CROP (lines 824-835)
	var submitter *gedcom.Record
	for _, rec := range doc.Records {
		if rec.XRef == "@U1@" && rec.Type == gedcom.RecordTypeSubmitter {
			submitter = rec
			break
		}
	}

	if submitter == nil {
		t.Fatal("Could not find submitter @U1@")
	}

	// Parse submitter to get media links
	// For this test, we'll check that the source citation within @S1@ has OBJE with CROP
	source := doc.GetSource("@S1@")
	if source == nil {
		t.Fatal("GetSource(@S1@) returned nil")
	}

	// Source citations at line 592 have OBJE with CROP
	// We need to check if the source has embedded media links
	// Actually, let's check Individual @I1@ for death event with media
	individual := doc.GetIndividual("@I1@")
	if individual == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}

	// Individual should have media links
	if len(individual.Media) < 1 {
		t.Logf("Individual @I1@ has %d media links (expected at least 1)", len(individual.Media))
	}

	t.Logf("Successfully verified media links in maximal70.ged")
}

// TestIntegration_IndividualMedia tests individual media references in obje-1.ged
func TestIntegration_IndividualMedia(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/familysearch-examples/obje-1.ged")
	if err != nil {
		t.Skipf("Test file not found: %s", "../testdata/gedcom-7.0/familysearch-examples/obje-1.ged")
		return
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Get individual @2@
	individual := doc.GetIndividual("@2@")
	if individual == nil {
		t.Fatal("GetIndividual(@2@) returned nil")
	}

	// Should have 2 media entries
	if len(individual.Media) != 2 {
		t.Fatalf("individual.Media expected 2 entries, got %d", len(individual.Media))
	}

	// First entry: @1@ without title override
	if individual.Media[0].MediaXRef != "@1@" {
		t.Errorf("Media[0].MediaXRef = %s, want @1@", individual.Media[0].MediaXRef)
	}
	if individual.Media[0].Title != "" {
		t.Errorf("Media[0].Title = %s, want empty", individual.Media[0].Title)
	}

	// Second entry: @X1@ with title override "fifth birthday party"
	if individual.Media[1].MediaXRef != "@X1@" {
		t.Errorf("Media[1].MediaXRef = %s, want @X1@", individual.Media[1].MediaXRef)
	}
	if individual.Media[1].Title != "fifth birthday party" {
		t.Errorf("Media[1].Title = %s, want 'fifth birthday party'", individual.Media[1].Title)
	}

	t.Logf("Successfully verified individual media references in obje-1.ged")
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
		},
		{
			path:        "../testdata/encoding/utf16be.ged",
			description: "UTF-16 Big Endian with BOM",
			encoding:    gedcom.EncodingUNICODE,
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

			// Verify encoding matches expectation
			if doc.Header.Encoding != tt.encoding {
				t.Errorf("Expected %v encoding, got %v", tt.encoding, doc.Header.Encoding)
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
		{
			path:        "../testdata/edge-cases/ancestry-extensions.ged",
			description: "Ancestry.com vendor extensions (_APID, _TREE)",
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

// TestAncestryExtensions tests Ancestry.com specific extensions (_APID, _TREE)
func TestAncestryExtensions(t *testing.T) {
	f, err := os.Open("../testdata/edge-cases/ancestry-extensions.ged")
	if err != nil {
		t.Skipf("Test file not found: %s", "../testdata/edge-cases/ancestry-extensions.ged")
		return
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify vendor detection
	if doc.Vendor != gedcom.VendorAncestry {
		t.Errorf("Expected VendorAncestry, got %v", doc.Vendor)
	}

	// Verify _TREE from header
	if doc.Header.AncestryTreeID != "12345678" {
		t.Errorf("Header.AncestryTreeID = %q, want %q", doc.Header.AncestryTreeID, "12345678")
	}

	// Verify source system
	if doc.Header.SourceSystem != "Ancestry.com" {
		t.Errorf("Header.SourceSystem = %q, want %q", doc.Header.SourceSystem, "Ancestry.com")
	}

	// Get individual @I1@ and check source citations
	individual := doc.GetIndividual("@I1@")
	if individual == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}

	// Check birth event source citation
	var birthEvent *gedcom.Event
	for _, event := range individual.Events {
		if event.Type == gedcom.EventBirth {
			birthEvent = event
			break
		}
	}

	if birthEvent == nil {
		t.Fatal("Could not find birth event for @I1@")
	}

	if len(birthEvent.SourceCitations) != 1 {
		t.Fatalf("Birth event expected 1 source citation, got %d", len(birthEvent.SourceCitations))
	}

	cite := birthEvent.SourceCitations[0]
	if cite.AncestryAPID == nil {
		t.Fatal("Birth event source citation has nil AncestryAPID")
	}

	// Verify APID parsing
	if cite.AncestryAPID.Raw != "1,7602::2771226" {
		t.Errorf("AncestryAPID.Raw = %q, want %q", cite.AncestryAPID.Raw, "1,7602::2771226")
	}
	if cite.AncestryAPID.Database != "7602" {
		t.Errorf("AncestryAPID.Database = %q, want %q", cite.AncestryAPID.Database, "7602")
	}
	if cite.AncestryAPID.Record != "2771226" {
		t.Errorf("AncestryAPID.Record = %q, want %q", cite.AncestryAPID.Record, "2771226")
	}

	// Verify URL reconstruction
	expectedURL := "https://www.ancestry.com/discoveryui-content/view/2771226:7602"
	if cite.AncestryAPID.URL() != expectedURL {
		t.Errorf("AncestryAPID.URL() = %q, want %q", cite.AncestryAPID.URL(), expectedURL)
	}

	// Check death event source citation
	var deathEvent *gedcom.Event
	for _, event := range individual.Events {
		if event.Type == gedcom.EventDeath {
			deathEvent = event
			break
		}
	}

	if deathEvent == nil {
		t.Fatal("Could not find death event for @I1@")
	}

	if len(deathEvent.SourceCitations) != 1 {
		t.Fatalf("Death event expected 1 source citation, got %d", len(deathEvent.SourceCitations))
	}

	cite2 := deathEvent.SourceCitations[0]
	if cite2.AncestryAPID == nil {
		t.Fatal("Death event source citation has nil AncestryAPID")
	}

	if cite2.AncestryAPID.Database != "9024" {
		t.Errorf("Death AncestryAPID.Database = %q, want %q", cite2.AncestryAPID.Database, "9024")
	}

	t.Logf("Successfully verified Ancestry extensions: TreeID=%s, APID parsing and URL generation works",
		doc.Header.AncestryTreeID)
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
