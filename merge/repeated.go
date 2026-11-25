package merge

import (
	"errors"
	"fmt"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/errmark"
)

// EventGroup is the level-1 substructures of one record that record the same
// event, by the rule RecordMergeOptions.FoldEvents defines.
type EventGroup struct {
	// XRef is the record's XRef.
	XRef string
	// Members are the substructures in document order, at least two.
	// FoldRepeatedEvents folds the others into the first. Each member after
	// the first records the same event as the first once the members before
	// it are folded in, as FoldEvents compares drop's substructure with
	// keep's after earlier folds. The tags are those of the document given,
	// not copies.
	Members []Substructure
}

// FoldReport describes what FoldRepeatedEvents did.
type FoldReport struct {
	// Folded lists one fold per substructure removed, in the order of
	// RepeatedEvents' groups and of their members. Fold.Drop is the member
	// removed and Fold.Into the group's first member after every fold into
	// it, both with the tags of the folded document.
	Folded []RecordFold
}

// RecordFold is one substructure of a record folded into another of the
// same record.
type RecordFold struct {
	// XRef is the record's XRef.
	XRef string
	Fold
}

// RepeatedEvents returns the groups of level-1 substructures that record
// the same event, by the rule RecordMergeOptions.FoldEvents defines, within
// each individual and family record of doc, or within those of them xrefs
// names when it names any. Groups are in document order of record and of
// first member. Each substructure is compared with the substructures before
// it that are not themselves in a group as a later member, and joins the
// group of the first that records the same event.
//
// An empty xrefs means every record, not none: spreading an empty selection,
// RepeatedEvents(doc, selected...), searches the whole document. Check
// len(selected) first when an empty selection should search nothing.
//
// A record with no raw Tags (a hand-built one), or one xrefs names that is
// missing or neither an individual nor a family, has no groups; so does a
// nil doc. doc is not modified.
func RepeatedEvents(doc *gedcom.Document, xrefs ...string) []EventGroup {
	if doc == nil {
		return nil
	}
	var groups []EventGroup
	for _, r := range foldTargets(doc, xrefs) {
		_, repeats := foldRecord(doc.Header, r)
		for _, rep := range repeats {
			groups = append(groups, EventGroup{XRef: r.XRef, Members: rep.members})
		}
	}
	return groups
}

// FoldRepeatedEvents returns a copy of doc with each group RepeatedEvents
// finds folded into its first member, as RecordMergeOptions.FoldEvents folds
// drop's substructure into keep's: the first member keeps its lines, and
// gains each line of the others it does not hold identically (citations,
// notes, a place it lacks). Each other member is removed from the record.
//
// An empty xrefs means every record, not none, as for RepeatedEvents:
// spreading an empty selection, FoldRepeatedEvents(doc, selected...), folds
// the whole document. Check len(selected) first when an empty selection
// should fold nothing.
//
// Only the records folded change: their raw Tags are rebuilt, and their
// Entity with them, as Decode builds it (decoder.SyncEntities), so typed
// edits to them not yet carried into Tags (encoder.SyncTags) are discarded.
// Every other record keeps its Tags line for line. Folding the result
// again folds nothing.
//
// doc is never modified. A nil doc returns an error wrapping
// ErrNilDocument. An XRef in xrefs that no record has returns one wrapping
// ErrInvalidMerge and gedcom.ErrUnknownXRef, and one that names a record
// that is not an individual or a family, or has no raw Tags, returns one
// wrapping ErrInvalidMerge.
func FoldRepeatedEvents(doc *gedcom.Document, xrefs ...string) (*gedcom.Document, *FoldReport, error) {
	if doc == nil {
		return nil, nil, errmark.Mark(ErrNilDocument, errors.New("merge: doc is nil"))
	}
	if err := checkFoldTargets(doc, xrefs); err != nil {
		return nil, nil, err
	}
	out := doc.Clone()
	report := &FoldReport{}
	var changed []*gedcom.Record
	for _, r := range foldTargets(out, xrefs) {
		kept, repeats := foldRecord(out.Header, r)
		if len(repeats) == 0 {
			continue
		}
		r.Tags = flatten(kept, nil)
		changed = append(changed, r)
		for _, rep := range repeats {
			into := Substructure{Tags: flatten([]*node{rep.into}, nil)}
			for _, m := range rep.members[1:] {
				report.Folded = append(report.Folded, RecordFold{XRef: r.XRef, Fold: Fold{Drop: m, Into: into}})
			}
		}
	}
	decoder.SyncEntities(&gedcom.Document{Header: out.Header, Records: changed})
	return out, report, nil
}

// checkFoldTargets reports why xrefs cannot be folded in doc, or nil when
// they can. Of several XRefs that cannot, it reports the first in xrefs.
func checkFoldTargets(doc *gedcom.Document, xrefs []string) error {
	if len(xrefs) == 0 {
		return nil
	}
	errs := make(map[string]error, len(xrefs))
	for _, xref := range xrefs {
		errs[xref] = fmt.Errorf("merge: %q: %w: %w", xref, ErrInvalidMerge, gedcom.ErrUnknownXRef)
	}
	found := make(map[string]bool, len(xrefs))
	for _, r := range doc.Records {
		if r == nil {
			continue
		}
		if _, named := errs[r.XRef]; !named {
			continue
		}
		if !found[r.XRef] {
			found[r.XRef] = true
			errs[r.XRef] = nil
		}
		if errs[r.XRef] != nil {
			continue
		}
		if r.Type != gedcom.RecordTypeIndividual && r.Type != gedcom.RecordTypeFamily {
			errs[r.XRef] = fmt.Errorf("merge: %q is not an individual or family record: %w", r.XRef, ErrInvalidMerge)
		} else if len(r.Tags) == 0 {
			errs[r.XRef] = fmt.Errorf("merge: %q has no raw Tags: %w", r.XRef, ErrInvalidMerge)
		}
	}
	for _, xref := range xrefs {
		if errs[xref] != nil {
			return errs[xref]
		}
	}
	return nil
}

// foldTargets returns doc's individual and family records with raw Tags,
// only those xrefs names when it names any, in document order.
func foldTargets(doc *gedcom.Document, xrefs []string) []*gedcom.Record {
	var named map[string]bool
	if len(xrefs) > 0 {
		named = make(map[string]bool, len(xrefs))
		for _, xref := range xrefs {
			named[xref] = true
		}
	}
	var out []*gedcom.Record
	for _, r := range doc.Records {
		if r == nil || len(r.Tags) == 0 || (r.Type != gedcom.RecordTypeIndividual && r.Type != gedcom.RecordTypeFamily) {
			continue
		}
		if named != nil && !named[r.XRef] {
			continue
		}
		out = append(out, r)
	}
	return out
}

// repeat is one group of substructures recording the same event.
type repeat struct {
	into    *node          // the first member, every later one folded in
	members []Substructure // each member's lines as they were
}

// foldRecord folds each level-1 substructure of r into the first earlier
// one left that records the same event (folder.match, foldEvent) and
// returns the substructures left and the groups folded, in order of first
// member. Only the tree is changed: r's Tags and the tags in them are not.
func foldRecord(header *gedcom.Header, r *gedcom.Record) ([]*node, []*repeat) {
	f := &folder{header: header, typ: r.Type}
	var kept []*node
	byInto := map[*node]*repeat{}
	for _, n := range buildTree(r.Tags) {
		i := f.match(kept, n)
		if i < 0 {
			kept = append(kept, n)
			continue
		}
		rep := byInto[kept[i]]
		if rep == nil {
			rep = &repeat{into: kept[i], members: []Substructure{{Tags: flatten(kept[i:i+1], nil)}}}
			byInto[kept[i]] = rep
		}
		rep.members = append(rep.members, Substructure{Tags: flatten([]*node{n}, nil)})
		foldEvent(kept[i], n)
	}
	ordered := make([]*repeat, 0, len(byInto))
	for _, k := range kept {
		if rep := byInto[k]; rep != nil {
			ordered = append(ordered, rep)
		}
	}
	return kept, ordered
}
