package merge

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/errmark"
)

// ErrInvalidMerge is wrapped by the error Individuals, Families, Sources and
// Repositories return when the two records named cannot be merged: they are the same record,
// one is missing or of the wrong type, one is defined more than once, or a
// record involved has no raw Tags. A missing record's error also wraps
// gedcom.ErrUnknownXRef. FoldRepeatedEvents wraps it the same way for a
// record named that is missing, of the wrong type or has no raw Tags.
var ErrInvalidMerge = errors.New("invalid record merge")

// RecordMergeOptions configures Individuals, Families, Sources and
// Repositories.
type RecordMergeOptions struct {
	// Context allows cancellation and timeout control. It is checked before
	// the merge starts and before each record copied; once it is done the
	// merge returns a nil document and report and the context's error
	// wrapped. The zero value (nil) means no cancellation.
	Context context.Context

	// FoldEvents folds a substructure of drop into the substructure of keep
	// that records the same event, instead of adding it beside it.
	//
	// It applies to the level-1 substructures the decoder reads into
	// Individual.Events, Individual.Attributes, Family.Events or
	// Family.Attributes for the type of record merged: the events and
	// attributes of the GEDCOM grammar (BIRT, DEAT, RESI, OCCU, MARR, EVEN,
	// FACT and the rest) and the 7.0 NO assertion. Any other substructure is
	// handled as with FoldEvents unset, whatever DATE it has: CHAN and CREA,
	// which date the record and not an event of the person or family, an LDS
	// ordinance, an extension tag such as _MILT, or an event tag on the wrong
	// type of record (MARR on an individual).
	//
	// Two such substructures record the same event when they have the same
	// tag, each has exactly one DATE line and at most one PLAC and one TYPE
	// line directly under it, and:
	//   - the two level-1 values, read with their CONT and CONC lines as
	//     Record.FullValue reads them, are equal, or one is empty and the other is
	//     the "Y" that says a named event occurred (the decoder's
	//     Event.Occurred), which the DATE says too: "1 BIRT Y" and "1 BIRT"
	//     are the same. Values that differ in any other way are not, and nor
	//     are "Y" and an empty value on an EVEN or an attribute, where "Y" is
	//     a descriptor or a value;
	//   - the two DATE values parse with gedcom.ParseDate to the same date
	//     and pass Date.Validate. The same date means every parsed field but
	//     Original is equal: "7 DEC 1977" and "07 Dec 1977" are the same,
	//     "ABT 1977" and "1977" are not, nor are one date in two calendars.
	//     A DATE that does not parse, fails Validate ("32 DEC 1977") or is a
	//     date phrase is the same as none;
	//   - each DATE has at most one TIME and one PHRASE line under it, and
	//     where both have one the values are equal;
	//   - the two PLAC values, read the same way, are equal, or only one of
	//     them has a PLAC;
	//   - the two TYPE values are equal, or neither has a TYPE: the TYPE of
	//     an EVEN, FACT or IDNO says which event or fact it is, so two
	//     events of differing TYPE are different events.
	//
	// Folding keeps keep's lines, its level-1 value included ("1 BIRT" stays
	// "1 BIRT" when drop's is "1 BIRT Y"), and adds each line directly under
	// drop's substructure that keep's does not hold identically (its citations,
	// notes, a place keep lacks), after keep's last line with that tag.
	// drop's DATE and PLAC are matched with keep's instead of added; the
	// lines under them are added under keep's DATE and PLAC the same way,
	// so a TIME or PHRASE only drop's DATE has is added to keep's. A CONT or
	// CONC continuing a matched value is not added, since keep's holds it.
	//
	// drop's substructure folds into the first substructure keep held before
	// the merge that records the same event. One identical to a substructure
	// keep holds is deduplicated, as without FoldEvents. Each fold is listed
	// in RecordMergeReport.Folded. The zero value (false) folds nothing.
	FoldEvents bool
}

// RecordMergeReport describes what Individuals, Families, Sources or
// Repositories did. It is nil on every error path.
type RecordMergeReport struct {
	// Keep is the surviving record's XRef; Drop is the absorbed record's.
	Keep, Drop string
	// Moved lists Drop's substructures now under Keep, in Drop's order.
	Moved []Substructure
	// Deduplicated lists Drop's substructures that were not moved because
	// Keep already held one with the same lines.
	Deduplicated []Substructure
	// Folded lists Drop's substructures folded into one of Keep's that
	// records the same event (see RecordMergeOptions.FoldEvents), in Drop's
	// order. It is empty unless FoldEvents is set.
	Folded []Fold
	// Repointed lists every pointer that named Drop and now names Keep, as
	// gedcom.Document.RemoveRecord reports them (Reference.XRef is "" for
	// the header).
	Repointed []gedcom.Reference
	// Collapsed lists the lines that named Drop and were removed instead of
	// repointed, because once repointed they would repeat a sibling with the
	// same lines in the same record: the second CHIL of a family that listed
	// both merged individuals, say.
	Collapsed []gedcom.Reference
	// Conflicts lists Drop's substructures that were not carried to Keep
	// because the line may occur only once in the record and Keep holds a
	// different one (its own, or the first of Drop's it was given), in
	// Drop's order. Only Sources and Repositories report
	// any.
	Conflicts []Conflict
}

// Substructure is one level-1 structure of a record: its first line and
// every line subordinate to it, in document order. The tags are those of
// the merged document, already repointed; they are not copies.
type Substructure struct {
	Tags []*gedcom.Tag
}

// Fold is one substructure of drop folded into one of keep. Drop is drop's
// substructure as it was, repointed; its lines Into already held, its DATE
// and PLAC lines, and a level-1 "Y" where Into's value is empty, are not in
// the merged document. Into is keep's
// substructure after every fold into it, and its tags are those of the
// merged document.
type Fold struct {
	Drop, Into Substructure
}

// Individuals returns a copy of doc with the individual drop merged into
// the individual keep. Deciding that the two describe one person is the
// caller's job; Individuals does the bookkeeping:
//
//   - Every substructure of drop (names, events, attributes, family links,
//     citations, notes, media links, vendor tags) is added under keep,
//     after keep's last substructure with the same tag, or at the end. One
//     identical, line for line, to a substructure keep already holds is not
//     added, and with RecordMergeOptions.FoldEvents one that records the
//     same event as one of keep's is folded into it. No other policy
//     applies: a second SEX that differs from keep's is added, and the
//     validator reports it.
//   - Every pointer to drop anywhere in the document, in typed fields and
//     raw Tags alike (FAMS, FAMC, HUSB, WIFE, CHIL, ASSO, ALIA, vendor
//     tags), is rewritten to name keep, except a level-1 line that would
//     then repeat a sibling line for line, which is removed (see
//     RecordMergeReport.Collapsed).
//   - drop is removed from Records and XRefMap.
//
// The merged records' raw Tags are authoritative: keep's Entity, and that
// of every record a line was collapsed from, is rebuilt from them as Decode
// builds it (decoder.SyncEntities). Typed edits not yet carried into Tags
// (encoder.SyncTags) are therefore discarded on those records. Merging
// individuals often leaves two families for one couple; Families merges
// them.
//
// doc is never modified. Merging a record into itself, naming a missing
// record or one that is not an individual, a keep or drop defined more than
// once, or a document where keep, drop or a record pointing at drop has no
// raw Tags (a hand-built record) returns an error wrapping ErrInvalidMerge.
// A nil doc returns one wrapping ErrNilDocument.
func Individuals(doc *gedcom.Document, keep, drop string, opts RecordMergeOptions) (*gedcom.Document, *RecordMergeReport, error) {
	return mergeRecords(doc, gedcom.RecordTypeIndividual, keep, drop, opts)
}

// Families is Individuals for two family records: drop's substructures
// (HUSB, WIFE, CHIL, events, citations, notes, media links, vendor tags)
// are added under keep, and a child both families list is listed once.
// Pointers to drop (each spouse's FAMS, each child's FAMC) are repointed to
// keep, and one that would repeat a sibling line, such as the FAMS of a
// spouse already linked to keep, is removed. A differing HUSB or WIFE is
// added like any other substructure, and the validator reports it.
func Families(doc *gedcom.Document, keep, drop string, opts RecordMergeOptions) (*gedcom.Document, *RecordMergeReport, error) {
	return mergeRecords(doc, gedcom.RecordTypeFamily, keep, drop, opts)
}

func mergeRecords(doc *gedcom.Document, typ gedcom.RecordType, keep, drop string, opts RecordMergeOptions) (*gedcom.Document, *RecordMergeReport, error) {
	if doc == nil {
		return nil, nil, errmark.Mark(ErrNilDocument, errors.New("merge: doc is nil"))
	}
	if err := cancelled(opts.Context); err != nil {
		return nil, nil, err
	}
	if err := checkMerge(doc, typ, keep, drop); err != nil {
		return nil, nil, err
	}

	out, err := cloneDoc(opts.Context, doc)
	if err != nil {
		return nil, nil, err
	}
	report := &RecordMergeReport{Keep: keep, Drop: drop}
	keepRec, dropRec := out.GetRecord(keep), out.GetRecord(drop)

	// Lines that would repeat a sibling once repointed are removed before
	// repointing, so only pointers that named drop are ever collapsed.
	synced := []*gedcom.Record{keepRec}
	for _, r := range out.Records {
		if r == nil || r == dropRec || !collapsesRepeats(typ) {
			continue
		}
		if collapsed := collapseRepeats(r, keep, drop); len(collapsed) > 0 {
			report.Collapsed = append(report.Collapsed, collapsed...)
			if r != keepRec {
				synced = append(synced, r)
			}
		}
	}

	refs, err := out.RemoveRecord(drop)
	if err != nil {
		return nil, nil, err // unreachable: checkMerge found drop
	}
	report.Repointed = refs
	mapping := map[string]string{drop: keep}
	gedcom.Apply(out, mapping)
	// drop is detached from out now, so its own pointers to itself are
	// repointed separately, before its substructures move under keep.
	gedcom.Apply(&gedcom.Document{Header: out.Header, Records: []*gedcom.Record{dropRec}}, mapping)

	var fold *folder
	if opts.FoldEvents {
		fold = &folder{header: out.Header, typ: typ}
	}
	keepRec.Tags = mergeSubstructures(keepRec.Tags, dropRec.Tags, fold, onceOnly(out, typ), report)

	decoder.SyncEntities(&gedcom.Document{Header: out.Header, Records: synced})
	return out, report, nil
}

// mergeSubstructures returns keep's lines with each level-1 substructure of
// drop deduplicated, left out as a conflict (when its tag is in once and
// keep holds one, an earlier line of drop's included, so the merged record
// never gains a repeat), folded (when fold is not nil) or moved under them, and
// records which in report.
func mergeSubstructures(keep, drop []*gedcom.Tag, fold *folder, once map[string]bool, report *RecordMergeReport) []*gedcom.Tag {
	roots := buildTree(keep)
	keepRoots := slices.Clone(roots)
	var into []*node // into[i] is the keep substructure of report.Folded[i]
	for _, n := range buildTree(drop) {
		sub := Substructure{Tags: flatten([]*node{n}, nil)}
		if slices.ContainsFunc(roots, func(k *node) bool { return sameTree(k, n, nil) }) {
			report.Deduplicated = append(report.Deduplicated, sub)
			continue
		}
		if once[n.tag.Tag] {
			if i := slices.IndexFunc(roots, func(k *node) bool { return k.tag.Tag == n.tag.Tag }); i >= 0 {
				report.Conflicts = append(report.Conflicts, Conflict{Keep: Substructure{Tags: flatten(roots[i:i+1], nil)}, Drop: sub})
				continue
			}
		}
		if fold != nil {
			if i := fold.match(keepRoots, n); i >= 0 {
				foldEvent(keepRoots[i], n)
				report.Folded = append(report.Folded, Fold{Drop: sub})
				into = append(into, keepRoots[i])
				continue
			}
		}
		roots = insertAfterLastSameTag(roots, n)
		report.Moved = append(report.Moved, sub)
	}
	for i, k := range into {
		report.Folded[i].Into = Substructure{Tags: flatten([]*node{k}, nil)}
	}
	return flatten(roots, nil)
}

// checkMerge reports why keep and drop, both of type typ, cannot be merged
// in doc, or nil when they can.
func checkMerge(doc *gedcom.Document, typ gedcom.RecordType, keep, drop string) error {
	if keep == drop {
		return fmt.Errorf("merge: %q into itself: %w", keep, ErrInvalidMerge)
	}
	defined := map[string]int{}
	for _, r := range doc.Records {
		if r != nil && (r.XRef == keep || r.XRef == drop) {
			defined[r.XRef]++
		}
	}
	for _, xref := range []string{keep, drop} {
		switch defined[xref] {
		case 0:
			return fmt.Errorf("merge: %q: %w: %w", xref, ErrInvalidMerge, gedcom.ErrUnknownXRef)
		case 1:
		default:
			return fmt.Errorf("merge: %q is defined %d times: %w", xref, defined[xref], ErrInvalidMerge)
		}
		r := doc.GetRecord(xref)
		if r == nil || r.Type != typ {
			return fmt.Errorf("merge: %q is not a %s record: %w", xref, typ, ErrInvalidMerge)
		}
		if len(r.Tags) == 0 {
			return fmt.Errorf("merge: %q has no raw Tags: %w", xref, ErrInvalidMerge)
		}
	}
	return checkUntaggedPointers(doc, drop)
}

// checkUntaggedPointers reports a record with no raw Tags that points at
// drop: its pointer would be repointed in its Entity alone, never collapsed.
func checkUntaggedPointers(doc *gedcom.Document, drop string) error {
	for _, r := range doc.Records {
		if r == nil || len(r.Tags) > 0 || r.XRef == drop {
			continue
		}
		points := false
		gedcom.Visit(r, func(ref string) { points = points || ref == drop })
		if points {
			return fmt.Errorf("merge: %q points at %q and has no raw Tags: %w", r.XRef, drop, ErrInvalidMerge)
		}
	}
	return nil
}

// collapseRepeats removes from r each level-1 structure whose line names
// drop and that, once drop is read as keep, has the same lines as a
// sibling: an earlier one still kept, or any one whose line does not name
// drop (and so is never removed), wherever it sits. Of two such structures
// that both name drop, the first stays. It returns where each removed line
// was.
func collapseRepeats(r *gedcom.Record, keep, drop string) []gedcom.Reference {
	if !slices.ContainsFunc(r.Tags, func(t *gedcom.Tag) bool { return t != nil && t.Value == drop }) {
		return nil
	}
	asKeep := func(v string) string {
		if v == drop {
			return keep
		}
		return v
	}
	roots := buildTree(r.Tags)
	var kept []*node
	var removed []gedcom.Reference
	for i, n := range roots {
		repeats := func(k *node) bool { return sameTree(k, n, asKeep) }
		laterKeeper := func(k *node) bool { return k.tag.Value != drop && repeats(k) }
		if n.tag.Value == drop && (slices.ContainsFunc(kept, repeats) || slices.ContainsFunc(roots[i+1:], laterKeeper)) {
			removed = append(removed, gedcom.Reference{XRef: r.XRef, Path: []string{string(r.Type), n.tag.Tag}})
			continue
		}
		kept = append(kept, n)
	}
	if len(removed) > 0 {
		r.Tags = flatten(kept, nil)
	}
	return removed
}

// sameTree reports whether a and b write the same lines once each value is
// read through norm (nil reads values as they are).
func sameTree(a, b *node, norm func(string) string) bool {
	if norm == nil {
		norm = func(v string) string { return v }
	}
	if a.tag.Level != b.tag.Level || a.tag.Tag != b.tag.Tag || a.tag.XRef != b.tag.XRef ||
		norm(a.tag.Value) != norm(b.tag.Value) || len(a.children) != len(b.children) {
		return false
	}
	for i := range a.children {
		if !sameTree(a.children[i], b.children[i], norm) {
			return false
		}
	}
	return true
}

// folder holds the rule RecordMergeOptions.FoldEvents defines, for
// Individuals, Families and FoldRepeatedEvents alike: which substructures it
// applies to, by how the decoder reads them in a record of type typ under
// header, and which of them record the same event.
type folder struct {
	header *gedcom.Header
	typ    gedcom.RecordType
}

// applies reports whether the decoder reads the level-1 substructure n into
// the record's Events or Attributes.
func (f *folder) applies(n *node) bool {
	events, attributes := f.decode(n)
	return len(events)+len(attributes) > 0
}

// decode returns the events and attributes the decoder reads from the
// level-1 substructure n.
func (f *folder) decode(n *node) ([]*gedcom.Event, []*gedcom.Attribute) {
	rec := &gedcom.Record{Type: f.typ, Tags: flatten([]*node{n}, nil)}
	decoder.SyncEntities(&gedcom.Document{Header: f.header, Records: []*gedcom.Record{rec}})
	switch e := rec.Entity.(type) {
	case *gedcom.Individual:
		return e.Events, e.Attributes
	case *gedcom.Family:
		return e.Events, e.Attributes
	}
	return nil, nil
}

// match returns the index of the first of subs that records the same event
// as the level-1 substructure d, by the rule RecordMergeOptions.FoldEvents
// defines, or -1 when none does or the rule does not apply to d.
func (f *folder) match(subs []*node, d *node) int {
	if !f.applies(d) {
		return -1
	}
	return slices.IndexFunc(subs, func(k *node) bool { return f.sameEvent(k, d) })
}

// sameValue reports whether the level-1 values of k and d, CONT and CONC
// lines included, are equal, or one is empty and the other the "Y" the
// decoder reads as Event.Occurred.
func (f *folder) sameValue(k, d *node) bool {
	kv, dv := fullValue(k), fullValue(d)
	if kv == dv {
		return true
	}
	flag := k
	if kv == "" {
		flag = d
	} else if dv != "" {
		return false
	}
	events, _ := f.decode(flag)
	return flag.tag.Value == "Y" && len(events) == 1 && events[0].Occurred
}

// sameEvent reports whether keep's substructure k and drop's substructure d
// record the same event, as RecordMergeOptions.FoldEvents defines it.
func (f *folder) sameEvent(k, d *node) bool {
	if k.tag.Tag != d.tag.Tag {
		return false
	}
	kDate, kOK := onlyChild(k, "DATE")
	dDate, dOK := onlyChild(d, "DATE")
	if !kOK || !dOK || kDate == nil || dDate == nil || !sameDate(kDate.tag.Value, dDate.tag.Value) || !f.sameValue(k, d) {
		return false
	}
	kType, _ := onlyChild(k, "TYPE")
	dType, _ := onlyChild(d, "TYPE")
	return agree(kDate, dDate, "TIME") && agree(kDate, dDate, "PHRASE") &&
		agree(k, d, "PLAC") && agree(k, d, "TYPE") && (kType == nil) == (dType == nil)
}

// agree reports whether k and d each have at most one child with tag and,
// when both have one, the two values, CONT and CONC lines included, are
// equal.
func agree(k, d *node, tag string) bool {
	kc, kOK := onlyChild(k, tag)
	dc, dOK := onlyChild(d, tag)
	return kOK && dOK && (kc == nil || dc == nil || fullValue(kc) == fullValue(dc))
}

// fullValue returns n's value joined with its CONT and CONC lines, as
// Record.FullValue reads it. With none it returns n's value as it is.
func fullValue(n *node) string {
	var b strings.Builder
	folded := false
	for _, c := range n.children {
		if !isContinuation(n, c) {
			continue
		}
		if !folded {
			b.WriteString(n.tag.Value)
			folded = true
		}
		if c.tag.Tag == "CONT" {
			b.WriteByte('\n')
		}
		b.WriteString(c.tag.Value)
	}
	if !folded {
		return n.tag.Value
	}
	return b.String()
}

// isContinuation reports whether c, a child of parent, continues parent's
// value: a CONT or CONC one level below it, as Record.FullValue reads them.
// A CONT or CONC deeper than that, which a strict decode keeps as written,
// is an ordinary line.
func isContinuation(parent, c *node) bool {
	return c.tag.Level == parent.tag.Level+1 && (c.tag.Tag == "CONT" || c.tag.Tag == "CONC")
}

// onlyChild returns n's child with tag, or nil when it has none; ok is
// false when it has more than one.
func onlyChild(n *node, tag string) (child *node, ok bool) {
	for _, c := range n.children {
		if c.tag.Tag != tag {
			continue
		}
		if child != nil {
			return nil, false
		}
		child = c
	}
	return child, true
}

// sameDate reports whether two DATE values parse to the same date: every
// parsed field but Original equal. A value that does not parse, fails
// Date.Validate, or is a date phrase is the same as none.
func sameDate(a, b string) bool {
	da, errA := gedcom.ParseDate(a)
	db, errB := gedcom.ParseDate(b)
	if errA != nil || errB != nil || da.IsPhrase || db.IsPhrase || da.Validate() != nil || db.Validate() != nil {
		return false
	}
	for _, d := range []*gedcom.Date{da, db} {
		for ; d != nil; d = d.EndDate {
			d.Original = ""
		}
	}
	return reflect.DeepEqual(da, db)
}

// foldEvent adds to k each child of d that k does not hold identically.
// d's DATE, PLAC and TYPE are matched with k's instead, and their children
// added under k's the same way; d's PLAC is added whole when k has none.
// A CONT or CONC under a matched line continues a value k already holds,
// so it is not added.
func foldEvent(k, d *node) {
	for _, c := range d.children {
		if isContinuation(d, c) {
			continue
		}
		if c.tag.Tag == "DATE" || c.tag.Tag == "PLAC" || c.tag.Tag == "TYPE" {
			if kc, _ := onlyChild(k, c.tag.Tag); kc != nil {
				for _, cc := range c.children {
					if !isContinuation(c, cc) {
						addMissing(kc, cc)
					}
				}
				continue
			}
		}
		addMissing(k, c)
	}
}

// addMissing adds c under parent, after parent's last child with c's tag,
// unless parent already holds a child with the same lines.
func addMissing(parent, c *node) {
	if !slices.ContainsFunc(parent.children, func(x *node) bool { return sameTree(x, c, nil) }) {
		parent.children = insertAfterLastSameTag(parent.children, c)
	}
}

// insertAfterLastSameTag inserts n after the last of roots with n's tag, so
// a record's names, say, stay together and its first name stays first, or
// at the end when there is none.
func insertAfterLastSameTag(roots []*node, n *node) []*node {
	for i := len(roots) - 1; i >= 0; i-- {
		if roots[i].tag.Tag == n.tag.Tag {
			return slices.Insert(roots, i+1, n)
		}
	}
	return append(roots, n)
}
