package decoder

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestSourceFieldsFoldContinuation pins issue #442: the free-text SOUR fields
// and citation PAGE fold their CONT/CONC lines, where they previously kept the
// first line alone.
func TestSourceFieldsFoldContinuation(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 BIRT
2 SOUR @S1@
3 PAGE Year: 1850; Roll:
4 CONC  M432_397
4 CONT Page: 12
3 QUAY 2
0 @S1@ SOUR
1 TITL Vital Records of Boston
2 CONT Volume II
1 AUTH City Clerk
2 CONC , Boston
1 PUBL Boston, 1930
2 CONT Reprinted 1961
1 TEXT First line.
2 CONT Second line.
2 CONC  continued.
1 ABBR Boston VR
0 TRLR
`
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	src := doc.GetSource("@S1@")
	if src == nil {
		t.Fatal("source @S1@ not found")
	}
	for _, tt := range []struct{ field, got, want string }{
		{"Title", src.Title, "Vital Records of Boston\nVolume II"},
		{"Author", src.Author, "City Clerk, Boston"},
		{"Publication", src.Publication, "Boston, 1930\nReprinted 1961"},
		{"Text", src.Text, "First line.\nSecond line. continued."},
	} {
		if tt.got != tt.want {
			t.Errorf("Source.%s = %q, want %q", tt.field, tt.got, tt.want)
		}
	}

	ind := doc.GetIndividual("@I1@")
	if ind == nil || len(ind.Events) != 1 || len(ind.Events[0].SourceCitations) != 1 {
		t.Fatalf("expected one BIRT with one citation, got %+v", ind)
	}
	cite := ind.Events[0].SourceCitations[0]
	if want := "Year: 1850; Roll: M432_397\nPage: 12"; cite.Page != want {
		t.Errorf("SourceCitation.Page = %q, want %q", cite.Page, want)
	}
	if cite.Quality == nil || *cite.Quality != 2 {
		t.Errorf("SourceCitation.Quality = %v, want 2 (fold must not swallow siblings)", cite.Quality)
	}
}

// TestFoldedValue covers the helper's boundaries: only direct CONT/CONC
// children fold, the walk stops at the next sibling, and a value with nothing
// to fold is returned without building a new string.
func TestFoldedValue(t *testing.T) {
	tags := []*gedcom.Tag{
		{Level: 1, Tag: "TEXT", Value: "a"},
		{Level: 2, Tag: "MIME", Value: "text/plain"},
		{Level: 2, Tag: "CONT", Value: "b"},
		{Level: 3, Tag: "CONT", Value: "too deep"},
		{Level: 2, Tag: "CONC", Value: "c"},
		{Level: 1, Tag: "TITL", Value: "only"},
		{Level: 2, Tag: "_VEND", Value: "x"},
		{Level: 1, Tag: "CONT", Value: "not a child"},
	}
	if got := foldedValue(tags, 0); got != "a\nbc" {
		t.Errorf("foldedValue(TEXT) = %q, want %q", got, "a\nbc")
	}
	if got := foldedValue(tags, 5); got != "only" {
		t.Errorf("foldedValue(TITL) = %q, want %q", got, "only")
	}
	if got := foldedValue(tags, len(tags)-1); got != "not a child" {
		t.Errorf("foldedValue(last) = %q, want %q", got, "not a child")
	}
}
