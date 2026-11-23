package merge

import (
	"context"
	"errors"
	"reflect"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// ErrRecordConflict is returned by Combine with CollisionStrategy ==
// ZipSharedXRefs when doc1 and doc2 hold a record with the same XRef whose
// difference is not confined to pointer lines, and
// CombineOptions.KeepFirstOnConflict is not set. Use errors.As with
// *CombineError to recover the conflicting XRefs and reasons.
var ErrRecordConflict = errors.New("record conflict")

// RecordConflict describes one XRef that doc1 and doc2 both define with
// differing content that ZipSharedXRefs cannot union.
type RecordConflict struct {
	// XRef is the record's cross-reference identifier.
	XRef string
	// Reason says what differs.
	Reason ConflictReason
}

// ConflictReason says what differs in a RecordConflict.
type ConflictReason string

const (
	// ConflictType: the records have different types.
	ConflictType ConflictReason = "type"
	// ConflictValue: the records' level-0 values differ.
	ConflictValue ConflictReason = "value"
	// ConflictLines: the records' raw lines differ beyond pointer lines
	// naming an individual or family.
	ConflictLines ConflictReason = "lines"
	// ConflictUntagged: a record has no raw Tags, and the two records'
	// typed Entities differ. Such records are compared in full, never
	// unioned.
	ConflictUntagged ConflictReason = "untagged"
)

// zipCombine implements Combine for ZipSharedXRefs. Header compatibility has
// already been checked.
func zipCombine(doc1, doc2 *gedcom.Document, opts CombineOptions) (*gedcom.Document, *CombineReport, error) {
	ctx := opts.Context
	out, err := cloneDoc(ctx, doc1)
	if err != nil {
		return nil, nil, err
	}
	in2, err := cloneDoc(ctx, doc2)
	if err != nil {
		return nil, nil, err
	}
	z := zipper{doc1: doc1, doc2: doc2}
	unioned, conflicts, err := z.zipInto(ctx, out, in2.Records)
	if err != nil {
		return nil, nil, err
	}
	if len(conflicts) > 0 && !opts.KeepFirstOnConflict {
		xrefs := make([]string, len(conflicts))
		for i, c := range conflicts {
			xrefs[i] = c.XRef
		}
		return nil, nil, &CombineError{Kind: "record-conflict", Colliding: xrefs, Conflicts: conflicts}
	}

	out.ReindexXRefs()
	mergedHeader, headerConflicts := z.zipHeaders(out.Header, in2.Header)
	out.Header = mergedHeader
	headerConflicts = mergeDocMetadata(out, doc1, in2, headerConflicts)
	// A unioned record's Tags are new, so its typed Entity is rebuilt from
	// them as Decode would build it.
	decoder.SyncEntities(&gedcom.Document{Header: out.Header, Records: unioned})

	return out, &CombineReport{
		RemappedXRefs:   map[string]string{},
		HeaderConflicts: headerConflicts,
		RecordConflicts: conflicts,
	}, nil
}

// zipper unions two records with the same XRef. doc1 and doc2 are the
// documents they come from, used to tell which pointers name individuals or
// families.
type zipper struct {
	doc1, doc2 *gedcom.Document
}

// zipHeaders unions two headers cut from one master as it unions a record:
// their typed fields must be equal and their raw lines may differ only in
// pointer lines naming an individual or family, which a bounded subset
// drops from its header as from a record. Other headers are merged as the
// other strategies merge them. Merging two copies of one header that way
// would report each of its structures as a conflict and repeat its
// extension tags. h1 and h2 are clones the result may share lines with.
func (z zipper) zipHeaders(h1, h2 *gedcom.Header) (*gedcom.Header, []HeaderConflict) {
	if h1 != nil && h2 != nil {
		a, b := h1.Clone(), h2.Clone()
		a.Tags, b.Tags = nil, nil
		if reflect.DeepEqual(a, b) {
			if nodes, ok := z.zipNodes(buildTree(h1.Tags), buildTree(h2.Tags)); ok {
				a.Tags = flatten(nodes, nil)
				return a, nil
			}
		}
	}
	return mergeHeaders(h1, h2)
}

// zipInto adds records to out, in place: one whose XRef out lacks is
// appended, and one whose XRef out holds is zipped into out's record. It
// returns the records it replaced with a union, and the conflicts.
func (z zipper) zipInto(ctx context.Context, out *gedcom.Document, records []*gedcom.Record) (unioned []*gedcom.Record, conflicts []RecordConflict, err error) {
	byXRef := make(map[string]int, len(out.Records))
	index := func(i int) {
		if r := out.Records[i]; r != nil && r.XRef != "" {
			if _, seen := byXRef[r.XRef]; !seen {
				byXRef[r.XRef] = i
			}
		}
	}
	for i := range out.Records {
		index(i)
	}
	for _, r2 := range records {
		if err := cancelled(ctx); err != nil {
			return nil, nil, err
		}
		i, shared := -1, false
		if r2 != nil && r2.XRef != "" {
			i, shared = byXRef[r2.XRef]
		}
		if !shared {
			out.Records = append(out.Records, r2)
			index(len(out.Records) - 1)
			continue
		}
		merged, reason := z.zipRecord(out.Records[i], r2)
		switch {
		case reason != "":
			conflicts = append(conflicts, RecordConflict{XRef: r2.XRef, Reason: reason})
		case merged != nil:
			out.Records[i] = merged
			unioned = append(unioned, merged)
		}
	}
	return unioned, conflicts, nil
}

// zipRecord returns the union of r1 and r2. It returns a nil record when r1
// already holds everything r2 does, and a reason when they conflict. The
// union has no Entity: the caller rebuilds it from the Tags.
func (z zipper) zipRecord(r1, r2 *gedcom.Record) (merged *gedcom.Record, conflict ConflictReason) {
	switch {
	case r1.Type != r2.Type:
		return nil, ConflictType
	case r1.Value != r2.Value:
		return nil, ConflictValue
	case len(r1.Tags) == 0 || len(r2.Tags) == 0:
		if len(r1.Tags) == 0 && len(r2.Tags) == 0 && reflect.DeepEqual(r1.Entity, r2.Entity) {
			return nil, ""
		}
		return nil, ConflictUntagged
	}
	nodes, ok := z.zipNodes(buildTree(r1.Tags), buildTree(r2.Tags))
	if !ok {
		return nil, ConflictLines
	}
	tags := flatten(nodes, nil)
	if sameLines(tags, r1.Tags) {
		return nil, ""
	}
	return &gedcom.Record{XRef: r1.XRef, Type: r1.Type, Value: r1.Value, Tags: tags, LineNumber: r1.LineNumber}, ""
}

// node is a raw line with its subordinate lines.
type node struct {
	tag      *gedcom.Tag
	children []*node
}

// buildTree arranges a record's flat Tags into a forest by Level. A nil tag,
// which writes nothing, is skipped.
func buildTree(tags []*gedcom.Tag) []*node {
	var roots []*node
	var stack []*node
	for _, t := range tags {
		if t == nil {
			continue
		}
		n := &node{tag: t}
		for len(stack) > 0 && stack[len(stack)-1].tag.Level >= t.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, n)
		} else {
			parent := stack[len(stack)-1]
			parent.children = append(parent.children, n)
		}
		stack = append(stack, n)
	}
	return roots
}

// flatten appends the lines of nodes, depth first, to dst.
func flatten(nodes []*node, dst []*gedcom.Tag) []*gedcom.Tag {
	for _, n := range nodes {
		dst = append(dst, n.tag)
		dst = flatten(n.children, dst)
	}
	return dst
}

// zipNodes unions two sibling lists. Lines that are not boundary pointers
// (anchors) must match one for one, in order. Boundary pointer lines between
// two anchors are unioned: a line both lists hold is kept once, and a line
// only b holds is merged into a's run in source line order.
func (z zipper) zipNodes(a, b []*node) ([]*node, bool) {
	aAnchors, aRuns := z.split(a)
	bAnchors, bRuns := z.split(b)
	if len(aAnchors) != len(bAnchors) {
		return nil, false
	}
	var out []*node
	for i := range aRuns {
		run, ok := z.zipRun(aRuns[i], bRuns[i])
		if !ok {
			return nil, false
		}
		out = append(out, run...)
		if i == len(aAnchors) {
			break
		}
		anchor, ok := z.zipNode(aAnchors[i], bAnchors[i])
		if !ok {
			return nil, false
		}
		out = append(out, anchor)
	}
	return out, true
}

// zipNode unions two lines that must be the same line.
func (z zipper) zipNode(a, b *node) (*node, bool) {
	if !sameLine(a.tag, b.tag) {
		return nil, false
	}
	children, ok := z.zipNodes(a.children, b.children)
	if !ok {
		return nil, false
	}
	return &node{tag: a.tag, children: children}, true
}

// zipRun unions two runs of boundary pointer lines.
func (z zipper) zipRun(a, b []*node) ([]*node, bool) {
	run := append([]*node(nil), a...)
	used := make([]bool, len(a))
	var extra []*node
	for _, nb := range b {
		matched := false
		for i, na := range a {
			if used[i] || !sameLine(na.tag, nb.tag) {
				continue
			}
			merged, ok := z.zipNode(na, nb)
			if !ok {
				return nil, false
			}
			run[i], used[i], matched = merged, true, true
			break
		}
		if !matched {
			extra = append(extra, nb)
		}
	}
	return mergeByLine(run, extra), true
}

// mergeByLine merges two runs into one, keeping the order within each and
// interleaving them by source line number; on a tie a's line comes first.
// Subsets cut in memory from one decoded document keep that document's line
// numbers, so their runs interleave back into its order.
func mergeByLine(a, b []*node) []*node {
	if len(b) == 0 {
		return a
	}
	out := make([]*node, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i].tag.LineNumber <= b[j].tag.LineNumber {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	return append(out, b[j:]...)
}

// split separates nodes into anchors and the runs of boundary pointer lines
// around them: runs[i] precedes anchors[i], and the last run follows the
// last anchor, so len(runs) == len(anchors)+1.
func (z zipper) split(nodes []*node) (anchors []*node, runs [][]*node) {
	runs = [][]*node{nil}
	for _, n := range nodes {
		if z.isBoundaryPointer(n.tag) {
			runs[len(runs)-1] = append(runs[len(runs)-1], n)
			continue
		}
		anchors = append(anchors, n)
		runs = append(runs, nil)
	}
	return anchors, runs
}

// isBoundaryPointer reports whether t is a line BoundedSubset may have
// dropped: one whose Value or XRef names an individual or family in either
// document. It must cover every line gedcom's pruneRecord drops (see the
// dropped predicate in gedcom.Document.BoundedSubset); change the two
// together.
func (z zipper) isBoundaryPointer(t *gedcom.Tag) bool {
	return z.namesPersonOrFamily(t.Value) || z.namesPersonOrFamily(t.XRef)
}

func (z zipper) namesPersonOrFamily(ref string) bool {
	for _, d := range []*gedcom.Document{z.doc1, z.doc2} {
		if !d.IsRecordPointer(ref) {
			continue
		}
		if r := d.GetRecord(ref); r.IsIndividual() || r.IsFamily() {
			return true
		}
	}
	return false
}

// sameLine reports whether two tags write the same line. LineNumber is
// where the line was read from, not part of it.
func sameLine(a, b *gedcom.Tag) bool {
	return a.Level == b.Level && a.Tag == b.Tag && a.Value == b.Value && a.XRef == b.XRef
}

// sameLines reports whether a and b write the same lines, skipping nil tags.
func sameLines(a, b []*gedcom.Tag) bool {
	a, b = nonNil(a), nonNil(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameLine(a[i], b[i]) {
			return false
		}
	}
	return true
}

func nonNil(tags []*gedcom.Tag) []*gedcom.Tag {
	out := make([]*gedcom.Tag, 0, len(tags))
	for _, t := range tags {
		if t != nil {
			out = append(out, t)
		}
	}
	return out
}
