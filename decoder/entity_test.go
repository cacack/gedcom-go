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

// TestEventAddressStructure tests parsing of event address structures.
// Tests ADDR tag with subordinates (ADR1, CITY, STAE, POST, CTRY) and contact info.
// Priority: P1 (Critical for residence and location events)
// Ref: Issue #12
func TestEventAddressStructure(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 RESI
2 DATE 1950
2 ADDR 123 Main St
3 ADR1 123 Main St
3 ADR2 Apt 4B
3 CITY Springfield
3 STAE IL
3 POST 62701
3 CTRY USA
2 PHON (555) 123-4567
2 PHON (555) 987-6543
2 EMAIL john.doe@example.com
2 FAX (555) 111-2222
2 WWW http://www.example.com
1 BIRT
2 DATE 15 JAN 1920
2 PLAC Boston, MA
2 ADDR Boston General Hospital
3 CITY Boston
3 STAE MA
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

	// Test residence event with full address structure
	resi := indi.Events[0]
	if resi.Type != "RESI" {
		t.Errorf("Event.Type = %s, want RESI", resi.Type)
	}
	if resi.Date != "1950" {
		t.Errorf("Event.Date = %s, want 1950", resi.Date)
	}

	// Test address structure
	if resi.Address == nil {
		t.Fatal("Event.Address is nil, want non-nil")
	}
	if resi.Address.Line1 != "123 Main St" {
		t.Errorf("Address.Line1 = %s, want '123 Main St'", resi.Address.Line1)
	}
	if resi.Address.Line2 != "Apt 4B" {
		t.Errorf("Address.Line2 = %s, want 'Apt 4B'", resi.Address.Line2)
	}
	if resi.Address.City != "Springfield" {
		t.Errorf("Address.City = %s, want Springfield", resi.Address.City)
	}
	if resi.Address.State != "IL" {
		t.Errorf("Address.State = %s, want IL", resi.Address.State)
	}
	if resi.Address.PostalCode != "62701" {
		t.Errorf("Address.PostalCode = %s, want 62701", resi.Address.PostalCode)
	}
	if resi.Address.Country != "USA" {
		t.Errorf("Address.Country = %s, want USA", resi.Address.Country)
	}

	// Test contact information (event-level, not address-level)
	if len(resi.Phone) != 2 {
		t.Fatalf("len(Phone) = %d, want 2", len(resi.Phone))
	}
	if resi.Phone[0] != "(555) 123-4567" {
		t.Errorf("Phone[0] = %s, want '(555) 123-4567'", resi.Phone[0])
	}
	if resi.Phone[1] != "(555) 987-6543" {
		t.Errorf("Phone[1] = %s, want '(555) 987-6543'", resi.Phone[1])
	}

	if len(resi.Email) != 1 {
		t.Fatalf("len(Email) = %d, want 1", len(resi.Email))
	}
	if resi.Email[0] != "john.doe@example.com" {
		t.Errorf("Email[0] = %s, want john.doe@example.com", resi.Email[0])
	}

	if len(resi.Fax) != 1 {
		t.Fatalf("len(Fax) = %d, want 1", len(resi.Fax))
	}
	if resi.Fax[0] != "(555) 111-2222" {
		t.Errorf("Fax[0] = %s, want '(555) 111-2222'", resi.Fax[0])
	}

	if len(resi.Website) != 1 {
		t.Fatalf("len(Website) = %d, want 1", len(resi.Website))
	}
	if resi.Website[0] != "http://www.example.com" {
		t.Errorf("Website[0] = %s, want http://www.example.com", resi.Website[0])
	}

	// Test birth event with minimal address (no ADR1 subordinate)
	birth := indi.Events[1]
	if birth.Type != "BIRT" {
		t.Errorf("Event.Type = %s, want BIRT", birth.Type)
	}

	if birth.Address == nil {
		t.Fatal("Birth.Address is nil, want non-nil")
	}
	if birth.Address.Line1 != "Boston General Hospital" {
		t.Errorf("Birth.Address.Line1 = %s, want 'Boston General Hospital'", birth.Address.Line1)
	}
	if birth.Address.City != "Boston" {
		t.Errorf("Birth.Address.City = %s, want Boston", birth.Address.City)
	}
	if birth.Address.State != "MA" {
		t.Errorf("Birth.Address.State = %s, want MA", birth.Address.State)
	}

	// Birth should have both PLAC and ADDR
	if birth.Place != "Boston, MA" {
		t.Errorf("Birth.Place = %s, want 'Boston, MA'", birth.Place)
	}
}

// TestEventAddressWithContinuation tests ADDR with CONT/CONC.
func TestEventAddressWithContinuation(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 RESI
2 ADDR 123 Main Street
3 CONT Suite 100
3 CITY Springfield
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

	resi := indi.Events[0]
	if resi.Address == nil {
		t.Fatal("Address is nil")
	}

	expected := "123 Main Street\nSuite 100"
	if resi.Address.Line1 != expected {
		t.Errorf("Address.Line1 = %q, want %q", resi.Address.Line1, expected)
	}
	if resi.Address.City != "Springfield" {
		t.Errorf("Address.City = %s, want Springfield", resi.Address.City)
	}
}

// TestEventWithoutAddress tests events without address fields.
func TestEventWithoutAddress(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1920
2 PLAC Boston, MA
1 DEAT
2 DATE 1990
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

	// Birth has no address
	birth := indi.Events[0]
	if birth.Address != nil {
		t.Errorf("Birth.Address = %v, want nil", birth.Address)
	}
	if len(birth.Phone) != 0 {
		t.Errorf("len(Birth.Phone) = %d, want 0", len(birth.Phone))
	}
	if len(birth.Email) != 0 {
		t.Errorf("len(Birth.Email) = %d, want 0", len(birth.Email))
	}

	// Death has no address
	death := indi.Events[1]
	if death.Address != nil {
		t.Errorf("Death.Address = %v, want nil", death.Address)
	}
}

// TestFamilyEventAddress tests address parsing in family events.
func TestFamilyEventAddress(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 15 JUN 1875
2 PLAC Chicago, IL
2 ADDR St. Patrick's Church
3 CITY Chicago
3 STAE IL
2 PHON (312) 555-1234
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

	if len(fam.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(fam.Events))
	}

	marr := fam.Events[0]
	if marr.Type != "MARR" {
		t.Errorf("Event.Type = %s, want MARR", marr.Type)
	}

	if marr.Address == nil {
		t.Fatal("Marriage.Address is nil, want non-nil")
	}
	if marr.Address.Line1 != "St. Patrick's Church" {
		t.Errorf("Address.Line1 = %s, want 'St. Patrick's Church'", marr.Address.Line1)
	}
	if marr.Address.City != "Chicago" {
		t.Errorf("Address.City = %s, want Chicago", marr.Address.City)
	}
	if marr.Address.State != "IL" {
		t.Errorf("Address.State = %s, want IL", marr.Address.State)
	}

	if len(marr.Phone) != 1 {
		t.Fatalf("len(Phone) = %d, want 1", len(marr.Phone))
	}
	if marr.Phone[0] != "(312) 555-1234" {
		t.Errorf("Phone[0] = %s, want '(312) 555-1234'", marr.Phone[0])
	}
}

// TestRecordMetadata tests parsing of record metadata tags.
// Tests CHAN, CREA, REFN, and UID tags on Individual, Family, and Source records.
// Priority: P1 (Critical for synchronization and version tracking)
// Ref: Issue #13
func TestRecordMetadata(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 CHAN
2 DATE 27 MAR 2022
3 TIME 08:56:00
1 CREA
2 DATE 15 JAN 2020
3 TIME 10:30:00
1 REFN 12345
1 UID 12345678-1234-1234-1234-123456789012
0 @F1@ FAM
1 HUSB @I1@
1 CHAN
2 DATE 10 APR 2022
3 TIME 14:20:15
1 REFN FAM-001
1 UID abcdef12-3456-7890-abcd-ef1234567890
0 @S1@ SOUR
1 TITL Test Source
1 CHAN
2 DATE 5 MAY 2022
3 TIME 09:15:30
1 CREA
2 DATE 1 JAN 2019
1 REFN SRC-999
1 UID fedcba98-7654-3210-fedc-ba9876543210
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Test Individual metadata
	indi := doc.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual @I1@ not found")
	}

	// Test CHAN (change date)
	if indi.ChangeDate == nil {
		t.Fatal("Individual.ChangeDate is nil, want non-nil")
	}
	if indi.ChangeDate.Date != "27 MAR 2022" {
		t.Errorf("Individual.ChangeDate.Date = %s, want '27 MAR 2022'", indi.ChangeDate.Date)
	}
	if indi.ChangeDate.Time != "08:56:00" {
		t.Errorf("Individual.ChangeDate.Time = %s, want '08:56:00'", indi.ChangeDate.Time)
	}

	// Test CREA (creation date)
	if indi.CreationDate == nil {
		t.Fatal("Individual.CreationDate is nil, want non-nil")
	}
	if indi.CreationDate.Date != "15 JAN 2020" {
		t.Errorf("Individual.CreationDate.Date = %s, want '15 JAN 2020'", indi.CreationDate.Date)
	}
	if indi.CreationDate.Time != "10:30:00" {
		t.Errorf("Individual.CreationDate.Time = %s, want '10:30:00'", indi.CreationDate.Time)
	}

	// Test REFN (reference number)
	if indi.RefNumber != "12345" {
		t.Errorf("Individual.RefNumber = %s, want '12345'", indi.RefNumber)
	}

	// Test UID (unique identifier)
	if indi.UID != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("Individual.UID = %s, want '12345678-1234-1234-1234-123456789012'", indi.UID)
	}

	// Test Family metadata
	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("Family @F1@ not found")
	}

	if fam.ChangeDate == nil {
		t.Fatal("Family.ChangeDate is nil, want non-nil")
	}
	if fam.ChangeDate.Date != "10 APR 2022" {
		t.Errorf("Family.ChangeDate.Date = %s, want '10 APR 2022'", fam.ChangeDate.Date)
	}
	if fam.ChangeDate.Time != "14:20:15" {
		t.Errorf("Family.ChangeDate.Time = %s, want '14:20:15'", fam.ChangeDate.Time)
	}

	if fam.RefNumber != "FAM-001" {
		t.Errorf("Family.RefNumber = %s, want 'FAM-001'", fam.RefNumber)
	}

	if fam.UID != "abcdef12-3456-7890-abcd-ef1234567890" {
		t.Errorf("Family.UID = %s, want 'abcdef12-3456-7890-abcd-ef1234567890'", fam.UID)
	}

	// Test Source metadata
	src := doc.GetSource("@S1@")
	if src == nil {
		t.Fatal("Source @S1@ not found")
	}

	if src.ChangeDate == nil {
		t.Fatal("Source.ChangeDate is nil, want non-nil")
	}
	if src.ChangeDate.Date != "5 MAY 2022" {
		t.Errorf("Source.ChangeDate.Date = %s, want '5 MAY 2022'", src.ChangeDate.Date)
	}
	if src.ChangeDate.Time != "09:15:30" {
		t.Errorf("Source.ChangeDate.Time = %s, want '09:15:30'", src.ChangeDate.Time)
	}

	if src.CreationDate == nil {
		t.Fatal("Source.CreationDate is nil, want non-nil")
	}
	if src.CreationDate.Date != "1 JAN 2019" {
		t.Errorf("Source.CreationDate.Date = %s, want '1 JAN 2019'", src.CreationDate.Date)
	}

	if src.RefNumber != "SRC-999" {
		t.Errorf("Source.RefNumber = %s, want 'SRC-999'", src.RefNumber)
	}

	if src.UID != "fedcba98-7654-3210-fedc-ba9876543210" {
		t.Errorf("Source.UID = %s, want 'fedcba98-7654-3210-fedc-ba9876543210'", src.UID)
	}
}

// TestChangeDateWithoutTime tests CHAN/CREA tags with only DATE (no TIME).
func TestChangeDateWithoutTime(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 CHAN
2 DATE 15 JAN 2020
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

	if indi.ChangeDate == nil {
		t.Fatal("ChangeDate is nil, want non-nil")
	}
	if indi.ChangeDate.Date != "15 JAN 2020" {
		t.Errorf("ChangeDate.Date = %s, want '15 JAN 2020'", indi.ChangeDate.Date)
	}
	if indi.ChangeDate.Time != "" {
		t.Errorf("ChangeDate.Time = %s, want empty", indi.ChangeDate.Time)
	}
}

// TestMetadataEdgeCases tests edge cases for metadata parsing.
func TestMetadataEdgeCases(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME No /Metadata/
0 @I2@ INDI
1 NAME Partial /Metadata/
1 REFN ONLY-REFN
0 @I3@ INDI
1 NAME Only /UID/
1 UID only-uid-value
0 @I4@ INDI
1 NAME Empty /CHAN/
1 CHAN
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Individual with no metadata
	indi1 := doc.GetIndividual("@I1@")
	if indi1 == nil {
		t.Fatal("@I1@ not found")
	}
	if indi1.ChangeDate != nil {
		t.Errorf("@I1@ ChangeDate = %v, want nil", indi1.ChangeDate)
	}
	if indi1.CreationDate != nil {
		t.Errorf("@I1@ CreationDate = %v, want nil", indi1.CreationDate)
	}
	if indi1.RefNumber != "" {
		t.Errorf("@I1@ RefNumber = %s, want empty", indi1.RefNumber)
	}
	if indi1.UID != "" {
		t.Errorf("@I1@ UID = %s, want empty", indi1.UID)
	}

	// Individual with only REFN
	indi2 := doc.GetIndividual("@I2@")
	if indi2 == nil {
		t.Fatal("@I2@ not found")
	}
	if indi2.RefNumber != "ONLY-REFN" {
		t.Errorf("@I2@ RefNumber = %s, want 'ONLY-REFN'", indi2.RefNumber)
	}
	if indi2.UID != "" {
		t.Errorf("@I2@ UID = %s, want empty", indi2.UID)
	}

	// Individual with only UID
	indi3 := doc.GetIndividual("@I3@")
	if indi3 == nil {
		t.Fatal("@I3@ not found")
	}
	if indi3.UID != "only-uid-value" {
		t.Errorf("@I3@ UID = %s, want 'only-uid-value'", indi3.UID)
	}
	if indi3.RefNumber != "" {
		t.Errorf("@I3@ RefNumber = %s, want empty", indi3.RefNumber)
	}

	// Individual with CHAN but no DATE subordinate
	indi4 := doc.GetIndividual("@I4@")
	if indi4 == nil {
		t.Fatal("@I4@ not found")
	}
	if indi4.ChangeDate == nil {
		t.Fatal("@I4@ ChangeDate is nil, want non-nil (empty ChangeDate)")
	}
	if indi4.ChangeDate.Date != "" {
		t.Errorf("@I4@ ChangeDate.Date = %s, want empty", indi4.ChangeDate.Date)
	}
	if indi4.ChangeDate.Time != "" {
		t.Errorf("@I4@ ChangeDate.Time = %s, want empty", indi4.ChangeDate.Time)
	}
}

// TestFamilyStatisticsAttributes tests parsing of family statistics attributes (NCHI, NMR, PROP).
// Tests number of children, number of marriages, and property attributes on individuals and families.
// Priority: P2 (Important for demographic/genealogical statistics)
// Ref: Issue #17
func TestFamilyStatisticsAttributes(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 NCHI 3
1 NMR 2
1 PROP House and land
2 DATE 1920
2 PLAC Springfield, IL
0 @I2@ INDI
1 NAME Jane /Smith/
1 NCHI 5
1 NMR 1
1 PROP Estate in Boston
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 NCHI 3
1 CHIL @I3@
1 CHIL @I4@
1 CHIL @I5@
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Test individual @I1@ with all three statistics attributes
	indi1 := doc.GetIndividual("@I1@")
	if indi1 == nil {
		t.Fatal("Individual @I1@ not found")
	}

	if len(indi1.Attributes) != 3 {
		t.Fatalf("@I1@ len(Attributes) = %d, want 3", len(indi1.Attributes))
	}

	// Build attribute map for easier testing
	attrMap := make(map[string]string)
	attrDates := make(map[string]string)
	attrPlaces := make(map[string]string)
	for _, attr := range indi1.Attributes {
		attrMap[attr.Type] = attr.Value
		attrDates[attr.Type] = attr.Date
		attrPlaces[attr.Type] = attr.Place
	}

	// Test NCHI (Number of Children)
	if nchi, ok := attrMap["NCHI"]; !ok {
		t.Error("NCHI attribute not found on @I1@")
	} else if nchi != "3" {
		t.Errorf("NCHI.Value = %s, want 3", nchi)
	}

	// Test NMR (Number of Marriages)
	if nmr, ok := attrMap["NMR"]; !ok {
		t.Error("NMR attribute not found on @I1@")
	} else if nmr != "2" {
		t.Errorf("NMR.Value = %s, want 2", nmr)
	}

	// Test PROP (Property/Possessions)
	if prop, ok := attrMap["PROP"]; !ok {
		t.Error("PROP attribute not found on @I1@")
	} else {
		if prop != "House and land" {
			t.Errorf("PROP.Value = %s, want 'House and land'", prop)
		}
		if attrDates["PROP"] != "1920" {
			t.Errorf("PROP.Date = %s, want 1920", attrDates["PROP"])
		}
		if attrPlaces["PROP"] != "Springfield, IL" {
			t.Errorf("PROP.Place = %s, want 'Springfield, IL'", attrPlaces["PROP"])
		}
	}

	// Test individual @I2@ with different values
	indi2 := doc.GetIndividual("@I2@")
	if indi2 == nil {
		t.Fatal("Individual @I2@ not found")
	}

	if len(indi2.Attributes) != 3 {
		t.Fatalf("@I2@ len(Attributes) = %d, want 3", len(indi2.Attributes))
	}

	attrMap2 := make(map[string]string)
	for _, attr := range indi2.Attributes {
		attrMap2[attr.Type] = attr.Value
	}

	if nchi, ok := attrMap2["NCHI"]; !ok {
		t.Error("NCHI attribute not found on @I2@")
	} else if nchi != "5" {
		t.Errorf("@I2@ NCHI.Value = %s, want 5", nchi)
	}

	if nmr, ok := attrMap2["NMR"]; !ok {
		t.Error("NMR attribute not found on @I2@")
	} else if nmr != "1" {
		t.Errorf("@I2@ NMR.Value = %s, want 1", nmr)
	}

	if prop, ok := attrMap2["PROP"]; !ok {
		t.Error("PROP attribute not found on @I2@")
	} else if prop != "Estate in Boston" {
		t.Errorf("@I2@ PROP.Value = %s, want 'Estate in Boston'", prop)
	}

	// Test family @F1@ with NCHI field
	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("Family @F1@ not found")
	}

	if fam.NumberOfChildren != "3" {
		t.Errorf("Family.NumberOfChildren = %s, want 3", fam.NumberOfChildren)
	}

	// Verify the family has 3 CHIL tags
	if len(fam.Children) != 3 {
		t.Errorf("len(fam.Children) = %d, want 3", len(fam.Children))
	}
}

// TestEventAdministrativeTags tests parsing of event administrative tags (RESN, UID, SDATE).
// Tests restriction notice, unique identifier, and sort date fields.
// Priority: P1 (Critical for privacy controls and proper date ordering)
// Ref: Issue #16
func TestEventAdministrativeTags(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1 JAN 1900
2 PLAC Springfield, IL
2 RESN confidential
2 UID 12345678-1234-1234-1234-123456789012
2 SDATE 1900-01-01
1 DEAT
2 DATE 20 MAR 1970
2 RESN privacy
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

	// Test birth event with all three administrative tags
	birth := indi.Events[0]
	if birth.Type != "BIRT" {
		t.Errorf("Event.Type = %s, want BIRT", birth.Type)
	}
	if birth.Date != "1 JAN 1900" {
		t.Errorf("Event.Date = %s, want '1 JAN 1900'", birth.Date)
	}
	if birth.Restriction != "confidential" {
		t.Errorf("Event.Restriction = %s, want 'confidential'", birth.Restriction)
	}
	if birth.UID != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("Event.UID = %s, want '12345678-1234-1234-1234-123456789012'", birth.UID)
	}
	if birth.SortDate != "1900-01-01" {
		t.Errorf("Event.SortDate = %s, want '1900-01-01'", birth.SortDate)
	}

	// Test death event with only RESN tag
	death := indi.Events[1]
	if death.Type != "DEAT" {
		t.Errorf("Event.Type = %s, want DEAT", death.Type)
	}
	if death.Restriction != "privacy" {
		t.Errorf("Event.Restriction = %s, want 'privacy'", death.Restriction)
	}
	if death.UID != "" {
		t.Errorf("Event.UID = %s, want empty", death.UID)
	}
	if death.SortDate != "" {
		t.Errorf("Event.SortDate = %s, want empty", death.SortDate)
	}
}

func TestSubmitterParsing(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR Test
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @U1@ SUBM
1 NAME John Q. Genealogist
1 ADDR 123 Main St
2 CITY Springfield
2 STAE IL
2 POST 62701
2 CTRY USA
1 PHON (555) 123-4567
1 EMAIL user@example.com
1 LANG English
1 LANG German
1 NOTE Submitter note
0 @U2@ SUBM
1 NAME Jane Smith
1 PHON (555) 987-6543
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Test Submitters() returns all submitters
	submitters := doc.Submitters()
	if len(submitters) != 2 {
		t.Fatalf("Submitters() = %d, want 2", len(submitters))
	}

	// Test GetSubmitter with first submitter
	subm1 := doc.GetSubmitter("@U1@")
	if subm1 == nil {
		t.Fatal("GetSubmitter(@U1@) returned nil")
	}

	// Test submitter fields
	if subm1.XRef != "@U1@" {
		t.Errorf("subm1.XRef = %s, want @U1@", subm1.XRef)
	}
	if subm1.Name != "John Q. Genealogist" {
		t.Errorf("subm1.Name = %s, want 'John Q. Genealogist'", subm1.Name)
	}

	// Test address
	if subm1.Address == nil {
		t.Fatal("subm1.Address is nil")
	}
	if subm1.Address.Line1 != "123 Main St" {
		t.Errorf("subm1.Address.Line1 = %s, want '123 Main St'", subm1.Address.Line1)
	}
	if subm1.Address.City != "Springfield" {
		t.Errorf("subm1.Address.City = %s, want 'Springfield'", subm1.Address.City)
	}
	if subm1.Address.State != "IL" {
		t.Errorf("subm1.Address.State = %s, want 'IL'", subm1.Address.State)
	}
	if subm1.Address.PostalCode != "62701" {
		t.Errorf("subm1.Address.PostalCode = %s, want '62701'", subm1.Address.PostalCode)
	}
	if subm1.Address.Country != "USA" {
		t.Errorf("subm1.Address.Country = %s, want 'USA'", subm1.Address.Country)
	}

	// Test phone
	if len(subm1.Phone) != 1 {
		t.Fatalf("len(subm1.Phone) = %d, want 1", len(subm1.Phone))
	}
	if subm1.Phone[0] != "(555) 123-4567" {
		t.Errorf("subm1.Phone[0] = %s, want '(555) 123-4567'", subm1.Phone[0])
	}

	// Test email
	if len(subm1.Email) != 1 {
		t.Fatalf("len(subm1.Email) = %d, want 1", len(subm1.Email))
	}
	if subm1.Email[0] != "user@example.com" {
		t.Errorf("subm1.Email[0] = %s, want 'user@example.com'", subm1.Email[0])
	}

	// Test languages (multiple)
	if len(subm1.Language) != 2 {
		t.Fatalf("len(subm1.Language) = %d, want 2", len(subm1.Language))
	}
	if subm1.Language[0] != "English" {
		t.Errorf("subm1.Language[0] = %s, want 'English'", subm1.Language[0])
	}
	if subm1.Language[1] != "German" {
		t.Errorf("subm1.Language[1] = %s, want 'German'", subm1.Language[1])
	}

	// Test notes
	if len(subm1.Notes) != 1 {
		t.Fatalf("len(subm1.Notes) = %d, want 1", len(subm1.Notes))
	}
	if subm1.Notes[0] != "Submitter note" {
		t.Errorf("subm1.Notes[0] = %s, want 'Submitter note'", subm1.Notes[0])
	}

	// Test second submitter (minimal)
	subm2 := doc.GetSubmitter("@U2@")
	if subm2 == nil {
		t.Fatal("GetSubmitter(@U2@) returned nil")
	}
	if subm2.XRef != "@U2@" {
		t.Errorf("subm2.XRef = %s, want @U2@", subm2.XRef)
	}
	if subm2.Name != "Jane Smith" {
		t.Errorf("subm2.Name = %s, want 'Jane Smith'", subm2.Name)
	}
	if len(subm2.Phone) != 1 {
		t.Fatalf("len(subm2.Phone) = %d, want 1", len(subm2.Phone))
	}
	if subm2.Phone[0] != "(555) 987-6543" {
		t.Errorf("subm2.Phone[0] = %s, want '(555) 987-6543'", subm2.Phone[0])
	}

	// Test GetSubmitter with non-existent xref
	nonExistent := doc.GetSubmitter("@U999@")
	if nonExistent != nil {
		t.Error("GetSubmitter(@U999@) should return nil")
	}
}

// TestRepositoryParsing tests parsing of Repository (REPO) records.
// Ref: Issue #15
func TestRepositoryParsing(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5
0 @R1@ REPO
1 NAME Family History Library
1 ADDR 35 North West Temple Street
2 CITY Salt Lake City
2 STAE Utah
2 POST 84150
2 CTRY USA
1 PHON (801) 240-2584
1 EMAIL fhl@familysearch.org
1 WWW https://www.familysearch.org
1 NOTE Great resource for genealogy research
0 @R2@ REPO
1 NAME National Archives
1 ADDR 8601 Adelphi Road
2 CITY College Park
2 STAE Maryland
2 POST 20740
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Test Repositories() method
	repositories := doc.Repositories()
	if len(repositories) != 2 {
		t.Fatalf("len(Repositories()) = %d, want 2", len(repositories))
	}

	// Test GetRepository for first repository
	repo1 := doc.GetRepository("@R1@")
	if repo1 == nil {
		t.Fatal("GetRepository(@R1@) returned nil")
	}
	if repo1.XRef != "@R1@" {
		t.Errorf("repo1.XRef = %s, want @R1@", repo1.XRef)
	}
	if repo1.Name != "Family History Library" {
		t.Errorf("repo1.Name = %s, want 'Family History Library'", repo1.Name)
	}

	// Test address structure
	if repo1.Address == nil {
		t.Fatal("repo1.Address is nil")
	}
	if repo1.Address.Line1 != "35 North West Temple Street" {
		t.Errorf("repo1.Address.Line1 = %s, want '35 North West Temple Street'", repo1.Address.Line1)
	}
	if repo1.Address.City != "Salt Lake City" {
		t.Errorf("repo1.Address.City = %s, want 'Salt Lake City'", repo1.Address.City)
	}
	if repo1.Address.State != "Utah" {
		t.Errorf("repo1.Address.State = %s, want 'Utah'", repo1.Address.State)
	}
	if repo1.Address.PostalCode != "84150" {
		t.Errorf("repo1.Address.PostalCode = %s, want '84150'", repo1.Address.PostalCode)
	}
	if repo1.Address.Country != "USA" {
		t.Errorf("repo1.Address.Country = %s, want 'USA'", repo1.Address.Country)
	}

	// Test contact information
	if repo1.Address.Phone != "(801) 240-2584" {
		t.Errorf("repo1.Address.Phone = %s, want '(801) 240-2584'", repo1.Address.Phone)
	}
	if repo1.Address.Email != "fhl@familysearch.org" {
		t.Errorf("repo1.Address.Email = %s, want 'fhl@familysearch.org'", repo1.Address.Email)
	}
	if repo1.Address.Website != "https://www.familysearch.org" {
		t.Errorf("repo1.Address.Website = %s, want 'https://www.familysearch.org'", repo1.Address.Website)
	}

	// Test notes
	if len(repo1.Notes) != 1 {
		t.Fatalf("len(repo1.Notes) = %d, want 1", len(repo1.Notes))
	}
	if repo1.Notes[0] != "Great resource for genealogy research" {
		t.Errorf("repo1.Notes[0] = %s, want 'Great resource for genealogy research'", repo1.Notes[0])
	}

	// Test second repository
	repo2 := doc.GetRepository("@R2@")
	if repo2 == nil {
		t.Fatal("GetRepository(@R2@) returned nil")
	}
	if repo2.XRef != "@R2@" {
		t.Errorf("repo2.XRef = %s, want @R2@", repo2.XRef)
	}
	if repo2.Name != "National Archives" {
		t.Errorf("repo2.Name = %s, want 'National Archives'", repo2.Name)
	}

	// Test GetRepository with non-existent xref
	nonExistent := doc.GetRepository("@R999@")
	if nonExistent != nil {
		t.Error("GetRepository(@R999@) should return nil")
	}
}

// TestNoteParsing tests parsing of Note (NOTE) records.
// Ref: Issue #15
func TestNoteParsing(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5
0 @N1@ NOTE This is a shared note that can be
1 CONT referenced from multiple records.
1 CONT It supports continuation lines.
0 @N2@ NOTE Short note
0 @N3@ NOTE This note has conc
1 CONC atenation without space.
1 CONT And continuation with newline.
0 TRLR
`
	doc, err := Decode(strings.NewReader(gedcom))
	if err != nil {
		t.Fatal(err)
	}

	// Test Notes() method
	notes := doc.Notes()
	if len(notes) != 3 {
		t.Fatalf("len(Notes()) = %d, want 3", len(notes))
	}

	// Test GetNote for first note with CONT
	note1 := doc.GetNote("@N1@")
	if note1 == nil {
		t.Fatal("GetNote(@N1@) returned nil")
	}
	if note1.XRef != "@N1@" {
		t.Errorf("note1.XRef = %s, want @N1@", note1.XRef)
	}
	if note1.Text != "This is a shared note that can be" {
		t.Errorf("note1.Text = %s, want 'This is a shared note that can be'", note1.Text)
	}

	// Test continuation lines
	if len(note1.Continuation) != 2 {
		t.Fatalf("len(note1.Continuation) = %d, want 2", len(note1.Continuation))
	}
	if note1.Continuation[0] != "referenced from multiple records." {
		t.Errorf("note1.Continuation[0] = %s, want 'referenced from multiple records.'", note1.Continuation[0])
	}
	if note1.Continuation[1] != "It supports continuation lines." {
		t.Errorf("note1.Continuation[1] = %s, want 'It supports continuation lines.'", note1.Continuation[1])
	}

	// Test FullText method
	expectedFullText := "This is a shared note that can be\nreferenced from multiple records.\nIt supports continuation lines."
	fullText := note1.FullText()
	if fullText != expectedFullText {
		t.Errorf("note1.FullText() = %q, want %q", fullText, expectedFullText)
	}

	// Test second note (short note without continuation)
	note2 := doc.GetNote("@N2@")
	if note2 == nil {
		t.Fatal("GetNote(@N2@) returned nil")
	}
	if note2.XRef != "@N2@" {
		t.Errorf("note2.XRef = %s, want @N2@", note2.XRef)
	}
	if note2.Text != "Short note" {
		t.Errorf("note2.Text = %s, want 'Short note'", note2.Text)
	}
	if len(note2.Continuation) != 0 {
		t.Errorf("len(note2.Continuation) = %d, want 0", len(note2.Continuation))
	}
	if note2.FullText() != "Short note" {
		t.Errorf("note2.FullText() = %s, want 'Short note'", note2.FullText())
	}

	// Test third note with CONC and CONT
	note3 := doc.GetNote("@N3@")
	if note3 == nil {
		t.Fatal("GetNote(@N3@) returned nil")
	}
	if note3.XRef != "@N3@" {
		t.Errorf("note3.XRef = %s, want @N3@", note3.XRef)
	}
	// CONC should concatenate to main text
	expectedText := "This note has concatenation without space."
	if note3.Text != expectedText {
		t.Errorf("note3.Text = %s, want %s", note3.Text, expectedText)
	}
	// CONT should add to continuation
	if len(note3.Continuation) != 1 {
		t.Fatalf("len(note3.Continuation) = %d, want 1", len(note3.Continuation))
	}
	if note3.Continuation[0] != "And continuation with newline." {
		t.Errorf("note3.Continuation[0] = %s, want 'And continuation with newline.'", note3.Continuation[0])
	}

	expectedFullText3 := "This note has concatenation without space.\nAnd continuation with newline."
	fullText3 := note3.FullText()
	if fullText3 != expectedFullText3 {
		t.Errorf("note3.FullText() = %q, want %q", fullText3, expectedFullText3)
	}

	// Test GetNote with non-existent xref
	nonExistent := doc.GetNote("@N999@")
	if nonExistent != nil {
		t.Error("GetNote(@N999@) should return nil")
	}
}
