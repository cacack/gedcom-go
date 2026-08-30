package gedcomgo

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// Entity round-trip: build a document programmatically, encode it, decode it,
// and require the typed entity to come back intact.
//
// This covers the path that neither other harness reaches. writeRecord prefers
// Record.Tags and only falls back to entityToTags when Tags is empty, so a
// decoded document never exercises the entity writers: measured, 46 of 20,724
// records across the whole corpus have empty Tags. TestByteRoundTrip therefore
// says nothing about them, and it cannot -- wiping Names, Events and Sex from a
// decoded document still produces byte-identical output.
//
// That path is what programmatic consumers use, and it is where #421, #404,
// #330, #334 and #326 all live. #438 lived here too and is fixed; what pins it
// now is TestEntityNilElementDoesNotPanic below.
//
// Note what is deliberately NOT asserted here: that the typed model carries
// everything the raw tags do. ADR 0003 gives raw tags the losslessness role and
// the typed entity the role of "typed access to known structures", with no
// commitment about which standard tags are modelled -- RIN and AFN, for two,
// are reachable only through Tags. Asserting the typed model against the raw
// one would therefore report design decisions as defects.
//
// The oracle here is narrower and sound: whatever you put in must come back.

// entityKnownBad maps "Type.Field" to the issue responsible. Self-cleaning: an
// entry that starts passing fails the test asking to be removed.
var entityKnownBad = map[string]string{
	// #442: the same defect #439 fixed on Note, one entity over — the decoder
	// assigns the tag value and never folds in its CONT/CONC lines, so a
	// multi-line value reads back as its first line alone. Title, Author and
	// Publication are worse than Text on the way out too: Text goes through
	// textToTags, they are written raw, so an embedded newline forges a line.
	"Source.Text":        "#442 decoder drops CONT/CONC from Source.Text",
	"Source.Title":       "#442 decoder drops CONT/CONC; encoder writes Title raw",
	"Source.Author":      "#442 decoder drops CONT/CONC; encoder writes Author raw",
	"Source.Publication": "#442 decoder drops CONT/CONC; encoder writes Publication raw",
}

// sampleEntities are populated by hand rather than by reflection. Reflection
// fills fields with values that are not valid GEDCOM -- a Sex of "v12", a date
// that cannot parse -- so a field failing to survive would say more about the
// input than about the code. Hand-built values keep every failure meaningful.
func sampleEntities() []struct {
	name   string
	xref   string
	typ    gedcom.RecordType
	entity interface{}
} {
	return []struct {
		name   string
		xref   string
		typ    gedcom.RecordType
		entity interface{}
	}{
		{"Individual", "@I1@", gedcom.RecordTypeIndividual, &gedcom.Individual{
			XRef: "@I1@",
			Sex:  "M",
			Names: []*gedcom.PersonalName{
				{Full: "John /Smith/", Given: "John", Surname: "Smith"},
			},
			Events: []*gedcom.Event{
				{Type: "BIRT", Date: "5 AUG 1901", Place: "Boston, Massachusetts"},
			},
			Attributes: []*gedcom.Attribute{
				{Type: "OCCU", Value: "Carpenter"},
			},
		}},
		{"Family", "@F1@", gedcom.RecordTypeFamily, &gedcom.Family{
			XRef:     "@F1@",
			Husband:  "@I1@",
			Wife:     "@I2@",
			Children: []string{"@I3@"},
			Events: []*gedcom.Event{
				{Type: "MARR", Date: "1 JAN 1925", Place: "Boston, Massachusetts"},
			},
		}},
		// Every free-text field here is multi-line on purpose. The single-line
		// values this sample used to carry are what hid #442 from the harness,
		// and the same sentence was true of Author and Publication.
		{"Source", "@S1@", gedcom.RecordTypeSource, &gedcom.Source{
			XRef:        "@S1@",
			Title:       "Vital Records of Boston\nVolume II",
			Author:      "City Clerk\nBoston, Massachusetts",
			Publication: "Boston, 1930\nReprinted 1961",
			Text:        "First line of the source text.\nSecond line of the source text.",
		}},
		{"Repository", "@R1@", gedcom.RecordTypeRepository, &gedcom.Repository{
			XRef: "@R1@",
			Name: "Boston Public Library",
		}},
		{"Submitter", "@U1@", gedcom.RecordTypeSubmitter, &gedcom.Submitter{
			XRef: "@U1@",
			Name: "Jane Researcher",
		}},
		// Multi-line text on a hand-built record is the shape #334 and #330
		// were about; kept here so a regression is caught rather than refiled.
		{"Note", "@N1@", gedcom.RecordTypeNote, &gedcom.Note{
			XRef: "@N1@",
			Text: "First line of the note.\nSecond line of the note.",
		}},
		{"SharedNote", "@SN1@", gedcom.RecordTypeSharedNote, &gedcom.SharedNote{
			XRef: "@SN1@",
			Text: "First line of the shared note.\nSecond line of the shared note.",
		}},
		{"MediaObject", "@O1@", gedcom.RecordTypeMedia, &gedcom.MediaObject{
			XRef: "@O1@",
			Files: []*gedcom.MediaFile{
				{FileRef: "portrait.jpg", Form: "image/jpeg", Title: "Portrait"},
			},
		}},
	}
}

// encodeDecode builds a one-record document around entity, encodes it, and
// decodes the result.
func encodeDecode(t *testing.T, xref string, typ gedcom.RecordType, entity interface{}) *gedcom.Document {
	t.Helper()

	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{{XRef: xref, Type: typ, Entity: entity}},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	back, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Decode of our own output: %v\n%s", err, buf.String())
	}
	return back
}

// TestEntityRoundTrip asserts that a hand-built entity survives encode/decode.
func TestEntityRoundTrip(t *testing.T) {
	seen := map[string]bool{}

	for _, tc := range sampleEntities() {
		t.Run(tc.name, func(t *testing.T) {
			back := encodeDecode(t, tc.xref, tc.typ, tc.entity)

			rec := back.XRefMap[tc.xref]
			if rec == nil {
				t.Fatalf("record %s missing after round-trip", tc.xref)
			}
			if rec.Entity == nil {
				t.Fatalf("record %s came back with no typed entity", tc.xref)
			}

			lost := compareEntity(tc.name, tc.entity, rec.Entity)
			sort.Strings(lost)
			for _, path := range lost {
				seen[path] = true
				if _, known := entityKnownBad[path]; !known {
					t.Errorf("field does not survive the entity path: %s", path)
				}
			}
		})
	}

	var fixed []string
	for path, reason := range entityKnownBad {
		if !seen[path] {
			fixed = append(fixed, fmt.Sprintf("%s (was: %s)", path, reason))
		}
	}
	if len(fixed) > 0 {
		sort.Strings(fixed)
		t.Errorf("these fields now survive; remove them from entityKnownBad:\n  %s",
			strings.Join(fixed, "\n  "))
	}
}

// compareEntity reports the paths of set fields that did not come back. Only
// fields that were populated on the way in are checked: the decoder legitimately
// fills in more than was set (parsed dates, derived name parts), and asserting
// equality in that direction would fail on correct behaviour.
func compareEntity(typeName string, want, got interface{}) []string {
	var lost []string
	w := reflect.ValueOf(want).Elem()
	g := reflect.ValueOf(got).Elem()
	if w.Type() != g.Type() {
		return []string{fmt.Sprintf("%s: came back as %s", typeName, g.Type())}
	}

	for i := 0; i < w.NumField(); i++ {
		sf := w.Type().Field(i)
		if sf.PkgPath != "" {
			continue
		}
		wf, gf := w.Field(i), g.Field(i)
		if wf.IsZero() {
			continue // not set on the way in; nothing to lose
		}
		path := typeName + "." + sf.Name

		switch wf.Kind() {
		case reflect.Slice:
			if gf.Len() < wf.Len() {
				lost = append(lost, path)
			}
		case reflect.Pointer, reflect.Interface:
			if gf.IsNil() {
				lost = append(lost, path)
			}
		default:
			if !reflect.DeepEqual(wf.Interface(), gf.Interface()) {
				lost = append(lost, path)
			}
		}
	}
	return lost
}

// TestEntityNilElementDoesNotPanic pins #438: a nil inside an entity's slice
// field must not panic the encoder. ADR 0007 makes the no-panic rule
// unconditional, so this is a contract test, not a robustness nicety.
//
// These four fields are the ones this harness found; the exhaustive matrix --
// every slice-of-pointer field on every entity type, asserting the output also
// matches an encode of the same document without the nil -- lives in
// encoder/nil_safety_test.go. Both are worth keeping: this one guards the entity
// path from the outside, where the harness that found the defect sits.
func TestEntityNilElementDoesNotPanic(t *testing.T) {
	cases := []struct {
		path   string
		entity func() interface{}
	}{
		{"Individual.Names", func() interface{} {
			return &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{nil}}
		}},
		{"Individual.Events", func() interface{} {
			return &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{nil}}
		}},
		{"Individual.Attributes", func() interface{} {
			return &gedcom.Individual{XRef: "@I1@", Attributes: []*gedcom.Attribute{nil}}
		}},
		{"Individual.Associations", func() interface{} {
			return &gedcom.Individual{XRef: "@I1@", Associations: []*gedcom.Association{nil}}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Encode panicked on a nil element in %s: %v (ADR 0007: the library never panics)", tc.path, r)
				}
			}()

			doc := &gedcom.Document{
				Header:  &gedcom.Header{Version: gedcom.Version551},
				Records: []*gedcom.Record{{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: tc.entity()}},
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Errorf("Encode() error = %v", err)
			}
		})
	}
}
