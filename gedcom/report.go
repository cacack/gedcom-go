package gedcom

import (
	"fmt"
	"strings"
)

// ConversionNote records a single per-item change during GEDCOM version conversion.
// It provides granular tracking of what changed, including the path to the affected
// element and the before/after values.
type ConversionNote struct {
	// Path identifies the location in the GEDCOM document using a hierarchical format.
	// Examples:
	//   - "Individual @I1@" (record level)
	//   - "Individual @I1@ > Birth > Date" (nested element)
	//   - "Header > CHAR" (header element)
	Path string

	// Original is the value before transformation.
	// May be empty for newly generated elements.
	Original string

	// Result is the value after transformation.
	// May be empty for dropped elements.
	Result string

	// Reason explains why the transformation occurred.
	Reason string
}

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

	// Dropped contains notes for data that couldn't be represented in the target version.
	// These represent actual information loss during conversion.
	Dropped []ConversionNote

	// Normalized contains notes for formatting or structural changes.
	// The semantic meaning is preserved, but the representation changed.
	Normalized []ConversionNote

	// Approximated contains notes for semantic equivalents that aren't exact.
	// The meaning is similar but not identical after conversion.
	Approximated []ConversionNote

	// Preserved contains notes for unknown tags kept through conversion.
	// These elements passed through unchanged despite being non-standard.
	Preserved []ConversionNote
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

// AddDropped adds a note for data that couldn't be represented in the target version.
func (r *ConversionReport) AddDropped(note ConversionNote) {
	r.Dropped = append(r.Dropped, note)
}

// AddNormalized adds a note for a formatting or structural change.
func (r *ConversionReport) AddNormalized(note ConversionNote) {
	r.Normalized = append(r.Normalized, note)
}

// AddApproximated adds a note for a semantic equivalent that isn't exact.
func (r *ConversionReport) AddApproximated(note ConversionNote) {
	r.Approximated = append(r.Approximated, note)
}

// AddPreserved adds a note for an unknown tag kept through conversion.
func (r *ConversionReport) AddPreserved(note ConversionNote) {
	r.Preserved = append(r.Preserved, note)
}

// AllNotes returns all conversion notes across all categories.
// Notes are returned in the order: Dropped, Normalized, Approximated, Preserved.
func (r *ConversionReport) AllNotes() []ConversionNote {
	total := len(r.Dropped) + len(r.Normalized) + len(r.Approximated) + len(r.Preserved)
	if total == 0 {
		return nil
	}

	notes := make([]ConversionNote, 0, total)
	notes = append(notes, r.Dropped...)
	notes = append(notes, r.Normalized...)
	notes = append(notes, r.Approximated...)
	notes = append(notes, r.Preserved...)
	return notes
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

	if len(r.Dropped) > 0 {
		sb.WriteString(fmt.Sprintf("Dropped: %d items\n", len(r.Dropped)))
		for _, n := range r.Dropped {
			sb.WriteString(formatConversionNote(n))
		}
	}

	if len(r.Normalized) > 0 {
		sb.WriteString(fmt.Sprintf("Normalized: %d items\n", len(r.Normalized)))
		for _, n := range r.Normalized {
			sb.WriteString(formatConversionNote(n))
		}
	}

	if len(r.Approximated) > 0 {
		sb.WriteString(fmt.Sprintf("Approximated: %d items\n", len(r.Approximated)))
		for _, n := range r.Approximated {
			sb.WriteString(formatConversionNote(n))
		}
	}

	if len(r.Preserved) > 0 {
		sb.WriteString(fmt.Sprintf("Preserved: %d items\n", len(r.Preserved)))
		for _, n := range r.Preserved {
			sb.WriteString(formatConversionNote(n))
		}
	}

	if len(r.ValidationIssues) > 0 {
		sb.WriteString(fmt.Sprintf("Validation Issues: %d\n", len(r.ValidationIssues)))
	}

	return sb.String()
}

// formatConversionNote formats a single conversion note for display.
func formatConversionNote(n ConversionNote) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  - %s\n", n.Path))
	if n.Original != "" {
		sb.WriteString(fmt.Sprintf("      Original: %s\n", n.Original))
	}
	if n.Result != "" {
		sb.WriteString(fmt.Sprintf("      Result: %s\n", n.Result))
	}
	if n.Reason != "" {
		sb.WriteString(fmt.Sprintf("      Reason: %s\n", n.Reason))
	}
	return sb.String()
}
