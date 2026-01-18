package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// normalizeXRefsToUppercase converts all XRefs to uppercase for GEDCOM 7.0.
// GEDCOM 7.0 requires XRefs to be uppercase only (@[A-Z0-9_]+@).
func normalizeXRefsToUppercase(doc *gedcom.Document, report *gedcom.ConversionReport) {
	mapping := buildXRefMapping(doc)

	if len(mapping) == 0 {
		return
	}

	updateXRefDefinitions(doc, mapping)
	updateXRefReferences(doc, mapping)
	updateXRefMap(doc, mapping)

	report.AddTransformation(gedcom.Transformation{
		Type:        "XREF_UPPERCASE",
		Description: "Normalized XRefs to uppercase (required for GEDCOM 7.0)",
		Count:       len(mapping),
	})
}

// buildXRefMapping creates a map of original XRefs to their uppercase versions.
// Only XRefs that differ from their uppercase form are included.
func buildXRefMapping(doc *gedcom.Document) map[string]string {
	mapping := make(map[string]string)

	for _, record := range doc.Records {
		if record.XRef == "" {
			continue
		}
		upper := strings.ToUpper(record.XRef)
		if record.XRef != upper {
			mapping[record.XRef] = upper
		}
	}

	return mapping
}

// updateXRefDefinitions updates Record.XRef fields and their corresponding Entity.XRef fields.
func updateXRefDefinitions(doc *gedcom.Document, mapping map[string]string) {
	for _, record := range doc.Records {
		if newXRef, ok := mapping[record.XRef]; ok {
			record.XRef = newXRef

			// Also update the XRef in the entity if present
			if record.Entity != nil {
				updateEntityXRef(record.Entity, newXRef)
			}
		}
	}
}

// updateEntityXRef updates the XRef field in typed entities.
func updateEntityXRef(entity interface{}, newXRef string) {
	switch e := entity.(type) {
	case *gedcom.Individual:
		e.XRef = newXRef
	case *gedcom.Family:
		e.XRef = newXRef
	case *gedcom.Source:
		e.XRef = newXRef
	case *gedcom.Repository:
		e.XRef = newXRef
	case *gedcom.Note:
		e.XRef = newXRef
	case *gedcom.MediaObject:
		e.XRef = newXRef
	case *gedcom.Submitter:
		e.XRef = newXRef
	}
}

// updateXRefReferences updates XRef references throughout the document.
func updateXRefReferences(doc *gedcom.Document, mapping map[string]string) {
	// Update Header references
	if doc.Header != nil {
		if newXRef, ok := mapping[doc.Header.Submitter]; ok {
			doc.Header.Submitter = newXRef
		}
		// Update any XRef values in header tags
		for _, tag := range doc.Header.Tags {
			updateXRefInTag(tag, mapping)
		}
	}

	// Update references in all records
	for _, record := range doc.Records {
		// Update XRef values in record tags
		for _, tag := range record.Tags {
			updateXRefInTag(tag, mapping)
		}

		// Update typed entity references
		if record.Entity != nil {
			updateEntityReferences(record.Entity, mapping)
		}
	}
}

// updateXRefInTag updates XRef references in a tag's value.
func updateXRefInTag(tag *gedcom.Tag, mapping map[string]string) {
	if tag == nil {
		return
	}

	if isXRef(tag.Value) {
		if newXRef, ok := mapping[tag.Value]; ok {
			tag.Value = newXRef
		}
	}
}

// updateEntityReferences updates all XRef references within typed entities.
func updateEntityReferences(entity interface{}, mapping map[string]string) {
	switch e := entity.(type) {
	case *gedcom.Individual:
		updateIndividualReferences(e, mapping)
	case *gedcom.Family:
		updateFamilyReferences(e, mapping)
	case *gedcom.Source:
		updateSourceReferences(e, mapping)
	case *gedcom.Repository:
		updateRepositoryReferences(e, mapping)
	case *gedcom.Note:
		updateNoteReferences(e, mapping)
	case *gedcom.MediaObject:
		updateMediaObjectReferences(e, mapping)
	case *gedcom.Submitter:
		updateSubmitterReferences(e, mapping)
	}
}

// updateIndividualReferences updates XRef references in an Individual entity.
func updateIndividualReferences(ind *gedcom.Individual, mapping map[string]string) {
	// ChildInFamilies
	for i := range ind.ChildInFamilies {
		if newXRef, ok := mapping[ind.ChildInFamilies[i].FamilyXRef]; ok {
			ind.ChildInFamilies[i].FamilyXRef = newXRef
		}
	}

	// SpouseInFamilies
	for i, xref := range ind.SpouseInFamilies {
		if newXRef, ok := mapping[xref]; ok {
			ind.SpouseInFamilies[i] = newXRef
		}
	}

	// Associations
	for _, assoc := range ind.Associations {
		if newXRef, ok := mapping[assoc.IndividualXRef]; ok {
			assoc.IndividualXRef = newXRef
		}
		updateSourceCitations(assoc.SourceCitations, mapping)
		updateStringSlice(assoc.Notes, mapping)
	}

	// SourceCitations
	updateSourceCitations(ind.SourceCitations, mapping)

	// Notes
	updateStringSlice(ind.Notes, mapping)

	// Media
	updateMediaLinks(ind.Media, mapping)

	// LDSOrdinances
	updateLDSOrdinances(ind.LDSOrdinances, mapping)

	// Events
	for _, event := range ind.Events {
		updateEventReferences(event, mapping)
	}

	// Attributes
	for _, attr := range ind.Attributes {
		updateSourceCitations(attr.SourceCitations, mapping)
	}

	// Tags
	for _, tag := range ind.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateFamilyReferences updates XRef references in a Family entity.
func updateFamilyReferences(fam *gedcom.Family, mapping map[string]string) {
	// Husband
	if newXRef, ok := mapping[fam.Husband]; ok {
		fam.Husband = newXRef
	}

	// Wife
	if newXRef, ok := mapping[fam.Wife]; ok {
		fam.Wife = newXRef
	}

	// Children
	for i, xref := range fam.Children {
		if newXRef, ok := mapping[xref]; ok {
			fam.Children[i] = newXRef
		}
	}

	// SourceCitations
	updateSourceCitations(fam.SourceCitations, mapping)

	// Notes
	updateStringSlice(fam.Notes, mapping)

	// Media
	updateMediaLinks(fam.Media, mapping)

	// LDSOrdinances
	updateLDSOrdinances(fam.LDSOrdinances, mapping)

	// Events
	for _, event := range fam.Events {
		updateEventReferences(event, mapping)
	}

	// Tags
	for _, tag := range fam.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateSourceReferences updates XRef references in a Source entity.
func updateSourceReferences(src *gedcom.Source, mapping map[string]string) {
	// RepositoryRef
	if newXRef, ok := mapping[src.RepositoryRef]; ok {
		src.RepositoryRef = newXRef
	}

	// Notes
	updateStringSlice(src.Notes, mapping)

	// Media
	updateMediaLinks(src.Media, mapping)

	// Tags
	for _, tag := range src.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateRepositoryReferences updates XRef references in a Repository entity.
func updateRepositoryReferences(repo *gedcom.Repository, mapping map[string]string) {
	// Notes
	updateStringSlice(repo.Notes, mapping)

	// Tags
	for _, tag := range repo.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateNoteReferences updates XRef references in a Note entity.
func updateNoteReferences(note *gedcom.Note, mapping map[string]string) {
	// Tags
	for _, tag := range note.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateMediaObjectReferences updates XRef references in a MediaObject entity.
func updateMediaObjectReferences(media *gedcom.MediaObject, mapping map[string]string) {
	// SourceCitations
	updateSourceCitations(media.SourceCitations, mapping)

	// Notes
	updateStringSlice(media.Notes, mapping)

	// Tags
	for _, tag := range media.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateSubmitterReferences updates XRef references in a Submitter entity.
func updateSubmitterReferences(subm *gedcom.Submitter, mapping map[string]string) {
	// Notes
	updateStringSlice(subm.Notes, mapping)

	// Tags
	for _, tag := range subm.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateEventReferences updates XRef references in an Event.
func updateEventReferences(event *gedcom.Event, mapping map[string]string) {
	if event == nil {
		return
	}

	// SourceCitations
	updateSourceCitations(event.SourceCitations, mapping)

	// Media
	updateMediaLinks(event.Media, mapping)

	// Notes
	updateStringSlice(event.Notes, mapping)

	// Tags
	for _, tag := range event.Tags {
		updateXRefInTag(tag, mapping)
	}
}

// updateSourceCitations updates XRef references in source citations.
func updateSourceCitations(citations []*gedcom.SourceCitation, mapping map[string]string) {
	for _, sc := range citations {
		if newXRef, ok := mapping[sc.SourceXRef]; ok {
			sc.SourceXRef = newXRef
		}
	}
}

// updateMediaLinks updates XRef references in media links.
func updateMediaLinks(links []*gedcom.MediaLink, mapping map[string]string) {
	for _, ml := range links {
		if newXRef, ok := mapping[ml.MediaXRef]; ok {
			ml.MediaXRef = newXRef
		}
	}
}

// updateLDSOrdinances updates XRef references in LDS ordinances.
func updateLDSOrdinances(ordinances []*gedcom.LDSOrdinance, mapping map[string]string) {
	for _, ord := range ordinances {
		if newXRef, ok := mapping[ord.FamilyXRef]; ok {
			ord.FamilyXRef = newXRef
		}
	}
}

// updateStringSlice updates XRef references in a string slice (e.g., Notes).
func updateStringSlice(slice []string, mapping map[string]string) {
	for i, xref := range slice {
		if newXRef, ok := mapping[xref]; ok {
			slice[i] = newXRef
		}
	}
}

// updateXRefMap rebuilds the XRefMap with new keys.
func updateXRefMap(doc *gedcom.Document, mapping map[string]string) {
	newMap := make(map[string]*gedcom.Record)
	for xref, record := range doc.XRefMap {
		if newXRef, ok := mapping[xref]; ok {
			newMap[newXRef] = record
		} else {
			newMap[xref] = record
		}
	}
	doc.XRefMap = newMap
}

// isXRef returns true if the value looks like an XRef reference (@...@).
func isXRef(value string) bool {
	if len(value) < 3 {
		return false
	}
	return value[0] == '@' && value[len(value)-1] == '@'
}

// Common XRef reference tags for reference (used for validation/documentation).
var xrefTags = map[string]bool{
	"FAMC": true, "FAMS": true, "HUSB": true, "WIFE": true, "CHIL": true,
	"SOUR": true, "REPO": true, "SUBM": true, "NOTE": true, "OBJE": true,
	"ASSO": true, "ALIA": true,
}
