package converter_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestConvertHeaderGEDC pins issue #602: a header whose GEDC has no VERS (common
// in sloppy exports) must get VERS added under that GEDC, not a second GEDC
// structure; a header with no GEDC at all gets exactly one. Each output must
// strict-decode as the target version.
func TestConvertHeaderGEDC(t *testing.T) {
	const gedcNoVers = "0 HEAD\n1 GEDC\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n0 @I1@ INDI\n0 TRLR\n"
	const noGEDC = "0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n0 TRLR\n"

	tests := []struct {
		name   string
		input  string
		source gedcom.Version // overrides the decoder's fallback (5.5) when set
		target gedcom.Version
		// wantForm reports whether GEDC.FORM is in the output: false for 7.0,
		// which removed the structure (GEDC_FORM_REMOVED), and false when the
		// input had no FORM to keep.
		wantForm bool
	}{
		{"GEDC without VERS to 7.0", gedcNoVers, "", gedcom.Version70, false},
		{"GEDC without VERS to 5.5.1", gedcNoVers, "", gedcom.Version551, true},
		{"GEDC without VERS to 5.5", gedcNoVers, gedcom.Version551, gedcom.Version55, true},
		{"no GEDC to 7.0", noGEDC, "", gedcom.Version70, false},
		{"no GEDC to 5.5.1", noGEDC, "", gedcom.Version551, false},
		{"no GEDC to 5.5", noGEDC, gedcom.Version551, gedcom.Version55, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if tt.source != "" {
				doc.Header.Version = tt.source
			}
			// Snapshot each tag's pointer and contents: an in-place
			// slices.Insert on a shared backing array would shift elements
			// without changing the input's length.
			type tagSnap struct {
				ptr   *gedcom.Tag
				level int
				tag   string
				value string
			}
			snap := func(tags []*gedcom.Tag) []tagSnap {
				out := make([]tagSnap, len(tags))
				for i, tg := range tags {
					out[i] = tagSnap{tg, tg.Level, tg.Tag, tg.Value}
				}
				return out
			}
			inputTags := snap(doc.Header.Tags)

			got, _, err := converter.Convert(doc, tt.target)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}

			if after := snap(doc.Header.Tags); !slices.Equal(after, inputTags) {
				t.Errorf("Convert mutated the input header:\n got %+v\nwant %+v", after, inputTags)
			}

			var buf bytes.Buffer
			if err := encoder.Encode(&buf, got); err != nil {
				t.Fatalf("encode: %v", err)
			}
			out := buf.String()

			if n := strings.Count(out, "1 GEDC\n"); n != 1 {
				t.Errorf("want exactly one GEDC, got %d:\n%s", n, out)
			}
			// VERS is GEDC's first child; a kept FORM follows it.
			wantVers := "1 GEDC\n2 VERS " + tt.target.String() + "\n"
			if tt.wantForm {
				wantVers += "2 FORM LINEAGE-LINKED\n"
			}
			if !strings.Contains(out, wantVers) {
				t.Errorf("want %q directly under GEDC:\n%s", wantVers, out)
			}
			if hasForm := strings.Contains(out, "2 FORM LINEAGE-LINKED\n"); hasForm != tt.wantForm {
				t.Errorf("GEDC.FORM present = %v, want %v:\n%s", hasForm, tt.wantForm, out)
			}

			opts := decoder.DefaultOptions()
			opts.StrictMode = true
			redecoded, err := decoder.DecodeWithOptions(strings.NewReader(out), opts)
			if err != nil {
				t.Fatalf("strict decode of converted output: %v\n%s", err, out)
			}
			if redecoded.Header.Version != tt.target {
				t.Errorf("redecoded version = %q, want %q", redecoded.Header.Version, tt.target)
			}
		})
	}
}
