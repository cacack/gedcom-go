// Package converter provides GEDCOM version conversion functionality.
//
// This package converts GEDCOM documents between versions 5.5, 5.5.1, and 7.0.
// It handles all necessary transformations including encoding changes, tag
// modifications, and XRef normalization.
//
// Basic usage:
//
//	doc, _ := decoder.Decode(reader)
//	converted, report, err := converter.Convert(doc, gedcom.Version70)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(report)
//
// The converter returns a ConversionReport detailing all transformations
// and any data loss that occurred during conversion.
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
		validateConverted(converted, report)
	}

	report.Success = true
	return converted, report, nil
}

// convert55To551 converts GEDCOM 5.5 to 5.5.1.
func convert55To551(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformHeader(doc, gedcom.Version551, report)
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_UPGRADE",
		Description: "Upgraded from GEDCOM 5.5 to 5.5.1 (backward compatible)",
		Count:       1,
	})
	return nil
}

// convert55To70 converts GEDCOM 5.5 to 7.0.
func convert55To70(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version70, report)
	normalizeXRefsToUppercase(doc, report)
	transformMediaTypes(doc, gedcom.Version70, report)
	transformHeader(doc, gedcom.Version70, report)
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_UPGRADE",
		Description: "Upgraded from GEDCOM 5.5 to 7.0",
		Count:       1,
	})
	return nil
}

// convert551To55 converts GEDCOM 5.5.1 to 5.5.
func convert551To55(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformHeader(doc, gedcom.Version55, report)
	record551Tags(doc, report)
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_DOWNGRADE",
		Description: "Downgraded from GEDCOM 5.5.1 to 5.5",
		Count:       1,
	})
	return nil
}

// convert551To70 converts GEDCOM 5.5.1 to 7.0.
func convert551To70(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version70, report)
	normalizeXRefsToUppercase(doc, report)
	transformMediaTypes(doc, gedcom.Version70, report)
	transformHeader(doc, gedcom.Version70, report)
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_UPGRADE",
		Description: "Upgraded from GEDCOM 5.5.1 to 7.0",
		Count:       1,
	})
	return nil
}

// convert70To55 converts GEDCOM 7.0 to 5.5.
func convert70To55(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version55, report)
	transformMediaTypes(doc, gedcom.Version55, report)
	transformHeader(doc, gedcom.Version55, report)
	record70DataLoss(doc, report, gedcom.Version55)
	report.AddTransformation(gedcom.Transformation{
		Type:        "VERSION_DOWNGRADE",
		Description: "Downgraded from GEDCOM 7.0 to 5.5",
		Count:       1,
	})
	return nil
}

// convert70To551 converts GEDCOM 7.0 to 5.5.1.
func convert70To551(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	transformTextForVersion(doc, gedcom.Version551, report)
	transformMediaTypes(doc, gedcom.Version551, report)
	transformHeader(doc, gedcom.Version551, report)
	record70DataLoss(doc, report, gedcom.Version551)
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

	for _, record := range doc.Records {
		countTagsInRecord(record.Tags, tags551, found)
	}

	for tag, count := range found {
		if count > 0 {
			report.AddDataLoss(gedcom.DataLossItem{
				Feature: tag + " tags",
				Reason:  "Tag introduced in GEDCOM 5.5.1, may not be recognized by strict 5.5 parsers",
			})
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
