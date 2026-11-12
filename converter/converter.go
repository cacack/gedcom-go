package converter

import (
	"errors"
	"fmt"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

// ErrNilDocument is returned by Convert and ConvertWithOptions when doc is nil.
var ErrNilDocument = errors.New("document is nil")

// ErrInvalidVersion is wrapped by the error Convert and ConvertWithOptions
// return when the target version is not one gedcom.Version.IsValid accepts.
var ErrInvalidVersion = errors.New("invalid target version")

// ErrUnsupportedConversion is wrapped by the error Convert and
// ConvertWithOptions return when no conversion path joins the document's
// version to the target version.
var ErrUnsupportedConversion = errors.New("unsupported conversion")

// ErrDataLoss is wrapped by the error ConvertWithOptions returns when
// ConvertOptions.StrictDataLoss is set and the conversion would lose data.
// The report returned with it lists what would be lost.
var ErrDataLoss = errors.New("conversion would result in data loss")

// Convert converts a GEDCOM document to the target version using default options.
//
// When the document is already at the target version, nothing is converted
// and the document is returned as-is, with two exceptions. A header whose
// GEDC has no VERS gets "2 VERS <target>" added under that GEDC, on a copy,
// as a cross-version conversion would add it. And a 7.0 document
// converted to 7.0 has any XRef the 7.0 grammar rejects (such as "@i 2@")
// rewritten, together with the pointers to it, on a copy, and each rename is
// recorded in the report as for a 5.5.x -> 7.0 upgrade. A dangling pointer
// that 7.0 does not read as a pointer (one with a space) is left unchanged.
// So a same-version call returns the input itself when nothing needs
// repairing and a copy otherwise; callers must not depend on the result being
// (or not being) the same pointer as doc.
func Convert(doc *gedcom.Document, targetVersion gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, error) {
	return ConvertWithOptions(doc, targetVersion, DefaultOptions())
}

// ConvertWithOptions converts a GEDCOM document with custom options.
//
// A document already at the target version is handled as described on
// [Convert], including the 7.0 -> 7.0 XRef normalization; opts does not
// apply on that path (no validation or strict data-loss check runs).
//
// Behavior note: on a 7.0 -> 5.5/5.5.1 downgrade, a FamilySearch ARK EXID on an
// individual is mapped to the vendor tag _FSFTID rather than being dropped (see
// transformEXIDToVendorTags). Such an EXID therefore appears under the report's
// Normalized notes instead of DataLoss. This mapping is skipped when
// opts.MapEXIDToVendorTags is false, and has no inverse on the upgrade path.
//
//nolint:gocyclo // Routing to 6 conversion paths requires this branching structure
func ConvertWithOptions(doc *gedcom.Document, targetVersion gedcom.Version, opts *ConvertOptions) (*gedcom.Document, *gedcom.ConversionReport, error) {
	if doc == nil {
		return nil, nil, ErrNilDocument
	}
	if opts == nil {
		opts = DefaultOptions()
	}
	if !targetVersion.IsValid() {
		return nil, nil, fmt.Errorf("%w: %s", ErrInvalidVersion, targetVersion)
	}
	if err := opts.cancelled(); err != nil {
		return nil, nil, err
	}

	// A nil header carries no version, so it falls through to the default below.
	var sourceVersion gedcom.Version
	if doc.Header != nil {
		sourceVersion = doc.Header.Version
	}
	if sourceVersion == "" {
		sourceVersion = gedcom.Version55 // Default assumption
	}

	// No conversion needed if versions match. The document comes back as-is,
	// except that a nil header is materialised into the empty one it is read as,
	// so the result is never a document a caller can dereference into a panic.
	// The version is deliberately not stamped here: this path does not convert,
	// and a document that already carries an empty header does not get one
	// either -- a nil header has to behave exactly like that empty twin. The
	// materialisation goes on a shallow copy, because the caller's document is
	// an input, not ours to mutate. A GEDC that is present but has no VERS is
	// the other repair: VERS is added under it on a deep copy (issue #615).
	//
	// The one exception is 7.0: XRefs its grammar rejects are normalized as on
	// the 5.5.x -> 7.0 paths (on a deep copy), so converting to 7.0 always
	// yields valid 7.0 identifiers. A 7.0 source always has a non-nil header
	// (a nil one reads as 5.5 above), so the header materialisation never
	// follows that copy; it copies converted, not doc, to keep it if it did.
	if sourceVersion == targetVersion {
		report := &gedcom.ConversionReport{
			SourceVersion: sourceVersion,
			TargetVersion: targetVersion,
			Success:       true,
		}
		converted := doc
		if targetVersion == gedcom.Version70 {
			// The mapping is built on the caller's document (read-only) so a
			// document with nothing to rename is returned as the same pointer,
			// without a deep copy. Clone keeps every record, pointer and the
			// declared version, so the mapping applies to the copy unchanged.
			if mapping, dangling := buildXRefMapping(doc); len(mapping) > 0 {
				copied, err := ctxcheck.Clone(opts.Context, doc)
				if err != nil {
					return nil, nil, cancelledErr(err)
				}
				converted = copied
				applyV7XRefMapping(converted, mapping, dangling, report)
			}
		}
		// A GEDC with no VERS (common in sloppy exports, whose version is
		// then detected by tag fallback) gets VERS as a cross-version
		// conversion would add it (#615); a declared VERS is left alone.
		if gedcLacksVers(converted.Header) {
			if converted == doc {
				copied, err := ctxcheck.Clone(opts.Context, doc)
				if err != nil {
					return nil, nil, cancelledErr(err)
				}
				converted = copied
			}
			updateVersionTag(converted.Header, targetVersion)
			// The header now declares VERS, so its identifier grammar
			// applies, as on the cross-version path.
			converted.Header.DeclaredVersion = targetVersion
		}
		if converted != doc {
			// A copy whose Tags were rewritten: its entities are re-derived
			// from them, as on the cross-version path (#772).
			decoder.SyncEntities(converted)
		}
		if converted.Header == nil {
			shallow := *converted
			shallow.Header = &gedcom.Header{}
			converted = &shallow
		}
		return converted, report, nil
	}

	// Deep copy to avoid mutating original
	converted, err := ctxcheck.Clone(opts.Context, doc)
	if err != nil {
		return nil, nil, cancelledErr(err)
	}

	// Create report
	report := &gedcom.ConversionReport{
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
	}

	// Route to appropriate converter
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
		return nil, nil, fmt.Errorf("%w: %s to %s", ErrUnsupportedConversion, sourceVersion, targetVersion)
	}

	if err != nil {
		report.Success = false
		return nil, report, err
	}

	// Check for data loss in strict mode
	if opts.StrictDataLoss && report.HasDataLoss() {
		report.Success = false
		return nil, report, fmt.Errorf("%w (strict mode enabled)", ErrDataLoss)
	}

	// Update header version. A nil header is treated as an empty one (as the
	// encoder does) so the converted document still declares its target version.
	if converted.Header == nil {
		converted.Header = &gedcom.Header{}
	}
	converted.Header.Version = targetVersion
	// The header now declares the target version (updateVersionTag), so its
	// identifier grammar applies from here on.
	converted.Header.DeclaredVersion = targetVersion

	// The transforms above rewrite each record's Tags, the authoritative
	// result; its typed Entity is re-derived from them by the target version,
	// so the two agree whatever a transform touched (#772). A record with no
	// Tags keeps the Entity the transforms left it.
	decoder.SyncEntities(converted)

	// Validate if requested
	if opts.Validate {
		_ = validateConverted(opts.Context, converted, report)
	}

	// The steps since the last check, validation among them, may have been
	// cut short.
	if err := opts.cancelled(); err != nil {
		report.Success = false
		return nil, report, err
	}

	report.Success = true
	return converted, report, nil
}

// convert55To551 converts GEDCOM 5.5 to 5.5.1.
func convert55To551(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	return opts.runSteps(
		func() { transformHeader(doc, gedcom.Version551, report) },
		func() { synthesizePedigrees(doc, report, gedcom.Version551) },
		func() { opts.reportPreservedTags(doc, report) },
		func() {
			report.AddTransformation(gedcom.Transformation{
				Type:        "VERSION_UPGRADE",
				Description: "Upgraded from GEDCOM 5.5 to 5.5.1 (backward compatible)",
				Count:       1,
			})
		},
	)
}

// convert55To70 converts GEDCOM 5.5 to 7.0.
func convert55To70(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	var variations droppedNameVariations
	return opts.runSteps(
		// First, so no later transform reports a change inside a removed record.
		func() { dropSubmissionRecords(doc, report) },
		// 5.5 does not define FONE/ROMN either, but a nonconforming 5.5 file can
		// carry them, and 7.0 output must not. Removed before the text and XRef
		// transforms for the same reason; reported after XRef normalization.
		func() { variations = removeNameVariations(doc) },
		func() { transformTextForVersion(doc, gedcom.Version70, report) },
		func() { normalizeXRefsForV7(doc, report) },
		// After XRef normalization, so the records it reports (dual-year DataLoss
		// AffectedRecords) are named by their XRefs in the converted document.
		func() { transformDateCalendars(doc, gedcom.Version70, report) },
		func() { upgradeNoteRecordsForV7(doc, report) },
		func() { transformMediaTypes(doc, gedcom.Version70, report) },
		func() { transformHeader(doc, gedcom.Version70, report) },
		func() { transformUIDTags(doc, report, gedcom.Version70) },
		func() { synthesizePedigrees(doc, report, gedcom.Version70) },
		func() { marriedNamesToV7(doc, report) },
		func() { variations.report(doc, report, gedcom.Version70) },
		func() { opts.reportPreservedTags(doc, report) },
		func() {
			report.AddTransformation(gedcom.Transformation{
				Type:        "VERSION_UPGRADE",
				Description: "Upgraded from GEDCOM 5.5 to 7.0",
				Count:       1,
			})
		},
	)
}

// convert551To55 converts GEDCOM 5.5.1 to 5.5.
func convert551To55(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	return opts.runSteps(
		func() { transformHeader(doc, gedcom.Version55, report) },
		func() { record551Tags(doc, report) },
		func() { dropNameVariations(doc, report, gedcom.Version55) },
		func() { opts.reportPreservedTags(doc, report) },
		func() {
			report.AddTransformation(gedcom.Transformation{
				Type:        "VERSION_DOWNGRADE",
				Description: "Downgraded from GEDCOM 5.5.1 to 5.5",
				Count:       1,
			})
		},
	)
}

// convert551To70 converts GEDCOM 5.5.1 to 7.0.
func convert551To70(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	var variations droppedNameVariations
	return opts.runSteps(
		// First, so no later transform reports a change inside a removed record.
		func() { dropSubmissionRecords(doc, report) },
		// Removed before the text and XRef transforms for the same reason, and
		// reported after XRef normalization, so records are named by new XRefs.
		func() { variations = removeNameVariations(doc) },
		func() { transformTextForVersion(doc, gedcom.Version70, report) },
		func() { normalizeXRefsForV7(doc, report) },
		// After XRef normalization, so the records it reports (dual-year DataLoss
		// AffectedRecords) are named by their XRefs in the converted document.
		func() { transformDateCalendars(doc, gedcom.Version70, report) },
		func() { upgradeNoteRecordsForV7(doc, report) },
		func() { transformMediaTypes(doc, gedcom.Version70, report) },
		func() { transformHeader(doc, gedcom.Version70, report) },
		func() { transformUIDTags(doc, report, gedcom.Version70) },
		func() { synthesizePedigrees(doc, report, gedcom.Version70) },
		func() { marriedNamesToV7(doc, report) },
		func() { variations.report(doc, report, gedcom.Version70) },
		func() { opts.reportPreservedTags(doc, report) },
		func() {
			report.AddTransformation(gedcom.Transformation{
				Type:        "VERSION_UPGRADE",
				Description: "Upgraded from GEDCOM 5.5.1 to 7.0",
				Count:       1,
			})
		},
	)
}

// convert70To55 converts GEDCOM 7.0 to 5.5.
func convert70To55(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	return opts.runSteps(
		// Drop negative assertions and @VOID@ pointers first, so no later
		// transform reports a change to text inside a block that is then removed.
		func() { dropNegativeAssertions(doc, report, gedcom.Version55) },
		func() { dropVoidPointers(doc, report, gedcom.Version55) },
		// 7.0 does not define FONE/ROMN, but a nonconforming 7.0 file or a
		// hand-built document can carry them, and 5.5 output must not.
		func() { dropNameVariations(doc, report, gedcom.Version55) },
		func() {
			if opts.MapEXIDToVendorTags {
				transformEXIDToVendorTags(doc, report, gedcom.Version55)
			}
		},
		func() { transformTextForVersion(doc, gedcom.Version55, report) },
		func() { transformDateCalendars(doc, gedcom.Version55, report) },
		func() { transformMediaTypes(doc, gedcom.Version55, report) },
		func() { transformHeader(doc, gedcom.Version55, report) },
		// Before dropV7OnlyTags, which reports whatever 7.0-only tag remains.
		func() { transformUIDTags(doc, report, gedcom.Version55) },
		func() { dropV7OnlyTags(doc, report, gedcom.Version55) },
		func() { downgradeSharedNotes(doc, report, gedcom.Version55) },
		func() { opts.reportPreservedTags(doc, report) },
		func() {
			report.AddTransformation(gedcom.Transformation{
				Type:        "VERSION_DOWNGRADE",
				Description: "Downgraded from GEDCOM 7.0 to 5.5",
				Count:       1,
			})
		},
	)
}

// convert70To551 converts GEDCOM 7.0 to 5.5.1.
func convert70To551(doc *gedcom.Document, report *gedcom.ConversionReport, opts *ConvertOptions) error {
	return opts.runSteps(
		// Drop negative assertions and @VOID@ pointers first, so no later
		// transform reports a change to text inside a block that is then removed.
		func() { dropNegativeAssertions(doc, report, gedcom.Version551) },
		func() { dropVoidPointers(doc, report, gedcom.Version551) },
		func() {
			if opts.MapEXIDToVendorTags {
				transformEXIDToVendorTags(doc, report, gedcom.Version551)
			}
		},
		func() { transformTextForVersion(doc, gedcom.Version551, report) },
		func() { transformDateCalendars(doc, gedcom.Version551, report) },
		func() { transformMediaTypes(doc, gedcom.Version551, report) },
		func() { transformHeader(doc, gedcom.Version551, report) },
		// Before dropV7OnlyTags, which reports whatever 7.0-only tag remains.
		func() { transformUIDTags(doc, report, gedcom.Version551) },
		func() { dropV7OnlyTags(doc, report, gedcom.Version551) },
		func() { downgradeSharedNotes(doc, report, gedcom.Version551) },
		func() { opts.reportPreservedTags(doc, report) },
		func() {
			report.AddTransformation(gedcom.Transformation{
				Type:        "VERSION_DOWNGRADE",
				Description: "Downgraded from GEDCOM 7.0 to 5.5.1",
				Count:       1,
			})
		},
	)
}

// record551Tags notes 5.5.1-specific tags that may not be recognized in 5.5.
func record551Tags(doc *gedcom.Document, report *gedcom.ConversionReport) {
	tags551 := []string{"EMAIL", "FAX", "WWW", "FACT", "MAP", "LATI", "LONG"}
	found := make(map[string]int)
	// Track per-record occurrences for granular notes
	perRecord := make(map[string]map[string]bool) // tag -> xref -> true

	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		for _, tag := range record.Tags {
			if tag == nil {
				continue
			}
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

// countTagsInRecord counts occurrences of specific tags.
func countTagsInRecord(tags []*gedcom.Tag, targetTags []string, found map[string]int) {
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		for _, target := range targetTags {
			if tag.Tag == target {
				found[target]++
			}
		}
	}
}

// getRecordTypeByXRef returns the record type string for a given XRef.
func getRecordTypeByXRef(doc *gedcom.Document, xref string) string {
	if record, ok := doc.XRefMap[xref]; ok && record != nil {
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
		if record == nil {
			continue
		}
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

// runSteps runs each conversion step in order, checking the options' Context
// before each, and returns its error, wrapped, once it is done.
func (opts *ConvertOptions) runSteps(steps ...func()) error {
	for _, step := range steps {
		if err := opts.cancelled(); err != nil {
			return err
		}
		step()
	}
	return nil
}

// reportPreservedTags records the preserved unknown tags in report when the
// options ask for it (ReportPreservedTags).
func (opts *ConvertOptions) reportPreservedTags(doc *gedcom.Document, report *gedcom.ConversionReport) {
	if opts.ReportPreservedTags {
		recordPreservedUnknownTags(doc, report)
	}
}
