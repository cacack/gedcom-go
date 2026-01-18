package gedcom

import (
	"fmt"
	"strings"
)

// ConversionReport contains the results of a GEDCOM version conversion.
// It tracks all transformations applied, any data loss that occurred,
// and validation issues found after conversion.
type ConversionReport struct {
	// SourceVersion is the original GEDCOM version before conversion
	SourceVersion Version

	// TargetVersion is the GEDCOM version after conversion
	TargetVersion Version

	// Transformations lists all changes made during conversion
	Transformations []Transformation

	// DataLoss lists features that were lost during conversion (typically downgrades)
	DataLoss []DataLossItem

	// ValidationIssues lists any problems found after conversion
	ValidationIssues []string

	// Success indicates whether the conversion completed successfully
	Success bool
}

// Transformation records a single type of change made during conversion.
type Transformation struct {
	// Type is a short identifier for the transformation (e.g., "XREF_UPPERCASE", "CONC_REMOVED")
	Type string

	// Description is a human-readable explanation of what was changed
	Description string

	// Count is the number of instances transformed
	Count int

	// Details contains optional specific information about what was transformed
	Details []string
}

// DataLossItem records a feature that was lost during conversion.
// This typically occurs when downgrading from a newer GEDCOM version
// to an older one that doesn't support certain features.
type DataLossItem struct {
	// Feature is the name of the lost feature (e.g., "SCHMA tag", "EXID external IDs")
	Feature string

	// Reason explains why the feature was lost (e.g., "Not supported in GEDCOM 5.5.1")
	Reason string

	// AffectedRecords lists the XRefs of records that were affected
	AffectedRecords []string
}

// AddTransformation adds a transformation record to the report.
func (r *ConversionReport) AddTransformation(t Transformation) {
	r.Transformations = append(r.Transformations, t)
}

// AddDataLoss adds a data loss record to the report.
func (r *ConversionReport) AddDataLoss(d DataLossItem) {
	r.DataLoss = append(r.DataLoss, d)
}

// HasDataLoss returns true if any data was lost during conversion.
func (r *ConversionReport) HasDataLoss() bool {
	return len(r.DataLoss) > 0
}

// String returns a human-readable summary of the conversion report.
func (r *ConversionReport) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Conversion: %s -> %s\n", r.SourceVersion, r.TargetVersion))
	sb.WriteString(fmt.Sprintf("Success: %t\n", r.Success))

	if len(r.Transformations) > 0 {
		sb.WriteString(fmt.Sprintf("Transformations: %d\n", len(r.Transformations)))
		for _, t := range r.Transformations {
			sb.WriteString(fmt.Sprintf("  - %s: %s (%d instances)\n", t.Type, t.Description, t.Count))
		}
	}

	if len(r.DataLoss) > 0 {
		sb.WriteString(fmt.Sprintf("Data Loss: %d items\n", len(r.DataLoss)))
		for _, d := range r.DataLoss {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", d.Feature, d.Reason))
		}
	}

	if len(r.ValidationIssues) > 0 {
		sb.WriteString(fmt.Sprintf("Validation Issues: %d\n", len(r.ValidationIssues)))
	}

	return sb.String()
}
