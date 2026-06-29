package gedcom

// resolveSharedNoteText returns the text of the NOTE or SNOTE record that xref
// points to. NOTE records contribute their full text (including CONT/CONC
// continuations); SNOTE records contribute their text. The bool reports whether
// the XRef resolved to a known note record.
func resolveSharedNoteText(doc *Document, xref string) (string, bool) {
	if doc == nil {
		return "", false
	}
	if note := doc.GetNote(xref); note != nil {
		return note.FullText(), true
	}
	if snote := doc.GetSharedNote(xref); snote != nil {
		return snote.Text, true
	}
	return "", false
}

// allNotes combines a record's inline note text with the resolved text of any
// shared notes referenced by XRef. Shared notes that do not resolve against doc
// are skipped. Inline notes are returned first, in order, followed by resolved
// XRef notes in order. This backs the AllNotes method on each note-bearing
// record type.
func allNotes(doc *Document, inline, xrefs []string) []string {
	if len(inline) == 0 && len(xrefs) == 0 {
		return nil
	}
	result := make([]string, 0, len(inline)+len(xrefs))
	result = append(result, inline...)
	for _, xref := range xrefs {
		if text, ok := resolveSharedNoteText(doc, xref); ok {
			result = append(result, text)
		}
	}
	return result
}
