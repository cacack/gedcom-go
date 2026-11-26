package gedcom

import "github.com/cacack/gedcom-go/v3/internal/tagtree"

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
// earlier tag at a shallower level, and a tag with none is a top-level
// structure, so matching follows the nesting the Level fields describe, not
// tag names alone. A tag whose level skips past its parent's child level (a
// level 3 directly after a level 1, kept by a strict decode or built in code)
// is that parent's child: r.FindAll("BIRT", "DATE") finds it. Every package
// reads Tags with this rule (SI-023). Tag names compare exactly. Nil entries
// are skipped, and a tag below level 1 matches no path and ends its branch.
//
// A tag's Value is one physical line. The decoder keeps CONT and CONC lines as
// separate child tags, so a continued value's later segments are in Subtree
// (or FindAll with a trailing "CONT" or "CONC"), not in Value; setting Value
// replaces only the first segment and leaves those children in place.
// FullValue reads the whole value and SetFullValue replaces it.
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

// findTags walks tags once, recording for each tag how many leading path
// elements it and its ancestors match, or 0 when they do not. A tag extends
// its parent's match by one element.
func findTags(tags []*Tag, path []string, first bool) []*Tag {
	if len(path) == 0 {
		return nil
	}
	parents := tagParents(tags)
	matched := make([]int, len(tags))
	var found []*Tag
	for i, t := range tags {
		if t == nil || t.Level < 1 {
			continue
		}
		depth := 1
		if p := parents[i]; p >= 0 {
			if matched[p] == 0 {
				continue
			}
			depth = matched[p] + 1
		}
		if depth > len(path) || t.Tag != path[depth-1] {
			continue
		}
		matched[i] = depth
		if depth == len(path) {
			found = append(found, t)
			if first {
				return found
			}
		}
	}
	return found
}

// tagParents returns the index of each tag's parent in tags, or -1 for a
// top-level or nil tag, by the rule every package shares (SI-023).
func tagParents(tags []*Tag) []int {
	return tagtree.Parents(tags, func(t *Tag) (int, bool) {
		if t == nil {
			return 0, false
		}
		return t.Level, true
	})
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
