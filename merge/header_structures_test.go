package merge_test

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

func decodeHeaderDoc(t *testing.T, header string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader("0 HEAD\n" + header +
		"1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n0 TRLR\n"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

// The typed header-structure views follow their raw structure (#676): doc1's
// win, and the dropped doc2 structures are reported as Tags conflicts.
func TestCombineHeaderStructuresConflict(t *testing.T) {
	doc1 := decodeHeaderDoc(t, "1 SOUR A\n2 VERS 1\n2 NAME App A\n2 CORP Corp A\n1 DEST X\n1 DATE 1 JAN 2000\n2 TIME 10:00\n1 PLAC\n2 FORM Town\n")
	doc2 := decodeHeaderDoc(t, "1 SOUR B\n2 VERS 2\n2 NAME App B\n2 CORP Corp B\n1 DEST Y\n1 DATE 2 JAN 2000\n2 TIME 11:00\n1 PLAC\n2 FORM City\n")

	got, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	h := got.Header
	if h.SourceVersion != "1" || h.SourceName != "App A" || h.SourceCorporation.Name != "Corp A" ||
		h.Destination != "X" || h.Time != "10:00" || h.PlaceForm != "Town" {
		t.Errorf("doc1 values not kept: %+v", h)
	}

	want := map[string]bool{
		"Tags.SOUR": true, "Tags.DEST": true, "Tags.DATE": true, "Tags.PLAC": true,
	}
	for _, c := range report.HeaderConflicts {
		delete(want, c.Field)
	}
	if len(want) != 0 {
		t.Errorf("missing conflicts for %v; got %+v", want, report.HeaderConflicts)
	}
}

// An empty doc1 view takes doc2's only when doc2's structure is the one the
// combined Tags carry.
func TestCombineHeaderStructuresPromote(t *testing.T) {
	// doc1 has a SOUR without VERS, so doc2's SOUR is dropped from Tags and
	// its VERS must not be promoted. doc1 has no DEST or PLAC, so doc2's are
	// adopted along with their views.
	doc1 := decodeHeaderDoc(t, "1 SOUR A\n")
	doc2 := decodeHeaderDoc(t, "1 SOUR B\n2 VERS 2\n2 CORP Corp B\n3 PHON 555\n1 DEST Y\n1 PLAC\n2 FORM City\n")

	got, _, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	h := got.Header
	if h.SourceVersion != "" || h.SourceCorporation != nil {
		t.Errorf("doc2's SOUR views promoted although its SOUR was dropped: %q %+v", h.SourceVersion, h.SourceCorporation)
	}
	if h.Destination != "Y" || h.PlaceForm != "City" {
		t.Errorf("Destination, PlaceForm = %q, %q; want Y, City", h.Destination, h.PlaceForm)
	}

	// With no SOUR in doc1 at all, doc2's SOUR and its views come across, and
	// the promoted corporation is a copy.
	doc1 = decodeHeaderDoc(t, "")
	got, _, err = merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	h = got.Header
	if h.SourceVersion != "2" || h.SourceCorporation == nil || h.SourceCorporation.Name != "Corp B" {
		t.Fatalf("doc2's SOUR views not promoted: %q %+v", h.SourceVersion, h.SourceCorporation)
	}
	h.SourceCorporation.Phone[0] = "changed"
	if doc2.Header.SourceCorporation.Phone[0] != "555" {
		t.Error("promoted SourceCorporation shares memory with doc2")
	}
}
