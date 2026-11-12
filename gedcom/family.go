package gedcom

// Family represents a family unit (husband, wife, and children).
type Family struct {
	// XRef is the cross-reference identifier for this family
	XRef string

	// Restriction is the record's restriction notice (RESN); see
	// Individual.Restriction.
	Restriction string

	// Husband is the XRef to the husband individual
	Husband string

	// HusbandPhrase is the PHRASE under HUSB (GEDCOM 7.0), e.g. describing
	// a husband recorded as @VOID@. Written only beneath a Husband.
	HusbandPhrase string

	// Wife is the XRef to the wife individual
	Wife string

	// WifePhrase is the PHRASE under WIFE (GEDCOM 7.0). Written only beneath
	// a Wife.
	WifePhrase string

	// Children are XRefs to child individuals.
	//
	// A CHIL's PHRASE (GEDCOM 7.0) has no typed field: Children is a plain
	// XRef slice, so it stays in Tags (ADR 0010).
	Children []string

	// ChildLinks are the per-child relationships to the father and mother
	// that vendor exports record under a CHIL line as _FREL and _MREL
	// (Family Tree Maker, Ancestry, Legacy), in file order. A CHIL with
	// neither tag has no entry, so Children stays the list of children and
	// ChildLinks only adds to it. The encoder writes each link under the
	// first CHIL in Children with the same XRef that has no link yet; a link
	// whose XRef is not in Children is not written. On a decoded family,
	// edits reach the output only through encoder.SyncTags. See ChildLink.
	ChildLinks []ChildLink

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
	// a lossless read-side record per ADR 0003. While Record.Tags is non-empty
	// it does not drive encoding; a record written from its entity appends
	// the extension (_-prefixed) level-1 tags here after its typed fields.
	// See Record.Tags for the rule that governs what a decoded record writes.
	Tags []*Tag
}

// ChildLink is a child's relationship to each parent of a family, from the
// vendor extensions _FREL (to the father) and _MREL (to the mother) beneath
// the family's CHIL line:
//
//	1 CHIL @I3@
//	2 _FREL Natural
//	2 _MREL Adopted
//
// The values are kept as written (Natural, Adopted, Step, Foster, Guardian,
// ...); an absent tag leaves its field empty. This is how Family Tree Maker,
// Ancestry and Legacy record adoption and step relationships, which GEDCOM
// models only as a single PEDI on the child's FAMC (FamilyLink.Pedigree).
type ChildLink struct {
	// XRef is the child the relationships describe, matching an entry in
	// Family.Children.
	XRef string

	// FatherRelation is the _FREL value: the child's relationship to the
	// family's husband.
	FatherRelation string

	// MotherRelation is the _MREL value: the child's relationship to the
	// family's wife.
	MotherRelation string
}

// LinkForChild returns the first ChildLink whose XRef is xref, or nil if the
// family records no relationship for that child. The pointer refers into
// ChildLinks, so writes through it edit the family; appending to ChildLinks
// may invalidate it. Safe on a nil receiver.
func (f *Family) LinkForChild(xref string) *ChildLink {
	if f == nil {
		return nil
	}
	for k := range f.ChildLinks {
		if f.ChildLinks[k].XRef == xref {
			return &f.ChildLinks[k]
		}
	}
	return nil
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

// MarriageEvent returns the first marriage event (MARR) for this family, or nil
// if none found. A negative assertion (NO MARR, IsNegative) is skipped. Safe on a
// nil receiver.
func (f *Family) MarriageEvent() *Event {
	if f == nil {
		return nil
	}
	return firstOccurredEvent(f.Events, EventMarriage)
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
