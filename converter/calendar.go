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
//   - record.Tags: every DATE and SDATE value, the form a decoded document carries;
//   - the entity: Date and SortDate on the events, attributes and LDS
//     ordinances of individuals and families, with ParsedDate.Original kept in
//     step (the parsed calendar itself is unchanged).
//
// Each rewrite is reported as a Normalized note. A calendar with no standard
// equivalent (an escape such as @#DROMAN@ or @#DUNKNOWN@, or a 7.0 extension
// calendar such as _MAYAN) is left unchanged and reported as a Preserved note.
// Changes are reported from record.Tags when the record has any, and from the
// entity otherwise, so a date is not reported twice.
func transformDateCalendars(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	t := &calendarTranslator{target: targetVersion, report: report}
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		t.translateTags(record)
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

// translateTags rewrites the DATE and SDATE values in record.Tags, reporting
// each against the path of tag names leading to it.
func (t *calendarTranslator) translateTags(record *gedcom.Record) {
	var path []string // tag names by level, starting at level 1
	for _, tag := range record.Tags {
		if tag == nil || tag.Level < 1 {
			continue
		}
		path = append(path[:min(tag.Level-1, len(path))], tag.Tag)
		if tag.Tag != "DATE" && tag.Tag != "SDATE" {
			continue
		}
		t.translate(&tag.Value, nil, true, BuildNestedPath(string(record.Type), record.XRef, path...))
	}
}

// translateEntity rewrites the dates on an individual's or family's events,
// attributes and LDS ordinances. Other record types carry no such dates.
func (t *calendarTranslator) translateEntity(record *gedcom.Record, report bool) {
	var events []*gedcom.Event
	var attributes []*gedcom.Attribute
	var ordinances []*gedcom.LDSOrdinance
	if ind, ok := record.GetIndividual(); ok && ind != nil {
		events, attributes, ordinances = ind.Events, ind.Attributes, ind.LDSOrdinances
	} else if fam, ok := record.GetFamily(); ok && fam != nil {
		events, attributes, ordinances = fam.Events, fam.Attributes, fam.LDSOrdinances
	} else {
		return
	}

	recordType := string(record.Type)
	for _, e := range events {
		if e != nil {
			t.translate(&e.Date, e.ParsedDate, report, BuildNestedPath(recordType, record.XRef, string(e.Type), "DATE"))
			t.translate(&e.SortDate, nil, report, BuildNestedPath(recordType, record.XRef, string(e.Type), "SDATE"))
		}
	}
	for _, a := range attributes {
		if a != nil {
			t.translate(&a.Date, a.ParsedDate, report, BuildNestedPath(recordType, record.XRef, string(a.Type), "DATE"))
			t.translate(&a.SortDate, nil, report, BuildNestedPath(recordType, record.XRef, string(a.Type), "SDATE"))
		}
	}
	for _, o := range ordinances {
		if o != nil {
			t.translate(&o.Date, o.ParsedDate, report, BuildNestedPath(recordType, record.XRef, string(o.Type), "DATE"))
		}
	}
}

// translate rewrites *value in place, keeping parsed (when non-nil) in step,
// and reports the change at path when report is set.
func (t *calendarTranslator) translate(value *string, parsed *gedcom.Date, report bool, path string) {
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
		t.report.AddNormalized(gedcom.ConversionNote{Path: path, Original: original, Result: result, Reason: reason})
	}
	if unknown {
		t.report.AddPreserved(gedcom.ConversionNote{
			Path:     path,
			Original: original,
			Result:   result,
			Reason:   "Calendar has no GEDCOM " + t.target.String() + " equivalent; left unchanged",
		})
	}
}

// translateDateCalendars returns value with its calendar syntax rewritten for
// targetVersion, and whether it holds a calendar with no standard equivalent.
// Text inside parentheses (a date phrase) is copied unchanged.
func translateDateCalendars(value string, targetVersion gedcom.Version) (result string, unknown bool) {
	rewrite := keywordsToEscapes
	if targetVersion == gedcom.Version70 {
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
