package gedcom

import "fmt"

// MediaLinks returns r's record-level media links: the OBJE structures directly
// under an individual, family or source record. Links nested deeper, on an
// event, attribute or source citation, are not included; read those from the
// structure that holds them.
//
// ok is false when the typed entity of r's record type holds no record-level
// media links, and when r or its Entity is nil. A nil slice with ok true is a
// record that can hold links and has none. Individual, family and source
// records hold them. A submitter record's OBJE links are not typed, so ok is
// false for it and they are read from Record.Tags.
//
// The links are read from the typed entity, not from Record.Tags, so a link
// added with AddMediaLink is listed here before encoder.SyncTags has put it
// where the encoder writes it from.
//
// The slice is the entity's own, not a copy: changing an element changes the
// record. Use AddMediaLink to append.
func (r *Record) MediaLinks() (links []*MediaLink, ok bool) {
	slot := mediaLinkSlot(r)
	if slot == nil {
		return nil, false
	}
	return *slot, true
}

// AddMediaLink appends link to r's record-level media links. It does not look
// for an equal link already present, so adding one twice links it twice.
//
// The link is added to the typed entity only. Record.Tags is authoritative when
// encoding, so on a record that has raw Tags (any decoded record) call
// encoder.SyncTags on the document afterwards for the encoder to write the new
// link.
//
// It returns ErrNilMediaLink for a nil link, and an error wrapping
// ErrNoMediaLinks, leaving r unchanged, when MediaLinks would report ok false
// for r.
func (r *Record) AddMediaLink(link *MediaLink) error {
	if link == nil {
		return ErrNilMediaLink
	}
	slot := mediaLinkSlot(r)
	if slot == nil {
		if r == nil {
			return fmt.Errorf("add media link: nil record: %w", ErrNoMediaLinks)
		}
		return fmt.Errorf("add media link to %s record %q: %w", r.Type, r.XRef, ErrNoMediaLinks)
	}
	*slot = append(*slot, link)
	return nil
}

// mediaLinkSlot returns the address of the record-level Media field of r's
// entity, or nil when r is nil or its entity is nil, a typed nil pointer, or a
// type with no such field. It is the switch behind MediaLinks and AddMediaLink;
// TestMediaLinksCompleteness fails when an entity type it lists gains a Media
// field without joining it. The encoder (entity_writer.go), the converter
// (void.go) and the validator (streaming.go) each name the same three entity
// types for their own walks, so a new carrier must be added there too.
func mediaLinkSlot(r *Record) *[]*MediaLink {
	if r == nil {
		return nil
	}
	switch e := r.Entity.(type) {
	case *Individual:
		if e != nil {
			return &e.Media
		}
	case *Family:
		if e != nil {
			return &e.Media
		}
	case *Source:
		if e != nil {
			return &e.Media
		}
	}
	return nil
}
