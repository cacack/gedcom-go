package gedcom

import (
	"reflect"
	"testing"
)

// buildGenealogyFixture constructs a small multi-generation tree:
//
//	@I1@ (grandpa) + @I2@ (grandma) -> @F1@
//	  child: @I3@ (parent) + @I4@ (parent-in-law) -> @F2@
//	    children: @I5@ (sibling-A), @I6@ (sibling-B)
//	  child: @I7@ (uncle, no spouse, no children)
//
// Half-sibling structure: @I3@ also has a second marriage with @I8@ -> @F3@
//
//	child: @I9@ (half-sibling)
func buildGenealogyFixture() *Document {
	doc := &Document{
		Header:  &Header{Version: Version551, Encoding: EncodingUTF8},
		XRefMap: make(map[string]*Record),
	}

	addIndividual := func(xref string, fams []string, famc []string) {
		ind := &Individual{XRef: xref, SpouseInFamilies: fams}
		for _, f := range famc {
			ind.ChildInFamilies = append(ind.ChildInFamilies, FamilyLink{FamilyXRef: f})
		}
		rec := &Record{XRef: xref, Type: RecordTypeIndividual, Entity: ind}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	addFamily := func(xref, husband, wife string, children ...string) {
		fam := &Family{XRef: xref, Husband: husband, Wife: wife, Children: children}
		rec := &Record{XRef: xref, Type: RecordTypeFamily, Entity: fam}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	addIndividual("@I1@", []string{"@F1@"}, nil)                      // grandpa
	addIndividual("@I2@", []string{"@F1@"}, nil)                      // grandma
	addIndividual("@I3@", []string{"@F2@", "@F3@"}, []string{"@F1@"}) // parent
	addIndividual("@I4@", []string{"@F2@"}, nil)                      // parent-in-law
	addIndividual("@I5@", nil, []string{"@F2@"})                      // sibling-A
	addIndividual("@I6@", nil, []string{"@F2@"})                      // sibling-B
	addIndividual("@I7@", nil, []string{"@F1@"})                      // uncle
	addIndividual("@I8@", []string{"@F3@"}, nil)                      // second spouse
	addIndividual("@I9@", nil, []string{"@F3@"})                      // half-sibling

	addFamily("@F1@", "@I1@", "@I2@", "@I3@", "@I7@")
	addFamily("@F2@", "@I3@", "@I4@", "@I5@", "@I6@")
	addFamily("@F3@", "@I3@", "@I8@", "@I9@")

	return doc
}

func TestDescendants_Multigeneration(t *testing.T) {
	doc := buildGenealogyFixture()
	got := doc.Descendants("@I1@")

	// Exact BFS order from @I1@: gen1 [@I3@, @I7@] (children of @F1@ in
	// declared order), then gen2 reached via @I3@'s spouse families —
	// [@I5@, @I6@] from @F2@, then [@I9@] from @F3@. @I7@ has no
	// children so contributes nothing to gen2.
	want := []string{"@I3@", "@I7@", "@I5@", "@I6@", "@I9@"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(@I1@) = %v, want %v", got, want)
	}
}

func TestDescendants_ExcludesSeed(t *testing.T) {
	doc := buildGenealogyFixture()
	for _, x := range doc.Descendants("@I1@") {
		if x == "@I1@" {
			t.Error("Descendants should not include the seed individual")
		}
	}
}

func TestDescendants_LeafReturnsEmpty(t *testing.T) {
	doc := buildGenealogyFixture()
	got := doc.Descendants("@I5@")
	if got == nil {
		t.Error("Descendants of valid leaf individual should be non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("Descendants of leaf individual = %v, want empty", got)
	}
}

func TestDescendants_HalfSiblingsViaSecondMarriage(t *testing.T) {
	doc := buildGenealogyFixture()
	got := doc.Descendants("@I3@")
	// @I3@ has two spouse families: @F2@ (children [@I5@, @I6@]) then
	// @F3@ (child [@I9@]). BFS visits them in SpouseInFamilies order.
	want := []string{"@I5@", "@I6@", "@I9@"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Descendants(@I3@) = %v, want %v", got, want)
	}
}

func TestDescendants_UnknownXRef(t *testing.T) {
	doc := buildGenealogyFixture()
	if got := doc.Descendants("@I999@"); got != nil {
		t.Errorf("Descendants of unknown xref should be nil, got %v", got)
	}
}

func TestDescendants_NonIndividualXRef(t *testing.T) {
	doc := buildGenealogyFixture()
	if got := doc.Descendants("@F1@"); got != nil {
		t.Errorf("Descendants of non-individual xref should be nil, got %v", got)
	}
}

func TestDescendants_EmptyXRef(t *testing.T) {
	doc := buildGenealogyFixture()
	if got := doc.Descendants(""); got != nil {
		t.Errorf("Descendants of empty xref should be nil, got %v", got)
	}
}

func TestDescendants_NilDocument(t *testing.T) {
	var doc *Document
	if got := doc.Descendants("@I1@"); got != nil {
		t.Errorf("Descendants on nil doc should be nil, got %v", got)
	}
}

func TestDescendants_CycleSafety(t *testing.T) {
	// Construct a malformed but possible structure: @I1@ is both ancestor
	// and descendant of @I2@. Subset relies on visited-set termination.
	doc := &Document{Header: &Header{Version: Version551}, XRefMap: make(map[string]*Record)}
	addInd := func(xref string, fams []string, famc []string) {
		ind := &Individual{XRef: xref, SpouseInFamilies: fams}
		for _, f := range famc {
			ind.ChildInFamilies = append(ind.ChildInFamilies, FamilyLink{FamilyXRef: f})
		}
		rec := &Record{XRef: xref, Type: RecordTypeIndividual, Entity: ind}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}
	addFam := func(xref, h, w string, children ...string) {
		fam := &Family{XRef: xref, Husband: h, Wife: w, Children: children}
		rec := &Record{XRef: xref, Type: RecordTypeFamily, Entity: fam}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[xref] = rec
	}

	addInd("@I1@", []string{"@F1@"}, []string{"@F2@"})
	addInd("@I2@", []string{"@F2@"}, []string{"@F1@"})
	addFam("@F1@", "@I1@", "", "@I2@")
	addFam("@F2@", "@I2@", "", "@I1@") // cycle: @I1@'s child @I2@ has child @I1@

	got := doc.Descendants("@I1@")
	if len(got) != 1 || got[0] != "@I2@" {
		t.Errorf("Cycle termination broken: Descendants(@I1@) = %v, want [@I2@]", got)
	}
}

func TestAncestors_Multigeneration(t *testing.T) {
	doc := buildGenealogyFixture()
	got := doc.Ancestors("@I5@")
	// BFS from @I5@: parents [@I3@, @I4@] at @F2@ (husband-before-wife),
	// then grandparents [@I1@, @I2@] reached via @I3@'s @F1@. @I4@ has
	// no parent families so contributes nothing to gen 2.
	want := []string{"@I3@", "@I4@", "@I1@", "@I2@"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Ancestors(@I5@) = %v, want %v", got, want)
	}
}

func TestAncestors_ExcludesSeed(t *testing.T) {
	doc := buildGenealogyFixture()
	for _, x := range doc.Ancestors("@I5@") {
		if x == "@I5@" {
			t.Error("Ancestors should not include the seed individual")
		}
	}
}

func TestAncestors_NoParentsReturnsEmpty(t *testing.T) {
	doc := buildGenealogyFixture()
	got := doc.Ancestors("@I1@")
	if got == nil {
		t.Error("Ancestors of valid root individual should be non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("Ancestors of root individual = %v, want empty", got)
	}
}

func TestAncestors_UnknownXRef(t *testing.T) {
	doc := buildGenealogyFixture()
	if got := doc.Ancestors("@I999@"); got != nil {
		t.Errorf("Ancestors of unknown xref should be nil, got %v", got)
	}
}

func TestAncestors_NonIndividualXRef(t *testing.T) {
	doc := buildGenealogyFixture()
	if got := doc.Ancestors("@F1@"); got != nil {
		t.Errorf("Ancestors of non-individual xref should be nil, got %v", got)
	}
}

func TestAncestors_NilDocument(t *testing.T) {
	var doc *Document
	if got := doc.Ancestors("@I1@"); got != nil {
		t.Errorf("Ancestors on nil doc should be nil, got %v", got)
	}
}

func TestIndividualDescendants_Wrapper(t *testing.T) {
	doc := buildGenealogyFixture()
	i1 := doc.GetIndividual("@I1@")
	if i1 == nil {
		t.Fatal("setup: @I1@ not found")
	}

	individuals := i1.Descendants(doc)
	if len(individuals) == 0 {
		t.Fatal("Individual.Descendants returned empty")
	}

	// Compare against xref-returning sibling for parity.
	xrefs := doc.Descendants("@I1@")
	gotXRefs := make([]string, len(individuals))
	for i, ind := range individuals {
		gotXRefs[i] = ind.XRef
	}
	if !reflect.DeepEqual(gotXRefs, xrefs) {
		t.Errorf("Individual.Descendants order/contents diverge from Document.Descendants:\n got %v\nwant %v", gotXRefs, xrefs)
	}
}

func TestIndividualAncestors_Wrapper(t *testing.T) {
	doc := buildGenealogyFixture()
	i5 := doc.GetIndividual("@I5@")
	if i5 == nil {
		t.Fatal("setup: @I5@ not found")
	}

	individuals := i5.Ancestors(doc)
	xrefs := doc.Ancestors("@I5@")
	gotXRefs := make([]string, len(individuals))
	for i, ind := range individuals {
		gotXRefs[i] = ind.XRef
	}
	if !reflect.DeepEqual(gotXRefs, xrefs) {
		t.Errorf("Individual.Ancestors order/contents diverge from Document.Ancestors:\n got %v\nwant %v", gotXRefs, xrefs)
	}
}

func TestIndividualDescendants_NilReceiver(t *testing.T) {
	doc := buildGenealogyFixture()
	var i *Individual
	if got := i.Descendants(doc); got != nil {
		t.Errorf("nil receiver should return nil, got %v", got)
	}
}

func TestIndividualAncestors_NilReceiver(t *testing.T) {
	doc := buildGenealogyFixture()
	var i *Individual
	if got := i.Ancestors(doc); got != nil {
		t.Errorf("nil receiver should return nil, got %v", got)
	}
}

func TestIndividualDescendants_NilDoc(t *testing.T) {
	doc := buildGenealogyFixture()
	i1 := doc.GetIndividual("@I1@")
	if got := i1.Descendants(nil); got != nil {
		t.Errorf("nil doc should return nil, got %v", got)
	}
}

func TestIndividualAncestors_NilDoc(t *testing.T) {
	doc := buildGenealogyFixture()
	i5 := doc.GetIndividual("@I5@")
	if got := i5.Ancestors(nil); got != nil {
		t.Errorf("nil doc should return nil, got %v", got)
	}
}

func TestIndividualDescendants_LeafReturnsEmpty(t *testing.T) {
	doc := buildGenealogyFixture()
	leaf := doc.GetIndividual("@I5@")
	got := leaf.Descendants(doc)
	if got == nil {
		t.Error("Individual.Descendants on leaf (in doc, no descendants) should be non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("Individual.Descendants on leaf = %v, want empty", got)
	}
}

func TestIndividualAncestors_RootReturnsEmpty(t *testing.T) {
	doc := buildGenealogyFixture()
	root := doc.GetIndividual("@I1@")
	got := root.Ancestors(doc)
	if got == nil {
		t.Error("Individual.Ancestors on root (in doc, no ancestors) should be non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("Individual.Ancestors on root = %v, want empty", got)
	}
}

func TestIndividualDescendants_NotInDocReturnsNil(t *testing.T) {
	doc := buildGenealogyFixture()
	orphan := &Individual{XRef: "@NOT_IN_DOC@"}
	if got := orphan.Descendants(doc); got != nil {
		t.Errorf("Individual not in doc should return nil, got %v", got)
	}
}

func TestIndividualAncestors_NotInDocReturnsNil(t *testing.T) {
	doc := buildGenealogyFixture()
	orphan := &Individual{XRef: "@NOT_IN_DOC@"}
	if got := orphan.Ancestors(doc); got != nil {
		t.Errorf("Individual not in doc should return nil, got %v", got)
	}
}
