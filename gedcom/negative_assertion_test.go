package gedcom_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// negativeAssertion70 records, for @I1@, a birth and a negative assertion
// that he did NOT die between 1900 and 1910 -- a period before his birth, so
// any reader that mistakes the NO DEAT for a death sees an impossible date.
// @F1@ records only that the couple did NOT marry, again before either birth.
const negativeAssertion70 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 BIRT
2 DATE 1 JAN 1920
1 NO DEAT
2 DATE FROM 1900 TO 1910
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 NO BIRT
2 DATE FROM 1800 TO 1810
1 BIRT
2 DATE 1 JAN 1921
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 NO MARR
2 DATE FROM 1800 TO 1850
0 TRLR
`

func decodeNegativeAssertion70(t *testing.T) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(negativeAssertion70))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return doc
}

func decodeMaximal70(t *testing.T) *gedcom.Document {
	t.Helper()
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatalf("open maximal70.ged: %v", err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("Decode(maximal70.ged) error = %v", err)
	}
	return doc
}

func eventTypes(events []*gedcom.Event) []gedcom.EventType {
	var out []gedcom.EventType
	for _, e := range events {
		out = append(out, e.Type)
	}
	return out
}

func TestIndividualOccurredEventsAndNegativeAssertions(t *testing.T) {
	birth := &gedcom.Event{Type: gedcom.EventBirth}
	noDeath := &gedcom.Event{Type: gedcom.EventDeath, IsNegative: true}
	death := &gedcom.Event{Type: gedcom.EventDeath}
	ind := &gedcom.Individual{Events: []*gedcom.Event{birth, noDeath, nil, death}}

	if got, want := ind.OccurredEvents(), []*gedcom.Event{birth, death}; !reflect.DeepEqual(got, want) {
		t.Errorf("OccurredEvents() = %v, want %v", eventTypes(got), eventTypes(want))
	}
	if got, want := ind.NegativeAssertions(), []*gedcom.Event{noDeath}; !reflect.DeepEqual(got, want) {
		t.Errorf("NegativeAssertions() = %v, want %v", eventTypes(got), eventTypes(want))
	}
	if len(ind.Events) != 4 {
		t.Errorf("helpers must not modify Events; len = %d, want 4", len(ind.Events))
	}

	var nilInd *gedcom.Individual
	if got := nilInd.OccurredEvents(); got != nil {
		t.Errorf("nil receiver OccurredEvents() = %v, want nil", got)
	}
	if got := nilInd.NegativeAssertions(); got != nil {
		t.Errorf("nil receiver NegativeAssertions() = %v, want nil", got)
	}
	empty := &gedcom.Individual{Events: []*gedcom.Event{birth}}
	if got := empty.NegativeAssertions(); got != nil {
		t.Errorf("NegativeAssertions() with none = %v, want nil", got)
	}
}

func TestFamilyOccurredEventsAndNegativeAssertions(t *testing.T) {
	noMarr := &gedcom.Event{Type: gedcom.EventMarriage, IsNegative: true}
	div := &gedcom.Event{Type: gedcom.EventDivorce}
	fam := &gedcom.Family{Events: []*gedcom.Event{nil, noMarr, div}}

	if got, want := fam.OccurredEvents(), []*gedcom.Event{div}; !reflect.DeepEqual(got, want) {
		t.Errorf("OccurredEvents() = %v, want %v", eventTypes(got), eventTypes(want))
	}
	if got, want := fam.NegativeAssertions(), []*gedcom.Event{noMarr}; !reflect.DeepEqual(got, want) {
		t.Errorf("NegativeAssertions() = %v, want %v", eventTypes(got), eventTypes(want))
	}

	var nilFam *gedcom.Family
	if got := nilFam.OccurredEvents(); got != nil {
		t.Errorf("nil receiver OccurredEvents() = %v, want nil", got)
	}
	if got := nilFam.NegativeAssertions(); got != nil {
		t.Errorf("nil receiver NegativeAssertions() = %v, want nil", got)
	}
	onlyNegated := &gedcom.Family{Events: []*gedcom.Event{noMarr}}
	if got := onlyNegated.OccurredEvents(); got != nil {
		t.Errorf("OccurredEvents() with only NO events = %v, want nil", got)
	}
}

// TestBirthDeathAccessors_SkipNegativeAssertions pins that a NO BIRT / NO DEAT
// is never returned as the birth or death: it records that the event did NOT
// happen, so returning it (or its DATE period) would report the opposite.
func TestBirthDeathAccessors_SkipNegativeAssertions(t *testing.T) {
	doc := decodeNegativeAssertion70(t)

	john := doc.GetIndividual("@I1@")
	if john == nil {
		t.Fatal("@I1@ not decoded")
	}
	if got := john.DeathEvent(); got != nil {
		t.Errorf("DeathEvent() = %+v, want nil for an individual with only NO DEAT", got)
	}
	if got := john.DeathDate(); got != nil {
		t.Errorf("DeathDate() = %q, want nil for an individual with only NO DEAT", got.Original)
	}
	if got := john.BirthDate(); got == nil || got.Year != 1920 {
		t.Errorf("BirthDate() = %v, want the 1920 birth", got)
	}

	// NO BIRT precedes the real BIRT in file order: the accessor must look past it.
	jane := doc.GetIndividual("@I2@")
	if jane == nil {
		t.Fatal("@I2@ not decoded")
	}
	birth := jane.BirthEvent()
	if birth == nil || birth.IsNegative {
		t.Fatalf("BirthEvent() = %+v, want the positive BIRT", birth)
	}
	if got := jane.BirthDate(); got == nil || got.Year != 1921 {
		t.Errorf("BirthDate() = %v, want the 1921 birth, not the NO BIRT period", got)
	}
}

func TestNegativeAssertions_DecodedSnippet(t *testing.T) {
	doc := decodeNegativeAssertion70(t)

	john := doc.GetIndividual("@I1@")
	if got, want := eventTypes(john.NegativeAssertions()), []gedcom.EventType{gedcom.EventDeath}; !reflect.DeepEqual(got, want) {
		t.Errorf("@I1@ NegativeAssertions() = %v, want %v", got, want)
	}
	if got, want := eventTypes(john.OccurredEvents()), []gedcom.EventType{gedcom.EventBirth}; !reflect.DeepEqual(got, want) {
		t.Errorf("@I1@ OccurredEvents() = %v, want %v", got, want)
	}

	fam := doc.GetFamily("@F1@")
	if got, want := eventTypes(fam.NegativeAssertions()), []gedcom.EventType{gedcom.EventMarriage}; !reflect.DeepEqual(got, want) {
		t.Errorf("@F1@ NegativeAssertions() = %v, want %v", got, want)
	}
	if got := fam.OccurredEvents(); got != nil {
		t.Errorf("@F1@ OccurredEvents() = %v, want nil", eventTypes(got))
	}
}

func TestNegativeAssertions_Maximal70(t *testing.T) {
	doc := decodeMaximal70(t)

	ind := doc.GetIndividual("@I1@")
	if ind == nil {
		t.Fatal("@I1@ not decoded")
	}
	if got, want := eventTypes(ind.NegativeAssertions()), []gedcom.EventType{gedcom.EventNaturalization, gedcom.EventEmigration}; !reflect.DeepEqual(got, want) {
		t.Errorf("@I1@ NegativeAssertions() = %v, want %v", got, want)
	}
	occurred := ind.OccurredEvents()
	if len(occurred)+len(ind.NegativeAssertions()) != len(ind.Events) {
		t.Errorf("OccurredEvents (%d) + NegativeAssertions (%d) != Events (%d)",
			len(occurred), len(ind.NegativeAssertions()), len(ind.Events))
	}
	for _, e := range occurred {
		if e.IsNegative {
			t.Errorf("OccurredEvents() includes negated %s", e.Type)
		}
	}
	// maximal70 also records a positive EMIG and NATU; those must remain.
	var sawEmig bool
	for _, e := range occurred {
		if e.Type == gedcom.EventEmigration {
			sawEmig = true
		}
	}
	if !sawEmig {
		t.Error("OccurredEvents() lost the positive EMIG")
	}

	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("@F1@ not decoded")
	}
	if got, want := eventTypes(fam.NegativeAssertions()), []gedcom.EventType{gedcom.EventDivorce, gedcom.EventAnnulment}; !reflect.DeepEqual(got, want) {
		t.Errorf("@F1@ NegativeAssertions() = %v, want %v", got, want)
	}
	for _, e := range fam.OccurredEvents() {
		if e.IsNegative {
			t.Errorf("@F1@ OccurredEvents() includes negated %s", e.Type)
		}
	}
}
