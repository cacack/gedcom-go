package decoder

import (
	"reflect"
	"strings"
	"testing"
)

// TestPersonalNameCitationsAndNotes pins issue #643: SOUR, NOTE and SNOTE
// under INDI.NAME reach PersonalName rather than only Record.Tags.
func TestPersonalNameCitationsAndNotes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantXRefs  []string
		wantInline []string
	}{
		{
			name: "5.5.1 NOTE pointer and inline text",
			input: `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @N1@ NOTE Shared
0 @S1@ SOUR
1 TITL Parish register
0 @I1@ INDI
1 NAME Liutgarde /de Vermandois/
2 SOUR @S1@
3 PAGE Folio 12
4 CONT Entry 412
3 QUAY 2
3 DATA
4 DATE 6 AUG 1901
3 NOTE Citation note
2 NOTE @N1@
2 NOTE Spelling varies.
3 CONT See also Leutgarde.
0 TRLR`,
			wantXRefs:  []string{"@N1@"},
			wantInline: []string{"Spelling varies.\nSee also Leutgarde."},
		},
		{
			name: "7.0 SNOTE pointer",
			input: `0 HEAD
1 GEDC
2 VERS 7.0
0 @N1@ SNOTE Shared
0 @S1@ SOUR
1 TITL Parish register
0 @I1@ INDI
1 NAME Liutgarde /de Vermandois/
2 SOUR @S1@
3 PAGE Folio 12
4 CONT Entry 412
3 QUAY 2
3 DATA
4 DATE 6 AUG 1901
3 NOTE Citation note
2 SNOTE @N1@
2 NOTE Spelling varies.
3 CONT See also Leutgarde.
0 TRLR`,
			wantXRefs:  []string{"@N1@"},
			wantInline: []string{"Spelling varies.\nSee also Leutgarde."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := DecodeWithDiagnostics(strings.NewReader(tt.input), DefaultOptions())
			if err != nil {
				t.Fatalf("DecodeWithDiagnostics failed: %v", err)
			}
			for _, d := range res.Diagnostics {
				t.Errorf("unexpected diagnostic: %s", d.String())
			}

			indi := res.Document.GetIndividual("@I1@")
			if indi == nil || len(indi.Names) != 1 {
				t.Fatalf("expected @I1@ with one NAME, got %+v", indi)
			}
			name := indi.Names[0]

			if !reflect.DeepEqual(name.NoteXRefs, tt.wantXRefs) {
				t.Errorf("NoteXRefs = %#v, want %#v", name.NoteXRefs, tt.wantXRefs)
			}
			if !reflect.DeepEqual(name.InlineNotes, tt.wantInline) {
				t.Errorf("InlineNotes = %#v, want %#v", name.InlineNotes, tt.wantInline)
			}

			if len(name.SourceCitations) != 1 {
				t.Fatalf("SourceCitations = %d, want 1", len(name.SourceCitations))
			}
			cite := name.SourceCitations[0]
			if cite.SourceXRef != "@S1@" {
				t.Errorf("SourceXRef = %q, want @S1@", cite.SourceXRef)
			}
			if cite.Page != "Folio 12\nEntry 412" {
				t.Errorf("Page = %q, want folded PAGE", cite.Page)
			}
			if cite.Quality == nil || *cite.Quality != 2 {
				t.Errorf("Quality = %v, want 2", cite.Quality)
			}
			if cite.Data == nil || cite.Data.Date != "6 AUG 1901" {
				t.Errorf("Data = %+v, want DATE 6 AUG 1901", cite.Data)
			}
			if want := []string{"Citation note"}; !reflect.DeepEqual(cite.InlineNotes, want) {
				t.Errorf("citation InlineNotes = %#v, want %#v", cite.InlineNotes, want)
			}
		})
	}
}
