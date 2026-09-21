package merge_test

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/merge"
)

// Header.Submitter was never populated by the decoder before issue #503, so
// Combine's "doc1's submitter wins, else doc2's already-remapped one" rule and
// the XRef remapper's rewrite of the field had only ever run against
// hand-built documents. These tests drive both from a decoded document, where
// the typed field and the raw `1 SUBM` header tag now both carry a pointer and
// must stay in step through a collision remap.

// submitterGED is a 5.5.1 document whose header points at its own submitter
// record. NAME is templated so the two sides of a merge are distinguishable.
const submitterGED = `0 HEAD
1 SOUR MERGETEST
1 SUBM @U1@
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @U1@ SUBM
1 NAME Submitter NAMESUFFIX
0 @I1@ INDI
1 NAME Test /NAMESUFFIX/
0 TRLR
`

// decodeSubmitterDoc decodes submitterGED with suffix substituted in. Both
// calls produce the same XRefs (@U1@, @I1@), so combining two of them always
// exercises a collision strategy.
func decodeSubmitterDoc(t *testing.T, suffix string) *gedcom.Document {
	t.Helper()

	doc, err := decoder.Decode(strings.NewReader(strings.ReplaceAll(submitterGED, "NAMESUFFIX", suffix)))
	if err != nil {
		t.Fatalf("decode %s: %v", suffix, err)
	}
	if doc.Header.Submitter == "" {
		t.Fatalf("%s: decoder left Header.Submitter empty; the fixture is not exercising the merge", suffix)
	}
	return doc
}

// headerSubmitterTag returns the value of the level-1 SUBM header tag, or
// "" when the header does not carry one.
func headerSubmitterTag(h *gedcom.Header) string {
	for _, tag := range h.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "SUBM" {
			return tag.Value
		}
	}
	return ""
}

// TestCombine_HeaderSubmitterResolvesAfterCollisionRemap combines two decoded
// documents that both name a header submitter at the same XRef. The combined
// header must name a submitter record that exists in the combined document --
// a dangling header pointer is the failure mode a remap can introduce.
func TestCombine_HeaderSubmitterResolvesAfterCollisionRemap(t *testing.T) {
	strategies := []struct {
		name string
		opts merge.CombineOptions
	}{
		{"PrefixDoc2", merge.CombineOptions{CollisionStrategy: merge.PrefixDoc2, Prefix: "b_"}},
		{"RenumberDoc2", merge.CombineOptions{CollisionStrategy: merge.RenumberDoc2}},
	}

	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			doc1 := decodeSubmitterDoc(t, "A")
			doc2 := decodeSubmitterDoc(t, "B")

			out, _, err := merge.Combine(doc1, doc2, strategy.opts)
			if err != nil {
				t.Fatalf("Combine returned error: %v", err)
			}

			// doc1 is never remapped, so its submitter keeps its name.
			if out.Header.Submitter != "@U1@" {
				t.Errorf("Header.Submitter = %q, want @U1@ (doc1 wins)", out.Header.Submitter)
			}
			subm := out.GetSubmitter(out.Header.Submitter)
			if subm == nil {
				t.Fatalf("Header.Submitter %q names no record in the combined document", out.Header.Submitter)
			}
			if subm.Name != "Submitter A" {
				t.Errorf("combined header names %q, want doc1's submitter", subm.Name)
			}
			// The encoder writes a decoded header from Tags, so the raw tag is
			// what actually reaches the file and has to agree with the field.
			if got := headerSubmitterTag(out.Header); got != out.Header.Submitter {
				t.Errorf("raw header SUBM tag = %q, typed Submitter = %q; they must agree",
					got, out.Header.Submitter)
			}
		})
	}
}

// TestCombine_HeaderSubmitterAdoptedFromDoc2IsRemapped covers the other branch
// of the merge rule: doc1 has no header submitter, so doc2's is adopted -- and
// doc2 is the document a collision strategy renames.
func TestCombine_HeaderSubmitterAdoptedFromDoc2IsRemapped(t *testing.T) {
	doc1 := decodeSubmitterDoc(t, "A")
	doc2 := decodeSubmitterDoc(t, "B")

	// Strip doc1's header submitter entirely -- field and raw tag -- so the
	// merged header has to take doc2's.
	doc1.Header.Submitter = ""
	kept := doc1.Header.Tags[:0]
	for _, tag := range doc1.Header.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "SUBM" {
			continue
		}
		kept = append(kept, tag)
	}
	doc1.Header.Tags = kept

	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "b_",
	})
	if err != nil {
		t.Fatalf("Combine returned error: %v", err)
	}

	if out.Header.Submitter != "@b_U1@" {
		t.Errorf("Header.Submitter = %q, want @b_U1@ (doc2's, remapped)", out.Header.Submitter)
	}
	if report.RemappedXRefs["@U1@"] != "@b_U1@" {
		t.Errorf("report does not record the submitter remap: %v", report.RemappedXRefs)
	}
	subm := out.GetSubmitter(out.Header.Submitter)
	if subm == nil {
		t.Fatalf("Header.Submitter %q names no record in the combined document", out.Header.Submitter)
	}
	if subm.Name != "Submitter B" {
		t.Errorf("combined header names %q, want doc2's submitter", subm.Name)
	}
	if got := headerSubmitterTag(out.Header); got != out.Header.Submitter {
		t.Errorf("raw header SUBM tag = %q, typed Submitter = %q; they must agree",
			got, out.Header.Submitter)
	}
}

// TestRemapXRefs_DecodedHeaderSubmitterIsRewritten pins gedcom.Apply's header
// pass against a decoded document: both the typed field and the raw header tag
// have to follow the record to its new XRef.
func TestRemapXRefs_DecodedHeaderSubmitterIsRewritten(t *testing.T) {
	doc := decodeSubmitterDoc(t, "A")

	out, mapping, err := merge.RemapXRefs(doc, func(old string) string {
		return "@X" + strings.Trim(old, "@") + "@"
	})
	if err != nil {
		t.Fatalf("RemapXRefs returned error: %v", err)
	}
	if mapping["@U1@"] != "@XU1@" {
		t.Fatalf("mapping does not rename the submitter: %v", mapping)
	}
	if out.Header.Submitter != "@XU1@" {
		t.Errorf("Header.Submitter = %q, want @XU1@", out.Header.Submitter)
	}
	if out.GetSubmitter(out.Header.Submitter) == nil {
		t.Errorf("Header.Submitter %q names no record after the remap", out.Header.Submitter)
	}
	if got := headerSubmitterTag(out.Header); got != "@XU1@" {
		t.Errorf("raw header SUBM tag = %q, want @XU1@", got)
	}
	if doc.Header.Submitter != "@U1@" {
		t.Errorf("RemapXRefs mutated the input header: Submitter = %q", doc.Header.Submitter)
	}
}

// A hand-built document carries its submitter in the typed field with no raw
// tag; a decoded one carries both. Combining the two used to let doc2's raw
// `1 SUBM` through unopposed -- nothing in doc1's Tags collided with it --
// while out.Submitter kept doc1's pointer. Since the encoder writes a header
// from Tags whenever they are non-empty, the file named doc2's submitter and
// the document named doc1's: the documented "doc1 wins" rule inverting on the
// way to disk, with no HeaderConflict to show for it.
func TestCombineHeaderSubmitterMixedProvenance(t *testing.T) {
	handBuilt := &gedcom.Document{
		Header: &gedcom.Header{
			Version:   gedcom.Version551,
			Submitter: "@U1@",
		},
		Records: []*gedcom.Record{
			{XRef: "@U1@", Type: gedcom.RecordTypeSubmitter, Entity: &gedcom.Submitter{
				XRef: "@U1@", Name: "Submitter HandBuilt",
			}},
		},
		XRefMap: map[string]*gedcom.Record{},
	}
	for _, rec := range handBuilt.Records {
		handBuilt.XRefMap[rec.XRef] = rec
	}

	decoded := decodeSubmitterDoc(t, "Decoded")

	out, report, err := merge.Combine(handBuilt, decoded, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "b_",
	})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}

	if out.Header.Submitter != "@U1@" {
		t.Errorf("Header.Submitter = %q, want doc1's @U1@ (doc1 wins)", out.Header.Submitter)
	}

	// The invariant that actually matters: whatever reaches the file must
	// name the same record the document does. An absent tag is acceptable
	// here -- doc1 had none -- but a tag naming doc2's submitter is not.
	if got := headerSubmitterTag(out.Header); got != "" && got != out.Header.Submitter {
		t.Errorf("raw header SUBM tag = %q, typed Submitter = %q; the encoded file would "+
			"name a different submitter than the document does", got, out.Header.Submitter)
	}

	// Dropping doc2's submitter structure is a real decision about the
	// merged header, so it has to be reported rather than done silently.
	var reported bool
	for _, c := range report.HeaderConflicts {
		if c.Field == "Tags.SUBM" {
			reported = true
		}
	}
	if !reported {
		t.Error("dropping doc2's SUBM header structure must be reported as a HeaderConflict")
	}
}
