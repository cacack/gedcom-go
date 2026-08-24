package gedcom

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// XRef reachability: every pointer a typed entity holds must be reachable
// from Visit, and rewritten by Apply.
//
// This is the counterpart to TestCloneCompleteness. That one guards against
// a new field being dropped by a clone; this one guards against a new field
// being invisible to the XRef walker. Both are the same defect shape -- a
// field added to an entity and not added to the code that has to traverse it
// -- and both are silent until a caller loses data.
//
// The failure this pins actually shipped: Family.Attributes, and the
// Associations / NoteXRefs added alongside it, were unreachable from
// walkFamily / walkEvent / walkAttribute, so merge.RemapXRefs left every
// pointer inside a family attribute pointing at the pre-merge XRef. When two
// documents both used @S1@ that is not a dangling pointer but a silent
// resolution to the other document's record.
//
// NoteXRefs is the subtle half: it was walked nowhere at all, for any type,
// since the note split was introduced. It stayed harmless only because the
// legacy Notes slice carries the same pointers and is walked, so a remap
// still fixed the value the encoder happened to read. Once the encoder began
// preferring the split fields when the two disagree, remapping one and not
// the other made the stale copy win.

// fullyPopulated builds one record of each entity type with an XRef-shaped
// value in every pointer-bearing field the walker is expected to reach. Add
// to this when you add a pointer field; the test then tells you whether the
// walker needs updating too.
func fullyPopulatedRecords() []*Record {
	cite := func(x string) *SourceCitation {
		return &SourceCitation{
			SourceXRef: x,
			NoteXRefs:  []string{"@CITE_NX@"},
			Notes:      []string{"@CITE_NOTE@"},
		}
	}
	assoc := func(x string) *Association {
		return &Association{
			IndividualXRef:  x,
			Notes:           []string{"@ASSOC_NOTE@"},
			SourceCitations: []*SourceCitation{cite("@ASSOC_SOUR@")},
		}
	}
	// rawTags stands in for the Record.Tags backstop. walkTag visits both
	// Tag.XRef and Tag.Value, and this is the path that keeps Subset's
	// closure correct for anything the typed model does not cover -- so a
	// regression here would be invisible everywhere else.
	rawTags := func(prefix string) []*Tag {
		return []*Tag{{Level: 1, Tag: "NOTE", Value: "@" + prefix + "_TAG_VAL@", XRef: "@" + prefix + "_TAG_XREF@"}}
	}
	attr := func(prefix string) *Attribute {
		return &Attribute{
			Type:            "NCHI",
			Value:           "3",
			SourceCitations: []*SourceCitation{cite("@" + prefix + "_ATTR_SOUR@")},
			Media:           []*MediaLink{{MediaXRef: "@" + prefix + "_ATTR_OBJE@"}},
			Associations:    []*Association{assoc("@" + prefix + "_ATTR_ASSO@")},
			NoteXRefs:       []string{"@" + prefix + "_ATTR_NX@"},
			Notes:           []string{"@" + prefix + "_ATTR_NOTE@"},
		}
	}
	event := func(prefix string) *Event {
		return &Event{
			Type:            "BIRT",
			SourceCitations: []*SourceCitation{cite("@" + prefix + "_EV_SOUR@")},
			Media:           []*MediaLink{{MediaXRef: "@" + prefix + "_EV_OBJE@"}},
			Associations:    []*Association{assoc("@" + prefix + "_EV_ASSO@")},
			NoteXRefs:       []string{"@" + prefix + "_EV_NX@"},
			Notes:           []string{"@" + prefix + "_EV_NOTE@"},
			Tags:            rawTags(prefix + "_EV"),
		}
	}
	chg := func(prefix string) *ChangeDate {
		return &ChangeDate{
			Date:      "1 JAN 2020",
			NoteXRefs: []string{"@" + prefix + "_CHAN_NX@"},
			Notes:     []string{"@" + prefix + "_CHAN_NOTE@"},
		}
	}
	ord := func(prefix string) *LDSOrdinance {
		return &LDSOrdinance{
			FamilyXRef: "@" + prefix + "_ORD_FAMC@",
			NoteXRefs:  []string{"@" + prefix + "_ORD_NX@"},
			Notes:      []string{"@" + prefix + "_ORD_NOTE@"},
		}
	}

	indi := &Individual{
		XRef:             "@I1@",
		ChildInFamilies:  []FamilyLink{{FamilyXRef: "@I_FAMC@"}},
		SpouseInFamilies: []string{"@I_FAMS@"},
		NoteXRefs:        []string{"@I_NX@"},
		Notes:            []string{"@I_NOTE@"},
		Associations:     []*Association{assoc("@I_ASSO@")},
		SourceCitations:  []*SourceCitation{cite("@I_SOUR@")},
		Media:            []*MediaLink{{MediaXRef: "@I_OBJE@"}},
		Events:           []*Event{event("I")},
		Attributes:       []*Attribute{attr("I")},
		LDSOrdinances:    []*LDSOrdinance{ord("I")},
		ChangeDate:       chg("I"),
		CreationDate:     chg("ICREA"),
		Tags:             rawTags("I"),
	}

	fam := &Family{
		XRef:            "@F1@",
		Husband:         "@F_HUSB@",
		Wife:            "@F_WIFE@",
		Children:        []string{"@F_CHIL@"},
		NoteXRefs:       []string{"@F_NX@"},
		Notes:           []string{"@F_NOTE@"},
		SourceCitations: []*SourceCitation{cite("@F_SOUR@")},
		Media:           []*MediaLink{{MediaXRef: "@F_OBJE@"}},
		Events:          []*Event{event("F")},
		Attributes:      []*Attribute{attr("F")},
		LDSOrdinances:   []*LDSOrdinance{ord("F")},
		ChangeDate:      chg("F"),
		CreationDate:    chg("FCREA"),
		Tags:            rawTags("F"),
	}

	src := &Source{
		XRef:           "@S1@",
		RepositoryLink: &SourceRepositoryLink{XRef: "@S_REPO@", Notes: []string{"@S_REPO_NOTE@"}},
		NoteXRefs:      []string{"@S_NX@"},
		Notes:          []string{"@S_NOTE@"},
		Media:          []*MediaLink{{MediaXRef: "@S_OBJE@"}},
		ChangeDate:     chg("S"),
		CreationDate:   chg("SCREA"),
		Tags:           rawTags("S"),
	}

	obje := &MediaObject{
		XRef:            "@M1@",
		NoteXRefs:       []string{"@M_NX@"},
		Notes:           []string{"@M_NOTE@"},
		SourceCitations: []*SourceCitation{cite("@M_SOUR@")},
		ChangeDate:      chg("M"),
		CreationDate:    chg("MCREA"),
		SharedNoteXRefs: []string{"@M_SNOTE@"},
		Tags:            rawTags("M"),
	}

	repo := &Repository{XRef: "@R1@", NoteXRefs: []string{"@R_NX@"},
		Notes: []string{"@R_NOTE@"}, Tags: rawTags("R")}
	subm := &Submitter{XRef: "@U1@", NoteXRefs: []string{"@U_NX@"},
		Notes: []string{"@U_NOTE@"}, Tags: rawTags("U")}
	snote := &SharedNote{
		XRef:            "@N1@",
		SourceCitations: []*SourceCitation{cite("@N_SOUR@")},
		ChangeDate:      chg("N"),
		Tags:            rawTags("N"),
	}

	return []*Record{
		{XRef: "@I1@", Type: RecordTypeIndividual, Entity: indi},
		{XRef: "@F1@", Type: RecordTypeFamily, Entity: fam},
		{XRef: "@S1@", Type: RecordTypeSource, Entity: src},
		{XRef: "@M1@", Type: RecordTypeMedia, Entity: obje},
		{XRef: "@R1@", Type: RecordTypeRepository, Entity: repo},
		{XRef: "@U1@", Type: RecordTypeSubmitter, Entity: subm},
		{XRef: "@N1@", Type: RecordTypeSharedNote, Entity: snote},
	}
}

// plantedPointers reflect-walks the fixture and returns every pointer-shaped
// string it contains, keyed by the field path that holds it.
//
// Deriving this rather than hand-listing it is the point. A written-out list
// only asserts "want is a subset of seen", so a value planted in the fixture
// and forgotten in the list is silently uncounted -- the assertion passes
// while the walker never touched it.
//
// Definition sites are excluded: Visit deliberately does not report the XRef
// that names a record. InlineNotes is excluded because it holds note text,
// where an XRef-shaped string is payload rather than a reference.
func plantedPointers(t *testing.T) map[string]string {
	t.Helper()
	planted := map[string]string{}

	var collect func(v reflect.Value, path string, depth int)
	collect = func(v reflect.Value, path string, depth int) {
		if depth > 8 || !v.IsValid() {
			return
		}
		switch v.Kind() {
		case reflect.Ptr:
			if !v.IsNil() {
				collect(v.Elem(), path, depth+1)
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				collect(v.Index(i), fmt.Sprintf("%s[%d]", path, i), depth+1)
			}
		case reflect.String:
			if IsPointerXRef(v.String()) {
				planted[path] = v.String()
			}
		case reflect.Struct:
			rt := v.Type()
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				if f.PkgPath != "" {
					continue
				}
				if isDefinitionSite(rt, f.Name) || f.Name == "InlineNotes" {
					continue // definition site / inline text
				}
				collect(v.Field(i), path+"."+f.Name, depth+1)
			}
		}
	}

	for _, r := range fullyPopulatedRecords() {
		collect(reflect.ValueOf(r.Entity), reflect.TypeOf(r.Entity).Elem().Name(), 0)
	}
	if len(planted) == 0 {
		t.Fatal("fixture planted no pointers; the collector is broken, not the walker")
	}
	return planted
}

// isDefinitionSite reports whether Type.Field is the XRef that *names* a
// record, which Visit deliberately does not report.
//
// Skipping by bare field name would also skip SourceRepositoryLink.XRef and
// Tag.XRef, which are genuine pointers rather than definition sites -- so the
// harness would silently stop asserting anything about the REPO link, which
// is the one path the fixture guard's exemption list points at as covered.
func isDefinitionSite(rt reflect.Type, field string) bool {
	if field != "XRef" {
		return false
	}
	switch rt.Name() {
	case "Individual", "Family", "Source", "Repository",
		"Note", "MediaObject", "Submitter", "SharedNote":
		return true
	}
	return false
}

func TestVisitReachesEveryPointerField(t *testing.T) {
	planted := plantedPointers(t)

	seen := map[string]bool{}
	for _, r := range fullyPopulatedRecords() {
		Visit(r, func(ref string) { seen[ref] = true })
	}

	var missing []string
	for path, ref := range planted {
		if !seen[ref] {
			missing = append(missing, fmt.Sprintf("%s = %s", path, ref))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("Visit did not reach %d of %d planted pointer(s):\n  %s\n\n"+
			"A pointer the walker cannot see is a pointer merge.RemapXRefs will not "+
			"rewrite and Document.Subset will not pull into its closure.",
			len(missing), len(planted), strings.Join(missing, "\n  "))
	}
}

func TestApplyRewritesEveryPointerField(t *testing.T) {
	// The mapping is derived from what the fixture plants, NOT from what
	// Visit reports. Deriving it from Visit would make the test circular: it
	// could never catch a pointer that Visit fails to reach, which is the
	// only interesting failure.
	planted := plantedPointers(t)
	mapping := map[string]string{}
	for _, ref := range planted {
		mapping[ref] = "@X" + ref[1:]
	}

	records := fullyPopulatedRecords()
	Apply(&Document{Records: records}, mapping)

	// Re-derive from the mutated fixture: any planted value still present is
	// one Apply failed to rewrite.
	var stale []string
	var check func(v reflect.Value, path string, depth int)
	check = func(v reflect.Value, path string, depth int) {
		if depth > 8 || !v.IsValid() {
			return
		}
		switch v.Kind() {
		case reflect.Ptr:
			if !v.IsNil() {
				check(v.Elem(), path, depth+1)
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				check(v.Index(i), fmt.Sprintf("%s[%d]", path, i), depth+1)
			}
		case reflect.String:
			if _, wasPlanted := mapping[v.String()]; wasPlanted {
				stale = append(stale, fmt.Sprintf("%s = %s", path, v.String()))
			}
		case reflect.Struct:
			rt := v.Type()
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				if f.PkgPath != "" || isDefinitionSite(rt, f.Name) || f.Name == "InlineNotes" {
					continue
				}
				check(v.Field(i), path+"."+f.Name, depth+1)
			}
		}
	}
	for _, r := range records {
		check(reflect.ValueOf(r.Entity), reflect.TypeOf(r.Entity).Elem().Name(), 0)
	}

	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("Apply left %d pointer(s) un-rewritten:\n  %s", len(stale), strings.Join(stale, "\n  "))
	}
}
