package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestDecodeWithDiagnostics_DuplicateXRef pins issue #724: a second record
// defining an XRef is reported as DUPLICATE_XREF at its own line, both
// records are kept, and XRefMap (so every pointer) resolves to the first.
func TestDecodeWithDiagnostics_DuplicateXRef(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n" +
		"0 @I1@ INDI\n1 NAME Jane /Doe/\n" +
		"0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := result.Document

	if len(doc.Records) != 2 || doc.Records[0].LineNumber != 4 || doc.Records[1].LineNumber != 6 {
		t.Fatalf("Records = %+v, want both definitions kept (lines 4 and 6)", doc.Records)
	}
	if doc.XRefMap["@I1@"] != doc.Records[0] {
		t.Errorf("XRefMap[@I1@] is the record at line %d, want the first definition (line 4)", doc.XRefMap["@I1@"].LineNumber)
	}
	indi := doc.GetIndividual("@I1@")
	if indi == nil || len(indi.Names) != 1 || indi.Names[0].Full != "John /Smith/" {
		t.Errorf("GetIndividual(@I1@) = %+v, want the first definition, John /Smith/", indi)
	}

	dups := diagnosticsWithCode(result.Diagnostics, CodeDuplicateXRef)
	if len(dups) != 1 {
		t.Fatalf("DUPLICATE_XREF diagnostics = %v, want exactly one", dups)
	}
	d := dups[0]
	if d.Line != 6 || d.Severity != SeverityError || d.Context != "0 @I1@ INDI" ||
		!strings.Contains(d.Message, "@I1@") || !strings.Contains(d.Message, "line 4") {
		t.Errorf("diagnostic = %s, want an ERROR at line 6 naming @I1@ and its first definition at line 4", d)
	}

	t.Run("strict mode keeps the first definition", func(t *testing.T) {
		doc, err := DecodeWithOptions(strings.NewReader(input), &DecodeOptions{StrictMode: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := doc.XRefMap["@I1@"]; got == nil || got.LineNumber != 4 {
			t.Errorf("XRefMap[@I1@] = %+v, want the record at line 4", got)
		}
	})
}

// TestDecodeWithDiagnostics_UnknownRecordType pins issue #724: a level-0 tag
// that is neither a standard record type nor an extension is reported as
// UNKNOWN_TAG, and a standard type in the wrong case is named as such.
func TestDecodeWithDiagnostics_UnknownRecordType(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" + // lines 1-3
		"0 @I1@ indi\n1 NAME John /Smith/\n" + // line 4
		"0 @X1@ FOO\n" + // line 6
		"0 CONT world\n" + // line 7
		"0 @X2@ _CUSTOM\n" + // line 8: extension, allowed
		"0 @I2@ INDI\n1 NAME Ann /Lee/\n" + // line 9: standard
		"0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[int]struct{ context, message string }{
		4: {"0 @I1@ indi", "unknown record type: indi (tags are case-sensitive; did you mean INDI?)"},
		6: {"0 @X1@ FOO", "unknown record type: FOO"},
		7: {"0 CONT world", "unknown record type: CONT"},
	}
	got := diagnosticsWithCode(result.Diagnostics, CodeUnknownTag)
	if len(got) != len(want) {
		t.Errorf("UNKNOWN_TAG diagnostics = %v, want lines 4, 6 and 7 only", got)
	}
	for _, d := range got {
		w, ok := want[d.Line]
		if !ok {
			t.Errorf("unexpected diagnostic %s", d)
			continue
		}
		if d.Severity != SeverityWarning || d.Context != w.context || d.Message != w.message {
			t.Errorf("line %d diagnostic = %s, want WARNING %q (context %q)", d.Line, d, w.message, w.context)
		}
	}

	// The lower-case record is kept as it is: no entity, and lossless.
	rec := result.Document.XRefMap["@I1@"]
	if rec == nil || rec.Type != gedcom.RecordType("indi") || rec.Entity != nil || len(rec.Tags) != 1 {
		t.Errorf("@I1@ = %+v, want the record kept with Type \"indi\", its NAME tag and no entity", rec)
	}
}

// TestDecodeWithDiagnostics_UnknownRecordTypeExemptions pins that level-0
// lines already reported as INVALID_XREF (an XRef on HEAD or TRLR, an XRef
// with no tag) are not also reported as unknown record types.
func TestDecodeWithDiagnostics_UnknownRecordTypeExemptions(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @X1@ HEAD\n" + // line 4
		"0 @X2@\n" + // line 5
		"0 @X3@ TRLR\n" + // line 6
		"0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := diagnosticsWithCode(result.Diagnostics, CodeUnknownTag); len(got) != 0 {
		t.Errorf("UNKNOWN_TAG diagnostics = %v, want none", got)
	}
	lines := map[int]bool{}
	for _, d := range diagnosticsWithCode(result.Diagnostics, CodeInvalidXRef) {
		lines[d.Line] = true
	}
	if !lines[4] || !lines[5] || !lines[6] {
		t.Errorf("INVALID_XREF lines = %v, want 4, 5 and 6", lines)
	}
}

// diagnosticsWithCode returns the diagnostics in ds that carry code.
func diagnosticsWithCode(ds Diagnostics, code string) Diagnostics {
	var out Diagnostics
	for _, d := range ds {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}
