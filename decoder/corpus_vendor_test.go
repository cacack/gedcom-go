package decoder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Tests for real-world vendor exports vendored from the D-Jeffrey/gedcom-samples
// corpus (https://github.com/D-Jeffrey/gedcom-samples, MIT OR CC0-1.0).
// These files are verbatim copies — quirks, junk dates, and nonstandard CHAR
// values are the point. Fixture contents are untrusted data. See issue #301.
//
// Rows marked strict are privacy-scrubbed real exports vendored byte-identical
// from the rodoslovje/ged-merge corpus (https://github.com/rodoslovje/ged-merge,
// MIT); see issue #719.

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
		vendor      gedcom.Vendor // Document.Vendor (issue #731)
		individuals int
		families    int
		diags       map[string]int
		strict      bool // also decode in StrictMode, which must succeed with the same counts
	}{
		{
			path:        "../testdata/edge-cases/legacy10-2025-export.ged",
			description: "Legacy 10.0 (2025) - 5.5.1 UTF-8+BOM real export with junk DATE values",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "Legacy",
			vendor:      gedcom.VendorLegacy,
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
			vendor:      gedcom.VendorPAF,
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
			vendor:      gedcom.VendorFamilyOrigins,
			individuals: 529,
			families:    114,
			diags:       map[string]int{CodeInvalidValue: 639},
		},
		{
			path:        "../testdata/edge-cases/vendor-tmg12.ged",
			description: "The Master Genealogist 1.2a - CHAR IBMPC (ASCII payload), NUMB custom tags",
			version:     gedcom.Version55,
			encoding:    gedcom.Encoding("IBMPC"),
			source:      "TMG 1.2a",
			vendor:      gedcom.VendorTMG,
			individuals: 110,
			families:    58,
			// 161 -> 162 for issue #449: buildHeader now reports unknown
			// header tags, and this file's header has one -- "1 TIME 14:54"
			// beside "1 DATE" rather than under it (HEAD.DATE.TIME). It is
			// the only header tag the change flags anywhere in the corpus.
			// 162 -> 163 for issue #724: the nonstandard level-0 record
			// "0 @R1@ TITL" is reported as an unknown record type.
			diags: map[string]int{CodeUnknownTag: 163},
		},
		{
			path:        "../testdata/edge-cases/vendor-ancestris11-export.ged",
			description: "Ancestris 11 - 5.5.1 French data with OBJE FILE refs",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "ANCESTRIS",
			vendor:      gedcom.VendorAncestris,
			individuals: 303,
			families:    139,
			// 12 individual EVEN structures are decoded rather than flagged
			// (issue #378); one of them nests a NOTE under OBJE, which
			// parseMediaLink still reports. Net: 66 - 12 + 1 = 55.
			//
			// 55 -> 42 for issue #402: parseAttribute now reads the whole
			// EVENT_DETAIL instead of NOTE/AGE/DATE/PLAC/SOUR only, so the
			// file's 13 "2 CAUS" lines under attributes stop being flagged.
			// 55 - 13 CAUS = 42, and the histogram is now a single entry:
			// 42 NOTE, every one of them under an inline OBJE (issue #470).
			// This file contains no FACT line at all -- the 13 were always
			// false positives on other attribute tags, which is why #402 had
			// to be its own issue.
			//
			// 42 -> 0 for issue #472: parseMediaLink now decodes NOTE into
			// MediaLink.NoteXRefs/InlineNotes instead of letting it fall
			// through to default. 42 - 42 NOTE = 0, measured from the
			// diagnostic histogram rather than assumed: all 42 really were
			// the `2 NOTE` under `1 OBJE` path (#470), so nothing remains.
			// The entry is dropped rather than set to 0 because the table
			// asserts len(hist) == len(tt.diags).
			diags: map[string]int{},
		},
		{
			path:        "../testdata/edge-cases/mhftb8-export.ged",
			description: "MyHeritage Family Tree Builder 8 - empty HEAD.SOUR, MH: RINs, _PUBLISH record",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "",
			vendor:      gedcom.VendorMyHeritage,
			individuals: 4683,
			families:    2863,
			// UnknownTag is unchanged by issue #378 because the counts
			// cancel: the 12 individual EVEN diagnostics are gone, and each
			// of those now-walked subtrees contains a "2 RIN MH:..." that is
			// reported in their place (#375 recognizes RIN at INDI level 1,
			// but the subordinate-structure table still flags it).
			diags: map[string]int{CodeUnknownTag: 14480, CodeInvalidValue: 117},
		},
		{
			path:        "../testdata/edge-cases/vendor-myroots-palmos.ged",
			description: "My Roots 4.00 for Palm OS - ANSEL 5.5 export",
			version:     gedcom.Version55,
			encoding:    gedcom.EncodingANSEL,
			source:      "My_Roots",
			vendor:      gedcom.VendorMyRoots,
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
			vendor:      gedcom.VendorWebTreePrint,
			individuals: 14,
			families:    4,
			diags:       map[string]int{CodeInvalidValue: 1},
		},
		{
			path:        "../testdata/edge-cases/bare-header-geo-coords.ged",
			description: "Bare '0 HEAD' with zero subrecords - version/encoding fallback, MAP LATI/LONG",
			version:     gedcom.Version551, // fallback when the header declares nothing
			encoding:    gedcom.Encoding(""),
			source:      "",
			// The header has no subordinates at all, so nothing names the
			// producer; this is the one real export detection cannot cover.
			vendor:      gedcom.VendorUnknown,
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
			vendor:      gedcom.VendorAncestry,
			individuals: 178,
			families:    113,
			// All 78 UNKNOWN_TAG diagnostics are gone: 67 individual EVEN
			// structures are decoded (#378) and 11 ALIA are recognized as
			// standard (#375). The entry is dropped rather than set to 0
			// because the table asserts len(hist) == len(tt.diags).
			diags: map[string]int{},
		},
		{
			path:        "../testdata/encoding/ibmpc-cp437-broskeep.ged",
			description: "Brother's Keeper 5.2 - CHAR IBMPC with real CP437 byte",
			version:     gedcom.Version55,
			encoding:    gedcom.Encoding("IBMPC"),
			source:      "BROSKEEP",
			vendor:      gedcom.VendorBrothersKeeper,
			individuals: 2145,
			families:    1042,
			// The one CP437 byte decodes as 'é' (#735); QUAY is written
			// directly under events (see misplacedTags).
			diags: map[string]int{CodeInvalidValue: 1, CodeUnknownTag: 1206},
		},
		{
			path:        "../testdata/encoding/ibm-windows-easytree.ged",
			description: "EasyTree V1.0 - nonstandard CHAR value 'IBM WINDOWS'",
			version:     gedcom.Version55,
			encoding:    gedcom.Encoding("IBM WINDOWS"),
			source:      "EasyTree",
			vendor:      gedcom.VendorEasyTree,
			individuals: 69,
			families:    19,
			// 34 -> 41 for issue #724: the seven nonstandard level-0
			// "0 @Cn@ CSTA" records are reported as unknown record types.
			diags: map[string]int{CodeUnknownTag: 41},
		},
		{
			path:        "../testdata/edge-cases/synium-7.0-utf8.ged",
			description: "MacFamilyTree 11.2.3 - real desktop GEDCOM 7.0.3 export, GEDC.FORM, mixed-case months",
			version:     gedcom.Version70,
			encoding:    gedcom.EncodingUTF8,
			source:      "SyniumFamilyTree",
			vendor:      gedcom.VendorMacFamilyTree,
			individuals: 88,
			families:    108,
			// UNKNOWN_TAG: CREA/CHAN under events and names, FILN, AGNC,
			// DATE and PLAC directly on SOUR records, SECG under NAME, RACE.
			// INVALID_VALUE: "Abt."/"Bef."/"about" qualifiers and a full
			// month name ("August 1978"); abbreviated months in mixed case
			// ("10 Dec 1868") parse cleanly.
			diags:  map[string]int{CodeUnknownTag: 470, CodeInvalidValue: 19},
			strict: true,
		},
		{
			path:        "../testdata/edge-cases/reunion-5.5.1-utf8.ged",
			description: "Reunion 14 - 5.5.1 UTF-8 with classic-Mac CR-only line endings",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "Reunion",
			vendor:      gedcom.VendorReunion,
			individuals: 377,
			families:    284,
			diags:       map[string]int{CodeUnknownTag: 1, CodeInvalidValue: 1},
			strict:      true,
		},
		{
			path:        "../testdata/edge-cases/geneanet-5.5.1-utf8.ged",
			description: "Geneanet - 5.5.1 UTF-8 export, ASSO with 5.5-style TYPE",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "Geneanet",
			vendor:      gedcom.VendorGeneanet,
			individuals: 195,
			families:    71,
			diags:       map[string]int{CodeUnknownTag: 5},
			strict:      true,
		},
		{
			path:        "../testdata/encoding/geneanet-5.5.1-w1250.ged",
			description: "Geneanet - CHAR ANSI over Windows-1250 bytes (decoded as Windows-1252; known gap)",
			version:     gedcom.Version551,
			encoding:    gedcom.Encoding("ANSI"),
			source:      "Geneanet",
			vendor:      gedcom.VendorGeneanet,
			individuals: 270,
			families:    176,
			diags:       map[string]int{CodeUnknownTag: 1},
			strict:      true,
		},
		{
			path:        "../testdata/edge-cases/webtrees-5.5.1-utf8.ged",
			description: "webtrees 2.1.27 - 5.5.1 UTF-8 export, RESN on SOUR, non-numeric MAP coordinates",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "webtrees",
			vendor:      gedcom.VendorWebtrees,
			individuals: 110,
			families:    68,
			diags:       map[string]int{CodeUnknownTag: 2, CodeInvalidValue: 4},
			strict:      true,
		},
		{
			path:        "../testdata/edge-cases/familyhistorian-5.5.1-utf8.ged",
			description: "Family Historian 7.0.24 - 5.5.1 UTF-8+BOM export, address parts directly under RESI",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "FAMILY_HISTORIAN",
			vendor:      gedcom.VendorFamilyHistorian,
			individuals: 198,
			families:    173,
			diags:       map[string]int{CodeUnknownTag: 6},
			strict:      true,
		},
		{
			path:        "../testdata/encoding/familyhistorian-5.5.1-w1250.ged",
			description: "Family Historian 7.0.24 - CHAR ANSI over Windows-1250 bytes (decoded as Windows-1252; known gap)",
			version:     gedcom.Version551,
			encoding:    gedcom.Encoding("ANSI"),
			source:      "FAMILY_HISTORIAN",
			vendor:      gedcom.VendorFamilyHistorian,
			individuals: 208,
			families:    62,
			// All 62 are the standard 5.5.1 FAM.RIN ("1 RIN MH:F2"), which
			// parseFamily does not recognize yet.
			diags:  map[string]int{CodeUnknownTag: 62},
			strict: true,
		},
		{
			path:        "../testdata/encoding/brotherskeeper-5.5.1-w1250.ged",
			description: "Brother's Keeper 7.5.4 - CHAR ANSI over Windows-1250 bytes, placeholder DATEs like \"__.<>.1680\"",
			version:     gedcom.Version551,
			encoding:    gedcom.Encoding("ANSI"),
			source:      "BROSKEEP",
			vendor:      gedcom.VendorBrothersKeeper,
			individuals: 257,
			families:    206,
			diags:       map[string]int{CodeInvalidValue: 103},
			strict:      true,
		},
		{
			path:        "../testdata/edge-cases/ancestry-5.5.1-utf8.ged",
			description: "Ancestry.com 2025.08 - 5.5.1 UTF-8 export, full month names, DATE/PLAC on OBJE records",
			version:     gedcom.Version551,
			encoding:    gedcom.EncodingUTF8,
			source:      "Ancestry.com Family Trees",
			vendor:      gedcom.VendorAncestry,
			individuals: 12,
			families:    13,
			diags:       map[string]int{CodeUnknownTag: 15, CodeInvalidValue: 19},
			strict:      true,
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
			if doc.Vendor != tt.vendor {
				t.Errorf("Vendor = %q, want %q", doc.Vendor, tt.vendor)
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

			if tt.strict {
				fs := corpusOpen(t, tt.path)
				defer fs.Close()
				sres, err := DecodeWithDiagnostics(fs, &DecodeOptions{StrictMode: true})
				if err != nil {
					t.Fatalf("strict DecodeWithDiagnostics() error = %v", err)
				}
				sIndi, sFam := len(sres.Document.Individuals()), len(sres.Document.Families())
				if sIndi != tt.individuals || sFam != tt.families {
					t.Errorf("strict decode: %d individuals, %d families; want %d, %d",
						sIndi, sFam, tt.individuals, tt.families)
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
// AFN is a standard GEDCOM 5.5 tag (Ancestral File Number), so it must be
// preserved in Individual.Tags without any UNKNOWN_TAG diagnostic — that code
// is reserved for genuinely nonstandard tags (issue #375).
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

	// AFN is standard: none of those 529 tags may produce an UNKNOWN_TAG.
	// (TestCorpusVendorFiles pins the file-wide count; this scopes it to AFN so
	// an unrelated regression does not blame the wrong tag.)
	afnUnknown := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag && strings.Contains(d.Message, "AFN") {
			afnUnknown++
		}
	}
	if afnUnknown != 0 {
		t.Errorf("UNKNOWN_TAG diagnostics for AFN = %d, want 0 (AFN is a standard GEDCOM 5.5 tag)", afnUnknown)
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
// ALIA is a standard GEDCOM 5.5/5.5.1 tag, so it is preserved losslessly in
// Individual.Tags and must not be reported as UNKNOWN_TAG (issue #375) — even
// with a non-pointer payload, which is a payload-grammar matter, not an
// unknown tag.
func TestCorpusVendorWebtreeprintQuirks(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/vendor-webtreeprint.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	doc := result.Document

	// ALIA preserved on @I0001@ and classified as a known tag.
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
	aliaUnknown := 0
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag && strings.Contains(d.Message, "ALIA") {
			aliaUnknown++
		}
	}
	if aliaUnknown != 0 {
		t.Errorf("UNKNOWN_TAG diagnostics for ALIA = %d, want 0 (ALIA is a standard GEDCOM 5.5 tag)", aliaUnknown)
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

// TestCorpusVendorCP1252C1Range decodes a CHAR ANSI file whose text uses
// Windows-1252 bytes in 0x80-0x9F: curly quotes, dashes and the euro sign
// must arrive as those characters, not as C1 controls, and the undefined byte
// 0x81 must survive as U+0081 rather than U+FFFD (#564). The FTM 17 fixture
// above has no byte in that range, so it cannot catch either defect.
func TestCorpusVendorCP1252C1Range(t *testing.T) {
	f := corpusOpen(t, "../testdata/encoding/ansi-cp1252-c1-range.ged")
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("@I1@ not decoded")
	}
	if len(indi.Names) != 1 || indi.Names[0].Full != "Seán /O’Brien/" {
		t.Errorf("Names = %+v, want one name \"Seán /O’Brien/\"", indi.Names)
	}
	wantNote := "Lived 1850–1900; paid €20 for “the farm” — and \u0081kept\u0081."
	if len(indi.InlineNotes) != 1 || indi.InlineNotes[0] != wantNote {
		t.Errorf("InlineNotes = %q, want [%q]", indi.InlineNotes, wantNote)
	}
}

// TestCorpusVendorCP437 decodes the Brother's Keeper 5.2 export, CHAR IBMPC
// with a genuine CP437 byte (0x82 'é' in "Frémont" at line 15398), as CP437
// (#735). Before, CP437 was unsupported and the file was read as UTF-8: strict
// decoding failed at that byte and lenient mode replaced it with U+FFFD.
func TestCorpusVendorCP437(t *testing.T) {
	const want = "Was elected in 1856 over John C. Frémont and Millard Fillmore by a popular"
	for _, strict := range []bool{true, false} {
		f := corpusOpen(t, "../testdata/encoding/ibmpc-cp437-broskeep.ged")
		result, err := DecodeWithDiagnostics(f, &DecodeOptions{StrictMode: strict})
		f.Close()
		if err != nil {
			t.Fatalf("StrictMode=%v: DecodeWithDiagnostics() error = %v", strict, err)
		}
		if got := len(result.Document.Individuals()); got != 2145 {
			t.Errorf("StrictMode=%v: Individuals() = %d, want all 2145", strict, got)
		}
		for _, d := range result.Diagnostics {
			if d.Code == CodeInvalidEncoding {
				t.Errorf("StrictMode=%v: unexpected %s diagnostic: %+v", strict, CodeInvalidEncoding, d)
			}
		}
		if !hasTagValue(result.Document, "NOTE", want) {
			t.Errorf("StrictMode=%v: no NOTE line %q; the CP437 byte 0x82 was not decoded as 'é'", strict, want)
		}
	}
}

// TestCorpusVendorMacFamilyTree70Quirks exercises the MacFamilyTree 11.2.3
// GEDCOM 7.0.3 export, the only real desktop 7.0 export in the corpus: the
// 5.5.1-style "2 FORM LINEAGE-LINKED" under GEDC (no such substructure in 7.0)
// is tolerated silently and kept in Header.Tags, mixed-case abbreviated months
// parse, nonstandard qualifiers ("Abt.") are flagged with the raw value, and
// the undeclared _FID extension (no SCHMA) survives as a raw tag.
func TestCorpusVendorMacFamilyTree70Quirks(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/synium-7.0-utf8.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	doc := result.Document

	if doc.Header.DeclaredVersion != gedcom.Version70 {
		t.Errorf("Header.DeclaredVersion = %q, want %q", doc.Header.DeclaredVersion, gedcom.Version70)
	}

	// GEDC.FORM preserved under GEDC, with no diagnostic on its line (9).
	foundForm := false
	for i, tg := range doc.Header.Tags {
		if tg.Level == 1 && tg.Tag == "GEDC" {
			for _, sub := range doc.Header.Tags[i+1:] {
				if sub.Level < 2 {
					break
				}
				if sub.Tag == "FORM" && sub.Value == "LINEAGE-LINKED" {
					foundForm = true
				}
			}
		}
	}
	if !foundForm {
		t.Error("HEAD.GEDC.FORM LINEAGE-LINKED not preserved in Header.Tags")
	}
	for _, d := range result.Diagnostics {
		if d.Line == 9 {
			t.Errorf("unexpected diagnostic on the GEDC.FORM line: %+v", d)
		}
	}

	ind := doc.GetIndividual("@55280784@")
	if ind == nil {
		t.Fatal("GetIndividual(@55280784@) returned nil")
	}

	// Mixed-case month "10 Dec 1868" parses like "10 DEC 1868".
	var birth *gedcom.Date
	for _, ev := range ind.Events {
		if ev.Type == gedcom.EventBirth {
			birth = ev.ParsedDate
		}
	}
	if birth == nil || birth.Day != 10 || birth.Month != 12 || birth.Year != 1868 {
		t.Errorf("birth ParsedDate = %+v, want 10 Dec 1868", birth)
	}

	foundAbt := false
	for _, d := range result.Diagnostics {
		if d.Code == CodeInvalidValue && d.Context == "Abt. 1790" {
			foundAbt = true
			break
		}
	}
	if !foundAbt {
		t.Error("no INVALID_VALUE diagnostic with context \"Abt. 1790\" found")
	}

	foundFID := false
	for _, tg := range ind.Tags {
		if tg.Tag == "_FID" && tg.Value == "GJMK-JZG" {
			foundFID = true
			break
		}
	}
	if !foundFID {
		t.Error("individual @55280784@ missing raw _FID tag with value \"GJMK-JZG\"")
	}
}

// TestCorpusVendorReunionCROnly pins that the Reunion 14 export's CR-only
// line endings are counted as line terminators: its single nonstandard tag
// (CLAW) is reported at physical line 3769.
func TestCorpusVendorReunionCROnly(t *testing.T) {
	f := corpusOpen(t, "../testdata/edge-cases/reunion-5.5.1-utf8.ged")
	defer f.Close()

	result, err := DecodeWithDiagnostics(f, DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	for _, d := range result.Diagnostics {
		if d.Code == CodeUnknownTag {
			if d.Line != 3769 || !strings.Contains(d.Message, "CLAW") {
				t.Errorf("UNKNOWN_TAG at line %d (%s), want CLAW at line 3769", d.Line, d.Message)
			}
		}
	}
}

// TestCorpusVendorANSIOver1250KnownGap documents the decoder's current
// behavior on exports declaring CHAR ANSI over Windows-1250 bytes (Geneanet,
// Family Historian 7, Brother's Keeper 7). ANSI is decoded as Windows-1252,
// so letters the two code pages share (š, ž, Š, Ž) arrive intact, but
// 0xE8 Windows-1250 "č" arrives as "è" with no diagnostic. If Windows-1250
// support lands, this test should be rewritten to expect "č".
func TestCorpusVendorANSIOver1250KnownGap(t *testing.T) {
	tests := []struct {
		path string
		xref string
		got  string // current, Windows-1252 reading
	}{
		{"../testdata/encoding/geneanet-5.5.1-w1250.ged", "@I3@", "Matevž2 /Logariè/"},
		{"../testdata/encoding/familyhistorian-5.5.1-w1250.ged", "@I1@", "Vinko3 /Jenkoiè/"},
		{"../testdata/encoding/brotherskeeper-5.5.1-w1250.ged", "@I1@", "Marjeta2 /Oblakiè/"},
	}
	for _, tt := range tests {
		t.Run(filepath.Base(tt.path), func(t *testing.T) {
			f := corpusOpen(t, tt.path)
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			ind := doc.GetIndividual(tt.xref)
			if ind == nil || len(ind.Names) == 0 {
				t.Fatalf("%s has no decoded name", tt.xref)
			}
			if ind.Names[0].Full != tt.got {
				t.Errorf("%s name = %q, want current Windows-1252 reading %q; if Windows-1250 support landed, update this test",
					tt.xref, ind.Names[0].Full, tt.got)
			}
		})
	}
}

// hasTagValue reports whether any record in doc has a tag named tag whose own
// value (without its CONT/CONC lines) is value.
func hasTagValue(doc *gedcom.Document, tag, value string) bool {
	for _, rec := range doc.Records {
		for _, t := range rec.Tags {
			if t.Tag == tag && t.Value == value {
				return true
			}
		}
	}
	return false
}
