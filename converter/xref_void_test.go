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
