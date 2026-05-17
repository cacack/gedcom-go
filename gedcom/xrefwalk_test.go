package gedcom

import (
	"sort"
	"testing"
)

func TestIsPointerXRef(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"too short", "@@", false},
		{"missing leading @", "I1@", false},
		{"missing trailing @", "@I1", false},
		{"void sentinel", "@VOID@", false},
		{"contains space", "@I 1@", false},
		{"contains tab", "@I\t1@", false},
		{"contains newline", "@I\n1@", false},
		{"contains carriage return", "@I\r1@", false},
		{"interior at sign", "@x@y@", false},
		{"interior at sign deeper", "@pfx@_I1@", false},
		{"plain text", "John Doe", false},
		{"valid simple", "@I1@", true},
		{"valid with underscore", "@I_1@", true},
		{"valid alphanumeric", "@F123@", true},
		{"valid lowercase", "@i1@", true},
		{"minimum length", "@A@", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsPointerXRef(tc.in); got != tc.want {
				t.Errorf("IsPointerXRef(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// xrefwalkFullDocument constructs a hand-built Document populated with every
// entity type and every XRef-bearing field that the walker covers.
// Each XRef uses a unique sentinel so tests can verify per-field coverage.
func xrefwalkFullDocument() *Document {
	individual := &Individual{
		XRef: "@I1@",
		ChildInFamilies: []FamilyLink{
			{FamilyXRef: "@F-CHILD@"},
		},
		SpouseInFamilies: []string{"@F-SPOUSE@"},
		Notes:            []string{"@N-IND@", "inline note text"},
		Associations: []*Association{
			{
				IndividualXRef: "@I-ASSOC@",
				Notes:          []string{"@N-ASSOC@"},
				SourceCitations: []*SourceCitation{
					{SourceXRef: "@S-ASSOC@"},
				},
			},
			nil, // exercise nil-element handling
		},
		SourceCitations: []*SourceCitation{
			{SourceXRef: "@S-IND@"},
			nil,
		},
		Media: []*MediaLink{
			{MediaXRef: "@M-IND@"},
			nil,
		},
		Events: []*Event{
			{
				Notes: []string{"@N-EVENT@"},
				SourceCitations: []*SourceCitation{
					{SourceXRef: "@S-EVENT@"},
				},
				Media: []*MediaLink{
					{MediaXRef: "@M-EVENT@"},
				},
				Tags: []*Tag{
					{Tag: "_CUSTOM", XRef: "@T-EVENT@"},
				},
			},
			nil,
		},
		Attributes: []*Attribute{
			{
				Type: "OCCU",
				SourceCitations: []*SourceCitation{
					{SourceXRef: "@S-ATTR@"},
				},
			},
			nil,
		},
		LDSOrdinances: []*LDSOrdinance{
			{FamilyXRef: "@F-LDS@"},
			nil,
		},
		Tags: []*Tag{
			{Tag: "_X", XRef: "@T-IND-XREF@"},
			{Tag: "_Y", Value: "@T-IND-VAL@"},
		},
	}

	family := &Family{
		XRef:     "@F1@",
		Husband:  "@I-HUSB@",
		Wife:     "@I-WIFE@",
		Children: []string{"@I-CHILD1@", "@I-CHILD2@"},
		Notes:    []string{"@N-FAM@"},
		SourceCitations: []*SourceCitation{
			{SourceXRef: "@S-FAM@"},
		},
		Media: []*MediaLink{
			{MediaXRef: "@M-FAM@"},
		},
		Events: []*Event{
			{
				Notes: []string{"@N-FAM-EVENT@"},
			},
		},
		LDSOrdinances: []*LDSOrdinance{
			{FamilyXRef: "@F-FAM-LDS@"},
		},
		Tags: []*Tag{
			{Tag: "_F", XRef: "@T-FAM@"},
		},
	}

	source := &Source{
		XRef:          "@S1@",
		RepositoryRef: "@R-SRC@",
		Notes:         []string{"@N-SRC@"},
		Media: []*MediaLink{
			{MediaXRef: "@M-SRC@"},
		},
		Tags: []*Tag{
			{Tag: "_S", Value: "@T-SRC@"},
		},
	}

	repo := &Repository{
		XRef:  "@R1@",
		Notes: []string{"@N-REPO@"},
		Tags: []*Tag{
			{Tag: "_R", XRef: "@T-REPO@"},
		},
	}

	note := &Note{
		XRef: "@N1@",
		Tags: []*Tag{
			{Tag: "_N", XRef: "@T-NOTE@"},
		},
	}

	media := &MediaObject{
		XRef:  "@M1@",
		Notes: []string{"@N-MEDIA@"},
		SourceCitations: []*SourceCitation{
			{SourceXRef: "@S-MEDIA@"},
		},
		Tags: []*Tag{
			{Tag: "_M", XRef: "@T-MEDIA@"},
		},
	}

	submitter := &Submitter{
		XRef:  "@U1@",
		Notes: []string{"@N-SUBM@"},
		Tags: []*Tag{
			{Tag: "_U", Value: "@T-SUBM@"},
		},
	}

	snote := &SharedNote{
		XRef: "@SN1@",
		SourceCitations: []*SourceCitation{
			{SourceXRef: "@S-SNOTE@"},
		},
		Tags: []*Tag{
			{Tag: "_SN", XRef: "@T-SNOTE@"},
		},
	}

	records := []*Record{
		{XRef: "@I1@", Type: RecordTypeIndividual, Entity: individual,
			Tags: []*Tag{{Tag: "_REC", XRef: "@T-REC@"}}},
		{XRef: "@F1@", Type: RecordTypeFamily, Entity: family},
		{XRef: "@S1@", Type: RecordTypeSource, Entity: source},
		{XRef: "@R1@", Type: RecordTypeRepository, Entity: repo},
		{XRef: "@N1@", Type: RecordTypeNote, Entity: note},
		{XRef: "@M1@", Type: RecordTypeMedia, Entity: media},
		{XRef: "@U1@", Type: RecordTypeSubmitter, Entity: submitter},
		{XRef: "@SN1@", Type: RecordTypeSharedNote, Entity: snote},
		nil, // exercise nil-record skip
	}

	header := &Header{
		Submitter: "@U1@",
		Tags: []*Tag{
			{Tag: "_HDR", XRef: "@T-HDR@"},
			{Tag: "_HDV", Value: "@T-HDR-VAL@"},
			nil,
		},
	}

	xrefMap := make(map[string]*Record)
	for _, r := range records {
		if r != nil && r.XRef != "" {
			xrefMap[r.XRef] = r
		}
	}

	return &Document{
		Header:  header,
		Records: records,
		XRefMap: xrefMap,
	}
}

// xrefwalkExpectedRefs lists every pointer-shaped XRef reference the Visit
// walker should emit for xrefwalkFullDocument (definition sites excluded).
// Used by both Visit and Apply tests.
func xrefwalkExpectedRefs() []string {
	return []string{
		// Individual record
		"@F-CHILD@", "@F-SPOUSE@", "@N-IND@",
		"@I-ASSOC@", "@N-ASSOC@", "@S-ASSOC@",
		"@S-IND@", "@M-IND@",
		"@N-EVENT@", "@S-EVENT@", "@M-EVENT@", "@T-EVENT@",
		"@S-ATTR@", "@F-LDS@",
		"@T-IND-XREF@", "@T-IND-VAL@",
		"@T-REC@",
		// Family record
		"@I-HUSB@", "@I-WIFE@", "@I-CHILD1@", "@I-CHILD2@",
		"@N-FAM@", "@S-FAM@", "@M-FAM@",
		"@N-FAM-EVENT@", "@F-FAM-LDS@", "@T-FAM@",
		// Source record
		"@R-SRC@", "@N-SRC@", "@M-SRC@", "@T-SRC@",
		// Repository record
		"@N-REPO@", "@T-REPO@",
		// Note record
		"@T-NOTE@",
		// MediaObject record
		"@N-MEDIA@", "@S-MEDIA@", "@T-MEDIA@",
		// Submitter record
		"@N-SUBM@", "@T-SUBM@",
		// SharedNote record
		"@S-SNOTE@", "@T-SNOTE@",
	}
}

func xrefwalkSortedUnique(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func TestVisit_CoversAllEntities(t *testing.T) {
	doc := xrefwalkFullDocument()
	var got []string
	for _, r := range doc.Records {
		Visit(r, func(s string) {
			got = append(got, s)
		})
	}

	gotSorted := xrefwalkSortedUnique(got)
	wantSorted := xrefwalkSortedUnique(xrefwalkExpectedRefs())

	if len(gotSorted) != len(wantSorted) {
		t.Fatalf("visit count mismatch: got %d unique refs, want %d\n  got:  %v\n  want: %v",
			len(gotSorted), len(wantSorted), gotSorted, wantSorted)
	}
	for i := range gotSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Errorf("ref[%d] = %q, want %q", i, gotSorted[i], wantSorted[i])
		}
	}
}

func TestVisit_SkipsVoidAndNonPointer(t *testing.T) {
	individual := &Individual{
		XRef:             "@I1@",
		SpouseInFamilies: []string{"@VOID@", "inline note text", "", "@F-REAL@"},
		Notes:            []string{"@VOID@", "literal text", "@N-REAL@"},
	}
	rec := &Record{XRef: "@I1@", Type: RecordTypeIndividual, Entity: individual}

	var seen []string
	Visit(rec, func(s string) { seen = append(seen, s) })

	want := map[string]bool{"@F-REAL@": true, "@N-REAL@": true}
	if len(seen) != len(want) {
		t.Fatalf("expected %d refs, got %d: %v", len(want), len(seen), seen)
	}
	for _, s := range seen {
		if !want[s] {
			t.Errorf("unexpected ref %q", s)
		}
	}
}

func TestVisit_NilRecordAndCallback(t *testing.T) {
	// nil record: no panic, no calls
	Visit(nil, func(string) { t.Fatal("visit should not be called for nil record") })
	// nil callback: should not panic
	Visit(&Record{XRef: "@I1@"}, nil)
}

func TestApply_EmptyMappingIsNoop(t *testing.T) {
	doc := xrefwalkFullDocument()
	before := doc.Records[0].Entity.(*Individual).SpouseInFamilies[0]

	Apply(doc, nil)
	Apply(doc, map[string]string{})

	after := doc.Records[0].Entity.(*Individual).SpouseInFamilies[0]
	if before != after {
		t.Errorf("empty mapping mutated state: before=%q after=%q", before, after)
	}
}

func TestApply_NilDocument(t *testing.T) {
	// Should not panic.
	Apply(nil, map[string]string{"@I1@": "@I2@"})
}

func TestApply_RewritesEverything(t *testing.T) {
	doc := xrefwalkFullDocument()

	// Build a mapping that renames every XRef the Visit walker emits,
	// plus every definition site and header reference.
	defs := []string{"@I1@", "@F1@", "@S1@", "@R1@", "@N1@", "@M1@", "@U1@", "@SN1@"}
	headerRefs := []string{"@T-HDR@", "@T-HDR-VAL@"}
	mapping := make(map[string]string)
	for _, ref := range xrefwalkExpectedRefs() {
		mapping[ref] = ref + "X"
	}
	for _, d := range defs {
		mapping[d] = d + "X"
	}
	for _, h := range headerRefs {
		mapping[h] = h + "X"
	}

	Apply(doc, mapping)

	// Definition sites: Record.XRef and entity XRef.
	wantDefs := map[string]bool{
		"@I1@X": true, "@F1@X": true, "@S1@X": true, "@R1@X": true,
		"@N1@X": true, "@M1@X": true, "@U1@X": true, "@SN1@X": true,
	}
	for _, r := range doc.Records {
		if r == nil {
			continue
		}
		if !wantDefs[r.XRef] {
			t.Errorf("Record.XRef = %q, not in expected set", r.XRef)
		}
		// XRefMap should map to the same record under the new key.
		if doc.XRefMap[r.XRef] != r {
			t.Errorf("XRefMap[%q] does not point to record after Apply", r.XRef)
		}
	}

	// Entity XRefs.
	ind := doc.Records[0].Entity.(*Individual)
	if ind.XRef != "@I1@X" {
		t.Errorf("Individual.XRef = %q, want @I1@X", ind.XRef)
	}
	fam := doc.Records[1].Entity.(*Family)
	if fam.XRef != "@F1@X" {
		t.Errorf("Family.XRef = %q, want @F1@X", fam.XRef)
	}

	// After Apply, Visit should see only mapped values (with "X" suffix).
	var seen []string
	for _, r := range doc.Records {
		Visit(r, func(s string) { seen = append(seen, s) })
	}
	for _, s := range seen {
		// Either it's a mapped value (ends in X) or it wasn't in mapping.
		if _, isMapped := mapping[s]; isMapped {
			t.Errorf("Visit returned an unmapped pre-image %q after Apply", s)
		}
	}

	// Spot-check specific deep fields to make sure each per-entity branch
	// actually rewrote.
	if got := ind.ChildInFamilies[0].FamilyXRef; got != "@F-CHILD@X" {
		t.Errorf("Individual.ChildInFamilies[0].FamilyXRef = %q", got)
	}
	if got := ind.Associations[0].SourceCitations[0].SourceXRef; got != "@S-ASSOC@X" {
		t.Errorf("Association.SourceCitations[0].SourceXRef = %q", got)
	}
	if got := ind.Events[0].Tags[0].XRef; got != "@T-EVENT@X" {
		t.Errorf("Event.Tags[0].XRef = %q", got)
	}
	if got := ind.LDSOrdinances[0].FamilyXRef; got != "@F-LDS@X" {
		t.Errorf("Individual.LDSOrdinances[0].FamilyXRef = %q", got)
	}
	if got := fam.Husband; got != "@I-HUSB@X" {
		t.Errorf("Family.Husband = %q", got)
	}
	if got := fam.Children[1]; got != "@I-CHILD2@X" {
		t.Errorf("Family.Children[1] = %q", got)
	}
	src := doc.Records[2].Entity.(*Source)
	if got := src.RepositoryRef; got != "@R-SRC@X" {
		t.Errorf("Source.RepositoryRef = %q", got)
	}
	if got := src.Tags[0].Value; got != "@T-SRC@X" {
		t.Errorf("Source.Tags[0].Value = %q", got)
	}
	subm := doc.Records[6].Entity.(*Submitter)
	if got := subm.Tags[0].Value; got != "@T-SUBM@X" {
		t.Errorf("Submitter.Tags[0].Value = %q", got)
	}
	if got := doc.Header.Submitter; got != "@U1@X" {
		t.Errorf("Header.Submitter = %q", got)
	}
	if got := doc.Header.Tags[0].XRef; got != "@T-HDR@X" {
		t.Errorf("Header.Tags[0].XRef = %q", got)
	}
	if got := doc.Header.Tags[1].Value; got != "@T-HDR-VAL@X" {
		t.Errorf("Header.Tags[1].Value = %q", got)
	}
}

func TestApply_UpdatesXRefMapKeys(t *testing.T) {
	doc := xrefwalkFullDocument()
	mapping := map[string]string{"@I1@": "@I999@"}

	originalRec := doc.XRefMap["@I1@"]
	if originalRec == nil {
		t.Fatal("test setup: @I1@ missing from XRefMap")
	}

	Apply(doc, mapping)

	if _, exists := doc.XRefMap["@I1@"]; exists {
		t.Error("old key @I1@ should be gone from XRefMap")
	}
	got, ok := doc.XRefMap["@I999@"]
	if !ok {
		t.Fatal("new key @I999@ missing from XRefMap")
	}
	if got != originalRec {
		t.Error("XRefMap[@I999@] points to a different record than expected")
	}
	// Unmapped keys are preserved.
	if _, ok := doc.XRefMap["@F1@"]; !ok {
		t.Error("unmapped key @F1@ should still be present")
	}
}

func TestApply_HandlesTagXRefAndTagValue(t *testing.T) {
	// Tag.XRef path: parser-populated XRef field.
	// Tag.Value path: converter-populated XRef-shaped value.
	individual := &Individual{
		XRef: "@I1@",
		Tags: []*Tag{
			{Tag: "_REFA", XRef: "@A@"},
			{Tag: "_REFB", Value: "@B@"},
			{Tag: "_REFC", XRef: "@C@", Value: "@D@"},
			{Tag: "_PLAIN", Value: "not an xref"},
		},
	}
	rec := &Record{XRef: "@I1@", Type: RecordTypeIndividual, Entity: individual}
	doc := &Document{
		Records: []*Record{rec},
		XRefMap: map[string]*Record{"@I1@": rec},
	}

	Apply(doc, map[string]string{
		"@A@": "@A2@",
		"@B@": "@B2@",
		"@C@": "@C2@",
		"@D@": "@D2@",
	})

	if individual.Tags[0].XRef != "@A2@" {
		t.Errorf("Tags[0].XRef = %q, want @A2@", individual.Tags[0].XRef)
	}
	if individual.Tags[1].Value != "@B2@" {
		t.Errorf("Tags[1].Value = %q, want @B2@", individual.Tags[1].Value)
	}
	if individual.Tags[2].XRef != "@C2@" {
		t.Errorf("Tags[2].XRef = %q, want @C2@", individual.Tags[2].XRef)
	}
	if individual.Tags[2].Value != "@D2@" {
		t.Errorf("Tags[2].Value = %q, want @D2@", individual.Tags[2].Value)
	}
	if individual.Tags[3].Value != "not an xref" {
		t.Errorf("plain Tag.Value mutated: %q", individual.Tags[3].Value)
	}
}

func TestVisit_RoundTripAfterApply(t *testing.T) {
	doc := xrefwalkFullDocument()
	mapping := map[string]string{
		"@F-CHILD@":   "@F-CHILD-NEW@",
		"@I-HUSB@":    "@I-HUSB-NEW@",
		"@S-SNOTE@":   "@S-SNOTE-NEW@",
		"@T-IND-VAL@": "@T-IND-VAL-NEW@",
	}

	Apply(doc, mapping)

	// After Apply, Visit should see the mapped image and not the pre-image.
	seen := make(map[string]bool)
	for _, r := range doc.Records {
		Visit(r, func(s string) { seen[s] = true })
	}
	for old, mapped := range mapping {
		if seen[old] {
			t.Errorf("Visit still sees pre-image %q after Apply", old)
		}
		if !seen[mapped] {
			t.Errorf("Visit does not see mapped value %q after Apply", mapped)
		}
	}
}

func TestApply_NoMappingMatchPreservesValue(t *testing.T) {
	doc := xrefwalkFullDocument()
	Apply(doc, map[string]string{"@NEVER-PRESENT@": "@OTHER@"})

	// Spot check: nothing changed.
	ind := doc.Records[0].Entity.(*Individual)
	if ind.SpouseInFamilies[0] != "@F-SPOUSE@" {
		t.Errorf("unrelated field mutated: %q", ind.SpouseInFamilies[0])
	}
	if _, ok := doc.XRefMap["@I1@"]; !ok {
		t.Error("XRefMap key @I1@ was unexpectedly removed")
	}
}

func TestApply_NilEntityAndUnknownType(t *testing.T) {
	rec := &Record{XRef: "@X@", Type: "WEIRD", Entity: nil}
	rec2 := &Record{XRef: "@Y@", Type: "WEIRD", Entity: struct{ Foo string }{Foo: "bar"}}
	doc := &Document{
		Records: []*Record{rec, rec2},
		XRefMap: map[string]*Record{"@X@": rec, "@Y@": rec2},
	}
	// Should not panic.
	Apply(doc, map[string]string{"@X@": "@XX@", "@Y@": "@YY@"})

	if rec.XRef != "@XX@" {
		t.Errorf("Record.XRef = %q, want @XX@", rec.XRef)
	}
	if rec2.XRef != "@YY@" {
		t.Errorf("Record.XRef = %q, want @YY@", rec2.XRef)
	}
}

func TestApply_NilTypedEntity(t *testing.T) {
	// Each entity case with a typed nil should not panic.
	var (
		ind   *Individual
		fam   *Family
		src   *Source
		repo  *Repository
		note  *Note
		media *MediaObject
		subm  *Submitter
		snote *SharedNote
	)
	entities := []interface{}{ind, fam, src, repo, note, media, subm, snote}
	for _, e := range entities {
		rec := &Record{XRef: "@Z@", Entity: e}
		doc := &Document{
			Records: []*Record{rec},
			XRefMap: map[string]*Record{"@Z@": rec},
		}
		Apply(doc, map[string]string{"@Z@": "@ZZ@"})
		Visit(rec, func(string) {})
	}
}
