package converter

import (
	"slices"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/concsplit"
)

// transformTextForVersion handles CONC/CONT transformation based on target version.
func transformTextForVersion(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	switch targetVersion {
	case gedcom.Version70:
		// GEDCOM 7.0: fold CONC and CONT into values in a single pass
		consolidateCONCAndCONT(doc, report)
	case gedcom.Version55, gedcom.Version551:
		// Downgrade: Convert embedded newlines back to CONT
		expandNewlinesToCONT(doc, report)
	}
}

// consolidateCONCAndCONT merges both CONC and CONT continuation tags into parent values.
// CONC tags concatenate directly (no separator), CONT tags add a newline.
// Both can be interleaved in GEDCOM 5.x files.
// GEDCOM 7.0 removes CONC but keeps CONT. A folded CONT becomes an embedded
// newline, which the encoder writes back out as the same CONT line, so only
// CONC is reported: the report describes the output, not this internal step.
func consolidateCONCAndCONT(doc *gedcom.Document, report *gedcom.ConversionReport) {
	concCount := 0

	// Process header tags
	if doc.Header != nil {
		var c1 int
		var notes []consolidationNote
		doc.Header.Tags, c1, _, notes = consolidateCONCAndCONTInTagsWithNotes(doc.Header.Tags)
		concCount += c1
		// Add per-item notes for header
		for _, n := range notes {
			path := BuildNestedPath("HEAD", "", n.parentTag)
			report.AddNormalized(gedcom.ConversionNote{
				Path:     path,
				Original: n.original,
				Result:   n.result,
				Reason:   n.reason,
			})
		}
	}

	// Process all record tags
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		var c1 int
		var notes []consolidationNote
		// The level-1 CONC/CONT lines of a record continue its level-0 value
		// (a 5.x NOTE record's text), which is not one of its Tags.
		if fold, ok := foldRecordValueContinuations(record); ok && fold.concCount > 0 {
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildRecordPath(string(record.Type), record.XRef),
				Original: fold.original,
				Result:   fold.result,
				Reason:   concConsolidationReason,
			})
			concCount += fold.concCount
		}
		record.Tags, c1, _, notes = consolidateCONCAndCONTInTagsWithNotes(record.Tags)
		concCount += c1
		// Add per-item notes for this record
		for _, n := range notes {
			path := BuildNestedPath(string(record.Type), record.XRef, n.parentTag)
			report.AddNormalized(gedcom.ConversionNote{
				Path:     path,
				Original: n.original,
				Result:   n.result,
				Reason:   n.reason,
			})
		}
	}

	if concCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "CONC_REMOVED",
			Description: "Consolidated CONC continuation tags into parent values",
			Count:       concCount,
		})
	}
}

// consolidationNote captures details about a single CONC/CONT consolidation for per-item tracking.
type consolidationNote struct {
	parentTag string
	original  string
	result    string
	reason    string
}

// consolidateCONCAndCONTInTags processes a flat slice of tags, consolidating both
// CONC and CONT tags into their parent values in a single pass.
// Returns the new tag slice, CONC count, and CONT count.
func consolidateCONCAndCONTInTags(tags []*gedcom.Tag) (result []*gedcom.Tag, concCount, contCount int) {
	result, concCount, contCount, _ = consolidateCONCAndCONTInTagsWithNotes(tags)
	return result, concCount, contCount
}

// consolidateCONCAndCONTInTagsWithNotes processes a flat slice of tags, consolidating both
// CONC and CONT tags into their parent values in a single pass.
// Returns the new tag slice, CONC count, CONT count, and per-item notes. A note
// is recorded only for a value that absorbed a CONC: a CONT-only value is
// written back out unchanged.
func consolidateCONCAndCONTInTagsWithNotes(tags []*gedcom.Tag) (result []*gedcom.Tag, concCount, contCount int, notes []consolidationNote) {
	if len(tags) == 0 {
		return tags, 0, 0, nil
	}

	// Every tag is visited, so continuations nested at any depth are folded
	// into their own parent; consumed marks the continuation lines already
	// folded, which are dropped from the result.
	consumed := make([]bool, len(tags))
	for i, tag := range tags {
		if consumed[i] {
			continue
		}
		if tag == nil {
			// A nil tag has no level, so it neither owns continuations nor is
			// one. Keep it where the caller put it rather than shortening the
			// slice.
			result = append(result, nil)
			continue
		}
		baseLevel := tag.Level
		originalValue := tag.Value

		// Look ahead for CONC/CONT children at baseLevel+1
		var valueBuilder strings.Builder
		valueBuilder.WriteString(tag.Value)
		localConcCount := 0
		localContCount := 0

		for j := i + 1; j < len(tags); j++ {
			nextTag := tags[j]
			// A nil tag is never a continuation; it stays in place.
			if nextTag == nil {
				continue
			}
			// If level goes back to baseLevel or lower, we're done with children
			if nextTag.Level <= baseLevel {
				break
			}
			// Process CONC/CONT at exactly baseLevel+1
			if nextTag.Level != baseLevel+1 {
				continue
			}
			switch nextTag.Tag {
			case "CONC":
				valueBuilder.WriteString(nextTag.Value)
				localConcCount++
				consumed[j] = true
			case "CONT":
				valueBuilder.WriteString("\n")
				valueBuilder.WriteString(nextTag.Value)
				localContCount++
				consumed[j] = true
			}
		}

		// Update tag value if we found continuation tags
		if localConcCount+localContCount > 0 {
			tag.Value = valueBuilder.String()
			concCount += localConcCount
			contCount += localContCount
		}
		if localConcCount > 0 {
			notes = append(notes, consolidationNote{
				parentTag: tag.Tag,
				original:  originalValue,
				result:    tag.Value,
				reason:    concConsolidationReason,
			})
		}

		result = append(result, tag)
	}

	return result, concCount, contCount, notes
}

// concConsolidationReason is the Normalized note reason for a value that
// absorbed a CONC. A value that absorbed only CONT gets no note: 7.0 keeps
// CONT and the encoder writes each embedded newline back out as one.
const concConsolidationReason = "GEDCOM 7.0 removes CONC tags; text concatenated into parent value"

// recordValueFold describes a fold of a record's level-0 continuations.
type recordValueFold struct {
	original, result string
	concCount        int
}

// foldRecordValueContinuations moves each level-1 CONC/CONT of record.Tags
// into record.Value: those lines continue the level-0 line, such as a GEDCOM
// 5.x NOTE record's text, which consolidateCONCAndCONTInTagsWithNotes never
// sees because the level-0 line is not a tag. It reports whether any was found.
func foldRecordValueContinuations(record *gedcom.Record) (recordValueFold, bool) {
	fold := recordValueFold{original: record.Value}
	if !slices.ContainsFunc(record.Tags, isRecordValueContinuation) {
		return fold, false
	}
	// A Builder, not +=, keeps a long note's fold linear in its size.
	var value strings.Builder
	value.WriteString(record.Value)
	kept := make([]*gedcom.Tag, 0, len(record.Tags))
	for _, tag := range record.Tags {
		switch {
		case !isRecordValueContinuation(tag):
			kept = append(kept, tag)
		case tag.Tag == "CONC":
			value.WriteString(tag.Value)
			fold.concCount++
		default:
			value.WriteByte('\n')
			value.WriteString(tag.Value)
		}
	}
	record.Value = value.String()
	record.Tags = kept
	fold.result = record.Value
	return fold, true
}

// isRecordValueContinuation reports whether tag is a level-1 CONC or CONT,
// which continues its record's level-0 line.
func isRecordValueContinuation(tag *gedcom.Tag) bool {
	return tag != nil && tag.Level == 1 && (tag.Tag == "CONC" || tag.Tag == "CONT")
}

// consolidateCONC is provided for explicit CONC-only consolidation.
// Most callers should use consolidateCONCAndCONT for GEDCOM 7.0 conversion.
func consolidateCONC(doc *gedcom.Document, report *gedcom.ConversionReport) {
	concCount := 0

	// Process header tags
	if doc.Header != nil {
		var c int
		doc.Header.Tags, c = consolidateCONCOnlyInTags(doc.Header.Tags)
		concCount += c
	}

	// Process all record tags
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		var c int
		record.Tags, c = consolidateCONCOnlyInTags(record.Tags)
		concCount += c
	}

	if concCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "CONC_REMOVED",
			Description: "Consolidated CONC continuation tags into parent values",
			Count:       concCount,
		})
	}
}

// consolidateCONCOnlyInTags processes tags and consolidates only CONC tags.
// CONT tags are preserved.
func consolidateCONCOnlyInTags(tags []*gedcom.Tag) (result []*gedcom.Tag, concCount int) {
	if len(tags) == 0 {
		return tags, 0
	}

	i := 0
	for i < len(tags) {
		tag := tags[i]
		if tag == nil {
			result = append(result, nil)
			i++
			continue
		}
		baseLevel := tag.Level

		// Look ahead for CONC children at baseLevel+1
		var valueBuilder strings.Builder
		valueBuilder.WriteString(tag.Value)
		j := i + 1
		foundCONC := false

		// Collect non-CONC children to preserve
		var otherChildren []*gedcom.Tag

		for j < len(tags) {
			nextTag := tags[j]
			// A nil tag is never a continuation; preserve it in place.
			if nextTag == nil {
				otherChildren = append(otherChildren, nil)
				j++
				continue
			}
			// If level goes back to baseLevel or lower, we're done with children
			if nextTag.Level <= baseLevel {
				break
			}
			// Process CONC at exactly baseLevel+1
			if nextTag.Level == baseLevel+1 && nextTag.Tag == "CONC" {
				valueBuilder.WriteString(nextTag.Value)
				concCount++
				foundCONC = true
				j++
				continue
			}
			// Not a CONC tag, keep it
			otherChildren = append(otherChildren, nextTag)
			j++
		}

		// Update tag value if we found CONC tags
		if foundCONC {
			tag.Value = valueBuilder.String()
		}

		result = append(result, tag)
		result = append(result, otherChildren...)

		i = j
	}

	return result, concCount
}

// expansionNote captures details about a single value expansion for per-item tracking.
type expansionNote struct {
	tag      string
	original string
	result   string
	reason   string
}

// Normalized note reasons for a value expanded on a downgrade to 5.x.
const (
	contExpansionReason = "GEDCOM 5.x uses CONT tags for line continuation instead of embedded newlines"
	concSplitReason     = "GEDCOM 5.x limits a line to 255 characters; the value continues on CONC lines"
)

// expandNewlinesToCONT writes each value back in 5.x continuation form when
// downgrading from GEDCOM 7.0: an embedded newline becomes a CONT line, and a
// line longer than 5.x allows (see maxValueLength) continues on CONC lines:
// 5.x limits a line to 255 characters and 7.0 has no such limit (issue #671).
// The encoder writes raw tags verbatim, so this is the only place a decoded
// 7.0 value gets split.
func expandNewlinesToCONT(doc *gedcom.Document, report *gedcom.ConversionReport) {
	contCount, concCount := 0, 0
	addNotes := func(recordType, xref string, notes []expansionNote) {
		for _, n := range notes {
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(recordType, xref, n.tag),
				Original: n.original,
				Result:   n.result,
				Reason:   n.reason,
			})
		}
	}

	// Process header tags
	if doc.Header != nil {
		var cont, conc int
		var notes []expansionNote
		doc.Header.Tags, cont, conc, notes = expandNewlinesInTagsWithNotes(doc.Header.Tags)
		contCount += cont
		concCount += conc
		addNotes("HEAD", "", notes)
	}

	// Process all record tags
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		original := record.Value
		if cont, conc := splitLongRecordValue(record); conc > 0 {
			contCount += cont
			concCount += conc
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildRecordPath(string(record.Type), record.XRef),
				Original: original,
				Result:   record.Value,
				Reason:   concSplitReason,
			})
		}
		var cont, conc int
		var notes []expansionNote
		record.Tags, cont, conc, notes = expandNewlinesInTagsWithNotes(record.Tags)
		contCount += cont
		concCount += conc
		addNotes(string(record.Type), record.XRef, notes)
	}

	if contCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "CONT_EXPANDED",
			Description: "Expanded embedded newlines to CONT tags",
			Count:       contCount,
		})
	}
	if concCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "CONC_ADDED",
			Description: "Split over-length values onto CONC tags",
			Count:       concCount,
		})
	}
}

// splitLongRecordValue moves the continuation of an over-length level-0
// value, such as a 7.0 SNOTE record's text, onto level-1 CONT/CONC tags at the
// front of record.Tags, where a decoded 5.x NOTE record keeps them. It returns
// the CONT and CONC tags it added, and adds none to a value with no line over
// the limit: the encoder already writes a line break in it as CONT.
//
// A record with no Tags but an Entity is left alone, unless it is a note: the
// encoder writes it from its entity only while Tags is empty, so adding tags
// would drop it. A NOTE or SNOTE record is the exception. Its text is the
// level-0 value split here, and the encoder writes nothing else from a NOTE
// entity, which is what an SNOTE becomes later in the downgrade. Leaving it
// alone wrote a one-line shared note past the limit.
func splitLongRecordValue(record *gedcom.Record) (contCount, concCount int) {
	isNote := record.Type == gedcom.RecordTypeNote || record.Type == gedcom.RecordTypeSharedNote
	if len(record.Tags) == 0 && record.Entity != nil && !isNote {
		return 0, 0
	}
	maxLen := maxValueLength(0, record.XRef, string(record.Type), 1)
	if !slices.ContainsFunc(splitLines(record.Value), func(line string) bool {
		return len(line) > maxLen
	}) {
		return 0, 0
	}
	first, tags, contCount, concCount := continuationTags(record.Value, maxLen, 1)
	record.Value = first
	record.Tags = append(tags, record.Tags...)
	return contCount, concCount
}

// expandNewlinesInTags processes a flat slice of tags and expands newlines to CONT tags.
// It returns the new tag slice and the count of created CONT tags.
func expandNewlinesInTags(tags []*gedcom.Tag) (result []*gedcom.Tag, contCount int) {
	result, contCount, _, _ = expandNewlinesInTagsWithNotes(tags)
	return result, contCount
}

// expandNewlinesInTagsWithNotes processes a flat slice of tags, expanding
// embedded newlines to CONT tags and splitting over-length lines onto CONC
// tags. It returns the new tag slice, the counts of created CONT and CONC
// tags, and per-item notes.
func expandNewlinesInTagsWithNotes(tags []*gedcom.Tag) (result []*gedcom.Tag, contCount, concCount int, notes []expansionNote) {
	if len(tags) == 0 {
		return tags, 0, 0, nil
	}

	for _, tag := range tags {
		// A nil tag has no value to expand; preserve it in place.
		if tag == nil {
			result = append(result, nil)
			continue
		}
		originalValue := tag.Value
		// A CONT or CONC line is itself a continuation, so the lines that
		// continue it are its siblings, not its children.
		level := tag.Level + 1
		if tag.Tag == "CONT" || tag.Tag == "CONC" {
			level = tag.Level
		}
		maxLen := maxValueLength(tag.Level, tag.XRef, tag.Tag, level)
		first, added, cont, conc := continuationTags(tag.Value, maxLen, level)
		tag.Value = first
		result = append(result, tag)
		result = append(result, added...)
		contCount += cont
		concCount += conc

		// Record the expansion
		if cont > 0 {
			notes = append(notes, expansionNote{
				tag:      tag.Tag,
				original: originalValue,
				result:   first, // The first line (remaining text now in CONT tags)
				reason:   contExpansionReason,
			})
		}
		if conc > 0 {
			notes = append(notes, expansionNote{
				tag:      tag.Tag,
				original: originalValue,
				result:   first,
				reason:   concSplitReason,
			})
		}
	}

	return result, contCount, concCount, notes
}

// continuationTags splits value into the text kept on its own line and the
// 5.x CONT and CONC lines, at level, that carry the rest: a CONT for each
// newline and a CONC for each piece of a line longer than maxLen. It returns
// how many of each it made.
func continuationTags(value string, maxLen, level int) (first string, tags []*gedcom.Tag, contCount, concCount int) {
	for i, line := range splitLines(value) {
		pieces := concsplit.Split(line, maxLen)
		if i == 0 {
			first = pieces[0]
		} else {
			tags = append(tags, &gedcom.Tag{Level: level, Tag: "CONT", Value: pieces[0]})
			contCount++
		}
		for _, piece := range pieces[1:] {
			tags = append(tags, &gedcom.Tag{Level: level, Tag: "CONC", Value: piece})
			concCount++
		}
	}
	return first, tags, contCount, concCount
}

// maxLineLength is GEDCOM 5.x's limit on the characters in a line.
const maxLineLength = 255

// maxValueLength returns the longest value that keeps both the line opened by
// level, xref and tag and its CONT/CONC lines at contLevel within
// maxLineLength, capped at [concsplit.MaxValueLength]. Only a long XRef, tag
// or level lowers it: "0 @N1@ NOTE " leaves 243 bytes, not 248. It is at
// least 1: a prefix that alone fills the line cannot be fixed by splitting,
// and a length below 1 would leave nothing to put on each line.
func maxValueLength(level int, xref, tag string, contLevel int) int {
	prefix := func(level int, xref, tag string) int {
		n := len(strconv.Itoa(level)) + 1 + len(tag) + 1
		if xref != "" {
			n += len(xref) + 1
		}
		return n
	}
	return max(1, min(concsplit.MaxValueLength,
		maxLineLength-prefix(level, xref, tag),
		maxLineLength-prefix(contLevel, "", "CONC")))
}

// splitLines splits value on its embedded newlines, returning at least one
// element.
func splitLines(value string) []string {
	return strings.Split(value, "\n")
}
