package merge_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
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
	add("@I1@", gedcom.RecordTypeIndividual, &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}})
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

// ExampleCombine_errors shows telling a caller mistake from a data
// problem with errors.Is: ErrNilDocument and ErrInvalidOptions report
// bad arguments, ErrXRefCollision a property of the documents.
func ExampleCombine_errors() {
	newDoc := func() *gedcom.Document {
		rec := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual}
		return &gedcom.Document{
			Header:  &gedcom.Header{Version: gedcom.Version70},
			Records: []*gedcom.Record{rec},
			XRefMap: map[string]*gedcom.Record{"@I1@": rec},
		}
	}

	_, _, err := merge.Combine(nil, newDoc(), merge.CombineOptions{})
	fmt.Println("nil document:", errors.Is(err, merge.ErrNilDocument))

	_, _, err = merge.Combine(newDoc(), newDoc(), merge.CombineOptions{CollisionStrategy: merge.PrefixDoc2})
	fmt.Println("missing prefix:", errors.Is(err, merge.ErrInvalidOptions))

	_, _, err = merge.Combine(newDoc(), newDoc(), merge.CombineOptions{})
	fmt.Println("collision:", errors.Is(err, merge.ErrXRefCollision))

	// Output:
	// nil document: true
	// missing prefix: true
	// collision: true
}

// ExampleIndividuals folds a duplicate individual into the one the caller
// keeps: the second record's death moves over, its identical name is not
// repeated, and the family that named it now names the survivor.
func ExampleIndividuals() {
	doc, err := decoder.Decode(strings.NewReader(`0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Example/
0 @I2@ INDI
1 NAME John /Example/
1 DEAT
2 DATE 1970
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I2@
0 TRLR
`))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	out, report, err := merge.Individuals(doc, "@I1@", "@I2@", merge.RecordMergeOptions{})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, s := range report.Moved {
		fmt.Println("moved:", s.Tags[0].Tag)
	}
	for _, s := range report.Deduplicated {
		fmt.Println("already held:", s.Tags[0].Tag, s.Tags[0].Value)
	}
	fmt.Println("husband:", out.GetFamily("@F1@").Husband)
	fmt.Println("@I2@ left:", out.GetRecord("@I2@") != nil)

	// Output:
	// moved: DEAT
	// moved: FAMS
	// already held: NAME John /Example/
	// husband: @I1@
	// @I2@ left: false
}

// ExampleFoldRepeatedEvents reports a birth recorded twice in one record,
// its date written two ways, then folds the second into the first: the
// citation and the note both survive.
func ExampleFoldRepeatedEvents() {
	doc, err := decoder.Decode(strings.NewReader(`0 HEAD
1 GEDC
2 VERS 5.5.1
0 @S1@ SOUR
1 TITL Census
0 @I1@ INDI
1 NAME Ann /Example/
1 BIRT
2 DATE 7 DEC 1977
2 SOUR @S1@
1 BIRT
2 DATE 07 Dec 1977
2 NOTE Family bible.
0 TRLR
`))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	for _, g := range merge.RepeatedEvents(doc) {
		fmt.Println("repeated:", g.XRef, g.Members[0].Tags[0].Tag, len(g.Members), "times")
	}
	out, report, err := merge.FoldRepeatedEvents(doc)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, f := range report.Folded {
		fmt.Println("folded:", f.XRef, f.Drop.Tags[1].Value, "into", f.Into.Tags[1].Value)
	}
	for _, t := range out.GetRecord("@I1@").Tags {
		fmt.Println(strings.TrimSpace(fmt.Sprint(t.Level, " ", t.Tag, " ", t.Value)))
	}

	// Output:
	// repeated: @I1@ BIRT 2 times
	// folded: @I1@ 07 Dec 1977 into 7 DEC 1977
	// 1 NAME Ann /Example/
	// 1 BIRT
	// 2 DATE 7 DEC 1977
	// 2 SOUR @S1@
	// 2 NOTE Family bible.
}
