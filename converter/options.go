package converter

import (
	"context"
	"fmt"

	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

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

	// Context allows cancellation and timeout control. The conversion checks
	// it before each conversion step and, while copying the document and
	// validating the result, before each record. Once it is done the
	// conversion stops and returns no document and the context's error
	// wrapped (errors.Is matches context.Canceled or
	// context.DeadlineExceeded); the caller's document is never modified.
	// Zero value: nil (no cancellation).
	Context context.Context
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

// cancelled returns the options' Context error, wrapped, once that context is
// done, and nil otherwise or when no context is set.
func (opts *ConvertOptions) cancelled() error {
	return cancelledErr(ctxcheck.Err(opts.Context))
}

// cancelledErr wraps a context error as a cancelled conversion; nil stays nil.
func cancelledErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("conversion cancelled: %w", err)
}
