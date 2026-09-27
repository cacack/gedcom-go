package converter

import "github.com/cacack/gedcom-go/v3/gedcom"

// negativeAssertionTag is the GEDCOM 7.0 tag for a negative assertion
// (n NO <EVENT>), which has no GEDCOM 5.5/5.5.1 equivalent.
const negativeAssertionTag = "NO"

// dropNegativeAssertions removes GEDCOM 7.0 negative assertions from doc on a
// downgrade to 5.5/5.5.1, and reports each affected record as data loss.
//
// GEDCOM 5.x has no NO structure, so writing one would produce an invalid
// file. Both representations are cleared, so the result does not depend on
// which one the encoder writes from:
//   - record.Tags: every level-1 NO line together with its subordinate block
//     (DATE, NOTE, SOUR, ...), the form a decoded document carries;
//   - the entity: every event with IsNegative set in Individual.Events and
//     Family.Events, the form a document built in code carries.
//
// NO is therefore not in record70DataLoss's report-only list. Anything nested
// under a dropped NO (a SOUR pointer, a PHRASE, an SNOTE) goes with it and is
// covered by the NO entry; it runs before record70DataLoss so that sweep does
// not also report those nested tags as if they were still present.
func dropNegativeAssertions(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	var affected []string
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		droppedTags := dropNegativeAssertionTags(record)
		droppedEvents := dropNegativeAssertionEvents(record)
		if droppedTags || droppedEvents {
			affected = append(affected, record.XRef)
		}
	}
	if len(affected) == 0 {
		return
	}

	reason := "Tag not supported in GEDCOM " + targetVersion.String()
	report.AddDataLoss(gedcom.DataLossItem{
		Feature:         negativeAssertionTag + " tags",
		Reason:          reason,
		AffectedRecords: affected,
	})
	for _, xref := range affected {
		report.AddDropped(gedcom.ConversionNote{
			Path:     BuildNestedPath(getRecordTypeByXRef(doc, xref), xref, negativeAssertionTag),
			Original: negativeAssertionTag,
			Result:   "",
			Reason:   reason,
		})
	}
}

// dropNegativeAssertionTags removes each level-1 NO tag and its subordinate
// block from record.Tags, reporting whether anything was removed. It leaves
// record.Tags untouched (and allocates nothing) when there is no level-1 NO.
// A nil tag inside a removed block has no level to compare and is kept, as
// the caller's element rather than part of the NO structure.
func dropNegativeAssertionTags(record *gedcom.Record) bool {
	tags := record.Tags
	if !hasLevel1Tag(tags, negativeAssertionTag) {
		return false
	}

	out := make([]*gedcom.Tag, 0, len(tags))
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag == nil || tag.Level != 1 || tag.Tag != negativeAssertionTag {
			out = append(out, tag)
			continue
		}
		j := i + 1
		for j < len(tags) && (tags[j] == nil || tags[j].Level > tag.Level) {
			if tags[j] == nil {
				out = append(out, nil)
			}
			j++
		}
		i = j - 1
	}
	record.Tags = out
	return true
}

// dropNegativeAssertionEvents removes the negated events (IsNegative) from an
// individual's or family's Events, reporting whether anything was removed.
// Nil entries are kept, as the caller's elements. Other record types carry no
// events and are left alone.
func dropNegativeAssertionEvents(record *gedcom.Record) bool {
	var events *[]*gedcom.Event
	if ind, ok := record.GetIndividual(); ok {
		events = &ind.Events
	} else if fam, ok := record.GetFamily(); ok {
		events = &fam.Events
	} else {
		return false
	}

	kept := make([]*gedcom.Event, 0, len(*events))
	for _, event := range *events {
		if event == nil || !event.IsNegative {
			kept = append(kept, event)
		}
	}
	if len(kept) == len(*events) {
		return false
	}
	*events = kept
	return true
}

// hasLevel1Tag reports whether tags include a level-1 tag with the given name.
func hasLevel1Tag(tags []*gedcom.Tag, name string) bool {
	for _, t := range tags {
		if t != nil && t.Level == 1 && t.Tag == name {
			return true
		}
	}
	return false
}
