package parser_test

import (
	"os"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/parser"
)

// longsword.ged is a real Legacy 10.0 export that starts with a UTF-8 BOM.
// LazyParser indexes it, and every record it reads back carries the same tags
// decoder.Decode gives that record (issue #792).
func TestLazyParser_LongswordBOM_MatchesDecoder(t *testing.T) {
	const path = "../testdata/gedcom-5.5.1/longsword.ged"
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	lp := parser.NewLazyParser(f)
	if err := lp.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}
	head, err := lp.FindRecordByType("HEAD")
	if err != nil {
		t.Fatalf("FindRecordByType(HEAD) error = %v", err)
	}
	if head.ByteOffset != 3 {
		t.Errorf("HEAD ByteOffset = %d, want 3", head.ByteOffset)
	}

	df, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer df.Close()
	doc, err := decoder.Decode(df)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	xrefs := lp.XRefs()
	if len(xrefs) == 0 {
		t.Fatal("index holds no XRefs")
	}
	for _, xref := range xrefs {
		raw, err := lp.FindRecord(xref)
		if err != nil {
			t.Fatalf("FindRecord(%s) error = %v", xref, err)
		}
		rec := doc.XRefMap[xref]
		if rec == nil {
			t.Fatalf("decoder has no record %s", xref)
		}
		if raw.Type != string(rec.Type) {
			t.Errorf("%s: type %q, decoder %q", xref, raw.Type, rec.Type)
		}
		sub := raw.Lines[1:]
		if len(sub) != len(rec.Tags) {
			t.Fatalf("%s: %d subordinate lines, decoder has %d tags", xref, len(sub), len(rec.Tags))
		}
		for i, line := range sub {
			tag := rec.Tags[i]
			if line.Level != tag.Level || line.Tag != tag.Tag || line.Value != tag.Value || line.XRef != tag.XRef {
				t.Errorf("%s line %d: %d %s %q %q, decoder %d %s %q %q", xref, i,
					line.Level, line.Tag, line.XRef, line.Value, tag.Level, tag.Tag, tag.XRef, tag.Value)
			}
		}
	}
}
