package encoder

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// ErrSyncBaseline is returned by SyncTags when the document's raw tags do not
// read back as the same records, so there is no baseline to compare the typed
// fields against. Nothing is changed when it is returned.
var ErrSyncBaseline = errors.New("raw tags do not read back as the same records")

// SyncTags brings the raw Tags of doc's header and records in step with their
// typed fields, so a typed edit on a decoded document reaches the encoded
// output.
//
// [gedcom.Record].Tags is authoritative on encode (see the package
// documentation, "Raw Tags Are Authoritative"), so setting
// indi.Names[0].Full on a decoded record changes nothing the encoder writes.
// SyncTags is the step that makes it do so. For each record with both Tags and
// an Entity, and for a header with Tags, it compares what the typed fields
// encode to now with what the raw Tags decode and encode to, and applies only
// the difference to Tags, one structure at a time:
//
//   - A structure whose typed fields did not change keeps its raw lines
//     exactly, including every tag the typed model does not represent and its
//     original CONT/CONC split. An unmodified document is therefore left as it
//     was, line for line.
//   - A structure whose value changed takes the new value (and CONT/CONC) and
//     keeps its unchanged substructures, so editing a header SOUR keeps its
//     VERS, NAME and CORP.
//   - A structure added in the typed model is inserted near its siblings, and
//     one removed from it is removed.
//   - When raw and typed structures of one tag cannot be paired one to one,
//     that tag's structures are rewritten from the typed fields, losing any
//     untyped tags beneath them.
//
// Afterwards the Tags read back as the typed fields, which is why SyncTags
// can add lines beyond the edit itself. The decoder fills some fields from
// another line, such as PersonalName.Given and Surname from the NAME value
// when there is no GIVN or SURN; editing Full alone leaves those holding the
// old name, and SyncTags then writes them as GIVN and SURN so the record still
// reads back with them. Edit such fields together.
//
// The header's typed fields are those a hand-built header is written from:
// SourceSystem, Date, Submitter, Language, Version and whether Encoding is set.
// The encoder always writes CHAR UTF-8, so Encoding's value is not carried,
// only its presence, and for 7.0 output, which has no CHAR, not even that.
//
// SyncTags makes the typed fields win. It cannot tell a typed edit from an
// edit made to Tags alone, so on a record whose Tags were changed without its
// Entity -- by hand, or by a tool that edits Tags, such as converter.Convert
// -- it writes the stale typed fields back over that change, and drops raw
// structures the Entity does not hold. Edit one side only: to make typed
// edits on such a document, encode and decode it first, so its Entity reads
// as its Tags.
//
// SyncTags replaces Header.Tags, Record.Tags and Record.Value; it never edits
// a Tag in place, nor an Entity. Records with no Tags (written from the Entity
// anyway), records with no Entity or one of the wrong type, and nil records
// are left alone. The encoder's own write-time rules are unchanged: SyncTags
// edits the document, Encode still writes Tags verbatim.
//
// Returns ErrNilDocument if doc is nil, ErrSyncBaseline if the raw tags do not
// read back as the same records, and ErrLineBreak if a typed field holds a
// line break where GEDCOM cannot write one. On error doc is unchanged.
func SyncTags(doc *gedcom.Document) error {
	if doc == nil {
		return ErrNilDocument
	}

	opts := DefaultOptions().forOutput(doc.Header)
	sharedNotes := sharedNotePointers(doc, opts)

	// The work is done on copies and applied at the end, so an error leaves
	// doc as it was.
	header := doc.Header
	syncHead := header != nil && len(header.Tags) > 0
	if syncHead {
		copied := *header
		header = &copied
	}
	var originals, records []*gedcom.Record
	for _, record := range doc.Records {
		if syncable(record) {
			copied := *record
			originals = append(originals, record)
			records = append(records, &copied)
		}
	}

	// Applying a difference can leave a field the decoder fills from another
	// line reading back differently (see above), so what was written is read
	// back and synced again, record by record, until nothing changes.
	pending := records
	for round := 0; round < maxSyncRounds && (syncHead || len(pending) > 0); round++ {
		headChanged, changed, err := syncRound(header, syncHead, pending, opts, sharedNotes)
		if err != nil {
			return err
		}
		syncHead, pending = headChanged, changed
	}

	if header != doc.Header {
		doc.Header.Tags = header.Tags
	}
	for i, record := range originals {
		record.Value, record.Tags = records[i].Value, records[i].Tags
	}
	return nil
}

// maxSyncRounds bounds how often SyncTags reads back what it wrote. One round
// applies the edit and a second confirms it; a field filled from another line
// takes one more.
const maxSyncRounds = 4

// syncRound applies one round of SyncTags to records, and to header when
// syncHead is set, and reports which of them changed.
func syncRound(header *gedcom.Header, syncHead bool, records []*gedcom.Record, opts *EncodeOptions,
	sharedNotes *sharedNoteSet,
) (bool, []*gedcom.Record, error) {
	baseline, err := decodeBaseline(header, records)
	if err != nil {
		return false, nil, err
	}

	headChanged := false
	if syncHead {
		tags, err := syncHeader(header, baseline.Header, opts)
		if err != nil {
			return false, nil, err
		}
		headChanged = !sameLines(header.Tags, tags)
		header.Tags = tags
	}

	var changed []*gedcom.Record
	for i, record := range records {
		value, tags := syncRecord(record, baseline.Records[i], opts, sharedNotes)
		if value != record.Value || !sameLines(record.Tags, tags) {
			changed = append(changed, record)
		}
		record.Value, record.Tags = value, tags
	}
	return headChanged, changed, nil
}

// sameLines reports whether two tag lists hold the same tags. SyncTags keeps
// every line it does not change as the original *Tag, so comparing pointers
// is exact.
func sameLines(a, b []*gedcom.Tag) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// syncable reports whether SyncTags brings record's Tags in step with its
// Entity: it has Tags, and an Entity of the type its record type decodes to.
// A record with no Tags is written from its Entity already. One whose Entity
// is missing, a typed nil (ADR 0007) or of another type has no typed fields
// to compare, and deriving it would read as every structure deleted.
//
//nolint:gocyclo // One accessor per record type that has an entity.
func syncable(record *gedcom.Record) bool {
	if record == nil || len(record.Tags) == 0 {
		return false
	}
	var ok bool
	switch record.Type {
	case gedcom.RecordTypeIndividual:
		_, ok = record.GetIndividual()
	case gedcom.RecordTypeFamily:
		_, ok = record.GetFamily()
	case gedcom.RecordTypeSource:
		_, ok = record.GetSource()
	case gedcom.RecordTypeSubmitter:
		_, ok = record.GetSubmitter()
	case gedcom.RecordTypeSubmission:
		_, ok = record.GetSubmission()
	case gedcom.RecordTypeRepository:
		_, ok = record.GetRepository()
	case gedcom.RecordTypeNote:
		_, ok = record.GetNote()
	case gedcom.RecordTypeMedia:
		_, ok = record.GetMediaObject()
	case gedcom.RecordTypeSharedNote:
		_, ok = record.GetSharedNote()
	}
	return ok
}

// decodeBaseline reads the raw tags of header and records back through the
// decoder, giving the typed fields those tags decode to: the baseline the
// current typed fields are compared against. The decoded document's records
// are records, in order.
func decodeBaseline(header *gedcom.Header, records []*gedcom.Record) (*gedcom.Document, error) {
	var buf bytes.Buffer
	if err := Encode(&buf, &gedcom.Document{Header: header, Records: records}); err != nil {
		return nil, err
	}
	baseline, err := decoder.Decode(&buf)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSyncBaseline, err)
	}

	if len(baseline.Records) != len(records) {
		return nil, fmt.Errorf("%w: %d records read back as %d", ErrSyncBaseline, len(records), len(baseline.Records))
	}
	for i, record := range records {
		got := baseline.Records[i]
		if got.Type != record.Type || got.XRef != record.XRef {
			return nil, fmt.Errorf("%w: record %d %s %s read back as %s %s",
				ErrSyncBaseline, i, record.XRef, record.Type, got.XRef, got.Type)
		}
	}
	return baseline, nil
}

// syncHeader returns header's raw tags with the change from baseline's typed
// fields to header's applied.
func syncHeader(header, baseline *gedcom.Header, opts *EncodeOptions) ([]*gedcom.Tag, error) {
	if baseline == nil {
		baseline = &gedcom.Header{}
	}
	// GEDCOM 7.0 has no CHAR, and its decoder leaves Encoding empty while the
	// converter sets it to UTF-8, so for 7.0 output Encoding is not a field
	// SyncTags carries: a raw CHAR line, if any, is kept as it is.
	if opts.is70() {
		copied, base := *header, *baseline
		copied.Encoding, base.Encoding = "", ""
		header, baseline = &copied, &base
	}
	was, err := headerFieldTags(baseline, opts)
	if err != nil {
		return nil, err
	}
	now, err := headerFieldTags(header, opts)
	if err != nil {
		return nil, err
	}
	return mergeChildren(tagTree(header.Tags), tagTree(was), tagTree(now), false, 0), nil
}

// syncRecord returns record's level-0 value and Tags with the change from
// baseline's typed fields to record's applied.
func syncRecord(record, baseline *gedcom.Record, opts *EncodeOptions, sharedNotes *sharedNoteSet) (string, []*gedcom.Tag) {
	wasValue, was := entityRecordLines(baseline, opts, sharedNotes)
	nowValue, now := entityRecordLines(record, opts, sharedNotes)

	wasRoot := &tagNode{tag: &gedcom.Tag{Value: wasValue}, kids: tagTree(was)}
	nowRoot := &tagNode{tag: &gedcom.Tag{Value: nowValue}, kids: tagTree(now)}
	changed := foldedValue(wasRoot) != foldedValue(nowRoot)

	value := record.Value
	if changed {
		value = nowValue
	}
	return value, mergeChildren(tagTree(record.Tags), wasRoot.kids, nowRoot.kids, changed, 0)
}

// tagNode is one GEDCOM structure: its opening line and the structures beneath
// it, CONT and CONC included, in document order.
type tagNode struct {
	tag  *gedcom.Tag
	kids []*tagNode
}

// tagTree builds the structure tree of a flat tag list. A tag belongs to the
// nearest preceding tag at a lower level, so a level jump in malformed input
// still attaches somewhere and every line is kept. Nil tags write nothing and
// are dropped.
func tagTree(tags []*gedcom.Tag) []*tagNode {
	var roots []*tagNode
	var stack []*tagNode
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		node := &tagNode{tag: tag}
		for len(stack) > 0 && stack[len(stack)-1].tag.Level >= tag.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1]
			parent.kids = append(parent.kids, node)
		}
		stack = append(stack, node)
	}
	return roots
}

// isContinuation reports whether a tag continues its parent's value.
func isContinuation(tag string) bool {
	return tag == "CONT" || tag == "CONC"
}

// foldedValue returns a structure's whole value: its own line's value joined
// with its CONT and CONC lines, the way the decoder reads it.
func foldedValue(n *tagNode) string {
	value := n.tag.Value
	for _, kid := range n.kids {
		switch kid.tag.Tag {
		case "CONT":
			value += "\n" + kid.tag.Value
		case "CONC":
			value += kid.tag.Value
		}
	}
	return value
}

// sameTree reports whether two structures write the same lines, ignoring
// their absolute levels and source line numbers.
func sameTree(a, b *tagNode) bool {
	if a.tag.Tag != b.tag.Tag || a.tag.Value != b.tag.Value || a.tag.XRef != b.tag.XRef || len(a.kids) != len(b.kids) {
		return false
	}
	for i := range a.kids {
		if !sameTree(a.kids[i], b.kids[i]) {
			return false
		}
	}
	return true
}

func sameTrees(a, b []*tagNode) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameTree(a[i], b[i]) {
			return false
		}
	}
	return true
}

// continuationKey groups CONT and CONC together: they are one value, not two
// kinds of structure.
const continuationKey = "\x00CONT"

func groupKey(n *tagNode) string {
	if isContinuation(n.tag.Tag) {
		return continuationKey
	}
	return n.tag.Tag
}

// groupIndexes returns, for each group key, the indexes of nodes in that group
// in order, and the keys in order of first appearance.
func groupIndexes(nodes []*tagNode) (groups map[string][]int, order []string) {
	groups = make(map[string][]int)
	for i, n := range nodes {
		key := groupKey(n)
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], i)
	}
	return groups, order
}

// slot is what replaces one raw child in the merged output: the lines
// inserted before it, its own lines (nil once removed), and those after it.
type slot struct {
	before, self, after []*gedcom.Tag
}

// mergeChildren returns the lines of the raw children raw, a parent at level
// parentLevel, with the change from was to now applied, where was and now are
// the same parent's children as derived from the baseline and the current
// typed fields. valueChanged says whether the parent's own value changed,
// which decides its CONT and CONC lines.
//
// Children are compared in groups of one tag. A group that derives the same
// before and after keeps its raw lines; otherwise its structures are paired
// raw to baseline and aligned baseline to current (see mergeGroup).
func mergeChildren(raw, was, now []*tagNode, valueChanged bool, parentLevel int) []*gedcom.Tag {
	m := &childMerge{raw: raw, was: was, now: now, parentLevel: parentLevel, slots: make([]slot, len(raw))}
	for i, n := range raw {
		m.slots[i].self = flatten(n, 0)
	}
	m.rawGroups, _ = groupIndexes(raw)
	wasGroups, _ := groupIndexes(was)
	nowGroups, nowOrder := groupIndexes(now)
	m.nowOrder = nowOrder

	for _, key := range unionKeys(raw, now, was) {
		r, b, c := m.rawGroups[key], wasGroups[key], nowGroups[key]
		switch {
		case key == continuationKey:
			if valueChanged {
				m.remove(r)
				// A continuation belongs directly after the line it continues.
				m.lead = append(pick(now, c, parentLevel), m.lead...)
			}
		case sameTrees(pickNodes(was, b), pickNodes(now, c)):
			// Unchanged: the raw lines stay as they are.
		default:
			if pairs := pairRaw(raw, r, was, b); pairs != nil {
				m.mergeGroup(key, r, b, c, pairs)
			} else {
				m.remove(r)
				m.place(key, r, pick(now, c, parentLevel))
			}
		}
	}

	out := m.lead
	for _, s := range m.slots {
		out = append(out, s.before...)
		out = append(out, s.self...)
		out = append(out, s.after...)
	}
	return out
}

// childMerge is the state of one mergeChildren call.
type childMerge struct {
	raw, was, now []*tagNode
	parentLevel   int

	slots     []slot           // what each raw child is replaced by
	lead      []*gedcom.Tag    // lines placed before every raw child
	rawGroups map[string][]int // raw child indexes by group key
	nowOrder  []string         // group keys in current order
}

// remove drops the raw children at indexes r.
func (m *childMerge) remove(r []int) {
	for _, i := range r {
		m.slots[i].self = nil
	}
}

// place inserts lines for group key, whose raw children are r: before the
// first of them, or for a group with none, after the last raw child of the
// nearest group before it in the current order, or else first.
func (m *childMerge) place(key string, r []int, lines []*gedcom.Tag) {
	if len(lines) == 0 {
		return
	}
	if len(r) > 0 {
		m.slots[r[0]].before = append(m.slots[r[0]].before, lines...)
		return
	}
	if at := anchorAfter(key, m.nowOrder, m.rawGroups); at >= 0 {
		m.slots[at].after = append(m.slots[at].after, lines...)
		return
	}
	m.lead = append(m.lead, lines...)
}

// mergeGroup applies the change from the baseline structures b to the current
// structures c of group key to their raw structures r. pairs[k] is the raw
// structure (an index into raw) baseline structure b[k] was read from.
//
// b and c are aligned by their longest common run of identical structures.
// Those keep their raw lines. Between two aligned structures, unaligned
// baseline and current structures of equal count are paired in order and
// merged one level down; otherwise the baseline ones are removed and the
// current ones inserted in their place.
func (m *childMerge) mergeGroup(key string, r, b, c, pairs []int) {
	matches := alignTrees(pickNodes(m.was, b), pickNodes(m.now, c))
	bi, ci := 0, 0
	lastRaw := -1 // raw index of the last structure kept or merged
	for _, match := range append(matches, [2]int{len(b), len(c)}) {
		gapB, gapC := match[0]-bi, match[1]-ci
		if gapB == gapC {
			for k := 0; k < gapB; k++ {
				ri := pairs[bi+k]
				m.slots[ri].self = mergeNode(m.raw[ri], m.was[b[bi+k]], m.now[c[ci+k]])
				lastRaw = ri
			}
		} else {
			removed := pairs[bi:match[0]]
			m.remove(removed)
			added := pick(m.now, c[ci:match[1]], m.parentLevel)
			switch {
			case len(removed) > 0:
				m.slots[removed[0]].before = append(m.slots[removed[0]].before, added...)
			case lastRaw >= 0:
				m.slots[lastRaw].after = append(m.slots[lastRaw].after, added...)
			default:
				m.place(key, r, added)
			}
		}
		if match[0] < len(b) {
			lastRaw = pairs[match[0]]
		}
		bi, ci = match[0]+1, match[1]+1
	}
}

// mergeNode returns the lines of raw structure rn with the change from its
// baseline wn to its current nn applied: nn's value if that changed, and its
// children merged one level down.
func mergeNode(rn, wn, nn *tagNode) []*gedcom.Tag {
	changed := foldedValue(wn) != foldedValue(nn)
	head := rn.tag
	if changed {
		copied := *rn.tag
		copied.Value = nn.tag.Value
		head = &copied
	}
	return append([]*gedcom.Tag{head}, mergeChildren(rn.kids, wn.kids, nn.kids, changed, rn.tag.Level)...)
}

// alignTrees returns the index pairs of a longest common subsequence of
// identical structures in a and b, in order.
func alignTrees(a, b []*tagNode) [][2]int {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case sameTree(a[i], b[j]):
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var matches [][2]int
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case sameTree(a[i], b[j]):
			matches = append(matches, [2]int{i, j})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			i++
		default:
			j++
		}
	}
	return matches
}

// pairRaw returns, for each baseline structure b[k], the raw structure it was
// read from, or nil when the two cannot be paired one to one. A structure is
// paired with the first unpaired raw structure of the same value, so a writer
// that orders a tag's structures differently from the source (record NOTE
// pointers before inline notes) still pairs them correctly; any left over are
// paired in order.
func pairRaw(raw []*tagNode, r []int, was []*tagNode, b []int) []int {
	if len(r) != len(b) || len(r) == 0 {
		return nil
	}
	pairs := make([]int, len(b))
	used := make([]bool, len(r))
	for k, bi := range b {
		pairs[k] = -1
		for j, ri := range r {
			if !used[j] && raw[ri].tag.XRef == was[bi].tag.XRef && foldedValue(raw[ri]) == foldedValue(was[bi]) {
				pairs[k], used[j] = ri, true
				break
			}
		}
	}
	next := 0
	for k := range pairs {
		if pairs[k] >= 0 {
			continue
		}
		for used[next] {
			next++
		}
		pairs[k], used[next] = r[next], true
	}
	return pairs
}

// anchorAfter returns the raw child after which a group with no raw
// structures goes: the last raw child of the nearest group before it in the
// current order that has raw structures, or -1 for the front.
func anchorAfter(key string, nowOrder []string, rawGroups map[string][]int) int {
	pos := -1
	for i, k := range nowOrder {
		if k == key {
			pos = i
			break
		}
	}
	for i := pos - 1; i >= 0; i-- {
		if r := rawGroups[nowOrder[i]]; len(r) > 0 {
			return r[len(r)-1]
		}
	}
	return -1
}

// unionKeys returns every group key present in raw, now or was, in that order
// of first appearance.
func unionKeys(raw, now, was []*tagNode) []string {
	seen := make(map[string]bool)
	var keys []string
	for _, nodes := range [][]*tagNode{raw, now, was} {
		for _, n := range nodes {
			if key := groupKey(n); !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func pickNodes(nodes []*tagNode, idx []int) []*tagNode {
	out := make([]*tagNode, len(idx))
	for k, i := range idx {
		out[k] = nodes[i]
	}
	return out
}

// pick returns the lines of the derived structures nodes[idx], re-levelled to
// sit under a parent at parentLevel.
func pick(nodes []*tagNode, idx []int, parentLevel int) []*gedcom.Tag {
	var out []*gedcom.Tag
	for _, i := range idx {
		n := nodes[i]
		out = append(out, flatten(n, parentLevel+1-n.tag.Level)...)
	}
	return out
}

// flatten returns a structure's lines in document order, each moved by shift
// levels. An unmoved line is the original tag; a moved one is a copy, since a
// derived tag may be one the entity's own Tags still holds.
func flatten(n *tagNode, shift int) []*gedcom.Tag {
	tag := n.tag
	if shift != 0 {
		copied := *tag
		copied.Level += shift
		tag = &copied
	}
	out := []*gedcom.Tag{tag}
	for _, kid := range n.kids {
		out = append(out, flatten(kid, shift)...)
	}
	return out
}
