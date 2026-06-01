package gedcom

import "testing"

// indiDoc wraps a single Individual in a Document for detection tests.
func indiDoc(i *Individual) *Document {
	return &Document{Records: []*Record{{Type: RecordTypeIndividual, Entity: i}}}
}

func TestDocument_RequiresGEDCOM7_True(t *testing.T) {
	tests := []struct {
		name string
		doc  *Document
	}{
		{
			name: "SCHMA header substructure",
			doc:  &Document{Schema: &SchemaDefinition{TagMappings: map[string]string{"_X": "http://e.com/x"}}},
		},
		{
			name: "SNOTE record",
			doc:  &Document{Records: []*Record{{Type: RecordTypeSharedNote, Entity: &SharedNote{XRef: "@N1@"}}}},
		},
		{
			name: "EXID on individual",
			doc:  indiDoc(&Individual{ExternalIDs: []*ExternalID{{Value: "12345", Type: "http://fs.org"}}}),
		},
		{
			name: "EXID on repository",
			doc:  &Document{Records: []*Record{{Type: RecordTypeRepository, Entity: &Repository{ExternalIDs: []*ExternalID{{Value: "x"}}}}}},
		},
		{
			name: "EXID on submitter",
			doc:  &Document{Records: []*Record{{Type: RecordTypeSubmitter, Entity: &Submitter{ExternalIDs: []*ExternalID{{Value: "x"}}}}}},
		},
		{
			name: "EXID on note",
			doc:  &Document{Records: []*Record{{Type: RecordTypeNote, Entity: &Note{ExternalIDs: []*ExternalID{{Value: "x"}}}}}},
		},
		{
			name: "EXID on source",
			doc:  &Document{Records: []*Record{{Type: RecordTypeSource, Entity: &Source{ExternalIDs: []*ExternalID{{Value: "x"}}}}}},
		},
		{
			name: "CROP on source media",
			doc:  &Document{Records: []*Record{{Type: RecordTypeSource, Entity: &Source{Media: []*MediaLink{{Crop: &CropRegion{Width: 5, Height: 5}}}}}}},
		},
		{
			name: "EXID on family",
			doc:  &Document{Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{ExternalIDs: []*ExternalID{{Value: "x"}}}}}},
		},
		{
			name: "CREA on family",
			doc:  &Document{Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{CreationDate: &ChangeDate{Date: "1 JAN 2020"}}}}},
		},
		{
			name: "NO negative event on family",
			doc:  &Document{Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{Events: []*Event{{Type: EventMarriage, IsNegative: true}}}}}},
		},
		{
			name: "CROP on family media",
			doc:  &Document{Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{Media: []*MediaLink{{Crop: &CropRegion{Width: 5, Height: 5}}}}}}},
		},
		{
			name: "CROP on event media link",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, Media: []*MediaLink{{Crop: &CropRegion{Width: 5, Height: 5}}}}}}),
		},
		{
			name: "CREA creation date",
			doc:  indiDoc(&Individual{CreationDate: &ChangeDate{Date: "1 JAN 2020"}}),
		},
		{
			name: "NO negative event",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, IsNegative: true}}}),
		},
		{
			name: "SDATE sort date",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, SortDate: "1 JAN 2020"}}}),
		},
		{
			name: "NAME TRAN transliteration",
			doc:  indiDoc(&Individual{Names: []*PersonalName{{Full: "John /Doe/", Transliterations: []*Transliteration{{Value: "ジョン"}}}}}),
		},
		{
			name: "ASSO PHRASE",
			doc:  indiDoc(&Individual{Associations: []*Association{{IndividualXRef: "@I2@", Phrase: "Mr Stockdale"}}}),
		},
		{
			name: "media CROP on individual link",
			doc:  indiDoc(&Individual{Media: []*MediaLink{{Crop: &CropRegion{Width: 10, Height: 10}}}}),
		},
		{
			name: "EXID on media object",
			doc:  &Document{Records: []*Record{{Type: RecordTypeMedia, Entity: &MediaObject{ExternalIDs: []*ExternalID{{Value: "x"}}}}}},
		},
		{
			name: "CREA on media object",
			doc:  &Document{Records: []*Record{{Type: RecordTypeMedia, Entity: &MediaObject{CreationDate: &ChangeDate{Date: "1 JAN 2020"}}}}},
		},
		{
			name: "media file TRAN",
			doc:  &Document{Records: []*Record{{Type: RecordTypeMedia, Entity: &MediaObject{Files: []*MediaFile{{FileRef: "a.jpg", Translations: []*MediaTranslation{{FileRef: "b.png"}}}}}}}},
		},
		{
			name: "media SNOTE reference",
			doc:  &Document{Records: []*Record{{Type: RecordTypeMedia, Entity: &MediaObject{SharedNoteXRefs: []string{"@N1@"}}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.doc.RequiresGEDCOM7() {
				t.Errorf("RequiresGEDCOM7() = false, want true")
			}
			if got := tt.doc.MinimumVersion(); got != Version70 {
				t.Errorf("MinimumVersion() = %v, want %v", got, Version70)
			}
		})
	}
}

func TestDocument_RequiresGEDCOM7_False(t *testing.T) {
	tests := []struct {
		name string
		doc  *Document
	}{
		{name: "nil document", doc: nil},
		{name: "empty document", doc: &Document{}},
		{
			name: "plain individual with name and event",
			doc:  indiDoc(&Individual{Names: []*PersonalName{{Full: "John /Doe/"}}, Events: []*Event{{Type: EventBirth, Date: "1 JAN 1900"}}}),
		},
		{
			// INT (interpreted) dates are valid in 5.5.1 and must NOT force 7.0.
			name: "INT interpreted date",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, Date: "INT 1 JAN 1900 (guess)", ParsedDate: &Date{Modifier: ModifierInterpreted, IsInterpreted: true}}}}),
		},
		{
			// Non-Gregorian calendars are valid in 5.5.1 and must NOT force 7.0.
			name: "Hebrew calendar date",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, ParsedDate: &Date{Calendar: CalendarHebrew, Year: 5785}}}}),
		},
		{
			// MAP/LATI/LONG coordinates are a 5.5.1 feature, not 7.0.
			name: "place coordinates (5.5.1)",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, PlaceDetail: &PlaceDetail{Name: "Boston", Coordinates: &Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"}}}}}),
		},
		{
			// ASSO with a source citation but no PHRASE: 5.5.1 allows SOUR under ASSO.
			name: "ASSO with source citation but no phrase",
			doc:  indiDoc(&Individual{Associations: []*Association{{IndividualXRef: "@I2@", Role: "WITN", SourceCitations: []*SourceCitation{{SourceXRef: "@S1@"}}}}}),
		},
		{
			// CHAN (change date) is 5.5.1; only CREA is 7.0.
			name: "CHAN change date only",
			doc:  indiDoc(&Individual{ChangeDate: &ChangeDate{Date: "1 JAN 2020"}}),
		},
		{
			name: "plain family with event and media",
			doc: &Document{Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{
				Events: []*Event{{Type: EventMarriage, Date: "1 JAN 1900"}},
				Media:  []*MediaLink{{MediaXRef: "@M1@"}},
			}}}},
		},
		{
			name: "plain media object with untranslated file",
			doc: &Document{Records: []*Record{{Type: RecordTypeMedia, Entity: &MediaObject{
				Files: []*MediaFile{{FileRef: "a.jpg", Form: "image/jpeg"}},
			}}}},
		},
		{
			name: "plain source and submitter records",
			doc: &Document{Records: []*Record{
				{Type: RecordTypeSource, Entity: &Source{Title: "Census"}},
				{Type: RecordTypeSubmitter, Entity: &Submitter{Name: "Jane"}},
				{Type: RecordTypeNote, Entity: &Note{Text: "hi"}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doc.RequiresGEDCOM7() {
				t.Errorf("RequiresGEDCOM7() = true, want false")
			}
			if got := tt.doc.MinimumVersion(); got != Version551 {
				t.Errorf("MinimumVersion() = %v, want %v", got, Version551)
			}
		})
	}
}
