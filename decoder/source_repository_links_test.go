package decoder

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// callNumberTriples flattens call numbers to "Value|MediaType|MediaPhrase"
// strings so a mismatch prints readably and a nil entry shows as "<nil>".
func callNumberTriples(calns []*gedcom.CallNumber) []string {
	if calns == nil {
		return nil
	}
	out := make([]string, len(calns))
	for i, c := range calns {
		if c == nil {
			out[i] = "<nil>"
			continue
		}
		out[i] = c.Value + "|" + c.MediaType + "|" + c.MediaPhrase
	}
	return out
}

// decodeTestFile decodes a corpus file relative to the decoder package.
func decodeTestFile(t *testing.T, path string) *gedcom.Document {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode(%s): %v", path, err)
	}
	return doc
}

// TestSourceRepositoryLinksMaximal70 pins #553 against the GEDCOM 7.0
// reference file. @S1@ carries two REPO links; v2 kept only the last one, and
// collapsed the second link's ten same-text CALNs to a single MEDI.
func TestSourceRepositoryLinksMaximal70(t *testing.T) {
	doc := decodeTestFile(t, "../testdata/gedcom-7.0/maximal70.ged")

	src := doc.GetSource("@S1@")
	if src == nil {
		t.Fatal("GetSource(@S1@) returned nil")
	}
	if len(src.RepositoryLinks) != 2 {
		t.Fatalf("len(RepositoryLinks) = %d, want 2", len(src.RepositoryLinks))
	}

	first := src.RepositoryLinks[0]
	if first.XRef != "@R1@" {
		t.Errorf("RepositoryLinks[0].XRef = %q, want @R1@", first.XRef)
	}
	if want := []string{"Note text"}; !reflect.DeepEqual(first.InlineNotes, want) {
		t.Errorf("RepositoryLinks[0].InlineNotes = %q, want %q", first.InlineNotes, want)
	}
	if want := []string{"@N1@"}; !reflect.DeepEqual(first.NoteXRefs, want) {
		t.Errorf("RepositoryLinks[0].NoteXRefs = %q, want %q", first.NoteXRefs, want)
	}
	if got, want := callNumberTriples(first.CallNumbers), []string{"Call number|BOOK|Booklet"}; !reflect.DeepEqual(got, want) {
		t.Errorf("RepositoryLinks[0].CallNumbers = %q, want %q", got, want)
	}

	second := src.RepositoryLinks[1]
	if second.XRef != "@R2@" {
		t.Errorf("RepositoryLinks[1].XRef = %q, want @R2@", second.XRef)
	}
	var want []string
	for _, medi := range []string{
		"VIDEO", "CARD", "FICHE", "FILM", "MAGAZINE",
		"MANUSCRIPT", "MAP", "NEWSPAPER", "PHOTO", "TOMBSTONE",
	} {
		want = append(want, "Call number|"+medi+"|")
	}
	if got := callNumberTriples(second.CallNumbers); !reflect.DeepEqual(got, want) {
		t.Errorf("RepositoryLinks[1].CallNumbers = %q, want %q", got, want)
	}
}

// TestSourceRepositoryLinksAncestris pins #553 against a vendor export: @S1@
// links to @R1@ and @R4@, and v2 kept only @R4@.
func TestSourceRepositoryLinksAncestris(t *testing.T) {
	doc := decodeTestFile(t, "../testdata/edge-cases/vendor-ancestris11-export.ged")

	src := doc.GetSource("@S1@")
	if src == nil {
		t.Fatal("GetSource(@S1@) returned nil")
	}
	var got []string
	for _, link := range src.RepositoryLinks {
		got = append(got, link.XRef)
	}
	if want := []string{"@R1@", "@R4@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("RepositoryLinks XRefs = %q, want %q", got, want)
	}
}

// TestSourceRepositoryLinksCallNumberShape covers the CALN substructure edge
// cases: a valueless CALN with a MEDI, a valueless MEDI carrying only a
// PHRASE, and a repeated MEDI, which violates 0:1 and is left to the raw tags.
func TestSourceRepositoryLinksCallNumberShape(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 7.0
0 @S1@ SOUR
1 REPO @R1@
2 CALN
3 MEDI BOOK
2 CALN Box 7
3 MEDI
4 PHRASE Loose leaves
2 CALN Box 8
3 MEDI MAP
3 MEDI PHOTO
0 @R1@ REPO
1 NAME Archive
0 TRLR
`
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	link := doc.GetSource("@S1@").RepositoryLinks[0]
	want := []string{"|BOOK|", "Box 7||Loose leaves", "Box 8|MAP|"}
	if got := callNumberTriples(link.CallNumbers); !reflect.DeepEqual(got, want) {
		t.Errorf("CallNumbers = %q, want %q", got, want)
	}
}
