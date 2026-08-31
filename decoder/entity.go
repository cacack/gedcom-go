package decoder

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// tagToken returns a tag value with surrounding whitespace removed, for a typed
// field whose value is a token — an XRef pointer, a name part, an enumerated
// code — rather than prose.
//
// GEDCOM's delimiter is exactly one space and the parser keeps every space past
// it, because in a CONC continuation that space is payload (#426). In a token
// position it never is: a producer writing "1 FAMS  @F1@" means "@F1@". Left
// padded, such a value fails gedcom.IsPointerXRef, so the link is silently
// demoted to inline text and merge's XRef remap skips it — and a padded "SURN"
// disagrees with the same name part derived from the NAME line, which
// parsePersonalName already trims.
//
// Only the typed field is normalized. The raw Tag.Value keeps the original
// bytes, so byte fidelity is untouched; this is ADR-003's dual storage doing
// what it is for. Prose fields — note text, CONT/CONC continuations — must not
// use this, because there the extra space is the payload.
func tagToken(s string) string {
	return strings.TrimSpace(s)
}

// diagnosticCollector accumulates diagnostics during entity population.
// It is nil-safe: all methods check for nil receiver before acting.
type diagnosticCollector struct {
	diagnostics Diagnostics
	lenient     bool
}

// add appends a diagnostic to the collector if the collector is non-nil.
func (c *diagnosticCollector) add(d Diagnostic) {
	if c != nil {
		c.diagnostics = append(c.diagnostics, d)
	}
}

// addUnknownTag records an unknown tag diagnostic.
func (c *diagnosticCollector) addUnknownTag(lineNumber int, tag, context string) {
	if c != nil {
		c.add(NewDiagnostic(
			lineNumber,
			SeverityWarning,
			CodeUnknownTag,
			fmt.Sprintf("unknown tag: %s", tag),
			context,
		))
	}
}

// addInvalidValue records an invalid value diagnostic.
func (c *diagnosticCollector) addInvalidValue(lineNumber int, tag, value, reason string) {
	if c != nil {
		c.add(NewDiagnostic(
			lineNumber,
			SeverityWarning,
			CodeInvalidValue,
			fmt.Sprintf("invalid value for %s: %s", tag, reason),
			value,
		))
	}
}

// populateEntities converts raw tags in each record into proper entities.
// If collector is nil, no diagnostics are collected (backward compatible behavior).
func populateEntities(doc *gedcom.Document, collector *diagnosticCollector) {
	for _, record := range doc.Records {
		switch record.Type {
		case gedcom.RecordTypeIndividual:
			record.Entity = parseIndividual(record, collector)
		case gedcom.RecordTypeFamily:
			record.Entity = parseFamily(record, collector)
		case gedcom.RecordTypeSource:
			record.Entity = parseSource(record, collector)
		case gedcom.RecordTypeSubmitter:
			record.Entity = parseSubmitter(record, collector)
		case gedcom.RecordTypeRepository:
			record.Entity = parseRepository(record, collector)
		case gedcom.RecordTypeNote:
			record.Entity = parseNote(record, collector)
		case gedcom.RecordTypeMedia:
			record.Entity = parseMediaObject(record, collector)
		case gedcom.RecordTypeSharedNote:
			record.Entity = parseSharedNote(record, collector)
		}
	}
}

// parseIndividual converts record tags to an Individual entity.
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseIndividual(record *gedcom.Record, collector *diagnosticCollector) *gedcom.Individual {
	indi := &gedcom.Individual{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "NAME":
			name := parsePersonalName(record.Tags, i, collector)
			indi.Names = append(indi.Names, name)

		case "SEX":
			indi.Sex = tagToken(tag.Value)

		case "BIRT", "DEAT", "BAPM", "BURI", "CENS", "CHR", "ADOP", "RESI", "IMMI", "EMIG",
			"BARM", "BASM", "BLES", "CHRA", "CONF", "FCOM",
			"GRAD", "RETI", "NATU", "ORDN", "PROB", "WILL", "CREM", "EVEN":
			event := parseEvent(record.Tags, i, tag.Tag, collector)
			indi.Events = append(indi.Events, event)

		case "NO":
			// GEDCOM 7.0: NO tag indicates event did not occur
			// tag.Value contains the event type (e.g., "MARR", "DEAT")
			if strings.TrimSpace(tag.Value) == "" {
				collector.addInvalidValue(tag.LineNumber, "NO", tag.Value, "missing event type")
				continue
			}
			event := parseEvent(record.Tags, i, tag.Value, collector)
			event.IsNegative = true
			indi.Events = append(indi.Events, event)

		case "BAPL", "CONL", "ENDL", "SLGC":
			ord := parseLDSOrdinance(record.Tags, i, ldsOrdinanceType(tag.Tag), collector)
			indi.LDSOrdinances = append(indi.LDSOrdinances, ord)

		case "OCCU", "CAST", "DSCR", "EDUC", "IDNO", "NATI", "SSN", "TITL", "RELI", "NCHI", "NMR", "PROP", "FACT":
			attr := parseAttribute(record.Tags, i, tag.Tag, collector)
			indi.Attributes = append(indi.Attributes, attr)

		case "FAMC":
			famLink := parseFamilyLink(record.Tags, i, collector)
			indi.ChildInFamilies = append(indi.ChildInFamilies, famLink)

		case "FAMS":
			indi.SpouseInFamilies = append(indi.SpouseInFamilies, tagToken(tag.Value))

		case "ASSO":
			assoc := parseAssociation(record.Tags, i, collector)
			indi.Associations = append(indi.Associations, assoc)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			indi.SourceCitations = append(indi.SourceCitations, cite)

		case "NOTE", "SNOTE":
			indi.NoteXRefs, indi.InlineNotes, indi.Notes = appendRecordNote(record.Tags, i, indi.NoteXRefs, indi.InlineNotes, indi.Notes)

		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level, collector)
			indi.Media = append(indi.Media, link)

		case "CHAN":
			indi.ChangeDate = parseChangeDate(record.Tags, i, collector)

		case "CREA":
			indi.CreationDate = parseChangeDate(record.Tags, i, collector)

		case "REFN":
			indi.RefNumber = tag.Value

		case "UID":
			indi.UID = tag.Value

		case "_FSFTID":
			// A leading "@@" is the escaped form of a literal leading "@" (see
			// gedcom.EscapeLeadingAt, applied by the EXID->_FSFTID converter).
			// Unescape it so callers reading Individual.FamilySearchID /
			// FamilySearchURL() get the logical id; the raw tag retains "@@".
			indi.FamilySearchID = gedcom.UnescapeLeadingAt(tag.Value)

		case "EXID":
			indi.ExternalIDs = append(indi.ExternalIDs, parseExternalID(record.Tags, i))

		case "ALIA", "AFN", "RFN", "RIN", "ANCI", "DESI", "SUBM", "RESN", "INIL":
			// Standard INDI substructures not yet parsed into typed fields; the
			// raw tags remain on Individual.Tags (ADR 0003). Recognizing them
			// keeps UNKNOWN_TAG meaning "nonstandard tag" for callers who filter
			// on it (issue #375). AFN/RFN/RIN are 5.5/5.5.1, INIL is 7.0.
			// INIL stays here by decision (issue #386), not for want of a
			// fixture — maximal70.ged and maximal70-lds.ged both carry one.
			// A typed INIL needs a sixth gedcom.LDSOrdinanceType constant, but
			// nothing in this repo switches on that type and the downstream
			// consumer drops unrecognized ordinances at its default case, so
			// the constant would ship for no reader. Revisit when a consumer
			// grows a domain type for the initiatory ordinance.
			// FACT is deliberately absent: it now decodes into
			// Individual.Attributes, its subordinate TYPE landing in
			// Attribute.TypeDetail.
			// EVEN is deliberately absent: issue #378 tracks decoding it into
			// Individual.Events, which reclassifies it as a side effect.

		default:
			// Unknown tag - record diagnostic but continue processing
			// Tags starting with _ are vendor extensions and expected
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return indi
}

//nolint:gocyclo // Name parsing requires handling many tag types and edge cases
func parsePersonalName(tags []*gedcom.Tag, nameIdx int, collector *diagnosticCollector) *gedcom.PersonalName {
	name := &gedcom.PersonalName{
		Full: tags[nameIdx].Value,
	}

	// Parse the full name to extract given and surname
	// GEDCOM format: "Given /Surname/"
	full := tags[nameIdx].Value
	if slashIdx := strings.Index(full, "/"); slashIdx >= 0 {
		name.Given = strings.TrimSpace(full[:slashIdx])
		surname := full[slashIdx+1:]
		if endSlash := strings.Index(surname, "/"); endSlash >= 0 {
			name.Surname = surname[:endSlash]
		} else {
			name.Surname = strings.TrimSpace(surname)
		}
	} else {
		name.Given = strings.TrimSpace(full)
	}

	// Look for subordinate tags (level 2)
	for i := nameIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "GIVN":
				name.Given = tagToken(tag.Value)
			case "SURN":
				name.Surname = tagToken(tag.Value)
			case "NPFX":
				name.Prefix = tagToken(tag.Value)
			case "NSFX":
				name.Suffix = tagToken(tag.Value)
			case "NICK":
				name.Nickname = tagToken(tag.Value)
			case "SPFX":
				name.SurnamePrefix = tagToken(tag.Value)
			case "TYPE":
				name.Type = tag.Value
			case "TRAN":
				tran := parseNameTransliteration(tags, i, collector)
				name.Transliterations = append(name.Transliterations, tran)
			case "SOUR", "NOTE", "SNOTE", "FONE", "ROMN":
				// Known tags that we don't parse into typed fields (yet)
				// SOUR/NOTE are common, FONE/ROMN are GEDCOM 5.5.1 phonetic/romanized variants
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return name
}

// parseNameTransliteration extracts a transliteration from tags starting at tranIdx.
// TRAN tags under NAME contain the transliterated name value and optional component tags.
func parseNameTransliteration(tags []*gedcom.Tag, tranIdx int, collector *diagnosticCollector) *gedcom.Transliteration {
	baseLevel := tags[tranIdx].Level

	tran := &gedcom.Transliteration{
		Value: tags[tranIdx].Value,
	}

	// Look for subordinate tags at baseLevel+1 (level 3 for NAME.TRAN)
	for i := tranIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "LANG":
				tran.Language = tag.Value
			case "GIVN":
				tran.Given = tag.Value
			case "SURN":
				tran.Surname = tag.Value
			case "NPFX":
				tran.Prefix = tag.Value
			case "NSFX":
				tran.Suffix = tag.Value
			case "NICK":
				tran.Nickname = tag.Value
			case "SPFX":
				tran.SurnamePrefix = tag.Value
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return tran
}

// parseFamilyLink extracts a family link from tags starting at famcIdx.
func parseFamilyLink(tags []*gedcom.Tag, famcIdx int, collector *diagnosticCollector) gedcom.FamilyLink {
	famLink := gedcom.FamilyLink{
		FamilyXRef: tagToken(tags[famcIdx].Value),
	}

	// Look for subordinate tags (level 2)
	for i := famcIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "PEDI":
				famLink.Pedigree = tag.Value
			case "STAT", "NOTE", "SNOTE":
				// Known tags not yet parsed into typed fields
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return famLink
}

// parseAssociation extracts an association from tags starting at assoIdx.
func parseAssociation(tags []*gedcom.Tag, assoIdx int, collector *diagnosticCollector) *gedcom.Association {
	baseLevel := tags[assoIdx].Level

	assoc := &gedcom.Association{
		IndividualXRef: tagToken(tags[assoIdx].Value),
	}

	// Look for subordinate tags at baseLevel+1
	for i := assoIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "RELA", "ROLE": // RELA in 5.5.1, ROLE in 7.0
				assoc.Role = tag.Value
			case "PHRASE":
				assoc.Phrase = tag.Value
			case "NOTE", "SNOTE":
				assoc.Notes = append(assoc.Notes, tag.Value)
			case "SOUR":
				cite := parseSourceCitation(tags, i, tag.Level, collector)
				assoc.SourceCitations = append(assoc.SourceCitations, cite)
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return assoc
}

// parseExternalID parses an EXID tag with optional TYPE subordinate.
func parseExternalID(tags []*gedcom.Tag, exidIdx int) *gedcom.ExternalID {
	baseLevel := tags[exidIdx].Level
	exid := &gedcom.ExternalID{
		Value: tags[exidIdx].Value,
	}

	// Look for TYPE subordinate at baseLevel+1
	for i := exidIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 && tag.Tag == "TYPE" {
			exid.Type = tag.Value
			break
		}
	}
	return exid
}

// parseSourceCitation extracts a source citation from tags starting at sourIdx.
func parseSourceCitation(tags []*gedcom.Tag, sourIdx, baseLevel int, collector *diagnosticCollector) *gedcom.SourceCitation {
	cite := &gedcom.SourceCitation{
		SourceXRef: tagToken(tags[sourIdx].Value),
	}

	// Look for subordinate tags at baseLevel+1
	for i := sourIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "PAGE":
				cite.Page = tag.Value
			case "QUAY":
				// Parse quality as integer (0-3). QUAY 0 is a real assertion,
				// so the pointer is set for every in-range value.
				if q, err := strconv.Atoi(tag.Value); err == nil && q >= 0 && q <= 3 {
					quality := q
					cite.Quality = &quality
				} else {
					collector.addInvalidValue(tag.LineNumber, "QUAY", tag.Value, "expected integer 0-3")
				}
			case "DATA":
				// Parse DATA subordinates at baseLevel+2
				cite.Data = parseSourceCitationData(tags, i, baseLevel+1, collector)
			case "_APID":
				// Parse Ancestry Permanent Identifier (vendor extension)
				cite.AncestryAPID = gedcom.ParseAPID(tag.Value)
			case "NOTE", "SNOTE":
				cite.NoteXRefs, cite.InlineNotes, cite.Notes = appendRecordNote(tags, i, cite.NoteXRefs, cite.InlineNotes, cite.Notes)
			case "OBJE", "EVEN", "TEXT":
				// Known tags not yet parsed into typed fields
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return cite
}

// parseSourceCitationData extracts source citation data from tags starting at dataIdx.
func parseSourceCitationData(tags []*gedcom.Tag, dataIdx, baseLevel int, collector *diagnosticCollector) *gedcom.SourceCitationData {
	data := &gedcom.SourceCitationData{}

	// Look for subordinate tags at baseLevel+1
	for i := dataIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				data.Date = tag.Value
			case "TEXT":
				data.Text = tag.Value
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return data
}

// eventDetail is a writable view over the GEDCOM EVENT_DETAIL substructures
// that gedcom.Event and gedcom.Attribute both carry under identical field
// names. parseEvent and parseAttribute each build one over their own struct and
// hand it to parseEventDetailTag, so the shared subtag list lives in exactly one
// switch. Duplicating that switch is what let parseAttribute fall a dozen tags
// behind parseEvent (issue #402); a shared view cannot drift the same way.
//
// Every field must be set by the constructors below: parseEventDetailTag writes
// through the pointers unguarded.
type eventDetail struct {
	date            *string
	parsedDate      **gedcom.Date
	placeDetail     **gedcom.PlaceDetail
	cause           *string
	agency          *string
	religion        *string
	address         **gedcom.Address
	phone           *[]string
	email           *[]string
	fax             *[]string
	website         *[]string
	restriction     *string
	uid             *string
	sortDate        *string
	sourceCitations *[]*gedcom.SourceCitation
	media           *[]*gedcom.MediaLink
	associations    *[]*gedcom.Association
	noteXRefs       *[]string
	inlineNotes     *[]string
	notes           *[]string
}

// eventDetailOf returns the EVENT_DETAIL view of an event.
func eventDetailOf(e *gedcom.Event) eventDetail {
	return eventDetail{
		date:            &e.Date,
		parsedDate:      &e.ParsedDate,
		placeDetail:     &e.PlaceDetail,
		cause:           &e.Cause,
		agency:          &e.Agency,
		religion:        &e.ReligiousAffiliation,
		address:         &e.Address,
		phone:           &e.Phone,
		email:           &e.Email,
		fax:             &e.Fax,
		website:         &e.Website,
		restriction:     &e.Restriction,
		uid:             &e.UID,
		sortDate:        &e.SortDate,
		sourceCitations: &e.SourceCitations,
		media:           &e.Media,
		associations:    &e.Associations,
		noteXRefs:       &e.NoteXRefs,
		inlineNotes:     &e.InlineNotes,
		notes:           &e.Notes,
	}
}

// eventDetailOfAttribute returns the EVENT_DETAIL view of an attribute.
func eventDetailOfAttribute(a *gedcom.Attribute) eventDetail {
	return eventDetail{
		date:            &a.Date,
		parsedDate:      &a.ParsedDate,
		placeDetail:     &a.PlaceDetail,
		cause:           &a.Cause,
		agency:          &a.Agency,
		religion:        &a.ReligiousAffiliation,
		address:         &a.Address,
		phone:           &a.Phone,
		email:           &a.Email,
		fax:             &a.Fax,
		website:         &a.Website,
		restriction:     &a.Restriction,
		uid:             &a.UID,
		sortDate:        &a.SortDate,
		sourceCitations: &a.SourceCitations,
		media:           &a.Media,
		associations:    &a.Associations,
		noteXRefs:       &a.NoteXRefs,
		inlineNotes:     &a.InlineNotes,
		notes:           &a.Notes,
	}
}

// parseEventDetailTag decodes the EVENT_DETAIL subtag at tags[i] into detail.
// It covers every substructure gedcom.Event and gedcom.Attribute share; the
// tags that belong to only one of them (TYPE, and AGE, which is typed on an
// event and not on an attribute) stay with their own parser, which delegates
// here for the rest.
// Unrecognized standard tags are reported to collector (ADR 0007).
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseEventDetailTag(detail *eventDetail, tags []*gedcom.Tag, i int, collector *diagnosticCollector) {
	tag := tags[i]
	switch tag.Tag {
	case "DATE":
		*detail.date = tag.Value
		if parsed, err := gedcom.ParseDate(tag.Value); err == nil {
			*detail.parsedDate = parsed
		} else {
			// Date parsing failed - raw value is preserved, add diagnostic
			collector.addInvalidValue(tag.LineNumber, "DATE", tag.Value, err.Error())
		}
		// PHRASE (7.0) is a DATE subordinate carrying the date in prose. An
		// empty one is ignored rather than assigned, because ParseDate itself
		// fills Phrase for a parenthesized date phrase.
		if phrase := findSubordinate(tags, i, "PHRASE"); phrase != "" {
			if parsed := *detail.parsedDate; parsed != nil {
				parsed.Phrase = phrase
			}
		}
	case "PLAC":
		*detail.placeDetail = parsePlaceDetail(tags, i, tag.Level, collector)
	case "CAUS":
		*detail.cause = tag.Value
	case "AGNC":
		*detail.agency = tag.Value
	case "RELI":
		*detail.religion = tag.Value
	case "ADDR":
		*detail.address = parseEventAddress(tags, i, tag.Level, collector)
	case "PHON":
		*detail.phone = append(*detail.phone, tag.Value)
	case "EMAIL":
		*detail.email = append(*detail.email, tag.Value)
	case "FAX":
		*detail.fax = append(*detail.fax, tag.Value)
	case "WWW":
		*detail.website = append(*detail.website, tag.Value)
	case "RESN":
		*detail.restriction = tag.Value
	case "UID":
		*detail.uid = tag.Value
	case "SDATE":
		*detail.sortDate = tag.Value
	case "SOUR":
		cite := parseSourceCitation(tags, i, tag.Level, collector)
		*detail.sourceCitations = append(*detail.sourceCitations, cite)
	case "OBJE":
		link := parseMediaLink(tags, i, tag.Level, collector)
		*detail.media = append(*detail.media, link)
	case "ASSO":
		assoc := parseAssociation(tags, i, collector)
		*detail.associations = append(*detail.associations, assoc)
	case "NOTE", "SNOTE":
		*detail.noteXRefs, *detail.inlineNotes, *detail.notes = appendRecordNote(
			tags, i, *detail.noteXRefs, *detail.inlineNotes, *detail.notes)
	case "HUSB", "WIFE":
		// FAMILY_EVENT_DETAIL: the spouse ages on a family event or family
		// attribute (maximal70.ged carries one under FAM.FACT). Known tags not
		// yet parsed into typed fields.
	default:
		if !strings.HasPrefix(tag.Tag, "_") {
			collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
		}
	}
}

// parseEvent extracts an event from tags starting at eventIdx.
func parseEvent(tags []*gedcom.Tag, eventIdx int, eventTag string, collector *diagnosticCollector) *gedcom.Event {
	event := &gedcom.Event{
		Type: gedcom.EventType(eventTag),
	}

	// A generic event carries its descriptor as the EVEN line's own payload;
	// named event tags carry the [Y|<NULL>] flag instead, which has no typed
	// field. Guard on the raw tag so "NO EVEN" (7.0 negative assertion, where
	// eventTag comes from the NO value) is not mistaken for a descriptor.
	if tags[eventIdx].Tag == "EVEN" {
		event.Description = tags[eventIdx].Value
	}

	baseLevel := tags[eventIdx].Level
	detail := eventDetailOf(event)

	// Look for subordinate tags at baseLevel+1
	for i := eventIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "TYPE":
				event.EventTypeDetail = tag.Value
			case "AGE":
				event.Age = tag.Value
			default:
				// Everything else is EVENT_DETAIL, shared with parseAttribute.
				parseEventDetailTag(&detail, tags, i, collector)
			}
		}
	}

	return event
}

// parseEventAddress extracts an address structure from tags starting at addrIdx.
func parseEventAddress(tags []*gedcom.Tag, addrIdx, baseLevel int, collector *diagnosticCollector) *gedcom.Address {
	addr := &gedcom.Address{
		Line1: tags[addrIdx].Value,
	}

	// Look for subordinate tags at baseLevel+1
	for i := addrIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "ADR1":
				addr.Line1 = tag.Value
			case "ADR2":
				addr.Line2 = tag.Value
			case "ADR3":
				addr.Line3 = tag.Value
			case "CITY":
				addr.City = tag.Value
			case "STAE":
				addr.State = tag.Value
			case "POST":
				addr.PostalCode = tag.Value
			case "CTRY":
				addr.Country = tag.Value
			case "CONT":
				// Continue address on next line. Not foldContinuation: an empty
				// first line takes the value directly, with no leading newline.
				if addr.Line1 != "" {
					addr.Line1 += "\n" + tag.Value
				} else {
					addr.Line1 = tag.Value
				}
			case "CONC":
				// Concatenate to address
				addr.Line1 += tag.Value
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return addr
}

// parsePlaceDetail extracts a place structure with optional coordinates from tags starting at placIdx.
func parsePlaceDetail(tags []*gedcom.Tag, placIdx, baseLevel int, collector *diagnosticCollector) *gedcom.PlaceDetail {
	place := &gedcom.PlaceDetail{
		Name: tags[placIdx].Value,
	}

	// Look for subordinate tags at baseLevel+1
	for i := placIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "FORM":
				place.Form = tag.Value
			case "MAP":
				place.Coordinates = parseCoordinates(tags, i, tag.Level, collector)
			case "FONE", "ROMN", "TRAN", "NOTE", "SNOTE", "EXID", "LANG":
				// Known tags not yet parsed into typed fields
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return place
}

// parseCoordinates extracts geographic coordinates from tags starting at mapIdx.
func parseCoordinates(tags []*gedcom.Tag, mapIdx, baseLevel int, collector *diagnosticCollector) *gedcom.Coordinates {
	coords := &gedcom.Coordinates{}

	// Look for subordinate tags at baseLevel+1
	for i := mapIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "LATI":
				coords.Latitude = tag.Value
			case "LONG":
				coords.Longitude = tag.Value
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return coords
}

// parseAttribute extracts an attribute from tags starting at attrIdx.
func parseAttribute(tags []*gedcom.Tag, attrIdx int, attrTag string, collector *diagnosticCollector) *gedcom.Attribute {
	attr := &gedcom.Attribute{
		Type:  attrTag,
		Value: tags[attrIdx].Value,
	}

	baseLevel := tags[attrIdx].Level
	detail := eventDetailOfAttribute(attr)

	// Look for subordinate tags at baseLevel+1
	for i := attrIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "TYPE":
				attr.TypeDetail = tag.Value
			case "AGE":
				// Known tag not parsed into a typed field: gedcom.Attribute has
				// no Age counterpart to Event.Age, and nothing asks for one. The
				// raw tag remains on the owning record's Tags (ADR 0003).
			default:
				// Everything else is EVENT_DETAIL, shared with parseEvent.
				parseEventDetailTag(&detail, tags, i, collector)
			}
		}
	}

	return attr
}

// ldsOrdinanceType maps a GEDCOM tag to its LDSOrdinanceType.
func ldsOrdinanceType(tag string) gedcom.LDSOrdinanceType {
	switch tag {
	case "BAPL":
		return gedcom.LDSBaptism
	case "CONL":
		return gedcom.LDSConfirmation
	case "ENDL":
		return gedcom.LDSEndowment
	case "SLGC":
		return gedcom.LDSSealingChild
	case "SLGS":
		return gedcom.LDSSealingSpouse
	default:
		return gedcom.LDSOrdinanceType(tag)
	}
}

// parseLDSOrdinance extracts an LDS ordinance from tags starting at ordIdx.
func parseLDSOrdinance(tags []*gedcom.Tag, ordIdx int, ordType gedcom.LDSOrdinanceType, collector *diagnosticCollector) *gedcom.LDSOrdinance {
	ord := &gedcom.LDSOrdinance{
		Type: ordType,
	}

	baseLevel := tags[ordIdx].Level

	// Look for subordinate tags at baseLevel+1
	for i := ordIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				ord.Date = tag.Value
				if parsed, err := gedcom.ParseDate(tag.Value); err == nil {
					ord.ParsedDate = parsed
				} else {
					collector.addInvalidValue(tag.LineNumber, "DATE", tag.Value, err.Error())
				}
			case "TEMP":
				ord.Temple = tag.Value
			case "PLAC":
				ord.Place = tag.Value
			case "STAT":
				ord.Status = tag.Value
			case "FAMC":
				ord.FamilyXRef = tagToken(tag.Value)
			case "NOTE", "SNOTE":
				ord.NoteXRefs, ord.InlineNotes, ord.Notes = appendRecordNote(tags, i, ord.NoteXRefs, ord.InlineNotes, ord.Notes)
			case "SOUR":
				// Known tag not yet parsed into typed fields
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return ord
}

// parseFamily converts record tags to a Family entity.
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseFamily(record *gedcom.Record, collector *diagnosticCollector) *gedcom.Family {
	fam := &gedcom.Family{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "HUSB":
			fam.Husband = tagToken(tag.Value)

		case "WIFE":
			fam.Wife = tagToken(tag.Value)

		case "CHIL":
			fam.Children = append(fam.Children, tagToken(tag.Value))

		case "NCHI", "FACT":
			attr := parseAttribute(record.Tags, i, tag.Tag, collector)
			fam.Attributes = append(fam.Attributes, attr)

		case "MARR", "DIV", "ENGA", "ANUL", "MARB", "MARC", "MARL", "MARS", "DIVF", "CENS", "RESI", "EVEN":
			event := parseEvent(record.Tags, i, tag.Tag, collector)
			fam.Events = append(fam.Events, event)

		case "NO":
			// GEDCOM 7.0: NO tag indicates event did not occur
			// tag.Value contains the event type (e.g., "MARR", "DIV")
			if strings.TrimSpace(tag.Value) == "" {
				collector.addInvalidValue(tag.LineNumber, "NO", tag.Value, "missing event type")
				continue
			}
			event := parseEvent(record.Tags, i, tag.Value, collector)
			event.IsNegative = true
			fam.Events = append(fam.Events, event)

		case "SLGS":
			ord := parseLDSOrdinance(record.Tags, i, ldsOrdinanceType(tag.Tag), collector)
			fam.LDSOrdinances = append(fam.LDSOrdinances, ord)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			fam.SourceCitations = append(fam.SourceCitations, cite)

		case "NOTE", "SNOTE":
			fam.NoteXRefs, fam.InlineNotes, fam.Notes = appendRecordNote(record.Tags, i, fam.NoteXRefs, fam.InlineNotes, fam.Notes)

		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level, collector)
			fam.Media = append(fam.Media, link)

		case "CHAN":
			fam.ChangeDate = parseChangeDate(record.Tags, i, collector)

		case "CREA":
			fam.CreationDate = parseChangeDate(record.Tags, i, collector)

		case "REFN":
			fam.RefNumber = tag.Value

		case "UID":
			fam.UID = tag.Value

		case "EXID":
			fam.ExternalIDs = append(fam.ExternalIDs, parseExternalID(record.Tags, i))

		case "RESN", "SUBM", "ASSO":
			// Known tags not yet parsed into typed fields; the raw tags remain
			// on Family.Tags (ADR 0003). Recognizing them keeps UNKNOWN_TAG
			// meaning "nonstandard tag" for callers who filter on it (#375).
			// FACT is deliberately absent: like the INDI side (issue #386) it
			// now decodes into Family.Attributes, so its own substructures are
			// walked and can be diagnosed (issue #448).

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return fam
}

//nolint:gocyclo // Source parsing requires handling many tag types
func parseSource(record *gedcom.Record, collector *diagnosticCollector) *gedcom.Source {
	src := &gedcom.Source{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "TITL":
			src.Title = tag.Value
		case "AUTH":
			src.Author = tag.Value
		case "PUBL":
			src.Publication = tag.Value
		case "TEXT":
			src.Text = tag.Value
		case "REPO":
			src.RepositoryLink = parseSourceRepositoryLink(record.Tags, i, collector)
		case "NOTE", "SNOTE":
			src.NoteXRefs, src.InlineNotes, src.Notes = appendRecordNote(record.Tags, i, src.NoteXRefs, src.InlineNotes, src.Notes)
		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level, collector)
			src.Media = append(src.Media, link)
		case "CHAN":
			src.ChangeDate = parseChangeDate(record.Tags, i, collector)
		case "CREA":
			src.CreationDate = parseChangeDate(record.Tags, i, collector)
		case "REFN":
			src.RefNumber = tag.Value
		case "UID":
			src.UID = tag.Value
		case "EXID":
			src.ExternalIDs = append(src.ExternalIDs, parseExternalID(record.Tags, i))
		case "DATA", "ABBR":
			// Known tags not yet parsed into typed fields
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return src
}

// parseSourceRepositoryLink extracts the structured REPO link of a source from
// tags starting at repoIdx. The REPO substructure may carry a repository XRef
// (or an inline repository by NAME), CALN call numbers (each with an optional
// MEDI media type), and NOTE subordinates.
func parseSourceRepositoryLink(tags []*gedcom.Tag, repoIdx int, collector *diagnosticCollector) *gedcom.SourceRepositoryLink {
	link := &gedcom.SourceRepositoryLink{}

	repoTag := tags[repoIdx]
	baseLevel := repoTag.Level

	if repoTag.Value != "" {
		link.XRef = tagToken(repoTag.Value)
	} else {
		// No XRef value: this is an inline repository referenced by name.
		link.Inline = &gedcom.InlineRepository{}
	}

	for i := repoIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level != baseLevel+1 {
			continue
		}
		switch tag.Tag {
		case "NAME":
			// A NAME subordinate denotes an inline repository. If the REPO tag
			// already carried an XRef value, the record is malformed (XRef and
			// inline name are mutually exclusive); keep the XRef as canonical
			// and ignore the stray NAME rather than populate both fields.
			if link.XRef == "" {
				if link.Inline == nil {
					link.Inline = &gedcom.InlineRepository{}
				}
				link.Inline.Name = tag.Value
			}
		case "CALN":
			link.CallNumbers = append(link.CallNumbers, tag.Value)
			// MEDI is a subordinate of CALN at baseLevel+2.
			if medi := findSubordinate(tags, i, "MEDI"); medi != "" {
				if link.MediaType == "" {
					link.MediaType = medi
				}
				if link.CallNumberMedia == nil {
					link.CallNumberMedia = make(map[string]string)
				}
				// Duplicate CALN strings collapse to last-writer-wins here; the
				// CallNumbers slice retains every entry. See the CallNumberMedia
				// doc comment in gedcom/repository.go.
				link.CallNumberMedia[tag.Value] = medi
			}
		case "NOTE", "SNOTE":
			link.Notes = append(link.Notes, tag.Value)
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return link
}

// findSubordinate returns the value of the first direct child tag of the tag at
// parentIdx that matches name, or "" if none exists.
func findSubordinate(tags []*gedcom.Tag, parentIdx int, name string) string {
	parentLevel := tags[parentIdx].Level
	for i := parentIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= parentLevel {
			break
		}
		if tag.Level == parentLevel+1 && tag.Tag == name {
			return tag.Value
		}
	}
	return ""
}

// parseChangeDate extracts a change date structure from tags starting at chanIdx.
// Used for both CHAN (change date) and CREA (creation date) tags.
func parseChangeDate(tags []*gedcom.Tag, chanIdx int, collector *diagnosticCollector) *gedcom.ChangeDate {
	cd := &gedcom.ChangeDate{}

	baseLevel := tags[chanIdx].Level

	// Look for subordinate tags at baseLevel+1
	for i := chanIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				cd.Date = tag.Value
				// Look for TIME subordinate at baseLevel+2
				for j := i + 1; j < len(tags); j++ {
					timeTag := tags[j]
					if timeTag.Level <= baseLevel+1 {
						break
					}
					if timeTag.Level == baseLevel+2 && timeTag.Tag == "TIME" {
						cd.Time = timeTag.Value
						break
					}
				}
			case "NOTE", "SNOTE":
				cd.NoteXRefs, cd.InlineNotes, cd.Notes = appendRecordNote(tags, i, cd.NoteXRefs, cd.InlineNotes, cd.Notes)
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return cd
}

// parseSubmitter converts record tags to a Submitter entity.
func parseSubmitter(record *gedcom.Record, collector *diagnosticCollector) *gedcom.Submitter {
	subm := &gedcom.Submitter{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "NAME":
			subm.Name = tag.Value

		case "ADDR":
			subm.Address = parseEventAddress(record.Tags, i, tag.Level, collector)

		case "PHON":
			subm.Phone = append(subm.Phone, tag.Value)

		case "EMAIL":
			subm.Email = append(subm.Email, tag.Value)

		case "LANG":
			subm.Language = append(subm.Language, tag.Value)

		case "NOTE", "SNOTE":
			subm.NoteXRefs, subm.InlineNotes, subm.Notes = appendRecordNote(record.Tags, i, subm.NoteXRefs, subm.InlineNotes, subm.Notes)

		case "EXID":
			subm.ExternalIDs = append(subm.ExternalIDs, parseExternalID(record.Tags, i))

		case "CHAN", "FAX", "WWW", "OBJE", "RIN", "UID":
			// Known tags not yet parsed into typed fields

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return subm
}

// parseRepository converts record tags to a Repository entity.
func parseRepository(record *gedcom.Record, collector *diagnosticCollector) *gedcom.Repository {
	repo := &gedcom.Repository{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "NAME":
			repo.Name = tag.Value

		case "ADDR":
			repo.Address = parseEventAddress(record.Tags, i, tag.Level, collector)

		case "PHON":
			if repo.Address == nil {
				repo.Address = &gedcom.Address{}
			}
			repo.Address.Phone = tag.Value

		case "EMAIL":
			if repo.Address == nil {
				repo.Address = &gedcom.Address{}
			}
			repo.Address.Email = tag.Value

		case "WWW":
			if repo.Address == nil {
				repo.Address = &gedcom.Address{}
			}
			repo.Address.Website = tag.Value

		case "NOTE", "SNOTE":
			repo.NoteXRefs, repo.InlineNotes, repo.Notes = appendRecordNote(record.Tags, i, repo.NoteXRefs, repo.InlineNotes, repo.Notes)

		case "EXID":
			repo.ExternalIDs = append(repo.ExternalIDs, parseExternalID(record.Tags, i))

		case "CHAN", "REFN", "UID", "FAX":
			// Known tags not yet parsed into typed fields

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return repo
}

// foldContinuation folds a CONT/CONC continuation tag into the builder b:
// CONT writes a newline followed by the value, CONC writes the value directly.
// Any other tag is a no-op. Writing into a single builder keeps folding O(n)
// across a run of continuation lines. Callers that special-case an empty base
// (parseEventAddress) intentionally do not use this.
func foldContinuation(b *strings.Builder, tag *gedcom.Tag) {
	switch tag.Tag {
	case "CONT":
		b.WriteString("\n")
		b.WriteString(tag.Value)
	case "CONC":
		b.WriteString(tag.Value)
	}
}

// appendRecordNote classifies a record-level NOTE tag at noteIdx and appends it
// to the appropriate slice. A pointer-shaped value (e.g. "@N1@") is an XRef to a
// shared NOTE/SNOTE record and is appended to *xrefs. Any other value is inline
// note text and is appended to *inline, with subordinate CONT/CONC lines folded
// in (CONT joins with a newline, CONC concatenates). The legacy combined Notes
// slice is appended to in the same order for backward compatibility.
//
// It returns the updated xrefs, inline, and legacy slices.
func appendRecordNote(tags []*gedcom.Tag, noteIdx int, xrefs, inline, legacy []string) (newXRefs, newInline, newLegacy []string) {
	tag := tags[noteIdx]
	// The pointer test runs on the trimmed value, and only a value that passes
	// it is stored trimmed. "1 NOTE  @N1@" is a padded pointer (#426 keeps every
	// space past the delimiter), and testing the raw value would classify it as
	// inline text, silently dropping the link to the shared note. Inline text
	// keeps its raw value, because there a leading space is payload.
	if ptr := tagToken(tag.Value); gedcom.IsPointerXRef(ptr) {
		// XRef pointer to a shared note: the GEDCOM specs do not permit
		// subordinate CONT/CONC lines here, so there is nothing to fold in.
		return append(xrefs, ptr), inline, append(legacy, ptr)
	}

	var b strings.Builder
	b.WriteString(tag.Value)
	baseLevel := tag.Level
	for i := noteIdx + 1; i < len(tags); i++ {
		sub := tags[i]
		if sub.Level <= baseLevel {
			break
		}
		if sub.Level != baseLevel+1 {
			continue
		}
		foldContinuation(&b, sub)
	}
	text := b.String()
	return xrefs, append(inline, text), append(legacy, text)
}

// parseNote converts record tags to a Note entity.
func parseNote(record *gedcom.Record, collector *diagnosticCollector) *gedcom.Note {
	note := &gedcom.Note{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	// Seed the fold with the level-0 NOTE value; continuation lines fold in
	// below. note.Text is assigned once after the loop to keep folding O(n).
	var b strings.Builder
	b.WriteString(record.Value)

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "CONT", "CONC":
			// Fold continuation lines back into the primary text, symmetric
			// with parseSharedNote. Before #439 only CONC was folded and CONT
			// went to the Continuation slice, so every multi-line NOTE read
			// back through Text as its first line alone. Continuation is left
			// nil on decode; see its doc comment.
			foldContinuation(&b, tag)

		case "EXID":
			note.ExternalIDs = append(note.ExternalIDs, parseExternalID(record.Tags, i))

		case "MIME", "LANG", "TRAN", "SOUR", "REFN", "UID", "CHAN":
			// Known tags not yet parsed into typed fields

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	note.Text = b.String()

	return note
}

// parseSharedNote converts record tags to a SharedNote entity (GEDCOM 7.0).
// SharedNote records are distinct from NOTE records and support MIME types,
// language tags, and translations for internationalization.
func parseSharedNote(record *gedcom.Record, collector *diagnosticCollector) *gedcom.SharedNote {
	note := &gedcom.SharedNote{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	// Seed the fold with the level-0 SNOTE value; continuation lines fold in
	// below. note.Text is assigned once after the loop to keep folding O(n).
	var b strings.Builder
	b.WriteString(record.Value)

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "MIME":
			note.MIME = tag.Value

		case "LANG":
			note.Language = tag.Value

		case "TRAN":
			tran := parseSharedNoteTranslation(record.Tags, i)
			note.Translations = append(note.Translations, tran)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			note.SourceCitations = append(note.SourceCitations, cite)

		case "EXID":
			note.ExternalIDs = append(note.ExternalIDs, parseExternalID(record.Tags, i))

		case "CHAN":
			note.ChangeDate = parseChangeDate(record.Tags, i, collector)

		case "CONT", "CONC":
			// Fold continuation lines back into the primary text so consumers
			// reading SharedNote.Text get the full multi-line body, not just the
			// first line (which lives in record.Value). CONC is invalid in GEDCOM
			// 7.0 SNOTE but is folded in for robustness against malformed input.
			foldContinuation(&b, tag)

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	note.Text = b.String()

	return note
}

// parseSharedNoteTranslation extracts a translation from TRAN tag in a SharedNote.
func parseSharedNoteTranslation(tags []*gedcom.Tag, tranIdx int) *gedcom.SharedNoteTranslation {
	baseLevel := tags[tranIdx].Level

	tran := &gedcom.SharedNoteTranslation{}

	// Seed the fold with the TRAN value; CONT/CONC continuation lines fold in
	// below, symmetric with parseSharedNote's primary-text fold (issue #339).
	// tran.Value is assigned once after the loop to keep folding O(n).
	var b strings.Builder
	b.WriteString(tags[tranIdx].Value)

	// Look for subordinate tags at baseLevel+1
	for i := tranIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "MIME":
				tran.MIME = tag.Value
			case "LANG":
				tran.Language = tag.Value
			case "CONT", "CONC":
				foldContinuation(&b, tag)
			}
		}
	}

	tran.Value = b.String()

	return tran
}

// parseMediaObject converts record tags to a MediaObject entity.
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseMediaObject(record *gedcom.Record, collector *diagnosticCollector) *gedcom.MediaObject {
	media := &gedcom.MediaObject{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "FILE":
			file := parseMediaFile(record.Tags, i, tag.Level, collector)
			media.Files = append(media.Files, file)
		case "NOTE":
			media.NoteXRefs, media.InlineNotes, media.Notes = appendRecordNote(record.Tags, i, media.NoteXRefs, media.InlineNotes, media.Notes)
		case "SNOTE":
			// Route shared-note pointers through the split-note path so they
			// reach the typed NoteXRefs API, the legacy Notes slice, and survive
			// re-encode. Also track them in SharedNoteXRefs, which records the
			// GEDCOM 7.0 SNOTE form used for version detection.
			media.NoteXRefs, media.InlineNotes, media.Notes = appendRecordNote(record.Tags, i, media.NoteXRefs, media.InlineNotes, media.Notes)
			media.SharedNoteXRefs = append(media.SharedNoteXRefs, tagToken(tag.Value))
		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			media.SourceCitations = append(media.SourceCitations, cite)
		case "CHAN":
			media.ChangeDate = parseChangeDate(record.Tags, i, collector)
		case "CREA":
			media.CreationDate = parseChangeDate(record.Tags, i, collector)
		case "REFN":
			media.RefNumbers = append(media.RefNumbers, tag.Value)
		case "UID":
			media.UIDs = append(media.UIDs, tag.Value)
		case "RESN":
			media.Restriction = tag.Value
		case "EXID":
			media.ExternalIDs = append(media.ExternalIDs, parseExternalID(record.Tags, i))
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return media
}

// parseMediaFile extracts a MediaFile from FILE tag and its subordinates.
func parseMediaFile(tags []*gedcom.Tag, fileIdx, baseLevel int, collector *diagnosticCollector) *gedcom.MediaFile {
	file := &gedcom.MediaFile{
		FileRef: tags[fileIdx].Value,
	}

	for i := fileIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "FORM":
				file.Form = tag.Value
				// Look for MEDI at baseLevel+2
				for j := i + 1; j < len(tags); j++ {
					mediTag := tags[j]
					if mediTag.Level <= baseLevel+1 {
						break
					}
					if mediTag.Level == baseLevel+2 && mediTag.Tag == "MEDI" {
						file.MediaType = mediTag.Value
						break
					}
				}
			case "TITL":
				file.Title = tag.Value
			case "TRAN":
				tran := parseMediaTranslation(tags, i, tag.Level, collector)
				file.Translations = append(file.Translations, tran)
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return file
}

// parseMediaTranslation extracts a MediaTranslation from TRAN tag and its subordinates.
func parseMediaTranslation(tags []*gedcom.Tag, tranIdx, baseLevel int, collector *diagnosticCollector) *gedcom.MediaTranslation {
	tran := &gedcom.MediaTranslation{
		FileRef: tags[tranIdx].Value,
	}

	for i := tranIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "FORM":
				tran.Form = tag.Value
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return tran
}

// parseMediaLink extracts a MediaLink from OBJE reference tag and its subordinates.
func parseMediaLink(tags []*gedcom.Tag, objeIdx, baseLevel int, collector *diagnosticCollector) *gedcom.MediaLink {
	link := &gedcom.MediaLink{
		MediaXRef: tagToken(tags[objeIdx].Value),
	}

	for i := objeIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "CROP":
				link.Crop = parseCropRegion(tags, i, tag.Level, collector)
			case "TITL":
				link.Title = tag.Value
			case "FILE":
				// Known tag for inline media references
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return link
}

// parseCropRegion extracts a CropRegion from CROP tag and its subordinates.
func parseCropRegion(tags []*gedcom.Tag, cropIdx, baseLevel int, collector *diagnosticCollector) *gedcom.CropRegion {
	crop := &gedcom.CropRegion{}

	for i := cropIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "TOP":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Top = v
				} else {
					collector.addInvalidValue(tag.LineNumber, "TOP", tag.Value, "expected integer")
				}
			case "LEFT":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Left = v
				} else {
					collector.addInvalidValue(tag.LineNumber, "LEFT", tag.Value, "expected integer")
				}
			case "HEIGHT":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Height = v
				} else {
					collector.addInvalidValue(tag.LineNumber, "HEIGHT", tag.Value, "expected integer")
				}
			case "WIDTH":
				if v, err := strconv.Atoi(tag.Value); err == nil {
					crop.Width = v
				} else {
					collector.addInvalidValue(tag.LineNumber, "WIDTH", tag.Value, "expected integer")
				}
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return crop
}
