package gedcom

// Family represents a family unit (husband, wife, and children).
type Family struct {
	// XRef is the cross-reference identifier for this family
	XRef string

	// Husband is the XRef to the husband individual
	Husband string

	// Wife is the XRef to the wife individual
	Wife string

	// Children are XRefs to child individuals
	Children []string

	// Events contains family events (marriage, divorce, etc.) in file order.
	//
	// It also holds GEDCOM 7.0 negative assertions (n NO <EVENT>), decoded as
	// events with IsNegative set: a record that an event did NOT happen, not a
	// fact that it did. Use OccurredEvents for the events that happened and
	// NegativeAssertions for the NO assertions.
	Events []*Event

	// Attributes contains family attributes (NCHI, FACT).
	// This is the family counterpart of Individual.Attributes.
	//
	// CENS and RESI are not here: they decode into Events, matching how
	// Individual models the same two tags.
	Attributes []*Attribute

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this record
	// (1 NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

	// LDSOrdinances are LDS (Latter-Day Saints) ordinances (SLGS - spouse sealing)
	LDSOrdinances []*LDSOrdinance

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate

	// RefNumbers are user reference numbers (REFN tags, repeatable), each
	// with its optional TYPE, in file order.
	RefNumbers []RefNumber

	// UIDs are unique identifiers (UID tags, repeatable in GEDCOM 7.0), in
	// file order.
	UIDs []string

	// ExternalIDs are external identifiers (EXID tags, GEDCOM 7.0).
	// Links this record to external systems like FamilySearch, Ancestry, etc.
	ExternalIDs []*ExternalID

	// Tags contains all raw tags for this family (for unknown/custom tags),
	// a lossless read-side record per ADR 0003. It does not drive encoding --
	// see Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag
}

// AllNotes returns this family's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (f *Family) AllNotes(doc *Document) []string {
	return allNotes(doc, f.InlineNotes, f.NoteXRefs)
}

// OccurredEvents returns the events recorded as having happened: Events in file
// order, without nil entries and without GEDCOM 7.0 negative assertions
// (IsNegative). It returns nil when there are none, and is safe on a nil
// receiver. The returned slice is new; the events are shared with Events.
func (f *Family) OccurredEvents() []*Event {
	if f == nil {
		return nil
	}
	return filterEvents(f.Events, false)
}

// NegativeAssertions returns the GEDCOM 7.0 negative assertions (n NO <EVENT>)
// recorded on this family -- the events with IsNegative set, in file order.
// Each one states that the event did not happen, optionally within its DATE
// period. It returns nil when there are none, and is safe on a nil receiver.
// The returned slice is new; the events are shared with Events.
func (f *Family) NegativeAssertions() []*Event {
	if f == nil {
		return nil
	}
	return filterEvents(f.Events, true)
}

// NumberOfChildren returns the value of the NCHI entry in Attributes, or "" if
// the family has none. It is nil-safe.
//
// This replaces the NumberOfChildren field removed in v3. That field was a
// second store for a fact Attributes already held, and the richer of the two
// won on encode -- so writing the field on a decoded family was silently
// discarded. Attributes is now the single store, and it carries the NCHI
// line's subordinates as well as its value.
func (f *Family) NumberOfChildren() string {
	if f == nil {
		return ""
	}
	for _, attr := range f.Attributes {
		if attr != nil && attr.Type == AttributeNumberOfChildren {
			return attr.Value
		}
	}
	return ""
}

// SetNumberOfChildren sets the value of the family's NCHI attribute, adding the
// attribute if it has none and leaving any subordinates on an existing entry
// intact. It is a no-op on a nil Family.
//
// This keeps the write typed and discoverable. Without it the only way to set a
// child count would be to find or append &Attribute{Type: AttributeNumberOfChildren}
// by hand.
//
// Note that on a *decoded* family this changes the typed model only. Record.Tags
// is authoritative on encode, so edit the NCHI tag there (or clear Tags) to
// change encoded output -- see the Record.Tags documentation.
func (f *Family) SetNumberOfChildren(value string) {
	if f == nil {
		return
	}
	for _, attr := range f.Attributes {
		if attr != nil && attr.Type == AttributeNumberOfChildren {
			attr.Value = value
			return
		}
	}
	f.Attributes = append(f.Attributes, &Attribute{Type: AttributeNumberOfChildren, Value: value})
}

// HusbandIndividual returns the Individual record for the husband.
// Returns nil if the document is nil, Husband xref is empty, or the individual is not found.
func (f *Family) HusbandIndividual(doc *Document) *Individual {
	if doc == nil || f.Husband == "" {
		return nil
	}
	return doc.GetIndividual(f.Husband)
}

// WifeIndividual returns the Individual record for the wife.
// Returns nil if the document is nil, Wife xref is empty, or the individual is not found.
func (f *Family) WifeIndividual(doc *Document) *Individual {
	if doc == nil || f.Wife == "" {
		return nil
	}
	return doc.GetIndividual(f.Wife)
}

// ChildrenIndividuals returns Individual records for all children in this family.
// Invalid xrefs are filtered out. Order is preserved from the GEDCOM file.
// Returns an empty slice if the document is nil or there are no children.
func (f *Family) ChildrenIndividuals(doc *Document) []*Individual {
	if doc == nil {
		return []*Individual{}
	}
	result := make([]*Individual, 0, len(f.Children))
	for _, childXRef := range f.Children {
		if child := doc.GetIndividual(childXRef); child != nil {
			result = append(result, child)
		}
	}
	return result
}

// AllMembers returns all Individual records for this family (husband, wife, children).
// Order: husband first (if present), wife second (if present), then children.
// Invalid xrefs are filtered out.
// Returns an empty slice if the document is nil or no members are found.
func (f *Family) AllMembers(doc *Document) []*Individual {
	if doc == nil {
		return []*Individual{}
	}
	result := make([]*Individual, 0, 2+len(f.Children))

	if husband := f.HusbandIndividual(doc); husband != nil {
		result = append(result, husband)
	}
	if wife := f.WifeIndividual(doc); wife != nil {
		result = append(result, wife)
	}
	result = append(result, f.ChildrenIndividuals(doc)...)
	return result
}
