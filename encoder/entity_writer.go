package encoder

import (
	"slices"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/concsplit"
)

// textToTags converts a potentially multiline string value to GEDCOM tags.
// The first line becomes the primary tag at the specified level, and subsequent
// lines become CONT (continuation) tags at level+1.
//
// When opts is provided and DisableLineWrap is false, lines exceeding
// MaxLineLength are automatically split using CONC tags (see splitLineForLength).
// Each line is escaped for the output version before the split (opts.text), so
// the escaped line is what has to fit, and [concsplit.Split] never cuts inside
// an escaped "@@".
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

	// splitValueLines rather than strings.Split(value, "\n"): a CRLF source
	// leaves a trailing "\r" on every segment of a plain "\n" split, and that
	// stray CR is a line break to every reader of the output. It used to be
	// invisible because it landed immediately before the line ending the
	// encoder wrote anyway; now that writeValue treats a CR in a value as a
	// break to fold, it would emit an extra empty CONT and put a blank line
	// into the text on re-decode. One splitter for the whole package keeps the
	// encoder's idea of "a line break" from differing by call site.
	lines := splitValueLines(value)

	tags := make([]*gedcom.Tag, 0, len(lines))

	// Process first line - it becomes the primary tag
	firstLineSegments := splitLineForLength(opts.text(lines[0]), opts)
	tags = append(tags, &gedcom.Tag{Level: level, Tag: tagName, Value: firstLineSegments[0]})

	// Add CONC tags for remaining segments of first line
	for i := 1; i < len(firstLineSegments); i++ {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CONC", Value: firstLineSegments[i]})
	}

	// Process remaining lines (from newlines) - they become CONT tags
	for i := 1; i < len(lines); i++ {
		lineSegments := splitLineForLength(opts.text(lines[i]), opts)

		// First segment becomes CONT
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CONT", Value: lineSegments[0]})

		// Remaining segments become CONC at level+1 (same as CONT)
		for j := 1; j < len(lineSegments); j++ {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CONC", Value: lineSegments[j]})
		}
	}

	return tags
}

// appendPhrase appends a GEDCOM 7.0 PHRASE line at level when phrase is
// non-empty. Callers invoke it right after writing the value the PHRASE
// qualifies, so a phrase is never written without its value (ADR 0010).
func appendPhrase(tags []*gedcom.Tag, phrase string, level int, opts *EncodeOptions) []*gedcom.Tag {
	if phrase == "" {
		return tags
	}
	return append(tags, &gedcom.Tag{Level: level, Tag: "PHRASE", Value: opts.text(phrase)})
}

// ageToTags writes an AGE line at level with its GEDCOM 7.0 PHRASE beneath.
// A phrase-only age (GEDCOM 7.0 allows an empty AGE payload qualified only by
// a PHRASE) writes an AGE with an empty value so the PHRASE has a line to
// qualify, for 7.0 output only: 5.x output writes no empty AGE line, as
// dateToTags writes no empty DATE line there.
func ageToTags(age, phrase string, level int, opts *EncodeOptions) []*gedcom.Tag {
	if age == "" && (phrase == "" || !opts.is70()) {
		return nil
	}
	tags := []*gedcom.Tag{{Level: level, Tag: "AGE", Value: opts.text(age)}}
	return appendPhrase(tags, phrase, level+1, opts)
}

// dateToTags writes a DATE line at level with its GEDCOM 7.0 PHRASE beneath.
// The PHRASE comes from parsed.PhraseSubstructure and is written for 7.0
// output only; 5.x output drops it rather than folding it into the value, as
// the converter does on a 7.0 downgrade. A phrase-only date writes a DATE
// with an empty value so the PHRASE has a line to qualify.
func dateToTags(date string, parsed *gedcom.Date, level int, opts *EncodeOptions) []*gedcom.Tag {
	phrase := ""
	if opts.is70() {
		phrase = parsed.PhraseSubstructure(date)
	}
	if date == "" && phrase == "" {
		return nil
	}
	tags := []*gedcom.Tag{{Level: level, Tag: "DATE", Value: opts.text(date)}}
	return appendPhrase(tags, phrase, level+1, opts)
}

// notesToTags writes the NOTE lines for a structure at level, in order: the
// shared-note XRef pointers first (emitted as "NOTE @Nn@"; writeRecord retags a
// 7.0 pointer to an SNOTE record as SNOTE, see retagSharedNotePointers), then
// the inline note text with its CONT/CONC lines. Every writer that emits notes
// goes through here, so notes are resolved in exactly one place -- the
// invariant TestDecodedNotesEncodeOnce asserts.
//
// The two are written differently because they are different things: a pointer
// verbatim, and text escaped by textToTags, so inline text shaped like a
// pointer ("@I1@") is written "@@I1@" and does not re-decode as a link to
// whatever record carries that identifier (#708).
func notesToTags(noteXRefs, inlineNotes []string, level int, opts *EncodeOptions) []*gedcom.Tag {
	tags := make([]*gedcom.Tag, 0, len(noteXRefs)+len(inlineNotes))
	for _, xref := range noteXRefs {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "NOTE", Value: xref})
	}
	for _, note := range inlineNotes {
		tags = append(tags, textToTags(note, level, "NOTE", opts)...)
	}
	return tags
}

// splitLineForLength splits a single line into segments that fit within MaxLineLength.
// Returns a slice with at least one element (the original line if no splitting needed).
// Each later segment is written as CONC, so this must not run for GEDCOM 7.0
// output, which has no CONC: the encoder entry points arrange that through
// [EncodeOptions.forOutput], which sets DisableLineWrap for 7.0. Where the cut
// falls is [concsplit.Split]'s rule, which the converter shares.
func splitLineForLength(line string, opts *EncodeOptions) []string {
	if opts != nil && opts.DisableLineWrap {
		return []string{line}
	}

	maxLen := DefaultMaxLineLength
	if opts != nil {
		maxLen = opts.effectiveMaxLineLength()
	}
	return concsplit.Split(line, maxLen)
}

// entityToTags converts an entity to tags based on record type.
// Returns nil if no conversion is needed (entity is nil or type not supported).
//
// Every *ToTags writer below returns nil for a nil argument, which is what makes
// a nil element in a hand-built slice harmless: the loops append the writer's
// result without inspecting the element, and appending nothing is a no-op. Only
// externalIDsToTags, which reads its elements inline instead of delegating,
// skips nils itself. Callers construct these values, so a nil element is a
// caller bug the encoder cannot repair -- but it carries no data, so skipping it
// loses nothing and keeps the document encodable. gedcom.Visit and gedcom.Apply
// walk the same entities the same way. See the package documentation for the
// full nil policy.
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
	case gedcom.RecordTypeSubmission:
		if subn, ok := record.Entity.(*gedcom.Submission); ok {
			return submissionToTags(subn, opts)
		}
	case gedcom.RecordTypeRepository:
		if repo, ok := record.Entity.(*gedcom.Repository); ok {
			return repositoryToTags(repo, opts)
		}
	case gedcom.RecordTypeNote:
		// A Note carries its whole body in Text, which entityRecordText writes
		// as the level-0 value plus CONT/CONC. Nothing is left for this path.
		return nil
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
	if indi == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Restriction notice (level 1) - RESN, first as every version orders it
	if indi.Restriction != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "RESN", Value: opts.text(indi.Restriction)})
	}

	// Names (level 1)
	for _, name := range indi.Names {
		tags = append(tags, nameToTags(name, 1, opts)...)
	}

	// Sex (level 1)
	if indi.Sex != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "SEX", Value: opts.text(indi.Sex)})
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
		tags = append(tags, ldsOrdinanceToTags(ord, 1, opts)...)
	}

	// Family links as child (level 1) - FAMC
	for i := range indi.ChildInFamilies {
		tags = append(tags, familyLinkToTags(&indi.ChildInFamilies[i], "FAMC", 1, opts)...)
	}

	// Family links as spouse (level 1) - FAMS
	for i := range indi.SpouseInFamilies {
		tags = append(tags, familyLinkToTags(&indi.SpouseInFamilies[i], "FAMS", 1, opts)...)
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
	tags = append(tags, notesToTags(indi.NoteXRefs, indi.InlineNotes, 1, opts)...)

	// Media links (level 1) - OBJE
	for _, media := range indi.Media {
		tags = append(tags, mediaLinkToTags(media, 1, opts)...)
	}

	// Change date (level 1) - CHAN
	if indi.ChangeDate != nil {
		tags = append(tags, changeDateToTags(indi.ChangeDate, 1, "CHAN", opts)...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if indi.CreationDate != nil {
		tags = append(tags, changeDateToTags(indi.CreationDate, 1, "CREA", opts)...)
	}

	// Reference numbers (level 1) - REFN, each with its optional TYPE
	tags = append(tags, refNumbersToTags(indi.RefNumbers, opts)...)

	// UIDs (level 1)
	tags = append(tags, uidsToTags(indi.UIDs, 1, opts)...)

	// External IDs (level 1) - EXID (GEDCOM 7.0)
	tags = append(tags, externalIDsToTags(indi.ExternalIDs, 1, opts)...)

	// FamilySearch Family Tree ID (level 1) - _FSFTID, escaped like any other
	// text so a pointer-shaped id is not written as an XRef pointer.
	if indi.FamilySearchID != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "_FSFTID", Value: opts.text(indi.FamilySearchID)})
	}

	return tags
}

// familyToTags converts a Family entity to GEDCOM tags.
//
//nolint:gocyclo // Converting all family fields requires handling many cases
func familyToTags(fam *gedcom.Family, opts *EncodeOptions) []*gedcom.Tag {
	if fam == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Restriction notice (level 1) - RESN, first as every version orders it
	if fam.Restriction != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "RESN", Value: opts.text(fam.Restriction)})
	}

	// Husband (level 1) - HUSB
	if fam.Husband != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "HUSB", Value: fam.Husband})
		tags = appendPhrase(tags, fam.HusbandPhrase, 2, opts)
	}

	// Wife (level 1) - WIFE
	if fam.Wife != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "WIFE", Value: fam.Wife})
		tags = appendPhrase(tags, fam.WifePhrase, 2, opts)
	}

	// Children (level 1) - CHIL, each with its _FREL/_MREL from ChildLinks
	tags = append(tags, childrenToTags(fam, opts)...)

	// Events (level 1) - MARR, DIV, etc.
	for _, event := range fam.Events {
		if event == nil {
			continue
		}
		tags = append(tags, withSpouseAges(eventToTags(event, 1, opts),
			event.HusbandAge, event.HusbandAgePhrase, event.WifeAge, event.WifeAgePhrase, opts)...)
	}

	// Attributes (level 1) - NCHI, FACT (CENS and RESI decode as Events), etc.
	for _, attr := range fam.Attributes {
		if attr == nil {
			continue
		}
		tags = append(tags, withSpouseAges(attributeToTags(attr, 1, opts),
			attr.HusbandAge, attr.HusbandAgePhrase, attr.WifeAge, attr.WifeAgePhrase, opts)...)
	}

	// LDS Ordinances (level 1) - SLGS
	for _, ord := range fam.LDSOrdinances {
		tags = append(tags, ldsOrdinanceToTags(ord, 1, opts)...)
	}

	// Source citations (level 1) - SOUR
	for _, cite := range fam.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, 1, opts)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(fam.NoteXRefs, fam.InlineNotes, 1, opts)...)

	// Media links (level 1) - OBJE
	for _, media := range fam.Media {
		tags = append(tags, mediaLinkToTags(media, 1, opts)...)
	}

	// Change date (level 1) - CHAN
	if fam.ChangeDate != nil {
		tags = append(tags, changeDateToTags(fam.ChangeDate, 1, "CHAN", opts)...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if fam.CreationDate != nil {
		tags = append(tags, changeDateToTags(fam.CreationDate, 1, "CREA", opts)...)
	}

	// Reference numbers (level 1) - REFN, each with its optional TYPE
	tags = append(tags, refNumbersToTags(fam.RefNumbers, opts)...)

	// UIDs (level 1)
	tags = append(tags, uidsToTags(fam.UIDs, 1, opts)...)

	// External IDs (level 1) - EXID (GEDCOM 7.0)
	tags = append(tags, externalIDsToTags(fam.ExternalIDs, 1, opts)...)

	return tags
}

// sourceToTags converts a Source entity to GEDCOM tags.
//
//nolint:gocyclo // Converting all source fields requires handling many cases
func sourceToTags(src *gedcom.Source, opts *EncodeOptions) []*gedcom.Tag {
	if src == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Data (level 1) - DATA, with the events recorded, agency and notes
	tags = append(tags, sourceDataToTags(src.Data, opts)...)

	// Title (level 1) - TITL
	if src.Title != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "TITL", Value: opts.text(src.Title)})
	}

	// Abbreviation (level 1) - ABBR
	if src.Abbreviation != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "ABBR", Value: opts.text(src.Abbreviation)})
	}

	// Author (level 1) - AUTH
	if src.Author != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "AUTH", Value: opts.text(src.Author)})
	}

	// Publication (level 1) - PUBL
	if src.Publication != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "PUBL", Value: opts.text(src.Publication)})
	}

	// Text (level 1) - TEXT (with CONT/CONC for multiline/long)
	if src.Text != "" {
		tags = append(tags, textToTags(src.Text, 1, "TEXT", opts)...)
		tags = appendTextQualifiers(tags, src.TextMIME, src.TextLanguage, 2, opts)
	}

	// Repository links (level 1) - REPO, one per link
	for _, link := range src.RepositoryLinks {
		tags = append(tags, sourceRepositoryLinkToTags(link, opts)...)
	}

	// Media links (level 1) - OBJE
	for _, media := range src.Media {
		tags = append(tags, mediaLinkToTags(media, 1, opts)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(src.NoteXRefs, src.InlineNotes, 1, opts)...)

	// Change date (level 1) - CHAN
	if src.ChangeDate != nil {
		tags = append(tags, changeDateToTags(src.ChangeDate, 1, "CHAN", opts)...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if src.CreationDate != nil {
		tags = append(tags, changeDateToTags(src.CreationDate, 1, "CREA", opts)...)
	}

	// Reference numbers (level 1) - REFN, each with its optional TYPE
	tags = append(tags, refNumbersToTags(src.RefNumbers, opts)...)

	// UIDs (level 1)
	tags = append(tags, uidsToTags(src.UIDs, 1, opts)...)

	// External IDs (level 1) - EXID (GEDCOM 7.0)
	tags = append(tags, externalIDsToTags(src.ExternalIDs, 1, opts)...)

	return tags
}

// sourceDataToTags converts a source record's DATA structure to GEDCOM tags at
// level 1: DATA, each EVEN with its DATE (and PHRASE) and PLAC, AGNC, and
// notes.
func sourceDataToTags(data *gedcom.SourceData, opts *EncodeOptions) []*gedcom.Tag {
	if data == nil {
		return nil
	}

	tags := []*gedcom.Tag{{Level: 1, Tag: "DATA"}}
	for _, event := range data.Events {
		if event == nil {
			continue
		}
		tags = append(tags, &gedcom.Tag{Level: 2, Tag: "EVEN", Value: opts.text(event.Types)})
		if event.Date != "" {
			tags = append(tags, &gedcom.Tag{Level: 3, Tag: "DATE", Value: opts.text(event.Date)})
			tags = appendPhrase(tags, event.DatePhrase, 4, opts)
		}
		if event.PlaceDetail != nil {
			tags = append(tags, placeToTags(event.PlaceDetail, 3, opts)...)
		}
	}
	if data.Agency != "" {
		tags = append(tags, &gedcom.Tag{Level: 2, Tag: "AGNC", Value: opts.text(data.Agency)})
	}
	tags = append(tags, notesToTags(data.NoteXRefs, data.InlineNotes, 2, opts)...)

	return tags
}

// appendTextQualifiers appends the GEDCOM 7.0 MIME and LANG qualifiers of a
// TEXT line at level, each only when set.
func appendTextQualifiers(tags []*gedcom.Tag, mime, lang string, level int, opts *EncodeOptions) []*gedcom.Tag {
	if mime != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "MIME", Value: opts.text(mime)})
	}
	if lang != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "LANG", Value: opts.text(lang)})
	}
	return tags
}

// repositoryLinkIsDegenerate reports whether a link carries neither pointer,
// inline name, nor any subordinate data. Encoding one would emit a meaningless
// bare `1 REPO` that round-trips as an empty inline repository.
func repositoryLinkIsDegenerate(link *gedcom.SourceRepositoryLink) bool {
	hasInlineName := link.Inline != nil && link.Inline.Name != ""
	hasNotes := len(link.NoteXRefs) > 0 || len(link.InlineNotes) > 0
	return link.XRef == "" && !hasInlineName && !hasCallNumber(link.CallNumbers) && !hasNotes
}

// hasCallNumber reports whether calns holds at least one non-nil entry, i.e.
// whether encoding the link would write any CALN line.
func hasCallNumber(calns []*gedcom.CallNumber) bool {
	for _, caln := range calns {
		if caln != nil {
			return true
		}
	}
	return false
}

// sourceRepositoryLinkToTags converts a SourceRepositoryLink to GEDCOM tags,
// emitting the REPO pointer (or inline NAME) plus CALN (each with its optional
// MEDI and MEDI.PHRASE) and NOTE subordinates.
func sourceRepositoryLinkToTags(link *gedcom.SourceRepositoryLink, opts *EncodeOptions) []*gedcom.Tag {
	if link == nil || repositoryLinkIsDegenerate(link) {
		return nil
	}

	var tags []*gedcom.Tag

	if link.XRef != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REPO", Value: link.XRef})
	} else {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REPO"})
		if link.Inline != nil && link.Inline.Name != "" {
			tags = append(tags, &gedcom.Tag{Level: 2, Tag: "NAME", Value: opts.text(link.Inline.Name)})
		}
	}

	// Call numbers (level 2) - CALN, each with an optional MEDI (level 3)
	// and, under MEDI, an optional PHRASE (level 4, GEDCOM 7.0).
	for _, caln := range link.CallNumbers {
		if caln == nil {
			continue
		}
		tags = append(tags, &gedcom.Tag{Level: 2, Tag: "CALN", Value: opts.text(caln.Value)})
		// A PHRASE has no other parent than MEDI, so a phrase set without a
		// media type still writes a valueless MEDI to carry it.
		if caln.MediaType != "" || caln.MediaPhrase != "" {
			tags = append(tags, &gedcom.Tag{Level: 3, Tag: "MEDI", Value: opts.text(caln.MediaType)})
		}
		if caln.MediaPhrase != "" {
			tags = append(tags, &gedcom.Tag{Level: 4, Tag: "PHRASE", Value: opts.text(caln.MediaPhrase)})
		}
	}

	// Per-link notes (level 2) - NOTE (with CONT/CONC for multiline/long).
	tags = append(tags, notesToTags(link.NoteXRefs, link.InlineNotes, 2, opts)...)

	return tags
}

// submitterToTags converts a Submitter entity to GEDCOM tags.
func submitterToTags(subm *gedcom.Submitter, opts *EncodeOptions) []*gedcom.Tag {
	if subm == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Name (level 1) - NAME
	if subm.Name != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "NAME", Value: opts.text(subm.Name)})
	}

	// Address (level 1) - ADDR
	if subm.Address != nil {
		tags = append(tags, addressToTags(subm.Address, 1, opts)...)
	}

	// Phone numbers (level 1) - PHON
	for _, phone := range subm.Phone {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "PHON", Value: opts.text(phone)})
	}

	// Email addresses (level 1) - EMAIL
	for _, email := range subm.Email {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "EMAIL", Value: opts.text(email)})
	}

	// Languages (level 1) - LANG
	for _, lang := range subm.Language {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "LANG", Value: opts.text(lang)})
	}

	// External IDs (level 1) - EXID (GEDCOM 7.0)
	// Emitted before NOTE to match GEDCOM 7 structure order (IDENTIFIER_STRUCTURE
	// precedes NOTE_STRUCTURE) and the placement used by the other record writers.
	tags = append(tags, externalIDsToTags(subm.ExternalIDs, 1, opts)...)

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(subm.NoteXRefs, subm.InlineNotes, 1, opts)...)

	return tags
}

// submissionToTags converts a Submission entity to GEDCOM tags, in the
// SUBMISSION_RECORD order of GEDCOM 5.5.1.
func submissionToTags(subn *gedcom.Submission, opts *EncodeOptions) []*gedcom.Tag {
	if subn == nil {
		return nil
	}

	var tags []*gedcom.Tag
	// SUBM is a pointer, written verbatim; the rest are text.
	if subn.SubmitterXRef != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "SUBM", Value: subn.SubmitterXRef})
	}
	scalars := []struct{ tag, value string }{
		{"FAMF", subn.FamilyFileName},
		{"TEMP", subn.TempleCode},
		{"ANCE", subn.AncestorGenerations},
		{"DESC", subn.DescendantGenerations},
		{"ORDI", subn.OrdinanceProcessFlag},
		{"RIN", subn.AutomatedRecordID},
	}
	for _, s := range scalars {
		if s.value != "" {
			tags = append(tags, &gedcom.Tag{Level: 1, Tag: s.tag, Value: opts.text(s.value)})
		}
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(subn.NoteXRefs, subn.InlineNotes, 1, opts)...)

	return append(tags, changeDateToTags(subn.ChangeDate, 1, "CHAN", opts)...)
}

// repositoryToTags converts a Repository entity to GEDCOM tags.
func repositoryToTags(repo *gedcom.Repository, opts *EncodeOptions) []*gedcom.Tag {
	if repo == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Name (level 1) - NAME
	if repo.Name != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "NAME", Value: opts.text(repo.Name)})
	}

	// Address (level 1) - ADDR
	if repo.Address != nil {
		tags = append(tags, addressToTags(repo.Address, 1, opts)...)
	}

	// Contact details (level 1) - PHON, EMAIL, FAX, WWW, in ADDRESS_STRUCTURE
	// grammar order after ADDR.
	tags = append(tags, contactTags(repo.Phone, repo.Email, repo.Fax, repo.Website, 1, opts)...)

	// External IDs (level 1) - EXID (GEDCOM 7.0)
	// Emitted before NOTE to match GEDCOM 7 structure order (IDENTIFIER_STRUCTURE
	// precedes NOTE_STRUCTURE) and the placement used by the other record writers.
	tags = append(tags, externalIDsToTags(repo.ExternalIDs, 1, opts)...)

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(repo.NoteXRefs, repo.InlineNotes, 1, opts)...)

	return tags
}

// mediaObjectToTags converts a MediaObject entity to GEDCOM tags.
func mediaObjectToTags(media *gedcom.MediaObject, opts *EncodeOptions) []*gedcom.Tag {
	if media == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Files (level 1) - FILE
	for _, file := range media.Files {
		tags = append(tags, mediaFileToTags(file, 1, opts)...)
	}

	// Notes (level 1) - NOTE (with CONT/CONC for multiline/long).
	// NoteXRefs and SharedNoteXRefs partition this record's note pointers
	// (#499), so they concatenate without deduping. Both are written as NOTE
	// here; writeRecord retags a 7.0 pointer to an SNOTE record as SNOTE (#471).
	noteXRefs := make([]string, 0, len(media.NoteXRefs)+len(media.SharedNoteXRefs))
	noteXRefs = append(noteXRefs, media.NoteXRefs...)
	noteXRefs = append(noteXRefs, media.SharedNoteXRefs...)
	tags = append(tags, notesToTags(noteXRefs, media.InlineNotes, 1, opts)...)

	// Source citations (level 1) - SOUR
	for _, cite := range media.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, 1, opts)...)
	}

	// Change date (level 1) - CHAN
	if media.ChangeDate != nil {
		tags = append(tags, changeDateToTags(media.ChangeDate, 1, "CHAN", opts)...)
	}

	// Creation date (level 1) - CREA (GEDCOM 7.0)
	if media.CreationDate != nil {
		tags = append(tags, changeDateToTags(media.CreationDate, 1, "CREA", opts)...)
	}

	// Reference numbers (level 1) - REFN, each with its optional TYPE
	tags = append(tags, refNumbersToTags(media.RefNumbers, opts)...)

	// UIDs (level 1)
	tags = append(tags, uidsToTags(media.UIDs, 1, opts)...)

	// External IDs (level 1) - EXID (GEDCOM 7.0)
	tags = append(tags, externalIDsToTags(media.ExternalIDs, 1, opts)...)

	// Restriction (level 1) - RESN
	if media.Restriction != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "RESN", Value: opts.text(media.Restriction)})
	}

	return tags
}

// childrenToTags converts a family's Children to CHIL tags. Each ChildLink
// is written beneath the first CHIL of its XRef not already given one, as
// _FREL and _MREL; a link whose XRef is not among Children is not written.
func childrenToTags(fam *gedcom.Family, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag
	// pending queues each child's links in order, so matching is linear in
	// the family size rather than children x links.
	var pending map[string][]gedcom.ChildLink
	if len(fam.ChildLinks) > 0 {
		pending = make(map[string][]gedcom.ChildLink, len(fam.ChildLinks))
		for _, link := range fam.ChildLinks {
			pending[link.XRef] = append(pending[link.XRef], link)
		}
	}
	for _, child := range fam.Children {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "CHIL", Value: child})
		queue := pending[child]
		if len(queue) == 0 {
			continue
		}
		link := queue[0]
		pending[child] = queue[1:]
		if link.FatherRelation != "" {
			tags = append(tags, &gedcom.Tag{Level: 2, Tag: "_FREL", Value: opts.text(link.FatherRelation)})
		}
		if link.MotherRelation != "" {
			tags = append(tags, &gedcom.Tag{Level: 2, Tag: "_MREL", Value: opts.text(link.MotherRelation)})
		}
	}
	return tags
}

// nameToTags converts a PersonalName to GEDCOM tags at the specified level.
func nameToTags(name *gedcom.PersonalName, level int, opts *EncodeOptions) []*gedcom.Tag {
	if name == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// NAME tag with full name value
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "NAME", Value: opts.text(name.Full)})

	// Subordinate tags at level+1
	if name.Given != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "GIVN", Value: opts.text(name.Given)})
	}
	if name.Surname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SURN", Value: opts.text(name.Surname)})
	}
	if name.Prefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NPFX", Value: opts.text(name.Prefix)})
	}
	if name.Suffix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NSFX", Value: opts.text(name.Suffix)})
	}
	if name.Nickname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NICK", Value: opts.text(name.Nickname)})
	}
	if name.SurnamePrefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SPFX", Value: opts.text(name.SurnamePrefix)})
	}
	if name.MarriedName != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "_MARNM", Value: opts.text(name.MarriedName)})
	}
	if name.Type != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TYPE", Value: opts.text(name.Type)})
		tags = appendPhrase(tags, name.TypePhrase, level+2, opts)
	}

	tags = append(tags, nameVariationsToTags(name, level+1, opts)...)

	// Source citations
	for _, cite := range name.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, level+1, opts)...)
	}

	// Notes (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(name.NoteXRefs, name.InlineNotes, level+1, opts)...)

	return tags
}

// nameVariationsToTags converts a name's variations to GEDCOM tags at level.
// Variations are version-specific (issue #692), so each is written only for
// the output version that defines it and dropped otherwise: TRAN only for
// 7.0 output, FONE and ROMN only for 5.5.1 output (5.5 defines neither).
// When the output version is unknown (no TargetVersion and no header
// version) nothing is dropped, so such output is unchanged.
func nameVariationsToTags(name *gedcom.PersonalName, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	// Transliterations (GEDCOM 7.0 TRAN tag)
	if !opts.outputLacks(gedcom.Version70) {
		for _, tran := range name.Transliterations {
			tags = append(tags, transliterationToTags(tran, level, opts)...)
		}
	}
	if opts.outputLacks(gedcom.Version551) {
		return tags
	}

	// Phonetic and romanized variations (GEDCOM 5.5.1 FONE and ROMN tags)
	for _, v := range name.Phonetic {
		tags = append(tags, nameVariationToTags(v, "FONE", level, opts)...)
	}
	for _, v := range name.Romanized {
		tags = append(tags, nameVariationToTags(v, "ROMN", level, opts)...)
	}
	return tags
}

// transliterationToTags converts a Transliteration to GEDCOM tags at the specified level.
func transliterationToTags(tran *gedcom.Transliteration, level int, opts *EncodeOptions) []*gedcom.Tag {
	if tran == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// TRAN tag with transliterated name value
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "TRAN", Value: opts.text(tran.Value)})

	// Subordinate tags at level+1
	if tran.Language != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LANG", Value: opts.text(tran.Language)})
	}
	if tran.Given != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "GIVN", Value: opts.text(tran.Given)})
	}
	if tran.Surname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SURN", Value: opts.text(tran.Surname)})
	}
	if tran.Prefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NPFX", Value: opts.text(tran.Prefix)})
	}
	if tran.Suffix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NSFX", Value: opts.text(tran.Suffix)})
	}
	if tran.Nickname != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "NICK", Value: opts.text(tran.Nickname)})
	}
	if tran.SurnamePrefix != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "SPFX", Value: opts.text(tran.SurnamePrefix)})
	}

	return tags
}

// nameVariationToTags converts a NameVariation to a FONE or ROMN tag (tagName)
// and its subordinates at the specified level.
func nameVariationToTags(v *gedcom.NameVariation, tagName string, level int, opts *EncodeOptions) []*gedcom.Tag {
	if v == nil {
		return nil
	}

	tags := []*gedcom.Tag{{Level: level, Tag: tagName, Value: opts.text(v.Value)}}

	for _, piece := range []struct{ tag, value string }{
		{"TYPE", v.Type},
		{"NPFX", v.Prefix},
		{"GIVN", v.Given},
		{"NICK", v.Nickname},
		{"SPFX", v.SurnamePrefix},
		{"SURN", v.Surname},
		{"NSFX", v.Suffix},
	} {
		if piece.value != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: piece.tag, Value: opts.text(piece.value)})
		}
	}

	return tags
}

// eventDetail is the EVENT_DETAIL substructure set that gedcom.Event and
// gedcom.Attribute hold under identically named fields. Both writers fill one
// in and hand it to eventDetailToTags so the shared block has a single writer
// instead of two copies that drift apart. The spouse ages of
// FAMILY_EVENT_DETAIL are not part of it: see spouseAgesToTags.
type eventDetail struct {
	date                 string
	parsedDate           *gedcom.Date
	placeDetail          *gedcom.PlaceDetail
	typeDetail           string
	cause                string
	age                  string
	agePhrase            string
	agency               string
	religiousAffiliation string
	address              *gedcom.Address
	phone                []string
	email                []string
	fax                  []string
	website              []string
	restriction          string
	uids                 []string
	sortDate             string
	sortDatePhrase       string
	associations         []*gedcom.Association
	noteXRefs            []string
	inlineNotes          []string
	sourceCitations      []*gedcom.SourceCitation
	media                []*gedcom.MediaLink
}

// eventDetailToTags converts the EVENT_DETAIL substructures shared by events and
// attributes to GEDCOM tags. level is the level of the subordinate lines
// themselves; the owning event or attribute line sits one level above.
//
//nolint:gocyclo // One guarded append per EVENT_DETAIL substructure
func eventDetailToTags(d *eventDetail, level int, opts *EncodeOptions) []*gedcom.Tag {
	tags := dateToTags(d.date, d.parsedDate, level, opts)

	// Place. A non-nil PlaceDetail is enough, even with an empty Name: the
	// decoder parses a bare "2 PLAC" line to a non-nil PlaceDetail whose Name
	// is "", so gating on Name would drop the line and any MAP children with
	// it.
	if d.placeDetail != nil {
		tags = append(tags, placeToTags(d.placeDetail, level, opts)...)
	}

	if d.typeDetail != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "TYPE", Value: opts.text(d.typeDetail)})
	}

	if d.cause != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "CAUS", Value: opts.text(d.cause)})
	}

	tags = append(tags, ageToTags(d.age, d.agePhrase, level, opts)...)

	if d.agency != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "AGNC", Value: opts.text(d.agency)})
	}

	if d.religiousAffiliation != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "RELI", Value: opts.text(d.religiousAffiliation)})
	}

	// Address
	if d.address != nil {
		tags = append(tags, addressToTags(d.address, level, opts)...)
	}

	// Contact info
	tags = append(tags, contactTags(d.phone, d.email, d.fax, d.website, level, opts)...)

	if d.restriction != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "RESN", Value: opts.text(d.restriction)})
	}

	tags = append(tags, uidsToTags(d.uids, level, opts)...)

	if d.sortDate != "" {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "SDATE", Value: opts.text(d.sortDate)})
		tags = appendPhrase(tags, d.sortDatePhrase, level+1, opts)
	}

	// Associations - ASSO (witnesses, officiants, godparents)
	for _, assoc := range d.associations {
		tags = append(tags, associationToTags(assoc, level, opts)...)
	}

	// Notes - NOTE (XRef pointers and inline text, with CONT/CONC for
	// multiline/long). Pointers are written as "NOTE @Nn@" exactly as the
	// record-level writers do; see notesToTags.
	tags = append(tags, notesToTags(d.noteXRefs, d.inlineNotes, level, opts)...)

	// Source citations
	for _, cite := range d.sourceCitations {
		tags = append(tags, sourceCitationToTags(cite, level, opts)...)
	}

	// Media links
	for _, media := range d.media {
		tags = append(tags, mediaLinkToTags(media, level, opts)...)
	}

	return tags
}

// eventToTags converts an Event to GEDCOM tags at the specified level.
func eventToTags(event *gedcom.Event, level int, opts *EncodeOptions) []*gedcom.Tag {
	if event == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Event tag - for negative assertions (GEDCOM 7.0), use NO tag with event type as value
	if event.IsNegative {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "NO", Value: string(event.Type)})
	} else {
		// Description is the event line's own payload (the descriptor on a
		// generic EVEN, any exporter text on a named event), written with
		// CONT/CONC lines as the output version allows (#734).
		tags = append(tags, textToTags(event.Description, level, string(event.Type), opts)...)
	}

	// Subordinate tags at level+1
	tags = append(tags, eventDetailToTags(&eventDetail{
		date:                 event.Date,
		parsedDate:           event.ParsedDate,
		placeDetail:          event.PlaceDetail,
		typeDetail:           event.EventTypeDetail,
		cause:                event.Cause,
		age:                  event.Age,
		agePhrase:            event.AgePhrase,
		agency:               event.Agency,
		religiousAffiliation: event.ReligiousAffiliation,
		address:              event.Address,
		phone:                event.Phone,
		email:                event.Email,
		fax:                  event.Fax,
		website:              event.Website,
		restriction:          event.Restriction,
		uids:                 event.UIDs,
		sortDate:             event.SortDate,
		sortDatePhrase:       event.SortDatePhrase,
		associations:         event.Associations,
		noteXRefs:            event.NoteXRefs,
		inlineNotes:          event.InlineNotes,
		sourceCitations:      event.SourceCitations,
		media:                event.Media,
	}, level+1, opts)...)

	// FAMC, with ADOP and its PHRASE beneath it (issue #740): written only
	// where every version defines it -- FAMC on a BIRT, CHR or ADOP event,
	// and ADOP only beneath an ADOP event's FAMC -- so a hand-built value set
	// on any other event is not written as a structure no version allows.
	if event.FamilyXRef != "" && eventTakesFamilyLink(event) {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FAMC", Value: event.FamilyXRef})
		if event.AdoptedBy != "" && event.Type == gedcom.EventAdoption {
			tags = append(tags, &gedcom.Tag{Level: level + 2, Tag: "ADOP", Value: opts.text(event.AdoptedBy)})
			tags = appendPhrase(tags, event.AdoptedByPhrase, level+3, opts)
		}
	}

	return tags
}

// withSpouseAges inserts the spouse ages of FAMILY_EVENT_DETAIL (HUSB and
// WIFE, each with its AGE and 7.0 PHRASE) directly beneath the event or
// attribute line that heads tags, where the grammar lists them. Only a
// family's events and attributes take them, so only familyToTags calls it: a
// spouse age set on an individual's event is not written as a structure no
// version allows (the decoder reports one there as UNKNOWN_TAG).
func withSpouseAges(tags []*gedcom.Tag, husbandAge, husbandPhrase, wifeAge, wifePhrase string, opts *EncodeOptions) []*gedcom.Tag {
	if len(tags) == 0 {
		return tags
	}
	level := tags[0].Level + 1
	var ages []*gedcom.Tag
	for _, spouse := range []struct{ tag, age, phrase string }{
		{"HUSB", husbandAge, husbandPhrase},
		{"WIFE", wifeAge, wifePhrase},
	} {
		age := ageToTags(spouse.age, spouse.phrase, level+1, opts)
		if age == nil {
			continue
		}
		ages = append(ages, &gedcom.Tag{Level: level, Tag: spouse.tag})
		ages = append(ages, age...)
	}
	return slices.Insert(tags, 1, ages...)
}

// eventTakesFamilyLink reports whether event may carry a FAMC: an individual's
// BIRT, CHR or ADOP that occurred (a 7.0 NO assertion has no substructure but
// DATE, NOTE and SOUR). It mirrors the decoder's parseEventFamilyLink.
func eventTakesFamilyLink(event *gedcom.Event) bool {
	if event.IsNegative {
		return false
	}
	switch event.Type {
	case gedcom.EventBirth, gedcom.EventChristening, gedcom.EventAdoption:
		return true
	}
	return false
}

// attributeToTags converts an Attribute to GEDCOM tags at the specified level.
func attributeToTags(attr *gedcom.Attribute, level int, opts *EncodeOptions) []*gedcom.Tag {
	if attr == nil {
		return nil
	}

	// Attribute tag (OCCU, EDUC, etc.) with value, continued on CONT/CONC
	// lines as the output version allows (#734).
	tags := textToTags(attr.Value, level, string(attr.Type), opts)

	// Subordinate tags at level+1
	return append(tags, eventDetailToTags(&eventDetail{
		date:                 attr.Date,
		parsedDate:           attr.ParsedDate,
		placeDetail:          attr.PlaceDetail,
		typeDetail:           attr.TypeDetail,
		cause:                attr.Cause,
		age:                  attr.Age,
		agePhrase:            attr.AgePhrase,
		agency:               attr.Agency,
		religiousAffiliation: attr.ReligiousAffiliation,
		address:              attr.Address,
		phone:                attr.Phone,
		email:                attr.Email,
		fax:                  attr.Fax,
		website:              attr.Website,
		restriction:          attr.Restriction,
		uids:                 attr.UIDs,
		sortDate:             attr.SortDate,
		sortDatePhrase:       attr.SortDatePhrase,
		associations:         attr.Associations,
		noteXRefs:            attr.NoteXRefs,
		inlineNotes:          attr.InlineNotes,
		sourceCitations:      attr.SourceCitations,
		media:                attr.Media,
	}, level+1, opts)...)
}

// sourceCitationToTags converts a SourceCitation to GEDCOM tags at the specified level.
func sourceCitationToTags(cite *gedcom.SourceCitation, level int, opts *EncodeOptions) []*gedcom.Tag {
	if cite == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// SOUR tag with source XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "SOUR", Value: cite.SourceXRef})

	// Subordinate tags at level+1
	if cite.Page != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PAGE", Value: opts.text(cite.Page)})
	}

	if cite.Quality != nil {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "QUAY", Value: strconv.Itoa(*cite.Quality)})
	}

	// DATA subordinate
	if cite.Data != nil {
		tags = append(tags, sourceCitationDataToTags(cite.Data, level+1, opts)...)
	}

	// Event cited from - EVEN, with its PHRASE and ROLE (and ROLE.PHRASE)
	if event := cite.Event; event != nil {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "EVEN", Value: opts.text(event.Type)})
		tags = appendPhrase(tags, event.Phrase, level+2, opts)
		if event.Role != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 2, Tag: "ROLE", Value: opts.text(event.Role)})
			tags = appendPhrase(tags, event.RolePhrase, level+3, opts)
		}
	}

	// Media links - OBJE
	for _, media := range cite.Media {
		tags = append(tags, mediaLinkToTags(media, level+1, opts)...)
	}

	// Notes - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(cite.NoteXRefs, cite.InlineNotes, level+1, opts)...)

	// Ancestry APID (vendor extension)
	if cite.AncestryAPID != nil {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "_APID", Value: opts.text(cite.AncestryAPID.Raw)})
	}

	return tags
}

// sourceCitationDataToTags converts SourceCitationData to GEDCOM tags at the specified level.
func sourceCitationDataToTags(data *gedcom.SourceCitationData, level int, opts *EncodeOptions) []*gedcom.Tag {
	if data == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// DATA tag
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "DATA"})

	// Subordinate tags at level+1
	if data.Date != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: opts.text(data.Date)})
	}

	// Text: every entry, in order. Each Value goes through textToTags so a
	// multi-line passage splits into TEXT + CONT/CONC rather than writing an
	// embedded newline onto the TEXT line (which would forge GEDCOM lines).
	for _, text := range data.Text {
		if text == nil {
			continue
		}
		tags = append(tags, textToTags(text.Value, level+1, "TEXT", opts)...)
		tags = appendTextQualifiers(tags, text.MIME, text.Language, level+2, opts)
	}

	return tags
}

// contactTags writes the ADDRESS_STRUCTURE contact siblings of ADDR -- PHON,
// EMAIL, FAX and WWW -- at the given level, in grammar order, one line per
// entry.
func contactTags(phone, email, fax, website []string, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag
	for _, v := range phone {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "PHON", Value: opts.text(v)})
	}
	for _, v := range email {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "EMAIL", Value: opts.text(v)})
	}
	for _, v := range fax {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "FAX", Value: opts.text(v)})
	}
	for _, v := range website {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "WWW", Value: opts.text(v)})
	}
	return tags
}

// addressToTags converts an Address to GEDCOM tags at the specified level.
//
// Each tag is written from its own field: the ADDR value and its CONT/CONC
// lines from Full, ADR1-ADR3 from Line1-Line3. The grammar gives ADR1-ADR3 no
// CONT, so a line break in Line1-Line3 is written as a space rather than
// folded into a CONT the line cannot carry.
func addressToTags(addr *gedcom.Address, level int, opts *EncodeOptions) []*gedcom.Tag {
	if addr == nil {
		return nil
	}

	tags := textToTags(addr.Full, level, "ADDR", opts)

	// Subordinate tags at level+1
	for _, line := range []struct{ tag, value string }{
		{"ADR1", addr.Line1}, {"ADR2", addr.Line2}, {"ADR3", addr.Line3},
	} {
		if line.value != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: line.tag, Value: opts.text(singleLine(line.value))})
		}
	}
	if addr.City != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CITY", Value: opts.text(addr.City)})
	}
	if addr.State != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "STAE", Value: opts.text(addr.State)})
	}
	if addr.PostalCode != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "POST", Value: opts.text(addr.PostalCode)})
	}
	if addr.Country != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "CTRY", Value: opts.text(addr.Country)})
	}

	return tags
}

// singleLine joins the lines of value with a space, for a tag whose grammar
// allows no CONT. It deliberately does not split an over-long value: the
// grammar gives ADR1-ADR3 no CONC either, and writeValue leaves long lines
// alone, so such a value is written as one over-length line, which still
// parses.
func singleLine(value string) string {
	return strings.Join(splitValueLines(value), " ")
}

// placeToTags converts a place structure to GEDCOM tags at the specified level.
//
// The PLAC value is detail.Name, which may be empty: an empty Name writes a
// valueless PLAC line carrying only its subordinates, which is what a bare
// "2 PLAC" line decodes to. A nil detail means no place was recorded and
// writes nothing.
func placeToTags(detail *gedcom.PlaceDetail, level int, opts *EncodeOptions) []*gedcom.Tag {
	if detail == nil {
		return nil
	}

	// PLAC tag with place name
	tags := []*gedcom.Tag{{Level: level, Tag: "PLAC", Value: opts.text(detail.Name)}}

	if detail.Form != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FORM", Value: opts.text(detail.Form)})
	}

	// Coordinates via MAP
	if detail.Coordinates != nil {
		tags = append(tags, coordinatesToTags(detail.Coordinates, level+1, opts)...)
	}

	// Notes - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(detail.NoteXRefs, detail.InlineNotes, level+1, opts)...)

	return tags
}

// coordinatesToTags converts Coordinates to GEDCOM tags at the specified level.
func coordinatesToTags(coords *gedcom.Coordinates, level int, opts *EncodeOptions) []*gedcom.Tag {
	if coords == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// MAP tag
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "MAP"})

	// Subordinate tags at level+1
	if coords.Latitude != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LATI", Value: opts.text(coords.Latitude)})
	}
	if coords.Longitude != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LONG", Value: opts.text(coords.Longitude)})
	}

	return tags
}

// ldsOrdinanceToTags converts an LDSOrdinance to GEDCOM tags at the specified level.
func ldsOrdinanceToTags(ord *gedcom.LDSOrdinance, level int, opts *EncodeOptions) []*gedcom.Tag {
	if ord == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Ordinance tag (BAPL, CONL, ENDL, SLGC, SLGS)
	tags = append(tags, &gedcom.Tag{Level: level, Tag: string(ord.Type)})

	// Subordinate tags at level+1
	tags = append(tags, dateToTags(ord.Date, ord.ParsedDate, level+1, opts)...)

	if ord.Temple != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TEMP", Value: opts.text(ord.Temple)})
	}

	if ord.Place != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PLAC", Value: opts.text(ord.Place)})
	}

	tags = append(tags, ldsStatusToTags(ord, level+1, opts)...)

	// FAMC for SLGC (sealing to parents)
	if ord.FamilyXRef != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FAMC", Value: ord.FamilyXRef})
	}

	// Notes - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(ord.NoteXRefs, ord.InlineNotes, level+1, opts)...)

	return tags
}

// ldsStatusToTags writes an ordinance's STAT at level (issue #740), with its
// DATE (StatusDate) beneath it except for 5.5 output, which defines no
// STAT.DATE. TIME beneath the DATE is 7.0 only. A Status with no StatusDate
// is written alone for every version: 5.5.1 and 7.0 require the DATE, but
// dropping the status would lose data the caller set, so supplying the date
// is the caller's part. An unknown output version drops nothing.
func ldsStatusToTags(ord *gedcom.LDSOrdinance, level int, opts *EncodeOptions) []*gedcom.Tag {
	if ord.Status == "" {
		return nil
	}
	tags := []*gedcom.Tag{{Level: level, Tag: "STAT", Value: opts.text(ord.Status)}}
	if ord.StatusDate == "" || opts.outputIs(gedcom.Version55) {
		return tags
	}
	tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: opts.text(ord.StatusDate)})
	if ord.StatusTime != "" && !opts.outputLacks(gedcom.Version70) {
		tags = append(tags, &gedcom.Tag{Level: level + 2, Tag: "TIME", Value: opts.text(ord.StatusTime)})
	}
	return tags
}

// familyLinkToTags converts a FamilyLink to GEDCOM tags at the specified level.
// tagName is the link's own tag: "FAMC" for a child link, "FAMS" for a spouse link.
func familyLinkToTags(link *gedcom.FamilyLink, tagName string, level int, opts *EncodeOptions) []*gedcom.Tag {
	if link == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// Link tag with family XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: tagName, Value: link.FamilyXRef})

	// Subordinate tags at level+1
	if link.Pedigree != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PEDI", Value: opts.text(link.Pedigree)})
		tags = appendPhrase(tags, link.PedigreePhrase, level+2, opts)
	}

	// Notes - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(link.NoteXRefs, link.InlineNotes, level+1, opts)...)

	return tags
}

// associationToTags converts an Association to GEDCOM tags at the specified level.
func associationToTags(assoc *gedcom.Association, level int, opts *EncodeOptions) []*gedcom.Tag {
	if assoc == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// ASSO tag with individual XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "ASSO", Value: assoc.IndividualXRef})

	// Subordinate tags at level+1
	// PHRASE (GEDCOM 7.0) - human-readable description of the association
	if assoc.Phrase != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "PHRASE", Value: opts.text(assoc.Phrase)})
	}

	// TYPE is 5.5 only, where it is required: written there (INDI when
	// unset), dropped for any other known output version (issue #740).
	if !opts.outputLacks(gedcom.Version55) {
		recordType := assoc.RecordType
		if recordType == "" && opts.outputIs(gedcom.Version55) {
			recordType = "INDI"
		}
		if recordType != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TYPE", Value: opts.text(recordType)})
		}
	}

	// The role is RELA in 5.5 and 5.5.1 and ROLE in 7.0 (issue #740). An
	// unknown output version keeps ROLE, as before.
	if assoc.Role != "" {
		roleTag := "ROLE"
		if opts.outputIs(gedcom.Version55) || opts.outputIs(gedcom.Version551) {
			roleTag = "RELA"
		}
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: roleTag, Value: opts.text(assoc.Role)})
		tags = appendPhrase(tags, assoc.RolePhrase, level+2, opts)
	}

	// Source citations (GEDCOM 7.0)
	for _, cite := range assoc.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, level+1, opts)...)
	}

	// Notes (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(assoc.NoteXRefs, assoc.InlineNotes, level+1, opts)...)

	return tags
}

// changeDateToTags converts a ChangeDate to GEDCOM tags at the specified level.
func changeDateToTags(cd *gedcom.ChangeDate, level int, tagName string, opts *EncodeOptions) []*gedcom.Tag {
	if cd == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// CHAN or CREA tag
	tags = append(tags, &gedcom.Tag{Level: level, Tag: tagName})

	// Subordinate DATE at level+1
	if cd.Date != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "DATE", Value: opts.text(cd.Date)})

		// TIME subordinate at level+2
		if cd.Time != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 2, Tag: "TIME", Value: opts.text(cd.Time)})
		}
	}

	// Notes - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(cd.NoteXRefs, cd.InlineNotes, level+1, opts)...)

	return tags
}

// mediaLinkToTags converts a MediaLink to GEDCOM tags at the specified level.
//
// An inline link (Files and no MediaXRef, issue #733) is written in the layout
// of the output version: for 5.5 FORM beside FILE, the first file's FORM only,
// as 5.5 has no MEDI and no FILE.TITL; for 7.0, which has no inline link, as
// a pointer to an OBJE record holding the files (see inlineMediaSet); and
// otherwise in the 5.5.1 layout, FILE with FORM and MEDI beneath it.
func mediaLinkToTags(link *gedcom.MediaLink, level int, opts *EncodeOptions) []*gedcom.Tag {
	if link == nil {
		return nil
	}

	if mediaLinkIsDegenerate(link) {
		return nil
	}

	xref := link.MediaXRef
	inline := xref == "" && len(link.Files) > 0
	if inline && opts.is70() && opts.inlineMedia != nil {
		xref, inline = opts.inlineMedia.promote(link), false
	}

	var tags []*gedcom.Tag

	// OBJE tag with media XRef
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "OBJE", Value: xref})

	if inline {
		tags = append(tags, inlineMediaFilesToTags(link.Files, level+1, opts)...)
	}

	// Subordinate tags at level+1
	if link.Crop != nil {
		tags = append(tags, cropRegionToTags(link.Crop, level+1)...)
	}

	if link.Title != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TITL", Value: opts.text(link.Title)})
	}

	// Notes - NOTE (with CONT/CONC for multiline/long)
	tags = append(tags, notesToTags(link.NoteXRefs, link.InlineNotes, level+1, opts)...)

	return tags
}

// inlineMediaFilesToTags writes an inline link's files at level, in the 5.5
// layout for 5.5 output and the 5.5.1 layout otherwise (see mediaLinkToTags).
func inlineMediaFilesToTags(files []*gedcom.MediaFile, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag
	if !opts.outputIs(gedcom.Version55) {
		for _, file := range files {
			tags = append(tags, mediaFileToTags(file, level, opts)...)
		}
		return tags
	}
	for _, file := range files {
		if file != nil && file.Form != "" {
			tags = append(tags, &gedcom.Tag{Level: level, Tag: "FORM", Value: opts.text(file.Form)})
			break
		}
	}
	for _, file := range files {
		if file != nil {
			tags = append(tags, &gedcom.Tag{Level: level, Tag: "FILE", Value: opts.text(file.FileRef)})
		}
	}
	return tags
}

// mediaLinkIsDegenerate reports whether a link carries neither a pointer nor
// any subordinate data (issue #740). Encoding one would emit a bare `1 OBJE`
// that carries nothing, so it is skipped, as repositoryLinkIsDegenerate skips
// a bare REPO.
//
// A link with no pointer and no Files but a title, crop or note is still
// written, without the FILE every 5.x version requires: skipping it would drop
// the typed data it does carry.
func mediaLinkIsDegenerate(link *gedcom.MediaLink) bool {
	hasNotes := len(link.NoteXRefs) > 0 || len(link.InlineNotes) > 0
	return link.MediaXRef == "" && len(link.Files) == 0 && link.Title == "" && link.Crop == nil && !hasNotes
}

// cropRegionToTags converts a CropRegion to GEDCOM tags at the specified level.
func cropRegionToTags(crop *gedcom.CropRegion, level int) []*gedcom.Tag {
	if crop == nil {
		return nil
	}

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
func mediaFileToTags(file *gedcom.MediaFile, level int, opts *EncodeOptions) []*gedcom.Tag {
	if file == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// FILE tag with file reference
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "FILE", Value: opts.text(file.FileRef)})

	// FORM subordinate at level+1
	if file.Form != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FORM", Value: opts.text(file.Form)})

		// The media type at level+2: TYPE in a 5.5/5.5.1 multimedia record,
		// MEDI in 7.0.
		if file.MediaType != "" {
			mediaTag := "MEDI"
			if opts.outputIs(gedcom.Version55) || opts.outputIs(gedcom.Version551) {
				mediaTag = "TYPE"
			}
			tags = append(tags, &gedcom.Tag{Level: level + 2, Tag: mediaTag, Value: opts.text(file.MediaType)})
			tags = appendPhrase(tags, file.MediaPhrase, level+3, opts)
		}
	}

	// TITL subordinate at level+1
	if file.Title != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TITL", Value: opts.text(file.Title)})
	}

	// Translations (TRAN)
	for _, tran := range file.Translations {
		tags = append(tags, mediaTranslationToTags(tran, level+1, opts)...)
	}

	return tags
}

// mediaTranslationToTags converts a MediaTranslation to GEDCOM tags at the specified level.
func mediaTranslationToTags(tran *gedcom.MediaTranslation, level int, opts *EncodeOptions) []*gedcom.Tag {
	if tran == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// TRAN tag with file reference
	tags = append(tags, &gedcom.Tag{Level: level, Tag: "TRAN", Value: opts.text(tran.FileRef)})

	// FORM subordinate at level+1
	if tran.Form != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "FORM", Value: opts.text(tran.Form)})
	}

	return tags
}

// entityRecordText returns the level-0 line value and any leading CONT/CONC
// continuation tags for a record whose entity carries text on the level-0 line —
// GEDCOM 7.0 SharedNote (SNOTE) and NOTE. Consulted only for records written
// from their entity (no raw Tags), whatever record.Value holds (#707); returns
// ("", nil) for any other record type or an empty Text.
//
// Both types run Text through textToTags: the first segment is the level-0 value
// and the remainder are leading CONT/CONC tags. This splits on embedded newlines
// and enforces MaxLineLength — critical for correctness, since writing Text
// verbatim onto the level-0 line would let an embedded newline forge additional
// GEDCOM records.
//
// For NOTE this covers the Text field, which since v3 is the only carrier of a
// note's body: entityRecordText is the single CONT/CONC producer for notes, so
// there is no second path that could emit the body twice.
//
// A Text that record.Value already spells is written as record.Value, verbatim
// (see wireSpells): a decoded one-line note has no Tags, so it is written from
// its entity, and re-escaping its text would rewrite a source "@" the producer
// did not double, which a byte round trip keeps (#708).
func entityRecordText(record *gedcom.Record, opts *EncodeOptions) (string, []*gedcom.Tag) {
	var text, tagName string
	switch record.Type {
	case gedcom.RecordTypeSharedNote:
		if snote, ok := record.Entity.(*gedcom.SharedNote); ok && snote != nil {
			text, tagName = snote.Text, "SNOTE"
		}
	case gedcom.RecordTypeNote:
		if note, ok := record.Entity.(*gedcom.Note); ok && note != nil {
			text, tagName = note.Text, "NOTE"
		}
	}
	if text == "" {
		return "", nil
	}
	if opts.wireSpells(record.Value, text) {
		return record.Value, nil
	}
	tags := textToTags(text, 0, tagName, opts)
	return tags[0].Value, tags[1:]
}

// sharedNoteToTags converts a SharedNote entity to GEDCOM tags.
// GEDCOM 7.0 SNOTE records include MIME types, language tags, and translations.
// The note's Text (level-0 value plus CONT/CONC continuation) is handled by
// writeRecord via entityRecordText, and its own raw Tags by entityOwnTags,
// not here.
func sharedNoteToTags(note *gedcom.SharedNote, opts *EncodeOptions) []*gedcom.Tag {
	if note == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// MIME (level 1)
	if note.MIME != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "MIME", Value: opts.text(note.MIME)})
	}

	// LANG (level 1)
	if note.Language != "" {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "LANG", Value: opts.text(note.Language)})
	}

	// Translations (level 1) - TRAN with nested MIME/LANG at level 2
	for _, tran := range note.Translations {
		tags = append(tags, sharedNoteTranslationToTags(tran, 1, opts)...)
	}

	// Source citations (level 1) - SOUR
	for _, cite := range note.SourceCitations {
		tags = append(tags, sourceCitationToTags(cite, 1, opts)...)
	}

	// External IDs (level 1) - EXID
	tags = append(tags, externalIDsToTags(note.ExternalIDs, 1, opts)...)

	// Change date (level 1) - CHAN
	if note.ChangeDate != nil {
		tags = append(tags, changeDateToTags(note.ChangeDate, 1, "CHAN", opts)...)
	}

	// The note's own extension Tags are appended by writeRecord through
	// entityOwnTags, as for every other entity type. Appending the whole
	// field here, as before #711, wrote a decoded shared note's standard
	// substructures twice once its Record.Tags were cleared.

	return tags
}

// entityOwnTags returns the extension tags an entity carries in its own Tags
// field, so a record written from its entity can carry a vendor or
// application tag beside its typed structures (#711): a hand-built Individual
// with a level-1 _MYAPP_ID in Tags writes it after NAME, and a decoded record
// whose Record.Tags were cleared keeps its _-prefixed lines.
//
// Only extension tags (a leading underscore) are taken, each with its
// subordinate block, and not the two the typed model has fields for, _UID
// (Individual.UIDs and friends, written by uidsToTags) and _FSFTID
// (Individual.FamilySearchID). Standard tags are never written from here,
// typed or not: the decoder sets every entity's Tags to the whole raw
// record, so for a decoded record a standard tag that the typed fields did
// not produce is as likely a structure the caller deleted, or one the
// converter dropped, as one the model has no field for. Writing it back
// would resurrect deleted data, so the typed fields stay the only source of
// standard structures and a caller who wants an untyped standard tag kept
// edits Record.Tags. Anything before the first level-1 tag is skipped.
//
// Record.Tags, when non-empty, is still written verbatim and this is never
// consulted (see the package documentation, "Raw Tags Are Authoritative").
func entityOwnTags(entity any) []*gedcom.Tag {
	own := entityTagsField(entity)
	if len(own) == 0 {
		return nil
	}

	var out []*gedcom.Tag
	keep := false
	for _, t := range own {
		if t == nil {
			continue
		}
		if t.Level <= 1 {
			keep = t.Level == 1 && isOwnExtensionTag(t.Tag)
		}
		if keep {
			out = append(out, t)
		}
	}
	return out
}

// isOwnExtensionTag reports whether a level-1 tag in an entity's Tags is
// written by entityOwnTags: an extension tag the typed model has no field
// for. typedExtensionTags are the extensions it does have fields for; they
// are written from those fields, by uidsToTags and individualToTags, and
// writing them from Tags too would duplicate or resurrect them.
func isOwnExtensionTag(tag string) bool {
	return strings.HasPrefix(tag, "_") && !typedExtensionTags[tag]
}

// typedExtensionTags are the level-1 extension tags the typed model holds in
// a field, so isOwnExtensionTag excludes them. Keep in step with uidTagName
// (_UID) and the FamilySearchID writer in individualToTags (_FSFTID).
var typedExtensionTags = map[string]bool{"_UID": true, "_FSFTID": true}

// entityTagsField returns the Tags field of a record entity, or nil for an
// entity type that has none. A typed nil entity has no Tags (ADR 0007).
//
//nolint:gocyclo // One nil-checked case per entity type that has a Tags field.
func entityTagsField(entity any) []*gedcom.Tag {
	switch e := entity.(type) {
	case *gedcom.Individual:
		if e != nil {
			return e.Tags
		}
	case *gedcom.Family:
		if e != nil {
			return e.Tags
		}
	case *gedcom.Source:
		if e != nil {
			return e.Tags
		}
	case *gedcom.Submitter:
		if e != nil {
			return e.Tags
		}
	case *gedcom.Submission:
		if e != nil {
			return e.Tags
		}
	case *gedcom.Repository:
		if e != nil {
			return e.Tags
		}
	case *gedcom.Note:
		if e != nil {
			return e.Tags
		}
	case *gedcom.MediaObject:
		if e != nil {
			return e.Tags
		}
	case *gedcom.SharedNote:
		if e != nil {
			return e.Tags
		}
	}
	return nil
}

// sharedNoteTranslationToTags converts a SharedNoteTranslation to GEDCOM tags at the specified level.
func sharedNoteTranslationToTags(tran *gedcom.SharedNoteTranslation, level int, opts *EncodeOptions) []*gedcom.Tag {
	if tran == nil {
		return nil
	}

	var tags []*gedcom.Tag

	// TRAN tag with translated text. Route Value through textToTags so a
	// multi-line translation splits into TRAN + leading CONT/CONC tags rather
	// than writing an embedded newline onto the TRAN line (which would forge
	// additional GEDCOM records). Symmetric with the decode-side fold (issue #339).
	tags = append(tags, textToTags(tran.Value, level, "TRAN", opts)...)

	// MIME subordinate at level+1
	if tran.MIME != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "MIME", Value: opts.text(tran.MIME)})
	}

	// LANG subordinate at level+1
	if tran.Language != "" {
		tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "LANG", Value: opts.text(tran.Language)})
	}

	return tags
}

// refNumbersToTags converts record-level reference numbers to level-1 REFN
// tags, each followed by its level-2 TYPE subordinate when set. Every entry is
// written, in order, so repeated REFN values survive a round trip.
func refNumbersToTags(refNumbers []gedcom.RefNumber, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag
	for _, refn := range refNumbers {
		tags = append(tags, &gedcom.Tag{Level: 1, Tag: "REFN", Value: opts.text(refn.Value)})
		if refn.Type != "" {
			tags = append(tags, &gedcom.Tag{Level: 2, Tag: "TYPE", Value: opts.text(refn.Type)})
		}
	}
	return tags
}

// uidsToTags converts unique identifiers to UID tags at the given level, in
// order, spelled for the output version (uidTagName).
func uidsToTags(uids []string, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag
	name := uidTagName(opts)
	for _, uid := range uids {
		tags = append(tags, &gedcom.Tag{Level: level, Tag: name, Value: opts.text(uid)})
	}
	return tags
}

// uidTagName is the tag a unique identifier is written under: _UID, the
// vendor extension every GEDCOM 5.5/5.5.1 desktop producer uses for the same
// identifier and the decoder reads into the same field (#715), when the
// output version forOutput resolved is 5.5 or 5.5.1; otherwise UID, the
// GEDCOM 7.0 structure. An unknown output version, or any other version
// such as a 7.x draft, writes UID, as before.
func uidTagName(opts *EncodeOptions) string {
	if opts != nil && (opts.version == gedcom.Version55 || opts.version == gedcom.Version551) {
		return "_UID"
	}
	return "UID"
}

// externalIDsToTags converts a slice of ExternalIDs to GEDCOM tags at the specified level.
func externalIDsToTags(externalIDs []*gedcom.ExternalID, level int, opts *EncodeOptions) []*gedcom.Tag {
	var tags []*gedcom.Tag

	for _, exid := range externalIDs {
		if exid == nil {
			continue
		}

		// EXID tag with external identifier value
		tags = append(tags, &gedcom.Tag{Level: level, Tag: "EXID", Value: opts.text(exid.Value)})

		// TYPE subordinate at level+1
		if exid.Type != "" {
			tags = append(tags, &gedcom.Tag{Level: level + 1, Tag: "TYPE", Value: opts.text(exid.Type)})
		}
	}

	return tags
}
