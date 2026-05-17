package gedcom_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/encoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/validator"
)

// TestSubset_RoundtripMaximal70 exercises Subset against the most
// reference-rich fixture: decode → subset → encode → decode → confirm
// closure preservation. The maximal70 fixture intentionally contains
// constructs (@VOID@ sentinels, fixture-internal broken refs) that the
// current validator does not understand, so this test asserts
// structural preservation rather than zero validator issues. The clean
// validation contract is covered by TestSubset_RoundtripComprehensive551.
func TestSubset_RoundtripMaximal70(t *testing.T) {
	doc := mustDecode(t, "../testdata/gedcom-7.0/maximal70.ged")

	seed := firstIndividualXRef(t, doc)
	sub, err := doc.Subset([]string{seed})
	if err != nil {
		t.Fatalf("Subset returned error: %v", err)
	}

	if len(sub.Records) == 0 {
		t.Fatal("subset has no records")
	}
	if _, ok := sub.XRefMap[seed]; !ok {
		t.Fatalf("seed %q missing from subset XRefMap", seed)
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, sub); err != nil {
		t.Fatalf("encode subset: %v", err)
	}

	roundtripped, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("re-decode subset: %v", err)
	}

	if len(roundtripped.Records) != len(sub.Records) {
		t.Errorf("record count after round-trip = %d, want %d",
			len(roundtripped.Records), len(sub.Records))
	}
	for xref := range sub.XRefMap {
		if _, ok := roundtripped.XRefMap[xref]; !ok {
			t.Errorf("xref %q lost in round-trip", xref)
		}
	}
}

func TestSubset_RoundtripComprehensive551(t *testing.T) {
	doc := mustDecode(t, "../testdata/gedcom-5.5.1/comprehensive.ged")
	seed := firstIndividualXRef(t, doc)

	sub, err := doc.Subset([]string{seed})
	if err != nil {
		t.Fatalf("Subset returned error: %v", err)
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, sub); err != nil {
		t.Fatalf("encode subset: %v", err)
	}

	roundtripped, err := decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("re-decode subset: %v", err)
	}

	v := validator.New()
	if issues := v.Validate(roundtripped); len(issues) != 0 {
		t.Errorf("validator reported %d issues on round-tripped subset:", len(issues))
		for _, issue := range issues {
			t.Logf("  %v", issue)
		}
	}
}

func TestSubset_SourceUnchangedAfterRoundtrip(t *testing.T) {
	doc := mustDecode(t, "../testdata/gedcom-7.0/maximal70.ged")
	originalCount := len(doc.Records)
	originalXRefMapCount := len(doc.XRefMap)
	seed := firstIndividualXRef(t, doc)

	if _, err := doc.Subset([]string{seed}); err != nil {
		t.Fatalf("Subset errored: %v", err)
	}

	if len(doc.Records) != originalCount {
		t.Errorf("source Records mutated: %d -> %d", originalCount, len(doc.Records))
	}
	if len(doc.XRefMap) != originalXRefMapCount {
		t.Errorf("source XRefMap mutated: %d -> %d", originalXRefMapCount, len(doc.XRefMap))
	}
}

func mustDecode(t *testing.T, path string) *gedcom.Document {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return doc
}

func firstIndividualXRef(t *testing.T, doc *gedcom.Document) string {
	t.Helper()
	for _, rec := range doc.Records {
		if rec.Type == gedcom.RecordTypeIndividual && strings.HasPrefix(rec.XRef, "@") {
			return rec.XRef
		}
	}
	t.Fatal("no individuals in document")
	return ""
}
