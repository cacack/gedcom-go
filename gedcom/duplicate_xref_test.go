package gedcom

import "testing"

// duplicateXRefDocument returns a document whose two records share @I1@, its
// XRefMap holding the first, as the decoder builds it (#724).
func duplicateXRefDocument() *Document {
	first := &Record{XRef: "@I1@", Type: RecordTypeIndividual, LineNumber: 4}
	second := &Record{XRef: "@I1@", Type: RecordTypeIndividual, LineNumber: 6}
	return &Document{
		Header:  &Header{},
		Records: []*Record{first, second},
		XRefMap: map[string]*Record{"@I1@": first},
	}
}

// TestClone_DuplicateXRefKeepsFirst pins that Clone maps a shared XRef to the
// first record, as the decoder does, rather than to the last.
func TestClone_DuplicateXRefKeepsFirst(t *testing.T) {
	copied := duplicateXRefDocument().Clone()
	if got := copied.XRefMap["@I1@"]; got != copied.Records[0] {
		t.Errorf("XRefMap[@I1@] is the record at line %d, want the first (line 4)", got.LineNumber)
	}
}

// TestSubset_DuplicateXRefKeepsFirst pins the same rule for Subset.
func TestSubset_DuplicateXRefKeepsFirst(t *testing.T) {
	out, err := duplicateXRefDocument().Subset([]string{"@I1@"})
	if err != nil {
		t.Fatalf("Subset: %v", err)
	}
	if len(out.Records) != 2 {
		t.Fatalf("Records = %d, want both definitions copied", len(out.Records))
	}
	if got := out.XRefMap["@I1@"]; got != out.Records[0] || got.LineNumber != 4 {
		t.Errorf("XRefMap[@I1@] is the record at line %d, want the first (line 4)", got.LineNumber)
	}
}
