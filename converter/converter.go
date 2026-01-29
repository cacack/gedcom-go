package converter

import (
	"fmt"

	"github.com/cacack/gedcom-go/gedcom"
)

// Convert converts a GEDCOM document to the target version using default options.
func Convert(doc *gedcom.Document, targetVersion gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, error) {
	return ConvertWithOptions(doc, targetVersion, DefaultOptions())
}

// ConvertWithOptions converts a GEDCOM document with custom options.
//
//nolint:gocyclo // Routing to 6 conversion paths requires this branching structure
func ConvertWithOptions(doc *gedcom.Document, targetVersion gedcom.Version, opts *ConvertOptions) (*gedcom.Document, *gedcom.ConversionReport, error) {
	if doc == nil {
		return nil, nil, fmt.Errorf("document is nil")
	}
	if opts == nil {
		opts = DefaultOptions()
	}
	if !targetVersion.IsValid() {
		return nil, nil, fmt.Errorf("invalid target version: %s", targetVersion)
	}

	sourceVersion := doc.Header.Version
	if sourceVersion == "" {
		sourceVersion = gedcom.Version55 // Default assumption
	}

	// No conversion needed if versions match
	if sourceVersion == targetVersion {
		return doc, &gedcom.ConversionReport{
			SourceVersion: sourceVersion,
			TargetVersion: targetVersion,
			Success:       true,
		}, nil
	}

	// Deep copy to avoid mutating original
	converted := deepCopyDocument(doc)

	// Create report
	report := &gedcom.ConversionReport{
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
	}

	// Route to appropriate converter
	var err error
	switch {
	case sourceVersion == gedcom.Version55 && targetVersion == gedcom.Version551:
		err = convert55To551(converted, report, opts)
	case sourceVersion == gedcom.Version55 && targetVersion == gedcom.Version70:
		err = convert55To70(converted, report, opts)
	case sourceVersion == gedcom.Version551 && targetVersion == gedcom.Version55:
		err = convert551To55(converted, report, opts)
	case sourceVersion == gedcom.Version551 && targetVersion == gedcom.Version70:
		err = convert551To70(converted, report, opts)
	case sourceVersion == gedcom.Version70 && targetVersion == gedcom.Version55:
		err = convert70To55(converted, report, opts)
	case sourceVersion == gedcom.Version70 && targetVersion == gedcom.Version551:
		err = convert70To551(converted, report, opts)
	default:
		return nil, nil, fmt.Errorf("unsupported conversion: %s to %s", sourceVersion, targetVersion)
	}

	if err != nil {
		report.Success = false
		return nil, report, err
	}

	// Check for data loss in strict mode
	if opts.StrictDataLoss && report.HasDataLoss() {
		report.Success = false
		return nil, report, fmt.Errorf("conversion would result in data loss (strict mode enabled)")
	}

	// Update header version
	converted.Header.Version = targetVersion

	// Validate if requested
	if opts.Validate {
		_ = validateConverted(converted, report)
	}

	report.Success = true
	return converted, report, nil
}

// convert55To551 converts GEDCOM 5.5 to 5.5.1.
//
//nolint:unparam // error return kept for API consistency with other converters
func convert55To551(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformHeader(doc, gedcom.Version551, report)
	if opts.PreserveUnknownTags {
		recordPreservedUnknownTags(doc, report)
	}
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_UPGRADE",
		Description: "Upgraded from GEDCOM 5.5 to 5.5.1 (backward compatible)",
		Count:       1,
	})
	return nil
}

// convert55To70 converts GEDCOM 5.5 to 7.0.
//
//nolint:unparam // error return kept for API consistency with other converters
func convert55To70(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version70, report)
	normalizeXRefsToUppercase(doc, report)
	transformMediaTypes(doc, gedcom.Version70, report)
	transformHeader(doc, gedcom.Version70, report)
	if opts.PreserveUnknownTags {
		recordPreservedUnknownTags(doc, report)
	}
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_UPGRADE",
		Description: "Upgraded from GEDCOM 5.5 to 7.0",
		Count:       1,
	})
	return nil
}

// convert551To55 converts GEDCOM 5.5.1 to 5.5.
//
//nolint:unparam // error return kept for API consistency with other converters
func convert551To55(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformHeader(doc, gedcom.Version55, report)
	record551Tags(doc, report)
	if opts.PreserveUnknownTags {
		recordPreservedUnknownTags(doc, report)
	}
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_DOWNGRADE",
		Description: "Downgraded from GEDCOM 5.5.1 to 5.5",
		Count:       1,
	})
	return nil
}

// convert551To70 converts GEDCOM 5.5.1 to 7.0.
//
//nolint:unparam // error return kept for API consistency with other converters
func convert551To70(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version70, report)
	normalizeXRefsToUppercase(doc, report)
	transformMediaTypes(doc, gedcom.Version70, report)
	transformHeader(doc, gedcom.Version70, report)
	if opts.PreserveUnknownTags {
		recordPreservedUnknownTags(doc, report)
	}
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_UPGRADE",
		Description: "Upgraded from GEDCOM 5.5.1 to 7.0",
		Count:       1,
	})
	return nil
}

// convert70To55 converts GEDCOM 7.0 to 5.5.
//
//nolint:unparam // error return kept for API consistency with other converters
func convert70To55(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version55, report)
	transformMediaTypes(doc, gedcom.Version55, report)
	transformHeader(doc, gedcom.Version55, report)
	record70DataLoss(doc, report, gedcom.Version55)
	if opts.PreserveUnknownTags {
		recordPreservedUnknownTags(doc, report)
	}
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_DOWNGRADE",
		Description: "Downgraded from GEDCOM 7.0 to 5.5",
		Count:       1,
	})
	return nil
}

// convert70To551 converts GEDCOM 7.0 to 5.5.1.
//
//nolint:unparam // error return kept for API consistency with other converters
func convert70To551(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version551, report)
	transformMediaTypes(doc, gedcom.Version551, report)
	transformHeader(doc, gedcom.Version551, report)
	record70DataLoss(doc, report, gedcom.Version551)
	if opts.PreserveUnknownTags {
		recordPreservedUnknownTags(doc, report)
	}
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_DOWNGRADE",
		Description: "Downgraded from GEDCOM 7.0 to 5.5.1",
		Count:       1,
	})
	return nil
}

// record551Tags notes 5.5.1-specific tags that may not be recognized in 5.5.
func record551Tags(doc *gedcom.Document, report *gedcom.ConversionReport) {
	tags551 := []string{"EMAIL", "FAX", "WWW", "FACT", "MAP", "LATI", "LONG"}
	found := make(map[string]int)
	// Track per-record occurrences for granular notes
	perRecord := make(map[string]map[string]bool) // tag -> xref -> true

	for _, record := range doc.Records {
		for _, tag := range record.Tags {
			for _, target := range tags551 {
				if tag.Tag == target {
					found[target]++
					if perRecord[target] == nil {
						perRecord[target] = make(map[string]bool)
					}
					perRecord[target][record.XRef] = true
				}
			}
		}
	}

	for tag, count := range found {
		if count > 0 {
			report.AddDataLoss(gedcom.DataLossItem{
				Feature: tag + " tags",
				Reason:  "Tag introduced in GEDCOM 5.5.1, may not be recognized by strict 5.5 parsers",
			})

			// Add per-item dropped notes
			for xref := range perRecord[tag] {
				recordType := getRecordTypeByXRef(doc, xref)
				path := BuildNestedPath(recordType, xref, tag)
				report.AddDropped(gedcom.ConversionNote{
					Path:     path,
					Original: tag,
					Result:   "",
					Reason:   "Tag introduced in GEDCOM 5.5.1, may not be recognized by strict 5.5 parsers",
				})
			}
		}
	}
}

// record70DataLoss records data loss for GEDCOM 7.0-specific features.
func record70DataLoss(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	tags70 := []string{"EXID", "NO", "TRAN", "PHRASE", "UID", "CREA", "SNOTE"}
	found := make(map[string][]string)

	for _, record := range doc.Records {
		affected := findTagsInRecord(record.Tags, tags70)
		for _, tag := range affected {
			found[tag] = append(found[tag], record.XRef)
		}
	}

	for tag, xrefs := range found {
		if len(xrefs) > 0 {
			report.AddDataLoss(gedcom.DataLossItem{
				Feature:         tag + " tags",
				Reason:          "Tag not supported in GEDCOM " + targetVersion.String(),
				AffectedRecords: xrefs,
			})

			// Add per-item dropped notes
			for _, xref := range xrefs {
				recordType := getRecordTypeByXRef(doc, xref)
				path := BuildNestedPath(recordType, xref, tag)
				report.AddDropped(gedcom.ConversionNote{
					Path:     path,
					Original: tag,
					Result:   "",
					Reason:   "Tag not supported in GEDCOM " + targetVersion.String(),
				})
			}
		}
	}
}

// countTagsInRecord counts occurrences of specific tags.
func countTagsInRecord(tags []*gedcom.Tag, targetTags []string, found map[string]int) {
	for _, tag := range tags {
		for _, target := range targetTags {
			if tag.Tag == target {
				found[target]++
			}
		}
	}
}

// findTagsInRecord returns which target tags exist in the record.
func findTagsInRecord(tags []*gedcom.Tag, targetTags []string) []string {
	foundSet := make(map[string]bool)
	for _, tag := range tags {
		for _, target := range targetTags {
			if tag.Tag == target {
				foundSet[target] = true
			}
		}
	}

	var result []string
	for tag := range foundSet {
		result = append(result, tag)
	}
	return result
}

// getRecordTypeByXRef returns the record type string for a given XRef.
func getRecordTypeByXRef(doc *gedcom.Document, xref string) string {
	if record, ok := doc.XRefMap[xref]; ok {
		return string(record.Type)
	}
	return "UNKNOWN"
}

// recordPreservedUnknownTags identifies and records unknown/vendor tags that are preserved through conversion.
// Unknown tags are those starting with underscore (_) which represent vendor extensions.
func recordPreservedUnknownTags(doc *gedcom.Document, report *gedcom.ConversionReport) {
	// Process header tags
	if doc.Header != nil {
		for _, tag := range doc.Header.Tags {
			recordPreservedTagsRecursive(tag, "HEAD", "", report)
		}
	}

	// Process all record tags
	for _, record := range doc.Records {
		for _, tag := range record.Tags {
			recordPreservedTagsRecursive(tag, string(record.Type), record.XRef, report)
		}
	}
}

// recordPreservedTagsRecursive recursively checks tags for vendor extensions (starting with _).
func recordPreservedTagsRecursive(tag *gedcom.Tag, recordType, xref string, report *gedcom.ConversionReport) {
	if tag == nil {
		return
	}

	// Vendor extensions start with underscore
	if tag.Tag != "" && tag.Tag[0] == '_' {
		path := BuildNestedPath(recordType, xref, tag.Tag)
		report.AddPreserved(gedcom.ConversionNote{
			Path:     path,
			Original: tag.Tag,
			Result:   tag.Tag,
			Reason:   "Vendor extension preserved through conversion",
		})
	}
}
