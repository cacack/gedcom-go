package merge_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// combinedPlaceForm combines doc1 and doc2, encodes the result and decodes it
// again, returning the combined header, the report, the encoded text and the
// default place form a reader of the file gets.
func combinedPlaceForm(t *testing.T, doc1, doc2 *gedcom.Document) (h *gedcom.Header, report *merge.CombineReport, encoded, reread string) {
	t.Helper()
	out, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, out); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	again, err := decoder.Decode(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("decode combined output: %v\n%s", err, buf.String())
	}
	return out.Header, report, buf.String(), again.Header.PlaceForm
}

func hasHeaderConflict(report *merge.CombineReport, field string) bool {
	for _, c := range report.HeaderConflicts {
		if c.Field == field {
			return true
		}
	}
	return false
}

// The encoder writes a hand-built header's PlaceForm as PLAC.FORM (#875), so
// in Combine a typed form with no raw PLAC is that document's PLAC, as a typed
// Date is its DATE: the field and the encoded header must name the same form.
func TestCombineHeaderPlaceFormMixedProvenance(t *testing.T) {
	const form1, form2 = "City, County, State, Country", "Village, Parish, Province, Nation"

	t.Run("hand-built doc1 keeps its form over a decoded doc2", func(t *testing.T) {
		doc1 := handBuiltHeaderDoc(&gedcom.Header{PlaceForm: form1})
		doc2 := decodeHeaderDoc(t, "1 SOUR B\n1 PLAC\n2 FORM "+form2+"\n1 NOTE n\n")

		h, report, encoded, reread := combinedPlaceForm(t, doc1, doc2)
		if h.PlaceForm != form1 || reread != form1 {
			t.Errorf("PlaceForm = %q, file says %q; want doc1's %q in both\n%s", h.PlaceForm, reread, form1, encoded)
		}
		if strings.Contains(encoded, form2) {
			t.Errorf("doc2's form reached the file:\n%s", encoded)
		}
		if !hasHeaderConflict(report, "Tags.PLAC") {
			t.Errorf("dropped doc2 PLAC not reported: %+v", report.HeaderConflicts)
		}
		// Grammar order: PLAC precedes NOTE.
		if plac, note := strings.Index(encoded, "\n1 PLAC\n"), strings.Index(encoded, "\n1 NOTE "); plac < 0 || note < plac {
			t.Errorf("PLAC at %d, NOTE at %d; want PLAC before NOTE:\n%s", plac, note, encoded)
		}
	})

	t.Run("hand-built doc1 form reaches the file when doc2 has no PLAC", func(t *testing.T) {
		doc1 := handBuiltHeaderDoc(&gedcom.Header{PlaceForm: form1})
		doc2 := decodeHeaderDoc(t, "1 SOUR B\n")

		h, _, encoded, reread := combinedPlaceForm(t, doc1, doc2)
		if h.PlaceForm != form1 || reread != form1 {
			t.Errorf("PlaceForm = %q, file says %q; want %q in both\n%s", h.PlaceForm, reread, form1, encoded)
		}
		if !strings.HasSuffix(strings.SplitN(encoded, "\n0 TRLR", 2)[0], "\n1 PLAC\n2 FORM "+form1) {
			t.Errorf("PLAC is not the last header structure:\n%s", encoded)
		}
	})

	t.Run("hand-built doc2 form is used when a decoded doc1 has none", func(t *testing.T) {
		doc1 := decodeHeaderDoc(t, "1 SOUR A\n")
		doc2 := handBuiltHeaderDoc(&gedcom.Header{PlaceForm: form2})

		h, _, encoded, reread := combinedPlaceForm(t, doc1, doc2)
		if h.PlaceForm != form2 || reread != form2 {
			t.Errorf("PlaceForm = %q, file says %q; want doc2's %q in both\n%s", h.PlaceForm, reread, form2, encoded)
		}
	})

	t.Run("hand-built doc2 form is used when a hand-built doc1 has none", func(t *testing.T) {
		doc1 := handBuiltHeaderDoc(&gedcom.Header{})
		doc2 := handBuiltHeaderDoc(&gedcom.Header{PlaceForm: form2})

		h, _, encoded, reread := combinedPlaceForm(t, doc1, doc2)
		if h.PlaceForm != form2 || reread != form2 {
			t.Errorf("PlaceForm = %q, file says %q; want doc2's %q in both\n%s", h.PlaceForm, reread, form2, encoded)
		}
	})

	t.Run("decoded doc1 PLAC wins over a hand-built doc2 form", func(t *testing.T) {
		doc1 := decodeHeaderDoc(t, "1 PLAC\n2 FORM "+form1+"\n")
		doc2 := handBuiltHeaderDoc(&gedcom.Header{PlaceForm: form2})

		h, _, encoded, reread := combinedPlaceForm(t, doc1, doc2)
		if h.PlaceForm != form1 || reread != form1 {
			t.Errorf("PlaceForm = %q, file says %q; want doc1's %q in both\n%s", h.PlaceForm, reread, form1, encoded)
		}
		if strings.Count(encoded, "\n1 PLAC\n") != 1 {
			t.Errorf("want exactly one PLAC:\n%s", encoded)
		}
	})

	t.Run("whitespace-only doc1 form is no form", func(t *testing.T) {
		doc1 := handBuiltHeaderDoc(&gedcom.Header{PlaceForm: "  "})
		doc2 := decodeHeaderDoc(t, "1 PLAC\n2 FORM "+form2+"\n")

		h, report, encoded, reread := combinedPlaceForm(t, doc1, doc2)
		if h.PlaceForm != form2 || reread != form2 {
			t.Errorf("PlaceForm = %q, file says %q; want doc2's %q in both\n%s", h.PlaceForm, reread, form2, encoded)
		}
		if hasHeaderConflict(report, "Tags.PLAC") {
			t.Errorf("unexpected PLAC conflict: %+v", report.HeaderConflicts)
		}
	})

	t.Run("inserted form is escaped for the header version", func(t *testing.T) {
		const form = "City @ County"
		doc1 := handBuiltHeaderDoc(&gedcom.Header{PlaceForm: form})
		doc2 := decodeHeaderDoc(t, "1 SOUR B\n")

		_, _, encoded, reread := combinedPlaceForm(t, doc1, doc2)
		if !strings.Contains(encoded, "\n2 FORM City @@ County\n") || reread != form {
			t.Errorf("FORM not escaped, or read back as %q:\n%s", reread, encoded)
		}
	})
}
