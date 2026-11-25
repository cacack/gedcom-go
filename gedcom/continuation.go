package gedcom

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/internal/concsplit"
)

// FullValue returns t's value with its CONT and CONC lines folded in, as the
// decoder folds a typed value such as Event.Description: each CONT adds a line
// break and its value, each CONC its value alone. The continuation lines are
// t's direct subordinates in r.Tags. With none, or when t is not in r.Tags
// (compared by pointer), it returns t.Value; a nil t returns "".
//
// The value is the wire form, as Tag.Value is: an escaped "@@" is not
// unescaped.
func (r *Record) FullValue(t *Tag) string {
	if t == nil {
		return ""
	}
	var sub []*Tag
	if r != nil {
		sub = subtree(r.Tags, t)
	}
	var b strings.Builder
	folded := false
	for _, c := range sub {
		if !isContinuationOf(c, t) {
			continue
		}
		if !folded {
			b.WriteString(t.Value)
			folded = true
		}
		if c.Tag == "CONT" {
			b.WriteByte('\n')
		}
		b.WriteString(c.Value)
	}
	if !folded {
		return t.Value
	}
	return b.String()
}

// SetFullValue replaces t's value and its CONT and CONC lines with value,
// split as the encoder splits a typed value for GEDCOM version v: a CONT for
// each line break (CRLF, LF or CR), and a CONC for each piece of a line longer
// than encoder.DefaultMaxLineLength bytes, except for GEDCOM 7.0, which has no
// CONC. An unknown version is split as 5.x is, as the encoder does.
//
// t keeps its LineNumber and takes the first piece as its Value. The new
// continuation lines go directly after t, with no LineNumber, and t's other
// subordinates, such as a PLAC's MAP, are kept as they are. So under a
// layout-preserving encode only t and its continuation lines are rewritten.
//
// value is the wire form, as Tag.Value is: it is not escaped. SetFullValue
// reports whether t was in r.Tags (compared by pointer); when it was not,
// nothing changes. It replaces r.Tags with a new slice, so a view from
// Subtree taken before the call no longer reflects the record.
func (r *Record) SetFullValue(t *Tag, value string, v Version) bool {
	if r == nil || t == nil {
		return false
	}
	start := -1
	for i, c := range r.Tags {
		if c == t {
			start = i
			break
		}
	}
	if start < 0 {
		return false
	}
	end := start + len(subtree(r.Tags[start:], t))

	maxLen := concsplit.MaxValueLength
	if v == Version70 {
		maxLen = 0
	}
	segments := concsplit.Fold(value, maxLen, nil)

	tags := make([]*Tag, 0, len(r.Tags)+len(segments))
	tags = append(tags, r.Tags[:start+1]...)
	for _, seg := range segments[1:] {
		tags = append(tags, &Tag{Level: t.Level + 1, Tag: seg.Tag, Value: seg.Value})
	}
	for _, c := range r.Tags[start+1 : end] {
		if !isContinuationOf(c, t) {
			tags = append(tags, c)
		}
	}
	tags = append(tags, r.Tags[end:]...)

	t.Value = segments[0].Value
	r.Tags = tags
	return true
}

// isContinuationOf reports whether c, a tag in t's subtree, is one of t's
// CONT or CONC lines: a direct subordinate with one of those tags.
func isContinuationOf(c, t *Tag) bool {
	return c != nil && c.Level == t.Level+1 && (c.Tag == "CONT" || c.Tag == "CONC")
}
