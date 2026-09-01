package gedcom

// Note represents a textual note or annotation.
type Note struct {
	// XRef is the cross-reference identifier for this note
	XRef string

	// Text is the full note content. Multi-line bodies carried by CONT
	// continuation lines are folded in on decode (joined with "\n"), so Text
	// alone is the whole note.
	Text string

	// Continuation holds additional lines of a multi-line note, each joined to
	// the previous with "\n".
	//
	// Deprecated: the decoder no longer populates this — as of #439 it folds
	// every continuation line into Text, matching SharedNote.Text. Read Text
	// (or FullText, which is now equivalent) instead. The field is still
	// honoured on encode so hand-built notes that split their body across Text
	// and Continuation keep working.
	Continuation []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this note (for unknown/custom tags)
	Tags []*Tag
}

// FullText returns the complete note text including continuation lines.
//
// On a decoded note this is the same string as Text, because the decoder folds
// continuation lines in. It still differs for a hand-built note that populates
// the deprecated Continuation slice.
//
// Deprecated: read Text, which already holds the whole body with its newlines.
// Removed in v3 alongside Continuation, the only thing that made the two
// differ.
func (n *Note) FullText() string {
	if len(n.Continuation) == 0 {
		return n.Text
	}

	result := n.Text
	for _, line := range n.Continuation {
		result += "\n" + line
	}
	return result
}
