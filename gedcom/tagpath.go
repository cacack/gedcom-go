package gedcom

// Find returns the first tag in r.Tags, in document order, at the given path,
// or nil when none matches, the path is empty, or r is nil. It is the first
// element of FindAll, whose doc gives the path rules: r.Find("BIRT", "PLAC")
// is the first PLAC under any level-1 BIRT, so it comes from a later BIRT when
// the first has none.
func (r *Record) Find(path ...string) *Tag {
	if r == nil {
		return nil
	}
	return findTag(r.Tags, path)
}

// FindAll returns every tag in r.Tags at the given path, in document order,
// or nil when none matches, the path is empty, or r is nil. The returned tags
// are the record's own, so editing one edits the record.
//
// A path names tags from level 1 downward: r.FindAll("BIRT", "PLAC") returns
// the level-2 PLAC under each level-1 BIRT, and r.FindAll("NAME") returns the
// record's names but not a NAME nested deeper. A tag's parent is the nearest
// earlier tag one level shallower, so matching follows the nesting the Level
// fields describe, not tag names alone. Tag names compare exactly. Nil entries
// are skipped, a tag below level 1 ends every branch, and a tag whose level
// skips past its parent's child level (a level 3 directly under a level 1)
// matches no path.
//
// A tag's Value is one physical line. The decoder keeps CONT and CONC lines as
// separate child tags, so a continued value's later segments are in Subtree
// (or FindAll with a trailing "CONT" or "CONC"), not in Value; setting Value
// replaces only the first segment and leaves those children in place.
func (r *Record) FindAll(path ...string) []*Tag {
	if r == nil {
		return nil
	}
	return findTags(r.Tags, path, false)
}

// Subtree returns t followed by every tag nested beneath it, the contiguous
// run of r.Tags that starts at t and ends before the next tag at t's level or
// shallower, as measured by t's current Level. It returns nil when t is not
// in r.Tags (compared by pointer) or r is nil.
//
// The result is a view of r.Tags, not a copy: its elements are the record's
// tags, including any nil entries inside the run (the decoder never stores
// nil, but a hand-built Tags slice may). Its capacity is capped at its length, so appending to it copies
// rather than overwriting the tags that follow.
func (r *Record) Subtree(t *Tag) []*Tag {
	if r == nil {
		return nil
	}
	return subtree(r.Tags, t)
}

// Find returns the first tag in h.Tags at the given path, or nil. It follows
// the same rules as Record.Find: h.Find("SOUR", "VERS") is HEAD.SOUR.VERS.
func (h *Header) Find(path ...string) *Tag {
	if h == nil {
		return nil
	}
	return findTag(h.Tags, path)
}

// FindAll returns every tag in h.Tags at the given path, in document order,
// or nil. It follows the same rules as Record.FindAll.
func (h *Header) FindAll(path ...string) []*Tag {
	if h == nil {
		return nil
	}
	return findTags(h.Tags, path, false)
}

// Subtree returns t followed by every tag nested beneath it in h.Tags, or nil
// when t is not in h.Tags. It follows the same rules as Record.Subtree.
func (h *Header) Subtree(t *Tag) []*Tag {
	if h == nil {
		return nil
	}
	return subtree(h.Tags, t)
}

func findTag(tags []*Tag, path []string) *Tag {
	if found := findTags(tags, path, true); len(found) > 0 {
		return found[0]
	}
	return nil
}

// findTags walks tags once, tracking how many leading path elements the
// current chain of ancestors (levels 1..matched) matches. A tag at level L
// can extend the match only when its parent chain matched L-1 elements.
func findTags(tags []*Tag, path []string, first bool) []*Tag {
	if len(path) == 0 {
		return nil
	}
	var found []*Tag
	matched := 0
	for _, t := range tags {
		if t == nil {
			continue
		}
		level := t.Level
		if level < 1 {
			matched = 0
			continue
		}
		matched = min(matched, level-1)
		if matched != level-1 || level > len(path) || t.Tag != path[level-1] {
			continue
		}
		matched = level
		if level == len(path) {
			found = append(found, t)
			if first {
				return found
			}
		}
	}
	return found
}

func subtree(tags []*Tag, t *Tag) []*Tag {
	if t == nil {
		return nil
	}
	for i, candidate := range tags {
		if candidate != t {
			continue
		}
		end := i + 1
		for end < len(tags) && (tags[end] == nil || tags[end].Level > t.Level) {
			end++
		}
		return tags[i:end:end]
	}
	return nil
}
