package gedcomgo

// ADR 0007's nil policy is library-wide, not per package. This file is the
// acceptance test for that claim: one document carrying every nil shape at once
// -- a nil *Record, a nil *Tag, a typed-nil Record.Entity, a nil element inside
// an entity's slice field, and a nil Header -- goes through the whole facade,
// and every result has to equal the result for the same document with the nils
// removed. The per-package suites (gedcom, encoder, validator, converter) pin
// the individual guards; this one proves they compose.

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// facadeDoc builds the all-shapes document, or its nil-free twin. Both come
// from one function so the two cannot drift apart: withNils is the only
// difference between them.
//
// A nil Header's twin is an empty header rather than a missing one, because
// that is what "removed" means for a Header -- the encoder writes it as an
// empty header and the converter materialises one, so an empty header is the
// document the nil is meant to be indistinguishable from.
func facadeDoc(withNils bool) *gedcom.Document {
	// tags is the raw-tag path: a record decoded from a file replays its Tags,
	// so the nils sit between real tags rather than replacing them.
	tags := func() []*gedcom.Tag {
		clean := []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: "John /Doe/"},
			{Level: 1, Tag: "SEX", Value: "M"},
			{Level: 1, Tag: "NOTE", Value: "first "},
			{Level: 2, Tag: "CONC", Value: "second"},
			// A dangling pointer, so the rules that resolve XRefs have
			// something to report on both documents.
			{Level: 1, Tag: "FAMS", Value: "@F9@"},
		}
		if !withNils {
			return clean
		}
		return []*gedcom.Tag{nil, clean[0], clean[1], clean[2], nil, clean[3], clean[4], nil}
	}

	// names and events are the entity path: a nil element in a
	// slice-of-pointer field on an entity built in memory.
	names := func() []*gedcom.PersonalName {
		clean := []*gedcom.PersonalName{{Full: "Jane /Doe/", Given: "Jane", Surname: "Doe"}}
		if !withNils {
			return clean
		}
		return []*gedcom.PersonalName{nil, clean[0]}
	}
	events := func() []*gedcom.Event {
		clean := []*gedcom.Event{{Type: gedcom.EventBirth, Date: "1 JAN 1900"}}
		if !withNils {
			return clean
		}
		return []*gedcom.Event{clean[0], nil}
	}

	// A typed nil in Record.Entity is a non-nil interface holding a nil
	// pointer; removing it leaves the record with no entity at all, since the
	// record itself (XRef, Type) is legitimate.
	entity := func(typedNil interface{}) interface{} {
		if withNils {
			return typedNil
		}
		return nil
	}

	records := []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: tags()},
		{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
			XRef: "@I2@", Sex: "F", Names: names(), Events: events(),
		}},
		{XRef: "@I3@", Type: gedcom.RecordTypeIndividual, Entity: entity((*gedcom.Individual)(nil))},
		{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{
			// A dangling wife pointer, so the reference rules that read
			// Families() report on both documents.
			XRef: "@F1@", Husband: "@I2@", Wife: "@I9@",
		}},
		{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: &gedcom.Source{XRef: "@S1@", Title: "Census"}},
		{XRef: "@R1@", Type: gedcom.RecordTypeRepository, Entity: &gedcom.Repository{XRef: "@R1@", Name: "Archive"}},
		{XRef: "@U1@", Type: gedcom.RecordTypeSubmitter, Entity: &gedcom.Submitter{XRef: "@U1@", Name: "Chris"}},
		{XRef: "@N1@", Type: gedcom.RecordTypeNote, Entity: &gedcom.Note{XRef: "@N1@", Text: "A note"}},
		{XRef: "@O1@", Type: gedcom.RecordTypeMedia, Entity: entity((*gedcom.MediaObject)(nil))},
		{XRef: "@O2@", Type: gedcom.RecordTypeMedia, Entity: &gedcom.MediaObject{
			XRef: "@O2@", Files: []*gedcom.MediaFile{{FileRef: "photo.jpg", Form: "image/jpeg"}},
		}},
		{XRef: "@X1@", Type: gedcom.RecordTypeSharedNote, Entity: &gedcom.SharedNote{XRef: "@X1@", Text: "Shared"}},
	}

	if withNils {
		// Leading and trailing, so a guard that bailed out of the whole loop
		// instead of skipping one element fails rather than passes.
		records = append([]*gedcom.Record{nil}, append(records, nil)...)
	}

	doc := &gedcom.Document{Records: records}
	if !withNils {
		doc.Header = &gedcom.Header{}
	}

	doc.XRefMap = make(map[string]*gedcom.Record, len(records))
	for _, record := range records {
		if record != nil {
			doc.XRefMap[record.XRef] = record
		}
	}

	return doc
}

// TestFacade_Encode drives the nils through [Encode]. Depends on the encoder's
// skips for a nil record, tag, typed-nil entity, entity slice element and
// header.
func TestFacade_Encode(t *testing.T) {
	encode := func(doc *Document) string {
		t.Helper()

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		return buf.String()
	}

	want := encode(facadeDoc(false))
	if want == "" {
		t.Fatal("test setup: the nil-free document encoded to nothing")
	}

	if got := encode(facadeDoc(true)); got != want {
		t.Errorf("Encode() with nils =\n%s\nwant:\n%s", got, want)
	}
}

// TestFacade_Validate drives the nils through [Validate] and [ValidateAll].
// Depends on validator's record, tag and header skips, and on Record.Get* not
// handing a typed nil to the accessor-consuming rules.
func TestFacade_Validate(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		want := Validate(facadeDoc(false))
		if len(want) == 0 {
			t.Fatal("test setup: the nil-free document produced no errors, so the comparison is vacuous")
		}

		if got := Validate(facadeDoc(true)); !reflect.DeepEqual(got, want) {
			t.Errorf("Validate() with nils = %v, want %v", got, want)
		}
	})

	t.Run("ValidateAll", func(t *testing.T) {
		want := ValidateAll(facadeDoc(false))
		if len(want) == 0 {
			t.Fatal("test setup: the nil-free document produced no issues, so the comparison is vacuous")
		}

		if got := ValidateAll(facadeDoc(true)); !reflect.DeepEqual(got, want) {
			t.Errorf("ValidateAll() with nils = %+v, want %+v", got, want)
		}
	})
}

// TestFacade_Convert drives the nils through [Convert] on every target version.
// Depends on converter's record, tag and header handling -- including the
// CONC/CONT lookahead, which the fixture's NOTE/CONC pair with a nil between the
// continuations exercises.
//
// The two converted documents are compared by encoding them: the converter
// preserves a nil *Tag at its original index and passes an entity through
// untouched, so the document that held nils still holds them afterwards.
// Written out, those nils contribute nothing, which is what "the same result as
// the nil-free document" means for a caller.
func TestFacade_Convert(t *testing.T) {
	for _, target := range []Version{Version55, Version551, Version70} {
		t.Run(target.String(), func(t *testing.T) {
			wantDoc, wantReport, err := Convert(facadeDoc(false), target)
			if err != nil {
				t.Fatalf("Convert() without nils error = %v", err)
			}

			gotDoc, gotReport, err := Convert(facadeDoc(true), target)
			if err != nil {
				t.Fatalf("Convert() with nils error = %v", err)
			}

			if got, want := mustEncode(t, gotDoc), mustEncode(t, wantDoc); got != want {
				t.Errorf("Convert() with nils produced a different document\n got =\n%s\nwant =\n%s", got, want)
			}
			if !reflect.DeepEqual(gotReport, wantReport) {
				t.Errorf("Convert() with nils produced a different report\n got = %+v\nwant = %+v",
					gotReport, wantReport)
			}
		})
	}
}

// TestFacade_CollectionAccessors covers all eight collection accessors. Depends
// on Document's nil-record skip and on Record.Get* reporting (nil, false) for a
// typed-nil entity -- without the latter, Individuals() and MediaObjects() would
// each hand back a nil pointer the caller cannot use, which xrefs reports as
// "<nil>".
func TestFacade_CollectionAccessors(t *testing.T) {
	accessors := []struct {
		name string
		call func(*Document) interface{}
	}{
		{"Individuals", func(d *Document) interface{} { return d.Individuals() }},
		{"Families", func(d *Document) interface{} { return d.Families() }},
		{"Sources", func(d *Document) interface{} { return d.Sources() }},
		{"Repositories", func(d *Document) interface{} { return d.Repositories() }},
		{"Submitters", func(d *Document) interface{} { return d.Submitters() }},
		{"Notes", func(d *Document) interface{} { return d.Notes() }},
		{"MediaObjects", func(d *Document) interface{} { return d.MediaObjects() }},
		{"SharedNotes", func(d *Document) interface{} { return d.SharedNotes() }},
	}

	withNils, withoutNils := facadeDoc(true), facadeDoc(false)

	for _, accessor := range accessors {
		t.Run(accessor.name, func(t *testing.T) {
			want := xrefs(accessor.call(withoutNils))
			if len(want) == 0 {
				t.Fatalf("test setup: %s() on the nil-free document is empty, so the comparison is vacuous",
					accessor.name)
			}

			if got := xrefs(accessor.call(withNils)); !reflect.DeepEqual(got, want) {
				t.Errorf("%s() with nils = %v, want %v", accessor.name, got, want)
			}
		})
	}
}

// mustEncode writes a document out, failing the test on error.
func mustEncode(t *testing.T, doc *Document) string {
	t.Helper()

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	return buf.String()
}

// xrefs reads the XRef of every entity a collection accessor returned. The
// entities themselves cannot be compared: an accessor hands back the caller's
// own entity, nils inside it included, so a nil in Individual.Names survives
// into the result by design. Which entities appear, and in what order, is what
// the accessor decides. A nil entry renders as "<nil>", which no nil-free
// document can produce.
func xrefs(collection interface{}) []string {
	value := reflect.ValueOf(collection)

	out := make([]string, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		element := value.Index(i)
		if element.IsNil() {
			out = append(out, "<nil>")
			continue
		}
		out = append(out, element.Elem().FieldByName("XRef").String())
	}

	return out
}
