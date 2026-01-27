package encoder

import (
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// textToTags converts a potentially multiline string value to GEDCOM tags.
// The first line becomes the primary tag at the specified level, and subsequent
// lines become CONT (continuation) tags at level+1.
//
// When opts is provided and DisableLineWrap is false, lines exceeding
// MaxLineLength are automatically split using CONC tags at word boundaries.
//
// Examples:
//   - "Single line" -> [TAG value="Single line"]
//   - "Line1\nLine2" -> [TAG value="Line1", CONT value="Line2"]
//   - "" -> [TAG value=""]
//   - "Very long line..." -> [TAG value="Very long...", CONC value="line..."]
func textToTags(value string, level int, tagName string, opts *EncodeOptions) []*gedcom.Tag {
	// Handle empty value - return single tag with empty value
	if value == "" {
		return []*gedcom.Tag{{Level: level, Tag: tagName, Value: ""}}
	}

	// Split on newlines first
	lines := strings.Split(value, "\n")

	tags := make([]*gedcom.Tag, 0, len(lines))

	// Process first line - it becomes the primary tag
	firstLineSegments := splitLineForLength(lines[0], opts)
	tags = append(tags, &gedcom.Tag{Level: level, Tag: tagName, Value: firstLineSegments[0]})

	// Add CONC tags for remaining segments of first line
	for i := 1; i < len(firstLineSegments); i++ {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CONC", Value: firstLineSegments[i]})
	}

	// Process remaining lines (from newlines) - they become CONT tags
	for i := 1; i < len(lines); i++ {
		lineSegments := splitLineForLength(lines[i], opts)

		// First segment becomes CONT
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CONT", Value: lineSegments[0]})

		// Remaining segments become CONC at level+1 (same as CONT)
		for j := 1; j < len(lineSegments); j++ {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CONC", Value: lineSegments[j]})
		}
	}

	return tags
}

// splitLineForLength splits a single line into segments that fit within MaxLineLength.
// Returns a slice with at least one element (the original line if no splitting needed).
// Attempts to split at word boundaries (spaces) when possible.
func splitLineForLength(line string, opts *EncodeOptions) []string {
	// If line wrapping is disabled or line is short enough, return as-is
	if opts != nil && opts.DisableLineWrap {
		return []string{line}
	}

	maxLen := DefaultMaxLineLength
	if opts != nil {
		maxLen = opts.effectiveMaxLineLength()
	}

	if len(line) <= maxLen {
		return []string{line}
	}

	var segments []string
	remaining := line

	for len(remaining) > maxLen {
		// Find the best split point - prefer word boundary (space)
		splitAt := findWordBoundary(remaining, maxLen)

		segments = append(segments, remaining[:splitAt])
		remaining = remaining[splitAt:]
	}

	// Add the final segment
	if remaining != "" {
		segments = append(segments, remaining)
	}

	return segments
}

// findWordBoundary finds the best position to split a line at or before maxLen.
// Prefers splitting at a space (word boundary) but falls back to maxLen if no space found.
func findWordBoundary(line string, maxLen int) int {
	if len(line) <= maxLen {
		return len(line)
	}

	// Look for last space within the maxLen limit
	lastSpace := strings.LastIndex(line[:maxLen], " ")

	if lastSpace > 0 {
		// Found a space - split after the space to keep space with first segment
		return lastSpace + 1
	}

	// No word boundary found, split at maxLen exactly
	return maxLen
}

// entityToTags converts an entity to tags based on record type.
// Returns nil if no conversion is needed (entity is nil or type not supported).
//
//nolint:gocyclo // Type switch for all GEDCOM record types requires many cases
func entityToTags(record *gedcom.Record, opts *EncodeOptions) []*gedcom.Tag {
	if record.Entity == nil {
		return nil
	}

	switch record.Type {
	case gedcom.RecordTypeIndividual:
		if indi, ok := record.Entity.(*gedcom.Individual); ok {
			return individualToTags(indi, opts)
		}
	case gedcom.RecordTypeFamily:
		if fam, ok := record.Entity.(*gedcom.Family); ok {
			return familyToTags(fam, opts)
		}
	case gedcom.RecordTypeSource:
		if src, ok := record.Entity.(*gedcom.Source); ok {
			return sourceToTags(src, opts)
		}
	case gedcom.RecordTypeSubmitter:
		if subm, ok := record.Entity.(*gedcom.Submitter); ok {
			return submitterToTags(subm, opts)
		}
	case gedcom.RecordTypeRepository:
		if repo, ok := record.Entity.(*gedcom.Repository); ok {
			return repositoryToTags(repo, opts)
		}
	case gedcom.RecordTypeNote:
		if note, ok := record.Entity.(*gedcom.Note); ok {
			return noteToTags(note)
		}
	case gedcom.RecordTypeMedia:
		if media, ok := record.Entity.(*gedcom.MediaObject); ok {
			return mediaObjectToTags(media, opts)
		}
	case gedcom.RecordTypeSharedNote:
		if snote, ok := record.Entity.(*gedcom.SharedNote); ok {
			return sharedNoteToTags(snote, opts)
		}
	}

	return nil
}

// individualToTags converts an Individual entity to GEDCOM tags.
//
//nolint:gocyclo // Converting all individual fields requires handling many cases
func individualToTags(indi *gedcom.Individual, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Names (level 1)
	for _, name := range indi.Names {
		tags = append(tags, nameToTags(name, 1)...)
	}

	// Sex (level 1)
	if indi.Sex != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "SEX", Value: indi.Sex})
	}

	// Events (level 1) - BIRT, DEAT, etc.
	for _, event := range indi.Events {
		tags = append(tags, eventToTags(event, 1, opts)...)
	}

	// Attributes (level 1) - OCCU, EDUC, etc.
	for _, attr := range indi.Attributes {
		tags = append(tags, attributeToTags(attr, 1, opts)...)
	}

	// LDS Ordinances (level 1) - BAPL, CONL, ENDL, SLGC
	for _, ord := range indi.LDSOrdinances {
		tags = append(tags, ldsOrdinanceToTags(ord, 1)...)
	}

	// Family links as child (level 1) - FAMC
	for i := range indi.ChildInFamilies {
		tags = append(tags, familyLinkToTags(&indi.ChildInFamilies[i], 1)...)
	}

	// Family links as spouse (level 1) - FAMS
	for _, famXRef := range indi.SpouseInFamilies {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "FAMS", Value: famXRef})
	}

	// Associations (level 1) - ASSO
	for _, assoc := range indi.Associations {
		tags = append(tags, associationToTags(assoc, 1, opts)...)
	}

	// Source citations (level 1) - SOUR
	for _, cite := range indi.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, 1, opts)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	for _, note := range indi.Notes {
		tags = append(tags, textToTags(note, 1, "NOTE", opts)...)
	}

	// Media links (level 1) - OBJE
	for _, media := range indi.Media {
		tags = append(tags, mediaLinkToTags(media, 1)...)
	}

	// Change date (level 1) - CHAN
	if indi.ChangeDate != nil {
		tags = append(tags, changeDateToTags(indi.ChangeDate, 1, "CHAN")...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if indi.CreationDate != nil {
		tags = append(tags, changeDateToTags(indi.CreationDate, 1, "CREA")...)
	}

	// Reference number (level 1) - REFN
	if indi.RefNumber != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REFN", Value: indi.RefNumber})
	}

	// UID (level 1)
	if indi.UID != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "UID", Value: indi.UID})
	}

	// FamilySearch Family Tree ID (level 1) - _FSFTID
	if indi.FamilySearchID != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "_FSFTID", Value: indi.FamilySearchID})
	}

	return tags
}

// familyToTags converts a Family entity to GEDCOM tags.
//
//nolint:gocyclo // Converting all family fields requires handling many cases
func familyToTags(fam *gedcom.Family, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Husband (level 1) - HUSB
	if fam.Husband != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "HUSB", Value: fam.Husband})
	}

	// Wife (level 1) - WIFE
	if fam.Wife != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "WIFE", Value: fam.Wife})
	}

	// Children (level 1) - CHIL
	for _, child := range fam.Children {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "CHIL", Value: child})
	}

	// Number of children (level 1) - NCHI
	if fam.NumberOfChildren != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "NCHI", Value: fam.NumberOfChildren})
	}

	// Events (level 1) - MARR, DIV, etc.
	for _, event := range fam.Events {
		tags = append(tags, eventToTags(event, 1, opts)...)
	}

	// LDS Ordinances (level 1) - SLGS
	for _, ord := range fam.LDSOrdinances {
		tags = append(tags, ldsOrdinanceToTags(ord, 1)...)
	}

	// Source citations (level 1) - SOUR
	for _, cite := range fam.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, 1, opts)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	for _, note := range fam.Notes {
		tags = append(tags, textToTags(note, 1, "NOTE", opts)...)
	}

	// Media links (level 1) - OBJE
	for _, media := range fam.Media {
		tags = append(tags, mediaLinkToTags(media, 1)...)
	}

	// Change date (level 1) - CHAN
	if fam.ChangeDate != nil {
		tags = append(tags, changeDateToTags(fam.ChangeDate, 1, "CHAN")...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if fam.CreationDate != nil {
		tags = append(tags, changeDateToTags(fam.CreationDate, 1, "CREA")...)
	}

	// Reference number (level 1) - REFN
	if fam.RefNumber != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REFN", Value: fam.RefNumber})
	}

	// UID (level 1)
	if fam.UID != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "UID", Value: fam.UID})
	}

	return tags
}

// sourceToTags converts a Source entity to GEDCOM tags.
func sourceToTags(src *gedcom.Source, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Title (level 1) - TITL
	if src.Title != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "TITL", Value: src.Title})
	}

	// Author (level 1) - AUTH
	if src.Author != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "AUTH", Value: src.Author})
	}

	// Publication (level 1) - PUBL
	if src.Publication != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "PUBL", Value: src.Publication})
	}

	// Text (level 1) - TEXT (with CONT/CONC for multiline/long)
	if src.Text != "" {
		tags = append(tags, textToTags(src.Text, 1, "TEXT", opts)...)
	}

	// Repository reference or inline (level 1) - REPO
	if src.RepositoryRef != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REPO", Value: src.RepositoryRef})
	} else if src.Repository != nil && src.Repository.Name != "" {
		tags = append(tags,
			&gedcom.Tag{Level: 1, Tag: "REPO"},
			&gedcom.Tag{Level: 2, Tag: "NAME", Value: src.Repository.Name},
		)
	}

	// Media links (level 1) - OBJE
	for _, media := range src.Media {
		tags = append(tags, mediaLinkToTags(media, 1)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	for _, note := range src.Notes {
		tags = append(tags, textToTags(note, 1, "NOTE", opts)...)
	}

	// Change date (level 1) - CHAN
	if src.ChangeDate != nil {
		tags = append(tags, changeDateToTags(src.ChangeDate, 1, "CHAN")...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if src.CreationDate != nil {
		tags = append(tags, changeDateToTags(src.CreationDate, 1, "CREA")...)
	}

	// Reference number (level 1) - REFN
	if src.RefNumber != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REFN", Value: src.RefNumber})
	}

	// UID (level 1)
	if src.UID != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "UID", Value: src.UID})
	}

	return tags
}

// submitterToTags converts a Submitter entity to GEDCOM tags.
func submitterToTags(subm *gedcom.Submitter, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Name (level 1) - NAME
	if subm.Name != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "NAME", Value: subm.Name})
	}

	// Address (level 1) - ADDR
	if subm.Address != nil {
		tags = append(tags, addressToTags(subm.Address, 1)...)
	}

	// Phone numbers (level 1) - PHON
	for _, phone := range subm.Phone {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "PHON", Value: phone})
	}

	// Email addresses (level 1) - EMAIL
	for _, email := range subm.Email {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "EMAIL", Value: email})
	}

	// Languages (level 1) - LANG
	for _, lang := range subm.Language {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "LANG", Value: lang})
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	for _, note := range subm.Notes {
		tags = append(tags, textToTags(note, 1, "NOTE", opts)...)
	}

	return tags
}

// repositoryToTags converts a Repository entity to GEDCOM tags.
func repositoryToTags(repo *gedcom.Repository, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Name (level 1) - NAME
	if repo.Name != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "NAME", Value: repo.Name})
	}

	// Address (level 1) - ADDR
	if repo.Address != nil {
		tags = append(tags, addressToTags(repo.Address, 1)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	for _, note := range repo.Notes {
		tags = append(tags, textToTags(note, 1, "NOTE", opts)...)
	}

	return tags
}

// noteToTags converts a Note entity to GEDCOM tags.
func noteToTags(note *gedcom.Note) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Note continuation lines (level 1) - CONT
	for _, cont := range note.Continuation {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "CONT", Value: cont})
	}

	return tags
}

// mediaObjectToTags converts a MediaObject entity to GEDCOM tags.
func mediaObjectToTags(media *gedcom.MediaObject, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Files (level 1) - FILE
	for _, file := range media.Files {
		tags = append(tags, mediaFileToTags(file, 1)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	for _, note := range media.Notes {
		tags = append(tags, textToTags(note, 1, "NOTE", opts)...)
	}

	// Source citations (level 1) - SOUR
	for _, cite := range media.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, 1, opts)...)
	}

	// Change date (level 1) - CHAN
	if media.ChangeDate != nil {
		tags = append(tags, changeDateToTags(media.ChangeDate, 1, "CHAN")...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if media.CreationDate != nil {
		tags = append(tags, changeDateToTags(media.CreationDate, 1, "CREA")...)
	}

	// Reference numbers (level 1) - REFN
	for _, refn := range media.RefNumbers {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REFN", Value: refn})
	}

	// UIDs (level 1)
	for _, uid := range media.UIDs {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "UID", Value: uid})
	}

	// Restriction (level 1) - RESN
	if media.Restriction != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "RESN", Value: media.Restriction})
	}

	return tags
}

// nameToTags converts a PersonalName to GEDCOM tags at the specified level.
func nameToTags(name *gedcom.PersonalName, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// NAME tag with full name value
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "NAME", Value: name.Full})

	// Subordinate tags at level+1
	if name.Given != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "GIVN", Value: name.Given})
	}
	if name.Surname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SURN", Value: name.Surname})
	}
	if name.Prefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NPFX", Value: name.Prefix})
	}
	if name.Suffix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NSFX", Value: name.Suffix})
	}
	if name.Nickname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NICK", Value: name.Nickname})
	}
	if name.SurnamePrefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SPFX", Value: name.SurnamePrefix})
	}
	if name.Type != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TYPE", Value: name.Type})
	}

	// Transliterations (GEDCOM 7.0 TRAN tag)
	for _, tran := range name.Transliterations {
		tags = append(tags, transliterationToTags(tran, level+1)...)
	}

	return tags
}

// transliterationToTags converts a Transliteration to GEDCOM tags at the specified level.
func transliterationToTags(tran *gedcom.Transliteration, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// TRAN tag with transliterated name value
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "TRAN", Value: tran.Value})

	// Subordinate tags at level+1
	if tran.Language != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LANG", Value: tran.Language})
	}
	if tran.Given != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "GIVN", Value: tran.Given})
	}
	if tran.Surname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SURN", Value: tran.Surname})
	}
	if tran.Prefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NPFX", Value: tran.Prefix})
	}
	if tran.Suffix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NSFX", Value: tran.Suffix})
	}
	if tran.Nickname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NICK", Value: tran.Nickname})
	}
	if tran.SurnamePrefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SPFX", Value: tran.SurnamePrefix})
	}

	return tags
}

// eventToTags converts an Event to GEDCOM tags at the specified level.
//
//nolint:gocyclo // Converting all event fields requires handling many cases
func eventToTags(event *gedcom.Event, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Event tag - for negative assertions (GEDCOM 7.0), use NO tag with event type as value
	if event.IsNegative {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "NO", Value: string(event.Type)})
	} else {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: string(event.Type)})
	}

	// Subordinate tags at level+1
	if event.Date != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: event.Date})
	}

	// Place with optional details
	if event.Place != "" {
		tags = append(tags, placeToTags(event.Place, event.PlaceDetail, level+1)...)
	}

	if event.EventTypeDetail != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TYPE", Value: event.EventTypeDetail})
	}

	if event.Cause != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CAUS", Value: event.Cause})
	}

	if event.Age != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "AGE", Value: event.Age})
	}

	if event.Agency != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "AGNC", Value: event.Agency})
	}

	// Address
	if event.Address != nil {
		tags = append(tags, addressToTags(event.Address, level+1)...)
	}

	// Contact info
	for _, phone := range event.Phone {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PHON", Value: phone})
	}
	for _, email := range event.Email {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "EMAIL", Value: email})
	}
	for _, fax := range event.Fax {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FAX", Value: fax})
	}
	for _, www := range event.Website {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "WWW", Value: www})
	}

	if event.Restriction != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "RESN", Value: event.Restriction})
	}

	if event.UID != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "UID", Value: event.UID})
	}

	if event.SortDate != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SDATE", Value: event.SortDate})
	}

	// Notes (with CONT/CONC for multiline/long)
	for _, note := range event.Notes {
		tags = append(tags, textToTags(note, level+1, "NOTE", opts)...)
	}

	// Source citations
	for _, cite := range event.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, level+1, opts)...)
	}

	// Media links
	for _, media := range event.Media {
		tags = append(tags, mediaLinkToTags(media, level+1)...)
	}

	return tags
}

// attributeToTags converts an Attribute to GEDCOM tags at the specified level.
func attributeToTags(attr *gedcom.Attribute, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Attribute tag (OCCU, EDUC, etc.) with value
	tags = append(tags, &gedcom.Tag{Level: level, Tag: attr.Type, Value: attr.Value})

	// Subordinate tags at level+1
	if attr.Date != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: attr.Date})
	}

	if attr.Place != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PLAC", Value: attr.Place})
	}

	// Source citations
	for _, cite := range attr.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, level+1, opts)...)
	}

	return tags
}

// sourceCitationToTags converts a SourceCitation to GEDCOM tags at the specified level.
func sourceCitationToTags(cite *gedcom.SourceCitation, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// SOUR tag with source XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "SOUR", Value: cite.SourceXRef})

	// Subordinate tags at level+1
	if cite.Page != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PAGE", Value: cite.Page})
	}

	if cite.Quality > 0 {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "QUAY", Value: strconv.Itoa(cite.Quality)})
	}

	// DATA subordinate
	if cite.Data != nil {
		tags = append(tags, sourceCitationDataToTags(cite.Data, level+1, opts)...)
	}

	// Ancestry APID (vendor extension)
	if cite.AncestryAPID != nil {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "_APID", Value: cite.AncestryAPID.Raw})
	}

	return tags
}

// sourceCitationDataToTags converts SourceCitationData to GEDCOM tags at the specified level.
func sourceCitationDataToTags(data *gedcom.SourceCitationData, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// DATA tag
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "DATA"})

	// Subordinate tags at level+1
	if data.Date != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: data.Date})
	}

	// Text (with CONT/CONC for multiline/long)
	if data.Text != "" {
		tags = append(tags, textToTags(data.Text, level+1, "TEXT", opts)...)
	}

	return tags
}

// addressToTags converts an Address to GEDCOM tags at the specified level.
func addressToTags(addr *gedcom.Address, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// ADDR tag with optional first line value
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "ADDR", Value: addr.Line1})

	// Subordinate tags at level+1
	if addr.Line1 != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "ADR1", Value: addr.Line1})
	}
	if addr.Line2 != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "ADR2", Value: addr.Line2})
	}
	if addr.Line3 != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "ADR3", Value: addr.Line3})
	}
	if addr.City != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CITY", Value: addr.City})
	}
	if addr.State != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "STAE", Value: addr.State})
	}
	if addr.PostalCode != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "POST", Value: addr.PostalCode})
	}
	if addr.Country != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CTRY", Value: addr.Country})
	}

	return tags
}

// placeToTags converts place information to GEDCOM tags at the specified level.
func placeToTags(placeName string, detail *gedcom.PlaceDetail, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// PLAC tag with place name
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "PLAC", Value: placeName})

	// Add detail subordinates if present
	if detail != nil {
		if detail.Form != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FORM", Value: detail.Form})
		}

		// Coordinates via MAP
		if detail.Coordinates != nil {
			tags = append(tags, coordinatesToTags(detail.Coordinates, level+1)...)
		}
	}

	return tags
}

// coordinatesToTags converts Coordinates to GEDCOM tags at the specified level.
func coordinatesToTags(coords *gedcom.Coordinates, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// MAP tag
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "MAP"})

	// Subordinate tags at level+1
	if coords.Latitude != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LATI", Value: coords.Latitude})
	}
	if coords.Longitude != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LONG", Value: coords.Longitude})
	}

	return tags
}

// ldsOrdinanceToTags converts an LDSOrdinance to GEDCOM tags at the specified level.
func ldsOrdinanceToTags(ord *gedcom.LDSOrdinance, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Ordinance tag (BAPL, CONL, ENDL, SLGC, SLGS)
	tags = append(tags, &gedcom.Tag{Level: level, Tag: string(ord.Type)})

	// Subordinate tags at level+1
	if ord.Date != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: ord.Date})
	}

	if ord.Temple != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TEMP", Value: ord.Temple})
	}

	if ord.Place != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PLAC", Value: ord.Place})
	}

	if ord.Status != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "STAT", Value: ord.Status})
	}

	// FAMC for SLGC (sealing to parents)
	if ord.FamilyXRef != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FAMC", Value: ord.FamilyXRef})
	}

	return tags
}

// familyLinkToTags converts a FamilyLink to GEDCOM tags at the specified level.
func familyLinkToTags(link *gedcom.FamilyLink, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// FAMC tag with family XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "FAMC", Value: link.FamilyXRef})

	// Subordinate tags at level+1
	if link.Pedigree != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PEDI", Value: link.Pedigree})
	}

	return tags
}

// associationToTags converts an Association to GEDCOM tags at the specified level.
func associationToTags(assoc *gedcom.Association, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// ASSO tag with individual XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "ASSO", Value: assoc.IndividualXRef})

	// Subordinate tags at level+1
	// PHRASE (GEDCOM 7.0) - human-readable description of the association
	if assoc.Phrase != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PHRASE", Value: assoc.Phrase})
	}

	// Use ROLE for GEDCOM 7.0 compatibility (also compatible with 5.5.1 RELA)
	if assoc.Role != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "ROLE", Value: assoc.Role})
	}

	// Source citations (GEDCOM 7.0)
	for _, cite := range assoc.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, level+1, opts)...)
	}

	// Notes (with CONT/CONC for multiline/long)
	for _, note := range assoc.Notes {
		tags = append(tags, textToTags(note, level+1, "NOTE", opts)...)
	}

	return tags
}

// changeDateToTags converts a ChangeDate to GEDCOM tags at the specified level.
func changeDateToTags(cd *gedcom.ChangeDate, level int, tagName string) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// CHAN or CREA tag
	tags = append(tags, &gedcom.Tag{Level: level, Tag: tagName})

	// Subordinate DATE at level+1
	if cd.Date != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: cd.Date})

		// TIME subordinate at level+2
		if cd.Time != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 2, Tag: "TIME", Value: cd.Time})
		}
	}

	return tags
}

// mediaLinkToTags converts a MediaLink to GEDCOM tags at the specified level.
func mediaLinkToTags(link *gedcom.MediaLink, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// OBJE tag with media XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "OBJE", Value: link.MediaXRef})

	// Subordinate tags at level+1
	if link.Crop != nil {
		tags = append(tags, cropRegionToTags(link.Crop, level+1)...)
	}

	if link.Title != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TITL", Value: link.Title})
	}

	return tags
}

// cropRegionToTags converts a CropRegion to GEDCOM tags at the specified level.
func cropRegionToTags(crop *gedcom.CropRegion, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// CROP tag
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "CROP"})

	// Subordinate tags at level+1
	if crop.Top != 0 {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TOP", Value: strconv.Itoa(crop.Top)})
	}
	if crop.Left != 0 {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LEFT", Value: strconv.Itoa(crop.Left)})
	}
	if crop.Height != 0 {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "HEIGHT", Value: strconv.Itoa(crop.Height)})
	}
	if crop.Width != 0 {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "WIDTH", Value: strconv.Itoa(crop.Width)})
	}

	return tags
}

// mediaFileToTags converts a MediaFile to GEDCOM tags at the specified level.
func mediaFileToTags(file *gedcom.MediaFile, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// FILE tag with file reference
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "FILE", Value: file.FileRef})

	// FORM subordinate at level+1
	if file.Form != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FORM", Value: file.Form})

		// MEDI subordinate at level+2
		if file.MediaType != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 2, Tag: "MEDI", Value: file.MediaType})
		}
	}

	// TITL subordinate at level+1
	if file.Title != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TITL", Value: file.Title})
	}

	// Translations (TRAN)
	for _, tran := range file.Translations {
		tags = append(tags, mediaTranslationToTags(tran, level+1)...)
	}

	return tags
}

// mediaTranslationToTags converts a MediaTranslation to GEDCOM tags at the specified level.
func mediaTranslationToTags(tran *gedcom.MediaTranslation, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// TRAN tag with file reference
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "TRAN", Value: tran.FileRef})

	// FORM subordinate at level+1
	if tran.Form != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FORM", Value: tran.Form})
	}

	return tags
}

// sharedNoteToTags converts a SharedNote entity to GEDCOM tags.
// GEDCOM 7.0 SNOTE records include MIME types, language tags, and translations.
func sharedNoteToTags(note *gedcom.SharedNote, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// MIME (level 1)
	if note.MIME != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "MIME", Value: note.MIME})
	}

	// LANG (level 1)
	if note.Language != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "LANG", Value: note.Language})
	}

	// Translations (level 1) - TRAN with nested MIME/LANG at level 2
	for _, tran := range note.Translations {
		tags = append(tags, sharedNoteTranslationToTags(tran, 1)...)
	}

	// Source citations (level 1) - SOUR
	for _, cite := range note.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, 1, opts)...)
	}

	// External IDs (level 1) - EXID
	tags = append(tags, externalIDsToTags(note.ExternalIDs, 1)...)

	// Change date (level 1) - CHAN
	if note.ChangeDate != nil {
		tags = append(tags, changeDateToTags(note.ChangeDate, 1, "CHAN")...)
	}

	// Preserved unknown tags
	tags = append(tags, note.Tags...)

	return tags
}

// sharedNoteTranslationToTags converts a SharedNoteTranslation to GEDCOM tags at the specified level.
func sharedNoteTranslationToTags(tran *gedcom.SharedNoteTranslation, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// TRAN tag with translated text
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "TRAN", Value: tran.Value})

	// MIME subordinate at level+1
	if tran.MIME != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "MIME", Value: tran.MIME})
	}

	// LANG subordinate at level+1
	if tran.Language != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LANG", Value: tran.Language})
	}

	return tags
}

// externalIDsToTags converts a slice of ExternalIDs to GEDCOM tags at the specified level.
func externalIDsToTags(externalIDs []*gedcom.ExternalID, level int) []*gedcom.Tag {
	var tags []*gedcom.Tag

	for _, exid := range externalIDs {
		// EXID tag with external identifier value
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "EXID", Value: exid.Value})

		// TYPE subordinate at level+1
		if exid.Type != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TYPE", Value: exid.Type})
		}
	}

	return tags
}
