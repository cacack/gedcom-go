package gedcom

// Note represents a textual note or annotation.
type Note struct {
	// XRef is the cross-reference identifier for this note
	XRef string

	// Text is the note content
	Text string

	// Continuation lines for multi-line notes
	Continuation []string

	// Tags contains all raw tags for this note (for unknown/custom tags)
	Tags []*Tag
}

// FullText returns the complete note text including continuation lines.
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
