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

	// NumberOfChildren is the declared number of children (NCHI tag)
	NumberOfChildren string

	// Events contains family events (marriage, divorce, etc.)
	Events []*Event

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Notes are references to note records
	Notes []string

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

	// LDSOrdinances are LDS (Latter-Day Saints) ordinances (SLGS - spouse sealing)
	LDSOrdinances []*LDSOrdinance

	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate

	// RefNumber is the user reference number (REFN tag)
	RefNumber string

	// UID is the unique identifier (UID tag)
	UID string

	// Tags contains all raw tags for this family (for unknown/custom tags)
	Tags []*Tag
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
