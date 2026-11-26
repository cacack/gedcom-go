package merge

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// NameGroup is the level-1 NAME substructures of one individual record
// that RepeatedNames finds to be the same name.
type NameGroup struct {
	// XRef is the record's XRef.
	XRef string
	// Members are the NAME substructures in document order, at least two.
	// FoldRepeatedNames folds the others into the first. Each member after
	// the first is the same name as the first once the members before it
	// are folded in. The tags are those of the document given, not copies.
	Members []Substructure
}

// namePieceTags are the lines under a NAME that sameName holds to agree:
// its TYPE and its name pieces.
var namePieceTags = []string{"TYPE", "NPFX", "GIVN", "NICK", "SPFX", "SURN", "NSFX"}

// RepeatedNames returns the groups of level-1 NAME substructures that are
// the same name within each individual record of doc, or within those of
// them xrefs names when it names any. Two NAMEs are the same name when:
//
//   - their values are, by gedcom.SamePersonalName: the same words in the
//     same order with case, spacing, punctuation and the surname slashes
//     ignored;
//   - both have a TYPE, written identically line for line, or neither has;
//   - each name piece (NPFX, GIVN, NICK, SPFX, SURN, NSFX) occurs at most
//     once under each, and one that both have is written identically line
//     for line. A piece only one has does not keep them apart.
//
// Other lines under a NAME (citations, notes, TRAN, FONE, ROMN) are not
// compared (spec-interpretation policy SI-024).
//
// Groups are in document order of record and of first member. Each NAME is
// compared with the NAMEs before it that are not themselves in a group as a
// later member, and joins the group of the first that is the same name, as
// that one stands with the members before it folded in.
//
// An empty xrefs means every record, not none, as for RepeatedEvents. A
// record with no raw Tags, or one xrefs names that is missing or not an
// individual, has no groups; so does a nil doc. doc is not modified.
func RepeatedNames(doc *gedcom.Document, xrefs ...string) []NameGroup {
	if doc == nil {
		return nil
	}
	var groups []NameGroup
	for _, r := range foldTargets(doc, nameScope, xrefs) {
		_, repeats := foldNames(nil, r)
		for _, rep := range repeats {
			groups = append(groups, NameGroup{XRef: r.XRef, Members: rep.members})
		}
	}
	return groups
}

// FoldRepeatedNames returns a copy of doc with each group RepeatedNames
// finds folded into its first member, as FoldSubstructure folds drop into
// keep: the first member keeps its lines, value included, and gains each
// line of the others it does not hold identically (a citation, a note, a
// name piece it lacks), so a citation both hold line for line is kept
// once. Each other member is removed from the record.
//
// An empty xrefs means every record, not none, as for FoldRepeatedEvents.
// Only the records folded change: their raw Tags are rebuilt, and their
// Entity with them (decoder.SyncEntities), so typed edits to them not yet
// carried into Tags are discarded. Every other record keeps its Tags line
// for line. Folding the result again folds nothing.
//
// doc is never modified. A nil doc returns an error wrapping
// ErrNilDocument. An XRef in xrefs that no record has returns one wrapping
// ErrInvalidMerge and gedcom.ErrUnknownXRef, and one that names a record
// that is not an individual, or has no raw Tags, returns one wrapping
// ErrInvalidMerge.
func FoldRepeatedNames(doc *gedcom.Document, xrefs ...string) (*gedcom.Document, *FoldReport, error) {
	return foldRepeated(doc, nameScope, xrefs, foldNames)
}

// foldNames folds each level-1 NAME of r into the first earlier one left
// that is the same name (sameName), as FoldSubstructure folds
// (foldChildren), as foldRecord does. The header is not read.
func foldNames(_ *gedcom.Header, r *gedcom.Record) ([]*node, []*repeat) {
	match := func(kept []*node, n *node) int {
		if n.tag.Tag != "NAME" {
			return -1
		}
		return slices.IndexFunc(kept, func(k *node) bool { return sameName(k, n) })
	}
	return foldRecord(r, match, foldChildren)
}

// sameName reports whether the level-1 substructures k and d are NAMEs
// that are the same name, as RepeatedNames defines it.
func sameName(k, d *node) bool {
	if k.tag.Tag != "NAME" || d.tag.Tag != "NAME" || !gedcom.SamePersonalName(fullValue(k), fullValue(d)) {
		return false
	}
	for _, tag := range namePieceTags {
		kc, kOK := onlyChild(k, tag)
		dc, dOK := onlyChild(d, tag)
		if !kOK || !dOK {
			return false
		}
		if kc != nil && dc != nil && !sameTree(kc, dc, nil) {
			return false
		}
		if tag == "TYPE" && (kc == nil) != (dc == nil) {
			return false
		}
	}
	return true
}
