package converter_test

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestConvertSpacedXRefTo70 pins issue #579: uppercasing a mixed-case spaced
// 5.5.1 identifier for GEDCOM 7.0 must rewrite its pointers too, not leave
// them dangling at the old identifier.
func TestConvertSpacedXRefTo70(t *testing.T) {
	f, err := os.Open("../testdata/edge-cases/xref-case.ged")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	opts := decoder.DefaultOptions()
	opts.StrictMode = true
	doc, err := decoder.DecodeWithOptions(f, opts)
	if err != nil {
		t.Fatalf("strict decode: %v", err)
	}

	got, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	subm := got.GetSubmitter("@TEST@")
	if subm == nil {
		t.Fatal("submitter @TEST@ missing after conversion")
	}
	if len(subm.NoteXRefs) != 2 {
		t.Fatalf("NoteXRefs = %q, want 2 entries", subm.NoteXRefs)
	}
	for _, ref := range subm.NoteXRefs {
		if got.GetRecord(ref) == nil {
			t.Errorf("NoteXRefs entry %q does not resolve", ref)
		}
	}
	for _, tag := range got.GetRecord("@TEST@").Tags {
		if tag.Tag == "NOTE" && got.GetRecord(tag.Value) == nil {
			t.Errorf("raw NOTE tag value %q does not resolve", tag.Value)
		}
	}
}

// spacedUppercase551 has already-uppercase spaced identifiers, which plain
// uppercasing leaves invalid in GEDCOM 7.0, and an identifier (@I_1@) that the
// obvious rewrite of @I 1@ would collide with.
const spacedUppercase551 = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SUBM @U 1@
0 @U 1@ SUBM
1 NAME Submitter
0 @I 1@ INDI
1 NAME John /Doe/
1 FAMS @F 1@
0 @I_1@ INDI
1 NAME Jane /Doe/
1 FAMS @F 1@
0 @F 1@ FAM
1 HUSB @I 1@
1 WIFE @I_1@
1 NOTE @N 1@
0 @N 1@ NOTE spaced
0 TRLR
`

var (
	v7XRef     = regexp.MustCompile(`^@[A-Z0-9_]+@$`)
	pointerish = regexp.MustCompile(`@[^@]+@`)
)

// TestConvertSpacedXRefsYieldValidV7 pins issue #590: every 5.5.1 identifier
// that is not a valid GEDCOM 7.0 Xref (spaces, lowercase) is rewritten to a
// valid, collision-free one, its pointers follow, and the 7.0 output decodes
// strictly with every pointer resolving.
func TestConvertSpacedXRefsYieldValidV7(t *testing.T) {
	xrefCase, err := os.ReadFile("../testdata/edge-cases/xref-case.ged")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		input     string
		wantNotes int
	}{
		// @NoTe@ and @NoTe ref@ records, plus the dangling header SUBM @test@.
		{"xref-case.ged", string(xrefCase), 3},
		// Records @U 1@, @I 1@ (suffixed past @I_1@), @F 1@ and @N 1@.
		{"uppercase spaced", spacedUppercase551, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := decoder.DefaultOptions()
			opts.StrictMode = true
			doc, err := decoder.DecodeWithOptions(strings.NewReader(tt.input), opts)
			if err != nil {
				t.Fatalf("strict decode 5.5.1: %v", err)
			}
			records := len(doc.Records)

			converted, report, err := converter.Convert(doc, gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			var notes int
			for _, n := range report.Normalized {
				if v7XRef.MatchString(n.Result) && !v7XRef.MatchString(n.Original) {
					notes++
				}
			}
			if notes != tt.wantNotes {
				t.Errorf("XRef rewrite notes = %d, want %d: %+v", notes, tt.wantNotes, report.Normalized)
			}

			var buf bytes.Buffer
			if err := encoder.Encode(&buf, converted); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			out := buf.String()
			for _, ref := range pointerish.FindAllString(out, -1) {
				if !v7XRef.MatchString(ref) {
					t.Errorf("output holds invalid 7.0 xref %q:\n%s", ref, out)
				}
			}

			result, err := decoder.DecodeWithDiagnostics(strings.NewReader(out), opts)
			if err != nil {
				t.Fatalf("strict decode 7.0: %v\n%s", err, out)
			}
			for _, d := range result.Diagnostics {
				t.Errorf("7.0 diagnostic: %s", d)
			}
			got := result.Document
			if got.Header.Version != gedcom.Version70 {
				t.Errorf("version = %s, want 7.0", got.Header.Version)
			}
			if len(got.Records) != records {
				t.Errorf("records = %d, want %d (an identifier collided)", len(got.Records), records)
			}
			if s := got.Header.Submitter; s != "" && got.GetRecord(s) == nil {
				t.Errorf("header SUBM %q does not resolve", s)
			}
			for _, r := range got.Records {
				gedcom.Visit(r, func(ref string) {
					if got.GetRecord(ref) == nil {
						t.Errorf("%s: pointer %q does not resolve\n%s", r.XRef, ref, out)
					}
				})
			}
		})
	}
}

// TestConvertSpacedXRefsAndCalendarsTo70 pins that XRef rewriting (#590) and
// date calendar translation (#589) both apply in one 5.5.1 -> 7.0 conversion:
// the output holds no spaced identifier and no calendar escape, both report
// their transformations, and it decodes strictly as 7.0.
func TestConvertSpacedXRefsAndCalendarsTo70(t *testing.T) {
	const input = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I 1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE @#DJULIAN@ 1 JAN 1700
1 DEAT
2 DATE BET @#DJULIAN@ 1750 AND @#DGREGORIAN@ 1760
1 FAMS @F 1@
0 @F 1@ FAM
1 HUSB @I 1@
1 MARR
2 DATE @#DJULIAN@ 1720
0 TRLR
`
	opts := decoder.DefaultOptions()
	opts.StrictMode = true
	doc, err := decoder.DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("strict decode 5.5.1: %v", err)
	}

	converted, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	types := map[string]bool{}
	for _, tr := range report.Transformations {
		types[tr.Type] = true
	}
	for _, want := range []string{"XREF_UPPERCASE", "DATE_CALENDAR_CONVERTED"} {
		if !types[want] {
			t.Errorf("report lacks %s transformation: %+v", want, report.Transformations)
		}
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	for _, ref := range pointerish.FindAllString(out, -1) {
		if !v7XRef.MatchString(ref) {
			t.Errorf("output holds %q, invalid in 7.0:\n%s", ref, out)
		}
	}
	for _, want := range []string{
		"0 @I_1@ INDI",
		"2 DATE JULIAN 1 JAN 1700",
		"2 DATE BET JULIAN 1750 AND GREGORIAN 1760",
		"2 DATE JULIAN 1720",
		"1 HUSB @I_1@",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}

	result, err := decoder.DecodeWithDiagnostics(strings.NewReader(out), opts)
	if err != nil {
		t.Fatalf("strict decode 7.0: %v\n%s", err, out)
	}
	for _, d := range result.Diagnostics {
		t.Errorf("7.0 diagnostic: %s", d)
	}
	if fam := result.Document.GetFamily("@F_1@"); fam == nil || result.Document.GetIndividual(fam.Husband) == nil {
		t.Errorf("@F_1@ HUSB does not resolve:\n%s", out)
	}
}

// TestConvertSetsDeclaredVersion pins that a converted document declares its
// target version, so pointer recognition follows the new grammar (#591).
func TestConvertSetsDeclaredVersion(t *testing.T) {
	doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version551, DeclaredVersion: gedcom.Version551}}
	got, _, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got.Header.DeclaredVersion != gedcom.Version70 {
		t.Errorf("DeclaredVersion = %q, want %q", got.Header.DeclaredVersion, gedcom.Version70)
	}
	if got.IsPointerXRef("@N 1@") {
		t.Error("converted 7.0 document treats @N 1@ as a pointer")
	}
}
