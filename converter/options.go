package converter

// ConvertOptions configures the conversion behavior.
type ConvertOptions struct {
	// Validate runs validation on the converted document.
	// Default: true
	Validate bool

	// StrictDataLoss fails the conversion if any data would be lost.
	// Default: false
	StrictDataLoss bool

	// PreserveUnknownTags keeps vendor extensions and unknown tags.
	// Default: true
	//
	// Deprecated: the name is wrong -- the converter never dropped anything
	// either way. v3 splits it into ReportPreservedTags (itemise preserved tags
	// in the conversion report) and MapEXIDToVendorTags (map a FamilySearch ARK
	// EXID to _FSFTID on a 7.0 downgrade). Neither exists in v2, so this is a
	// change to make at upgrade time; see docs/guides/migration-v3.md.
	PreserveUnknownTags bool
}

// DefaultOptions returns the default conversion options.
func DefaultOptions() *ConvertOptions {
	return &ConvertOptions{
		Validate:            true,
		StrictDataLoss:      false,
		PreserveUnknownTags: true,
	}
}
