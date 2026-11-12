package decoder

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/atsign"
	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
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

// diagnosticCollector is the parse context threaded through entity
// population. It accumulates diagnostics and, because it already reaches every
// parse function, also carries the GEDCOM versions that decide what counts as
// a pointer (isPointer) and how text is unescaped (text). Its methods accept a
// nil receiver, which collects nothing and knows no version.
type diagnosticCollector struct {
	diagnostics Diagnostics

	// version is the GEDCOM version the document's header declares, or ""
	// when it declares none this package recognizes.
	version gedcom.Version

	// textVersion is the version the document is read as, Header.Version: the
	// declared version, or the one detected when it declares none. The encoder
	// escapes a decoded document's text by that version, so the decoder must
	// unescape by it too, or a decode-encode round trip rewrites "@@" in a
	// file that declares no version.
	textVersion gedcom.Version

	// discard drops diagnostics instead of accumulating them, for a decode
	// that reports none but still needs version.
	discard bool

	// budget bounds the diagnostics kept; nil keeps them all.
	budget *diagnosticBudget

	// unknownTags counts, per tag name, the CodeUnknownTag diagnostics
	// addUnknownTag has seen, so it can stop reporting a tag one line at a
	// time once it passes MaxUnknownTagDiagnosticsPerTag (#730).
	unknownTags map[string]*unknownTagCount
}

// MaxUnknownTagDiagnosticsPerTag is the number of CodeUnknownTag diagnostics a
// lenient decode reports for one tag name before it stops reporting that tag
// line by line. Every line is still kept in the raw tags; the lines past the cap
// are counted in one closing CodeUnknownTag diagnostic per tag, at the line of
// the first of them, so an export that writes the same nonstandard tag on every
// event (MyHeritage's RIN, Brother's Keeper's QUAY) yields a handful of
// diagnostics rather than thousands that bury the rest. Unknown level-0 record
// types are not capped.
const MaxUnknownTagDiagnosticsPerTag = 10

// unknownTagCount is addUnknownTag's tally for one tag name.
type unknownTagCount struct {
	reported      int
	suppressed    int
	firstOverLine int
}

// quiet returns a collector that discards diagnostics but carries c's
// versions, for a parse whose diagnostics are reported elsewhere.
func (c *diagnosticCollector) quiet() *diagnosticCollector {
	q := &diagnosticCollector{discard: true}
	if c != nil {
		q.version = c.version
		q.textVersion = c.textVersion
	}
	return q
}

// collecting reports whether diagnostics added to c are kept. Helpers check
// it before building a diagnostic, so a decode that discards them does not
// pay for formatting each one.
func (c *diagnosticCollector) collecting() bool {
	return c != nil && !c.discard
}

// add appends a diagnostic to the collector if it is collecting, unless the
// budget is spent, in which case it is only counted.
func (c *diagnosticCollector) add(d Diagnostic) {
	if !c.collecting() {
		return
	}
	if c.budget != nil && !c.budget.keep(len(c.diagnostics), d.Severity) {
		return
	}
	c.diagnostics = append(c.diagnostics, d)
}

// text returns a typed text value read from a line: the value with the "@"
// escaping of the version the document is read as undone (atsign.Unescape),
// "@@" for "@" throughout in 5.5 and 5.5.1 and only at the start of a line in
// 7.0 (#708). A nil collector knows no version and takes the 7.0 rule. Every typed field that holds text
// is read through it, so it holds what the encoder's text escaping started
// from. Pointer fields are not -- a pointer is not text -- and neither is the
// pointer test that sorts a NOTE into NoteXRefs or InlineNotes, which runs on
// the line as written: "1 NOTE @@I1@" is the text "@I1@", not a link. The raw
// Tags keep the wire form.
func (c *diagnosticCollector) text(s string) string {
	var v gedcom.Version
	if c != nil {
		v = c.textVersion
	}
	return atsign.Unescape(s, v)
}

// isPointer reports whether a token-position value is an XRef pointer under
// the identifier grammar of the document's version: in GEDCOM 5.5 and 5.5.1 it
// may contain spaces (#579). A nil collector knows no version, and applies
// gedcom.IsPointerXRef.
func (c *diagnosticCollector) isPointer(s string) bool {
	var v gedcom.Version
	if c != nil {
		v = c.version
	}
	return gedcom.IsPointerXRefForVersion(s, v)
}

// addUnknownTag records an unknown tag diagnostic, or counts it once tag has
// been reported MaxUnknownTagDiagnosticsPerTag times (see unknownTagSummaries).
func (c *diagnosticCollector) addUnknownTag(lineNumber int, tag, content string) {
	if !c.collecting() {
		return
	}
	if c.unknownTags == nil {
		c.unknownTags = map[string]*unknownTagCount{}
	}
	count := c.unknownTags[tag]
	if count == nil {
		count = &unknownTagCount{}
		c.unknownTags[tag] = count
	}
	if count.reported < MaxUnknownTagDiagnosticsPerTag {
		count.reported++
		c.add(NewDiagnostic(
			lineNumber,
			SeverityWarning,
			CodeUnknownTag,
			fmt.Sprintf("unknown tag: %s", tag),
			content,
		))
		return
	}
	if count.suppressed == 0 {
		count.firstOverLine = lineNumber
	}
	count.suppressed++
}

// addUnknownTagOutside55 reports tag, a structure only GEDCOM 5.5 defines, as
// unknown in a document that declares another version. A document declaring no
// version this package recognizes is not reported.
//
// Only 5.5 structures that later versions moved or dropped are gated this way.
// Record identifiers that one version defines and another replaced (RIN, RFN,
// AFN; 7.0 CREA, REFN, UID, INIL) are accepted in every version, as on INDI
// since #375: they carry the same meaning wherever they appear, and real
// exports mix them across versions.
func (c *diagnosticCollector) addUnknownTagOutside55(tag *gedcom.Tag) {
	if c != nil && c.version != "" && c.version != gedcom.Version55 {
		c.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
	}
}

// unknownTagSummaries returns one CodeUnknownTag diagnostic for each tag whose
// lines passed MaxUnknownTagDiagnosticsPerTag, at the line of the first line
// not reported individually, counting those lines (that one included). They
// are ordered by that line, so the result does not depend on map order.
func (c *diagnosticCollector) unknownTagSummaries() Diagnostics {
	if !c.collecting() {
		return nil
	}
	var out Diagnostics
	for tag, count := range c.unknownTags {
		if count.suppressed == 0 {
			continue
		}
		out = append(out, NewDiagnostic(
			count.firstOverLine,
			SeverityWarning,
			CodeUnknownTag,
			fmt.Sprintf("unknown tag: %s (%d more lines with this tag, from this line on, not reported individually)", tag, count.suppressed),
			"",
		))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

// addInvalidValue records an invalid value diagnostic.
func (c *diagnosticCollector) addInvalidValue(lineNumber int, tag, value, reason string) {
	if c.collecting() {
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
// It checks ctx before each record and returns ctx's error once it is done.
func populateEntities(ctx context.Context, doc *gedcom.Document, collector *diagnosticCollector) error {
	for _, record := range doc.Records {
		if err := ctxcheck.Err(ctx); err != nil {
			return err
		}
		populateRecordEntity(record, collector)
	}
	return nil
}

// populateRecordEntity sets record.Entity to the entity its type parses into
// from its Tags. A record type with no entity is left as it is.
func populateRecordEntity(record *gedcom.Record, collector *diagnosticCollector) {
	switch record.Type {
	case gedcom.RecordTypeIndividual:
		record.Entity = parseIndividual(record, collector)
	case gedcom.RecordTypeFamily:
		record.Entity = parseFamily(record, collector)
	case gedcom.RecordTypeSource:
		record.Entity = parseSource(record, collector)
	case gedcom.RecordTypeSubmitter:
		record.Entity = parseSubmitter(record, collector)
	case gedcom.RecordTypeSubmission:
		record.Entity = parseSubmission(record, collector)
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
			indi.Sex = collector.text(tagToken(tag.Value))

		case "BIRT", "DEAT", "BAPM", "BURI", "CENS", "CHR", "ADOP", "RESI", "IMMI", "EMIG",
			"BARM", "BASM", "BLES", "CHRA", "CONF", "FCOM",
			"GRAD", "RETI", "NATU", "ORDN", "PROB", "WILL", "CREM", "EVEN":
			event := parseEvent(record.Tags, i, tag.Tag, individualOwner, collector)
			indi.Events = append(indi.Events, event)

		case "NO":
			// GEDCOM 7.0: NO tag indicates event did not occur
			// tag.Value contains the event type (e.g., "MARR", "DEAT")
			if strings.TrimSpace(tag.Value) == "" {
				collector.addInvalidValue(tag.LineNumber, "NO", tag.Value, "missing event type")
				continue
			}
			event := parseEvent(record.Tags, i, tag.Value, individualOwner, collector)
			event.IsNegative = true
			indi.Events = append(indi.Events, event)

		case "BAPL", "CONL", "ENDL", "SLGC":
			ord := parseLDSOrdinance(record.Tags, i, ldsOrdinanceType(tag.Tag), collector)
			indi.LDSOrdinances = append(indi.LDSOrdinances, ord)

		case "OCCU", "CAST", "DSCR", "EDUC", "IDNO", "NATI", "SSN", "TITL", "RELI", "NCHI", "NMR", "PROP", "FACT":
			attr := parseAttribute(record.Tags, i, tag.Tag, individualOwner, collector)
			indi.Attributes = append(indi.Attributes, attr)

		case "FAMC":
			famLink := parseFamilyLink(record.Tags, i, "FAMC", collector)
			indi.ChildInFamilies = append(indi.ChildInFamilies, famLink)

		case "FAMS":
			famLink := parseFamilyLink(record.Tags, i, "FAMS", collector)
			indi.SpouseInFamilies = append(indi.SpouseInFamilies, famLink)

		case "ASSO":
			assoc := parseAssociation(record.Tags, i, collector)
			indi.Associations = append(indi.Associations, assoc)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			indi.SourceCitations = append(indi.SourceCitations, cite)

		case "NOTE", "SNOTE":
			indi.NoteXRefs, indi.InlineNotes = appendRecordNote(record.Tags, i, indi.NoteXRefs, indi.InlineNotes, collector)

		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level, collector)
			indi.Media = append(indi.Media, link)

		case "CHAN":
			indi.ChangeDate = parseChangeDate(record.Tags, i, collector)

		case "CREA":
			indi.CreationDate = parseChangeDate(record.Tags, i, collector)

		case "REFN":
			indi.RefNumbers = append(indi.RefNumbers, parseRefNumber(record.Tags, i, collector))

		case "UID", "_UID":
			// _UID is the vendor spelling of UID every 5.5.x desktop producer
			// agrees on (#715); see uidTagName in the encoder for the write side.
			indi.UIDs = append(indi.UIDs, collector.text(tag.Value))

		case "_FSFTID":
			// Text like any other: a leading "@@" (written by the encoder, or
			// by the EXID->_FSFTID converter's gedcom.EscapeLeadingAt) is a
			// literal "@", so FamilySearchID / FamilySearchURL() get the
			// logical id; the raw tag retains "@@".
			indi.FamilySearchID = collector.text(tag.Value)

		case "EXID":
			indi.ExternalIDs = append(indi.ExternalIDs, parseExternalID(record.Tags, i, collector))

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
		Full: collector.text(tags[nameIdx].Value),
	}

	// Parse the full name to extract given and surname
	// GEDCOM format: "Given /Surname/"
	full := collector.text(tags[nameIdx].Value)
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
				name.Given = collector.text(tagToken(tag.Value))
			case "SURN":
				name.Surname = collector.text(tagToken(tag.Value))
			case "NPFX":
				name.Prefix = collector.text(tagToken(tag.Value))
			case "NSFX":
				name.Suffix = collector.text(tagToken(tag.Value))
			case "NICK":
				name.Nickname = collector.text(tagToken(tag.Value))
			case "SPFX":
				name.SurnamePrefix = collector.text(tagToken(tag.Value))
			case "_MARNM":
				name.MarriedName = collector.text(tagToken(tag.Value))
			case "TYPE":
				name.Type = collector.text(tag.Value)
				name.TypePhrase = collector.text(findSubordinate(tags, i, "PHRASE"))
			case "TRAN":
				tran := parseNameTransliteration(tags, i, collector)
				name.Transliterations = append(name.Transliterations, tran)
			case "SOUR":
				cite := parseSourceCitation(tags, i, tag.Level, collector)
				name.SourceCitations = append(name.SourceCitations, cite)
			case "NOTE", "SNOTE":
				name.NoteXRefs, name.InlineNotes = appendRecordNote(
					tags, i, name.NoteXRefs, name.InlineNotes, collector)
			case "FONE":
				name.Phonetic = append(name.Phonetic, parseNameVariation(tags, i, collector))
			case "ROMN":
				name.Romanized = append(name.Romanized, parseNameVariation(tags, i, collector))
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
		Value: collector.text(tags[tranIdx].Value),
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
				tran.Language = collector.text(tag.Value)
			case "GIVN":
				tran.Given = collector.text(tag.Value)
			case "SURN":
				tran.Surname = collector.text(tag.Value)
			case "NPFX":
				tran.Prefix = collector.text(tag.Value)
			case "NSFX":
				tran.Suffix = collector.text(tag.Value)
			case "NICK":
				tran.Nickname = collector.text(tag.Value)
			case "SPFX":
				tran.SurnamePrefix = collector.text(tag.Value)
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return tran
}

// parseNameVariation extracts a GEDCOM 5.5.1 FONE or ROMN name variation from
// tags starting at varIdx.
func parseNameVariation(tags []*gedcom.Tag, varIdx int, collector *diagnosticCollector) *gedcom.NameVariation {
	baseLevel := tags[varIdx].Level

	variation := &gedcom.NameVariation{
		Value: collector.text(tags[varIdx].Value),
	}

	for i := varIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "TYPE":
				variation.Type = collector.text(tag.Value)
			case "GIVN":
				variation.Given = collector.text(tagToken(tag.Value))
			case "SURN":
				variation.Surname = collector.text(tagToken(tag.Value))
			case "NPFX":
				variation.Prefix = collector.text(tagToken(tag.Value))
			case "NSFX":
				variation.Suffix = collector.text(tagToken(tag.Value))
			case "NICK":
				variation.Nickname = collector.text(tagToken(tag.Value))
			case "SPFX":
				variation.SurnamePrefix = collector.text(tagToken(tag.Value))
			case "NOTE", "SOUR":
				// Name-piece notes and citations, kept only in the raw tags
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return variation
}

// parseFamilyLink extracts a family link from tags starting at linkIdx.
//
// tagName is the link tag being parsed, "FAMC" or "FAMS". The two share the
// FAM-LINK shape in both 5.5.1 and 7.0, so one parser serves both; they differ
// in a single substructure, PEDI, which only a child link may carry.
func parseFamilyLink(tags []*gedcom.Tag, linkIdx int, tagName string, collector *diagnosticCollector) gedcom.FamilyLink {
	famLink := gedcom.FamilyLink{
		FamilyXRef: tagToken(tags[linkIdx].Value),
	}

	// Look for subordinate tags (level 2)
	for i := linkIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "PEDI":
				// A spouse link has no pedigree in either spec. The value is
				// still stored and re-encoded, per the lossless-representation
				// principle -- but it takes the recognized branch, so without
				// this diagnostic a caller auditing file quality would see a
				// clean FAMS while carrying a value invalid in 5.5.1 and 7.0.
				if tagName == "FAMS" {
					collector.addInvalidValue(tag.LineNumber, "PEDI", tag.Value,
						"not valid under FAMS; a spouse link has no pedigree")
				}
				famLink.Pedigree = collector.text(tag.Value)
				famLink.PedigreePhrase = collector.text(findSubordinate(tags, i, "PHRASE"))
			case "NOTE", "SNOTE":
				famLink.NoteXRefs, famLink.InlineNotes = appendRecordNote(
					tags, i, famLink.NoteXRefs, famLink.InlineNotes, collector)
			case "STAT":
				// Known tag not yet parsed into a typed field
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
				assoc.Role = collector.text(tag.Value)
				assoc.RolePhrase = collector.text(findSubordinate(tags, i, "PHRASE"))
			case "TYPE":
				// The associated record's type, defined by 5.5 alone. It is
				// kept whatever the version, and reported as before in a
				// document declaring another (5.5-style exports such as
				// Geneanet's 5.5.1 write it).
				assoc.RecordType = collector.text(tag.Value)
				collector.addUnknownTagOutside55(tag)
			case "PHRASE":
				assoc.Phrase = collector.text(tag.Value)
			case "NOTE", "SNOTE":
				assoc.NoteXRefs, assoc.InlineNotes = appendRecordNote(
					tags, i, assoc.NoteXRefs, assoc.InlineNotes, collector)
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

// parseRefNumber parses a REFN tag with its optional TYPE subordinate.
func parseRefNumber(tags []*gedcom.Tag, refnIdx int, collector *diagnosticCollector) gedcom.RefNumber {
	return gedcom.RefNumber{
		Value: collector.text(tags[refnIdx].Value),
		Type:  collector.text(findSubordinate(tags, refnIdx, "TYPE")),
	}
}

// parseExternalID parses an EXID tag with optional TYPE subordinate.
func parseExternalID(tags []*gedcom.Tag, exidIdx int, collector *diagnosticCollector) *gedcom.ExternalID {
	baseLevel := tags[exidIdx].Level
	exid := &gedcom.ExternalID{
		Value: collector.text(tags[exidIdx].Value),
	}

	// Look for TYPE subordinate at baseLevel+1
	for i := exidIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 && tag.Tag == "TYPE" {
			exid.Type = collector.text(tag.Value)
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
				cite.Page = collector.text(foldedValue(tags, i))
			case "QUAY":
				cite.Quality = parseQuality(tag, collector)
			case "DATA":
				// Parse DATA subordinates at baseLevel+2
				cite.Data = parseSourceCitationData(tags, i, baseLevel+1, collector)
			case "_APID":
				// Parse Ancestry Permanent Identifier (vendor extension)
				cite.AncestryAPID = gedcom.ParseAPID(collector.text(tag.Value))
			case "NOTE", "SNOTE":
				cite.NoteXRefs, cite.InlineNotes = appendRecordNote(tags, i, cite.NoteXRefs, cite.InlineNotes, collector)
			case "EVEN":
				cite.Event = parseSourceCitationEvent(tags, i, collector)
			case "OBJE":
				cite.Media = append(cite.Media, parseMediaLink(tags, i, tag.Level, collector))
			case "TEXT":
				// Known tag not yet parsed into a typed field (the 5.5/5.5.1
				// inline-description form of a citation)
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return cite
}

// parseQuality parses a citation's QUAY value as an integer 0-3, reporting any
// other value. QUAY 0 is a real assertion, so the result is non-nil for every
// in-range value.
func parseQuality(tag *gedcom.Tag, collector *diagnosticCollector) *int {
	q, err := strconv.Atoi(tag.Value)
	if err != nil || q < 0 || q > 3 {
		collector.addInvalidValue(tag.LineNumber, "QUAY", tag.Value, "expected integer 0-3")
		return nil
	}
	return &q
}

// parseSourceCitationEvent extracts the EVEN substructure of a source citation
// starting at evenIdx: the event type cited from, its GEDCOM 7.0 PHRASE, and
// the ROLE (with its PHRASE) the subject played in it.
func parseSourceCitationEvent(tags []*gedcom.Tag, evenIdx int, collector *diagnosticCollector) *gedcom.SourceCitationEvent {
	baseLevel := tags[evenIdx].Level
	event := &gedcom.SourceCitationEvent{Type: collector.text(tags[evenIdx].Value)}

	for i := evenIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level != baseLevel+1 {
			continue
		}
		switch tag.Tag {
		case "PHRASE":
			event.Phrase = collector.text(tag.Value)
		case "ROLE":
			event.Role = collector.text(tag.Value)
			event.RolePhrase = collector.text(findSubordinate(tags, i, "PHRASE"))
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return event
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
				data.Date = collector.text(tag.Value)
			case "TEXT":
				// TEXT is {0:M}: append, never overwrite, so every passage
				// survives in document order (issue #497).
				data.Text = append(data.Text, parseSourceText(tags, i, collector))
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return data
}

// parseSourceText extracts one DATA.TEXT substructure of a source citation
// starting at textIdx. CONT/CONC continuation lines fold into Value (issue
// #442), and the GEDCOM 7.0 MIME and LANG qualifiers are typed.
func parseSourceText(tags []*gedcom.Tag, textIdx int, collector *diagnosticCollector) *gedcom.SourceText {
	// foldedValue reuses tag.Value when there is no continuation, which saves
	// a copy and an allocation per citation (measured by
	// BenchmarkDecodeCitationHeavy).
	text := &gedcom.SourceText{Value: collector.text(foldedValue(tags, textIdx))}
	text.MIME, text.Language = parseTextQualifiers(tags, textIdx, collector)
	return text
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
	owner           detailOwner
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
	uids            *[]string
	sortDate        *string
	sortDatePhrase  *string
	sourceCitations *[]*gedcom.SourceCitation
	media           *[]*gedcom.MediaLink
	associations    *[]*gedcom.Association
	noteXRefs       *[]string
	inlineNotes     *[]string
}

// detailOwner names the record an event or attribute belongs to. It decides
// whether FAMILY_EVENT_DETAIL's HUSB/WIFE are in grammar: the spec defines them
// under every family event and family attribute, and under no individual one.
// The zero value is reserved so an eventDetail built without an owner matches
// neither constant, and the HUSB/WIFE check stays silent rather than flagging.
type detailOwner int

const (
	_ detailOwner = iota
	individualOwner
	familyOwner
)

// eventDetailOf returns the EVENT_DETAIL view of an event.
func eventDetailOf(e *gedcom.Event, owner detailOwner) eventDetail {
	return eventDetail{
		owner:           owner,
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
		uids:            &e.UIDs,
		sortDate:        &e.SortDate,
		sortDatePhrase:  &e.SortDatePhrase,
		sourceCitations: &e.SourceCitations,
		media:           &e.Media,
		associations:    &e.Associations,
		noteXRefs:       &e.NoteXRefs,
		inlineNotes:     &e.InlineNotes,
	}
}

// eventDetailOfAttribute returns the EVENT_DETAIL view of an attribute.
func eventDetailOfAttribute(a *gedcom.Attribute, owner detailOwner) eventDetail {
	return eventDetail{
		owner:           owner,
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
		uids:            &a.UIDs,
		sortDate:        &a.SortDate,
		sortDatePhrase:  &a.SortDatePhrase,
		sourceCitations: &a.SourceCitations,
		media:           &a.Media,
		associations:    &a.Associations,
		noteXRefs:       &a.NoteXRefs,
		inlineNotes:     &a.InlineNotes,
	}
}

// parseEventDetailTag decodes the EVENT_DETAIL subtag at tags[i] into detail.
// It covers every substructure gedcom.Event and gedcom.Attribute share; the
// tags that belong to only one of them (TYPE, and AGE, which is typed on an
// event and not on an attribute) stay with their own parser, which delegates
// here for the rest.
// Unrecognized standard tags are reported to collector (ADR 0007).
//
// The switch branches on the owning record, never on event versus attribute:
// HUSB/WIFE (FAMILY_EVENT_DETAIL) are in grammar under a family's events and
// attributes alike, and under neither for an individual (issue #475).
//
//nolint:gocyclo // GEDCOM parsing inherently requires handling many tag types
func parseEventDetailTag(detail *eventDetail, tags []*gedcom.Tag, i int, collector *diagnosticCollector) {
	tag := tags[i]
	switch tag.Tag {
	case "DATE":
		*detail.date = collector.text(tag.Value)
		if parsed := parseDateValue(tags, i, collector); parsed != nil {
			*detail.parsedDate = parsed
		}
	case "PLAC":
		*detail.placeDetail = parsePlaceDetail(tags, i, tag.Level, collector)
	case "CAUS":
		*detail.cause = collector.text(tag.Value)
	case "AGNC":
		*detail.agency = collector.text(tag.Value)
	case "RELI":
		*detail.religion = collector.text(tag.Value)
	case "ADDR":
		*detail.address = parseEventAddress(tags, i, tag.Level, collector)
	case "PHON":
		*detail.phone = append(*detail.phone, collector.text(tag.Value))
	case "EMAIL":
		*detail.email = append(*detail.email, collector.text(tag.Value))
	case "FAX":
		*detail.fax = append(*detail.fax, collector.text(tag.Value))
	case "WWW":
		*detail.website = append(*detail.website, collector.text(tag.Value))
	case "RESN":
		*detail.restriction = collector.text(tag.Value)
	case "UID", "_UID":
		*detail.uids = append(*detail.uids, collector.text(tag.Value))
	case "SDATE":
		*detail.sortDate = collector.text(tag.Value)
		*detail.sortDatePhrase = collector.text(findSubordinate(tags, i, "PHRASE"))
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
		*detail.noteXRefs, *detail.inlineNotes = appendRecordNote(
			tags, i, *detail.noteXRefs, *detail.inlineNotes, collector)
	case "HUSB", "WIFE":
		// FAMILY_EVENT_DETAIL: the spouse ages on a family event or family
		// attribute (maximal70.ged carries one under FAM.FACT). Known tags not
		// yet parsed into typed fields. An individual's events and attributes
		// have no such position, so there the tag is nonstandard.
		if detail.owner == individualOwner {
			collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
		}
	default:
		if !strings.HasPrefix(tag.Tag, "_") {
			collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
		}
	}
}

// parseDateValue parses the DATE at tags[i], with its PHRASE (7.0) — the
// date in prose — into a typed Date. When the payload does not parse, it
// reports the raw value (which stays in Tags) and returns nil, unless a
// PHRASE carries the date.
//
// A PHRASE is kept whatever the value: beside a value that does not parse
// (including the empty DateValue 7.0 permits when a PHRASE carries the date),
// it yields a phrase-only Date — IsPhrase set, Original the raw value, no
// date components — so the phrase reaches the typed model. An empty value is
// valid there and is not reported. An empty PHRASE is ignored rather than
// assigned, because ParseDate itself fills Phrase for a parenthesized date
// phrase.
func parseDateValue(tags []*gedcom.Tag, i int, collector *diagnosticCollector) *gedcom.Date {
	tag := tags[i]
	value := collector.text(tag.Value)
	phrase := collector.text(findSubordinate(tags, i, "PHRASE"))
	parsed, err := gedcom.ParseDate(value)
	if err == nil {
		if phrase != "" {
			parsed.Phrase = phrase
		}
		return parsed
	}
	if tag.Value != "" || phrase == "" {
		collector.addInvalidValue(tag.LineNumber, "DATE", tag.Value, err.Error())
	}
	if phrase == "" {
		return nil
	}
	return &gedcom.Date{Original: value, Calendar: gedcom.CalendarGregorian, Phrase: phrase, IsPhrase: true}
}

// parseEvent extracts an event from tags starting at eventIdx.
func parseEvent(tags []*gedcom.Tag, eventIdx int, eventTag string, owner detailOwner, collector *diagnosticCollector) *gedcom.Event {
	event := &gedcom.Event{
		Type: gedcom.EventType(eventTag),
	}

	// A generic event carries its descriptor as the EVEN line's own payload;
	// named event tags carry the [Y|<NULL>] flag instead, which has no typed
	// field. Guard on the raw tag so "NO EVEN" (7.0 negative assertion, where
	// eventTag comes from the NO value) is not mistaken for a descriptor.
	if tags[eventIdx].Tag == "EVEN" {
		event.Description = collector.text(tags[eventIdx].Value)
	}

	baseLevel := tags[eventIdx].Level
	detail := eventDetailOf(event, owner)

	// Look for subordinate tags at baseLevel+1
	for i := eventIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "TYPE":
				event.EventTypeDetail = collector.text(tag.Value)
			case "AGE":
				event.Age = collector.text(tag.Value)
				event.AgePhrase = collector.text(findSubordinate(tags, i, "PHRASE"))
			case "FAMC":
				if !parseEventFamilyLink(event, tags, i, tags[eventIdx].Tag, owner, collector) {
					parseEventDetailTag(&detail, tags, i, collector)
				}
			default:
				// Everything else is EVENT_DETAIL, shared with parseAttribute.
				parseEventDetailTag(&detail, tags, i, collector)
			}
		}
	}

	return event
}

// parseEventFamilyLink types the FAMC at tags[famcIdx] beneath an
// individual's BIRT, CHR or ADOP event (the only events every version gives
// one), with the ADOP and PHRASE beneath it on an ADOP event. eventTag is the
// event line's own tag, so a 7.0 "NO BIRT" is not taken for a birth. It
// reports false for any other position, which the caller treats as an
// unknown tag.
func parseEventFamilyLink(event *gedcom.Event, tags []*gedcom.Tag, famcIdx int, eventTag string, owner detailOwner, collector *diagnosticCollector) bool {
	if owner != individualOwner || (eventTag != "BIRT" && eventTag != "CHR" && eventTag != "ADOP") {
		return false
	}
	event.FamilyXRef = tagToken(tags[famcIdx].Value)
	if eventTag == "ADOP" {
		if adopIdx := findSubordinateIndex(tags, famcIdx, "ADOP"); adopIdx >= 0 {
			event.AdoptedBy = collector.text(tags[adopIdx].Value)
			event.AdoptedByPhrase = collector.text(findSubordinate(tags, adopIdx, "PHRASE"))
		}
	}
	return true
}

// parseEventAddress extracts an address structure from tags starting at addrIdx.
// The ADDR payload and its CONT/CONC lines fill Full; ADR1-ADR3 fill Line1-Line3
// alone, so neither overwrites the other.
func parseEventAddress(tags []*gedcom.Tag, addrIdx, baseLevel int, collector *diagnosticCollector) *gedcom.Address {
	// A payload already folded into one value (Tags rewritten by the
	// converter, #772) drops its leading empty lines, as the CONT case below
	// does for the same lines unfolded. A value read from a line has none.
	addr := &gedcom.Address{
		Full: strings.TrimLeft(collector.text(tags[addrIdx].Value), "\n"),
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
				addr.Line1 = collector.text(tag.Value)
			case "ADR2":
				addr.Line2 = collector.text(tag.Value)
			case "ADR3":
				addr.Line3 = collector.text(tag.Value)
			case "CITY":
				addr.City = collector.text(tag.Value)
			case "STAE":
				addr.State = collector.text(tag.Value)
			case "POST":
				addr.PostalCode = collector.text(tag.Value)
			case "CTRY":
				addr.Country = collector.text(tag.Value)
			case "CONT":
				// Continue the payload on the next line. Not foldContinuation: an
				// empty first line takes the value directly, with no leading
				// newline.
				if addr.Full != "" {
					addr.Full += "\n" + collector.text(tag.Value)
				} else {
					addr.Full = collector.text(tag.Value)
				}
			case "CONC":
				addr.Full += collector.text(tag.Value)
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
		Name: collector.text(tags[placIdx].Value),
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
				place.Form = collector.text(tag.Value)
			case "MAP":
				place.Coordinates = parseCoordinates(tags, i, tag.Level, collector)
			case "NOTE", "SNOTE":
				place.NoteXRefs, place.InlineNotes = appendRecordNote(
					tags, i, place.NoteXRefs, place.InlineNotes, collector)
			case "FONE", "ROMN", "TRAN", "EXID", "LANG":
				// Known tags not yet parsed into typed fields
			case "SOUR":
				// The 5.5 citation of the place name, kept raw and reported
				// in a document declaring another version (#730).
				collector.addUnknownTagOutside55(tag)
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
				coords.Latitude = collector.text(tag.Value)
				validateCoordinate(tag, gedcom.ParseLatitude, collector)
			case "LONG":
				coords.Longitude = collector.text(tag.Value)
				validateCoordinate(tag, gedcom.ParseLongitude, collector)
			default:
				if !strings.HasPrefix(tag.Tag, "_") {
					collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
				}
			}
		}
	}

	return coords
}

// validateCoordinate records a diagnostic when a LATI or LONG value is present
// but parse rejects it. The raw value is retained by the caller either way
// (ADR 0003): the diagnostic is the signal, not a dropped value.
//
// parse is gedcom.ParseLatitude or gedcom.ParseLongitude, so a swapped axis
// ("LATI E42.3601") is reported here rather than staying silent until some
// caller reaches Coordinates.AsDecimal — which a caller parsing one component
// at a time never does.
func validateCoordinate(tag *gedcom.Tag, parse func(string) (float64, error), collector *diagnosticCollector) {
	if tag.Value == "" {
		return
	}
	if _, err := parse(tag.Value); err != nil {
		collector.addInvalidValue(tag.LineNumber, tag.Tag, tag.Value, err.Error())
	}
}

// parseAttribute extracts an attribute from tags starting at attrIdx.
func parseAttribute(tags []*gedcom.Tag, attrIdx int, attrTag string, owner detailOwner, collector *diagnosticCollector) *gedcom.Attribute {
	attr := &gedcom.Attribute{
		Type:  gedcom.AttributeType(attrTag),
		Value: collector.text(tags[attrIdx].Value),
	}

	baseLevel := tags[attrIdx].Level
	detail := eventDetailOfAttribute(attr, owner)

	// Look for subordinate tags at baseLevel+1
	for i := attrIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "TYPE":
				attr.TypeDetail = collector.text(tag.Value)
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
				ord.Date = collector.text(tag.Value)
				if parsed := parseDateValue(tags, i, collector); parsed != nil {
					ord.ParsedDate = parsed
				}
			case "TEMP":
				ord.Temple = collector.text(tag.Value)
			case "PLAC":
				ord.Place = collector.text(tag.Value)
			case "STAT":
				ord.Status = collector.text(tag.Value)
				if dateIdx := findSubordinateIndex(tags, i, "DATE"); dateIdx >= 0 {
					ord.StatusDate = collector.text(tags[dateIdx].Value)
					ord.StatusTime = collector.text(findSubordinate(tags, dateIdx, "TIME"))
				}
			case "FAMC":
				ord.FamilyXRef = tagToken(tag.Value)
			case "NOTE", "SNOTE":
				ord.NoteXRefs, ord.InlineNotes = appendRecordNote(tags, i, ord.NoteXRefs, ord.InlineNotes, collector)
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
			fam.HusbandPhrase = collector.text(findSubordinate(record.Tags, i, "PHRASE"))

		case "WIFE":
			fam.Wife = tagToken(tag.Value)
			fam.WifePhrase = collector.text(findSubordinate(record.Tags, i, "PHRASE"))

		case "CHIL":
			child := tagToken(tag.Value)
			fam.Children = append(fam.Children, child)
			father := collector.text(tagToken(findSubordinate(record.Tags, i, "_FREL")))
			mother := collector.text(tagToken(findSubordinate(record.Tags, i, "_MREL")))
			if father != "" || mother != "" {
				fam.ChildLinks = append(fam.ChildLinks,
					gedcom.ChildLink{XRef: child, FatherRelation: father, MotherRelation: mother})
			}

		case "NCHI", "FACT":
			attr := parseAttribute(record.Tags, i, tag.Tag, familyOwner, collector)
			fam.Attributes = append(fam.Attributes, attr)

		case "MARR", "DIV", "ENGA", "ANUL", "MARB", "MARC", "MARL", "MARS", "DIVF", "CENS", "RESI", "EVEN":
			event := parseEvent(record.Tags, i, tag.Tag, familyOwner, collector)
			fam.Events = append(fam.Events, event)

		case "NO":
			// GEDCOM 7.0: NO tag indicates event did not occur
			// tag.Value contains the event type (e.g., "MARR", "DIV")
			if strings.TrimSpace(tag.Value) == "" {
				collector.addInvalidValue(tag.LineNumber, "NO", tag.Value, "missing event type")
				continue
			}
			event := parseEvent(record.Tags, i, tag.Value, familyOwner, collector)
			event.IsNegative = true
			fam.Events = append(fam.Events, event)

		case "SLGS":
			ord := parseLDSOrdinance(record.Tags, i, ldsOrdinanceType(tag.Tag), collector)
			fam.LDSOrdinances = append(fam.LDSOrdinances, ord)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			fam.SourceCitations = append(fam.SourceCitations, cite)

		case "NOTE", "SNOTE":
			fam.NoteXRefs, fam.InlineNotes = appendRecordNote(record.Tags, i, fam.NoteXRefs, fam.InlineNotes, collector)

		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level, collector)
			fam.Media = append(fam.Media, link)

		case "CHAN":
			fam.ChangeDate = parseChangeDate(record.Tags, i, collector)

		case "CREA":
			fam.CreationDate = parseChangeDate(record.Tags, i, collector)

		case "REFN":
			fam.RefNumbers = append(fam.RefNumbers, parseRefNumber(record.Tags, i, collector))

		case "UID", "_UID":
			fam.UIDs = append(fam.UIDs, collector.text(tag.Value))

		case "EXID":
			fam.ExternalIDs = append(fam.ExternalIDs, parseExternalID(record.Tags, i, collector))

		case "RESN", "SUBM", "ASSO", "RIN":
			// Known tags not yet parsed into typed fields; the raw tags remain
			// on Family.Tags (ADR 0003). Recognizing them keeps UNKNOWN_TAG
			// meaning "nonstandard tag" for callers who filter on it (#375).
			// RIN is the 5.5/5.5.1 automated record ID every record may carry
			// (#730).
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
			src.Title = collector.text(foldedValue(record.Tags, i))
		case "AUTH":
			src.Author = collector.text(foldedValue(record.Tags, i))
		case "PUBL":
			src.Publication = collector.text(foldedValue(record.Tags, i))
		case "ABBR":
			src.Abbreviation = collector.text(tag.Value)
		case "TEXT":
			src.Text = collector.text(foldedValue(record.Tags, i))
			src.TextMIME, src.TextLanguage = parseTextQualifiers(record.Tags, i, collector)
		case "DATA":
			src.Data = parseSourceData(record.Tags, i, collector)
		case "REPO":
			src.RepositoryLinks = append(src.RepositoryLinks, parseSourceRepositoryLink(record.Tags, i, collector))
		case "NOTE", "SNOTE":
			src.NoteXRefs, src.InlineNotes = appendRecordNote(record.Tags, i, src.NoteXRefs, src.InlineNotes, collector)
		case "OBJE":
			link := parseMediaLink(record.Tags, i, tag.Level, collector)
			src.Media = append(src.Media, link)
		case "CHAN":
			src.ChangeDate = parseChangeDate(record.Tags, i, collector)
		case "CREA":
			src.CreationDate = parseChangeDate(record.Tags, i, collector)
		case "REFN":
			src.RefNumbers = append(src.RefNumbers, parseRefNumber(record.Tags, i, collector))
		case "UID", "_UID":
			src.UIDs = append(src.UIDs, collector.text(tag.Value))
		case "EXID":
			src.ExternalIDs = append(src.ExternalIDs, parseExternalID(record.Tags, i, collector))
		case "RIN":
			// Known tag not yet parsed into a typed field (#730)
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return src
}

// parseTextQualifiers reads the GEDCOM 7.0 MIME and LANG qualifiers of the
// TEXT tag at textIdx, diagnosing any other non-continuation subordinate.
func parseTextQualifiers(tags []*gedcom.Tag, textIdx int, collector *diagnosticCollector) (mime, lang string) {
	baseLevel := tags[textIdx].Level
	for i := textIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level != baseLevel+1 {
			continue
		}
		switch tag.Tag {
		case "CONT", "CONC":
			// Folded into the value by the caller.
		case "MIME":
			mime = collector.text(tag.Value)
		case "LANG":
			lang = collector.text(tag.Value)
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}
	return mime, lang
}

// parseSourceData extracts the DATA substructure of a source record starting
// at dataIdx: the events recorded (EVEN, each with DATE and PLAC), the
// responsible agency (AGNC), and notes.
func parseSourceData(tags []*gedcom.Tag, dataIdx int, collector *diagnosticCollector) *gedcom.SourceData {
	baseLevel := tags[dataIdx].Level
	data := &gedcom.SourceData{}

	for i := dataIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level != baseLevel+1 {
			continue
		}
		switch tag.Tag {
		case "EVEN":
			data.Events = append(data.Events, parseSourceDataEvent(tags, i, collector))
		case "AGNC":
			data.Agency = collector.text(tag.Value)
		case "NOTE", "SNOTE":
			data.NoteXRefs, data.InlineNotes = appendRecordNote(tags, i, data.NoteXRefs, data.InlineNotes, collector)
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return data
}

// parseSourceDataEvent extracts one DATA.EVEN substructure of a source record
// starting at evenIdx: the recorded event types, the period (DATE, with its
// GEDCOM 7.0 PHRASE) and the jurisdiction (PLAC) they cover.
func parseSourceDataEvent(tags []*gedcom.Tag, evenIdx int, collector *diagnosticCollector) *gedcom.SourceDataEvent {
	baseLevel := tags[evenIdx].Level
	event := &gedcom.SourceDataEvent{Types: collector.text(tags[evenIdx].Value)}

	for i := evenIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level != baseLevel+1 {
			continue
		}
		switch tag.Tag {
		case "DATE":
			event.Date = collector.text(tag.Value)
			event.DatePhrase = collector.text(findSubordinate(tags, i, "PHRASE"))
		case "PLAC":
			event.PlaceDetail = parsePlaceDetail(tags, i, tag.Level, collector)
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return event
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
				link.Inline.Name = collector.text(tag.Value)
			}
		case "CALN":
			link.CallNumbers = append(link.CallNumbers, parseCallNumber(tags, i, collector))
		case "NOTE", "SNOTE":
			link.NoteXRefs, link.InlineNotes = appendRecordNote(
				tags, i, link.NoteXRefs, link.InlineNotes, collector)
		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return link
}

// parseCallNumber extracts one CALN entry of a REPO link, starting at calnIdx,
// together with its MEDI media type and (GEDCOM 7.0) the PHRASE under MEDI.
// MEDI and PHRASE are the only substructures any supported GEDCOM version
// defines under CALN, each 0:1; the first of each is typed, and anything else
// stays in the raw tags.
func parseCallNumber(tags []*gedcom.Tag, calnIdx int, collector *diagnosticCollector) *gedcom.CallNumber {
	caln := &gedcom.CallNumber{Value: collector.text(tags[calnIdx].Value)}
	if mediIdx := findSubordinateIndex(tags, calnIdx, "MEDI"); mediIdx >= 0 {
		caln.MediaType = collector.text(tags[mediIdx].Value)
		caln.MediaPhrase = collector.text(findSubordinate(tags, mediIdx, "PHRASE"))
	}
	return caln
}

// findSubordinate returns the value of the first direct child tag of the tag at
// parentIdx that matches name, or "" if none exists.
func findSubordinate(tags []*gedcom.Tag, parentIdx int, name string) string {
	if idx := findSubordinateIndex(tags, parentIdx, name); idx >= 0 {
		return tags[idx].Value
	}
	return ""
}

// findSubordinateIndex returns the index of the first direct child tag of the
// tag at parentIdx that matches name, or -1 if none exists.
func findSubordinateIndex(tags []*gedcom.Tag, parentIdx int, name string) int {
	parentLevel := tags[parentIdx].Level
	for i := parentIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= parentLevel {
			break
		}
		if tag.Level == parentLevel+1 && tag.Tag == name {
			return i
		}
	}
	return -1
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
				cd.Date = collector.text(tag.Value)
				// Look for TIME subordinate at baseLevel+2
				for j := i + 1; j < len(tags); j++ {
					timeTag := tags[j]
					if timeTag.Level <= baseLevel+1 {
						break
					}
					if timeTag.Level == baseLevel+2 && timeTag.Tag == "TIME" {
						cd.Time = collector.text(timeTag.Value)
						break
					}
				}
			case "NOTE", "SNOTE":
				cd.NoteXRefs, cd.InlineNotes = appendRecordNote(tags, i, cd.NoteXRefs, cd.InlineNotes, collector)
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
			subm.Name = collector.text(tag.Value)

		case "ADDR":
			subm.Address = parseEventAddress(record.Tags, i, tag.Level, collector)

		case "PHON":
			subm.Phone = append(subm.Phone, collector.text(tag.Value))

		case "EMAIL":
			subm.Email = append(subm.Email, collector.text(tag.Value))

		case "LANG":
			subm.Language = append(subm.Language, collector.text(tag.Value))

		case "NOTE", "SNOTE":
			subm.NoteXRefs, subm.InlineNotes = appendRecordNote(record.Tags, i, subm.NoteXRefs, subm.InlineNotes, collector)

		case "EXID":
			subm.ExternalIDs = append(subm.ExternalIDs, parseExternalID(record.Tags, i, collector))

		case "CHAN", "FAX", "WWW", "OBJE", "RIN", "UID", "CREA", "REFN", "RFN":
			// Known tags not yet parsed into typed fields. CREA and REFN are
			// 7.0, RFN 5.5/5.5.1 (#730); typed access is #786.

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return subm
}

// parseSubmission converts record tags to a Submission entity (SUBN, GEDCOM
// 5.5 and 5.5.1).
func parseSubmission(record *gedcom.Record, collector *diagnosticCollector) *gedcom.Submission {
	subn := &gedcom.Submission{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		case "SUBM":
			subn.SubmitterXRef = tagToken(tag.Value)

		case "FAMF":
			subn.FamilyFileName = collector.text(tag.Value)

		case "TEMP":
			subn.TempleCode = collector.text(tag.Value)

		case "ANCE":
			subn.AncestorGenerations = collector.text(tag.Value)

		case "DESC":
			subn.DescendantGenerations = collector.text(tag.Value)

		case "ORDI":
			subn.OrdinanceProcessFlag = collector.text(tag.Value)

		case "RIN":
			subn.AutomatedRecordID = collector.text(tag.Value)

		case "NOTE":
			subn.NoteXRefs, subn.InlineNotes = appendRecordNote(record.Tags, i, subn.NoteXRefs, subn.InlineNotes, collector)

		case "CHAN":
			subn.ChangeDate = parseChangeDate(record.Tags, i, collector)

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	return subn
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
			repo.Name = collector.text(tag.Value)

		case "ADDR":
			repo.Address = parseEventAddress(record.Tags, i, tag.Level, collector)

		case "PHON":
			repo.Phone = append(repo.Phone, collector.text(tag.Value))

		case "EMAIL":
			repo.Email = append(repo.Email, collector.text(tag.Value))

		case "FAX":
			repo.Fax = append(repo.Fax, collector.text(tag.Value))

		case "WWW":
			repo.Website = append(repo.Website, collector.text(tag.Value))

		case "NOTE", "SNOTE":
			repo.NoteXRefs, repo.InlineNotes = appendRecordNote(record.Tags, i, repo.NoteXRefs, repo.InlineNotes, collector)

		case "EXID":
			repo.ExternalIDs = append(repo.ExternalIDs, parseExternalID(record.Tags, i, collector))

		case "CHAN", "REFN", "UID", "CREA", "RIN":
			// Known tags not yet parsed into typed fields. CREA is 7.0, RIN
			// 5.5/5.5.1 (#730); typed access is #786.

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
// shared NOTE/SNOTE record and is appended to xrefs. Any other value is inline
// note text and is appended to inline, with subordinate CONT/CONC lines folded
// in (CONT joins with a newline, CONC concatenates).
//
// It returns the updated xrefs and inline slices.
func appendRecordNote(
	tags []*gedcom.Tag, noteIdx int, xrefs, inline []string, collector *diagnosticCollector,
) (newXRefs, newInline []string) {
	tag := tags[noteIdx]
	// The pointer test runs on the trimmed value, and only a value that passes
	// it is stored trimmed. "1 NOTE  @N1@" is a padded pointer (#426 keeps every
	// space past the delimiter), and testing the raw value would classify it as
	// inline text, silently dropping the link to the shared note. Inline text
	// keeps its raw value, because there a leading space is payload.
	if ptr := tagToken(tag.Value); collector.isPointer(ptr) {
		// XRef pointer to a shared note: the GEDCOM specs do not permit
		// subordinate CONT/CONC lines here, so there is nothing to fold in.
		return append(xrefs, ptr), inline
	}

	return xrefs, append(inline, collector.text(foldedValue(tags, noteIdx)))
}

// foldedValue returns the value of tags[idx] with its direct CONT/CONC
// subordinates folded in (CONT joins with a newline, CONC concatenates). A
// value with no continuation lines is returned as-is without allocating, so
// single-line fields on hot paths such as citation PAGE cost nothing extra.
func foldedValue(tags []*gedcom.Tag, idx int) string {
	baseLevel := tags[idx].Level
	var b strings.Builder
	folded := false
	for i := idx + 1; i < len(tags); i++ {
		sub := tags[i]
		if sub.Level <= baseLevel {
			break
		}
		if sub.Level != baseLevel+1 || (sub.Tag != "CONT" && sub.Tag != "CONC") {
			continue
		}
		if !folded {
			b.WriteString(tags[idx].Value)
			folded = true
		}
		foldContinuation(&b, sub)
	}
	if !folded {
		return tags[idx].Value
	}
	return b.String()
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
			note.ExternalIDs = append(note.ExternalIDs, parseExternalID(record.Tags, i, collector))

		case "MIME", "LANG", "TRAN", "SOUR", "REFN", "UID", "CHAN", "RIN":
			// Known tags not yet parsed into typed fields (RIN: #730)

		default:
			if !strings.HasPrefix(tag.Tag, "_") {
				collector.addUnknownTag(tag.LineNumber, tag.Tag, tag.Value)
			}
		}
	}

	note.Text = collector.text(b.String())

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
			note.MIME = collector.text(tag.Value)

		case "LANG":
			note.Language = collector.text(tag.Value)

		case "TRAN":
			tran := parseSharedNoteTranslation(record.Tags, i, collector)
			note.Translations = append(note.Translations, tran)

		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			note.SourceCitations = append(note.SourceCitations, cite)

		case "EXID":
			note.ExternalIDs = append(note.ExternalIDs, parseExternalID(record.Tags, i, collector))

		case "CHAN":
			note.ChangeDate = parseChangeDate(record.Tags, i, collector)

		case "CREA", "REFN", "UID":
			// Standard SNOTE substructures not yet parsed into typed fields;
			// the raw tags remain on SharedNote.Tags (ADR 0003). Recognizing
			// them keeps UNKNOWN_TAG meaning "nonstandard tag" (#730); typed
			// access is #786.

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

	note.Text = collector.text(b.String())

	return note
}

// parseSharedNoteTranslation extracts a translation from TRAN tag in a SharedNote.
func parseSharedNoteTranslation(tags []*gedcom.Tag, tranIdx int, collector *diagnosticCollector) *gedcom.SharedNoteTranslation {
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
				tran.MIME = collector.text(tag.Value)
			case "LANG":
				tran.Language = collector.text(tag.Value)
			case "CONT", "CONC":
				foldContinuation(&b, tag)
			}
		}
	}

	tran.Value = collector.text(b.String())

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
			media.NoteXRefs, media.InlineNotes = appendRecordNote(record.Tags, i, media.NoteXRefs, media.InlineNotes, collector)
		case "SNOTE":
			// SharedNoteXRefs holds the GEDCOM 7.0 SNOTE pointers and NoteXRefs
			// the NOTE ones, so the two partition (#499). A caller wanting every
			// note pointer concatenates them; nothing needs deduping.
			//
			// The pointer test is what keeps SharedNoteXRefs holding only what
			// its name promises. SNOTE is pointer-only in the 7.0 spec, but a
			// lenient decoder still meets "1 SNOTE some text" in the wild, and
			// an unresolvable entry here is dropped without trace by AllNotes
			// (see allNotes in gedcom/notes.go). Route a non-pointer value
			// through appendRecordNote instead, which files it as inline text
			// with its CONT/CONC continuations folded in.
			if ptr := tagToken(tag.Value); collector.isPointer(ptr) {
				media.SharedNoteXRefs = append(media.SharedNoteXRefs, ptr)
			} else {
				media.NoteXRefs, media.InlineNotes = appendRecordNote(record.Tags, i, media.NoteXRefs, media.InlineNotes, collector)
			}
		case "SOUR":
			cite := parseSourceCitation(record.Tags, i, tag.Level, collector)
			media.SourceCitations = append(media.SourceCitations, cite)
		case "CHAN":
			media.ChangeDate = parseChangeDate(record.Tags, i, collector)
		case "CREA":
			media.CreationDate = parseChangeDate(record.Tags, i, collector)
		case "REFN":
			media.RefNumbers = append(media.RefNumbers, parseRefNumber(record.Tags, i, collector))
		case "UID", "_UID":
			media.UIDs = append(media.UIDs, collector.text(tag.Value))
		case "RESN":
			media.Restriction = collector.text(tag.Value)
		case "EXID":
			media.ExternalIDs = append(media.ExternalIDs, parseExternalID(record.Tags, i, collector))
		case "RIN":
			// Known tag not yet parsed into a typed field; the raw tag remains
			// on MediaObject.Tags (ADR 0003) (#730).
		case "FORM", "TITL", "BLOB", "OBJE":
			// The 5.5 record layout, which 5.5.1 moved under FILE: kept raw
			// on MediaObject.Tags, and reported in a document declaring
			// another version (#730).
			collector.addUnknownTagOutside55(tag)
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
		FileRef: collector.text(tags[fileIdx].Value),
	}

	for i := fileIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "FORM":
				file.Form = collector.text(tag.Value)
				// Look for MEDI at baseLevel+2
				for j := i + 1; j < len(tags); j++ {
					mediTag := tags[j]
					if mediTag.Level <= baseLevel+1 {
						break
					}
					if mediTag.Level == baseLevel+2 && mediTag.Tag == "MEDI" {
						file.MediaType = collector.text(mediTag.Value)
						file.MediaPhrase = collector.text(findSubordinate(tags, j, "PHRASE"))
						break
					}
				}
			case "TITL":
				file.Title = collector.text(tag.Value)
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
		FileRef: collector.text(tags[tranIdx].Value),
	}

	for i := tranIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "FORM":
				tran.Form = collector.text(tag.Value)
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
				link.Title = collector.text(tag.Value)
			case "NOTE", "SNOTE":
				// Before #472 this fell through to default and every
				// OBJE-level NOTE was reported as an unknown tag (#470).
				link.NoteXRefs, link.InlineNotes = appendRecordNote(
					tags, i, link.NoteXRefs, link.InlineNotes, collector)
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
