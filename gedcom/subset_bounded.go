package gedcom

import (
	"errors"
	"fmt"
	"slices"
)

// ErrNotIndividual is returned by BoundedSubset when a seed XRef names a
// record that is not an individual. Use errors.Is to detect it.
var ErrNotIndividual = errors.New("not an individual")

// DroppedPointer records one pointer BoundedSubset removed from an included
// record because it named an individual or family outside the result.
type DroppedPointer struct {
	// XRef is the included record the pointer was removed from.
	XRef string
	// Tag is the tag of the removed line, such as "CHIL", "HUSB" or "ASSO".
	// It is empty when the record has no raw Tags and the pointer was
	// removed from its typed Entity only.
	Tag string
	// Target is the individual or family the pointer named.
	Target string
	// Restorable is true when merge.ZipSharedXRefs puts the line back on
	// combining this subset with one that holds Target: the line is a
	// family's level-1 HUSB, WIFE or CHIL, which every bounded subset
	// holding that individual also holds. Any other dropped line, such as
	// an ASSO naming an individual outside the set, exists only in a record
	// the other subset lacks, so it is lost from the cut.
	Restorable bool
}

// droppedMarker temporarily replaces a dropped pointer in a typed field, so
// list entries holding one can be told apart from entries that were already
// empty. No pointer can equal it: it is not @-delimited.
const droppedMarker = "\x00dropped"

// BoundedSubset returns a new self-contained document cut from d along the
// boundary of a set of individuals. Where [Document.Subset] follows every
// reference transitively, and so usually pulls in the whole connected tree,
// BoundedSubset stops at individuals and families. The result holds exactly:
//
//   - the individuals named by individuals;
//   - every family with at least one of them as a member: a family that names
//     one as HUSB, WIFE or CHIL, or that one names with FAMC or FAMS;
//   - everything those records reference, transitively, that is not an
//     individual or a family (sources, repositories, notes, shared notes,
//     media, submitters and submissions), and the header's submitter as
//     [Document.Subset] carries it.
//
// A pointer in an included record that names an individual or family outside
// the result is removed, so no included record points outside the result.
// In the raw Tags the line holding the pointer is removed with its
// subordinate lines (a CHIL's _FREL, an ASSO's RELA); in the typed Entity the
// matching entry (Family.Children and ChildLinks, Individual.ChildInFamilies
// and SpouseInFamilies, an Association) is removed and a single pointer field
// (Family.Husband, Event.FamilyXRef) is cleared. Each removed raw line is
// reported as a DroppedPointer, in record and line order; a record that has
// no raw Tags reports each pointer removed from its Entity instead.
// DroppedPointer.Restorable tells which of them zipping can put back.
//
// Seeds are checked strictly, as Subset checks them: each must resolve to a
// record (an error wrapping ErrUnknownXRef otherwise) that is an individual
// (an error wrapping ErrNotIndividual otherwise). Duplicates are ignored. A
// reference from an included record that resolves to nothing is an error
// wrapping ErrUnknownXRef, as in Subset, rather than a dropped pointer: a
// dangling pointer is a defect of d, and dropping it would lose it silently.
//
// The cut is designed to be undone. Records keep their XRefs, and a family
// on the boundary is in the subset of every member, each copy holding the
// HUSB, WIFE and CHIL lines of the members its subset holds. So, when every
// dropped pointer is Restorable, the bounded subsets of S1 and S2 cut from
// one document combine, with merge.ZipSharedXRefs, into the bounded subset
// of S1 ∪ S2; when S1 ∪ S2 holds every individual, and every other record is
// referenced from one, that is d itself. A pointer that is not Restorable
// (an ASSO, or an event's FAMC naming a family the individual is not a
// member of) is lost from the cut: keep the two ends in one set to keep it.
//
// The header, trailer, vendor and schema are carried over as Subset carries
// them, and d is not mutated.
func (d *Document) BoundedSubset(individuals []string) (*Document, []DroppedPointer, error) {
	if d == nil {
		return nil, nil, errors.New("bounded subset: source document is nil")
	}
	closure, err := d.boundedClosure(individuals)
	if err != nil {
		return nil, nil, err
	}
	out := d.subsetDocument(closure)

	// dropped reports whether a pointer value names an individual or family
	// outside the result. The closure walk has already rejected every
	// pointer that resolves to nothing.
	dropped := func(ref string) bool {
		if closure[ref] || !d.IsRecordPointer(ref) {
			return false
		}
		return isPersonOrFamily(d.GetRecord(ref))
	}
	var report []DroppedPointer
	for _, r := range out.Records {
		report = append(report, pruneRecord(r, dropped)...)
	}
	return out, report, nil
}

// boundedClosure computes the XRefs a BoundedSubset of seeds holds.
func (d *Document) boundedClosure(seeds []string) (map[string]bool, error) {
	people, err := d.boundedSeeds(seeds)
	if err != nil {
		return nil, err
	}
	closure := make(map[string]bool, len(people))
	queue := make([]string, 0, len(people))
	add := func(xref string) {
		if !closure[xref] {
			closure[xref] = true
			queue = append(queue, xref)
		}
	}
	for _, xref := range people {
		add(xref)
	}
	for _, xref := range d.memberFamilies(people) {
		add(xref)
	}
	// The header's submitter, as Subset carries it.
	if d.Header != nil {
		subm := d.Header.Submitter
		if d.IsRecordPointer(subm) && d.GetSubmitter(subm) != nil {
			add(subm)
		}
	}

	// Follow every reference except into individuals and families: those
	// are in the result already or are the boundary.
	var walkErr error
	visit := func(p *string) {
		ref := *p
		if walkErr != nil || !d.IsRecordPointer(ref) || closure[ref] {
			return
		}
		target := d.GetRecord(ref)
		switch {
		case target == nil:
			walkErr = &UnknownXRefError{XRef: ref}
		case !isPersonOrFamily(target):
			add(ref)
		}
	}
	for head := 0; head < len(queue); head++ {
		walkRecord(d.XRefMap[queue[head]], visit)
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return closure, nil
}

// boundedSeeds checks seeds and returns them without duplicates.
func (d *Document) boundedSeeds(seeds []string) ([]string, error) {
	seen := make(map[string]bool, len(seeds))
	people := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		if seen[seed] {
			continue
		}
		r := d.GetRecord(seed)
		if !d.IsRecordPointer(seed) || r == nil {
			return nil, &UnknownXRefError{XRef: seed, IsSeed: true}
		}
		if !r.IsIndividual() {
			return nil, fmt.Errorf("bounded subset: seed %q is a %s record: %w", seed, r.Type, ErrNotIndividual)
		}
		seen[seed] = true
		people = append(people, seed)
	}
	return people, nil
}

// memberFamilies returns the families with a member in people: those a
// person names with FAMC or FAMS, then those naming a person as HUSB, WIFE
// or CHIL. A FAMC or FAMS that resolves to nothing is left for the closure
// walk to report.
func (d *Document) memberFamilies(people []string) []string {
	set := make(map[string]bool, len(people))
	var families []string
	for _, xref := range people {
		set[xref] = true
		indi := d.GetIndividual(xref)
		if indi == nil {
			continue
		}
		for _, link := range slices.Concat(indi.ChildInFamilies, indi.SpouseInFamilies) {
			if d.IsRecordPointer(link.FamilyXRef) && d.GetRecord(link.FamilyXRef).IsFamily() {
				families = append(families, link.FamilyXRef)
			}
		}
	}
	for _, r := range d.Records {
		fam, ok := r.GetFamily()
		if ok && r.XRef != "" && familyHasMember(fam, set) {
			families = append(families, r.XRef)
		}
	}
	return families
}

// familyHasMember reports whether fam names one of people as HUSB, WIFE or
// CHIL.
func familyHasMember(fam *Family, people map[string]bool) bool {
	isPerson := func(xref string) bool { return people[xref] }
	if isPerson(fam.Husband) || isPerson(fam.Wife) {
		return true
	}
	if slices.ContainsFunc(fam.Children, isPerson) {
		return true
	}
	return slices.ContainsFunc(fam.ChildLinks, func(l ChildLink) bool { return isPerson(l.XRef) })
}

// isPersonOrFamily reports whether r is an individual or family record.
func isPersonOrFamily(r *Record) bool {
	return r.IsIndividual() || r.IsFamily()
}

// pruneRecord removes from r, in place, every pointer for which dropped is
// true, and reports what it removed.
func pruneRecord(r *Record, dropped func(string) bool) []DroppedPointer {
	var report []DroppedPointer
	var removed []*Tag
	hasTags := len(r.Tags) > 0
	r.Tags, removed = pruneTags(r.Tags, dropped)
	for _, t := range removed {
		target := t.Value
		if !dropped(target) {
			target = t.XRef
		}
		report = append(report, DroppedPointer{
			XRef: r.XRef, Tag: t.Tag, Target: target,
			Restorable: r.IsFamily() && t.Level == 1 && (t.Tag == "HUSB" || t.Tag == "WIFE" || t.Tag == "CHIL"),
		})
	}

	if tags := entityTags(r.Entity); tags != nil {
		*tags, _ = pruneTags(*tags, dropped)
	}
	walkEntity(r.Entity, func(p *string) {
		if dropped(*p) {
			if !hasTags {
				report = append(report, DroppedPointer{XRef: r.XRef, Target: *p})
			}
			*p = droppedMarker
		}
	})
	compactEntity(r.Entity)
	// Any marker left sits in a single-valued field (Family.Husband,
	// Event.FamilyXRef, a citation's raw line), which is cleared.
	walkEntity(r.Entity, func(p *string) {
		if *p == droppedMarker {
			*p = ""
		}
	})
	return report
}

// pruneTags returns tags without each line whose Value or XRef is dropped,
// and without that line's subordinate lines, plus the removed lines that
// held the pointers. tags itself is not modified.
func pruneTags(tags []*Tag, dropped func(string) bool) (kept, removed []*Tag) {
	kept = tags[:0:0]
	skipBelow := -1
	for _, t := range tags {
		if t == nil {
			kept = append(kept, t)
			continue
		}
		if skipBelow >= 0 && t.Level > skipBelow {
			continue
		}
		skipBelow = -1
		if dropped(t.Value) || dropped(t.XRef) {
			removed = append(removed, t)
			skipBelow = t.Level
			continue
		}
		kept = append(kept, t)
	}
	if len(removed) == 0 {
		return tags, nil
	}
	return kept, removed
}

// entityTags returns the address of the entity's own raw Tags, or nil for an
// unknown or nil entity.
//
//nolint:gocyclo // 9 entity types × per-case nil guard; intrinsic shape
func entityTags(entity interface{}) *[]*Tag {
	switch e := entity.(type) {
	case *Individual:
		if e != nil {
			return &e.Tags
		}
	case *Family:
		if e != nil {
			return &e.Tags
		}
	case *Source:
		if e != nil {
			return &e.Tags
		}
	case *Repository:
		if e != nil {
			return &e.Tags
		}
	case *Note:
		if e != nil {
			return &e.Tags
		}
	case *MediaObject:
		if e != nil {
			return &e.Tags
		}
	case *Submitter:
		if e != nil {
			return &e.Tags
		}
	case *Submission:
		if e != nil {
			return &e.Tags
		}
	case *SharedNote:
		if e != nil {
			return &e.Tags
		}
	}
	return nil
}

// compactEntity removes the list entries whose pointer pruneRecord replaced
// with droppedMarker: a family's children and child links, an individual's
// family links, and associations.
func compactEntity(entity interface{}) {
	isDropped := func(s string) bool { return s == droppedMarker }
	assoDropped := func(a *Association) bool { return a != nil && isDropped(a.IndividualXRef) }
	switch e := entity.(type) {
	case *Individual:
		if e == nil {
			return
		}
		linkDropped := func(l FamilyLink) bool { return isDropped(l.FamilyXRef) }
		e.ChildInFamilies = deleteFunc(e.ChildInFamilies, linkDropped)
		e.SpouseInFamilies = deleteFunc(e.SpouseInFamilies, linkDropped)
		e.Associations = deleteFunc(e.Associations, assoDropped)
		for _, ev := range e.Events {
			if ev != nil {
				ev.Associations = deleteFunc(ev.Associations, assoDropped)
			}
		}
		for _, at := range e.Attributes {
			if at != nil {
				at.Associations = deleteFunc(at.Associations, assoDropped)
			}
		}
	case *Family:
		if e == nil {
			return
		}
		e.Children = deleteFunc(e.Children, isDropped)
		e.ChildLinks = deleteFunc(e.ChildLinks, func(l ChildLink) bool { return isDropped(l.XRef) })
		for _, ev := range e.Events {
			if ev != nil {
				ev.Associations = deleteFunc(ev.Associations, assoDropped)
			}
		}
		for _, at := range e.Attributes {
			if at != nil {
				at.Associations = deleteFunc(at.Associations, assoDropped)
			}
		}
	}
}

// deleteFunc is slices.DeleteFunc, returning nil rather than an empty slice
// when nothing is left, as the decoder leaves a list it found no entries for.
func deleteFunc[S ~[]E, E any](s S, del func(E) bool) S {
	s = slices.DeleteFunc(s, del)
	if len(s) == 0 {
		return nil
	}
	return s
}
