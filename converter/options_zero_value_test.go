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

// TestZeroConvertOptionsMatchesDefaults pins the zero value of ConvertOptions.
// Before issue #491 only a *nil* pointer picked up DefaultOptions, so a bare
// &ConvertOptions{} silently ran with validation off and no EXID mapping,
// contradicting every field's documented default. All three call shapes must
// now agree.
func TestZeroConvertOptionsMatchesDefaults(t *testing.T) {
	shapes := map[string]*ConvertOptions{
		"nil":            nil,
		"zero struct":    {},
		"DefaultOptions": DefaultOptions(),
	}

	type outcome struct {
		bytes      string
		preserved  int
		dataLoss   int
		normalized int
	}
	got := make(map[string]outcome, len(shapes))

	for name, opts := range shapes {
		result, report, err := ConvertWithOptions(vendorDoc(), gedcom.Version551, opts)
		if err != nil {
			t.Fatalf("%s: ConvertWithOptions() error = %v", name, err)
		}
		var buf bytes.Buffer
		if err := encoder.EncodeWithOptions(&buf, result, &encoder.EncodeOptions{}); err != nil {
			t.Fatalf("%s: encode error = %v", name, err)
		}
		got[name] = outcome{
			bytes:      buf.String(),
			preserved:  len(report.Preserved),
			dataLoss:   len(report.DataLoss),
			normalized: len(report.Normalized),
		}
	}

	want := got["DefaultOptions"]
	for _, name := range []string{"nil", "zero struct"} {
		if got[name] != want {
			t.Errorf("%s differs from DefaultOptions:\n got  %+v\n want %+v", name, got[name], want)
		}
	}
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
