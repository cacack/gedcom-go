// Package testing provides round-trip test helpers for GEDCOM documents.
//
// This package enables users to verify that encode/decode cycles preserve
// their genealogical data, addressing the common fear of import/export corruption.
//
// # Quick Start
//
// Use AssertRoundTrip in your tests to verify that a GEDCOM file survives
// an encode/decode cycle:
//
//	func TestMyGEDCOM(t *testing.T) {
//	    data, _ := os.ReadFile("family.ged")
//	    gedcomtesting.AssertRoundTrip(t, data)
//	}
//
// For programmatic use (e.g., validation tools), use CheckRoundTrip which
// returns a detailed report:
//
//	report, err := gedcomtesting.CheckRoundTrip(file)
//	if !report.Equal {
//	    for _, diff := range report.Differences {
//	        log.Printf("Difference at %s: %q -> %q", diff.Path, diff.Before, diff.After)
//	    }
//	}
//
// # Fidelity Contract
//
// Round-trip testing compares documents at the semantic level using Record.Tags
// as the source of truth. The following describes what is preserved and what may change:
//
// Preserved (must be identical after round-trip):
//   - All tag names and values
//   - Hierarchical structure (nesting levels relative to parent)
//   - Tag order within each record
//   - Unknown and vendor-specific tags (e.g., _CUSTOM, _TREE)
//   - Cross-reference identifiers (XRef)
//   - Record order in the document
//
// May change (not compared):
//   - Line numbers (expected to differ due to header reconstruction)
//   - Byte-level formatting (line endings, whitespace normalization)
//   - CONC/CONT reorganization (may be split differently)
//   - Header structure (reconstructed from Header fields)
//
// # How Comparison Works
//
// The comparison algorithm:
//  1. Compares Header fields (Version, Encoding, SourceSystem, Language)
//  2. Compares record count and XRef presence
//  3. For each record, compares Record.Tags recursively:
//     - Matches tags by position (same index)
//     - Compares Level, Tag, Value, XRef fields
//     - Skips LineNumber field (expected to change)
//     - Reports path-based differences for easy debugging
//
// # Design Rationale
//
// Record.Tags is the source of truth for lossless preservation, not the Entity
// field. The encoder uses Tags when present, falling back to Entity conversion
// only when Tags is empty. This ensures that unknown/vendor tags and exact
// structure are preserved even when the library doesn't understand them.
package testing
