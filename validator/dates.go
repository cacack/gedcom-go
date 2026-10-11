// dates.go reports dates that do not exist on their calendar, such as
// 31 FEB 1900, as CodeInvalidDate.
//
// The decoder parses such a date without complaint -- its grammar is valid --
// so gedcom.Date.Validate, which checks the day against the month and year,
// is the only thing that catches it. This check runs Validate on every DATE in
// the document, including the end date of a range or period.

package validator

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// validateDates reports every date in doc that gedcom.Date.Validate rejects,
// as a CodeInvalidDate warning. Dates Validate skips by design -- partial
// dates such as "FEB 1900" or "ABT 1900", and non-Gregorian calendars -- are
// not reported, and neither is a value ParseDate cannot parse at all (the
// decoder already reports that as a diagnostic).
//
// Raw tags are walked where a record has them, so every DATE is checked --
// events, attributes, LDS ordinances, citations, CHAN -- and the issue
// carries the DATE line. Details["tag"] is the tag holding the date (e.g.
// "BIRT") and Details["path"] the tag path below the record (e.g.
// "BIRT.SOUR.DATA.DATE"). A record with no Tags, built in code, is checked
// through its typed entity's event, attribute and LDS ordinance ParsedDates
// instead, mirroring how the encoder writes such a record from the entity.
// The header is checked the same way, with an empty RecordXRef.
func validateDates(doc *gedcom.Document) []Issue {
	var issues []Issue
	if doc.Header != nil {
		if len(doc.Header.Tags) > 0 {
			issues = appendRawDateIssues(issues, "", "HEAD", doc.Header.Tags)
		} else if doc.Header.Date != "" {
			if parsed, err := gedcom.ParseDate(doc.Header.Date); err == nil {
				issues = appendDateIssue(issues, "", "HEAD", "DATE", parsed, 0)
			}
		}
	}
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		if len(record.Tags) > 0 {
			issues = appendRawDateIssues(issues, record.XRef, string(record.Type), record.Tags)
			continue
		}
		issues = appendTypedDateIssues(issues, record)
	}
	return issues
}

// appendRawDateIssues checks every DATE tag in tags, tracking the path of
// enclosing tags so the issue can name the tag that holds the date. holder
// is the tag reported for a level-1 DATE: the record type, or HEAD.
func appendRawDateIssues(issues []Issue, recordXRef, holder string, tags []*gedcom.Tag) []Issue {
	var path []string
	for _, tag := range tags {
		if tag == nil || tag.Level < 1 {
			continue
		}
		if len(path) >= tag.Level {
			path = path[:tag.Level-1]
		}
		path = append(path, tag.Tag)
		if tag.Tag != "DATE" {
			continue
		}
		parsed, err := gedcom.ParseDate(tag.Value)
		if err != nil {
			continue
		}
		tagHolder := holder
		if len(path) >= 2 {
			tagHolder = path[len(path)-2]
		}
		issues = appendDateIssue(issues, recordXRef, tagHolder, strings.Join(path, "."), parsed, tag.LineNumber)
	}
	return issues
}

// appendTypedDateIssues checks the ParsedDates of an individual's or
// family's events, attributes and LDS ordinances.
func appendTypedDateIssues(issues []Issue, record *gedcom.Record) []Issue {
	var events []*gedcom.Event
	var attributes []*gedcom.Attribute
	var ordinances []*gedcom.LDSOrdinance
	if ind, ok := record.GetIndividual(); ok {
		events, attributes, ordinances = ind.Events, ind.Attributes, ind.LDSOrdinances
	} else if fam, ok := record.GetFamily(); ok {
		events, attributes, ordinances = fam.Events, fam.Attributes, fam.LDSOrdinances
	}
	for _, e := range events {
		if e != nil {
			issues = appendDateIssue(issues, record.XRef, string(e.Type), string(e.Type)+".DATE", e.ParsedDate, 0)
		}
	}
	for _, a := range attributes {
		if a != nil {
			issues = appendDateIssue(issues, record.XRef, string(a.Type), string(a.Type)+".DATE", a.ParsedDate, 0)
		}
	}
	for _, o := range ordinances {
		if o != nil {
			issues = appendDateIssue(issues, record.XRef, string(o.Type), string(o.Type)+".DATE", o.ParsedDate, 0)
		}
	}
	return issues
}

// appendDateIssue appends a CodeInvalidDate issue when date fails
// gedcom.Date.Validate. A nil date is skipped.
func appendDateIssue(issues []Issue, recordXRef, holder, path string, date *gedcom.Date, line int) []Issue {
	if date == nil {
		return issues
	}
	err := date.Validate()
	if err == nil {
		return issues
	}
	return append(issues, NewIssue(
		SeverityWarning,
		CodeInvalidDate,
		fmt.Sprintf("%s date %q does not exist: %v", holder, date.Original, err),
		recordXRef,
	).
		WithLineNumber(line).
		WithDetail("tag", holder).
		WithDetail("path", path).
		WithDetail("value", date.Original))
}
