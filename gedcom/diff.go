package gedcom

import (
	"slices"
	"sort"
	"strconv"
	"strings"
)

// DiffOptions configures Diff. The zero value, and a nil *DiffOptions,
// compare every record tag and leave the header out.
type DiffOptions struct {
	// IgnoreTags names tags left out of the comparison, matched exactly
	// (case-sensitive). A substructure with one of these tags is skipped
	// together with everything under it, at any level, and a record whose
	// type is one of them is not compared at all. Typical candidates are
	// CHAN and vendor tags that change on every export of an unchanged tree.
	IgnoreTags []string

	// IncludeHeader compares the header too, reporting a difference in
	// DocumentDiff.Header. The header changes on every export (its DATE at
	// least), so it is left out by default.
	IncludeHeader bool
}

// DocumentDiff is the record-level difference between two documents, as
// returned by Diff. Records are matched by XRef and type.
type DocumentDiff struct {
	// Added holds the records only the new document has, in its order.
	Added []*Record `json:",omitempty"`

	// Removed holds the records only the old document has, in its order.
	Removed []*Record `json:",omitempty"`

	// Modified holds the records both documents have whose content differs,
	// in the old document's order.
	Modified []*RecordDiff `json:",omitempty"`

	// Header is the difference between the two headers. It is nil when the
	// headers are equal or when DiffOptions.IncludeHeader is not set.
	Header *RecordDiff `json:",omitempty"`
}

// IsEmpty reports whether the diff found no difference. It is true for a nil
// receiver.
func (d *DocumentDiff) IsEmpty() bool {
	return d == nil ||
		(len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Modified) == 0 && d.Header == nil)
}

// RecordDiff is the difference within one record both documents have, or
// within the header.
type RecordDiff struct {
	// XRef is the record's cross-reference identifier; empty for the header.
	XRef string `json:",omitempty"`

	// Type is the record type; "HEAD" for the header.
	Type RecordType `json:",omitempty"`

	// Old and New are the record as it is in each document; both nil for the
	// header.
	Old *Record `json:"-"`
	New *Record `json:"-"`

	// Changes lists the changed lines, grouped by the tag of the structure
	// they belong to in order of that tag's first appearance.
	Changes []LineChange `json:",omitempty"`
}

// LineChange is one hunk of a RecordDiff: lines the old record has that the
// new one does not, and the reverse, under a common parent structure.
//
// When both Removed and Added are set, a structure's own line changed (its
// value or pointer, including any CONT/CONC lines that continue the value).
// Its substructures are compared separately and reported in their own
// hunks. When only one is set, whole structures were removed or added, and
// the lines include every substructure.
type LineChange struct {
	// Path is the tags from the record down to the parent of the changed
	// lines, not including the record's own tag: empty for a change at level
	// 0 or 1, ["BIRT"] for a changed DATE under a level-1 BIRT. Each hunk
	// has its own Path slice. Path names tags, not structures, so it is not
	// unique: hunks under the first and second of two BIRTs share ["BIRT"],
	// and a level-0 hunk shares the empty Path with level-1 hunks. Use the
	// lines' Tag.Level and Tag.LineNumber to tell them apart.
	Path []string `json:",omitempty"`

	// Removed holds the old document's lines, in document order. They are
	// the document's own tags (so Tag.LineNumber points into the old file),
	// except that a level-0 line is built from the Record's XRef, Type, Value
	// and LineNumber, since Record.Tags does not hold it.
	Removed []*Tag `json:",omitempty"`

	// Added holds the new document's lines, on the same terms as Removed.
	Added []*Tag `json:",omitempty"`
}

// Diff compares two documents record by record and reports the records added,
// removed and modified between them, with the changed lines of each modified
// record. Neither document is modified; the records and tags in the result are
// the documents' own, not copies, except the level-0 lines described at
// LineChange.Removed.
//
// Records are matched by XRef and type: renumbering a record, or changing its
// type, reports it as removed and added. Two records sharing an XRef, which is
// malformed, are matched by occurrence (the first with the first). Finding the
// same person under different XRefs is duplicate detection, not diffing.
//
// Records are compared by their Tags, which govern encoding. A record whose
// typed fields were edited without encoder.SyncTags compares as its Tags say,
// and a record with no Tags (built in code, or edited by clearing Tags)
// compares only its level-0 line, so differences in its typed fields are not
// seen. SyncTags leaves such a record alone; encode and re-decode the document
// first to compare what it would write.
//
// Only differences in meaning are reported. These are not changes:
//
//   - Substructures appearing in a different order relative to siblings with
//     a different tag (BIRT before or after NAME). GEDCOM gives that order no
//     meaning.
//   - A value split differently across CONT/CONC lines, when the text they
//     join to is the same.
//   - Line numbers.
//
// Everything else is a change, including reordering among siblings with the
// same tag (the first NAME is the preferred one, CHIL is in birth order) and
// case or whitespace in a value.
//
// A nil document compares as an empty one, so Diff(nil, doc) reports every
// record of doc as added.
func Diff(oldDoc, newDoc *Document, opts *DiffOptions) *DocumentDiff {
	if opts == nil {
		opts = &DiffOptions{}
	}
	ignore := make(map[string]bool, len(opts.IgnoreTags))
	for _, tag := range opts.IgnoreTags {
		ignore[tag] = true
	}

	result := &DocumentDiff{}
	oldRecords, oldByKey := diffIndex(oldDoc, ignore)
	newRecords, newByKey := diffIndex(newDoc, ignore)

	for _, kr := range oldRecords {
		newRecord, ok := newByKey[kr.key]
		if !ok {
			result.Removed = append(result.Removed, kr.record)
			continue
		}
		if changes := diffTrees(recordTree(kr.record, ignore), recordTree(newRecord, ignore)); changes != nil {
			result.Modified = append(result.Modified, &RecordDiff{
				XRef:    kr.record.XRef,
				Type:    kr.record.Type,
				Old:     kr.record,
				New:     newRecord,
				Changes: changes,
			})
		}
	}
	for _, kr := range newRecords {
		if _, ok := oldByKey[kr.key]; !ok {
			result.Added = append(result.Added, kr.record)
		}
	}

	if opts.IncludeHeader {
		if changes := diffTrees(headerTree(oldDoc, ignore), headerTree(newDoc, ignore)); changes != nil {
			result.Header = &RecordDiff{Type: "HEAD", Changes: changes}
		}
	}
	return result
}

// diffKey identifies a record across two documents: its type, XRef and which
// occurrence of that pair it is.
type diffKey struct {
	recordType RecordType
	xref       string
	occurrence int
}

type keyedRecord struct {
	key    diffKey
	record *Record
}

// diffIndex lists a document's compared records in order, with each one's key.
func diffIndex(doc *Document, ignore map[string]bool) (list []keyedRecord, byKey map[diffKey]*Record) {
	if doc == nil {
		return nil, nil
	}
	list = make([]keyedRecord, 0, len(doc.Records))
	byKey = make(map[diffKey]*Record, len(doc.Records))
	for _, record := range doc.Records {
		if record == nil || ignore[string(record.Type)] {
			continue
		}
		key := diffKey{recordType: record.Type, xref: record.XRef}
		for byKey[key] != nil {
			key.occurrence++
		}
		byKey[key] = record
		list = append(list, keyedRecord{key: key, record: record})
	}
	return list, byKey
}

// diffNode is one structure of a record, with its CONT/CONC lines folded into
// its value.
type diffNode struct {
	tag, xref, value string
	lines            []*Tag // own line, then its CONT/CONC lines
	children         []*diffNode
	key              string // canonical form; see canonical
}

func recordTree(record *Record, ignore map[string]bool) *diffNode {
	root := &Tag{Level: 0, Tag: string(record.Type), XRef: record.XRef, Value: record.Value, LineNumber: record.LineNumber}
	return buildDiffTree(root, record.Tags, ignore)
}

func headerTree(doc *Document, ignore map[string]bool) *diffNode {
	var tags []*Tag
	if doc != nil && doc.Header != nil {
		tags = doc.Header.Tags
	}
	return buildDiffTree(&Tag{Level: 0, Tag: "HEAD"}, tags, ignore)
}

// buildDiffTree nests tags under root by level. A tag deeper than one below
// its parent (a malformed level jump) still nests under the nearest shallower
// tag.
func buildDiffTree(root *Tag, tags []*Tag, ignore map[string]bool) *diffNode {
	type frame struct {
		node  *diffNode
		level int
	}
	rootNode := &diffNode{tag: root.Tag, xref: root.XRef, value: root.Value, lines: []*Tag{root}}
	stack := []frame{{rootNode, root.Level}}
	skipBelow := -1 // level of an ignored tag whose subtree is being skipped

	for _, t := range tags {
		if t == nil {
			continue
		}
		if skipBelow >= 0 {
			if t.Level > skipBelow {
				continue
			}
			skipBelow = -1
		}
		for len(stack) > 1 && stack[len(stack)-1].level >= t.Level {
			stack = stack[:len(stack)-1]
		}
		if ignore[t.Tag] {
			skipBelow = t.Level
			continue
		}
		parent := stack[len(stack)-1].node
		switch t.Tag {
		case "CONT":
			parent.value += "\n" + t.Value
			parent.lines = append(parent.lines, t)
			continue
		case "CONC":
			parent.value += t.Value
			parent.lines = append(parent.lines, t)
			continue
		}
		node := &diffNode{tag: t.Tag, xref: t.XRef, value: t.Value, lines: []*Tag{t}}
		parent.children = append(parent.children, node)
		stack = append(stack, frame{node, t.Level})
	}
	return rootNode
}

// canonical returns a string equal for two nodes exactly when they have the
// same meaning: same tag, pointer and value, and the same children with the
// order kept only among children sharing a tag.
func (n *diffNode) canonical() string {
	if n.key != "" {
		return n.key
	}
	children := make([]*diffNode, len(n.children))
	copy(children, n.children)
	sort.SliceStable(children, func(i, j int) bool { return children[i].tag < children[j].tag })

	var b strings.Builder
	for _, field := range []string{n.tag, n.xref, n.value} {
		b.WriteString(strconv.Itoa(len(field)))
		b.WriteByte(':')
		b.WriteString(field)
	}
	b.WriteByte('[')
	for _, child := range children {
		key := child.canonical()
		b.WriteString(strconv.Itoa(len(key)))
		b.WriteByte(':')
		b.WriteString(key)
	}
	b.WriteByte(']')
	n.key = b.String()
	return n.key
}

// allLines returns a node's lines and those of every substructure, in
// document order.
func (n *diffNode) allLines() []*Tag {
	lines := append([]*Tag(nil), n.lines...)
	for _, child := range n.children {
		lines = append(lines, child.allLines()...)
	}
	return lines
}

// diffTrees returns the changes between two record (or header) trees, or nil
// when they are equal.
func diffTrees(oldRoot, newRoot *diffNode) []LineChange {
	if oldRoot.canonical() == newRoot.canonical() {
		return nil
	}
	var changes []LineChange
	if oldRoot.xref != newRoot.xref || oldRoot.value != newRoot.value {
		changes = append(changes, LineChange{Removed: oldRoot.lines, Added: newRoot.lines})
	}
	diffChildren(oldRoot, newRoot, nil, &changes)
	return changes
}

// diffChildren compares two nodes' children tag by tag, in order of each
// tag's first appearance (old first, then tags only the new node has).
func diffChildren(oldNode, newNode *diffNode, path []string, changes *[]LineChange) {
	var tags []string
	seen := make(map[string]bool)
	for _, children := range [][]*diffNode{oldNode.children, newNode.children} {
		for _, child := range children {
			if !seen[child.tag] {
				seen[child.tag] = true
				tags = append(tags, child.tag)
			}
		}
	}
	for _, tag := range tags {
		diffGroup(withTag(oldNode.children, tag), withTag(newNode.children, tag), path, changes)
	}
}

func withTag(nodes []*diffNode, tag string) []*diffNode {
	var out []*diffNode
	for _, n := range nodes {
		if n.tag == tag {
			out = append(out, n)
		}
	}
	return out
}

// maxLCSCells bounds the table diffGroup builds to align siblings sharing a
// tag. Past it, the unequal middle of the two runs is compared position by
// position instead.
const maxLCSCells = 1 << 20

// diffGroup compares two ordered runs of siblings sharing a tag. Equal
// siblings are aligned by longest common subsequence; between aligned ones,
// the unmatched old and new siblings are paired by position and compared
// line by line, and any left over are whole removals or additions.
func diffGroup(a, b []*diffNode, path []string, changes *[]LineChange) {
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix].canonical() == b[prefix].canonical() {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix &&
		a[len(a)-1-suffix].canonical() == b[len(b)-1-suffix].canonical() {
		suffix++
	}
	a, b = a[prefix:len(a)-suffix], b[prefix:len(b)-suffix]

	i, j := 0, 0
	for _, m := range lcsMatches(a, b) {
		diffGap(a[i:m[0]], b[j:m[1]], path, changes)
		i, j = m[0]+1, m[1]+1
	}
	diffGap(a[i:], b[j:], path, changes)
}

// lcsMatches returns the index pairs of a longest common subsequence of a and
// b under canonical equality, or none when the table would exceed maxLCSCells.
func lcsMatches(a, b []*diffNode) [][2]int {
	if len(a) == 0 || len(b) == 0 || len(a)*len(b) > maxLCSCells {
		return nil
	}
	// lengths[i][j] is the LCS length of a[i:] and b[j:].
	lengths := make([][]int, len(a)+1)
	for i := range lengths {
		lengths[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i].canonical() == b[j].canonical() {
				lengths[i][j] = lengths[i+1][j+1] + 1
			} else {
				lengths[i][j] = max(lengths[i+1][j], lengths[i][j+1])
			}
		}
	}
	var matches [][2]int
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch {
		case a[i].canonical() == b[j].canonical():
			matches = append(matches, [2]int{i, j})
			i++
			j++
		case lengths[i+1][j] >= lengths[i][j+1]:
			i++
		default:
			j++
		}
	}
	return matches
}

// diffGap reports unmatched siblings: pairs by position, compared line by
// line, then the rest as one hunk of whole structures.
func diffGap(a, b []*diffNode, path []string, changes *[]LineChange) {
	paired := min(len(a), len(b))
	for k := 0; k < paired; k++ {
		diffPair(a[k], b[k], path, changes)
	}
	var removed, added []*Tag
	for _, n := range a[paired:] {
		removed = append(removed, n.allLines()...)
	}
	for _, n := range b[paired:] {
		added = append(added, n.allLines()...)
	}
	if removed != nil || added != nil {
		*changes = append(*changes, LineChange{Path: slices.Clone(path), Removed: removed, Added: added})
	}
}

// diffPair compares two structures with the same tag: their own lines, then
// their substructures.
func diffPair(oldNode, newNode *diffNode, path []string, changes *[]LineChange) {
	if oldNode.xref != newNode.xref || oldNode.value != newNode.value {
		*changes = append(*changes, LineChange{Path: slices.Clone(path), Removed: oldNode.lines, Added: newNode.lines})
	}
	childPath := make([]string, len(path), len(path)+1)
	copy(childPath, path)
	diffChildren(oldNode, newNode, append(childPath, oldNode.tag), changes)
}
