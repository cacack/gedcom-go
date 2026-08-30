package decoder

// File-backed regression tests for the EVENT_DETAIL cycle (issues #447, #448,
// #402). These live in their own file rather than in entity_test.go because
// that file uses `gedcom` as a local variable name in 60-odd tests, so it
// cannot import the gedcom package without shadowing it.

import (
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// TestEventDetailFixtures is the file-backed half of issues #447, #448 and
// #402. Each fixture is a conformant file that produced UNKNOWN_TAG before this
// cycle. Each check asserts the values reach the typed model, because "no
// diagnostic" on its own would also be satisfied by silently ignoring the
// lines -- which is precisely the defect #448 describes.
func TestEventDetailFixtures(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		issue string
		check func(t *testing.T, doc *gedcom.Document)
	}{
		{
			name:  "event carries ASSO, RELI and SNOTE",
			path:  "../testdata/gedcom-7.0/event-detail-asso-reli-snote.ged",
			issue: "#447",
			check: checkEventDetailFixture,
		},
		{
			name:  "family carries CENS, FACT, NCHI and RESI with EVENT_DETAIL",
			path:  "../testdata/gedcom-7.0/family-event-detail.ged",
			issue: "#448",
			check: checkFamilyDetailFixture,
		},
		{
			name:  "attributes carry AGNC, CAUS and OBJE",
			path:  "../testdata/gedcom-5.5.1/attribute-event-detail.ged",
			issue: "#402",
			check: checkAttributeDetailFixture,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeFixture(t, tt.path)
			if got := unknownTagsIn(result.Diagnostics); len(got) != 0 {
				t.Errorf("%s: UNKNOWN_TAG diagnostics = %v, want none (%s)", tt.path, got, tt.issue)
			}
			tt.check(t, result.Document)
		})
	}
}

// checkEventDetailFixture asserts the three EVENT_DETAIL substructures #447
// names reach the typed model, plus the CHAN shared note that issue also lists.
func checkEventDetailFixture(t *testing.T, doc *gedcom.Document) {
	t.Helper()
	indi := doc.GetIndividual("@I1@")
	if indi == nil || len(indi.Events) != 1 {
		t.Fatalf("GetIndividual(@I1@) = %v, want one event", indi)
	}
	event := indi.Events[0]

	if event.Type != gedcom.EventBaptism {
		t.Errorf("Events[0].Type = %q, want %q", event.Type, gedcom.EventBaptism)
	}
	if got := event.ReligiousAffiliation; got != "Baptist" {
		t.Errorf("ReligiousAffiliation = %q, want %q", got, "Baptist")
	}
	if len(event.Associations) != 1 {
		t.Fatalf("len(Associations) = %d, want 1", len(event.Associations))
	}
	if got := event.Associations[0].IndividualXRef; got != "@I2@" {
		t.Errorf("Associations[0].IndividualXRef = %q, want %q", got, "@I2@")
	}
	if got := event.Associations[0].Role; got != "CLERGY" {
		t.Errorf("Associations[0].Role = %q, want %q", got, "CLERGY")
	}
	if want := []string{"@N1@"}; !reflect.DeepEqual(event.NoteXRefs, want) {
		t.Errorf("NoteXRefs = %v, want %v", event.NoteXRefs, want)
	}
	if want := []string{"Recorded in the parish register"}; !reflect.DeepEqual(event.InlineNotes, want) {
		t.Errorf("InlineNotes = %v, want %v", event.InlineNotes, want)
	}

	// The pointer resolves to the SNOTE record's text: a dangling XRef would
	// satisfy every assertion above and still be useless to a caller.
	wantNotes := []string{"Recorded in the parish register", "Enumerator remarks"}
	if got := event.AllNotes(doc); !reflect.DeepEqual(got, wantNotes) {
		t.Errorf("AllNotes() = %v, want %v", got, wantNotes)
	}

	if indi.ChangeDate == nil {
		t.Fatal("ChangeDate = nil, want the CHAN structure")
	}
	if want := []string{"@N1@"}; !reflect.DeepEqual(indi.ChangeDate.NoteXRefs, want) {
		t.Errorf("ChangeDate.NoteXRefs = %v, want %v", indi.ChangeDate.NoteXRefs, want)
	}
}

// checkFamilyDetailFixture asserts the four FAM contexts of #448 reach the
// typed model along with the EVENT_DETAIL hanging off each of them.
func checkFamilyDetailFixture(t *testing.T, doc *gedcom.Document) {
	t.Helper()
	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("GetFamily(@F1@) returned nil")
	}

	cens := findFamilyEvent(fam, gedcom.EventCensus)
	if cens == nil {
		t.Fatal("FAM.CENS did not reach Family.Events")
	}
	if cens.Date != "1900" || cens.Place != "Springfield, Illinois" {
		t.Errorf("CENS date/place = %q/%q, want \"1900\"/\"Springfield, Illinois\"", cens.Date, cens.Place)
	}
	if cens.Agency != "Bureau of the Census" {
		t.Errorf("CENS Agency = %q, want %q", cens.Agency, "Bureau of the Census")
	}
	if want := []string{"Household of six"}; !reflect.DeepEqual(cens.InlineNotes, want) {
		t.Errorf("CENS InlineNotes = %v, want %v", cens.InlineNotes, want)
	}
	if want := []string{"@N1@"}; !reflect.DeepEqual(cens.NoteXRefs, want) {
		t.Errorf("CENS NoteXRefs = %v, want %v", cens.NoteXRefs, want)
	}
	if len(cens.SourceCitations) != 1 || cens.SourceCitations[0].Page != "Sheet 4B" {
		t.Errorf("CENS SourceCitations = %+v, want one citation with page \"Sheet 4B\"", cens.SourceCitations)
	}

	resi := findFamilyEvent(fam, gedcom.EventResidence)
	if resi == nil {
		t.Fatal("FAM.RESI did not reach Family.Events")
	}
	if resi.Date != "FROM 1901 TO 1910" || resi.Place != "Chicago, Illinois" {
		t.Errorf("RESI date/place = %q/%q, want \"FROM 1901 TO 1910\"/\"Chicago, Illinois\"", resi.Date, resi.Place)
	}
	if resi.Address == nil || resi.Address.City != "Chicago" {
		t.Fatalf("RESI Address = %+v, want City \"Chicago\"", resi.Address)
	}
	if want := []string{"+1 555 0100"}; !reflect.DeepEqual(resi.Phone, want) {
		t.Errorf("RESI Phone = %v, want %v", resi.Phone, want)
	}

	nchi := findFamilyAttribute(fam, "NCHI")
	if nchi == nil {
		t.Fatal("FAM.NCHI did not reach Family.Attributes")
	}
	if nchi.Value != "3" || nchi.Date != "1910" || nchi.Cause != "Census enumeration" {
		t.Errorf("NCHI = %+v, want value 3, date 1910, cause \"Census enumeration\"", nchi)
	}
	if len(nchi.Associations) != 1 || nchi.Associations[0].IndividualXRef != "@I1@" {
		t.Errorf("NCHI Associations = %+v, want one link to @I1@", nchi.Associations)
	}
	// Family.Attributes is the single store for NCHI; the accessor reads it.
	if fam.NumberOfChildren() != "3" {
		t.Errorf("NumberOfChildren() = %q, want %q", fam.NumberOfChildren(), "3")
	}

	fact := findFamilyAttribute(fam, "FACT")
	if fact == nil {
		t.Fatal("FAM.FACT did not reach Family.Attributes")
	}
	if fact.Value != "Homesteaders" || fact.TypeDetail != "Lifestyle" {
		t.Errorf("FACT = %+v, want value \"Homesteaders\", TypeDetail \"Lifestyle\"", fact)
	}
	if fact.ReligiousAffiliation != "Methodist" || fact.Restriction != "CONFIDENTIAL" {
		t.Errorf("FACT RELI/RESN = %q/%q, want \"Methodist\"/\"CONFIDENTIAL\"", fact.ReligiousAffiliation, fact.Restriction)
	}
	if len(fact.Media) != 1 || fact.Media[0].MediaXRef != "@M1@" {
		t.Errorf("FACT Media = %+v, want one link to @M1@", fact.Media)
	}
}

// checkAttributeDetailFixture asserts #402's own example -- a FACT with AGNC,
// CAUS and OBJE -- reaches the typed model, and that the same holds for an
// OCCU, since #402 is about every attribute tag rather than FACT alone.
func checkAttributeDetailFixture(t *testing.T, doc *gedcom.Document) {
	t.Helper()
	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}

	want := []struct {
		attrType   string
		value      string
		typeDetail string
		agency     string
		cause      string
	}{
		{"OCCU", "Blacksmith", "Trade", "Guild of Smiths", "Apprenticeship"},
		{"FACT", "Purple Heart", "Award", "US Army", "Wounded in action"},
	}
	for _, w := range want {
		attr := findAttribute(indi.Attributes, w.attrType)
		if attr == nil {
			t.Errorf("%s did not reach Individual.Attributes", w.attrType)
			continue
		}
		if attr.Value != w.value || attr.TypeDetail != w.typeDetail {
			t.Errorf("%s value/TypeDetail = %q/%q, want %q/%q", w.attrType, attr.Value, attr.TypeDetail, w.value, w.typeDetail)
		}
		if attr.Agency != w.agency {
			t.Errorf("%s Agency = %q, want %q", w.attrType, attr.Agency, w.agency)
		}
		if attr.Cause != w.cause {
			t.Errorf("%s Cause = %q, want %q", w.attrType, attr.Cause, w.cause)
		}
		if len(attr.Media) != 1 || attr.Media[0].MediaXRef != "@M1@" {
			t.Errorf("%s Media = %+v, want one link to @M1@", w.attrType, attr.Media)
		}
	}
}

func findFamilyEvent(fam *gedcom.Family, typ gedcom.EventType) *gedcom.Event {
	for _, event := range fam.Events {
		if event.Type == typ {
			return event
		}
	}
	return nil
}

func findFamilyAttribute(fam *gedcom.Family, typ string) *gedcom.Attribute {
	return findAttribute(fam.Attributes, typ)
}

func findAttribute(attrs []*gedcom.Attribute, typ string) *gedcom.Attribute {
	for _, attr := range attrs {
		if attr.Type == typ {
			return attr
		}
	}
	return nil
}
