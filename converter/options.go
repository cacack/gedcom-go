package converter

// ConvertOptions configures the conversion behavior.
//
// A nil *ConvertOptions means DefaultOptions(). Any non-nil pointer is taken
// exactly as written, including a wholly zero &ConvertOptions{} -- so every
// field a literal omits takes that field's own zero value, not the
// DefaultOptions value. To change one setting, start from DefaultOptions:
//
//	opts := converter.DefaultOptions()
//	opts.StrictDataLoss = true
//
// Each field below documents its own zero value, and nothing overrides it.
// An earlier design silently substituted the defaults for a wholly zero
// struct; that made "every option off" impossible to express, since the
// literal for it was indistinguishable from "unset".
//
// Nothing here can cause data loss: the converter preserves every tag
// regardless of these settings.
type ConvertOptions struct {
	// Validate runs validation on the converted document.
	// Zero value: false (no validation). DefaultOptions sets it true.
	// Note that the converter discards the validation error either way; this
	// only controls whether the work is done.
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
