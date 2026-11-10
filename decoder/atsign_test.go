package decoder

import (
	"reflect"
	"strings"
	"testing"
)

// TestAtSign70 decodes gedcom7code 7/atsign.ged, the 7.0 conversion of
// 5/atsign.ged. GEDCOM 7.0 doubles only a leading "@", so N01-N04 hold one
// literal leading "@" -- the text their 5.5.1 originals carry -- while an "@"
// elsewhere is kept as written (#708). The raw value keeps the wire form.
func TestAtSign70(t *testing.T) {
	result := decodeFixture(t, "../testdata/gedcom7code/7/atsign.ged")
	doc := result.Document

	tests := []struct {
		xref string
		want string
	}{
		{"@N01@", "@ one leading"},
		{"@N02@", "@one leading no space"},
		{"@N03@", "@ two leading"},
		{"@N04@", "@two leading no space"},
		{"@N05@", "doubled @ internal"},
		{"@N08@", "single@internal no space"},
		{"@N18@", "@all in @one@thing WITH_DATES , etc"},
		{"@N19@", "@ at at front and @ at after CONC and \n@ at after CONT and @ inside CONT too."},
	}
	for _, tt := range tests {
		note := doc.GetSharedNote(tt.xref)
		if note == nil {
			t.Errorf("GetSharedNote(%s) returned nil", tt.xref)
			continue
		}
		if note.Text != tt.want {
			t.Errorf("SharedNote %s Text = %q, want %q", tt.xref, note.Text, tt.want)
		}
	}
	if rec := doc.GetRecord("@N01@"); rec == nil || rec.Value != "@@ one leading" {
		t.Errorf("raw Record.Value of @N01@ not kept as written: %+v", rec)
	}
}

// TestEscapedNoteIsInlineText checks that a NOTE whose line value is escaped
// pointer-shaped text is read as inline text, not as a pointer: the pointer
// test runs on the line as written, and only the text is unescaped (#708).
func TestEscapedNoteIsInlineText(t *testing.T) {
	tests := []struct {
		version string
		line    string
	}{
		{"5.5.1", "1 NOTE @@I1@@"},
		{"7.0", "1 NOTE @@I1@"},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			input := strings.Join([]string{
				"0 HEAD", "1 GEDC", "2 VERS " + tt.version, "0 @I1@ INDI", tt.line,
				"1 SOUR @S1@", "2 PAGE a@@b", "0 TRLR", "",
			}, "\n")
			doc, err := Decode(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			indi := doc.GetIndividual("@I1@")
			if !reflect.DeepEqual(indi.InlineNotes, []string{"@I1@"}) {
				t.Errorf("InlineNotes = %q, want [\"@I1@\"]", indi.InlineNotes)
			}
			if len(indi.NoteXRefs) != 0 {
				t.Errorf("NoteXRefs = %q, want empty", indi.NoteXRefs)
			}
			// An interior "@@" is an escape in 5.5.1 only.
			wantPage := map[string]string{"5.5.1": "a@b", "7.0": "a@@b"}[tt.version]
			if got := indi.SourceCitations[0].Page; got != wantPage {
				t.Errorf("Page = %q, want %q", got, wantPage)
			}
			if got := doc.GetRecord("@I1@").Tags[0].Value; got != strings.TrimPrefix(tt.line, "1 NOTE ") {
				t.Errorf("raw NOTE value = %q, want it as written", got)
			}
		})
	}
}
