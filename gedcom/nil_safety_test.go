package gedcom

import (
	"reflect"
	"testing"
)

// Record and Document accessors must never panic on a nil (ADR 0007). A decoded
// document never holds one -- the decoder fills Records and XRefMap together and
// only ever stores a populated entity -- so every case here builds the document
// in memory, which is the path a programmatic consumer uses.

// nilEntities pairs each entity type with a typed nil pointer of that type. A
// Record holding one is the shape that used to make Get* return (typedNil, true).
var nilEntities = []struct {
	name       string
	recordType RecordType
	entity     interface{}
}{
	{"Individual", RecordTypeIndividual, (*Individual)(nil)},
	{"Family", RecordTypeFamily, (*Family)(nil)},
	{"Source", RecordTypeSource, (*Source)(nil)},
	{"Submitter", RecordTypeSubmitter, (*Submitter)(nil)},
	{"Repository", RecordTypeRepository, (*Repository)(nil)},
	{"Note", RecordTypeNote, (*Note)(nil)},
	{"MediaObject", RecordTypeMedia, (*MediaObject)(nil)},
	{"SharedNote", RecordTypeSharedNote, (*SharedNote)(nil)},
}

// getters calls every Record.Get* by name, reporting the ok it returned and
// whether the returned pointer was nil.
var getters = []struct {
	name string
	call func(*Record) (isNil bool, ok bool)
}{
	{"GetIndividual", func(r *Record) (bool, bool) { v, ok := r.GetIndividual(); return v == nil, ok }},
	{"GetFamily", func(r *Record) (bool, bool) { v, ok := r.GetFamily(); return v == nil, ok }},
	{"GetSource", func(r *Record) (bool, bool) { v, ok := r.GetSource(); return v == nil, ok }},
	{"GetSubmitter", func(r *Record) (bool, bool) { v, ok := r.GetSubmitter(); return v == nil, ok }},
	{"GetRepository", func(r *Record) (bool, bool) { v, ok := r.GetRepository(); return v == nil, ok }},
	{"GetNote", func(r *Record) (bool, bool) { v, ok := r.GetNote(); return v == nil, ok }},
	{"GetMediaObject", func(r *Record) (bool, bool) { v, ok := r.GetMediaObject(); return v == nil, ok }},
	{"GetSharedNote", func(r *Record) (bool, bool) { v, ok := r.GetSharedNote(); return v == nil, ok }},
}

// TestRecord_NilReceiver covers every Is* and Get* on a nil *Record. None may
// panic, and each must report the record is not of the asked-for type.
func TestRecord_NilReceiver(t *testing.T) {
	var r *Record

	predicates := []struct {
		name string
		call func(*Record) bool
	}{
		{"IsIndividual", (*Record).IsIndividual},
		{"IsFamily", (*Record).IsFamily},
		{"IsSource", (*Record).IsSource},
		{"IsSharedNote", (*Record).IsSharedNote},
	}

	for _, p := range predicates {
		t.Run(p.name, func(t *testing.T) {
			if got := p.call(r); got {
				t.Errorf("(nil *Record).%s() = true, want false", p.name)
			}
		})
	}

	for _, g := range getters {
		t.Run(g.name, func(t *testing.T) {
			isNil, ok := g.call(r)
			if ok {
				t.Errorf("(nil *Record).%s() ok = true, want false", g.name)
			}
			if !isNil {
				t.Errorf("(nil *Record).%s() returned non-nil, want nil", g.name)
			}
		})
	}
}

// TestRecord_TypedNilEntity pins the boolean to the truth: an Entity holding a
// typed nil pointer is not a usable entity, so Get* must report (nil, false)
// rather than handing the caller a pointer that panics on first field access.
func TestRecord_TypedNilEntity(t *testing.T) {
	for _, e := range nilEntities {
		t.Run(e.name, func(t *testing.T) {
			r := &Record{XRef: "@X1@", Type: e.recordType, Entity: e.entity}

			for _, g := range getters {
				isNil, ok := g.call(r)
				if ok {
					t.Errorf("%s() ok = true on typed-nil %s entity, want false", g.name, e.name)
				}
				if !isNil {
					t.Errorf("%s() returned non-nil on typed-nil %s entity, want nil", g.name, e.name)
				}
			}
		})
	}
}

// realRecords returns one populated record of every type, fresh each call so a
// case can put the same records in both the with-nil and without-nil document.
func realRecords() []*Record {
	return []*Record{
		{XRef: "@I1@", Type: RecordTypeIndividual, Entity: &Individual{XRef: "@I1@"}},
		{XRef: "@F1@", Type: RecordTypeFamily, Entity: &Family{XRef: "@F1@"}},
		{XRef: "@S1@", Type: RecordTypeSource, Entity: &Source{XRef: "@S1@"}},
		{XRef: "@U1@", Type: RecordTypeSubmitter, Entity: &Submitter{XRef: "@U1@"}},
		{XRef: "@R1@", Type: RecordTypeRepository, Entity: &Repository{XRef: "@R1@"}},
		{XRef: "@N1@", Type: RecordTypeNote, Entity: &Note{XRef: "@N1@"}},
		{XRef: "@O1@", Type: RecordTypeMedia, Entity: &MediaObject{XRef: "@O1@"}},
		{XRef: "@SN1@", Type: RecordTypeSharedNote, Entity: &SharedNote{XRef: "@SN1@"}},
	}
}

// collections calls every Document collection accessor by name, returning the
// result as an interface so cases can compare two documents' output directly.
var collections = []struct {
	name string
	call func(*Document) interface{}
}{
	{"Individuals", func(d *Document) interface{} { return d.Individuals() }},
	{"Families", func(d *Document) interface{} { return d.Families() }},
	{"Sources", func(d *Document) interface{} { return d.Sources() }},
	{"Submitters", func(d *Document) interface{} { return d.Submitters() }},
	{"Repositories", func(d *Document) interface{} { return d.Repositories() }},
	{"Notes", func(d *Document) interface{} { return d.Notes() }},
	{"MediaObjects", func(d *Document) interface{} { return d.MediaObjects() }},
	{"SharedNotes", func(d *Document) interface{} { return d.SharedNotes() }},
}

// TestDocument_Collections_SkipNil asserts each accessor returns exactly what it
// would for the same document without the nils -- not an empty slice, which a
// guard that bailed out of the whole loop would also produce.
func TestDocument_Collections_SkipNil(t *testing.T) {
	tests := []struct {
		name    string
		withNil []*Record
	}{
		{
			name:    "nil *Record",
			withNil: append([]*Record{nil}, append(realRecords(), nil)...),
		},
		{
			name: "typed-nil Entity",
			withNil: append([]*Record{
				{XRef: "@I9@", Type: RecordTypeIndividual, Entity: (*Individual)(nil)},
				{XRef: "@F9@", Type: RecordTypeFamily, Entity: (*Family)(nil)},
			}, realRecords()...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withNil := &Document{Records: tt.withNil}
			withoutNil := &Document{Records: realRecords()}

			for _, c := range collections {
				got := c.call(withNil)
				want := c.call(withoutNil)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s() = %v, want %v", c.name, got, want)
				}
				if n := reflect.ValueOf(got).Len(); n != 1 {
					t.Errorf("%s() returned %d entities, want 1 -- the surviving record was dropped", c.name, n)
				}
			}
		})
	}
}

// TestDocument_XRefMapNilRecord covers a nil *Record reached through XRefMap
// rather than Records: the typed lookups must report not-found, not panic.
func TestDocument_XRefMapNilRecord(t *testing.T) {
	doc := &Document{XRefMap: map[string]*Record{"@X1@": nil}}

	if got := doc.GetRecord("@X1@"); got != nil {
		t.Errorf("GetRecord(@X1@) = %v, want nil", got)
	}

	lookups := []struct {
		name string
		call func(*Document, string) bool
	}{
		{"GetIndividual", func(d *Document, x string) bool { return d.GetIndividual(x) == nil }},
		{"GetFamily", func(d *Document, x string) bool { return d.GetFamily(x) == nil }},
		{"GetSource", func(d *Document, x string) bool { return d.GetSource(x) == nil }},
		{"GetSubmitter", func(d *Document, x string) bool { return d.GetSubmitter(x) == nil }},
		{"GetRepository", func(d *Document, x string) bool { return d.GetRepository(x) == nil }},
		{"GetNote", func(d *Document, x string) bool { return d.GetNote(x) == nil }},
		{"GetMediaObject", func(d *Document, x string) bool { return d.GetMediaObject(x) == nil }},
		{"GetSharedNote", func(d *Document, x string) bool { return d.GetSharedNote(x) == nil }},
	}

	for _, l := range lookups {
		t.Run(l.name, func(t *testing.T) {
			if !l.call(doc, "@X1@") {
				t.Errorf("%s(@X1@) returned non-nil for a nil *Record, want nil", l.name)
			}
		})
	}
}

// TestDocument_NilReceiver covers a nil *Document: every accessor returns nil,
// matching the d == nil guards already used across the package.
func TestDocument_NilReceiver(t *testing.T) {
	var d *Document

	if got := d.GetRecord("@I1@"); got != nil {
		t.Errorf("(nil *Document).GetRecord() = %v, want nil", got)
	}

	for _, c := range collections {
		t.Run(c.name, func(t *testing.T) {
			if got := reflect.ValueOf(c.call(d)); !got.IsNil() {
				t.Errorf("(nil *Document).%s() = %v, want nil", c.name, got)
			}
		})
	}
}

// TestIndividual_EventHelpersSkipNil covers a nil element in Individual.Events,
// which BirthEvent and DeathEvent walk directly. Reached from the facade through
// the validator's date-logic rule, which calls both on every individual.
func TestIndividual_EventHelpersSkipNil(t *testing.T) {
	birth := &Event{Type: EventBirth, Date: "1 JAN 1900"}
	death := &Event{Type: EventDeath, Date: "1 JAN 1970"}
	ind := &Individual{XRef: "@I1@", Events: []*Event{nil, birth, nil, death, nil}}

	if got := ind.BirthEvent(); got != birth {
		t.Errorf("BirthEvent() = %+v, want the birth event", got)
	}
	if got := ind.DeathEvent(); got != death {
		t.Errorf("DeathEvent() = %+v, want the death event", got)
	}

	none := &Individual{XRef: "@I2@", Events: []*Event{nil}}
	if got := none.BirthEvent(); got != nil {
		t.Errorf("BirthEvent() = %+v, want nil", got)
	}
	if got := none.DeathEvent(); got != nil {
		t.Errorf("DeathEvent() = %+v, want nil", got)
	}
}
