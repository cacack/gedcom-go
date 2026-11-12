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
			name: "MIME on source TEXT",
			doc:  &Document{Records: []*Record{{Type: RecordTypeSource, Entity: &Source{Text: "x", TextMIME: "text/html"}}}},
		},
		{
			name: "LANG on source TEXT",
			doc:  &Document{Records: []*Record{{Type: RecordTypeSource, Entity: &Source{Text: "x", TextLanguage: "en"}}}},
		},
		{
			name: "DATE PHRASE on source DATA event",
			doc: &Document{Records: []*Record{{Type: RecordTypeSource, Entity: &Source{Data: &SourceData{Events: []*SourceDataEvent{
				nil, {Types: "BIRT", Date: "1900", DatePhrase: "the year 1900"},
			}}}}}},
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
		{
			name: "DATE PHRASE on event",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, Date: "1850", ParsedDate: &Date{Original: "1850", Year: 1850, Phrase: "about 1850"}}}}),
		},
		{
			name: "DATE PHRASE on attribute",
			doc:  indiDoc(&Individual{Attributes: []*Attribute{{Type: AttributeOccupation, ParsedDate: &Date{Phrase: "while young", IsPhrase: true}}}}),
		},
		{
			name: "DATE PHRASE on individual LDS ordinance",
			doc:  indiDoc(&Individual{LDSOrdinances: []*LDSOrdinance{{Type: LDSBaptism, ParsedDate: &Date{Phrase: "as a child", IsPhrase: true}}}}),
		},
		{
			name: "ADOP PHRASE on an adoption's FAMC",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventAdoption, FamilyXRef: "@F1@", AdoptedBy: "BOTH", AdoptedByPhrase: "Both"}}}),
		},
		{
			name: "TIME on an LDS status date",
			doc:  indiDoc(&Individual{LDSOrdinances: []*LDSOrdinance{{Type: LDSBaptism, Status: "COMPLETED", StatusDate: "1 JAN 2000", StatusTime: "12:00"}}}),
		},
		{
			name: "DATE PHRASE on family LDS ordinance",
			doc: &Document{Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{
				LDSOrdinances: []*LDSOrdinance{{Type: LDSSealingSpouse, ParsedDate: &Date{Phrase: "after the war", IsPhrase: true}}},
			}}}},
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
			// A 5.5.1 date phrase lives in the DATE value, not a PHRASE.
			name: "date phrase in the value",
			doc:  indiDoc(&Individual{Events: []*Event{{Type: EventBirth, Date: "(unknown)", ParsedDate: &Date{Original: "(unknown)", Phrase: "unknown", IsPhrase: true}}}}),
		},
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
				{Type: RecordTypeSource, Entity: &Source{Text: "x", Data: &SourceData{Agency: "A", Events: []*SourceDataEvent{nil, {Types: "BIRT", Date: "1900"}}}}},
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

// citationHolders places a citation on every structure that can hold one,
// each in its own document, so a detection test reaches the citation from
// every position the walk must cover.
func citationHolders(c *SourceCitation) map[string]*Document {
	cits := []*SourceCitation{c}
	return map[string]*Document{
		"individual":  indiDoc(&Individual{SourceCitations: cits}),
		"name":        indiDoc(&Individual{Names: []*PersonalName{{Full: "John /Doe/", SourceCitations: cits}}}),
		"association": indiDoc(&Individual{Associations: []*Association{{IndividualXRef: "@I2@", Role: "WITN", SourceCitations: cits}}}),
		"event":       indiDoc(&Individual{Events: []*Event{{Type: EventBirth, SourceCitations: cits}}}),
		"attribute":   indiDoc(&Individual{Attributes: []*Attribute{{Type: "OCCU", Value: "Smith", SourceCitations: cits}}}),
		"family": {Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{
			SourceCitations: cits,
		}}}},
		"family event": {Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{
			Events: []*Event{{Type: EventMarriage, SourceCitations: cits}},
		}}}},
		"media object": {Records: []*Record{{Type: RecordTypeMedia, Entity: &MediaObject{
			Files: []*MediaFile{{FileRef: "a.jpg", Form: "image/jpeg"}}, SourceCitations: cits,
		}}}},
	}
}

func TestDocument_RequiresGEDCOM7_Citations(t *testing.T) {
	features := []struct {
		name     string
		citation *SourceCitation
		want     bool
	}{
		{
			name:     "DATA.TEXT MIME",
			citation: &SourceCitation{SourceXRef: "@S1@", Data: &SourceCitationData{Text: []*SourceText{{Value: "x", MIME: "text/html"}}}},
			want:     true,
		},
		{
			name:     "DATA.TEXT LANG",
			citation: &SourceCitation{SourceXRef: "@S1@", Data: &SourceCitationData{Text: []*SourceText{nil, {Value: "x", Language: "de"}}}},
			want:     true,
		},
		{
			name:     "EVEN PHRASE",
			citation: &SourceCitation{SourceXRef: "@S1@", Event: &SourceCitationEvent{Type: "BIRT", Phrase: "the baptism"}},
			want:     true,
		},
		{
			name:     "ROLE PHRASE",
			citation: &SourceCitation{SourceXRef: "@S1@", Event: &SourceCitationEvent{Type: "BIRT", Role: "OTHER", RolePhrase: "midwife"}},
			want:     true,
		},
		{
			name:     "media CROP",
			citation: &SourceCitation{SourceXRef: "@S1@", Media: []*MediaLink{{MediaXRef: "@M1@", Crop: &CropRegion{Top: 1}}}},
			want:     true,
		},
		{
			// Everything 5.5.1 can carry on a citation: PAGE, QUAY, DATA with
			// DATE and plain TEXT, EVEN with ROLE, OBJE without CROP, notes.
			name: "5.5.1-representable citation",
			citation: &SourceCitation{
				SourceXRef: "@S1@", Page: "p. 4", Quality: new(int),
				Data:        &SourceCitationData{Date: "1 JAN 1900", Text: []*SourceText{{Value: "x"}}},
				Event:       &SourceCitationEvent{Type: "BIRT", Role: "CHIL"},
				Media:       []*MediaLink{{MediaXRef: "@M1@"}},
				NoteXRefs:   []string{"@N1@"},
				InlineNotes: []string{"n"},
			},
			want: false,
		},
	}

	for _, f := range features {
		for holder, doc := range citationHolders(f.citation) {
			t.Run(f.name+" on "+holder, func(t *testing.T) {
				if got := doc.RequiresGEDCOM7(); got != f.want {
					t.Errorf("RequiresGEDCOM7() = %v, want %v", got, f.want)
				}
			})
		}
	}
}
