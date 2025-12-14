package decoder

import (
	"os"
	"strings"
	"testing"
)

const entityTestGedcom = `0 HEAD
1 SOUR Test
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
1 SEX M
1 BIRT
2 DATE 15 JAN 1850
2 PLAC Springfield, IL
1 DEAT
2 DATE 20 MAR 1920
1 FAMC @F1@
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 MARR
2 DATE 10 JUN 1875
2 PLAC Chicago, IL
0 @I3@ INDI
1 NAME Child /Doe/
0 @S1@ SOUR
1 TITL Test Source
1 AUTH John Author
1 PUBL Test Publisher
0 TRLR
`

func TestPopulateEntities(t *testing.T) {
	doc, err := Decode(strings.NewReader(entityTestGedcom))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Test individuals
	individuals := doc.Individuals()
	if len(individuals) != 3 {
		t.Errorf("Individuals() = %d, want 3", len(individuals))
	}

	// Find John Doe
	john := doc.GetIndividual("@I1@")
	if john == nil {
		t.Fatal("GetIndividual(@I1@) returned nil")
	}
	if john.XRef != "@I1@" {
		t.Errorf("john.XRef = %s, want @I1@", john.XRef)
	}
	if len(john.Names) != 1 {
		t.Fatalf("len(john.Names) = %d, want 1", len(john.Names))
	}
	if john.Names[0].Given != "John" {
		t.Errorf("john.Names[0].Given = %s, want John", john.Names[0].Given)
	}
	if john.Names[0].Surname != "Doe" {
		t.Errorf("john.Names[0].Surname = %s, want Doe", john.Names[0].Surname)
	}
	if john.Sex != "M" {
		t.Errorf("john.Sex = %s, want M", john.Sex)
	}

	// Check events
	if len(john.Events) != 2 {
		t.Errorf("len(john.Events) = %d, want 2", len(john.Events))
	}

	// Check birth event
	var birthEvent, deathEvent *struct {
		Type  string
		Date  string
		Place string
	}
	for _, ev := range john.Events {
		if ev.Type == "BIRT" {
			birthEvent = &struct {
				Type  string
				Date  string
				Place string
			}{string(ev.Type), ev.Date, ev.Place}
		}
		if ev.Type == "DEAT" {
			deathEvent = &struct {
				Type  string
				Date  string
				Place string
			}{string(ev.Type), ev.Date, ev.Place}
		}
	}
	if birthEvent == nil {
		t.Error("Birth event not found")
	} else {
		if birthEvent.Date != "15 JAN 1850" {
			t.Errorf("birthEvent.Date = %s, want 15 JAN 1850", birthEvent.Date)
		}
		if birthEvent.Place != "Springfield, IL" {
			t.Errorf("birthEvent.Place = %s, want Springfield, IL", birthEvent.Place)
		}
	}
	if deathEvent == nil {
		t.Error("Death event not found")
	}

	// Check family references
	if len(john.ChildInFamilies) != 1 || john.ChildInFamilies[0].FamilyXRef != "@F1@" {
		t.Errorf("john.ChildInFamilies[0].FamilyXRef = %v, want @F1@", john.ChildInFamilies)
	}
	if john.ChildInFamilies[0].Pedigree != "" {
		t.Errorf("john.ChildInFamilies[0].Pedigree = %s, want empty", john.ChildInFamilies[0].Pedigree)
	}

	jane := doc.GetIndividual("@I2@")
	if jane == nil {
		t.Fatal("GetIndividual(@I2@) returned nil")
	}
	if len(jane.SpouseInFamilies) != 1 || jane.SpouseInFamilies[0] != "@F1@" {
		t.Errorf("jane.SpouseInFamilies = %v, want [@F1@]", jane.SpouseInFamilies)
	}

	// Test families
	families := doc.Families()
	if len(families) != 1 {
		t.Errorf("Families() = %d, want 1", len(families))
	}

	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("GetFamily(@F1@) returned nil")
	}
	if fam.Husband != "@I1@" {
		t.Errorf("fam.Husband = %s, want @I1@", fam.Husband)
	}
	if fam.Wife != "@I2@" {
		t.Errorf("fam.Wife = %s, want @I2@", fam.Wife)
	}
	if len(fam.Children) != 1 || fam.Children[0] != "@I3@" {
		t.Errorf("fam.Children = %v, want [@I3@]", fam.Children)
	}

	// Check marriage event
	if len(fam.Events) != 1 {
		t.Errorf("len(fam.Events) = %d, want 1", len(fam.Events))
	} else {
		if fam.Events[0].Type != "MARR" {
			t.Errorf("fam.Events[0].Type = %s, want MARR", fam.Events[0].Type)
		}
		if fam.Events[0].Date != "10 JUN 1875" {
			t.Errorf("fam.Events[0].Date = %s, want 10 JUN 1875", fam.Events[0].Date)
		}
		if fam.Events[0].Place != "Chicago, IL" {
			t.Errorf("fam.Events[0].Place = %s, want Chicago, IL", fam.Events[0].Place)
		}
	}

	// Test sources
	sources := doc.Sources()
	if len(sources) != 1 {
		t.Errorf("Sources() = %d, want 1", len(sources))
	}

	src := doc.GetSource("@S1@")
	if src == nil {
		t.Fatal("GetSource(@S1@) returned nil")
	}
	if src.Title != "Test Source" {
		t.Errorf("src.Title = %s, want Test Source", src.Title)
	}
	if src.Author != "John Author" {
		t.Errorf("src.Author = %s, want John Author", src.Author)
	}
	if src.Publication != "Test Publisher" {
		t.Errorf("src.Publication = %s, want Test Publisher", src.Publication)
	}
}

func TestParsePersonalNameFromFull(t *testing.T) {
	// Test name parsing from full name only (no subordinate tags)
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME First Middle /Surname/
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}
	if indi.Names[0].Given != "First Middle" {
		t.Errorf("Given = %s, want First Middle", indi.Names[0].Given)
	}
	if indi.Names[0].Surname != "Surname" {
		t.Errorf("Surname = %s, want Surname", indi.Names[0].Surname)
	}
}

func TestParseNoSurname(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Unknown
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}
	if indi.Names[0].Given != "Unknown" {
		t.Errorf("Given = %s, want Unknown", indi.Names[0].Given)
	}
	if indi.Names[0].Surname != "" {
		t.Errorf("Surname = %s, want empty", indi.Names[0].Surname)
	}
}

func TestParsePedigreeLinks(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Child /One/
1 FAMC @F1@
2 PEDI birth
1 FAMC @F2@
2 PEDI adopted
1 FAMC @F3@
2 PEDI foster
1 FAMC @F4@
2 PEDI sealing
1 FAMC @F5@
0 @I2@ INDI
1 NAME Child /Two/
1 FAMC @F6@
2 PEDI BIRTH
1 FAMC @F7@
2 PEDI ADOPTED
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Test GEDCOM 5.5.1 lowercase values
	child1 := doc.GetIndividual("@I1@")
	if child1 == nil {
		t.Fatal("Individual @I1@ not found")
	}

	if len(child1.ChildInFamilies) != 5 {
		t.Fatalf("len(child1.ChildInFamilies) = %d, want 5", len(child1.ChildInFamilies))
	}

	tests := []struct {
		idx      int
		familyXR string
		pedigree string
	}{
		{0, "@F1@", "birth"},
		{1, "@F2@", "adopted"},
		{2, "@F3@", "foster"},
		{3, "@F4@", "sealing"},
		{4, "@F5@", ""}, // No PEDI tag
	}

	for _, tt := range tests {
		link := child1.ChildInFamilies[tt.idx]
		if link.FamilyXRef != tt.familyXR {
			t.Errorf("ChildInFamilies[%d].FamilyXRef = %s, want %s", tt.idx, link.FamilyXRef, tt.familyXR)
		}
		if link.Pedigree != tt.pedigree {
			t.Errorf("ChildInFamilies[%d].Pedigree = %s, want %s", tt.idx, link.Pedigree, tt.pedigree)
		}
	}

	// Test GEDCOM 7.0 uppercase values (preserving original casing)
	child2 := doc.GetIndividual("@I2@")
	if child2 == nil {
		t.Fatal("Individual @I2@ not found")
	}

	if len(child2.ChildInFamilies) != 2 {
		t.Fatalf("len(child2.ChildInFamilies) = %d, want 2", len(child2.ChildInFamilies))
	}

	if child2.ChildInFamilies[0].FamilyXRef != "@F6@" {
		t.Errorf("ChildInFamilies[0].FamilyXRef = %s, want @F6@", child2.ChildInFamilies[0].FamilyXRef)
	}
	if child2.ChildInFamilies[0].Pedigree != "BIRTH" {
		t.Errorf("ChildInFamilies[0].Pedigree = %s, want BIRTH (uppercase preserved)", child2.ChildInFamilies[0].Pedigree)
	}

	if child2.ChildInFamilies[1].FamilyXRef != "@F7@" {
		t.Errorf("ChildInFamilies[1].FamilyXRef = %s, want @F7@", child2.ChildInFamilies[1].FamilyXRef)
	}
	if child2.ChildInFamilies[1].Pedigree != "ADOPTED" {
		t.Errorf("ChildInFamilies[1].Pedigree = %s, want ADOPTED (uppercase preserved)", child2.ChildInFamilies[1].Pedigree)
	}
}

// === Feature Gap Tests ===
// These tests demonstrate missing GEDCOM features identified in docs/FEATURE-GAPS.md
// They are skipped until implementation is complete.

// TestEventSubordinateTags tests parsing of event subordinate tags.
// Tests Event struct support for TYPE, CAUS, AGE, AGNC subordinates.
// Priority: P1 (Critical)
// Ref: FEATURE-GAPS.md Section 3.1
func TestEventSubordinateTags(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 DEAT Y
2 DATE 20 MAR 1920
2 PLAC Springfield, IL
2 TYPE Natural death
2 CAUS Heart failure
2 AGE 70y
2 AGNC County Coroner
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(indi.Events))
	}

	death := indi.Events[0]
	if death.EventTypeDetail != "Natural death" {
		t.Errorf("Event.EventTypeDetail = %s, want 'Natural death'", death.EventTypeDetail)
	}
	if death.Cause != "Heart failure" {
		t.Errorf("Event.Cause = %s, want 'Heart failure'", death.Cause)
	}
	if death.Age != "70y" {
		t.Errorf("Event.Age = %s, want '70y'", death.Age)
	}
	if death.Agency != "County Coroner" {
		t.Errorf("Event.Agency = %s, want 'County Coroner'", death.Agency)
	}
}

// TestSourceCitationStructure tests parsing of source citation details.
// Validates support for PAGE, QUAY, and DATA subordinates in source citations.
func TestSourceCitationStructure(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 15 JAN 1850
2 SOUR @S1@
3 PAGE Page 42, Entry 103
3 QUAY 2
3 DATA
4 DATE 15 JAN 1850
4 TEXT Birth record shows...
0 @S1@ SOUR
1 TITL County Birth Records
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(indi.Events))
	}

	birth := indi.Events[0]
	if len(birth.SourceCitations) != 1 {
		t.Fatalf("len(SourceCitations) = %d, want 1", len(birth.SourceCitations))
	}
	cite := birth.SourceCitations[0]
	if cite.SourceXRef != "@S1@" {
		t.Errorf("SourceXRef = %s, want @S1@", cite.SourceXRef)
	}
	if cite.Page != "Page 42, Entry 103" {
		t.Errorf("Page = %s, want 'Page 42, Entry 103'", cite.Page)
	}
	if cite.Quality != 2 {
		t.Errorf("Quality = %d, want 2", cite.Quality)
	}
	if cite.Data == nil {
		t.Fatal("Data is nil, want non-nil")
	}
	if cite.Data.Date != "15 JAN 1850" {
		t.Errorf("Data.Date = %s, want '15 JAN 1850'", cite.Data.Date)
	}
	if cite.Data.Text != "Birth record shows..." {
		t.Errorf("Data.Text = %s, want 'Birth record shows...'", cite.Data.Text)
	}
}

// TestIndividualAttributes tests parsing of individual attributes.
// Tests parsing of CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL, RELI attributes.
// Priority: P2 (Important)
// Ref: FEATURE-GAPS.md Section 2
func TestIndividualAttributes(t *testing.T) {

	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 CAST Brahmin
1 DSCR Tall, brown hair
1 EDUC PhD Computer Science
2 DATE 1972
2 PLAC MIT, Cambridge, MA
1 IDNO 12345678
1 NATI American
1 SSN 123-45-6789
1 TITL Dr.
1 RELI Methodist
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	// Attributes should be parsed, but currently only OCCU is handled
	// Expected: len(Attributes) = 8
	// Actual: len(Attributes) = 0 (none of these are parsed)
	expectedAttrs := map[string]string{
		"CAST": "Brahmin",
		"DSCR": "Tall, brown hair",
		"EDUC": "PhD Computer Science",
		"IDNO": "12345678",
		"NATI": "American",
		"SSN":  "123-45-6789",
		"TITL": "Dr.",
		"RELI": "Methodist",
	}

	if len(indi.Attributes) != len(expectedAttrs) {
		t.Errorf("len(Attributes) = %d, want %d", len(indi.Attributes), len(expectedAttrs))
	}

	// Check each attribute type and value
	attrMap := make(map[string]string)
	for _, attr := range indi.Attributes {
		attrMap[attr.Type] = attr.Value
	}

	for attrType, expectedValue := range expectedAttrs {
		if value, found := attrMap[attrType]; !found {
			t.Errorf("Attribute %s not found", attrType)
		} else if value != expectedValue {
			t.Errorf("Attribute %s = %s, want %s", attrType, value, expectedValue)
		}
	}

	// Test EDUC with DATE, PLAC subordinates
	for _, attr := range indi.Attributes {
		if attr.Type == "EDUC" {
			if attr.Date != "1972" {
				t.Errorf("EDUC Date = %s, want 1972", attr.Date)
			}
			if attr.Place != "MIT, Cambridge, MA" {
				t.Errorf("EDUC Place = %s, want 'MIT, Cambridge, MA'", attr.Place)
			}
		}
	}
}

// TestAttributeSubordinates tests parsing of attribute subordinate tags.
// Validates that attributes can have DATE, PLAC, SOUR, NOTE subordinates.
func TestAttributeSubordinates(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 OCCU Software Engineer
2 DATE 2000
2 PLAC Silicon Valley, CA
2 SOUR @S1@
3 PAGE Employment Records
0 @S1@ SOUR
1 TITL Company HR Records
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Attributes) != 1 {
		t.Fatalf("len(Attributes) = %d, want 1", len(indi.Attributes))
	}

	occu := indi.Attributes[0]
	if occu.Type != "OCCU" {
		t.Errorf("Attribute.Type = %s, want OCCU", occu.Type)
	}
	if occu.Value != "Software Engineer" {
		t.Errorf("Attribute.Value = %s, want 'Software Engineer'", occu.Value)
	}
	if occu.Date != "2000" {
		t.Errorf("Attribute.Date = %s, want 2000", occu.Date)
	}
	if occu.Place != "Silicon Valley, CA" {
		t.Errorf("Attribute.Place = %s, want 'Silicon Valley, CA'", occu.Place)
	}
	if len(occu.SourceCitations) != 1 {
		t.Fatalf("len(SourceCitations) = %d, want 1", len(occu.SourceCitations))
	}
	if occu.SourceCitations[0].Page != "Employment Records" {
		t.Errorf("SourceCitation.Page = %s, want 'Employment Records'", occu.SourceCitations[0].Page)
	}
}

// TestReligiousEvents tests parsing of religious event types.
// Validates support for BARM, BASM, BLES, CONF, FCOM, CHRA event types.
// Priority: P2 (Important)
// Ref: FEATURE-GAPS.md Section 1.1
func TestReligiousEvents(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Jacob /Cohen/
1 BARM
2 DATE 15 MAR 1963
2 PLAC Temple Beth Israel
0 @I2@ INDI
1 NAME Sarah /Cohen/
1 BASM
2 DATE 20 APR 1964
2 PLAC Temple Beth Israel
0 @I3@ INDI
1 NAME John /Smith/
1 BLES
2 DATE 1 JAN 1950
1 CONF
2 DATE 1 JUN 1962
1 FCOM
2 DATE 1 MAY 1961
1 CHRA
2 DATE 15 AUG 1975
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Test BARM (Bar Mitzvah)
	jacob := doc.GetIndividual("@I1@")
	if jacob == nil {
		t.Fatal("Individual @I1@ not found")
	}
	if len(jacob.Events) != 1 {
		t.Errorf("@I1@ len(Events) = %d, want 1", len(jacob.Events))
	} else if string(jacob.Events[0].Type) != "BARM" {
		t.Errorf("@I1@ Event.Type = %s, want BARM", jacob.Events[0].Type)
	}

	// Test BASM (Bas Mitzvah)
	sarah := doc.GetIndividual("@I2@")
	if sarah == nil {
		t.Fatal("Individual @I2@ not found")
	}
	if len(sarah.Events) != 1 {
		t.Errorf("@I2@ len(Events) = %d, want 1", len(sarah.Events))
	} else if string(sarah.Events[0].Type) != "BASM" {
		t.Errorf("@I2@ Event.Type = %s, want BASM", sarah.Events[0].Type)
	}

	// Test BLES, CONF, FCOM, CHRA
	john := doc.GetIndividual("@I3@")
	if john == nil {
		t.Fatal("Individual @I3@ not found")
	}
	if len(john.Events) != 4 {
		t.Errorf("@I3@ len(Events) = %d, want 4", len(john.Events))
	} else {
		expectedTypes := []string{"BLES", "CONF", "FCOM", "CHRA"}
		for i, expected := range expectedTypes {
			if string(john.Events[i].Type) != expected {
				t.Errorf("@I3@ Event[%d].Type = %s, want %s", i, john.Events[i].Type, expected)
			}
		}
	}
}

// TestLifeEvents tests parsing of life status events.
// Validates support for GRAD, RETI, NATU, ORDN, PROB, WILL, CREM event types.
// Priority: P2 (Important)
// Ref: FEATURE-GAPS.md Sections 1.2, 1.3, 1.4, 1.5
func TestLifeEvents(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 GRAD
2 DATE 1972
2 PLAC MIT, Cambridge, MA
1 NATU
2 DATE 4 JUL 1980
2 PLAC Boston, MA
1 ORDN
2 DATE 15 JUN 1985
1 RETI
2 DATE 1 JAN 2015
1 WILL
2 DATE 5 MAR 2018
1 PROB
2 DATE 20 JUN 2020
1 CREM
2 DATE 25 JUN 2020
2 PLAC Forest Lawn Cemetery
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	expectedEvents := []string{"GRAD", "NATU", "ORDN", "RETI", "WILL", "PROB", "CREM"}
	if len(indi.Events) != len(expectedEvents) {
		t.Errorf("len(Events) = %d, want %d", len(indi.Events), len(expectedEvents))
	}

	for i, expected := range expectedEvents {
		if i >= len(indi.Events) {
			t.Errorf("Event[%d] missing, want %s", i, expected)
			continue
		}
		if string(indi.Events[i].Type) != expected {
			t.Errorf("Event[%d].Type = %s, want %s", i, indi.Events[i].Type, expected)
		}
	}
}

// TestLDSOrdinances tests parsing of LDS ordinance structures.
// Priority: P2 (Critical for LDS users)
// Ref: FEATURE-GAPS.md Section 5
func TestLDSOrdinances(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BAPL
2 DATE 1 JAN 1950
2 TEMP SLAKE
2 STAT COMPLETED
3 DATE 1 JAN 1950
1 CONL
2 DATE 1 FEB 1950
2 TEMP SLAKE
2 STAT COMPLETED
3 DATE 1 FEB 1950
1 ENDL
2 DATE 1 MAR 1970
2 TEMP LOGAN
1 SLGC
2 DATE 15 APR 1951
2 TEMP SLAKE
2 FAMC @F1@
0 @F1@ FAM
1 SLGS
2 DATE 10 JUN 1949
2 TEMP SLAKE
2 STAT COMPLETED
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	// LDS ordinances should be in a separate field, not Events
	if len(indi.LDSOrdinances) != 4 {
		t.Errorf("len(LDSOrdinances) = %d, want 4", len(indi.LDSOrdinances))
	}

	expectedOrds := []struct {
		ordType string
		date    string
		temple  string
		status  string
		famc    string
	}{
		{"BAPL", "1 JAN 1950", "SLAKE", "COMPLETED", ""},
		{"CONL", "1 FEB 1950", "SLAKE", "COMPLETED", ""},
		{"ENDL", "1 MAR 1970", "LOGAN", "", ""},
		{"SLGC", "15 APR 1951", "SLAKE", "", "@F1@"},
	}

	for i, expected := range expectedOrds {
		if i >= len(indi.LDSOrdinances) {
			t.Errorf("LDSOrdinance[%d] missing", i)
			continue
		}
		ord := indi.LDSOrdinances[i]
		if string(ord.Type) != expected.ordType {
			t.Errorf("LDSOrdinance[%d].Type = %s, want %s", i, ord.Type, expected.ordType)
		}
		if ord.Date != expected.date {
			t.Errorf("LDSOrdinance[%d].Date = %s, want %s", i, ord.Date, expected.date)
		}
		if ord.Temple != expected.temple {
			t.Errorf("LDSOrdinance[%d].Temple = %s, want %s", i, ord.Temple, expected.temple)
		}
		if ord.Status != expected.status {
			t.Errorf("LDSOrdinance[%d].Status = %s, want %s", i, ord.Status, expected.status)
		}
		if ord.FamilyXRef != expected.famc {
			t.Errorf("LDSOrdinance[%d].FamilyXRef = %s, want %s", i, ord.FamilyXRef, expected.famc)
		}
	}

	// Family ordinance
	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("Family not found")
	}

	if len(fam.LDSOrdinances) != 1 {
		t.Errorf("len(fam.LDSOrdinances) = %d, want 1", len(fam.LDSOrdinances))
	} else {
		slgs := fam.LDSOrdinances[0]
		if slgs.Type != "SLGS" {
			t.Errorf("fam.LDSOrdinances[0].Type = %s, want SLGS", slgs.Type)
		}
		if slgs.Date != "10 JUN 1949" {
			t.Errorf("fam.LDSOrdinances[0].Date = %s, want '10 JUN 1949'", slgs.Date)
		}
		if slgs.Temple != "SLAKE" {
			t.Errorf("fam.LDSOrdinances[0].Temple = %s, want SLAKE", slgs.Temple)
		}
		if slgs.Status != "COMPLETED" {
			t.Errorf("fam.LDSOrdinances[0].Status = %s, want COMPLETED", slgs.Status)
		}
	}
}

// TestNameExtensions tests parsing of extended name components.
// Validates support for NICK (nickname) and SPFX (surname prefix).
// Priority: P2 (Important for international genealogy)
// Ref: FEATURE-GAPS.md Section 7.1
func TestNameExtensions(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Johannes Ludwig /von Beethoven/
2 GIVN Johannes Ludwig
2 SURN Beethoven
2 SPFX von
2 NICK Ludwig
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Names) != 1 {
		t.Fatalf("len(Names) = %d, want 1", len(indi.Names))
	}

	name := indi.Names[0]
	if name.SurnamePrefix != "von" {
		t.Errorf("name.SurnamePrefix = %s, want von", name.SurnamePrefix)
	}
	if name.Nickname != "Ludwig" {
		t.Errorf("name.Nickname = %s, want Ludwig", name.Nickname)
	}
}

// TestIndividualAssociations tests parsing of ASSO tag.
// Tests ASSO tag with ROLE/RELA subordinate.
// Priority: P2 (Important for relationship context)
// Ref: FEATURE-GAPS.md Section 8.1
func TestIndividualAssociations(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 ASSO @I2@
2 ROLE GODP
1 ASSO @I3@
2 ROLE WITN
0 @I2@ INDI
1 NAME Jane /Smith/
0 @I3@ INDI
1 NAME Bob /Johnson/
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	// Assert: indi.Associations has 2 entries
	if len(indi.Associations) != 2 {
		t.Fatalf("len(Associations) = %d, want 2", len(indi.Associations))
	}

	// Associations[0]: IndividualXRef=@I2@, Role=GODP (Godparent)
	if indi.Associations[0].IndividualXRef != "@I2@" {
		t.Errorf("Associations[0].IndividualXRef = %s, want @I2@", indi.Associations[0].IndividualXRef)
	}
	if indi.Associations[0].Role != "GODP" {
		t.Errorf("Associations[0].Role = %s, want GODP", indi.Associations[0].Role)
	}

	// Associations[1]: IndividualXRef=@I3@, Role=WITN (Witness)
	if indi.Associations[1].IndividualXRef != "@I3@" {
		t.Errorf("Associations[1].IndividualXRef = %s, want @I3@", indi.Associations[1].IndividualXRef)
	}
	if indi.Associations[1].Role != "WITN" {
		t.Errorf("Associations[1].Role = %s, want WITN", indi.Associations[1].Role)
	}
}

// TestPlaceStructure tests parsing of place structure with coordinates.
// Tests PLAC with FORM and MAP/LATI/LONG subordinates.
// Priority: P2 (Medium - Geographic coordinates enable mapping)
// Ref: Issue #11
func TestPlaceStructure(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 15 JAN 1850
2 PLAC Boston, Suffolk, Massachusetts, USA
3 FORM City, County, State, Country
3 MAP
4 LATI N42.3601
4 LONG W71.0589
1 DEAT
2 DATE 20 MAR 1920
2 PLAC Springfield, IL
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(indi.Events))
	}

	// Test birth event with full place structure
	birth := indi.Events[0]
	if birth.Type != "BIRT" {
		t.Errorf("birth.Type = %s, want BIRT", birth.Type)
	}

	// Backward compatibility: Event.Place should still be populated
	if birth.Place != "Boston, Suffolk, Massachusetts, USA" {
		t.Errorf("birth.Place = %s, want 'Boston, Suffolk, Massachusetts, USA'", birth.Place)
	}

	// Test PlaceDetail structure
	if birth.PlaceDetail == nil {
		t.Fatal("birth.PlaceDetail is nil, want non-nil")
	}
	if birth.PlaceDetail.Name != "Boston, Suffolk, Massachusetts, USA" {
		t.Errorf("birth.PlaceDetail.Name = %s, want 'Boston, Suffolk, Massachusetts, USA'", birth.PlaceDetail.Name)
	}
	if birth.PlaceDetail.Form != "City, County, State, Country" {
		t.Errorf("birth.PlaceDetail.Form = %s, want 'City, County, State, Country'", birth.PlaceDetail.Form)
	}

	// Test coordinates
	if birth.PlaceDetail.Coordinates == nil {
		t.Fatal("birth.PlaceDetail.Coordinates is nil, want non-nil")
	}
	if birth.PlaceDetail.Coordinates.Latitude != "N42.3601" {
		t.Errorf("Coordinates.Latitude = %s, want 'N42.3601'", birth.PlaceDetail.Coordinates.Latitude)
	}
	if birth.PlaceDetail.Coordinates.Longitude != "W71.0589" {
		t.Errorf("Coordinates.Longitude = %s, want 'W71.0589'", birth.PlaceDetail.Coordinates.Longitude)
	}

	// Test death event without coordinates (backward compatibility)
	death := indi.Events[1]
	if death.Type != "DEAT" {
		t.Errorf("death.Type = %s, want DEAT", death.Type)
	}
	if death.Place != "Springfield, IL" {
		t.Errorf("death.Place = %s, want 'Springfield, IL'", death.Place)
	}
	if death.PlaceDetail == nil {
		t.Fatal("death.PlaceDetail is nil, want non-nil")
	}
	if death.PlaceDetail.Name != "Springfield, IL" {
		t.Errorf("death.PlaceDetail.Name = %s, want 'Springfield, IL'", death.PlaceDetail.Name)
	}
	// Death place has no coordinates
	if death.PlaceDetail.Coordinates != nil {
		t.Errorf("death.PlaceDetail.Coordinates = %v, want nil (no coordinates)", death.PlaceDetail.Coordinates)
	}
}

// TestFamilyEvents tests parsing of family event types.
// Validates support for extended marriage-related legal events.
func TestFamilyEvents(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARB
2 DATE 1 JAN 1875
2 PLAC Boston, MA
1 MARC
2 DATE 5 JAN 1875
1 MARL
2 DATE 8 JAN 1875
1 MARS
2 DATE 10 JAN 1875
1 MARR
2 DATE 15 JAN 1875
2 PLAC Boston, MA
1 DIVF
2 DATE 1 JUN 1900
1 DIV
2 DATE 1 JUL 1900
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Smith/
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("Family not found")
	}

	// Should have 7 events: MARB, MARC, MARL, MARS, MARR, DIVF, DIV
	if len(fam.Events) != 7 {
		t.Errorf("len(fam.Events) = %d, want 7", len(fam.Events))
	}

	expectedEvents := []struct {
		eventType string
		date      string
		place     string
	}{
		{"MARB", "1 JAN 1875", "Boston, MA"},
		{"MARC", "5 JAN 1875", ""},
		{"MARL", "8 JAN 1875", ""},
		{"MARS", "10 JAN 1875", ""},
		{"MARR", "15 JAN 1875", "Boston, MA"},
		{"DIVF", "1 JUN 1900", ""},
		{"DIV", "1 JUL 1900", ""},
	}

	for i, expected := range expectedEvents {
		if i >= len(fam.Events) {
			t.Errorf("Event[%d] missing, want %s", i, expected.eventType)
			continue
		}
		if string(fam.Events[i].Type) != expected.eventType {
			t.Errorf("Event[%d].Type = %s, want %s", i, fam.Events[i].Type, expected.eventType)
		}
		if fam.Events[i].Date != expected.date {
			t.Errorf("Event[%d].Date = %s, want %s", i, fam.Events[i].Date, expected.date)
		}
		if fam.Events[i].Place != expected.place {
			t.Errorf("Event[%d].Place = %s, want %s", i, fam.Events[i].Place, expected.place)
		}
	}
}

// === Integration Tests ===
// These tests validate parsing against real-world GEDCOM 7.0 test data.

// TestMaximal70Individual tests parsing of maximal70.ged individual @I1@.
// Validates comprehensive GEDCOM 7.0 features from the official test file.
func TestMaximal70Individual(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual @I1@ not found")
	}

	// Test name with extensions (NICK, SPFX)
	if len(indi.Names) < 1 {
		t.Fatal("No names found")
	}
	name := indi.Names[0]
	if name.Nickname != "John" {
		t.Errorf("Name.Nickname = %s, want John", name.Nickname)
	}
	if name.SurnamePrefix != "de" {
		t.Errorf("Name.SurnamePrefix = %s, want de", name.SurnamePrefix)
	}
	if name.Given != "Joseph" {
		t.Errorf("Name.Given = %s, want Joseph", name.Given)
	}
	if name.Surname != "Allen" {
		t.Errorf("Name.Surname = %s, want Allen", name.Surname)
	}
	if name.Prefix != "Lt. Cmndr." {
		t.Errorf("Name.Prefix = %s, want 'Lt. Cmndr.'", name.Prefix)
	}
	if name.Suffix != "jr." {
		t.Errorf("Name.Suffix = %s, want jr.", name.Suffix)
	}

	// Test attributes (RESI is parsed as an event, not attribute)
	attrTypes := make(map[string]bool)
	for _, attr := range indi.Attributes {
		attrTypes[attr.Type] = true
	}
	expectedAttrs := []string{"CAST", "DSCR", "EDUC", "IDNO", "NATI", "OCCU", "RELI", "SSN", "TITL"}
	for _, exp := range expectedAttrs {
		if !attrTypes[exp] {
			t.Errorf("Attribute %s not found", exp)
		}
	}

	// Test associations
	if len(indi.Associations) < 1 {
		t.Error("No associations found")
	} else {
		// Check that various roles are parsed
		roleFound := make(map[string]bool)
		for _, assoc := range indi.Associations {
			roleFound[assoc.Role] = true
		}
		expectedRoles := []string{"FRIEND", "NGHBR", "GODP"}
		for _, role := range expectedRoles {
			if !roleFound[role] {
				t.Errorf("Association role %s not found", role)
			}
		}
	}

	// Test LDS ordinances
	if len(indi.LDSOrdinances) < 4 {
		t.Errorf("len(LDSOrdinances) = %d, want at least 4", len(indi.LDSOrdinances))
	} else {
		ordTypes := make(map[string]bool)
		for _, ord := range indi.LDSOrdinances {
			ordTypes[string(ord.Type)] = true
		}
		expectedOrds := []string{"BAPL", "CONL", "ENDL", "SLGC"}
		for _, exp := range expectedOrds {
			if !ordTypes[exp] {
				t.Errorf("LDS ordinance %s not found", exp)
			}
		}
	}

	// Test pedigree links
	if len(indi.ChildInFamilies) < 1 {
		t.Error("No FAMC links found")
	} else {
		pediFound := make(map[string]bool)
		for _, link := range indi.ChildInFamilies {
			if link.Pedigree != "" {
				pediFound[link.Pedigree] = true
			}
		}
		expectedPedi := []string{"FOSTER", "ADOPTED", "BIRTH"}
		for _, exp := range expectedPedi {
			if !pediFound[exp] {
				t.Errorf("Pedigree %s not found", exp)
			}
		}
	}
}

// TestMaximal70Family tests parsing of maximal70.ged family @F1@.
func TestMaximal70Family(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("Family @F1@ not found")
	}

	// Test family events (CENS is not currently parsed as a family event)
	eventTypes := make(map[string]bool)
	for _, event := range fam.Events {
		eventTypes[string(event.Type)] = true
	}
	expectedEvents := []string{"ANUL", "DIV", "DIVF", "ENGA", "MARB", "MARC", "MARL", "MARS", "MARR"}
	for _, exp := range expectedEvents {
		if !eventTypes[exp] {
			t.Errorf("Family event %s not found", exp)
		}
	}

	// Test marriage event details
	for _, event := range fam.Events {
		if event.Type == "MARR" {
			if event.Agency != "Agency" {
				t.Errorf("MARR.Agency = %s, want Agency", event.Agency)
			}
			if event.Cause != "Cause" {
				t.Errorf("MARR.Cause = %s, want Cause", event.Cause)
			}
			if event.Date != "27 MAR 2022" {
				t.Errorf("MARR.Date = %s, want '27 MAR 2022'", event.Date)
			}
			break
		}
	}

	// Test family LDS ordinances
	if len(fam.LDSOrdinances) < 1 {
		t.Error("No family LDS ordinances found")
	} else {
		hasSlgs := false
		for _, ord := range fam.LDSOrdinances {
			if ord.Type == "SLGS" {
				hasSlgs = true
				if ord.Temple != "LOGAN" {
					t.Errorf("SLGS.Temple = %s, want LOGAN", ord.Temple)
				}
				break
			}
		}
		if !hasSlgs {
			t.Error("SLGS ordinance not found")
		}
	}

	// Test source citations with PAGE and QUAY
	if len(fam.SourceCitations) < 1 {
		t.Error("No source citations found")
	} else {
		for _, cite := range fam.SourceCitations {
			if cite.SourceXRef == "@S1@" && cite.Page == "1" {
				if cite.Quality != 1 {
					t.Errorf("Citation QUAY = %d, want 1", cite.Quality)
				}
				break
			}
		}
	}
}

// TestMaximal70Source tests parsing of source with coordinates in DATA.
func TestMaximal70Source(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	src := doc.GetSource("@S1@")
	if src == nil {
		t.Fatal("Source @S1@ not found")
	}

	if src.Title != "Title" {
		t.Errorf("Source.Title = %s, want Title", src.Title)
	}
	if src.Author != "Author" {
		t.Errorf("Source.Author = %s, want Author", src.Author)
	}
	if src.Publication != "Publication info" {
		t.Errorf("Source.Publication = %s, want 'Publication info'", src.Publication)
	}
}

// === Edge Case Tests ===
// These tests validate handling of empty, nil, and missing values.

// TestEmptyEventSubordinates tests events without subordinate tags.
func TestEmptyEventSubordinates(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
1 DEAT Y
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(indi.Events))
	}

	// Birth with no subordinates
	birth := indi.Events[0]
	if birth.Date != "" {
		t.Errorf("BIRT.Date = %s, want empty", birth.Date)
	}
	if birth.Place != "" {
		t.Errorf("BIRT.Place = %s, want empty", birth.Place)
	}
	if birth.PlaceDetail != nil {
		t.Errorf("BIRT.PlaceDetail = %v, want nil", birth.PlaceDetail)
	}

	// Death with just Y value
	death := indi.Events[1]
	if death.Date != "" {
		t.Errorf("DEAT.Date = %s, want empty", death.Date)
	}
}

// TestEmptyAttributeSubordinates tests attributes without subordinate tags.
func TestEmptyAttributeSubordinates(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 OCCU Farmer
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Attributes) != 1 {
		t.Fatalf("len(Attributes) = %d, want 1", len(indi.Attributes))
	}

	attr := indi.Attributes[0]
	if attr.Value != "Farmer" {
		t.Errorf("Attribute.Value = %s, want Farmer", attr.Value)
	}
	if attr.Date != "" {
		t.Errorf("Attribute.Date = %s, want empty", attr.Date)
	}
	if attr.Place != "" {
		t.Errorf("Attribute.Place = %s, want empty", attr.Place)
	}
	if len(attr.SourceCitations) != 0 {
		t.Errorf("len(SourceCitations) = %d, want 0", len(attr.SourceCitations))
	}
}

// TestEmptyLDSOrdinance tests LDS ordinances with minimal data.
func TestEmptyLDSOrdinance(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BAPL
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.LDSOrdinances) != 1 {
		t.Fatalf("len(LDSOrdinances) = %d, want 1", len(indi.LDSOrdinances))
	}

	ord := indi.LDSOrdinances[0]
	if ord.Type != "BAPL" {
		t.Errorf("Type = %s, want BAPL", ord.Type)
	}
	if ord.Date != "" {
		t.Errorf("Date = %s, want empty", ord.Date)
	}
	if ord.Temple != "" {
		t.Errorf("Temple = %s, want empty", ord.Temple)
	}
	if ord.Status != "" {
		t.Errorf("Status = %s, want empty", ord.Status)
	}
}

// TestEmptyAssociation tests associations with minimal data.
func TestEmptyAssociation(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 ASSO @I2@
0 @I2@ INDI
1 NAME Jane /Smith/
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Associations) != 1 {
		t.Fatalf("len(Associations) = %d, want 1", len(indi.Associations))
	}

	assoc := indi.Associations[0]
	if assoc.IndividualXRef != "@I2@" {
		t.Errorf("IndividualXRef = %s, want @I2@", assoc.IndividualXRef)
	}
	if assoc.Role != "" {
		t.Errorf("Role = %s, want empty", assoc.Role)
	}
	if len(assoc.Notes) != 0 {
		t.Errorf("len(Notes) = %d, want 0", len(assoc.Notes))
	}
}

// TestEmptySourceCitation tests source citations with minimal data.
func TestEmptySourceCitation(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 SOUR @S1@
0 @S1@ SOUR
1 TITL Test Source
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.SourceCitations) != 1 {
		t.Fatalf("len(SourceCitations) = %d, want 1", len(indi.SourceCitations))
	}

	cite := indi.SourceCitations[0]
	if cite.SourceXRef != "@S1@" {
		t.Errorf("SourceXRef = %s, want @S1@", cite.SourceXRef)
	}
	if cite.Page != "" {
		t.Errorf("Page = %s, want empty", cite.Page)
	}
	if cite.Quality != 0 {
		t.Errorf("Quality = %d, want 0", cite.Quality)
	}
	if cite.Data != nil {
		t.Errorf("Data = %v, want nil", cite.Data)
	}
}

// TestEmptyNameComponents tests name parsing with empty components.
func TestEmptyNameComponents(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME //
0 @I2@ INDI
1 NAME Unknown//
0 @I3@ INDI
1 NAME  /Smith/
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Empty name
	indi1 := doc.GetIndividual("@I1@")
	if indi1 == nil {
		t.Fatal("@I1@ not found")
	}
	if len(indi1.Names) != 1 {
		t.Fatalf("@I1@ len(Names) = %d, want 1", len(indi1.Names))
	}
	if indi1.Names[0].Given != "" {
		t.Errorf("@I1@ Given = %s, want empty", indi1.Names[0].Given)
	}
	if indi1.Names[0].Surname != "" {
		t.Errorf("@I1@ Surname = %s, want empty", indi1.Names[0].Surname)
	}

	// Given only, no surname
	indi2 := doc.GetIndividual("@I2@")
	if indi2 == nil {
		t.Fatal("@I2@ not found")
	}
	if indi2.Names[0].Given != "Unknown" {
		t.Errorf("@I2@ Given = %s, want Unknown", indi2.Names[0].Given)
	}
	if indi2.Names[0].Surname != "" {
		t.Errorf("@I2@ Surname = %s, want empty", indi2.Names[0].Surname)
	}

	// Surname only, no given
	indi3 := doc.GetIndividual("@I3@")
	if indi3 == nil {
		t.Fatal("@I3@ not found")
	}
	if indi3.Names[0].Surname != "Smith" {
		t.Errorf("@I3@ Surname = %s, want Smith", indi3.Names[0].Surname)
	}
}

// TestPlaceWithoutCoordinates tests place parsing without MAP subordinates.
func TestPlaceWithoutCoordinates(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 PLAC Boston, MA
3 FORM City, State
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(indi.Events))
	}

	birth := indi.Events[0]
	if birth.PlaceDetail == nil {
		t.Fatal("PlaceDetail is nil")
	}
	if birth.PlaceDetail.Name != "Boston, MA" {
		t.Errorf("PlaceDetail.Name = %s, want 'Boston, MA'", birth.PlaceDetail.Name)
	}
	if birth.PlaceDetail.Form != "City, State" {
		t.Errorf("PlaceDetail.Form = %s, want 'City, State'", birth.PlaceDetail.Form)
	}
	if birth.PlaceDetail.Coordinates != nil {
		t.Errorf("PlaceDetail.Coordinates = %v, want nil", birth.PlaceDetail.Coordinates)
	}
}

// TestSourceCitationInvalidQuay tests source citation with non-numeric QUAY.
func TestSourceCitationInvalidQuay(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 SOUR @S1@
2 QUAY invalid
0 @S1@ SOUR
1 TITL Test
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.SourceCitations) != 1 {
		t.Fatalf("len(SourceCitations) = %d, want 1", len(indi.SourceCitations))
	}

	// Invalid QUAY should result in 0 (default)
	if indi.SourceCitations[0].Quality != 0 {
		t.Errorf("Quality = %d, want 0 (invalid value ignored)", indi.SourceCitations[0].Quality)
	}
}

// TestMultipleSourceCitationsOnEvent tests multiple source citations on same event.
func TestMultipleSourceCitationsOnEvent(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1850
2 SOUR @S1@
3 PAGE p. 10
2 SOUR @S2@
3 PAGE p. 20
3 QUAY 3
0 @S1@ SOUR
1 TITL Source 1
0 @S2@ SOUR
1 TITL Source 2
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found")
	}

	if len(indi.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(indi.Events))
	}

	birth := indi.Events[0]
	if len(birth.SourceCitations) != 2 {
		t.Fatalf("len(SourceCitations) = %d, want 2", len(birth.SourceCitations))
	}

	// First citation
	if birth.SourceCitations[0].SourceXRef != "@S1@" {
		t.Errorf("Citation[0].SourceXRef = %s, want @S1@", birth.SourceCitations[0].SourceXRef)
	}
	if birth.SourceCitations[0].Page != "p. 10" {
		t.Errorf("Citation[0].Page = %s, want 'p. 10'", birth.SourceCitations[0].Page)
	}

	// Second citation
	if birth.SourceCitations[1].SourceXRef != "@S2@" {
		t.Errorf("Citation[1].SourceXRef = %s, want @S2@", birth.SourceCitations[1].SourceXRef)
	}
	if birth.SourceCitations[1].Page != "p. 20" {
		t.Errorf("Citation[1].Page = %s, want 'p. 20'", birth.SourceCitations[1].Page)
	}
	if birth.SourceCitations[1].Quality != 3 {
		t.Errorf("Citation[1].Quality = %d, want 3", birth.SourceCitations[1].Quality)
	}
}
