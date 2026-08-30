package converter

// ConvertOptions configures the conversion behavior.
//
// The zero value is treated as "unset": passing a nil *ConvertOptions, or a
// pointer to a wholly zero struct, is identical to passing DefaultOptions().
//
// A *partially* populated literal is taken at face value, so each field it
// leaves out takes its own zero value rather than the DefaultOptions value.
// To change one setting, start from DefaultOptions:
//
//	opts := converter.DefaultOptions()
//	opts.StrictDataLoss = true
//
// Every field below documents what its own zero value does, so a literal never
// behaves differently from what the field's own comment says.
type ConvertOptions struct {
	// Validate runs validation on the converted document.
	// Zero value: false (no validation). DefaultOptions sets it true.
	Validate bool

	// StrictDataLoss fails the conversion if any data would be lost.
	// Zero value: false (data loss is reported, not fatal).
	StrictDataLoss bool

	// ReportPreservedTags records each preserved vendor/unknown tag in the
	// conversion report's Preserved notes. It is report-only: the converter
	// preserves those tags either way, and no output byte depends on it.
	// Zero value: false (tags are still preserved, just not itemised).
	// DefaultOptions sets it true.
	ReportPreservedTags bool

	// MapEXIDToVendorTags maps a FamilySearch ARK EXID to the _FSFTID vendor
	// tag when downgrading from 7.0, so the identifier survives as a vendor
	// extension instead of being recorded as data loss. Has no inverse on the
	// upgrade path.
	// Zero value: false (the EXID is reported under DataLoss instead).
	// DefaultOptions sets it true.
	MapEXIDToVendorTags bool
}

// DefaultOptions returns the default conversion options.
func DefaultOptions() *ConvertOptions {
	return &ConvertOptions{
		Validate:            true,
		StrictDataLoss:      false,
		ReportPreservedTags: true,
		MapEXIDToVendorTags: true,
	}
}
