package merge

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/errmark"
)

// ErrInvalidMerge is wrapped by the error Individuals and Families return
// when the two records named cannot be merged: they are the same record,
// one is missing or of the wrong type, one is defined more than once, or a
// record involved has no raw Tags. A missing record's error also wraps
// gedcom.ErrUnknownXRef.
var ErrInvalidMerge = errors.New("invalid record merge")

// RecordMergeOptions configures Individuals and Families.
type RecordMergeOptions struct {
	// Context allows cancellation and timeout control. It is checked before
	// the merge starts and before each record copied; once it is done the
	// merge returns a nil document and report and the context's error
	// wrapped. The zero value (nil) means no cancellation.
	Context context.Context
}

// RecordMergeReport describes what Individuals or Families did. It is nil
// on every error path.
type RecordMergeReport struct {
	// Keep is the surviving record's XRef; Drop is the absorbed record's.
	Keep, Drop string
	// Moved lists Drop's substructures now under Keep, in Drop's order.
	Moved []Substructure
	// Deduplicated lists Drop's substructures that were not moved because
	// Keep already held one with the same lines.
	Deduplicated []Substructure
	// Repointed lists every pointer that named Drop and now names Keep, as
	// gedcom.Document.RemoveRecord reports them (Reference.XRef is "" for
	// the header).
	Repointed []gedcom.Reference
	// Collapsed lists the lines that named Drop and were removed instead of
	// repointed, because once repointed they would repeat a sibling with the
	// same lines in the same record: the second CHIL of a family that listed
	// both merged individuals, say.
	Collapsed []gedcom.Reference
}

// Substructure is one level-1 structure of a record: its first line and
// every line subordinate to it, in document order. The tags are those of
// the merged document, already repointed; they are not copies.
type Substructure struct {
	Tags []*gedcom.Tag
}

// Individuals returns a copy of doc with the individual drop merged into
// the individual keep. Deciding that the two describe one person is the
// caller's job; Individuals does the bookkeeping:
//
//   - Every substructure of drop (names, events, attributes, family links,
//     citations, notes, media links, vendor tags) is added under keep,
//     after keep's last substructure with the same tag, or at the end. One
//     identical, line for line, to a substructure keep already holds is not
//     added. No other policy applies: a second SEX that differs from
//     keep's is added, and the validator reports it.
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
		if r == nil || r == dropRec {
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

	roots := buildTree(keepRec.Tags)
	for _, n := range buildTree(dropRec.Tags) {
		sub := Substructure{Tags: flatten([]*node{n}, nil)}
		if slices.ContainsFunc(roots, func(k *node) bool { return sameTree(k, n, nil) }) {
			report.Deduplicated = append(report.Deduplicated, sub)
			continue
		}
		roots = insertAfterLastSameTag(roots, n)
		report.Moved = append(report.Moved, sub)
	}
	keepRec.Tags = flatten(roots, nil)

	decoder.SyncEntities(&gedcom.Document{Header: out.Header, Records: synced})
	return out, report, nil
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
