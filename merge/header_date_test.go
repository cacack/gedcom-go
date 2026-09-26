package merge_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/encoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/merge"
)

// dateGED is a 5.5.1 document whose header carries a transmission date with a
// TIME subordinate.
const dateGED = `0 HEAD
1 SOUR MERGETEST
1 DATE 7 AUG 2026
2 TIME 12:00:00
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I2@ INDI
1 NAME Test /Decoded/
0 TRLR
`

func decodeDateDoc(t *testing.T) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(dateGED))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Header.Date != "7 AUG 2026" {
		t.Fatalf("decoder left Header.Date = %q; the fixture is not exercising the merge", doc.Header.Date)
	}
	return doc
}

// headerTagValue returns the value of the level-1 header tag named name, or ""
// if there is none.
func headerTagValue(h *gedcom.Header, name string) (string, bool) {
	for _, tag := range h.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == name {
			return tag.Value, true
		}
	}
	return "", false
}

// A hand-built doc1 carries its Date in the typed field with no raw tag. Now
// that the decoder populates Header.Date (#495), a decoded doc2 carries both,
// and its "1 DATE" must not reach the merged Tags unopposed: the encoder
// writes the header from Tags, so the file would carry doc2's date while
// out.Date named doc1's. Mirrors TestCombineHeaderSubmitterMixedProvenance,
// and goes one further: doc1's date is written into Tags so it reaches the file.
func TestCombineHeaderDateMixedProvenance(t *testing.T) {
	handBuilt := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551, Date: "1 JAN 2000"},
		XRefMap: map[string]*gedcom.Record{},
	}

	out, report, err := merge.Combine(handBuilt, decodeDateDoc(t), merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}

	if out.Header.Date != "1 JAN 2000" {
		t.Errorf("Header.Date = %q, want doc1's 1 JAN 2000 (doc1 wins)", out.Header.Date)
	}
	if got, _ := headerTagValue(out.Header, "DATE"); got == "7 AUG 2026" {
		t.Errorf("doc2's raw header DATE tag %q survived; the encoded file would carry doc2's date", got)
	}
	// The whole DATE subtree goes, or TIME would be re-parented.
	for _, tag := range out.Header.Tags {
		if tag != nil && tag.Tag == "TIME" {
			t.Errorf("orphaned TIME %q kept after its DATE was dropped", tag.Value)
		}
	}

	// Keeping doc2's date out is not enough: out.Tags is non-empty (doc2's
	// SOUR/GEDC/CHAR), so the encoder writes the header from Tags, and doc1's
	// date must be there too or the file carries no date at all.
	if got, ok := headerTagValue(out.Header, "DATE"); !ok || got != "1 JAN 2000" {
		t.Errorf("raw header DATE tag = %q (present %v), want doc1's 1 JAN 2000", got, ok)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, out); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	encoded := buf.String()
	if !strings.Contains(encoded, "\n1 DATE 1 JAN 2000\n") {
		t.Errorf("encoded header lacks doc1's date:\n%s", encoded)
	}
	if strings.Contains(encoded, "7 AUG 2026") || strings.Contains(encoded, "TIME") {
		t.Errorf("encoded header carries doc2's date:\n%s", encoded)
	}
	// Grammar order: after SOUR, before GEDC.
	sour := strings.Index(encoded, "\n1 SOUR ")
	date := strings.Index(encoded, "\n1 DATE ")
	gedc := strings.Index(encoded, "\n1 GEDC")
	if sour >= date || date >= gedc {
		t.Errorf("DATE not placed between SOUR and GEDC:\n%s", encoded)
	}
	again, err := decoder.Decode(strings.NewReader(encoded))
	if err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if again.Header.Date != "1 JAN 2000" {
		t.Errorf("re-decoded Header.Date = %q, want 1 JAN 2000", again.Header.Date)
	}

	var reported bool
	for _, c := range report.HeaderConflicts {
		if c.Field == "Tags.DATE" {
			reported = true
		}
	}
	if !reported {
		t.Error("dropping doc2's DATE header structure must be reported as a HeaderConflict")
	}
}

// When doc1 has no Date, doc2's is adopted, and the typed field and the raw
// tag agree about it.
func TestCombineHeaderDateAdoptedFromDecodedDoc2(t *testing.T) {
	doc1 := &gedcom.Document{
		Header: &gedcom.Header{
			Version: gedcom.Version551,
			Tags:    []*gedcom.Tag{{Level: 1, Tag: "SOUR", Value: "Other"}},
		},
		XRefMap: map[string]*gedcom.Record{},
	}

	out, _, err := merge.Combine(doc1, decodeDateDoc(t), merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}

	if out.Header.Date != "7 AUG 2026" {
		t.Errorf("Header.Date = %q, want doc2's 7 AUG 2026", out.Header.Date)
	}
	if got, _ := headerTagValue(out.Header, "DATE"); got != out.Header.Date {
		t.Errorf("raw header DATE tag = %q, typed Date = %q; they must agree", got, out.Header.Date)
	}
}

// A hand-built doc1 Date is placed in grammar order among the merged raw tags,
// including for 7.0, where GEDC opens the header and so precedes DATE; with no
// structure that follows DATE in the grammar, it goes last. A doc1 that
// already carries its own raw DATE gets no second one.
func TestCombineHeaderDateSynthesizedPlacement(t *testing.T) {
	doc2With := func(tags ...*gedcom.Tag) *gedcom.Document {
		return &gedcom.Document{
			Header:  &gedcom.Header{Tags: tags},
			XRefMap: map[string]*gedcom.Record{},
		}
	}
	names := func(h *gedcom.Header) []string {
		var got []string
		for _, tag := range h.Tags {
			if tag.Level == 1 {
				got = append(got, tag.Tag)
			}
		}
		return got
	}

	cases := []struct {
		name    string
		version gedcom.Version
		doc1Tag []*gedcom.Tag
		doc2    *gedcom.Document
		want    []string
	}{
		{"7.0 after GEDC and SOUR", gedcom.Version70, nil, doc2With(
			&gedcom.Tag{Level: 1, Tag: "GEDC"},
			&gedcom.Tag{Level: 2, Tag: "VERS", Value: "7.0"},
			&gedcom.Tag{Level: 1, Tag: "SOUR", Value: "X"},
			&gedcom.Tag{Level: 1, Tag: "SUBM", Value: "@U1@"},
		), []string{"GEDC", "SOUR", "DATE", "SUBM"}},
		{"appended when nothing follows it", gedcom.Version551, nil, doc2With(
			&gedcom.Tag{Level: 1, Tag: "SOUR", Value: "X"},
			&gedcom.Tag{Level: 2, Tag: "VERS", Value: "1"},
		), []string{"SOUR", "DATE"}},
		{"doc1 raw DATE is not duplicated", gedcom.Version551,
			[]*gedcom.Tag{{Level: 1, Tag: "DATE", Value: "1 JAN 2000"}},
			doc2With(&gedcom.Tag{Level: 1, Tag: "SOUR", Value: "X"}),
			[]string{"DATE", "SOUR"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc1 := &gedcom.Document{
				Header:  &gedcom.Header{Version: tc.version, Date: "1 JAN 2000", Tags: tc.doc1Tag},
				XRefMap: map[string]*gedcom.Record{},
			}
			out, _, err := merge.Combine(doc1, tc.doc2, merge.CombineOptions{})
			if err != nil {
				t.Fatalf("Combine: %v", err)
			}
			if got := strings.Join(names(out.Header), ","); got != strings.Join(tc.want, ",") {
				t.Errorf("level-1 header tags = %s, want %s", got, strings.Join(tc.want, ","))
			}
		})
	}
}
