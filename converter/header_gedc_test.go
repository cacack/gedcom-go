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
		// which removed the structure (GEDC_FORM_REMOVED); true for 5.x, which
		// requires it, whether kept from the input or added (#465).
		wantForm bool
	}{
		{"GEDC without VERS to 7.0", gedcNoVers, "", gedcom.Version70, false},
		{"GEDC without VERS to 5.5.1", gedcNoVers, "", gedcom.Version551, true},
		{"GEDC without VERS to 5.5", gedcNoVers, gedcom.Version551, gedcom.Version55, true},
		// Same version (#615): the decoder's fallback reads the input as 5.5.
		{"GEDC without VERS to 5.5, same version", gedcNoVers, "", gedcom.Version55, true},
		{"GEDC without VERS to 5.5.1, same version", gedcNoVers, gedcom.Version551, gedcom.Version551, true},
		{"no GEDC to 7.0", noGEDC, "", gedcom.Version70, false},
		{"no GEDC to 5.5.1", noGEDC, "", gedcom.Version551, true},
		{"no GEDC to 5.5", noGEDC, gedcom.Version551, gedcom.Version55, true},
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
			// The header now declares VERS, so the typed twin must match it.
			if got.Header.DeclaredVersion != tt.target {
				t.Errorf("DeclaredVersion = %q, want %q", got.Header.DeclaredVersion, tt.target)
			}

			var buf bytes.Buffer
			if err := encoder.Encode(&buf, got); err != nil {
				t.Fatalf("encode: %v", err)
			}
			out := buf.String()

			if n := strings.Count(out, "1 GEDC\n"); n != 1 {
				t.Errorf("want exactly one GEDC, got %d:\n%s", n, out)
			}
			// VERS is GEDC's first child; FORM, kept or added, follows it.
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

// TestConvertSameVersionKeepsDeclaredVERS pins the other half of issue #615: a
// same-version conversion of a header that already declares GEDC.VERS returns
// the input document itself, unchanged.
func TestConvertSameVersionKeepsDeclaredVERS(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n0 @I1@ INDI\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, _, err := converter.Convert(doc, gedcom.Version55)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got != doc {
		t.Errorf("same-version conversion with VERS declared returned a copy, want the input")
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, got); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if buf.String() != input {
		t.Errorf("output changed:\n got %q\nwant %q", buf.String(), input)
	}
}

// TestConvertSameVersion70AddsVERSAndRenamesXRefs pins that a 7.0 same-version
// conversion needing both repairs -- a VERS-less GEDC and an XRef the 7.0
// grammar rejects -- applies both to one copy and leaves the input alone.
func TestConvertSameVersion70AddsVERSAndRenamesXRefs(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n1 CHAR UTF-8\n0 @i2@ INDI\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	doc.Header.Version = gedcom.Version70
	inputTags := len(doc.Header.Tags)

	got, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got == doc {
		t.Fatal("same-version conversion needing repairs returned the input")
	}
	if len(doc.Header.Tags) != inputTags || doc.GetIndividual("@i2@") == nil {
		t.Error("Convert mutated the input document")
	}
	if got.GetIndividual("@I2@") == nil {
		t.Error("XRef @i2@ was not renamed to @I2@")
	}
	if !slices.ContainsFunc(report.Transformations, func(tr gedcom.Transformation) bool {
		return tr.Type == "XREF_UPPERCASE"
	}) {
		t.Errorf("report lacks the XRef rename: %+v", report.Transformations)
	}
	if got.Header.DeclaredVersion != gedcom.Version70 {
		t.Errorf("DeclaredVersion = %q, want %q", got.Header.DeclaredVersion, gedcom.Version70)
	}

	var buf bytes.Buffer
	if err := encoder.Encode(&buf, got); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if n := strings.Count(out, "2 VERS 7.0\n"); n != 1 || !strings.Contains(out, "1 GEDC\n2 VERS 7.0\n") {
		t.Errorf("want exactly one VERS 7.0 directly under GEDC:\n%s", out)
	}
}
