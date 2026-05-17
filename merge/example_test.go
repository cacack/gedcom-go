package merge_test

import (
	"fmt"

	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/merge"
)

// Example_remapXRefs demonstrates the recommended transform pattern:
// take the existing @xref@ value, strip the leading "@", and prepend
// "@<prefix>_" to produce a well-formed XRef in a disjoint namespace.
// This is the canonical step before combining records from multiple
// documents that may otherwise share XRef identifiers.
func Example_remapXRefs() {
	// A tiny document with one individual and one family.
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		XRefMap: make(map[string]*gedcom.Record),
	}
	add := func(xref string, typ gedcom.RecordType, entity interface{}) {
		rec := &gedcom.Record{XRef: xref, Type: typ, Entity: entity}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}
	add("@I1@", gedcom.RecordTypeIndividual, &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []string{"@F1@"}})
	add("@F1@", gedcom.RecordTypeFamily, &gedcom.Family{XRef: "@F1@", Husband: "@I1@"})

	// Recommended transform: prefix every XRef into a disjoint namespace.
	// Note: the result MUST keep the @xref@ pointer shape, otherwise
	// RemapXRefs returns ErrInvalidRemap.
	out, mapping, err := merge.RemapXRefs(doc, func(old string) string {
		return "@A_" + old[1:]
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("mapping @I1@ ->", mapping["@I1@"])
	fmt.Println("mapping @F1@ ->", mapping["@F1@"])

	// Cross-references inside the remapped family now point at the
	// remapped husband.
	f := out.GetFamily("@A_F1@")
	fmt.Println("family husband:", f.Husband)

	// Output:
	// mapping @I1@ -> @A_I1@
	// mapping @F1@ -> @A_F1@
	// family husband: @A_I1@
}

// Example_combine shows the recommended pattern for merging two
// documents that share XRef identifiers. PrefixDoc2 keeps doc1's
// XRefs unchanged and prefixes only the colliding XRefs in doc2,
// preserving referential integrity in both directions.
func Example_combine() {
	// Two tiny documents that both define @I1@ — a collision.
	build := func(name string) *gedcom.Document {
		doc := &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version70, Encoding: gedcom.EncodingUTF8},
			XRefMap: make(map[string]*gedcom.Record),
		}
		rec := &gedcom.Record{
			XRef:   "@I1@",
			Type:   gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: name}}},
			Tags:   []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: name}},
		}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap["@I1@"] = rec
		return doc
	}
	doc1 := build("Alice")
	doc2 := build("Bob")

	combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2,
		Prefix:            "b_",
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// doc1's @I1@ kept its XRef; doc2's @I1@ was renamed.
	fmt.Println("doc2 remap @I1@ ->", report.RemappedXRefs["@I1@"])
	fmt.Println("combined has", len(combined.Records), "records")
	fmt.Println("doc1 individual:", combined.GetIndividual("@I1@").Names[0].Full)
	fmt.Println("doc2 individual:", combined.GetIndividual("@b_I1@").Names[0].Full)

	// Output:
	// doc2 remap @I1@ -> @b_I1@
	// combined has 2 records
	// doc1 individual: Alice
	// doc2 individual: Bob
}
