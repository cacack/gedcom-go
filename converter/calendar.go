package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// calendarEscapeToKeyword maps each standard GEDCOM 5.5/5.5.1 calendar escape
// name (the text between "@#D" and "@") to its GEDCOM 7.0 keyword.
var calendarEscapeToKeyword = map[string]string{
	"GREGORIAN": "GREGORIAN",
	"JULIAN":    "JULIAN",
	"HEBREW":    "HEBREW",
	"FRENCH R":  "FRENCH_R",
}

// calendarKeywordToEscape maps each standard GEDCOM 7.0 calendar keyword to its
// GEDCOM 5.5/5.5.1 escape.
var calendarKeywordToEscape = map[string]string{
	"GREGORIAN": "@#DGREGORIAN@",
	"JULIAN":    "@#DJULIAN@",
	"HEBREW":    "@#DHEBREW@",
	"FRENCH_R":  "@#DFRENCH R@",
}

// transformDateCalendars rewrites the calendar syntax of every date for the
// target version: GEDCOM 5.5/5.5.1 escapes (@#DJULIAN@) become GEDCOM 7.0
// keywords (JULIAN) and vice versa. A calendar belongs to each date
// production, so every endpoint of a range or period is translated; date
// phrase text in parentheses is left alone.
//
// Both representations are rewritten, so the result does not depend on which
// one the encoder writes from:
//   - record.Tags and doc.Header.Tags: every DATE and SDATE value, the form a
//     decoded document carries;
//   - the header's typed Date (HEAD.DATE);
//   - the entity: Date and SortDate on the events, attributes and LDS
//     ordinances of individuals and families, with ParsedDate.Original kept in
//     step (the parsed calendar itself is unchanged), and the DATA.DATE of
//     every source citation on individual, family, media and shared note
//     records, their events, attributes and associations.
//
// Each rewrite is reported as a Normalized note. A calendar with no standard
// equivalent (an escape such as @#DROMAN@ or @#DUNKNOWN@, or a 7.0 extension
// calendar such as _MAYAN) is left unchanged and reported as a Preserved note.
func transformDateCalendars(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	t := &calendarTranslator{target: targetVersion, report: report}
	if header := doc.Header; header != nil {
		base := datePath{recordType: "HEAD"}
		t.translateTags(header.Tags, base.recordType, base.xref)
		// Reported only when the encoder writes it: see the record loop below.
		t.translate(&header.Date, nil, len(header.Tags) == 0, base.at("DATE"))
	}
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		t.translateTags(record.Tags, string(record.Type), record.XRef)
		// Changes are reported from the representation the encoder writes:
		// record.Tags whenever it is non-empty, the entity only otherwise
		// (encoder.writeRecord). The entity is still rewritten when Tags exist,
		// so the two stay consistent, but reporting it too would count each
		// date twice; a date held only by the entity of a record with Tags is
		// never written, so it has nothing to report.
		t.translateEntity(record, len(record.Tags) == 0)
	}

	if t.count > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "DATE_CALENDAR_CONVERTED",
			Description: "Converted date calendar syntax for target GEDCOM version",
			Count:       t.count,
		})
	}
}

// calendarTranslator carries the target version and the running count of
// reported rewrites through one transformDateCalendars pass.
type calendarTranslator struct {
	target gedcom.Version
	report *gedcom.ConversionReport
	count  int
}

// datePath locates a date for reporting: the record it belongs to and the tag
// names leading to it. The report path is built only when a note is added.
type datePath struct {
	recordType string
	xref       string
	tags       []string
}

// at returns p extended by tags, without sharing p's backing array.
func (p datePath) at(tags ...string) datePath {
	return datePath{recordType: p.recordType, xref: p.xref, tags: append(p.tags[:len(p.tags):len(p.tags)], tags...)}
}

func (p datePath) String() string {
	return BuildNestedPath(p.recordType, p.xref, p.tags...)
}

// translateTags rewrites the DATE and SDATE values in the tags of a record or
// the header, reporting each against the path of tag names leading to it.
func (t *calendarTranslator) translateTags(tags []*gedcom.Tag, recordType, xref string) {
	var path []string // tag names by level, starting at level 1
	for _, tag := range tags {
		if tag == nil || tag.Level < 1 {
			continue
		}
		path = append(path[:min(tag.Level-1, len(path))], tag.Tag)
		if tag.Tag != "DATE" && tag.Tag != "SDATE" {
			continue
		}
		t.translate(&tag.Value, nil, true, datePath{recordType, xref, path})
	}
}

// translateEntity rewrites the typed dates of an individual, family, media or
// shared note record. Other record types carry no calendar dates.
func (t *calendarTranslator) translateEntity(record *gedcom.Record, report bool) {
	base := datePath{recordType: string(record.Type), xref: record.XRef}
	if ind, ok := record.GetIndividual(); ok && ind != nil {
		t.translateCitations(ind.SourceCitations, report, base)
		t.translateAssociations(ind.Associations, report, base)
		t.translateFacts(ind.Events, ind.Attributes, ind.LDSOrdinances, report, base)
	} else if fam, ok := record.GetFamily(); ok && fam != nil {
		t.translateCitations(fam.SourceCitations, report, base)
		t.translateFacts(fam.Events, fam.Attributes, fam.LDSOrdinances, report, base)
	} else if media, ok := record.GetMediaObject(); ok && media != nil {
		t.translateCitations(media.SourceCitations, report, base)
	} else if snote, ok := record.GetSharedNote(); ok && snote != nil {
		t.translateCitations(snote.SourceCitations, report, base)
	}
}

// translateFacts rewrites the dates on events, attributes and LDS ordinances,
// and on the citations and associations beneath them.
func (t *calendarTranslator) translateFacts(events []*gedcom.Event, attributes []*gedcom.Attribute, ordinances []*gedcom.LDSOrdinance, report bool, base datePath) {
	for _, e := range events {
		if e != nil {
			at := base.at(string(e.Type))
			t.translateDated(&e.Date, &e.SortDate, e.ParsedDate, report, at)
			t.translateCitations(e.SourceCitations, report, at)
			t.translateAssociations(e.Associations, report, at)
		}
	}
	for _, a := range attributes {
		if a != nil {
			at := base.at(string(a.Type))
			t.translateDated(&a.Date, &a.SortDate, a.ParsedDate, report, at)
			t.translateCitations(a.SourceCitations, report, at)
			t.translateAssociations(a.Associations, report, at)
		}
	}
	for _, o := range ordinances {
		if o != nil {
			t.translateDated(&o.Date, nil, o.ParsedDate, report, base.at(string(o.Type)))
		}
	}
}

// translateDated rewrites a fact's DATE and, when sortDate is non-nil, its SDATE.
func (t *calendarTranslator) translateDated(date, sortDate *string, parsed *gedcom.Date, report bool, at datePath) {
	t.translate(date, parsed, report, at.at("DATE"))
	if sortDate != nil {
		t.translate(sortDate, nil, report, at.at("SDATE"))
	}
}

// translateCitations rewrites each citation's DATA.DATE.
func (t *calendarTranslator) translateCitations(citations []*gedcom.SourceCitation, report bool, at datePath) {
	for _, c := range citations {
		if c != nil && c.Data != nil {
			t.translate(&c.Data.Date, nil, report, at.at("SOUR", "DATA", "DATE"))
		}
	}
}

// translateAssociations rewrites the citation dates beneath each association.
func (t *calendarTranslator) translateAssociations(associations []*gedcom.Association, report bool, at datePath) {
	for _, a := range associations {
		if a != nil {
			t.translateCitations(a.SourceCitations, report, at.at("ASSO"))
		}
	}
}

// translate rewrites *value in place, keeping parsed (when non-nil) in step,
// and reports the change at path when report is set.
func (t *calendarTranslator) translate(value *string, parsed *gedcom.Date, report bool, path datePath) {
	original := *value
	result, unknown := translateDateCalendars(original, t.target)
	*value = result
	if parsed != nil && result != original {
		parsed.Original = result
	}
	if !report {
		return
	}

	if result != original {
		t.count++
		reason := "GEDCOM 7.0 writes calendars as keywords instead of @#D...@ escapes"
		if t.target != gedcom.Version70 {
			reason = "GEDCOM 5.x writes calendars as @#D...@ escapes instead of keywords"
		}
		t.report.AddNormalized(gedcom.ConversionNote{Path: path.String(), Original: original, Result: result, Reason: reason})
	}
	if unknown {
		t.report.AddPreserved(gedcom.ConversionNote{
			Path:     path.String(),
			Original: original,
			Result:   result,
			Reason:   "Calendar has no GEDCOM " + t.target.String() + " equivalent; left unchanged",
		})
	}
}

// translateDateCalendars returns value with its calendar syntax rewritten for
// targetVersion, and whether it holds a calendar with no standard equivalent.
// Text inside parentheses (a date phrase) is copied unchanged. A phrase ends
// at its first ')': neither GEDCOM grammar nests parentheses, so the text after
// a nested ')' is treated as outside the phrase.
func translateDateCalendars(value string, targetVersion gedcom.Version) (result string, unknown bool) {
	rewrite := keywordsToEscapes
	if targetVersion == gedcom.Version70 {
		if !strings.Contains(value, "@#D") {
			return value, false // fast path: no escape to rewrite
		}
		rewrite = escapesToKeywords
	}

	var b strings.Builder
	for value != "" {
		open := strings.IndexByte(value, '(')
		if open < 0 {
			open = len(value)
		}
		outside, u := rewrite(value[:open])
		b.WriteString(outside)
		unknown = unknown || u
		value = value[open:]

		closing := strings.IndexByte(value, ')')
		if closing < 0 {
			closing = len(value) - 1
		}
		b.WriteString(value[:closing+1])
		value = value[closing+1:]
	}
	return b.String(), unknown
}

// escapesToKeywords replaces each standard @#D...@ calendar escape in s with
// its GEDCOM 7.0 keyword, reporting whether a non-standard escape remains.
// Escape names are matched case-insensitively, as gedcom.ParseDate does.
func escapesToKeywords(s string) (string, bool) {
	var b strings.Builder
	unknown := false
	for {
		start := strings.Index(s, "@#D")
		if start < 0 {
			break
		}
		end := strings.IndexByte(s[start+3:], '@')
		if end < 0 {
			break
		}
		end += start + 3 // index of the closing '@'
		b.WriteString(s[:start])
		if keyword, ok := calendarEscapeToKeyword[strings.ToUpper(s[start+3:end])]; ok {
			b.WriteString(keyword)
		} else {
			b.WriteString(s[start : end+1])
			unknown = true
		}
		s = s[end+1:]
	}
	b.WriteString(s)
	return b.String(), unknown
}

// keywordsToEscapes replaces each standard GEDCOM 7.0 calendar keyword in s
// with its @#D...@ escape, reporting whether an extension calendar (_XXX)
// remains. Keywords are matched case-sensitively, as the 7.0 grammar spells them.
func keywordsToEscapes(s string) (string, bool) {
	words := strings.Split(s, " ")
	unknown := false
	for i, word := range words {
		if escape, ok := calendarKeywordToEscape[word]; ok {
			words[i] = escape
		} else if strings.HasPrefix(word, "_") {
			unknown = true
		}
	}
	return strings.Join(words, " "), unknown
}
