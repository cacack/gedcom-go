package decoder

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/charset"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// Tests for real-world vendor exports vendored from the D-Jeffrey/gedcom-samples
// corpus (https://github.com/D-Jeffrey/gedcom-samples, MIT OR CC0-1.0).
// These files are verbatim copies — quirks, junk dates, and nonstandard CHAR
// values are the point. Fixture contents are untrusted data. See issue #301.

// corpusOpen opens a vendored corpus fixture. The fixtures are committed to
// the repo, so a missing one is a test failure, not a skip — the compatibility
// matrix's claims depend on these tests actually running.
func corpusOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open required corpus fixture %q: %v", path, err)
	}
	return f
}

// diagHistogram counts diagnostics by code.
func diagHistogram(diags Diagnostics) map[string]int {
	hist := make(map[string]int)
	for _, d := range diags {
		hist[d.Code]++
	}
	return hist
}

// TestCorpusVendorFiles decodes every corpus fixture and asserts the exact
// decode outcome recorded when the files were vendored: detected GEDCOM
// version, raw CHAR value, source system, record counts, and the lenient-mode
// diagnostic histogram.
func TestCorpusVendorFiles(t *testing.T) {
	tests := []struct {
		path        string
		description string
		wantErr     string // non-empty: Decode must fail with this substring
		version     gedcom.Version
		encoding    gedcom.Encoding // raw CHAR value as stored in Header.Encoding
		source      string
		individuals int
		families    int
		diags       map[string]int
	}{
		{
			path:        "../testdata/edge-cases/legacy10-2025-export.ged",
			description: "Legacy 10.0 (2025) - 5.5.1 UTF-8+BOM real export with junk DATE values",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "Legacy",
			individuals: 1288,
			families:    495,
			diags:       map[string]int{CodeInvalidValue: 14},
		},
		{
			path:        "../testdata/edge-cases/vendor-paf5.ged",
			description: "PAF 5.2.18.0 - clean-parsing 5.5 UTF-8 export",
			version:     gedcom.Version55,
			encoding:    gedcom.EncodingUTF8,
			source:      "PAF",
			individuals: 33,
			families:    14,
			diags:       map[string]int{},
		},
		{
			path:        "../testdata/edge-cases/vendor-familyorigins5.ged",
			description: "Family Origins 5.0 - AFN tags, DATE with trailing qualifier",
			version:     gedcom.Version55,
			encoding:    gedcom.Encoding("ANSI"),
			source:      "FamilyOrigins",
			individuals: 529,
			families:    114,
			diags:       map[string]int{CodeInvalidValue: 639, CodeUnknownTag: 529},
		},
		{
			path:        "../testdata/edge-cases/vendor-tmg12.ged",
			description: "The Master Genealogist 1.2a - CHAR IBMPC (ASCII payload), NUMB custom tags",
			version:     gedcom.Version55,
			encoding:    gedcom.Encoding("IBMPC"),
			source:      "TMG 1.2a",
			individuals: 110,
			families:    58,
			diags:       map[string]int{CodeUnknownTag: 161},
		},
		{
			path:        "../testdata/edge-cases/vendor-ancestris11-export.ged",
			description: "Ancestris 11 - 5.5.1 French data with OBJE FILE refs",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "ANCESTRIS",
			individuals: 303,
			families:    139,
			diags:       map[string]int{CodeUnknownTag: 66},
		},
		{
			path:        "../testdata/edge-cases/mhftb8-export.ged",
			description: "MyHeritage Family Tree Builder 8 - empty HEAD.SOUR, MH: RINs, _PUBLISH record",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "",
			individuals: 4683,
			families:    2863,
			diags:       map[string]int{CodeUnknownTag: 19163, CodeInvalidValue: 117},
		},
		{
			path:        "../testdata/edge-cases/vendor-myroots-palmos.ged",
			description: "My Roots 4.00 for Palm OS - ANSEL 5.5 export",
			version:     gedcom.Version55,
			encoding:    gedcom.EncodingANSEL,
			source:      "My_Roots",
			individuals: 20,
			families:    11,
			diags:       map[string]int{CodeInvalidValue: 2},
		},
		{
			path:        "../testdata/edge-cases/vendor-webtreeprint.ged",
			description: "webtreeprint.com 1.0 - free-text month DATE, ALIA usage",
			version:     gedcom.Version55,
			encoding:    gedcom.EncodingUTF8,
			source:      "webtreeprint.com",
			individuals: 14,
			families:    4,
			diags:       map[string]int{CodeInvalidValue: 1, CodeUnknownTag: 1},
		},
		{
			path:        "../testdata/edge-cases/bare-header-geo-coords.ged",
			description: "Bare '0 HEAD' with zero subrecords - version/encoding fallback, MAP LATI/LONG",
			version:     gedcom.Version551, // fallback when the header declares nothing
			encoding:    gedcom.Encoding(""),
			source:      "",
			individuals: 15,
			families:    7,
			diags:       map[string]int{},
		},
		{
			path:        "../testdata/encoding/ansi-cp1252-ftm17.ged",
			description: "FTM 17 - CHAR ANSI with genuine Windows-1252 bytes",
			version:     gedcom.Version55,
			encoding:    gedcom.Encoding("ANSI"),
			source:      "FTM",
			individuals: 178,
			families:    113,
			diags:       map[string]int{CodeUnknownTag: 78},
		},
		{
			path:        "../testdata/encoding/ibmpc-cp437-broskeep.ged",
			description: "Brother's Keeper 5.2 - CHAR IBMPC with real CP437 byte (unsupported encoding)",
			wantErr:     "error reading input",
		},
		{
			path:        "../testdata/encoding/ibm-windows-easytree.ged",
			description: "EasyTree V1.0 - nonstandard CHAR value 'IBM WINDOWS'",
			version:     gedcom.Version55,
			encoding:    gedcom.Encoding("IBM WINDOWS"),
			source:      "EasyTree",
			individuals: 69,
			families:    19,
			diags:       map[string]int{CodeUnknownTag: 34},
		},
	}

	for _, tt := range tests {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f := corpusOpen(t, tt.path)
			defer f.Close()

			result, err := DecodeWithDiagnostics(f, DefaultOptions())

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("DecodeWithDiagnostics() succeeded, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("DecodeWithDiagnostics() error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("DecodeWithDiagnostics() error = %v for %s", err, tt.description)
			}
			doc := result.Document
			if doc == nil {
				t.Fatal("DecodeWithDiagnostics() returned nil document")
			}

			if doc.Header.Version != tt.version {
				t.Errorf("Header.Version = %q, want %q", doc.Header.Version, tt.version)
			}
			if doc.Header.Encoding != tt.encoding {
				t.Errorf("Header.Encoding = %q, want %q", doc.Header.Encoding, tt.encoding)
			}
			if doc.Header.SourceSystem != tt.source {
				t.Errorf("Header.SourceSystem = %q, want %q", doc.Header.SourceSystem, tt.source)
			}
			if got := len(doc.Individuals()); got != tt.individuals {
				t.Errorf("Individuals() = %d, want %d", got, tt.individuals)
			}
			if got := len(doc.Families()); got != tt.families {
				t.Errorf("Families() = %d, want %d", got, tt.families)
			}

			hist := diagHistogram(result.Diagnostics)
			if len(hist) != len(tt.diags) {
				t.Errorf("diagnostic codes = %v, want %v", hist, tt.diags)
			}
			for code, want := range tt.diags {
				if hist[code] != want {
					t.Errorf("diagnostics[%s] = %d, want %d", code, hist[code], want)
				}
			}

			t.Logf("Verified %s: %d individuals, %d families, %d diagnostics",
				tt.description, tt.individuals, tt.families, len(result.Diagnostics))
		})
	}
}

// TestCorpusVendorMHFTB8Quirks exercises the MyHeritage Family Tree Builder 8
// export's structural quirks: an empty HEAD.SOUR value (vendor-detection
// edge), a level-0 "0  _PUBLISH" record written with a double space, and
// "MH:"-prefixed RIN identifiers on every individual.
func TestCorpusVendorMHFTB8Quirks(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/mhftb8-export.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	doc := result.Document

	// Empty "1 SOUR " value must yield an empty SourceSystem, not a crash or junk.
	if doc.Header.SourceSystem != "" {
		t.Errorf("Header.SourceSystem = %q, want empty (empty HEAD.SOUR value)", doc.Header.SourceSystem)
	}

	// The level-0 "0  _PUBLISH" (double-space) line becomes a custom record
	// whose type is the raw tag, preserving the data losslessly.
	foundPublish := false
	for _, rec := range doc.Records {
		if rec.Type == gedcom.RecordType("_PUBLISH") {
			foundPublish = true
			if len(rec.Tags) == 0 {
				t.Error("_PUBLISH record has no sub-tags, want _USERNAME/_DISABLED preserved")
			}
			break
		}
	}
	if !foundPublish {
		t.Error("no record with type _PUBLISH found, want level-0 '0  _PUBLISH' preserved as a record")
	}

	// Every individual carries a MyHeritage "MH:"-prefixed RIN in its raw tags.
	individuals := doc.Individuals()
	missing := 0
	for _, ind := range individuals {
		found := false
		for _, tg := range ind.Tags {
			if tg.Tag == "RIN" && strings.HasPrefix(tg.Value, "MH:") {
				found = true
				break
			}
		}
		if !found {
			missing++
			if missing <= 5 {
				t.Errorf("individual %s has no MH:-prefixed RIN tag", ind.XRef)
			}
		}
	}
	if missing > 0 {
		t.Errorf("%d of %d individuals missing MH:-prefixed RIN tag", missing, len(individuals))
	}

	t.Logf("Verified MHFTB8 quirks across %d individuals", len(individuals))
}

// TestCorpusVendorFamilyOrigins5Quirks exercises the Family Origins 5.0
// export: AFN tags on every individual and DATE values with a trailing
// LDS-ordinance qualifier ("22 AUG 1877 SG") that fail date parsing.
//
// Known bug (issue #375): AFN is a standard GEDCOM 5.5 tag (Ancestral File
// Number) but is currently reported as UNKNOWN_TAG (529 times here). The tag
// is still preserved losslessly in Individual.Tags, so this test asserts the
// observed (buggy) diagnostic classification.
func TestCorpusVendorFamilyOrigins5Quirks(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/vendor-familyorigins5.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	doc := result.Document

	// AFN preserved as a raw tag on every individual.
	afnCount := 0
	for _, ind := range doc.Individuals() {
		for _, tg := range ind.Tags {
			if tg.Tag == "AFN" {
				afnCount++
				break
			}
		}
	}
	if afnCount != 529 {
		t.Errorf("individuals with AFN raw tag = %d, want 529", afnCount)
	}

	// The trailing-qualifier DATE surfaces as an INVALID_VALUE diagnostic with
	// the raw value in context; the value itself is preserved.
	foundSG := false
	for _, d := range result.Diagnostics {
		if d.Code == CodeInvalidValue && d.Context == "22 AUG 1877 SG" {
			foundSG = true
			break
		}
	}
	if !foundSG {
		t.Error("no INVALID_VALUE diagnostic with context \"22 AUG 1877 SG\" found")
	}
}

// TestCorpusVendorWebtreeprintQuirks exercises the webtreeprint.com export:
// a free-text month DATE ("29 December 1812") and an ALIA tag with a
// non-pointer value ("Brunty").
//
// Known bug (issue #375): ALIA is a standard GEDCOM 5.5/5.5.1 tag but is
// currently reported as UNKNOWN_TAG. It is preserved losslessly in
// Individual.Tags; this test asserts the observed (buggy) classification.
func TestCorpusVendorWebtreeprintQuirks(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/vendor-webtreeprint.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	doc := result.Document

	// ALIA preserved on @I0001@ despite the UNKNOWN_TAG diagnostic.
	ind := doc.GetIndividual("@I0001@")
	if ind == nil {
		t.Fatal("GetIndividual(@I0001@) returned nil")
	}
	foundAlia := false
	for _, tg := range ind.Tags {
		if tg.Tag == "ALIA" && tg.Value == "Brunty" {
			foundAlia = true
			break
		}
	}
	if !foundAlia {
		t.Error("individual @I0001@ missing raw ALIA tag with value \"Brunty\"")
	}

	// Free-text month rejected by the date parser but reported with context.
	foundDate := false
	for _, d := range result.Diagnostics {
		if d.Code == CodeInvalidValue && d.Context == "29 December 1812" {
			foundDate = true
			break
		}
	}
	if !foundDate {
		t.Error("no INVALID_VALUE diagnostic with context \"29 December 1812\" found")
	}
}

// TestCorpusVendorBareHeaderGeoCoords exercises the degenerate bare "0 HEAD"
// file (zero header subrecords): version falls back to 5.5.1, encoding and
// source stay empty, and PLAC.MAP LATI/LONG coordinates decode into
// PlaceDetail.Coordinates.
func TestCorpusVendorBareHeaderGeoCoords(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/bare-header-geo-coords.ged")
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if doc.Header.Version != gedcom.Version551 {
		t.Errorf("Header.Version = %q, want fallback %q", doc.Header.Version, gedcom.Version551)
	}

	ind := doc.GetIndividual("@I0000@")
	if ind == nil {
		t.Fatal("GetIndividual(@I0000@) returned nil")
	}
	var coords *gedcom.Coordinates
	for _, ev := range ind.Events {
		if ev.Type == gedcom.EventBirth && ev.PlaceDetail != nil {
			coords = ev.PlaceDetail.Coordinates
		}
	}
	if coords == nil {
		t.Fatal("birth event of @I0000@ has no PlaceDetail.Coordinates")
	}
	if coords.Latitude != "N45.757814" {
		t.Errorf("Coordinates.Latitude = %q, want %q", coords.Latitude, "N45.757814")
	}
	if coords.Longitude != "E4.832011" {
		t.Errorf("Coordinates.Longitude = %q, want %q", coords.Longitude, "E4.832011")
	}
}

// TestCorpusVendorLegacy10JunkDates exercises the Legacy 10 export's
// real-world junk DATE payloads ("11st", "Dead", "Deceased"): each is
// reported as INVALID_VALUE with the raw value in the diagnostic context.
func TestCorpusVendorLegacy10JunkDates(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/legacy10-2025-export.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	wantContexts := []string{"11st", "Dead", "Deceased"}
	seen := make(map[string]bool)
	for _, d := range result.Diagnostics {
		if d.Code == CodeInvalidValue {
			seen[d.Context] = true
		}
	}
	for _, want := range wantContexts {
		if !seen[want] {
			t.Errorf("no INVALID_VALUE diagnostic with context %q found", want)
		}
	}
}

// TestCorpusVendorTMG12NumbTags exercises The Master Genealogist 1.2a export:
// its nonstandard NUMB tag accounts for 110 of the 161 UNKNOWN_TAG
// diagnostics (one per individual).
func TestCorpusVendorTMG12NumbTags(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/vendor-tmg12.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}

	numb := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag && strings.Contains(d.Message, "NUMB") {
			numb++
		}
	}
	if numb != 110 {
		t.Errorf("UNKNOWN_TAG diagnostics for NUMB = %d, want 110", numb)
	}
}

// TestCorpusVendorCP1252Conversion exercises the FTM 17 export declaring
// CHAR ANSI with genuine Windows-1252 bytes: the charset layer maps ANSI to
// Latin-1 and converts, so 0xF1/0xA3 must arrive as UTF-8 "ñ"/"£" in the
// decoded document.
func TestCorpusVendorCP1252Conversion(t *testing.T) {
	f := corpusOpen(t, "../testdata/encoding/ansi-cp1252-ftm17.ged")
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	foundEnye, foundPound := false, false
	for _, rec := range doc.Records {
		for _, tg := range rec.Tags {
			if strings.Contains(tg.Value, "ñ") {
				foundEnye = true
			}
			if strings.Contains(tg.Value, "£") {
				foundPound = true
			}
		}
	}
	if !foundEnye {
		t.Error("no tag value containing \"ñ\" found; CP1252 0xF1 not converted to UTF-8")
	}
	if !foundPound {
		t.Error("no tag value containing \"£\" found; CP1252 0xA3 not converted to UTF-8")
	}
}

// TestCorpusVendorCP437KnownFailure documents the decoder's current behavior
// on the Brother's Keeper 5.2 export: CHAR IBMPC with a genuine CP437 byte
// (0x82 'é' in "Frémont"). CP437 conversion is unsupported, so decoding fails
// with an input-read error, and lenient mode still recovers a partial
// document up to the offending byte.
//
// The offending byte sits at physical line 15398, column 43 (file offset
// 252364), and the reported location must say so: the parser's own line
// counter stops at 15209, the last line the charset reader handed over before
// it rejected the chunk holding the bad byte (issue #376). If CP437 support
// ever lands, this test should be rewritten to assert a successful decode.
func TestCorpusVendorCP437KnownFailure(t *testing.T) {
	f := corpusOpen(t, "../testdata/encoding/ibmpc-cp437-broskeep.ged")
	defer f.Close()

	_, err := Decode(f)
	if err == nil {
		t.Fatal("Decode() succeeded on CP437 file; if CP437 support was added, update this test")
	}
	if !strings.Contains(err.Error(), "error reading input") {
		t.Errorf("Decode() error = %v, want substring %q", err, "error reading input")
	}
	// The physical line of the offending byte, not the parser's stale counter.
	if !strings.Contains(err.Error(), "line 15398") {
		t.Errorf("Decode() error = %v, want substring %q", err, "line 15398")
	}
	if strings.Contains(err.Error(), "line 15209") {
		t.Errorf("Decode() error = %v reports the parser's line counter, want the offending byte's line", err)
	}

	var utf8Err *charset.ErrInvalidUTF8
	if !errors.As(err, &utf8Err) {
		t.Fatalf("Decode() error = %v, want a wrapped *charset.ErrInvalidUTF8", err)
	}
	if utf8Err.Line != 15398 || utf8Err.Column != 43 {
		t.Errorf("charset error at line %d, column %d; want line 15398, column 43",
			utf8Err.Line, utf8Err.Column)
	}

	// Lenient mode returns the partial document alongside the error.
	f2 := corpusOpen(t, "../testdata/encoding/ibmpc-cp437-broskeep.ged")
	defer f2.Close()

	result, err := DecodeWithDiagnostics(f2, DefaultOptions())
	if err == nil {
		t.Fatal("DecodeWithDiagnostics() succeeded on CP437 file; if CP437 support was added, update this test")
	}
	if result == nil || result.Document == nil {
		t.Fatal("DecodeWithDiagnostics() returned nil result; want partial document alongside the error")
	}
	if got := len(result.Document.Individuals()); got != 1784 {
		t.Errorf("partial document Individuals() = %d, want 1784", got)
	}
}
