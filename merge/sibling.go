package merge

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// FoldSubstructure folds drop, a substructure of r, into its sibling keep:
// each line directly under drop, with the lines beneath it, is moved under
// keep unless keep already holds one with the same lines, and drop is then
// removed. A moved line goes after keep's last line with its tag, or after
// keep's last line. So folding a record's second NAME into its first moves
// the NICK and the citations the first lacks, and an identical SOUR
// citation is kept once. Deciding that the two describe one thing is the
// caller's job; drop's own line, its value included, is removed, and so
// are the CONT and CONC lines that continue that value.
//
// It is the within-record counterpart of what Individuals does with a
// substructure, applied at any level and to any tag: unlike
// RecordMergeOptions.FoldEvents and FoldRepeatedEvents, no line is matched
// by value, so a TYPE that differs from keep's is moved beside it.
//
// r is edited in place, unlike the rest of this package. No tag is
// changed, only the order and membership of r.Tags: every line keeps its
// LineNumber, so under a layout-preserving encode the moved lines keep
// their own layout and no line but drop's changes. r.Entity is not
// updated; rebuild it with decoder.SyncEntities when it is read.
//
// FoldSubstructure reports whether it folded: keep and drop are distinct
// tags in r.Tags (compared by pointer) at one level with the same parent,
// the nearest earlier line at a shallower level. When they are not,
// nothing changes.
func FoldSubstructure(r *gedcom.Record, keep, drop *gedcom.Tag) bool {
	if r == nil || keep == nil || drop == nil || keep == drop || keep.Level != drop.Level {
		return false
	}
	roots := buildTree(r.Tags)
	keepParent, k := locate(nil, roots, keep)
	dropParent, d := locate(nil, roots, drop)
	if k == nil || d == nil || keepParent != dropParent {
		return false
	}
	for _, c := range d.children {
		if !isContinuation(c) {
			addMissing(k, c)
		}
	}
	if dropParent == nil {
		roots = slices.DeleteFunc(roots, func(n *node) bool { return n == d })
	} else {
		dropParent.children = slices.DeleteFunc(dropParent.children, func(n *node) bool { return n == d })
	}
	r.Tags = flatten(roots, nil)
	return true
}

// locate returns found, the node that holds t among nodes and their
// descendants, and holder, its parent node (parent when found is one of
// nodes). Both are nil when none holds t.
func locate(parent *node, nodes []*node, t *gedcom.Tag) (holder, found *node) {
	for _, n := range nodes {
		if n.tag == t {
			return parent, n
		}
		if h, f := locate(n, n.children, t); f != nil {
			return h, f
		}
	}
	return nil, nil
}
