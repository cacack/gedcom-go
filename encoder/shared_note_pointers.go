package encoder

import "github.com/cacack/gedcom-go/v3/gedcom"

// sharedNoteSet holds the XRefs of a document's SNOTE records. It is used
// through a pointer, with nil meaning "retag nothing", so the StreamEncoder
// that carries one stays a comparable type.
type sharedNoteSet struct {
	xrefs map[string]bool
}

// has reports whether xref names an SNOTE record. A nil set has none.
func (s *sharedNoteSet) has(xref string) bool {
	return s != nil && s.xrefs[xref]
}

// sharedNotePointers returns the set of doc's SNOTE record XRefs when the
// output is GEDCOM 7.0, and nil otherwise. The output version is
// opts.TargetVersion, or the header's version when no target is set -- the
// same version the header is written with.
//
// The entity model holds every note pointer as a bare XRef string (NoteXRefs),
// so the tag a pointer needs is decided by what it points at: in 7.0 a pointer
// to a SHARED_NOTE record is "SNOTE @X@", while "NOTE" takes text (issue
// #471). GEDCOM 5.5/5.5.1 has no SNOTE, so there every pointer stays NOTE.
func sharedNotePointers(doc *gedcom.Document, opts *EncodeOptions) *sharedNoteSet {
	version := opts.TargetVersion
	if version == "" && doc.Header != nil {
		version = doc.Header.Version
	}
	if version != gedcom.Version70 {
		return nil
	}

	// Scanned from Records rather than looked up in XRefMap: a document built
	// in Go need not have populated the map, and the records are what is
	// written.
	set := &sharedNoteSet{xrefs: make(map[string]bool)}
	for _, record := range doc.Records {
		if record != nil && record.Type == gedcom.RecordTypeSharedNote && record.XRef != "" {
			set.xrefs[record.XRef] = true
		}
	}
	return set
}

// retagSharedNotePointers rewrites each NOTE tag whose value is a pointer into
// sharedNotes as SNOTE and returns tags. It runs on the tags the entity writers
// produced, so every note-bearing structure -- records and substructures alike
// -- shares the one rule instead of each writer needing the document. A
// retagged entry is replaced by a copy rather than edited: some writers pass
// the entity's own preserved tags through (SharedNote.Tags), and encoding must
// not change the caller's document. A nil set changes nothing.
func retagSharedNotePointers(tags []*gedcom.Tag, sharedNotes *sharedNoteSet) []*gedcom.Tag {
	if sharedNotes == nil {
		return tags
	}
	for i, tag := range tags {
		if tag != nil && tag.Tag == "NOTE" && sharedNotes.has(tag.Value) {
			retagged := *tag
			retagged.Tag = "SNOTE"
			tags[i] = &retagged
		}
	}
	return tags
}
