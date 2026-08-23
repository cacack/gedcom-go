package gedcomgo

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
	"github.com/cacack/gedcom-go/v2/validator"
)

// The panel review of #458 found nil shapes that the per-package suites missed:
// each one reaches a reader that is not a range over Document.Records or
// Record.Tags, which is the shape those suites were built around. A positional
// read (Names[0]), a helper that walks an entity slice (hasPlace), and an inline
// clone of a slice element all dereference an element the caller supplied.
//
// Every case here panicked before its guard landed, through a public entry
// point, so each test fails if the corresponding guard is removed.

func panelIndividual(xref string, names []*gedcom.PersonalName, events []*gedcom.Event) *gedcom.Record {
	return &gedcom.Record{
		XRef: xref,
		Type: gedcom.RecordTypeIndividual,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Ada /Lovelace/"}},
		Entity: &gedcom.Individual{
			XRef:   xref,
			Names:  names,
			Events: events,
		},
	}
}

func panelDoc(records ...*gedcom.Record) *gedcom.Document {
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: records,
		XRefMap: make(map[string]*gedcom.Record, len(records)),
	}
	for _, record := range records {
		doc.XRefMap[record.XRef] = record
	}
	return doc
}

func named() []*gedcom.PersonalName { return []*gedcom.PersonalName{{Full: "Ada /Lovelace/"}} }

// A nil *PersonalName ahead of a real one must not be mistaken for the primary
// name. Duplicate detection needs two individuals to run at all, so a
// single-individual fixture cannot reach this.
func TestValidateAll_NilPersonalNameElement(t *testing.T) {
	withNil := panelDoc(
		panelIndividual("@I1@", []*gedcom.PersonalName{nil, {Full: "Ada /Lovelace/"}}, nil),
		panelIndividual("@I2@", named(), nil),
	)
	clean := panelDoc(
		panelIndividual("@I1@", named(), nil),
		panelIndividual("@I2@", named(), nil),
	)

	got, want := ValidateAll(withNil), ValidateAll(clean)
	if len(want) == 0 {
		t.Fatal("nil-free document reported no issues; the comparison would pass vacuously")
	}
	if len(got) != len(want) {
		t.Errorf("ValidateAll issues = %d, want %d (must match the nil-free document)", len(got), len(want))
	}
}

// Family.Events is the sibling of Individual.Events; only the latter was guarded.
func TestValidateAll_NilEventInFamilyEvents(t *testing.T) {
	build := func(events []*gedcom.Event) *gedcom.Document {
		ind := panelIndividual("@I1@", named(), []*gedcom.Event{
			{Type: gedcom.EventBirth, ParsedDate: &gedcom.Date{Year: 1900}},
		})
		ind.Entity.(*gedcom.Individual).SpouseInFamilies = []string{"@F1@"}
		fam := &gedcom.Record{
			XRef:   "@F1@",
			Type:   gedcom.RecordTypeFamily,
			Tags:   []*gedcom.Tag{{Level: 1, Tag: "HUSB", Value: "@I1@"}},
			Entity: &gedcom.Family{XRef: "@F1@", Husband: "@I1@", Events: events},
		}
		return panelDoc(ind, fam)
	}

	marriage := &gedcom.Event{Type: gedcom.EventMarriage, ParsedDate: &gedcom.Date{Year: 1890}}
	got := ValidateAll(build([]*gedcom.Event{nil, marriage}))
	want := ValidateAll(build([]*gedcom.Event{marriage}))
	if len(want) == 0 {
		t.Fatal("nil-free document reported no issues; the comparison would pass vacuously")
	}
	if len(got) != len(want) {
		t.Errorf("ValidateAll issues = %d, want %d (must match the nil-free document)", len(got), len(want))
	}
}

// hasPlace is the third reader of Individual.Events; BirthEvent and DeathEvent
// were the two that got guarded. Analyze is validator API, not root facade API.
func TestQualityAnalyzer_NilEventElement(t *testing.T) {
	event := &gedcom.Event{Type: gedcom.EventBirth, Place: "London"}
	got := validator.NewQualityAnalyzer().Analyze(panelDoc(
		panelIndividual("@I1@", named(), []*gedcom.Event{nil, event}),
	))
	want := validator.NewQualityAnalyzer().Analyze(panelDoc(
		panelIndividual("@I1@", named(), []*gedcom.Event{event}),
	))
	if got == nil || want == nil {
		t.Fatal("Analyze returned nil report")
	}
	if want.IndividualsWithPlaces != 1 {
		t.Fatalf("nil-free IndividualsWithPlaces = %d, want 1; the comparison would pass vacuously", want.IndividualsWithPlaces)
	}
	if got.IndividualsWithPlaces != want.IndividualsWithPlaces {
		t.Errorf("IndividualsWithPlaces = %d, want %d", got.IndividualsWithPlaces, want.IndividualsWithPlaces)
	}
}

// Document.Clone builds translation elements inline instead of delegating to a
// nil-guarded helper, so a nil element there panicked. Convert clones, so this
// reaches the converter too.
func TestClone_NilTranslationElement(t *testing.T) {
	doc := panelDoc(
		&gedcom.Record{
			XRef: "@N1@", Type: gedcom.RecordTypeSharedNote,
			Entity: &gedcom.SharedNote{XRef: "@N1@", Translations: []*gedcom.SharedNoteTranslation{nil, {Value: "bonjour"}}},
		},
		&gedcom.Record{
			XRef: "@M1@", Type: gedcom.RecordTypeMedia,
			Entity: &gedcom.MediaObject{XRef: "@M1@", Files: []*gedcom.MediaFile{
				{FileRef: "a.jpg", Translations: []*gedcom.MediaTranslation{nil, {FileRef: "b.jpg"}}},
			}},
		},
	)

	clone := doc.Clone()

	snote, _ := clone.GetRecord("@N1@").GetSharedNote()
	if len(snote.Translations) != 2 || snote.Translations[0] != nil || snote.Translations[1].Value != "bonjour" {
		t.Errorf("shared note translations = %#v, want [nil, {bonjour}]", snote.Translations)
	}
	media, _ := clone.GetRecord("@M1@").GetMediaObject()
	trans := media.Files[0].Translations
	if len(trans) != 2 || trans[0] != nil || trans[1].FileRef != "b.jpg" {
		t.Errorf("media translations = %#v, want [nil, {b.jpg}]", trans)
	}
}

// A nil header converted to the version it already defaults to took the
// no-conversion early return, which skipped the header materialisation and
// handed back a document whose Header was still nil. The guarantee is
// equivalence with the empty header a nil one is read as -- not a stamped
// version, which this path does not apply to an empty header either.
func TestConvert_NilHeaderEveryTarget(t *testing.T) {
	for _, target := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
		t.Run(string(target), func(t *testing.T) {
			nilHeader := &gedcom.Document{Records: []*gedcom.Record{}}
			emptyHeader := &gedcom.Document{Header: &gedcom.Header{}, Records: []*gedcom.Record{}}

			got, _, err := Convert(nilHeader, target)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			want, _, err := Convert(emptyHeader, target)
			if err != nil {
				t.Fatalf("Convert() with empty header error = %v", err)
			}

			if got.Header == nil {
				t.Fatal("converted document has a nil Header")
			}
			if got.Header.Version != want.Header.Version {
				t.Errorf("Header.Version = %q, want %q (must match the empty-header twin)",
					got.Header.Version, want.Header.Version)
			}
			if nilHeader.Header != nil {
				t.Error("Convert() mutated the caller's document")
			}
		})
	}
}
