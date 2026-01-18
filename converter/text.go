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
		doc.Header.Tags, c1, c2 = consolidateCONCAndCONTInTags(doc.Header.Tags)
		concCount += c1
		contCount += c2
	}

	// Process all record tags
	for _, record := range doc.Records {
		var c1, c2 int
		record.Tags, c1, c2 = consolidateCONCAndCONTInTags(record.Tags)
		concCount += c1
		contCount += c2
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

// consolidateCONCAndCONTInTags processes a flat slice of tags, consolidating both
// CONC and CONT tags into their parent values in a single pass.
// Returns the new tag slice, CONC count, and CONT count.
func consolidateCONCAndCONTInTags(tags []*gedcom.Tag) ([]*gedcom.Tag, int, int) {
	if len(tags) == 0 {
		return tags, 0, 0
	}

	concCount := 0
	contCount := 0
	var result []*gedcom.Tag

	i := 0
	for i < len(tags) {
		tag := tags[i]
		baseLevel := tag.Level

		// Look ahead for CONC/CONT children at baseLevel+1
		var valueBuilder strings.Builder
		valueBuilder.WriteString(tag.Value)
		j := i + 1
		foundContinuation := false

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
					foundContinuation = true
					j++
					continue
				case "CONT":
					valueBuilder.WriteString("\n")
					valueBuilder.WriteString(nextTag.Value)
					contCount++
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
		}

		result = append(result, tag)

		// Add other children back (non-CONC/CONT tags)
		result = append(result, otherChildren...)

		// Move to next unprocessed tag
		i = j
	}

	return result, concCount, contCount
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
func consolidateCONCOnlyInTags(tags []*gedcom.Tag) ([]*gedcom.Tag, int) {
	if len(tags) == 0 {
		return tags, 0
	}

	concCount := 0
	var result []*gedcom.Tag

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
func convertCONTOnlyInTags(tags []*gedcom.Tag) ([]*gedcom.Tag, int) {
	if len(tags) == 0 {
		return tags, 0
	}

	contCount := 0
	var result []*gedcom.Tag

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

// expandNewlinesToCONT converts embedded newlines back to CONT tags.
// This is used when downgrading from GEDCOM 7.0 to 5.x.
func expandNewlinesToCONT(doc *gedcom.Document, report *gedcom.ConversionReport) {
	contCount := 0

	// Process header tags
	if doc.Header != nil {
		doc.Header.Tags, contCount = expandNewlinesInTags(doc.Header.Tags)
	}

	// Process all record tags
	for _, record := range doc.Records {
		c := 0
		record.Tags, c = expandNewlinesInTags(record.Tags)
		contCount += c
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
func expandNewlinesInTags(tags []*gedcom.Tag) ([]*gedcom.Tag, int) {
	if len(tags) == 0 {
		return tags, 0
	}

	contCount := 0
	var result []*gedcom.Tag

	for _, tag := range tags {
		// Check if this tag has embedded newlines
		if strings.Contains(tag.Value, "\n") {
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
		} else {
			result = append(result, tag)
		}
	}

	return result, contCount
}
