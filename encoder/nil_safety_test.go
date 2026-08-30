package encoder

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// The encoder must never panic on a nil (ADR 0007). A decoded document never
// holds one -- writeRecord replays Record.Tags, which decoding always fills --
// so every case here builds the document in memory and leaves Tags empty, which
// is the path a programmatic consumer uses.

// entityDoc wraps a single entity in a document on the entity path.
func entityDoc(recordType gedcom.RecordType, entity interface{}) *gedcom.Document {
	return &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{
			{XRef: "@X1@", Type: recordType, Entity: entity},
		},
	}
}

// mustEncode encodes with the batch encoder, failing the test on error.
func mustEncode(t *testing.T, doc *gedcom.Document) string {
	t.Helper()

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	return buf.String()
}

// mustEncodeStreaming encodes with the streaming encoder, failing the test on error.
func mustEncodeStreaming(t *testing.T, doc *gedcom.Document) string {
	t.Helper()

	var buf bytes.Buffer
	if err := EncodeStreaming(&buf, doc); err != nil {
		t.Fatalf("EncodeStreaming() error = %v", err)
	}

	return buf.String()
}

// Fresh valid elements. Each returns a new value so a case can put the same
// element in both the with-nil and the without-nil document without sharing it.
func aName() *gedcom.PersonalName {
	return &gedcom.PersonalName{Full: "John /Doe/", Given: "John", Surname: "Doe"}
}

func aTransliteration() *gedcom.Transliteration {
	return &gedcom.Transliteration{Value: "Jon /Do/", Language: "en-Latn"}
}

func anEvent() *gedcom.Event {
	return &gedcom.Event{Type: gedcom.EventBirth, Date: "1 JAN 1900"}
}

func anAttribute() *gedcom.Attribute {
	return &gedcom.Attribute{Type: "OCCU", Value: "Farmer"}
}

func anOrdinance() *gedcom.LDSOrdinance {
	return &gedcom.LDSOrdinance{Type: gedcom.LDSBaptism, Date: "1 JAN 1900"}
}

func anAssociation() *gedcom.Association {
	return &gedcom.Association{IndividualXRef: "@I2@", Role: "GODP"}
}

func aCitation() *gedcom.SourceCitation {
	return &gedcom.SourceCitation{SourceXRef: "@S1@", Page: "Page 42"}
}

func aMediaLink() *gedcom.MediaLink {
	return &gedcom.MediaLink{MediaXRef: "@O1@"}
}

func aMediaFile() *gedcom.MediaFile {
	return &gedcom.MediaFile{FileRef: "photo.jpg", Form: "image/jpeg"}
}

func aMediaTranslation() *gedcom.MediaTranslation {
	return &gedcom.MediaTranslation{FileRef: "photo.png", Form: "image/png"}
}

func aSharedNoteTranslation() *gedcom.SharedNoteTranslation {
	return &gedcom.SharedNoteTranslation{Value: "Traduccion", Language: "es"}
}

func anExternalID() *gedcom.ExternalID {
	return &gedcom.ExternalID{Value: "12345", Type: "http://example.com/ids"}
}

// TestEncode_NilSliceElement covers every slice-of-pointer field the encoder
// writes, on every entity type that has one. A nil element must be skipped: the
// output has to match the same document built without the nil, byte for byte,
// and still contain the surviving element -- otherwise a writer that dropped
// the whole slice would pass.
func TestEncode_NilSliceElement(t *testing.T) {
	tests := []struct {
		name       string
		recordType gedcom.RecordType
		withNil    interface{}
		withoutNil interface{}
		wantLine   string
	}{
		{
			name:       "Individual.Names",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{nil, aName()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{aName()}},
			wantLine:   "1 NAME John /Doe/",
		},
		{
			name:       "Individual.Events",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{nil, anEvent()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{anEvent()}},
			wantLine:   "1 BIRT",
		},
		{
			name:       "Individual.Attributes",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", Attributes: []*gedcom.Attribute{nil, anAttribute()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Attributes: []*gedcom.Attribute{anAttribute()}},
			wantLine:   "1 OCCU Farmer",
		},
		{
			name:       "Individual.LDSOrdinances",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", LDSOrdinances: []*gedcom.LDSOrdinance{nil, anOrdinance()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", LDSOrdinances: []*gedcom.LDSOrdinance{anOrdinance()}},
			wantLine:   "1 BAPL",
		},
		{
			name:       "Individual.Associations",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", Associations: []*gedcom.Association{nil, anAssociation()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Associations: []*gedcom.Association{anAssociation()}},
			wantLine:   "1 ASSO @I2@",
		},
		{
			name:       "Individual.SourceCitations",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", SourceCitations: []*gedcom.SourceCitation{nil, aCitation()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", SourceCitations: []*gedcom.SourceCitation{aCitation()}},
			wantLine:   "1 SOUR @S1@",
		},
		{
			name:       "Individual.Media",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", Media: []*gedcom.MediaLink{nil, aMediaLink()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Media: []*gedcom.MediaLink{aMediaLink()}},
			wantLine:   "1 OBJE @O1@",
		},
		{
			name:       "Individual.ExternalIDs",
			recordType: gedcom.RecordTypeIndividual,
			withNil:    &gedcom.Individual{XRef: "@I1@", ExternalIDs: []*gedcom.ExternalID{nil, anExternalID()}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", ExternalIDs: []*gedcom.ExternalID{anExternalID()}},
			wantLine:   "1 EXID 12345",
		},
		{
			name:       "PersonalName.Transliterations",
			recordType: gedcom.RecordTypeIndividual,
			withNil: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{
				{Full: "John /Doe/", Transliterations: []*gedcom.Transliteration{nil, aTransliteration()}},
			}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{
				{Full: "John /Doe/", Transliterations: []*gedcom.Transliteration{aTransliteration()}},
			}},
			wantLine: "2 TRAN Jon /Do/",
		},
		{
			name:       "Event.SourceCitations",
			recordType: gedcom.RecordTypeIndividual,
			withNil: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{
				{Type: gedcom.EventBirth, SourceCitations: []*gedcom.SourceCitation{nil, aCitation()}},
			}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{
				{Type: gedcom.EventBirth, SourceCitations: []*gedcom.SourceCitation{aCitation()}},
			}},
			wantLine: "2 SOUR @S1@",
		},
		{
			name:       "Event.Media",
			recordType: gedcom.RecordTypeIndividual,
			withNil: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{
				{Type: gedcom.EventBirth, Media: []*gedcom.MediaLink{nil, aMediaLink()}},
			}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{
				{Type: gedcom.EventBirth, Media: []*gedcom.MediaLink{aMediaLink()}},
			}},
			wantLine: "2 OBJE @O1@",
		},
		{
			name:       "Attribute.SourceCitations",
			recordType: gedcom.RecordTypeIndividual,
			withNil: &gedcom.Individual{XRef: "@I1@", Attributes: []*gedcom.Attribute{
				{Type: "OCCU", Value: "Farmer", SourceCitations: []*gedcom.SourceCitation{nil, aCitation()}},
			}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Attributes: []*gedcom.Attribute{
				{Type: "OCCU", Value: "Farmer", SourceCitations: []*gedcom.SourceCitation{aCitation()}},
			}},
			wantLine: "2 SOUR @S1@",
		},
		{
			name:       "Association.SourceCitations",
			recordType: gedcom.RecordTypeIndividual,
			withNil: &gedcom.Individual{XRef: "@I1@", Associations: []*gedcom.Association{
				{IndividualXRef: "@I2@", SourceCitations: []*gedcom.SourceCitation{nil, aCitation()}},
			}},
			withoutNil: &gedcom.Individual{XRef: "@I1@", Associations: []*gedcom.Association{
				{IndividualXRef: "@I2@", SourceCitations: []*gedcom.SourceCitation{aCitation()}},
			}},
			wantLine: "2 SOUR @S1@",
		},
		{
			name:       "Family.Events",
			recordType: gedcom.RecordTypeFamily,
			withNil:    &gedcom.Family{XRef: "@F1@", Events: []*gedcom.Event{nil, anEvent()}},
			withoutNil: &gedcom.Family{XRef: "@F1@", Events: []*gedcom.Event{anEvent()}},
			wantLine:   "1 BIRT",
		},
		{
			name:       "Family.LDSOrdinances",
			recordType: gedcom.RecordTypeFamily,
			withNil:    &gedcom.Family{XRef: "@F1@", LDSOrdinances: []*gedcom.LDSOrdinance{nil, anOrdinance()}},
			withoutNil: &gedcom.Family{XRef: "@F1@", LDSOrdinances: []*gedcom.LDSOrdinance{anOrdinance()}},
			wantLine:   "1 BAPL",
		},
		{
			name:       "Family.SourceCitations",
			recordType: gedcom.RecordTypeFamily,
			withNil:    &gedcom.Family{XRef: "@F1@", SourceCitations: []*gedcom.SourceCitation{nil, aCitation()}},
			withoutNil: &gedcom.Family{XRef: "@F1@", SourceCitations: []*gedcom.SourceCitation{aCitation()}},
			wantLine:   "1 SOUR @S1@",
		},
		{
			name:       "Family.Media",
			recordType: gedcom.RecordTypeFamily,
			withNil:    &gedcom.Family{XRef: "@F1@", Media: []*gedcom.MediaLink{nil, aMediaLink()}},
			withoutNil: &gedcom.Family{XRef: "@F1@", Media: []*gedcom.MediaLink{aMediaLink()}},
			wantLine:   "1 OBJE @O1@",
		},
		{
			name:       "Family.ExternalIDs",
			recordType: gedcom.RecordTypeFamily,
			withNil:    &gedcom.Family{XRef: "@F1@", ExternalIDs: []*gedcom.ExternalID{nil, anExternalID()}},
			withoutNil: &gedcom.Family{XRef: "@F1@", ExternalIDs: []*gedcom.ExternalID{anExternalID()}},
			wantLine:   "1 EXID 12345",
		},
		{
			name:       "Source.Media",
			recordType: gedcom.RecordTypeSource,
			withNil:    &gedcom.Source{XRef: "@S1@", Title: "Census", Media: []*gedcom.MediaLink{nil, aMediaLink()}},
			withoutNil: &gedcom.Source{XRef: "@S1@", Title: "Census", Media: []*gedcom.MediaLink{aMediaLink()}},
			wantLine:   "1 OBJE @O1@",
		},
		{
			name:       "Source.ExternalIDs",
			recordType: gedcom.RecordTypeSource,
			withNil:    &gedcom.Source{XRef: "@S1@", Title: "Census", ExternalIDs: []*gedcom.ExternalID{nil, anExternalID()}},
			withoutNil: &gedcom.Source{XRef: "@S1@", Title: "Census", ExternalIDs: []*gedcom.ExternalID{anExternalID()}},
			wantLine:   "1 EXID 12345",
		},
		{
			name:       "Repository.ExternalIDs",
			recordType: gedcom.RecordTypeRepository,
			withNil:    &gedcom.Repository{XRef: "@R1@", Name: "Archive", ExternalIDs: []*gedcom.ExternalID{nil, anExternalID()}},
			withoutNil: &gedcom.Repository{XRef: "@R1@", Name: "Archive", ExternalIDs: []*gedcom.ExternalID{anExternalID()}},
			wantLine:   "1 EXID 12345",
		},
		{
			name:       "Submitter.ExternalIDs",
			recordType: gedcom.RecordTypeSubmitter,
			withNil:    &gedcom.Submitter{XRef: "@U1@", Name: "Chris", ExternalIDs: []*gedcom.ExternalID{nil, anExternalID()}},
			withoutNil: &gedcom.Submitter{XRef: "@U1@", Name: "Chris", ExternalIDs: []*gedcom.ExternalID{anExternalID()}},
			wantLine:   "1 EXID 12345",
		},
		{
			name:       "MediaObject.Files",
			recordType: gedcom.RecordTypeMedia,
			withNil:    &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{nil, aMediaFile()}},
			withoutNil: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{aMediaFile()}},
			wantLine:   "1 FILE photo.jpg",
		},
		{
			name:       "MediaObject.SourceCitations",
			recordType: gedcom.RecordTypeMedia,
			withNil: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{aMediaFile()},
				SourceCitations: []*gedcom.SourceCitation{nil, aCitation()}},
			withoutNil: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{aMediaFile()},
				SourceCitations: []*gedcom.SourceCitation{aCitation()}},
			wantLine: "1 SOUR @S1@",
		},
		{
			name:       "MediaObject.ExternalIDs",
			recordType: gedcom.RecordTypeMedia,
			withNil: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{aMediaFile()},
				ExternalIDs: []*gedcom.ExternalID{nil, anExternalID()}},
			withoutNil: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{aMediaFile()},
				ExternalIDs: []*gedcom.ExternalID{anExternalID()}},
			wantLine: "1 EXID 12345",
		},
		{
			name:       "MediaFile.Translations",
			recordType: gedcom.RecordTypeMedia,
			withNil: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{
				{FileRef: "photo.jpg", Form: "image/jpeg", Translations: []*gedcom.MediaTranslation{nil, aMediaTranslation()}},
			}},
			withoutNil: &gedcom.MediaObject{XRef: "@O1@", Files: []*gedcom.MediaFile{
				{FileRef: "photo.jpg", Form: "image/jpeg", Translations: []*gedcom.MediaTranslation{aMediaTranslation()}},
			}},
			wantLine: "2 TRAN photo.png",
		},
		{
			name:       "SharedNote.Translations",
			recordType: gedcom.RecordTypeSharedNote,
			withNil: &gedcom.SharedNote{XRef: "@N1@", Text: "A note",
				Translations: []*gedcom.SharedNoteTranslation{nil, aSharedNoteTranslation()}},
			withoutNil: &gedcom.SharedNote{XRef: "@N1@", Text: "A note",
				Translations: []*gedcom.SharedNoteTranslation{aSharedNoteTranslation()}},
			wantLine: "1 TRAN Traduccion",
		},
		{
			name:       "SharedNote.SourceCitations",
			recordType: gedcom.RecordTypeSharedNote,
			withNil: &gedcom.SharedNote{XRef: "@N1@", Text: "A note",
				SourceCitations: []*gedcom.SourceCitation{nil, aCitation()}},
			withoutNil: &gedcom.SharedNote{XRef: "@N1@", Text: "A note",
				SourceCitations: []*gedcom.SourceCitation{aCitation()}},
			wantLine: "1 SOUR @S1@",
		},
		{
			name:       "SharedNote.ExternalIDs",
			recordType: gedcom.RecordTypeSharedNote,
			withNil: &gedcom.SharedNote{XRef: "@N1@", Text: "A note",
				ExternalIDs: []*gedcom.ExternalID{nil, anExternalID()}},
			withoutNil: &gedcom.SharedNote{XRef: "@N1@", Text: "A note",
				ExternalIDs: []*gedcom.ExternalID{anExternalID()}},
			wantLine: "1 EXID 12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := mustEncode(t, entityDoc(tt.recordType, tt.withoutNil))

			if !strings.Contains(want, tt.wantLine) {
				t.Fatalf("test setup: encoding without the nil does not contain %q:\n%s", tt.wantLine, want)
			}

			if got := mustEncode(t, entityDoc(tt.recordType, tt.withNil)); got != want {
				t.Errorf("Encode() with a nil element =\n%s\nwant:\n%s", got, want)
			}

			if got := mustEncodeStreaming(t, entityDoc(tt.recordType, tt.withNil)); got != want {
				t.Errorf("EncodeStreaming() with a nil element =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestEncode_NilRecordElement covers a nil element in Document.Records, which
// is the same shape one level up from the entity fields.
func TestEncode_NilRecordElement(t *testing.T) {
	record := func() *gedcom.Record {
		return &gedcom.Record{
			XRef:   "@I1@",
			Type:   gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{aName()}},
		}
	}

	want := mustEncode(t, &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{record()},
	})

	withNil := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{nil, record(), nil},
	}

	if got := mustEncode(t, withNil); got != want {
		t.Errorf("Encode() with nil records =\n%s\nwant:\n%s", got, want)
	}

	if got := mustEncodeStreaming(t, withNil); got != want {
		t.Errorf("EncodeStreaming() with nil records =\n%s\nwant:\n%s", got, want)
	}
}

// TestEncode_NilTagElement covers a nil element in Record.Tags, on both sides of
// the DropUnknownTags branch: filterTags only does work when it is set.
func TestEncode_NilTagElement(t *testing.T) {
	tags := func(withNil bool) []*gedcom.Tag {
		clean := []*gedcom.Tag{
			{Level: 1, Tag: "NAME", Value: "John /Doe/"},
			{Level: 1, Tag: "_CUSTOM", Value: "vendor"},
			{Level: 2, Tag: "NOTE", Value: "child of custom"},
			{Level: 1, Tag: "SEX", Value: "M"},
		}
		if !withNil {
			return clean
		}

		return []*gedcom.Tag{nil, clean[0], clean[1], nil, clean[2], clean[3], nil}
	}

	doc := func(withNil bool) *gedcom.Document {
		return &gedcom.Document{
			Header: &gedcom.Header{Version: gedcom.Version551},
			Records: []*gedcom.Record{
				{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: tags(withNil)},
			},
		}
	}

	for _, drop := range []bool{false, true} {
		t.Run("DropUnknownTags="+map[bool]string{true: "true", false: "false"}[drop], func(t *testing.T) {
			opts := DefaultOptions()
			opts.DropUnknownTags = drop

			encode := func(d *gedcom.Document) string {
				t.Helper()

				var buf bytes.Buffer
				if err := EncodeWithOptions(&buf, d, opts); err != nil {
					t.Fatalf("EncodeWithOptions() error = %v", err)
				}

				return buf.String()
			}

			want := encode(doc(false))
			if got := encode(doc(true)); got != want {
				t.Errorf("EncodeWithOptions() with nil tags =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestEncode_TypedNilEntity covers a Record.Entity holding a typed nil pointer.
// The interface itself is non-nil, so it passes entityToTags' nil check and
// reaches the record-level writer.
func TestEncode_TypedNilEntity(t *testing.T) {
	tests := []struct {
		name       string
		recordType gedcom.RecordType
		entity     interface{}
	}{
		{"Individual", gedcom.RecordTypeIndividual, (*gedcom.Individual)(nil)},
		{"Family", gedcom.RecordTypeFamily, (*gedcom.Family)(nil)},
		{"Source", gedcom.RecordTypeSource, (*gedcom.Source)(nil)},
		{"Submitter", gedcom.RecordTypeSubmitter, (*gedcom.Submitter)(nil)},
		{"Repository", gedcom.RecordTypeRepository, (*gedcom.Repository)(nil)},
		{"Note", gedcom.RecordTypeNote, (*gedcom.Note)(nil)},
		{"MediaObject", gedcom.RecordTypeMedia, (*gedcom.MediaObject)(nil)},
		{"SharedNote", gedcom.RecordTypeSharedNote, (*gedcom.SharedNote)(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := mustEncode(t, entityDoc(tt.recordType, nil))

			if got := mustEncode(t, entityDoc(tt.recordType, tt.entity)); got != want {
				t.Errorf("Encode() with a typed nil entity =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestEncode_NilHeader checks that a document with no header encodes as one with
// an empty header, options included.
func TestEncode_NilHeader(t *testing.T) {
	records := []*gedcom.Record{
		{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{XRef: "@I1@"}},
	}

	t.Run("default options", func(t *testing.T) {
		want := mustEncode(t, &gedcom.Document{Header: &gedcom.Header{}, Records: records})

		if got := mustEncode(t, &gedcom.Document{Records: records}); got != want {
			t.Errorf("Encode() with a nil header =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("TargetVersion still applies", func(t *testing.T) {
		opts := DefaultOptions()
		opts.TargetVersion = gedcom.Version70

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, &gedcom.Document{Records: records}, opts); err != nil {
			t.Fatalf("EncodeWithOptions() error = %v", err)
		}

		if !strings.Contains(buf.String(), "2 VERS 7.0") {
			t.Errorf("EncodeWithOptions() with a nil header lost TargetVersion:\n%s", buf.String())
		}
	})
}

// TestEncode_NilDocument checks that a missing document is reported rather than
// written as an empty one -- the one nil the encoder refuses instead of skips.
func TestEncode_NilDocument(t *testing.T) {
	tests := []struct {
		name   string
		encode func(*bytes.Buffer) error
	}{
		{"Encode", func(b *bytes.Buffer) error { return Encode(b, nil) }},
		{"EncodeWithOptions", func(b *bytes.Buffer) error { return EncodeWithOptions(b, nil, DefaultOptions()) }},
		{"EncodeStreaming", func(b *bytes.Buffer) error { return EncodeStreaming(b, nil) }},
		{"EncodeStreamingWithOptions", func(b *bytes.Buffer) error {
			return EncodeStreamingWithOptions(b, nil, DefaultOptions())
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := tt.encode(&buf)
			if !errors.Is(err, ErrNilDocument) {
				t.Errorf("error = %v, want ErrNilDocument", err)
			}

			if buf.Len() != 0 {
				t.Errorf("wrote %q, want nothing", buf.String())
			}
		})
	}
}

// TestStreamEncoder_WriteRecordNil checks that a nil record is a no-op that
// leaves the state machine untouched, while a misordered call still reports the
// misordering.
func TestStreamEncoder_WriteRecordNil(t *testing.T) {
	t.Run("state unchanged", func(t *testing.T) {
		var buf bytes.Buffer

		enc := NewStreamEncoder(&buf)
		if err := enc.WriteHeader(&gedcom.Header{Version: gedcom.Version551}); err != nil {
			t.Fatalf("WriteHeader() error = %v", err)
		}

		afterHeader := buf.String()

		if err := enc.WriteRecord(nil); err != nil {
			t.Fatalf("WriteRecord(nil) error = %v", err)
		}

		if buf.String() != afterHeader {
			t.Errorf("WriteRecord(nil) wrote %q", strings.TrimPrefix(buf.String(), afterHeader))
		}

		if got := enc.State(); got != "HeaderWritten" {
			t.Errorf("State() = %q, want %q", got, "HeaderWritten")
		}

		if err := enc.WriteTrailer(); err != nil {
			t.Fatalf("WriteTrailer() error = %v", err)
		}
	})

	t.Run("misordered call still reports", func(t *testing.T) {
		var buf bytes.Buffer

		enc := NewStreamEncoder(&buf)
		if err := enc.WriteRecord(nil); !errors.Is(err, ErrHeaderNotWritten) {
			t.Errorf("WriteRecord(nil) before the header: error = %v, want ErrHeaderNotWritten", err)
		}
	})
}

// TestToTagsWriters_NilArgument calls every entity writer with a nil directly.
// Most are reachable with a nil through Encode, but the ones whose only call
// site is already nil-guarded (a *CropRegion, *Address, *SourceCitationData or
// *Coordinates field) are not -- their guard is the reason a future caller
// cannot reintroduce the panic, so it is asserted here rather than left untested.
func TestToTagsWriters_NilArgument(t *testing.T) {
	opts := DefaultOptions()

	tests := []struct {
		name  string
		write func() []*gedcom.Tag
	}{
		{"individualToTags", func() []*gedcom.Tag { return individualToTags(nil, opts) }},
		{"familyToTags", func() []*gedcom.Tag { return familyToTags(nil, opts) }},
		{"sourceToTags", func() []*gedcom.Tag { return sourceToTags(nil, opts) }},
		{"sourceRepositoryLinkToTags", func() []*gedcom.Tag { return sourceRepositoryLinkToTags(nil, opts) }},
		{"submitterToTags", func() []*gedcom.Tag { return submitterToTags(nil, opts) }},
		{"repositoryToTags", func() []*gedcom.Tag { return repositoryToTags(nil, opts) }},
		{"noteToTags", func() []*gedcom.Tag { return noteToTags(nil) }},
		{"mediaObjectToTags", func() []*gedcom.Tag { return mediaObjectToTags(nil, opts) }},
		{"sharedNoteToTags", func() []*gedcom.Tag { return sharedNoteToTags(nil, opts) }},
		{"nameToTags", func() []*gedcom.Tag { return nameToTags(nil, 1) }},
		{"transliterationToTags", func() []*gedcom.Tag { return transliterationToTags(nil, 2) }},
		{"eventToTags", func() []*gedcom.Tag { return eventToTags(nil, 1, opts) }},
		{"attributeToTags", func() []*gedcom.Tag { return attributeToTags(nil, 1, opts) }},
		{"sourceCitationToTags", func() []*gedcom.Tag { return sourceCitationToTags(nil, 1, opts) }},
		{"sourceCitationDataToTags", func() []*gedcom.Tag { return sourceCitationDataToTags(nil, 2, opts) }},
		{"addressToTags", func() []*gedcom.Tag { return addressToTags(nil, 2) }},
		{"coordinatesToTags", func() []*gedcom.Tag { return coordinatesToTags(nil, 3) }},
		{"ldsOrdinanceToTags", func() []*gedcom.Tag { return ldsOrdinanceToTags(nil, 1, opts) }},
		{"familyLinkToTags", func() []*gedcom.Tag { return familyLinkToTags(nil, 1) }},
		{"associationToTags", func() []*gedcom.Tag { return associationToTags(nil, 1, opts) }},
		{"changeDateToTags", func() []*gedcom.Tag { return changeDateToTags(nil, 1, "CHAN", opts) }},
		{"mediaLinkToTags", func() []*gedcom.Tag { return mediaLinkToTags(nil, 1) }},
		{"cropRegionToTags", func() []*gedcom.Tag { return cropRegionToTags(nil, 2) }},
		{"mediaFileToTags", func() []*gedcom.Tag { return mediaFileToTags(nil, 1) }},
		{"mediaTranslationToTags", func() []*gedcom.Tag { return mediaTranslationToTags(nil, 2) }},
		{"sharedNoteTranslationToTags", func() []*gedcom.Tag { return sharedNoteTranslationToTags(nil, 1, opts) }},
		{"externalIDsToTags", func() []*gedcom.Tag { return externalIDsToTags([]*gedcom.ExternalID{nil}, 1) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.write(); len(got) != 0 {
				t.Errorf("%s(nil) = %v, want no tags", tt.name, got)
			}
		})
	}
}
