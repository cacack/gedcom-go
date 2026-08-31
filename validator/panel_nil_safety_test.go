package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// Guards for nil elements in an entity's slice fields. These readers are not a
// range over Document.Records or Record.Tags -- one reads Names positionally,
// two walk an event slice from a helper -- which is why nil_safety_test.go's
// record-and-tag fixtures do not reach them.

func panelInd(xref string, names []*gedcom.PersonalName, events []*gedcom.Event) *gedcom.Record {
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

func aName() []*gedcom.PersonalName {
	return []*gedcom.PersonalName{{Full: "Ada /Lovelace/"}}
}

// A nil element ahead of the real name must not be taken for the primary name.
// Duplicate detection needs two individuals before it runs at all.
func TestValidateAll_NilPersonalNameElement(t *testing.T) {
	got := New().ValidateAll(panelDoc(
		panelInd("@I1@", []*gedcom.PersonalName{nil, {Full: "Ada /Lovelace/"}}, nil),
		panelInd("@I2@", aName(), nil),
	))
	want := New().ValidateAll(panelDoc(
		panelInd("@I1@", aName(), nil),
		panelInd("@I2@", aName(), nil),
	))
	if len(want) == 0 {
		t.Fatal("nil-free document reported no issues; the comparison would pass vacuously")
	}
	if len(got) != len(want) {
		t.Errorf("ValidateAll issues = %d, want %d (must match the nil-free document)", len(got), len(want))
	}
}

// extractSurname, getDisplayName and extractGivenName each read the primary
// name; all three go through primaryName now.
func TestPrimaryName_SkipsNilElements(t *testing.T) {
	ind := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{nil, {Given: "Ada", Surname: "Lovelace", Full: "Ada /Lovelace/"}},
	}
	detector := NewDuplicateDetector(nil)

	if got := detector.extractSurname(ind); got != "Lovelace" {
		t.Errorf("extractSurname() = %q, want %q", got, "Lovelace")
	}
	if got := extractGivenName(ind); got != "Ada" {
		t.Errorf("extractGivenName() = %q, want %q", got, "Ada")
	}
	if got := getDisplayName(ind); got != "Ada Lovelace" {
		t.Errorf("getDisplayName() = %q, want %q", got, "Ada Lovelace")
	}

	empty := &gedcom.Individual{XRef: "@I2@", Names: []*gedcom.PersonalName{nil, nil}}
	if got := getDisplayName(empty); got != "@I2@" {
		t.Errorf("getDisplayName() with only nils = %q, want the XRef", got)
	}
	if got := detector.extractSurname(empty); got != "" {
		t.Errorf("extractSurname() with only nils = %q, want empty", got)
	}
	if got := extractGivenName(empty); got != "" {
		t.Errorf("extractGivenName() with only nils = %q, want empty", got)
	}
}

// Family.Events is the sibling of Individual.Events, reached from date logic.
func TestValidateAll_NilEventInFamilyEvents(t *testing.T) {
	build := func(events []*gedcom.Event) *gedcom.Document {
		ind := panelInd("@I1@", aName(), []*gedcom.Event{
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
	got := New().ValidateAll(build([]*gedcom.Event{nil, marriage}))
	want := New().ValidateAll(build([]*gedcom.Event{marriage}))
	if len(want) == 0 {
		t.Fatal("nil-free document reported no issues; the comparison would pass vacuously")
	}
	if len(got) != len(want) {
		t.Errorf("ValidateAll issues = %d, want %d (must match the nil-free document)", len(got), len(want))
	}
}

// hasPlace is the third reader of Individual.Events; BirthEvent and DeathEvent
// were the two that were guarded first.
func TestQualityAnalyzer_NilEventElement(t *testing.T) {
	event := &gedcom.Event{Type: gedcom.EventBirth, PlaceDetail: &gedcom.PlaceDetail{Name: "London"}}
	got := NewQualityAnalyzer().Analyze(panelDoc(
		panelInd("@I1@", aName(), []*gedcom.Event{nil, event}),
	))
	want := NewQualityAnalyzer().Analyze(panelDoc(
		panelInd("@I1@", aName(), []*gedcom.Event{event}),
	))
	if want.IndividualsWithPlaces != 1 {
		t.Fatalf("nil-free IndividualsWithPlaces = %d, want 1; the comparison would pass vacuously",
			want.IndividualsWithPlaces)
	}
	if got.IndividualsWithPlaces != want.IndividualsWithPlaces {
		t.Errorf("IndividualsWithPlaces = %d, want %d",
			got.IndividualsWithPlaces, want.IndividualsWithPlaces)
	}
}

// primaryName, validateIndividual and validateFamily each guard an argument the
// public path already filters. The guards are defensive, so only a direct call
// reaches them -- and a guard no test can enter is a guard no test can defend.
func TestNilArgumentGuards(t *testing.T) {
	if got := primaryName(nil); got != nil {
		t.Errorf("primaryName(nil) = %#v, want nil", got)
	}

	v := New()
	v.validateIndividual(nil)
	v.validateFamily(nil)
	if len(v.errors) != 0 {
		t.Errorf("nil records produced %d error(s), want 0", len(v.errors))
	}
}

// A nil tag on a family record reaches validateFamily's own loop, which is a
// different loop from the individual one the record/tag fixtures exercise.
func TestValidate_NilTagOnFamilyRecord(t *testing.T) {
	build := func(tags []*gedcom.Tag) *gedcom.Document {
		return panelDoc(&gedcom.Record{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Tags: tags})
	}
	husband := &gedcom.Tag{Level: 1, Tag: "HUSB", Value: "@I1@"}

	withNil := New().Validate(build([]*gedcom.Tag{nil, husband, nil}))
	clean := New().Validate(build([]*gedcom.Tag{husband}))
	if len(withNil) != len(clean) {
		t.Errorf("Validate errors = %d, want %d (must match the nil-free document)", len(withNil), len(clean))
	}

	// A family whose only tags are nil has no members, exactly like an empty one.
	onlyNils := New().Validate(build([]*gedcom.Tag{nil, nil}))
	empty := New().Validate(build(nil))
	if len(onlyNils) != len(empty) {
		t.Errorf("all-nil family errors = %d, want %d", len(onlyNils), len(empty))
	}
	if len(empty) == 0 {
		t.Fatal("empty family reported no issues; the comparison would pass vacuously")
	}
}
