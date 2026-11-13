package gedcom

import "testing"

// TestRequiresGEDCOM7_787Fields pins that the typed fields #787 added, all
// 7.0-only, make RequiresGEDCOM7 report true wherever the encoder writes
// them, and that their empty forms do not.
func TestRequiresGEDCOM7_787Fields(t *testing.T) {
	timed := []*SourceCitation{{SourceXRef: "@S1@", Data: &SourceCitationData{Date: "1850", Time: "08:00"}}}
	phrased := []*SourceCitation{{SourceXRef: "@S1@", Data: &SourceCitationData{Date: "1850", DatePhrase: "about then"}}}
	exidPlace := &PlaceDetail{Name: "Sampletown", ExternalIDs: []*ExternalID{{Value: "999"}}}

	tests := []struct {
		name string
		doc  *Document
		want bool
	}{
		{"citation DATA.DATE.TIME on individual", indiDoc(&Individual{SourceCitations: timed}), true},
		{"citation DATA.DATE.PHRASE on event", indiDoc(&Individual{Events: []*Event{{Type: EventBirth, SourceCitations: phrased}}}), true},
		{"citation DATA.DATE.PHRASE on attribute", indiDoc(&Individual{Attributes: []*Attribute{{Type: AttributeOccupation, SourceCitations: phrased}}}), true},
		{"citation DATA.DATE.TIME on name", indiDoc(&Individual{Names: []*PersonalName{{Full: "A /B/", SourceCitations: timed}}}), true},
		{"citation DATA.DATE.TIME on association", indiDoc(&Individual{Associations: []*Association{{IndividualXRef: "@I2@", SourceCitations: timed}}}), true},
		{"citation DATA.DATE.TIME on family", &Document{Records: []*Record{{Type: RecordTypeFamily, Entity: &Family{SourceCitations: timed}}}}, true},
		{"citation DATA.DATE.TIME on media object", &Document{Records: []*Record{{Type: RecordTypeMedia, Entity: &MediaObject{SourceCitations: timed}}}}, true},
		{"PLAC EXID on event", indiDoc(&Individual{Events: []*Event{{Type: EventBirth, PlaceDetail: exidPlace}}}), true},
		{"PLAC EXID on attribute", indiDoc(&Individual{Attributes: []*Attribute{{Type: AttributeOccupation, PlaceDetail: exidPlace}}}), true},
		{"PLAC EXID on source DATA.EVEN", &Document{Records: []*Record{{Type: RecordTypeSource, Entity: &Source{Data: &SourceData{Events: []*SourceDataEvent{{Types: "BIRT", PlaceDetail: exidPlace}}}}}}}, true},
		{"inline NOTE LANG", indiDoc(&Individual{InlineNotes: []string{"n"}, InlineNoteDetails: []*InlineNoteDetail{{Language: "en"}}}), true},
		{"inline NOTE TRAN", indiDoc(&Individual{InlineNotes: []string{"n"}, InlineNoteDetails: []*InlineNoteDetail{{Translations: []*SharedNoteTranslation{{Value: "x"}}}}}), true},
		{"repeated GIVN", indiDoc(&Individual{Names: []*PersonalName{{Given: "Alex", AdditionalGiven: []string{"Alexander"}}}}), true},
		{"repeated SPFX", indiDoc(&Individual{Names: []*PersonalName{{SurnamePrefix: "van", AdditionalSurnamePrefixes: []string{"der"}}}}), true},
		{"citation DATA.DATE without TIME or PHRASE", indiDoc(&Individual{SourceCitations: []*SourceCitation{{SourceXRef: "@S1@", Data: &SourceCitationData{Date: "1850"}}}}), false},
		{"PLAC without EXID", indiDoc(&Individual{Events: []*Event{{Type: EventBirth, PlaceDetail: &PlaceDetail{Name: "Sampletown"}}}}), false},
		{"empty inline note detail", indiDoc(&Individual{InlineNotes: []string{"n"}, InlineNoteDetails: []*InlineNoteDetail{nil, {}}}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.doc.RequiresGEDCOM7(); got != tt.want {
				t.Errorf("RequiresGEDCOM7() = %v, want %v", got, tt.want)
			}
		})
	}
}
