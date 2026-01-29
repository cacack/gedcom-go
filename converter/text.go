package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// transformTextForVersion handles CONC/CONT transformation based on target version.
func transformTextForVersion(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	switch targetVersion {
	case gedcom.Version70:
		// GEDCOM 7.0: Remove CONC and CONT, embed newlines in a single pass
		consolidateCONCAndCONT(doc, report)
	case gedcom.Version55, gedcom.Version551:
		// Downgrade: Convert embedded newlines back to CONT
		expandNewlinesToCONT(doc, report)
	}
}

// consolidateCONCAndCONT merges both CONC and CONT continuation tags into parent values.
// CONC tags concatenate directly (no separator), CONT tags add a newline.
// Both can be interleaved in GEDCOM 5.x files.
// In GEDCOM 7.0, CONC is removed entirely and CONT is replaced with embedded newlines.
func consolidateCONCAndCONT(doc *gedcom.Document, report *gedcom.ConversionReport) {
	concCount := 0
	contCount := 0

	// Process header tags
	if doc.Header != nil {
		var c1, c2 int
		var notes []consolidationNote
		doc.Header.Tags, c1, c2, notes = consolidateCONCAndCONTInTagsWithNotes(doc.Header.Tags)
		concCount += c1
		contCount += c2
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
		var c1, c2 int
		var notes []consolidationNote
		record.Tags, c1, c2, notes = consolidateCONCAndCONTInTagsWithNotes(record.Tags)
		concCount += c1
		contCount += c2
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

	if contCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "CONT_CONVERTED",
			Description: "Converted CONT tags to embedded newlines",
			Count:       contCount,
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
// Returns the new tag slice, CONC count, CONT count, and per-item notes.
func consolidateCONCAndCONTInTagsWithNotes(tags []*gedcom.Tag) (result []*gedcom.Tag, concCount, contCount int, notes []consolidationNote) {
	if len(tags) == 0 {
		return tags, 0, 0, nil
	}

	i := 0
	for i < len(tags) {
		tag := tags[i]
		baseLevel := tag.Level
		originalValue := tag.Value

		// Look ahead for CONC/CONT children at baseLevel+1
		var valueBuilder strings.Builder
		valueBuilder.WriteString(tag.Value)
		j := i + 1
		foundContinuation := false
		localConcCount := 0
		localContCount := 0

		// Collect non-continuation children to preserve
		var otherChildren []*gedcom.Tag

		for j < len(tags) {
			nextTag := tags[j]
			// If level goes back to baseLevel or lower, we're done with children
			if nextTag.Level <= baseLevel {
				break
			}
			// Process CONC/CONT at exactly baseLevel+1
			if nextTag.Level == baseLevel+1 {
				switch nextTag.Tag {
				case "CONC":
					valueBuilder.WriteString(nextTag.Value)
					concCount++
					localConcCount++
					foundContinuation = true
					j++
					continue
				case "CONT":
					valueBuilder.WriteString("\n")
					valueBuilder.WriteString(nextTag.Value)
					contCount++
					localContCount++
					foundContinuation = true
					j++
					continue
				}
			}
			// Not a continuation tag at immediate child level, keep it
			otherChildren = append(otherChildren, nextTag)
			j++
		}

		// Update tag value if we found continuation tags
		if foundContinuation {
			tag.Value = valueBuilder.String()

			// Create a note for this consolidation
			var reason string
			switch {
			case localConcCount > 0 && localContCount > 0:
				reason = "GEDCOM 7.0 removes CONC tags and uses embedded newlines instead of CONT tags"
			case localConcCount > 0:
				reason = "GEDCOM 7.0 removes CONC tags; text concatenated into parent value"
			default:
				reason = "GEDCOM 7.0 uses embedded newlines instead of CONT tags"
			}
			notes = append(notes, consolidationNote{
				parentTag: tag.Tag,
				original:  originalValue,
				result:    tag.Value,
				reason:    reason,
			})
		}

		result = append(result, tag)

		// Add other children back (non-CONC/CONT tags)
		result = append(result, otherChildren...)

		// Move to next unprocessed tag
		i = j
	}

	return result, concCount, contCount, notes
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

// convertCONTToNewlines replaces CONT tags with newline characters in parent values.
// This is typically called after consolidateCONC, or use consolidateCONCAndCONT
// for a single-pass conversion.
func convertCONTToNewlines(doc *gedcom.Document, report *gedcom.ConversionReport) {
	contCount := 0

	// Process header tags
	if doc.Header != nil {
		var c int
		doc.Header.Tags, c = convertCONTOnlyInTags(doc.Header.Tags)
		contCount += c
	}

	// Process all record tags
	for _, record := range doc.Records {
		var c int
		record.Tags, c = convertCONTOnlyInTags(record.Tags)
		contCount += c
	}

	if contCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "CONT_CONVERTED",
			Description: "Converted CONT tags to embedded newlines",
			Count:       contCount,
		})
	}
}

// convertCONTOnlyInTags processes tags and converts only CONT tags to newlines.
func convertCONTOnlyInTags(tags []*gedcom.Tag) (result []*gedcom.Tag, contCount int) {
	if len(tags) == 0 {
		return tags, 0
	}

	i := 0
	for i < len(tags) {
		tag := tags[i]
		baseLevel := tag.Level

		// Look ahead for CONT children at baseLevel+1
		var valueBuilder strings.Builder
		valueBuilder.WriteString(tag.Value)
		j := i + 1
		foundCONT := false

		// Collect non-CONT children to preserve
		var otherChildren []*gedcom.Tag

		for j < len(tags) {
			nextTag := tags[j]
			// If level goes back to baseLevel or lower, we're done with children
			if nextTag.Level <= baseLevel {
				break
			}
			// Process CONT at exactly baseLevel+1
			if nextTag.Level == baseLevel+1 && nextTag.Tag == "CONT" {
				valueBuilder.WriteString("\n")
				valueBuilder.WriteString(nextTag.Value)
				contCount++
				foundCONT = true
				j++
				continue
			}
			// Not a CONT tag, keep it
			otherChildren = append(otherChildren, nextTag)
			j++
		}

		// Update tag value if we found CONT tags
		if foundCONT {
			tag.Value = valueBuilder.String()
		}

		result = append(result, tag)
		result = append(result, otherChildren...)

		i = j
	}

	return result, contCount
}

// expansionNote captures details about a single newline-to-CONT expansion for per-item tracking.
type expansionNote struct {
	tag      string
	original string
	result   string
}

// expandNewlinesToCONT converts embedded newlines back to CONT tags.
// This is used when downgrading from GEDCOM 7.0 to 5.x.
func expandNewlinesToCONT(doc *gedcom.Document, report *gedcom.ConversionReport) {
	contCount := 0

	// Process header tags
	if doc.Header != nil {
		var notes []expansionNote
		doc.Header.Tags, contCount, notes = expandNewlinesInTagsWithNotes(doc.Header.Tags)
		// Add per-item notes for header
		for _, n := range notes {
			path := BuildNestedPath("HEAD", "", n.tag)
			report.AddNormalized(gedcom.ConversionNote{
				Path:     path,
				Original: n.original,
				Result:   n.result,
				Reason:   "GEDCOM 5.x uses CONT tags for line continuation instead of embedded newlines",
			})
		}
	}

	// Process all record tags
	for _, record := range doc.Records {
		var c int
		var notes []expansionNote
		record.Tags, c, notes = expandNewlinesInTagsWithNotes(record.Tags)
		contCount += c
		// Add per-item notes for this record
		for _, n := range notes {
			path := BuildNestedPath(string(record.Type), record.XRef, n.tag)
			report.AddNormalized(gedcom.ConversionNote{
				Path:     path,
				Original: n.original,
				Result:   n.result,
				Reason:   "GEDCOM 5.x uses CONT tags for line continuation instead of embedded newlines",
			})
		}
	}

	if contCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "CONT_EXPANDED",
			Description: "Expanded embedded newlines to CONT tags",
			Count:       contCount,
		})
	}
}

// expandNewlinesInTags processes a flat slice of tags and expands newlines to CONT tags.
// It returns the new tag slice and the count of created CONT tags.
func expandNewlinesInTags(tags []*gedcom.Tag) (result []*gedcom.Tag, contCount int) {
	result, contCount, _ = expandNewlinesInTagsWithNotes(tags)
	return result, contCount
}

// expandNewlinesInTagsWithNotes processes a flat slice of tags and expands newlines to CONT tags.
// It returns the new tag slice, the count of created CONT tags, and per-item notes.
func expandNewlinesInTagsWithNotes(tags []*gedcom.Tag) (result []*gedcom.Tag, contCount int, notes []expansionNote) {
	if len(tags) == 0 {
		return tags, 0, nil
	}

	for _, tag := range tags {
		// Check if this tag has embedded newlines
		if strings.Contains(tag.Value, "\n") {
			originalValue := tag.Value
			lines := strings.Split(tag.Value, "\n")
			tag.Value = lines[0]
			result = append(result, tag)

			// Create CONT tags for remaining lines
			for _, line := range lines[1:] {
				result = append(result, &gedcom.Tag{
					Level: tag.Level + 1,
					Tag:   "CONT",
					Value: line,
				})
				contCount++
			}

			// Record the expansion
			notes = append(notes, expansionNote{
				tag:      tag.Tag,
				original: originalValue,
				result:   lines[0], // The first line (remaining text now in CONT tags)
			})
		} else {
			result = append(result, tag)
		}
	}

	return result, contCount, notes
}
