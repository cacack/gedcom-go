package converter

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// atSign551 holds "@" in text where 5.5.1 and 7.0 escape it differently, and
// pointers, which neither escapes (issue #773).
const atSign551 = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 NOTE a@@b
1 NOTE @@lead
1 NOTE first
2 CONT @@second a@@c
1 NOTE @N1@
1 BIRT
2 DATE INT 1900 (@@home a@@b)
1 FAMS @F1@
0 @N1@ NOTE x@@y
0 @F1@ FAM
1 HUSB @I1@
0 TRLR
`

// convertAndEncode decodes input, converts it to target and encodes the
// result, returning the output text and its decoded document.
func convertAndEncode(t *testing.T, input string, target gedcom.Version) (string, *gedcom.Document, *gedcom.ConversionReport) {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	converted, report, err := Convert(doc, target)
	if err != nil {
		t.Fatalf("convert to %s: %v", target, err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("encode %s: %v", target, err)
	}
	out, err := decoder.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode %s output: %v", target, err)
	}
	return buf.String(), out, report
}

// assertLines checks that every line of want appears in out.
func assertLines(t *testing.T, out string, want ...string) {
	t.Helper()
	lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	for _, w := range want {
		if !slices.Contains(lines, w) {
			t.Errorf("output lacks line %q:\n%s", w, out)
		}
	}
}

// assertAtSignText checks the decoded text the atSign551 fixture holds, which
// must read the same in every version it is converted to.
func assertAtSignText(t *testing.T, doc *gedcom.Document) {
	t.Helper()
	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("no @I1@")
	}
	wantNotes := []string{"a@b", "@lead", "first\n@second a@c"}
	if !slices.Equal(indi.InlineNotes, wantNotes) {
		t.Errorf("InlineNotes = %q, want %q", indi.InlineNotes, wantNotes)
	}
	if !slices.Equal(indi.NoteXRefs, []string{"@N1@"}) {
		t.Errorf("NoteXRefs = %q, want [@N1@]", indi.NoteXRefs)
	}
	if len(indi.SpouseInFamilies) != 1 || indi.SpouseInFamilies[0].FamilyXRef != "@F1@" {
		t.Errorf("SpouseInFamilies = %+v, want @F1@", indi.SpouseInFamilies)
	}
}

func TestConvertReescapesAtSign(t *testing.T) {
	original, err := decoder.Decode(strings.NewReader(atSign551))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertAtSignText(t, original)

	// 5.5.1 -> 7.0: only a leading "@" stays doubled.
	out70, doc70, report := convertAndEncode(t, atSign551, gedcom.Version70)
	assertLines(t, out70,
		"1 NOTE a@b",
		"1 NOTE @@lead",
		"1 NOTE first",
		"2 CONT @@second a@c",
		"3 PHRASE @@home a@b",
		"0 @N1@ SNOTE x@y",
		"1 SNOTE @N1@",
		"1 FAMS @F1@",
		"1 HUSB @I1@",
	)
	assertAtSignText(t, doc70)
	assertReescapeReported(t, report)

	// 7.0 -> 5.5.1: every "@" is doubled again, giving back the input's text.
	out551, doc551, report := convertAndEncode(t, out70, gedcom.Version551)
	assertLines(t, out551,
		"1 NOTE a@@b",
		"1 NOTE @@lead",
		"2 CONT @@second a@@c",
		"2 DATE INT 1900 (@@home a@@b)",
		"0 @N1@ NOTE x@@y",
		"1 NOTE @N1@",
		"1 FAMS @F1@",
	)
	assertAtSignText(t, doc551)
	assertReescapeReported(t, report)
}

// assertReescapeReported checks the report names the rewritten NOTE "a@@b".
func assertReescapeReported(t *testing.T, report *gedcom.ConversionReport) {
	t.Helper()
	if !slices.ContainsFunc(report.Transformations, func(tr gedcom.Transformation) bool {
		return tr.Type == "AT_SIGN_REESCAPED" && tr.Count > 0
	}) {
		t.Errorf("no AT_SIGN_REESCAPED transformation in %+v", report.Transformations)
	}
	if !slices.ContainsFunc(report.Normalized, func(n gedcom.ConversionNote) bool {
		return n.Reason == atSignReescapeReason && strings.Contains(n.Path, "NOTE") &&
			(n.Original == "a@@b" && n.Result == "a@b" || n.Original == "a@b" && n.Result == "a@@b")
	}) {
		t.Errorf("no Normalized note for the NOTE value in %+v", report.Normalized)
	}
}

func TestConvert55To70ReescapesAtSign(t *testing.T) {
	input := strings.Replace(atSign551, "2 VERS 5.5.1", "2 VERS 5.5", 1)
	out, doc, _ := convertAndEncode(t, input, gedcom.Version70)
	assertLines(t, out, "1 NOTE a@b", "1 NOTE @@lead", "0 @N1@ SNOTE x@y")
	assertAtSignText(t, doc)

	back, doc55, _ := convertAndEncode(t, out, gedcom.Version55)
	assertLines(t, back, "1 NOTE a@@b", "1 NOTE @@lead", "0 @N1@ NOTE x@@y")
	assertAtSignText(t, doc55)
}

// TestConvertReescapeKeepsNonText checks what is not text keeps its "@": a
// pointer with a space, a calendar escape with no 7.0 keyword, and a header
// value is re-escaped like any other.
func TestConvertReescapeKeepsNonText(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 COPR a@@b
0 @I1@ INDI
1 BIRT
2 DATE @#DROMAN@ 1700
1 FAMS @F 1@
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	converted, _, err := Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if copr := findTagIn(converted.Header.Tags, "COPR"); copr == nil || copr.Value != "a@b" {
		t.Errorf("HEAD.COPR = %+v, want a@b", copr)
	}
	rec := converted.Records[0]
	if date := findTag(rec, "DATE"); date == nil || date.Value != "@#DROMAN@ 1700" {
		t.Errorf("DATE = %+v, want @#DROMAN@ 1700", date)
	}
	if fams := findTag(rec, "FAMS"); fams == nil || strings.HasPrefix(fams.Value, "@@") {
		t.Errorf("FAMS = %+v, want a pointer", fams)
	}
}

// findTagIn returns the first tag in tags named name, or nil.
func findTagIn(tags []*gedcom.Tag, name string) *gedcom.Tag {
	for _, tag := range tags {
		if tag != nil && tag.Tag == name {
			return tag
		}
	}
	return nil
}
