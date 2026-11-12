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

// transformDateSyntax rewrites the syntax of every date for the target
// version:
//   - calendars: GEDCOM 5.5/5.5.1 escapes (@#DJULIAN@) become GEDCOM 7.0
//     keywords (JULIAN) and vice versa. A calendar belongs to each date
//     production, so every endpoint of a range or period is translated;
//   - epochs: B.C., BC and B.C.E. become BCE for 7.0, the only epoch it
//     spells, and every spelling becomes B.C. for 5.5/5.5.1 (issue #722);
//   - date phrases (issue #722): for 7.0, which has neither INT nor a
//     parenthesized phrase, INT <date> (<phrase>) becomes DATE <date> with a
//     PHRASE substructure and (<phrase>) an empty DATE with one; INT <date>
//     with no phrase becomes <date>. For 5.5/5.5.1, a DATE and its PHRASE
//     fold back into INT <date> (<phrase>), or (<phrase>) when the value is
//     empty.
//
// Text inside a phrase is never rewritten.
//
// Both representations are rewritten, so the result does not depend on which
// one the encoder writes from:
//   - record.Tags and doc.Header.Tags: every DATE and SDATE value, the form a
//     decoded document carries, with the PHRASE beneath it;
//   - the header's typed Date (HEAD.DATE);
//   - the entity: Date and SortDate on the events, attributes and LDS
//     ordinances of individuals and families, the DATA.DATE of every source
//     citation on individual, family, media and shared note records, their
//     events, attributes and associations, and the DATA.EVEN.DATE of every
//     source record. An event, attribute or ordinance ParsedDate is kept in
//     step: its Original follows a calendar or epoch rewrite, and a moved
//     phrase re-parses it as decoding the output would, with Phrase holding
//     the 7.0 PHRASE.
//
// Each rewrite is reported as a Normalized note and counted in the
// DATE_CALENDAR_CONVERTED, DATE_EPOCH_CONVERTED or DATE_PHRASE_CONVERTED
// transformation. A calendar with no standard equivalent (an escape such as
// @#DROMAN@ or @#DUNKNOWN@, or a 7.0 extension calendar such as _MAYAN) is
// left unchanged and reported as a Preserved note. So is a non-Gregorian
// calendar on a header date converted to 7.0 (HEAD.DATE, HEAD.SOUR.DATA.DATE):
// 7.0 types those as DateExact, which has no calendar, and the date is not
// converted to the Gregorian equivalent.
//
// A dual-year date (11 FEB 1731/32) converted to 7.0, which has no syntax for
// one, is left unchanged and reported as a Preserved note and, once per
// conversion, as DataLoss naming the affected records (issue #567). So is a
// date phrase converted to 7.0 that cannot become a PHRASE: one that is
// neither INT <date> (<phrase>) nor the whole value (ABT 1850 (circa)), or
// that sits where nothing can carry a PHRASE (an exact date such as CHAN.DATE
// or HEAD.DATE, a DATE that already has a PHRASE, or a typed citation or
// header date, which has no phrase field).
//
// On a conversion to 5.5/5.5.1, a PHRASE that cannot be folded is left for
// dropV7OnlyTags, which drops and reports it: one beside a value INT cannot
// qualify (a range, period or approximation such as ABT 1850), one whose text
// holds a parenthesis or line break, one with substructures, or one that
// would make the DATE line longer than 5.x allows.
func transformDateSyntax(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	t := &dateTranslator{target: targetVersion, report: report}
	if header := doc.Header; header != nil {
		base := datePath{recordType: "HEAD"}
		header.Tags = t.translateTags(header.Tags, base)
		// Reported only when the encoder writes it: see the record loop below.
		t.translate(&header.Date, dateSlot{}, len(header.Tags) == 0, base.at("DATE"))
	}
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		record.Tags = t.translateTags(record.Tags, datePath{recordType: string(record.Type), xref: record.XRef})
		// Changes are reported from the representation the encoder writes:
		// record.Tags whenever it is non-empty, the entity only otherwise
		// (encoder.writeRecord). The entity is still rewritten when Tags exist,
		// so the two stay consistent, but reporting it too would count each
		// date twice; a date held only by the entity of a record with Tags is
		// never written, so it has nothing to report.
		t.translateEntity(record, len(record.Tags) == 0)
	}

	t.addSummary()
}

// Report reasons for a date left unchanged and not valid 7.0.
const (
	dualYearReason        = "GEDCOM 7.0 has no dual-year syntax (1731/32); date left unchanged and not valid 7.0"
	unmappedPhraseReason  = "GEDCOM 7.0 has no INT or parenthesized date phrase, and this one cannot become a PHRASE; date left unchanged and not valid 7.0"
	unmappedPhraseFeature = "Date phrases"
)

// dateTranslator carries the target version and the running counts of
// reported rewrites through one transformDateSyntax pass.
type dateTranslator struct {
	target gedcom.Version
	report *gedcom.ConversionReport

	// calendars, epochs and phrases count the reported dates whose calendar,
	// epoch or phrase was rewritten; one date can count in each.
	calendars, epochs, phrases int

	// dualYears and unmappedPhrases count the reported dates converted to 7.0
	// with a dual year or a phrase left in the value, and the record lists
	// name the records holding them. A header date has no XRef: it is
	// counted and gets a Preserved note but names no record.
	dualYears             int
	dualYearRecords       recordList
	unmappedPhrases       int
	unmappedPhraseRecords recordList
}

// recordList collects record XRefs once each, in the order first added.
type recordList struct {
	xrefs []string
	seen  map[string]bool
}

// add appends xref unless it is empty or already listed.
func (l *recordList) add(xref string) {
	if xref == "" || l.seen[xref] {
		return
	}
	if l.seen == nil {
		l.seen = make(map[string]bool)
	}
	l.seen[xref] = true
	l.xrefs = append(l.xrefs, xref)
}

// addSummary adds the per-kind transformations and the DataLoss items for
// dates left invalid in the target.
func (t *dateTranslator) addSummary() {
	for _, tr := range []gedcom.Transformation{
		{Type: "DATE_CALENDAR_CONVERTED", Description: "Converted date calendar syntax for target GEDCOM version", Count: t.calendars},
		{Type: "DATE_EPOCH_CONVERTED", Description: "Converted B.C. epoch spelling for target GEDCOM version", Count: t.epochs},
		{Type: "DATE_PHRASE_CONVERTED", Description: "Moved date phrases between DATE values and PHRASE substructures for target GEDCOM version", Count: t.phrases},
	} {
		if tr.Count > 0 {
			t.report.AddTransformation(tr)
		}
	}
	if t.dualYears > 0 {
		t.report.AddDataLoss(gedcom.DataLossItem{
			Feature:         "Dual-year dates",
			Reason:          dualYearReason,
			AffectedRecords: t.dualYearRecords.xrefs,
		})
	}
	if t.unmappedPhrases > 0 {
		t.report.AddDataLoss(gedcom.DataLossItem{
			Feature:         unmappedPhraseFeature,
			Reason:          unmappedPhraseReason,
			AffectedRecords: t.unmappedPhraseRecords.xrefs,
		})
	}
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
// the header, reporting each against the path of tag names leading to it, and
// returns the tags with each PHRASE the rewrite added or folded away. tags
// itself is returned when no PHRASE moved.
func (t *dateTranslator) translateTags(tags []*gedcom.Tag, base datePath) []*gedcom.Tag {
	out := tags
	copied := false
	copyThrough := func(n int) {
		if !copied {
			out = append(make([]*gedcom.Tag, 0, len(tags)+1), tags[:n]...)
			copied = true
		}
	}
	var folded map[int]bool // indexes of PHRASE tags folded into their DATE
	var path []string       // tag names by level, starting at level 1
	for i, tag := range tags {
		if folded[i] {
			copyThrough(i)
			continue
		}
		if copied {
			out = append(out, tag)
		}
		if tag == nil || tag.Level < 1 {
			continue
		}
		path = append(path[:min(tag.Level-1, len(path))], tag.Tag)
		if tag.Tag != "DATE" && tag.Tag != "SDATE" {
			continue
		}
		at := datePath{base.recordType, base.xref, path}
		phraseAt := childIndex(tags, i, "PHRASE")
		r := t.translate(&tag.Value, t.tagSlot(tags, i, phraseAt, at), true, at)
		switch {
		case r.phrase != "":
			copyThrough(i + 1)
			out = append(out, &gedcom.Tag{Level: tag.Level + 1, Tag: "PHRASE", Value: r.phrase})
		case r.folded:
			if folded == nil {
				folded = make(map[int]bool)
			}
			folded[phraseAt] = true
		}
	}
	return out
}

// tagSlot returns the dateSlot of the DATE or SDATE at tags[i], whose PHRASE
// is at tags[phraseAt] (-1 for none) and whose path is at.
func (t *dateTranslator) tagSlot(tags []*gedcom.Tag, i, phraseAt int, at datePath) dateSlot {
	if t.target == gedcom.Version70 {
		return dateSlot{movable: phraseAt < 0 && !at.isExactDate()}
	}
	if phraseAt < 0 || hasChildren(tags, phraseAt) {
		return dateSlot{}
	}
	return dateSlot{
		phrase:  tags[phraseAt].Value,
		movable: true,
		maxLen:  maxValueLength(tags[i].Level, "", tags[i].Tag, tags[i].Level+1),
	}
}

// childIndex returns the index of the first direct child of tags[parent]
// named name, or -1.
func childIndex(tags []*gedcom.Tag, parent int, name string) int {
	level := tags[parent].Level
	for i := parent + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag == nil {
			continue
		}
		if tag.Level <= level {
			break
		}
		if tag.Level == level+1 && tag.Tag == name {
			return i
		}
	}
	return -1
}

// hasChildren reports whether tags[i] has a subordinate tag.
func hasChildren(tags []*gedcom.Tag, i int) bool {
	return i+1 < len(tags) && tags[i+1] != nil && tags[i+1].Level > tags[i].Level
}

// translateEntity rewrites the typed dates of an individual, family, media,
// shared note or source record. Other record types carry no calendar dates.
func (t *dateTranslator) translateEntity(record *gedcom.Record, report bool) {
	base := datePath{recordType: string(record.Type), xref: record.XRef}
	if ind, ok := record.GetIndividual(); ok && ind != nil {
		for _, n := range ind.Names {
			if n != nil {
				t.translateCitations(n.SourceCitations, report, base.at("NAME"))
			}
		}
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
	} else if src, ok := record.GetSource(); ok && src != nil {
		t.translateSourceData(src.Data, report, base)
	}
}

// translateSourceData rewrites the DATE of each event a source record's DATA
// records, keeping DatePhrase in step. The encoder writes DatePhrase only
// beneath a non-empty Date, so a phrase that would leave the 7.0 DATE empty
// stays in the value, kept and reported like any phrase with no 7.0 form.
func (t *dateTranslator) translateSourceData(data *gedcom.SourceData, report bool, at datePath) {
	if data == nil {
		return
	}
	for _, e := range data.Events {
		if e == nil {
			continue
		}
		var slot dateSlot
		if t.target == gedcom.Version70 {
			slot.movable = e.DatePhrase == "" && convertDate(e.Date, dateSlot{movable: true}, t.target).value != ""
		} else if e.DatePhrase != "" {
			// A source DATA.EVEN.DATE is a level-3 line.
			slot = dateSlot{phrase: e.DatePhrase, movable: true, maxLen: maxValueLength(3, "", "DATE", 4)}
		}
		r := t.translate(&e.Date, slot, report, at.at("DATA", "EVEN", "DATE"))
		switch {
		case r.phrase != "":
			e.DatePhrase = r.phrase
		case r.folded:
			e.DatePhrase = ""
		}
	}
}

// translateFacts rewrites the dates on events, attributes and LDS ordinances,
// and on the citations and associations beneath them.
func (t *dateTranslator) translateFacts(events []*gedcom.Event, attributes []*gedcom.Attribute, ordinances []*gedcom.LDSOrdinance, report bool, base datePath) {
	for _, e := range events {
		if e != nil {
			at := base.at(string(e.Type))
			t.translateDated(&e.Date, &e.SortDate, &e.ParsedDate, report, at)
			t.translateCitations(e.SourceCitations, report, at)
			t.translateAssociations(e.Associations, report, at)
		}
	}
	for _, a := range attributes {
		if a != nil {
			at := base.at(string(a.Type))
			t.translateDated(&a.Date, &a.SortDate, &a.ParsedDate, report, at)
			t.translateCitations(a.SourceCitations, report, at)
			t.translateAssociations(a.Associations, report, at)
		}
	}
	for _, o := range ordinances {
		if o != nil {
			t.translateDated(&o.Date, nil, &o.ParsedDate, report, base.at(string(o.Type)))
		}
	}
}

// translateDated rewrites a fact's DATE, keeping *parsed in step, and, when
// sortDate is non-nil, its SDATE. The DATE's phrase is the 7.0 PHRASE the
// encoder writes from parsed (gedcom.Date.PhraseSubstructure).
func (t *dateTranslator) translateDated(date, sortDate *string, parsed **gedcom.Date, report bool, at datePath) {
	original := *date
	phrase := (*parsed).PhraseSubstructure(original)
	var slot dateSlot
	if t.target == gedcom.Version70 {
		slot.movable = phrase == ""
	} else if phrase != "" {
		// An event, attribute or ordinance DATE is a level-2 line.
		slot = dateSlot{phrase: phrase, movable: true, maxLen: maxValueLength(2, "", "DATE", 3)}
	}
	r := t.translate(date, slot, report, at.at("DATE"))
	switch {
	case r.moved:
		*parsed = parsedAfterMove(r)
	case *parsed != nil && *date != original:
		(*parsed).Original = *date
	}
	if sortDate != nil {
		t.translate(sortDate, dateSlot{}, report, at.at("SDATE"))
	}
}

// parsedAfterMove returns the parsed form of a date whose phrase r moved, as
// the decoder reads the DATE (and PHRASE) written: a phrase-only Date when
// the value is empty, nil when nothing is left.
func parsedAfterMove(r dateResult) *gedcom.Date {
	parsed, err := gedcom.ParseDate(r.value)
	if err != nil {
		if r.phrase == "" {
			return nil
		}
		return &gedcom.Date{Original: r.value, Calendar: gedcom.CalendarGregorian, Phrase: r.phrase, IsPhrase: true}
	}
	if r.phrase != "" {
		parsed.Phrase = r.phrase
	}
	return parsed
}

// translateCitations rewrites each citation's DATA.DATE.
func (t *dateTranslator) translateCitations(citations []*gedcom.SourceCitation, report bool, at datePath) {
	for _, c := range citations {
		if c != nil && c.Data != nil {
			t.translate(&c.Data.Date, dateSlot{}, report, at.at("SOUR", "DATA", "DATE"))
		}
	}
}

// translateAssociations rewrites the citation dates beneath each association.
func (t *dateTranslator) translateAssociations(associations []*gedcom.Association, report bool, at datePath) {
	for _, a := range associations {
		if a != nil {
			t.translateCitations(a.SourceCitations, report, at.at("ASSO"))
		}
	}
}

// translate rewrites *value in place (see convertDate) and reports the change
// at path when report is set.
func (t *dateTranslator) translate(value *string, slot dateSlot, report bool, path datePath) dateResult {
	original := *value
	r := convertDate(original, slot, t.target)
	*value = r.value
	if report {
		t.reportRewrite(r, original, slot, path)
		t.reportUnconverted(r, original, path)
	}
	return r
}

// reportRewrite adds the Normalized note and counts for the rewrites r made
// to original, whose 5.x-target PHRASE was slot.phrase.
func (t *dateTranslator) reportRewrite(r dateResult, original string, slot dateSlot, path datePath) {
	if r.calendar || r.epoch || r.moved {
		before, after := original, r.value
		if r.folded {
			before = withPhrase(original, slot.phrase)
		}
		if r.phrase != "" {
			after = withPhrase(r.value, r.phrase)
		}
		t.report.AddNormalized(gedcom.ConversionNote{Path: path.String(), Original: before, Result: after, Reason: t.reasons(r)})
	}
	if r.calendar {
		t.calendars++
	}
	if r.epoch {
		t.epochs++
	}
	if r.moved {
		t.phrases++
	}
}

// reportUnconverted adds a Preserved note for each part of r's value the
// target cannot express: a non-standard calendar, a non-Gregorian header
// date, and, for 7.0, a dual year or a date phrase, which are also counted
// toward their DataLoss items.
func (t *dateTranslator) reportUnconverted(r dateResult, original string, path datePath) {
	result := r.value
	preserve := func(reason string) {
		t.report.AddPreserved(gedcom.ConversionNote{Path: path.String(), Original: original, Result: result, Reason: reason})
	}
	if r.unknownCalendar {
		preserve("Calendar has no GEDCOM " + t.target.String() + " equivalent; left unchanged")
	} else if t.target == gedcom.Version70 && path.isHeaderExactDate() && hasNonGregorianCalendar(result) {
		preserve("GEDCOM 7.0 header dates are exact Gregorian dates (DateExact); non-Gregorian calendar left unchanged and needs attention")
	}
	if t.target != gedcom.Version70 {
		return
	}
	if hasDualYear(result) {
		preserve(dualYearReason)
		t.dualYears++
		t.dualYearRecords.add(path.xref)
	}
	if hasV5PhraseSyntax(result) {
		preserve(unmappedPhraseReason)
		t.unmappedPhrases++
		t.unmappedPhraseRecords.add(path.xref)
	}
}

// reasons returns the Normalized note reason for the rewrites r made.
func (t *dateTranslator) reasons(r dateResult) string {
	v7 := t.target == gedcom.Version70
	var reasons []string
	if r.calendar {
		if v7 {
			reasons = append(reasons, "GEDCOM 7.0 writes calendars as keywords instead of @#D...@ escapes")
		} else {
			reasons = append(reasons, "GEDCOM 5.x writes calendars as @#D...@ escapes instead of keywords")
		}
	}
	if r.epoch {
		if v7 {
			reasons = append(reasons, "GEDCOM 7.0 writes the B.C. epoch as BCE")
		} else {
			reasons = append(reasons, "GEDCOM 5.x writes the B.C. epoch as B.C.")
		}
	}
	if r.moved {
		if v7 {
			reasons = append(reasons, "GEDCOM 7.0 has no INT or parenthesized date phrase; the phrase moves to a PHRASE substructure")
		} else {
			reasons = append(reasons, "GEDCOM 5.x has no DATE PHRASE; the phrase moves into the value as INT <date> (<phrase>) or (<phrase>)")
		}
	}
	return strings.Join(reasons, "; ")
}

// withPhrase describes a DATE value with the PHRASE beneath it, for a report
// note: "1850 + PHRASE about 1850", or "PHRASE unknown" for an empty value.
func withPhrase(value, phrase string) string {
	if value == "" {
		return "PHRASE " + phrase
	}
	return value + " + PHRASE " + phrase
}

// dateSlot describes where a date's phrase can go. On a 7.0 target, movable
// reports whether a PHRASE can be written beside the DATE. On a 5.x target,
// phrase is the 7.0 PHRASE beside the DATE and movable reports whether it can
// be folded into a value of at most maxLen bytes. The zero value moves nothing.
type dateSlot struct {
	phrase  string
	movable bool
	maxLen  int
}

// dateResult is one date value rewritten for a target version.
type dateResult struct {
	value  string // the rewritten value
	phrase string // 7.0 target: the PHRASE moved out of the value, "" for none
	folded bool   // 5.x target: the slot's PHRASE was folded into value

	calendar, epoch bool // the calendar or epoch syntax was rewritten
	moved           bool // a phrase moved either way, or INT was dropped
	unknownCalendar bool // a calendar with no standard equivalent remains
}

// convertDate rewrites a DATE value for target: its calendars and epochs and,
// where slot allows, its phrase (see transformDateSyntax).
func convertDate(value string, slot dateSlot, target gedcom.Version) dateResult {
	var r dateResult
	calendared, unknown := translateDateCalendars(value, target)
	r.calendar, r.unknownCalendar = calendared != value, unknown
	r.value = translateEpochs(calendared, target)
	r.epoch = r.value != calendared
	if !slot.movable {
		return r
	}
	if target == gedcom.Version70 {
		if date, phrase, ok := splitV5Phrase(r.value); ok {
			r.value, r.phrase, r.moved = date, phrase, true
		}
	} else if folded, ok := foldV7Phrase(r.value, slot.phrase); ok && len(folded) <= slot.maxLen {
		r.value, r.folded, r.moved = folded, true, true
	}
	return r
}

// splitV5Phrase splits a GEDCOM 5.5/5.5.1 date phrase into the 7.0 DATE value
// and PHRASE: "INT <date> (<phrase>)" gives <date> and <phrase>, "INT <date>"
// gives <date> and no phrase, and "(<phrase>)" gives an empty value and
// <phrase>. It reports false for anything else, including an INT whose date
// does not parse and an empty "()".
func splitV5Phrase(value string) (date, phrase string, ok bool) {
	s := strings.TrimSpace(value)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		phrase = s[1 : len(s)-1]
		return "", phrase, phrase != ""
	}
	keyword, rest, _ := strings.Cut(s, " ")
	if !strings.EqualFold(keyword, "INT") {
		return "", "", false
	}
	date = strings.TrimSpace(rest)
	if open := strings.IndexByte(date, '('); open >= 0 {
		if !strings.HasSuffix(date, ")") {
			return "", "", false
		}
		date, phrase = strings.TrimSpace(date[:open]), date[open+1:len(date)-1]
	}
	if strings.ContainsAny(date, "()") {
		return "", "", false
	}
	if _, err := gedcom.ParseDate(date); err != nil {
		return "", "", false
	}
	return date, phrase, true
}

// foldV7Phrase folds a GEDCOM 7.0 PHRASE into its DATE value for 5.5/5.5.1:
// "INT <value> (<phrase>)" for a single date with no modifier, "(<phrase>)"
// for an empty value. It reports false when the 5.x grammar has no form for
// the pair: a range, period or approximation, a value that does not parse,
// or a phrase that is empty or holds a parenthesis or line break.
func foldV7Phrase(value, phrase string) (string, bool) {
	if phrase == "" || strings.ContainsAny(phrase, "()\r\n") {
		return "", false
	}
	v := strings.TrimSpace(value)
	if v == "" {
		return "(" + phrase + ")", true
	}
	parsed, err := gedcom.ParseDate(v)
	if err != nil || parsed.Modifier != gedcom.ModifierNone || parsed.IsPhrase || parsed.EndDate != nil {
		return "", false
	}
	return "INT " + v + " (" + phrase + ")", true
}

// hasV5PhraseSyntax reports whether a date value holds syntax GEDCOM 7.0 has
// no form for: a parenthesis or an INT modifier.
func hasV5PhraseSyntax(value string) bool {
	keyword, _, _ := strings.Cut(strings.TrimSpace(value), " ")
	return strings.ContainsAny(value, "()") || strings.EqualFold(keyword, "INT")
}

// translateEpochs returns value with every B.C. epoch outside a date phrase
// spelled as target spells it: BCE for 7.0, B.C. for 5.5/5.5.1.
func translateEpochs(value string, target gedcom.Version) string {
	want := "B.C."
	if target == gedcom.Version70 {
		want = "BCE"
	}
	if !strings.ContainsAny(value, "Bb") {
		return value // fast path: no epoch to rewrite
	}
	result, _ := rewriteOutsidePhrases(value, func(s string) (string, bool) {
		words := strings.Split(s, " ")
		for i, word := range words {
			if word != want && isEpoch(word) {
				words[i] = want
			}
		}
		return strings.Join(words, " "), false
	})
	return result
}

// isEpoch reports whether word is a B.C. epoch in any spelling
// gedcom.ParseDate accepts.
func isEpoch(word string) bool {
	switch strings.ToUpper(word) {
	case "BC", "B.C.", "BCE", "B.C.E.":
		return true
	}
	return false
}

// hasDualYear reports whether a date value holds a dual year: a word of
// digits, a slash and digits (1731/32) outside any parenthesized date phrase.
// It scans the string rather than reading gedcom.Date.DualYear because most
// dates it sees are raw DATE tag values, which carry no parsed Date.
func hasDualYear(value string) bool {
	for value != "" {
		open := strings.IndexByte(value, '(')
		if open < 0 {
			open = len(value)
		}
		for _, word := range strings.Fields(value[:open]) {
			year, suffix, found := strings.Cut(word, "/")
			if found && isDigits(year) && isDigits(suffix) {
				return true
			}
		}
		value = value[open:]
		closing := strings.IndexByte(value, ')')
		if closing < 0 {
			return false
		}
		value = value[closing+1:]
	}
	return false
}

// isDigits reports whether s is a non-empty run of ASCII digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isHeaderExactDate reports whether p is HEAD.DATE or HEAD.SOUR.DATA.DATE,
// the header dates GEDCOM 7.0 types as DateExact, which has no calendar.
func (p datePath) isHeaderExactDate() bool {
	if p.recordType != "HEAD" {
		return false
	}
	path := strings.Join(p.tags, ".")
	return path == "DATE" || path == "SOUR.DATA.DATE"
}

// isExactDate reports whether p is a date GEDCOM 7.0 types as DateExact,
// which takes no PHRASE: a header date (isHeaderExactDate) or the DATE of a
// CHAN or CREA structure.
func (p datePath) isExactDate() bool {
	if p.isHeaderExactDate() {
		return true
	}
	n := len(p.tags)
	return n >= 2 && (p.tags[n-2] == "CHAN" || p.tags[n-2] == "CREA")
}

// hasNonGregorianCalendar reports whether a GEDCOM 7.0 date value opens with a
// calendar keyword other than GREGORIAN: a standard one or an extension (_XXX).
// GREGORIAN is exempt by design (#617 scopes the flag to non-Gregorian
// calendars): its date already reads correctly as Gregorian, although a strict
// DateExact has no keyword at all, so a GREGORIAN prefix is left unflagged.
func hasNonGregorianCalendar(value string) bool {
	first, _, _ := strings.Cut(strings.TrimSpace(value), " ")
	if first == "GREGORIAN" {
		return false
	}
	_, standard := calendarKeywordToEscape[first]
	return standard || strings.HasPrefix(first, "_")
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
	return rewriteOutsidePhrases(value, rewrite)
}

// rewriteOutsidePhrases applies rewrite to each run of value outside a
// parenthesized date phrase, copying the phrases unchanged, and reports
// whether any call of rewrite did. A phrase ends at its first ')' (see
// translateDateCalendars); an unclosed one runs to the end of value.
func rewriteOutsidePhrases(value string, rewrite func(string) (string, bool)) (result string, flagged bool) {
	var b strings.Builder
	for value != "" {
		open := strings.IndexByte(value, '(')
		if open < 0 {
			open = len(value)
		}
		outside, f := rewrite(value[:open])
		b.WriteString(outside)
		flagged = flagged || f
		value = value[open:]

		closing := strings.IndexByte(value, ')')
		if closing < 0 {
			closing = len(value) - 1
		}
		b.WriteString(value[:closing+1])
		value = value[closing+1:]
	}
	return b.String(), flagged
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
