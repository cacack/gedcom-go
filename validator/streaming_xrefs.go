package validator

import (
	"hash/maphash"
	"math"
)

// xrefTracker is the StreamingValidator's cross-reference bookkeeping: which
// XRefs records declare, which they reference, and where each reference to
// an XRef not yet declared appears. It is built to stay small on files of
// millions of records (#778):
//
//   - each distinct XRef is stored once, as bytes in one arena, under a
//     dense uint32 id, instead of as a string per map entry;
//   - a reference to an XRef already declared can never be orphaned, so it
//     is counted and dropped, not kept;
//   - a reference to an XRef not yet declared is kept as three ids, and is
//     discarded once that XRef is declared.
type xrefTracker struct {
	xrefs xrefTable

	// info holds each XRef's state, indexed by id.
	info blocks[xrefInfo]

	// pending lists, in reference order, the references made to XRefs not
	// declared at the time. Entries whose XRef has since been declared are
	// skipped by orphans, and dropped whenever pending reaches compactAt.
	pending   blocks[pendingRef]
	compactAt int

	// sites interns the (context, field) pairs pending references carry,
	// which repeat across records; siteIDs indexes them.
	sites   []refSite
	siteIDs map[refSite]uint32

	// bigLines holds the declaring line of each XRef whose line does not fit
	// in its xrefInfo.
	bigLines map[uint32]int

	seenCount, usedCount int
}

// xrefInfo is what the tracker knows about one XRef, packed into 32 bits
// because the tracker keeps one per XRef: two flags, and above them the line
// of the record that first declared it. A line past maxInfoLine, in a file of
// over a billion lines, is kept in xrefTracker.bigLines instead.
type xrefInfo uint32

const (
	// infoDeclared is set once a record declares the XRef.
	infoDeclared xrefInfo = 1 << iota
	// infoUsed is set once any record references the XRef.
	infoUsed
	infoLineShift = iota

	maxInfoLine = math.MaxUint32 >> infoLineShift
)

func (i xrefInfo) declared() bool { return i&infoDeclared != 0 }
func (i xrefInfo) used() bool     { return i&infoUsed != 0 }
func (i xrefInfo) line() int      { return int(i >> infoLineShift) }

// pendingRef is a reference, by record, to target, where target was not yet
// declared; site locates it within the record.
type pendingRef struct {
	target, record, site uint32
}

// refSite is where in a record a reference appears.
type refSite struct {
	context, field string
}

func newXRefTracker() xrefTracker {
	return xrefTracker{xrefs: newXRefTable(), siteIDs: make(map[refSite]uint32), bigLines: make(map[uint32]int)}
}

// declare registers xref as declared by a record at line. When an earlier
// record already declared it, declare reports that record's line and dup,
// and the first declaration stays registered.
func (t *xrefTracker) declare(xref string, line int) (firstLine int, dup bool) {
	id := t.id(xref)
	info := t.info.at(int(id))
	if info.declared() {
		if big, ok := t.bigLines[id]; ok {
			return big, true
		}
		return info.line(), true
	}
	*info |= infoDeclared
	switch {
	case line > maxInfoLine:
		t.bigLines[id] = line
	case line > 0:
		*info |= xrefInfo(line) << infoLineShift // #nosec G115 -- 0 < line <= maxInfoLine
	}
	t.seenCount++
	return 0, false
}

// reference registers that the record named record references target, at
// field within the structure context names ("FAMC", "NOTE", ...). When keep
// is false the reference is only counted, never reported as orphaned; the
// validator passes false for the null pointer.
func (t *xrefTracker) reference(target, record, context, field string, keep bool) {
	targetID := t.id(target)
	info := t.info.at(int(targetID))
	if !info.used() {
		*info |= infoUsed
		t.usedCount++
	}
	if info.declared() || !keep {
		return
	}
	if t.pending.len() >= t.compactAt {
		t.dropResolved()
		t.compactAt = max(minCompactAt, 2*t.pending.len())
	}
	t.pending.append(pendingRef{
		target: targetID,
		record: t.id(record),
		site:   t.site(context, field),
	})
}

// minCompactAt is the fewest pending references worth compacting.
const minCompactAt = 1 << 12

// orphans calls yield for each kept reference to an XRef never declared, in
// the order the references were made.
func (t *xrefTracker) orphans(yield func(xref string, usage usageLocation)) {
	for i := range t.pending.len() {
		p := *t.pending.at(i)
		if t.info.at(int(p.target)).declared() {
			continue
		}
		s := t.sites[p.site]
		yield(t.xrefs.get(p.target), usageLocation{
			RecordXRef: t.xrefs.get(p.record),
			Context:    s.context,
			Field:      s.field,
		})
	}
}

// id returns the id of xref, registering xref first if it is new.
func (t *xrefTracker) id(xref string) uint32 {
	id := t.xrefs.intern(xref)
	if int(id) == t.info.len() {
		t.info.append(0)
	}
	return id
}

// dropResolved removes, in place, the pending references whose XRef has
// since been declared.
func (t *xrefTracker) dropResolved() {
	kept := 0
	for i := range t.pending.len() {
		p := *t.pending.at(i)
		if !t.info.at(int(p.target)).declared() {
			*t.pending.at(kept) = p
			kept++
		}
	}
	t.pending.truncate(kept)
}

// site returns the id of the (context, field) pair, interning it if new.
func (t *xrefTracker) site(context, field string) uint32 {
	key := refSite{context: context, field: field}
	if id, ok := t.siteIDs[key]; ok {
		return id
	}
	id := uint32(len(t.sites)) // #nosec G115 -- bounded by the number of references, far below 2^32
	t.sites = append(t.sites, key)
	t.siteIDs[key] = id
	return id
}

// xrefTable interns strings as dense ids 0, 1, 2, ... It keeps every
// string's bytes back to back in one arena and indexes them with an
// open-addressed hash table of ids, so a string costs its length plus a few
// words, where a map[string] entry costs a string header, a map slot and a
// separate allocation.
type xrefTable struct {
	seed maphash.Seed

	// arena holds the strings' bytes; string id ends at ends[id] and starts
	// where id-1 ends (at 0 for id 0).
	arena []byte
	ends  blocks[int]

	// slots is the hash index: each slot holds id+1, or 0 when empty. Its
	// length is a power of two, kept above 4/3 the number of ids.
	slots []uint32
}

func newXRefTable() xrefTable {
	return xrefTable{seed: maphash.MakeSeed(), slots: make([]uint32, 64)}
}

// get returns the string interned as id.
func (x *xrefTable) get(id uint32) string {
	return string(x.bytes(id))
}

// bytes returns the bytes of the string interned as id, within the arena.
func (x *xrefTable) bytes(id uint32) []byte {
	start := 0
	if id > 0 {
		start = *x.ends.at(int(id) - 1)
	}
	return x.arena[start:*x.ends.at(int(id))]
}

// intern returns the id of s, adding s if it is new.
func (x *xrefTable) intern(s string) uint32 {
	mask := uint64(len(x.slots) - 1) // #nosec G115 -- len is a positive power of two
	i := maphash.String(x.seed, s) & mask
	for ; x.slots[i] != 0; i = (i + 1) & mask {
		if id := x.slots[i] - 1; string(x.bytes(id)) == s {
			return id
		}
	}
	id := uint32(x.ends.len()) // #nosec G115 -- one id per distinct XRef, far below 2^32
	x.arena = append(x.arena, s...)
	x.ends.append(len(x.arena))
	x.slots[i] = id + 1
	if 4*x.ends.len() > 3*len(x.slots) {
		x.rehash(2 * len(x.slots))
	}
	return id
}

// rehash rebuilds the index with n slots.
func (x *xrefTable) rehash(n int) {
	x.slots = make([]uint32, n)
	mask := uint64(n - 1) // #nosec G115 -- n is a positive power of two
	for id := range x.ends.len() {
		// maphash.Bytes hashes as maphash.String does the same bytes.
		i := maphash.Bytes(x.seed, x.bytes(uint32(id))) & mask // #nosec G115 -- id < len(ends), which fits uint32
		for x.slots[i] != 0 {
			i = (i + 1) & mask
		}
		x.slots[i] = uint32(id) + 1 // #nosec G115 -- as above
	}
}

// blockLen is the number of elements in each of a blocks' blocks.
const blockLen = 1 << 12

// blocks is an append-only list kept in fixed-size blocks. Growing it never
// copies what it holds, so a list of millions of elements does not, while it
// grows, briefly need room for two copies of itself, as a slice does.
type blocks[T any] struct {
	b [][]T
	n int
}

func (l *blocks[T]) len() int { return l.n }

// at returns a pointer to element i, which must be below len.
func (l *blocks[T]) at(i int) *T {
	return &l.b[i/blockLen][i%blockLen]
}

func (l *blocks[T]) append(v T) {
	if l.n == len(l.b)*blockLen {
		l.b = append(l.b, make([]T, blockLen))
	}
	l.b[l.n/blockLen][l.n%blockLen] = v
	l.n++
}

// truncate shortens the list to n elements, releasing the blocks past them.
func (l *blocks[T]) truncate(n int) {
	keep := (n + blockLen - 1) / blockLen
	clear(l.b[keep:])
	l.b = l.b[:keep]
	l.n = n
}
