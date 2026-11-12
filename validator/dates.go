// dates.go reports dates that do not exist on their calendar, such as
// 31 FEB 1900, or ranges that end before they start, as CodeInvalidDate.
//
// The decoder parses such a date without complaint -- its grammar is valid --
// so gedcom.Date.Validate, which checks the day against the month and year,
// is the only thing that catches it. This check runs Validate on every DATE in
// the document, including the end date of a range or period.

package validator

import (
	"fmt"
	"strings"
	"time"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// validateDates reports every date in doc that gedcom.Date.Validate rejects,
// as a CodeInvalidDate warning, on every calendar Validate checks: Gregorian,
// Julian, Hebrew and French Republican. Dates Validate skips by design --
// partial dates such as "FEB 1900" or "ABT 1900" -- are not reported, nor a
// leap day on a dual-dated (Old Style) endpoint such as "29 FEB 1723/24",
// nor a value ParseDate cannot parse at all (the decoder already reports
// that as a diagnostic).
//
// Raw tags are walked where a record has them, so every DATE is checked --
// events, attributes, LDS ordinances, citations, CHAN -- and the issue
// carries the DATE line. Details["tag"] is the tag holding the date (e.g.
// "BIRT") and Details["path"] the tag path below the record (e.g.
// "BIRT.SOUR.DATA.DATE"). A record with no Tags, built in code, is checked
// through its typed entity's event, attribute and LDS ordinance ParsedDates
// instead, mirroring how the encoder writes such a record from the entity;
// those issues carry LineNumber 0 and a path of just "<TYPE>.DATE". Raw tags
// are authoritative where present, as the encoder writes them verbatim. The
// header is checked the same way, with an empty RecordXRef.
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
// dateError. A nil date is skipped.
func appendDateIssue(issues []Issue, recordXRef, holder, path string, date *gedcom.Date, line int) []Issue {
	if date == nil {
		return issues
	}
	err := dateError(date)
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

// dateError is gedcom.Date.Validate, except that a dual-dated endpoint such
// as 29 FEB 1723/24 is checked only against the longest its month can be. A
// dual year is Old Style: its January to March fall in the second year, on the
// Julian calendar, so Validate's Gregorian check against the first year would
// reject real leap days. A day no year allows, such as 30 FEB 1723/24, is
// still reported, as is a dual year that is not the following year. A range
// or period that ends before it starts is reported as Validate reports it.
func dateError(date *gedcom.Date) error {
	start := *date
	start.EndDate = nil
	if err := endpointError(&start); err != nil {
		return err
	}
	end := date.EndDate
	if end == nil {
		return nil
	}
	if err := endpointError(end); err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}
	// Validate's remaining check is the order. It is not called on the whole
	// range, which would recheck a dual-dated endpoint's day strictly, so the
	// endpoints are bounded alone, as Validate bounds them.
	endOnly := *end
	start.Modifier, endOnly.Modifier, endOnly.EndDate = gedcom.ModifierNone, gedcom.ModifierNone, nil
	startEarliest, _, startOK := start.Bounds()
	_, endLatest, endOK := endOnly.Bounds()
	if !startOK || !endOK || startEarliest <= endLatest {
		return nil
	}
	if date.Original != "" {
		return fmt.Errorf("invalid date range %q: the end date is before the start date", date.Original)
	}
	return fmt.Errorf("invalid date range: the end date is before the start date")
}

// maxDaysInMonth is the longest each month (1-12) can be in any year.
var maxDaysInMonth = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// endpointError checks one date with no EndDate. A Gregorian dual-dated date
// has its dual year checked by Validate and its day checked against
// maxDaysInMonth; any other date goes to Validate, which skips partial dates
// and checks a dual Julian date's day in its New Style year.
func endpointError(d *gedcom.Date) error {
	if d.DualYear == 0 || d.Calendar != gedcom.CalendarGregorian {
		return d.Validate()
	}
	yearOnly := *d
	// Clearing Day makes a partial date, whose day Validate does not check
	// (a documented contract), so only the dual year is checked here.
	yearOnly.Day = 0
	if err := yearOnly.Validate(); err != nil {
		return err
	}
	if d.Month < 1 || d.Month > 12 || d.Day == 0 {
		return nil
	}
	if longest := maxDaysInMonth[d.Month]; d.Day > longest {
		return fmt.Errorf("invalid date: %s has at most %d days, got day %d", time.Month(d.Month), longest, d.Day)
	}
	return nil
}
