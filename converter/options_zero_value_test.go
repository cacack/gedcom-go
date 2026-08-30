package converter

import (
	"bytes"
	"testing"

	"github.com/cacack/gedcom-go/v2/encoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// vendorDoc returns a 7.0 document carrying a vendor extension, so a
// conversion has something the options could plausibly affect.
func vendorDoc() *gedcom.Document {
	rec := &gedcom.Record{
		XRef:   "@I1@",
		Type:   gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{XRef: "@I1@"},
		Tags: []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: "John /Doe/"},
			{Level: 1, Tag: "_VENDOR", Value: "custom"},
		},
	}
	return &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{rec},
		XRefMap: map[string]*gedcom.Record{"@I1@": rec},
	}
}

// TestConvertOptionsZeroValueIsHonoured pins that a non-nil *ConvertOptions is
// taken exactly as written. Only nil means "use the defaults".
//
// An earlier version of this change also substituted the defaults for a wholly
// zero struct. That looked friendlier but made "every option off" impossible to
// express: the literal for it is the zero struct, so the request was silently
// overridden and the struct's own doc comment became untrue (issue #491, panel
// review).
func TestConvertOptionsZeroValueIsHonoured(t *testing.T) {
	report := func(t *testing.T, opts *ConvertOptions) *gedcom.ConversionReport {
		t.Helper()
		_, rep, err := ConvertWithOptions(vendorDoc(), gedcom.Version551, opts)
		if err != nil {
			t.Fatalf("ConvertWithOptions() error = %v", err)
		}
		return rep
	}

	t.Run("nil means DefaultOptions", func(t *testing.T) {
		if got, want := len(report(t, nil).Preserved), len(report(t, DefaultOptions()).Preserved); got != want {
			t.Errorf("nil preserved %d entries, DefaultOptions %d; they should agree", got, want)
		}
	})

	t.Run("a zero struct means every option off, and is honoured", func(t *testing.T) {
		zero := report(t, &ConvertOptions{})
		if len(zero.Preserved) != 0 {
			t.Errorf("ReportPreservedTags was forced on: %d Preserved entries", len(zero.Preserved))
		}

		// Distinguishable from the defaults, which is the whole point.
		if def := report(t, DefaultOptions()); len(def.Preserved) == len(zero.Preserved) {
			t.Errorf("a zero struct is indistinguishable from DefaultOptions (%d entries each)",
				len(zero.Preserved))
		}
	})

	t.Run("every option off is constructible", func(t *testing.T) {
		// The configuration the earlier design could not express.
		opts := &ConvertOptions{
			Validate:            false,
			StrictDataLoss:      false,
			ReportPreservedTags: false,
			MapEXIDToVendorTags: false,
		}
		if *opts != (ConvertOptions{}) {
			t.Fatal("fixture is not the zero struct; the test would prove nothing")
		}
		if len(report(t, opts).Preserved) != 0 {
			t.Error("an all-false literal was overridden")
		}
	})
}

// TestConvertOutputBytesIgnoreReportOptions asserts the converter preserves
// everything regardless of either new flag: they gate a report and a vendor-tag
// mapping, never what survives the conversion. Only the EXID mapping may alter
// output, and this document carries no EXID.
func TestConvertOutputBytesIgnoreReportOptions(t *testing.T) {
	var first string
	for i, opts := range []*ConvertOptions{
		{Validate: true, ReportPreservedTags: true, MapEXIDToVendorTags: true},
		{Validate: true, ReportPreservedTags: false, MapEXIDToVendorTags: true},
		{Validate: true, ReportPreservedTags: true, MapEXIDToVendorTags: false},
		{Validate: true, ReportPreservedTags: false, MapEXIDToVendorTags: false},
	} {
		result, _, err := ConvertWithOptions(vendorDoc(), gedcom.Version551, opts)
		if err != nil {
			t.Fatalf("case %d: ConvertWithOptions() error = %v", i, err)
		}
		var buf bytes.Buffer
		if err := encoder.EncodeWithOptions(&buf, result, &encoder.EncodeOptions{}); err != nil {
			t.Fatalf("case %d: encode error = %v", i, err)
		}
		if i == 0 {
			first = buf.String()
			if !bytes.Contains(buf.Bytes(), []byte("_VENDOR")) {
				t.Fatal("fixture lost its vendor tag; the test would prove nothing")
			}
			continue
		}
		if buf.String() != first {
			t.Errorf("case %d changed the output bytes:\n got:\n%s\nwant:\n%s", i, buf.String(), first)
		}
	}
}
