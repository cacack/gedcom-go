package gedcom

import "slices"

// MoveSubstructure moves t, with every line beneath it, to directly before
// its sibling before, or after its last sibling when before is nil. A
// sibling is a tag at t's level with the same parent: the nearest earlier
// tag at a shallower level, or none for a level-1 tag. So
// r.MoveSubstructure(names[1], names[0]) makes a record's second NAME its
// first, carrying its parts, citations and notes with it.
//
// No tag is changed, only the order of r.Tags: every tag keeps its Level,
// Value and LineNumber, so under a layout-preserving encode the moved lines
// keep their own layout and no other line changes.
//
// MoveSubstructure reports whether t was in r.Tags (compared by pointer),
// before was nil or a sibling of t, and t is not a CONT or CONC continuing
// its parent's value, which moving would reorder (SetFullValue sets that);
// when not, nothing changes. Moving t before itself changes nothing and
// reports true. It replaces r.Tags with a new slice, so a view from Subtree
// taken before the call no longer reflects the record. r.Entity is not
// updated; rebuild it with decoder.SyncEntities when it is read.
func (r *Record) MoveSubstructure(t, before *Tag) bool {
	if r == nil || t == nil {
		return false
	}
	start := slices.Index(r.Tags, t)
	if start < 0 {
		return false
	}
	parent := parentIndex(r.Tags, start)
	if continuesParent(r.Tags, start, parent) {
		return false
	}
	if before == t {
		return true
	}
	if before != nil {
		b := slices.Index(r.Tags, before)
		if b < 0 || before.Level != t.Level || parentIndex(r.Tags, b) != parent {
			return false
		}
	}
	block := subtree(r.Tags[start:], t)
	rest := slices.Concat(r.Tags[:start], r.Tags[start+len(block):])

	var at int
	if before != nil {
		at = slices.Index(rest, before)
	} else {
		var p *Tag
		if parent >= 0 {
			p = r.Tags[parent]
		}
		_, at, _ = childSpan(rest, p)
	}
	r.Tags = slices.Concat(rest[:at], block, rest[at:])
	return true
}

// SetChild sets parent's first direct subordinate with tag to value, or
// adds one when parent has none: after parent's last line, or at the end of
// r.Tags when parent is nil, which names the record itself and so its
// level-1 lines. It returns the child set, or nil when parent is not in
// r.Tags (compared by pointer), tag is empty, or tag is CONT or CONC, which
// continue parent's own value (SetFullValue sets that).
//
// value is set as SetFullValue sets it, split for GEDCOM version v, which is
// doc.Header.Version, at the encoder's default width, and in wire form, with
// "@" escaped by the caller (see SetFullValue): the child's old CONT and CONC
// lines are dropped and the new ones follow it, and its other subordinates
// are kept. A child whose FullValue is already value is left as it is. Any
// other line, a second child with tag included, is not changed and keeps its
// LineNumber, so under a layout-preserving encode only the child set and its
// continuation lines are rewritten.
//
// It replaces r.Tags with a new slice when it adds a line, as SetFullValue
// does. r.Entity is not updated; rebuild it with decoder.SyncEntities when
// it is read.
func (r *Record) SetChild(parent *Tag, tag, value string, v Version) *Tag {
	if r == nil || tag == "" || tag == "CONT" || tag == "CONC" {
		return nil
	}
	start, end, ok := childSpan(r.Tags, parent)
	if !ok {
		return nil
	}
	level := childLevel(parent)
	for _, c := range r.Tags[start:end] {
		if c != nil && c.Level == level && c.Tag == tag {
			if r.FullValue(c) != value {
				r.SetFullValue(c, value, v)
			}
			return c
		}
	}
	c := &Tag{Level: level, Tag: tag}
	r.Tags = slices.Insert(slices.Clip(r.Tags), end, c)
	r.SetFullValue(c, value, v)
	return c
}

// RemoveChildren removes every direct subordinate of parent with tag, each
// with every line beneath it, and returns how many it removed. A nil parent
// names the record itself, so r.RemoveChildren(nil, "NOTE") removes the
// record's level-1 notes. It removes nothing when parent is not in r.Tags
// (compared by pointer), or when tag is CONT or CONC, which continue
// parent's own value (SetFullValue sets that). Every other line is kept as
// it is, with its LineNumber.
//
// It replaces r.Tags with a new slice when it removes a line, as
// SetFullValue does. r.Entity is not updated; rebuild it with
// decoder.SyncEntities when it is read.
func (r *Record) RemoveChildren(parent *Tag, tag string) int {
	if r == nil || tag == "CONT" || tag == "CONC" {
		return 0
	}
	start, end, ok := childSpan(r.Tags, parent)
	if !ok {
		return 0
	}
	level := childLevel(parent)
	removed := 0
	tags := make([]*Tag, 0, len(r.Tags))
	tags = append(tags, r.Tags[:start]...)
	for i := start; i < end; {
		c := r.Tags[i]
		if c != nil && c.Level == level && c.Tag == tag {
			i += len(subtree(r.Tags[i:], c))
			removed++
			continue
		}
		tags = append(tags, c)
		i++
	}
	if removed == 0 {
		return 0
	}
	r.Tags = append(tags, r.Tags[end:]...)
	return removed
}

// childLevel returns the level of parent's direct subordinates; a nil
// parent is the record, whose subordinates are at level 1.
func childLevel(parent *Tag) int {
	if parent == nil {
		return 1
	}
	return parent.Level + 1
}

// childSpan returns the run tags[start:end] that holds parent's
// subordinates: the lines after parent in its subtree, or all of tags for a
// nil parent. ok is false when parent is not in tags.
func childSpan(tags []*Tag, parent *Tag) (start, end int, ok bool) {
	if parent == nil {
		return 0, len(tags), true
	}
	i := slices.Index(tags, parent)
	if i < 0 {
		return 0, 0, false
	}
	return i + 1, i + len(subtree(tags[i:], parent)), true
}

// continuesParent reports whether tags[i], whose parent is tags[parent] (the
// record's level-0 line when parent is -1), is a CONT or CONC continuing
// that parent's value: one level below it.
func continuesParent(tags []*Tag, i, parent int) bool {
	parentLevel := 0
	if parent >= 0 {
		parentLevel = tags[parent].Level
	}
	c := tags[i]
	return c.Level == parentLevel+1 && (c.Tag == "CONT" || c.Tag == "CONC")
}

// parentIndex returns the index of the parent of tags[i], the nearest
// earlier tag at a shallower level (SI-023), or -1 when it has none.
func parentIndex(tags []*Tag, i int) int {
	return tagParents(tags)[i]
}
