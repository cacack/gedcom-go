package gedcomgo

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/converter"
	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/encoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
	gedcomtesting "github.com/cacack/gedcom-go/v2/gedcom/testing"
	"github.com/cacack/gedcom-go/v2/validator"
)

// Test GEDCOM content for basic tests.
const testGedcomMinimal = `0 HEAD
1 SOUR TestSoftware
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
1 SEX M
1 BIRT
2 DATE 15 JAN 1850
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR
`

// GEDCOM with validation issues for testing.
const testGedcomWithIssues = `0 HEAD
1 SOUR TestSoftware
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 FAMC @F999@
0 @F1@ FAM
0 TRLR
`

func TestDecode(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantErr      bool
		wantIndivs   int
		wantFamilies int
		checkVersion gedcom.Version
	}{
		{
			name:         "minimal valid GEDCOM",
			input:        testGedcomMinimal,
			wantErr:      false,
			wantIndivs:   2,
			wantFamilies: 1,
			checkVersion: gedcom.Version551,
		},
		{
			name: "GEDCOM 5.5",
			input: `0 HEAD
1 SOUR Test
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Test /Person/
0 TRLR
`,
			wantErr:      false,
			wantIndivs:   1,
			wantFamilies: 0,
			checkVersion: gedcom.Version55,
		},
		{
			name: "GEDCOM 7.0",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Test /Person/
0 TRLR
`,
			wantErr:      false,
			wantIndivs:   1,
			wantFamilies: 0,
			checkVersion: gedcom.Version70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if got := len(doc.Individuals()); got != tt.wantIndivs {
				t.Errorf("Individuals count = %d, want %d", got, tt.wantIndivs)
			}
			if got := len(doc.Families()); got != tt.wantFamilies {
				t.Errorf("Families count = %d, want %d", got, tt.wantFamilies)
			}
			if doc.Header.Version != tt.checkVersion {
				t.Errorf("Version = %s, want %s", doc.Header.Version, tt.checkVersion)
			}
		})
	}
}

func TestDecodeMatchesDirectCall(t *testing.T) {
	// Verify facade produces identical results to direct package call
	r1 := strings.NewReader(testGedcomMinimal)
	r2 := strings.NewReader(testGedcomMinimal)

	facadeDoc, facadeErr := Decode(r1)
	directDoc, directErr := decoder.Decode(r2)

	if (facadeErr != nil) != (directErr != nil) {
		t.Fatalf("error mismatch: facade=%v, direct=%v", facadeErr, directErr)
	}

	if facadeDoc == nil || directDoc == nil {
		if facadeDoc != directDoc {
			t.Fatal("nil mismatch between facade and direct call")
		}
		return
	}

	// Compare key attributes
	if len(facadeDoc.Individuals()) != len(directDoc.Individuals()) {
		t.Errorf("Individuals count mismatch: facade=%d, direct=%d",
			len(facadeDoc.Individuals()), len(directDoc.Individuals()))
	}
	if len(facadeDoc.Families()) != len(directDoc.Families()) {
		t.Errorf("Families count mismatch: facade=%d, direct=%d",
			len(facadeDoc.Families()), len(directDoc.Families()))
	}
	if facadeDoc.Header.Version != directDoc.Header.Version {
		t.Errorf("Version mismatch: facade=%s, direct=%s",
			facadeDoc.Header.Version, directDoc.Header.Version)
	}
}

func TestDecodeWithDiagnostics(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantErr         bool
		wantDiagnostics bool
		wantIndivs      int
	}{
		{
			name:            "valid GEDCOM",
			input:           testGedcomMinimal,
			wantErr:         false,
			wantDiagnostics: false,
			wantIndivs:      2,
		},
		{
			name: "GEDCOM with parse issues",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Test /Person/
  invalid line here
0 TRLR
`,
			wantErr:         false,
			wantDiagnostics: true,
			wantIndivs:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeWithDiagnostics(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeWithDiagnostics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			hasDiagnostics := len(result.Diagnostics) > 0
			if hasDiagnostics != tt.wantDiagnostics {
				t.Errorf("has diagnostics = %v, want %v (diagnostics: %v)",
					hasDiagnostics, tt.wantDiagnostics, result.Diagnostics)
			}
			if got := len(result.Document.Individuals()); got != tt.wantIndivs {
				t.Errorf("Individuals count = %d, want %d", got, tt.wantIndivs)
			}
		})
	}
}

func TestDecodeWithDiagnosticsMatchesDirectCall(t *testing.T) {
	r1 := strings.NewReader(testGedcomMinimal)
	r2 := strings.NewReader(testGedcomMinimal)

	facadeResult, facadeErr := DecodeWithDiagnostics(r1)
	directResult, directErr := decoder.DecodeWithDiagnostics(r2, nil)

	if (facadeErr != nil) != (directErr != nil) {
		t.Fatalf("error mismatch: facade=%v, direct=%v", facadeErr, directErr)
	}

	if facadeResult == nil || directResult == nil {
		return
	}

	if len(facadeResult.Diagnostics) != len(directResult.Diagnostics) {
		t.Errorf("Diagnostics count mismatch: facade=%d, direct=%d",
			len(facadeResult.Diagnostics), len(directResult.Diagnostics))
	}
}

func TestEncode(t *testing.T) {
	// First decode a document
	doc, err := Decode(strings.NewReader(testGedcomMinimal))
	if err != nil {
		t.Fatalf("setup Decode() failed: %v", err)
	}

	// Now encode it
	var buf bytes.Buffer
	err = Encode(&buf, doc)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()

	// Verify key elements are present
	if !strings.Contains(output, "0 HEAD") {
		t.Error("output missing HEAD")
	}
	if !strings.Contains(output, "0 TRLR") {
		t.Error("output missing TRLR")
	}
	if !strings.Contains(output, "@I1@ INDI") {
		t.Error("output missing individual @I1@")
	}
	if !strings.Contains(output, "@F1@ FAM") {
		t.Error("output missing family @F1@")
	}
}

// TestEncodeNilDocument checks that the sentinel a facade caller receives is
// matchable with the sentinel this package re-exports, without importing the
// encoder package to name it.
func TestEncodeNilDocument(t *testing.T) {
	var buf bytes.Buffer

	if err := Encode(&buf, nil); !errors.Is(err, ErrNilDocument) {
		t.Errorf("Encode(nil) error = %v, want ErrNilDocument", err)
	}

	if err := EncodeWithOptions(&buf, nil, nil); !errors.Is(err, ErrNilDocument) {
		t.Errorf("EncodeWithOptions(nil) error = %v, want ErrNilDocument", err)
	}

	if !errors.Is(ErrNilDocument, encoder.ErrNilDocument) {
		t.Error("ErrNilDocument is not the encoder package's sentinel")
	}
}

func TestEncodeMatchesDirectCall(t *testing.T) {
	doc, err := Decode(strings.NewReader(testGedcomMinimal))
	if err != nil {
		t.Fatalf("setup Decode() failed: %v", err)
	}

	var facadeBuf, directBuf bytes.Buffer
	facadeErr := Encode(&facadeBuf, doc)
	directErr := encoder.Encode(&directBuf, doc)

	if (facadeErr != nil) != (directErr != nil) {
		t.Fatalf("error mismatch: facade=%v, direct=%v", facadeErr, directErr)
	}

	if facadeBuf.String() != directBuf.String() {
		t.Error("output mismatch between facade and direct call")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrors bool
	}{
		{
			name:       "valid document",
			input:      testGedcomMinimal,
			wantErrors: false,
		},
		{
			name:       "document with broken xref",
			input:      testGedcomWithIssues,
			wantErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("setup Decode() failed: %v", err)
			}

			validationErrs := Validate(doc)
			hasErrors := len(validationErrs) > 0
			if hasErrors != tt.wantErrors {
				t.Errorf("has errors = %v, want %v (errors: %v)", hasErrors, tt.wantErrors, validationErrs)
			}
		})
	}
}

func TestValidateMatchesDirectCall(t *testing.T) {
	doc, err := Decode(strings.NewReader(testGedcomMinimal))
	if err != nil {
		t.Fatalf("setup Decode() failed: %v", err)
	}

	facadeErrors := Validate(doc)
	directErrors := validator.New().Validate(doc)

	if len(facadeErrors) != len(directErrors) {
		t.Errorf("error count mismatch: facade=%d, direct=%d",
			len(facadeErrors), len(directErrors))
	}
}

func TestValidateAll(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantIssues bool
	}{
		{
			// Note: Even "valid" GEDCOM 5.5.1 may report MISSING_SUBM warning
			// because the spec requires SUBM reference. This tests that
			// ValidateAll returns at least some issues for comprehensive validation.
			name:       "minimal document (may have warnings)",
			input:      testGedcomMinimal,
			wantIssues: true, // MISSING_SUBM is expected for 5.5.1 without SUBM
		},
		{
			name:       "document with issues",
			input:      testGedcomWithIssues,
			wantIssues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("setup Decode() failed: %v", err)
			}

			issues := ValidateAll(doc)
			hasIssues := len(issues) > 0
			if hasIssues != tt.wantIssues {
				t.Errorf("has issues = %v, want %v (issues: %v)", hasIssues, tt.wantIssues, issues)
			}
		})
	}
}

func TestValidateAllMatchesDirectCall(t *testing.T) {
	doc, err := Decode(strings.NewReader(testGedcomMinimal))
	if err != nil {
		t.Fatalf("setup Decode() failed: %v", err)
	}

	facadeIssues := ValidateAll(doc)
	directIssues := validator.New().ValidateAll(doc)

	if len(facadeIssues) != len(directIssues) {
		t.Errorf("issue count mismatch: facade=%d, direct=%d",
			len(facadeIssues), len(directIssues))
	}
}

func TestConvert(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		targetVersion gedcom.Version
		wantErr       bool
		wantSuccess   bool
	}{
		{
			name:          "5.5.1 to 7.0",
			input:         testGedcomMinimal,
			targetVersion: gedcom.Version70,
			wantErr:       false,
			wantSuccess:   true,
		},
		{
			name:          "5.5.1 to 5.5",
			input:         testGedcomMinimal,
			targetVersion: gedcom.Version55,
			wantErr:       false,
			wantSuccess:   true,
		},
		{
			name:          "same version (no-op)",
			input:         testGedcomMinimal,
			targetVersion: gedcom.Version551,
			wantErr:       false,
			wantSuccess:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("setup Decode() failed: %v", err)
			}

			converted, report, err := Convert(doc, tt.targetVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if report.Success != tt.wantSuccess {
				t.Errorf("report.Success = %v, want %v", report.Success, tt.wantSuccess)
			}
			if converted.Header.Version != tt.targetVersion {
				t.Errorf("converted version = %s, want %s",
					converted.Header.Version, tt.targetVersion)
			}
		})
	}
}

func TestConvertMatchesDirectCall(t *testing.T) {
	doc, err := Decode(strings.NewReader(testGedcomMinimal))
	if err != nil {
		t.Fatalf("setup Decode() failed: %v", err)
	}

	facadeDoc, facadeReport, facadeErr := Convert(doc, gedcom.Version70)
	directDoc, directReport, directErr := converter.Convert(doc, gedcom.Version70)

	if (facadeErr != nil) != (directErr != nil) {
		t.Fatalf("error mismatch: facade=%v, direct=%v", facadeErr, directErr)
	}

	if facadeReport.Success != directReport.Success {
		t.Errorf("success mismatch: facade=%v, direct=%v",
			facadeReport.Success, directReport.Success)
	}
	if facadeDoc.Header.Version != directDoc.Header.Version {
		t.Errorf("version mismatch: facade=%s, direct=%s",
			facadeDoc.Header.Version, directDoc.Header.Version)
	}
}

func TestConvertInvalidVersion(t *testing.T) {
	doc, err := Decode(strings.NewReader(testGedcomMinimal))
	if err != nil {
		t.Fatalf("setup Decode() failed: %v", err)
	}

	_, _, err = Convert(doc, "invalid")
	if err == nil {
		t.Error("Convert() with invalid version should return error")
	}
}

func TestConvertNilDocument(t *testing.T) {
	_, _, err := Convert(nil, gedcom.Version70)
	if err == nil {
		t.Error("Convert() with nil document should return error")
	}
}

// TestTypeReexports verifies that type aliases work correctly.
func TestTypeReexports(t *testing.T) {
	// Verify Document type works
	var doc *Document
	doc, _ = Decode(strings.NewReader(testGedcomMinimal))
	if doc == nil {
		t.Fatal("Document type alias failed")
	}

	// Verify Individual type works
	var ind *Individual
	individuals := doc.Individuals()
	if len(individuals) > 0 {
		ind = individuals[0]
	}
	if ind == nil {
		t.Fatal("Individual type alias failed")
	}

	// Verify Family type works
	var fam *Family
	families := doc.Families()
	if len(families) > 0 {
		fam = families[0]
	}
	if fam == nil {
		t.Fatal("Family type alias failed")
	}

	// Verify Version constants work
	if Version55 != gedcom.Version55 {
		t.Error("Version55 constant mismatch")
	}
	if Version551 != gedcom.Version551 {
		t.Error("Version551 constant mismatch")
	}
	if Version70 != gedcom.Version70 {
		t.Error("Version70 constant mismatch")
	}
}

// TestDecodeRealFile tests decoding a real GEDCOM file from testdata.
func TestDecodeRealFile(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantVersion gedcom.Version
		minIndivs   int
	}{
		{
			name:        "GEDCOM 5.5 minimal",
			path:        "testdata/gedcom-5.5/minimal.ged",
			wantVersion: gedcom.Version55,
			minIndivs:   0,
		},
		{
			name:        "GEDCOM 5.5.1 minimal",
			path:        "testdata/gedcom-5.5.1/minimal.ged",
			wantVersion: gedcom.Version551,
			minIndivs:   0,
		},
		{
			name:        "GEDCOM 7.0 minimal",
			path:        "testdata/gedcom-7.0/minimal.ged",
			wantVersion: gedcom.Version70,
			minIndivs:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.Open(tt.path)
			if err != nil {
				t.Skipf("testdata file not available: %v", err)
			}
			defer file.Close()

			doc, err := Decode(file)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if doc.Header.Version != tt.wantVersion {
				t.Errorf("Version = %s, want %s", doc.Header.Version, tt.wantVersion)
			}
			if len(doc.Individuals()) < tt.minIndivs {
				t.Errorf("Individuals count = %d, want >= %d",
					len(doc.Individuals()), tt.minIndivs)
			}
		})
	}
}

// TestRoundTrip verifies decode → encode → decode fidelity for the public API.
//
// It compares document *content*, not record cardinality: gedcomtesting.
// AssertRoundTrip walks records and their subordinate tags recursively, so a
// round-trip that preserved the record skeleton while dropping every NAME,
// DATE, SOUR and NOTE fails here (issue #410).
//
// Header.Tags are compared, unconditionally. They were gated behind an
// off-by-default option while the encoder rebuilt HEAD from four scalar fields
// and discarded the rest (#429); with the header preserved, the gate is gone
// and the whole header is covered. The one value the encoder deliberately
// rewrites is CHAR, which must equal the charset it actually wrote — asserted
// positively in compareHeaders, not skipped.
//
// One limit is worth knowing before trusting a green run:
//
//   - It compares decode(input) against decode(encode(decode(input))), so any
//     information both decode passes lose identically is invisible here.
//
// Lossless Representation is non-negotiable (CONSTITUTION.md); this test is a
// necessary part of guarding it, not the whole of it.
func TestRoundTrip(t *testing.T) {
	t.Run("minimal inline", func(t *testing.T) {
		gedcomtesting.AssertRoundTrip(t, []byte(testGedcomMinimal))
	})

	// Spans all three supported versions plus edge-case exports, so the
	// assertion covers version-specific structures rather than one dialect.
	// The 46 MiB longsword.ged is deliberately excluded: a round-trip decodes
	// it twice, and decoder.TestScaleFixture already skips it under -race for
	// memory reasons.
	fixtures := []string{
		"testdata/gedcom-5.5/555SAMPLE.GED",
		"testdata/gedcom-5.5/royal92.ged",
		"testdata/gedcom-5.5.1/comprehensive.ged",
		"testdata/gedcom-7.0/maximal70.ged",
		"testdata/gedcom-7.0/remarriage1.ged",
		"testdata/edge-cases/calendar-dates.ged",
		"testdata/edge-cases/cont-conc.ged",
		"testdata/edge-cases/deep-nesting-levels.ged",
		"testdata/edge-cases/structural-torture.ged",

		// Non-UTF-8 sources. The encoder writes UTF-8 bytes whatever it read,
		// so these are the fixtures that catch a CHAR line contradicting its
		// own file: ansel-lf.ged used to fail its re-decode outright, and
		// ansi-cp1252 came back double-encoded ("La Coruña" -> "La CoruÃ±a").
		// No fixture from this directory was in any round-trip table before
		// (issue #425).
		// ibmpc-cp437-broskeep.ged and ibm-windows-easytree.ged are excluded:
		// they fail the *first* decode ("1 CHAR IBMPC"), which charset/ does not
		// implement. That is a decoder gap, not a round-trip one, so it does not
		// belong to this table.
		"testdata/encoding/ansel-lf.ged",
		"testdata/encoding/ansi-cp1252-ftm17.ged",
		"testdata/encoding/utf16le.ged",
		"testdata/encoding/utf16be.ged",
		"testdata/encoding/utf8-bom.ged",
		"testdata/encoding/utf8-nobom-lf.ged",
		"testdata/encoding/utf8-unicode.ged",
	}

	for _, path := range fixtures {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", path, err)
			}
			gedcomtesting.AssertRoundTrip(t, data)
		})
	}
}

// These fixtures carry a value on a level-0 line (`0 _ROOT Root element`,
// `0 _EVENT_DEFN Military Service`, RootsMagic's `0 _EVDEF ...` block), which
// the encoder used to drop — issue #404. Each failed only on Record[...].Value
// differences; every other structure already round-tripped. They are kept as a
// group so the level-0 value path stays guarded by real vendor exports rather
// than by a synthetic case alone (issue #410).
func TestRoundTrip_Level0RecordValue(t *testing.T) {
	fixtures := []string{
		"testdata/edge-cases/vendor-customtags-torture.ged",
		"testdata/edge-cases/vendor-legacy.ged",
		"testdata/edge-cases/rootsmagic-2026-export.ged",
	}

	for _, path := range fixtures {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", path, err)
			}
			gedcomtesting.AssertRoundTrip(t, data)
		})
	}
}

// TestRoundTripXRefOnStructuralLine pins the end-to-end losslessness of a
// level-0 HEAD/TRLR line that carries a cross-reference identifier the GEDCOM
// grammar does not allow there (issue #396). The real header survives, the
// misplaced line is re-encoded as the record it became — subordinates and all —
// and a second decode/encode pass is stable.
func TestRoundTripXRefOnStructuralLine(t *testing.T) {
	input := `0 HEAD
1 SOUR RealVendor
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @X1@ TRLR
1 NOTE important
0 @I1@ INDI
1 NAME Valid /Person/
0 TRLR
`

	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := buf.String()
	for _, want := range []string{"1 SOUR RealVendor", "0 @X1@ TRLR", "1 NOTE important"} {
		if !strings.Contains(out, want) {
			t.Errorf("encoded output missing %q:\n%s", want, out)
		}
	}

	doc2, err := Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("second Decode() error = %v", err)
	}
	var buf2 bytes.Buffer
	if err := Encode(&buf2, doc2); err != nil {
		t.Fatalf("second Encode() error = %v", err)
	}
	if buf2.String() != out {
		t.Errorf("round trip is not stable:\nfirst:\n%s\nsecond:\n%s", out, buf2.String())
	}
}

// TestDecodeResultType verifies DecodeResult type alias.
func TestDecodeResultType(t *testing.T) {
	var result *DecodeResult
	result, _ = DecodeWithDiagnostics(strings.NewReader(testGedcomMinimal))
	if result == nil {
		t.Fatal("DecodeResult type alias failed")
	}
	if result.Document == nil {
		t.Error("DecodeResult.Document should not be nil")
	}
}

// TestIssueType verifies Issue type alias.
func TestIssueType(t *testing.T) {
	doc, _ := Decode(strings.NewReader(testGedcomWithIssues))
	issues := ValidateAll(doc)

	// We expect at least one issue for the broken reference
	var issue Issue
	if len(issues) > 0 {
		issue = issues[0]
	}

	// Verify Issue fields are accessible
	_ = issue.Severity
	_ = issue.Code
	_ = issue.Message
}

// TestConversionReportType verifies ConversionReport type alias.
func TestConversionReportType(t *testing.T) {
	doc, _ := Decode(strings.NewReader(testGedcomMinimal))
	_, report, _ := Convert(doc, gedcom.Version70)

	cr := report
	if cr == nil {
		t.Fatal("ConversionReport type alias failed")
	}

	// Verify fields are accessible
	_ = cr.SourceVersion
	_ = cr.TargetVersion
	_ = cr.Success
}

// TestValidateEmptyDocument verifies behavior with empty document.
func TestValidateEmptyDocument(t *testing.T) {
	// Create an empty but non-nil document
	doc := &Document{
		XRefMap: make(map[string]*gedcom.Record),
		Header:  &gedcom.Header{},
		Trailer: &gedcom.Trailer{},
	}
	validationErrs := Validate(doc)
	// Empty document is valid structurally
	if len(validationErrs) > 0 {
		t.Logf("Validate() on empty document returned %d errors: %v", len(validationErrs), validationErrs)
	}
}

// TestValidateAllNilDocument verifies ValidateAll behavior with nil document.
func TestValidateAllNilDocument(t *testing.T) {
	issues := ValidateAll(nil)
	if len(issues) > 0 {
		t.Errorf("ValidateAll(nil) should return nil or empty, got %v", issues)
	}
}

// TestConvert551To70MultilineNoteRoundTrips pins that converting a 5.5.1
// document with a multi-line note to 7.0 produces output this library can read
// back (issue #419).
//
// The converter consolidates CONT into an embedded newline as its internal
// representation, which is fine; the defect was that the encoder then wrote
// that value inline, emitting a line with no level and no tag. The result was a
// conversion whose own output failed strict decode — while the ConversionReport
// said DataLoss was empty, so a caller checking the documented safety signal got
// an explicit all-clear on an unreadable file.
//
// CONT is retained in GEDCOM 7.0 (7.0 removed CONC, not CONT), so re-emitting it
// is the correct target-version encoding, not a fallback.
func TestConvert551To70MultilineNoteRoundTrips(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME A /B/
1 NOTE line1
2 CONT line2
0 TRLR
`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	converted, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(report.DataLoss) != 0 {
		t.Errorf("DataLoss = %v, want none", report.DataLoss)
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := buf.String()

	// The line break must come back as CONT, not as bare continuation text.
	if !strings.Contains(out, "1 NOTE line1\n2 CONT line2\n") {
		t.Errorf("converted output does not re-emit the line break as CONT\ngot:\n%s", out)
	}

	// The point of the issue: the converter's own output must be readable.
	roundTripped, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-decoding the converted output failed: %v\ngot:\n%s", err, out)
	}

	// And the note text must survive intact, not merely parse.
	var got string
	for _, r := range roundTripped.Records {
		for i, tag := range r.Tags {
			if tag.Tag != "NOTE" {
				continue
			}
			got = tag.Value
			if i+1 < len(r.Tags) && r.Tags[i+1].Tag == "CONT" {
				got += "\n" + r.Tags[i+1].Value
			}
		}
	}
	if got != "line1\nline2" {
		t.Errorf("note text after round trip = %q, want %q", got, "line1\nline2")
	}
}
