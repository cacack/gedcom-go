package converter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestConvertVoidRecordTo70 pins issue #603: @VOID@ is an ordinary 5.5.1
// identifier but the null pointer in GEDCOM 7.0, so a record named @VOID@ is
// renamed and its pointers follow, while a @VOID@ pointer no record defines
// stays the null pointer.
func TestConvertVoidRecordTo70(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// wantVoid is the record @VOID@ is renamed to, "" when it is kept.
		wantVoid string
	}{
		{
			name: "record named @VOID@",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @VOID@ NOTE hello
0 @I1@ INDI
1 NOTE @VOID@
0 TRLR
`,
			wantVoid: "@VOID_2@",
		},
		{
			name: "rename avoids an existing @VOID_2@",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @VOID@ NOTE hello
0 @VOID_2@ NOTE taken
0 @I1@ INDI
1 NOTE @VOID@
0 TRLR
`,
			wantVoid: "@VOID_3@",
		},
		{
			name: "dangling @VOID@ pointer",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @N1@ NOTE hello
0 @I1@ INDI
1 NOTE @VOID@
0 TRLR
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("decode 5.5.1: %v", err)
			}
			converted, report, err := converter.Convert(doc, gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}

			var renamed string
			for _, n := range report.Normalized {
				if n.Original == "@VOID@" {
					renamed = n.Result
				}
			}
			if renamed != tt.wantVoid {
				t.Errorf("report renames @VOID@ to %q, want %q: %+v", renamed, tt.wantVoid, report.Normalized)
			}

			var buf bytes.Buffer
			if err := encoder.Encode(&buf, converted); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			out := buf.String()

			opts := decoder.DefaultOptions()
			opts.StrictMode = true
			result, err := decoder.DecodeWithDiagnostics(strings.NewReader(out), opts)
			if err != nil {
				t.Fatalf("strict decode 7.0: %v\n%s", err, out)
			}
			for _, d := range result.Diagnostics {
				t.Errorf("7.0 diagnostic: %s", d)
			}
			got := result.Document
			if got.GetRecord("@VOID@") != nil {
				t.Errorf("7.0 output still defines a @VOID@ record:\n%s", out)
			}

			if tt.wantVoid == "" {
				if !strings.Contains(out, "1 NOTE @VOID@") {
					t.Errorf("dangling @VOID@ pointer not kept:\n%s", out)
				}
				return
			}
			if strings.Contains(out, "@VOID@") {
				t.Errorf("output still holds @VOID@:\n%s", out)
			}
			note := got.GetNote(tt.wantVoid)
			if note == nil || note.Text != "hello" {
				t.Fatalf("renamed note %s = %+v, want text hello:\n%s", tt.wantVoid, note, out)
			}
			indi := got.GetIndividual("@I1@")
			if indi == nil || len(indi.NoteXRefs) != 1 || indi.NoteXRefs[0] != tt.wantVoid {
				t.Errorf("@I1@ NOTE does not point to %s:\n%s", tt.wantVoid, out)
			}
		})
	}
}

// TestConvertVoidIn70SourceIsNotRenamed pins that the #603 @VOID@ rename
// applies only to 5.5/5.5.1 sources: in a 7.0 source @VOID@ is the null
// pointer, so a 7.0 -> 7.0 conversion, which still normalizes invalid XRefs,
// never renames it, even when a (non-conforming) record is named @VOID@.
func TestConvertVoidIn70SourceIsNotRenamed(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "null pointer",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
0 @i1@ INDI
1 NOTE @VOID@
0 TRLR
`,
		},
		{
			name: "record named @VOID@",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
0 @VOID@ SNOTE hello
0 @i1@ INDI
1 NOTE @VOID@
0 TRLR
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("decode 7.0: %v", err)
			}
			hadVoidRecord := doc.GetRecord("@VOID@") != nil
			converted, report, err := converter.Convert(doc, gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}

			var renamedI1 bool
			for _, n := range report.Normalized {
				if n.Original == "@VOID@" {
					t.Errorf("report renames @VOID@ to %q in a 7.0 source", n.Result)
				}
				if n.Original == "@i1@" && n.Result == "@I1@" {
					renamedI1 = true
				}
			}
			if !renamedI1 {
				t.Errorf("invalid @i1@ not renamed to @I1@: %+v", report.Normalized)
			}

			if (converted.GetRecord("@VOID@") != nil) != hadVoidRecord {
				t.Errorf("@VOID@ record presence changed: had %v", hadVoidRecord)
			}
			indi := converted.GetIndividual("@I1@")
			if indi == nil {
				t.Fatalf("@I1@ not found after conversion")
			}
			var buf bytes.Buffer
			if err := encoder.Encode(&buf, converted); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, "0 @I1@ INDI\n1 NOTE @VOID@") {
				t.Errorf("@I1@ NOTE @VOID@ null pointer not kept:\n%s", out)
			}
		})
	}
}

// TestConvertVoidInGuessed70SourceIsNotRenamed pins that the @VOID@ gate keys
// on Header.Version, as gedcom.Apply does: a header with no VERS whose tags
// (here SNOTE) make detection guess 7.0 is treated as a 7.0 source, so a
// record named @VOID@ keeps its name.
func TestConvertVoidInGuessed70SourceIsNotRenamed(t *testing.T) {
	input := `0 HEAD
1 CHAR UTF-8
0 @VOID@ SNOTE hello
0 @I1@ INDI
1 NOTE @VOID@
0 TRLR
`
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Header.Version != gedcom.Version70 {
		t.Fatalf("premise: detected version = %q, want guessed %q", doc.Header.Version, gedcom.Version70)
	}

	converted, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, n := range report.Normalized {
		if n.Original == "@VOID@" {
			t.Errorf("report renames @VOID@ to %q in a guessed 7.0 source", n.Result)
		}
	}
	if converted.GetRecord("@VOID@") == nil {
		t.Errorf("@VOID@ record renamed or dropped")
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "0 @VOID@ SNOTE hello") || !strings.Contains(out, "0 @I1@ INDI\n1 NOTE @VOID@") {
		t.Errorf("@VOID@ record or pointer not kept:\n%s", out)
	}
}
