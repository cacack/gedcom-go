package gedcomgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// JSON round-trip: encoding/json on a decoded Document must stay a small
// multiple of the file and must lose nothing (#754).
//
// The json struct tags in package gedcom do two things. Document.XRefMap is
// `json:"-"`, because it indexes the records already in Records and would
// write each one a second time. Every other field is omitempty where its zero
// value means absent. The oracle for "means absent" is the round trip below:
// what encoding/json writes must read back into a model reflect.DeepEqual to
// the original, Tags and the rebuilt XRefMap included.
//
// One difference in form is accepted and pinned by TestJSONEmptyReadsBackNil:
// omitempty drops an empty slice or map as it drops a nil one, so an empty one
// reads back as nil. No fixture decodes to one, which is why DeepEqual holds
// for all of them.
//
// Record.Entity is an interface, so encoding/json cannot pick the concrete type
// when reading; unmarshalJSONDocument does it from Record.Type, as a caller
// reading the JSON back into this model has to.

// jsonAlwaysWritten lists the fields that carry no omitempty because their
// zero value is a value rather than an absence: level 0 is a level, Gregorian
// is a calendar and "no modifier" is how an exact date is told from an
// approximate one.
var jsonAlwaysWritten = map[string]bool{
	"Tag.Level":     true,
	"Date.Calendar": true,
	"Date.Modifier": true,
}

// newJSONEntity returns an empty entity of the type a record of typ carries.
func newJSONEntity(typ gedcom.RecordType) interface{} {
	switch typ {
	case gedcom.RecordTypeIndividual:
		return &gedcom.Individual{}
	case gedcom.RecordTypeFamily:
		return &gedcom.Family{}
	case gedcom.RecordTypeSource:
		return &gedcom.Source{}
	case gedcom.RecordTypeRepository:
		return &gedcom.Repository{}
	case gedcom.RecordTypeNote:
		return &gedcom.Note{}
	case gedcom.RecordTypeMedia:
		return &gedcom.MediaObject{}
	case gedcom.RecordTypeSubmitter:
		return &gedcom.Submitter{}
	case gedcom.RecordTypeSubmission:
		return &gedcom.Submission{}
	case gedcom.RecordTypeSharedNote:
		return &gedcom.SharedNote{}
	}
	return nil
}

// unmarshalJSONDocument reads json.Marshal(doc) back into a Document, giving
// each record the typed entity its Type names and rebuilding XRefMap, which
// is not written.
func unmarshalJSONDocument(data []byte) (*gedcom.Document, error) {
	var raw struct {
		Records []struct {
			Entity json.RawMessage
		}
	}
	doc := &gedcom.Document{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for i, rec := range doc.Records {
		rec.Entity = nil
		if len(raw.Records[i].Entity) == 0 {
			continue
		}
		entity := newJSONEntity(rec.Type)
		if entity == nil {
			return nil, fmt.Errorf("record %d (%s): entity on a record type that has none", i, rec.Type)
		}
		if err := json.Unmarshal(raw.Records[i].Entity, entity); err != nil {
			return nil, fmt.Errorf("record %d (%s): %w", i, rec.Type, err)
		}
		rec.Entity = entity
	}
	doc.ReindexXRefs()
	return doc, nil
}

// TestJSONRoundTripPres2020 is the measurement from #754: pres2020.ged is
// 1.1 MB and marshalled to 29.0 MB before the struct tags.
func TestJSONRoundTripPres2020(t *testing.T) {
	const path = "testdata/gedcom-5.5/pres2020.ged"
	const limit = 10 << 20

	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	doc, err := Decode(bytes.NewReader(orig))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	t.Logf("%s: %d bytes of GEDCOM, %d bytes of JSON", path, len(orig), len(data))
	if len(data) >= limit {
		t.Errorf("JSON is %d bytes, want under %d", len(data), limit)
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := top["XRefMap"]; ok {
		t.Error("JSON carries XRefMap, which repeats every record in Records")
	}

	got, err := unmarshalJSONDocument(data)
	if err != nil {
		t.Fatalf("unmarshalJSONDocument: %v", err)
	}
	if !reflect.DeepEqual(doc, got) {
		t.Error("document does not survive a JSON round trip")
	}
}

// TestJSONRoundTripFixtures runs the same round trip over every decodable
// fixture, so the fields pres2020.ged never fills (GEDCOM 7.0 structures, LDS
// ordinances, media) are held to it too.
func TestJSONRoundTripFixtures(t *testing.T) {
	for _, path := range gedcomFixtures(t) {
		t.Run(path, func(t *testing.T) {
			orig, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			doc, err := Decode(bytes.NewReader(orig))
			if err != nil {
				if _, expected := undecodable[path]; expected {
					return
				}
				t.Fatalf("Decode: %v", err)
			}
			data, err := json.Marshal(doc)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got, err := unmarshalJSONDocument(data)
			if err != nil {
				t.Fatalf("unmarshalJSONDocument: %v", err)
			}
			want := doc
			if reflect.DeepEqual(want, got) {
				return
			}
			if !reflect.DeepEqual(want.Header, got.Header) {
				t.Error("Header does not survive a JSON round trip")
			}
			for i, rec := range want.Records {
				if i >= len(got.Records) || !reflect.DeepEqual(rec, got.Records[i]) {
					t.Fatalf("record %d (%s %s) does not survive a JSON round trip", i, rec.XRef, rec.Type)
				}
			}
			t.Error("document does not survive a JSON round trip")
		})
	}
}

// TestJSONRoundTripSampleEntities holds every hand-built entity of
// TestEntityRoundTrip to the same round trip: those samples fill fields no
// fixture does.
func TestJSONRoundTripSampleEntities(t *testing.T) {
	for _, s := range sampleEntities() {
		t.Run(s.name, func(t *testing.T) {
			data, err := json.Marshal(s.entity)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got := newJSONEntity(s.typ)
			if err := json.Unmarshal(data, got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if !reflect.DeepEqual(s.entity, got) {
				t.Errorf("entity does not survive a JSON round trip\nJSON: %s", data)
			}
		})
	}
}

// TestJSONMeaningfulZeroValues pins the zero values that are data. omitempty
// drops a nil pointer but not a pointer to zero, so a citation quality of 0
// ("unreliable evidence") survives; the jsonAlwaysWritten fields are written
// even when zero.
func TestJSONMeaningfulZeroValues(t *testing.T) {
	zero := 0
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"citation quality 0", &gedcom.SourceCitation{Quality: &zero}, `{"Quality":0}`},
		{"citation quality absent", &gedcom.SourceCitation{}, `{}`},
		{"tag at level 0", &gedcom.Tag{Tag: "HEAD"}, `{"Level":0,"Tag":"HEAD"}`},
		{"exact Gregorian date", &gedcom.Date{Original: "1900", Year: 1900}, `{"Original":"1900","Year":1900,"Modifier":0,"Calendar":0}`},
		{"XRefMap is not written", &gedcom.Document{XRefMap: map[string]*gedcom.Record{"@I1@": {}}}, `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("got  %s\nwant %s", data, tt.want)
			}
			got := reflect.New(reflect.TypeOf(tt.value).Elem()).Interface()
			if err := json.Unmarshal(data, got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			want := tt.value
			if _, ok := want.(*gedcom.Document); ok {
				want = &gedcom.Document{}
			}
			if !reflect.DeepEqual(want, got) {
				t.Errorf("value does not survive a JSON round trip: %s", data)
			}
		})
	}
}

// TestJSONEmptyReadsBackNil pins the one way a round trip is not DeepEqual: an
// empty, non-nil slice or map is omitted and reads back as nil. The decoder
// builds such a map for a GEDCOM 7.0 header whose SCHMA has no TAG, and
// NewDocument such a slice for Records.
func TestJSONEmptyReadsBackNil(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SCHMA\n0 @I1@ INDI\n1 NAME A /B/\n0 TRLR\n"
	doc, err := Decode(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if doc.Schema == nil || doc.Schema.TagMappings == nil || len(doc.Schema.TagMappings) != 0 {
		t.Fatalf("Schema = %#v, want an empty non-nil TagMappings", doc.Schema)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := unmarshalJSONDocument(data)
	if err != nil {
		t.Fatalf("unmarshalJSONDocument: %v", err)
	}
	if got.Schema == nil || got.Schema.TagMappings != nil {
		t.Errorf("Schema read back as %#v, want a nil TagMappings", got.Schema)
	}
	got.Schema.TagMappings = map[string]string{}
	if !reflect.DeepEqual(doc, got) {
		t.Error("document differs by more than the empty TagMappings")
	}

	built := gedcom.NewDocument(gedcom.Version70)
	if built.Records == nil {
		t.Fatal("NewDocument: Records is nil, want empty")
	}
	if data, err = json.Marshal(built); err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, err = unmarshalJSONDocument(data); err != nil {
		t.Fatalf("unmarshalJSONDocument: %v", err)
	}
	if got.Records != nil {
		t.Errorf("Records read back as %#v, want nil", got.Records)
	}
}

// TestJSONMigrationGuideShapes pins the v3 JSON printed in
// docs/guides/migration-v3.md.
func TestJSONMigrationGuideShapes(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"SpouseInFamilies", []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}, `[{"FamilyXRef":"@F1@"}]`},
		{"SourceCitationData.Text", []*gedcom.SourceText{{Value: "Birth record shows..."}}, `[{"Value":"Birth record shows..."}]`},
		{
			"RefNumbers and UIDs",
			&gedcom.Individual{RefNumbers: []gedcom.RefNumber{{Value: "1234"}}, UIDs: []string{"abc"}},
			`{"RefNumbers":[{"Value":"1234"}],"UIDs":["abc"]}`,
		},
		{"Header.Date absent", &gedcom.Header{}, `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("got  %s\nwant %s", data, tt.want)
			}
		})
	}
}

// TestJSONTagsCoverDocumentModel walks every struct type reachable from a
// Document and requires each field to carry its json tag, so a field added
// later without one fails here rather than quietly growing the output.
func TestJSONTagsCoverDocumentModel(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(typ reflect.Type)
	walk = func(typ reflect.Type) {
		switch typ.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map:
			walk(typ.Elem())
			return
		case reflect.Struct:
		default:
			return
		}
		if seen[typ] {
			return
		}
		seen[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			name := typ.Name() + "." + f.Name
			want := `,omitempty`
			switch {
			case name == "Document.XRefMap":
				want = `-`
			case jsonAlwaysWritten[name]:
				want = ``
			}
			if got := f.Tag.Get("json"); got != want {
				t.Errorf("%s: json tag %q, want %q", name, got, want)
			}
			walk(f.Type)
		}
	}
	walk(reflect.TypeOf(gedcom.Document{}))
	for _, typ := range []gedcom.RecordType{
		gedcom.RecordTypeIndividual, gedcom.RecordTypeFamily, gedcom.RecordTypeSource,
		gedcom.RecordTypeRepository, gedcom.RecordTypeNote, gedcom.RecordTypeMedia,
		gedcom.RecordTypeSubmitter, gedcom.RecordTypeSubmission, gedcom.RecordTypeSharedNote,
	} {
		walk(reflect.TypeOf(newJSONEntity(typ)))
	}
	if len(seen) < 40 {
		t.Errorf("walked %d struct types, want the whole document model", len(seen))
	}
}
