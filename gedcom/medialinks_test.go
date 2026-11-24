package gedcom

import (
	"errors"
	"reflect"
	"testing"
)

// recordEntityTypes lists a zero value of every entity type Record.Entity can
// hold, with the record type it belongs to. It is kept by hand: add a new
// entity type here, or TestMediaLinksCompleteness never sees it.
var recordEntityTypes = []struct {
	recordType RecordType
	entity     interface{}
}{
	{RecordTypeIndividual, &Individual{}},
	{RecordTypeFamily, &Family{}},
	{RecordTypeSource, &Source{}},
	{RecordTypeRepository, &Repository{}},
	{RecordTypeNote, &Note{}},
	{RecordTypeMedia, &MediaObject{}},
	{RecordTypeSubmitter, &Submitter{}},
	{RecordTypeSubmission, &Submission{}},
	{RecordTypeSharedNote, &SharedNote{}},
}

// TestMediaLinksReadsEveryCarrier pins that one call returns the record-level
// media links of INDI, FAM and SOUR records, with no type switch in the caller.
func TestMediaLinksReadsEveryCarrier(t *testing.T) {
	link := &MediaLink{MediaXRef: "@O1@"}
	nested := &MediaLink{MediaXRef: "@O2@"}
	tests := []struct {
		name   string
		entity interface{}
		want   []*MediaLink
	}{
		{"INDI with links", &Individual{Media: []*MediaLink{link}, Events: []*Event{{Media: []*MediaLink{nested}}}}, []*MediaLink{link}},
		{"INDI without links", &Individual{}, nil},
		{"FAM with links", &Family{Media: []*MediaLink{link}}, []*MediaLink{link}},
		{"FAM without links", &Family{}, nil},
		{"SOUR with links", &Source{Media: []*MediaLink{link}}, []*MediaLink{link}},
		{"SOUR without links", &Source{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := (&Record{Entity: tt.entity}).MediaLinks()
			if !ok {
				t.Fatal("MediaLinks() ok = false, want true")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MediaLinks() = %v, want %v", got, tt.want)
			}
			if len(tt.want) > 0 && got[0] != link {
				t.Error("MediaLinks() returned a copy, want the entity's own links")
			}
		})
	}
}

// TestMediaLinksTellsCannotCarryFromNone pins that ok separates a record that
// cannot carry media links from one that carries none.
func TestMediaLinksTellsCannotCarryFromNone(t *testing.T) {
	var nilRecord *Record
	tests := []struct {
		name   string
		record *Record
	}{
		{"NOTE", &Record{Type: RecordTypeNote, Entity: &Note{}}},
		{"REPO", &Record{Type: RecordTypeRepository, Entity: &Repository{}}},
		{"OBJE", &Record{Type: RecordTypeMedia, Entity: &MediaObject{}}},
		{"SUBM", &Record{Type: RecordTypeSubmitter, Entity: &Submitter{}}},
		{"nil record", nilRecord},
		{"nil entity", &Record{Type: RecordTypeIndividual}},
		{"typed nil individual", &Record{Entity: (*Individual)(nil)}},
		{"typed nil family", &Record{Entity: (*Family)(nil)}},
		{"typed nil source", &Record{Entity: (*Source)(nil)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links, ok := tt.record.MediaLinks()
			if ok || links != nil {
				t.Errorf("MediaLinks() = %v, %v, want nil, false", links, ok)
			}
			err := tt.record.AddMediaLink(&MediaLink{MediaXRef: "@O1@"})
			if !errors.Is(err, ErrNoMediaLinks) {
				t.Errorf("AddMediaLink() error = %v, want ErrNoMediaLinks", err)
			}
		})
	}

	if links, ok := (&Record{Entity: &Individual{}}).MediaLinks(); !ok || links != nil {
		t.Errorf("empty INDI: MediaLinks() = %v, %v, want nil, true", links, ok)
	}
}

func TestAddMediaLink(t *testing.T) {
	existing := &MediaLink{MediaXRef: "@O1@"}
	added := &MediaLink{MediaXRef: "@O2@"}
	for _, entity := range []interface{}{
		&Individual{Media: []*MediaLink{existing}},
		&Family{Media: []*MediaLink{existing}},
		&Source{Media: []*MediaLink{existing}},
	} {
		record := &Record{Entity: entity}
		if err := record.AddMediaLink(added); err != nil {
			t.Fatalf("%T: AddMediaLink() error = %v", entity, err)
		}
		// Not deduplicated: the same link again is a second link.
		if err := record.AddMediaLink(added); err != nil {
			t.Fatalf("%T: AddMediaLink() again error = %v", entity, err)
		}
		got, _ := record.MediaLinks()
		want := []*MediaLink{existing, added, added}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%T: links = %v, want %v", entity, got, want)
		}
		field := reflect.ValueOf(entity).Elem().FieldByName("Media").Interface()
		if !reflect.DeepEqual(field, want) {
			t.Errorf("%T: Media field = %v, want %v", entity, field, want)
		}
	}
}

func TestAddMediaLinkNilLink(t *testing.T) {
	ind := &Individual{}
	if err := (&Record{Entity: ind}).AddMediaLink(nil); !errors.Is(err, ErrNilMediaLink) {
		t.Errorf("AddMediaLink(nil) error = %v, want ErrNilMediaLink", err)
	}
	if len(ind.Media) != 0 {
		t.Errorf("AddMediaLink(nil) appended: %v", ind.Media)
	}
}

// TestMediaLinksCompleteness fails when an entity type gains a record-level
// Media []*MediaLink field without joining the switch in mediaLinkSlot, or
// joins it without having the field.
func TestMediaLinksCompleteness(t *testing.T) {
	mediaType := reflect.TypeOf([]*MediaLink(nil))
	carriers := 0
	for _, tc := range recordEntityTypes {
		field, has := reflect.TypeOf(tc.entity).Elem().FieldByName("Media")
		want := has && field.Type == mediaType
		if want {
			carriers++
		}
		record := &Record{Type: tc.recordType, Entity: tc.entity}
		if _, ok := record.MediaLinks(); ok != want {
			t.Errorf("%s (%T): MediaLinks() ok = %v, want %v (has Media []*MediaLink: %v)",
				tc.recordType, tc.entity, ok, want, want)
		}
		err := record.AddMediaLink(&MediaLink{MediaXRef: "@O1@"})
		if (err == nil) != want {
			t.Errorf("%s (%T): AddMediaLink() error = %v, want success: %v", tc.recordType, tc.entity, err, want)
		}
	}
	if carriers == 0 {
		t.Error("no entity type has a Media []*MediaLink field; the reflection check is not looking at the right field")
	}
}
