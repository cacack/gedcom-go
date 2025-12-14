package decoder

import (
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

// TestEventSubordinateTags_NotImplemented tests parsing of event subordinate tags.
// Gap: Event struct only captures DATE, PLAC, Description - missing TYPE, CAUS, AGE, AGNC.
// Priority: P1 (Critical)
// Ref: FEATURE-GAPS.md Section 3.1
func TestEventSubordinateTags_NotImplemented(t *testing.T) {
	t.Skip("Not yet implemented: Event subordinate tags (TYPE, CAUS, AGE, AGNC)")

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
	// These fields don't exist yet - will fail compilation when uncommented
	// if death.Type != "Natural death" {
	// 	t.Errorf("Event.Type = %s, want 'Natural death'", death.Type)
	// }
	// if death.Cause != "Heart failure" {
	// 	t.Errorf("Event.Cause = %s, want 'Heart failure'", death.Cause)
	// }
	// if death.Age != "70y" {
	// 	t.Errorf("Event.Age = %s, want '70y'", death.Age)
	// }
	// if death.Agency != "County Coroner" {
	// 	t.Errorf("Event.Agency = %s, want 'County Coroner'", death.Agency)
	// }
	_ = death // Suppress unused variable warning
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

// TestIndividualAttributes_NotImplemented tests parsing of individual attributes.
// Gap: Only OCCU is parsed - missing CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL, RELI.
// Priority: P2 (Important)
// Ref: FEATURE-GAPS.md Section 2
func TestIndividualAttributes_NotImplemented(t *testing.T) {
	t.Skip("Not yet implemented: Individual attributes (CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL, RELI)")

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
}

// TestReligiousEvents_NotImplemented tests parsing of religious event types.
// Gap: BARM, BASM, BLES, CONF, FCOM, CHRA not parsed.
// Priority: P2 (Important)
// Ref: FEATURE-GAPS.md Section 1.1
func TestReligiousEvents_NotImplemented(t *testing.T) {
	t.Skip("Not yet implemented: Religious events (BARM, BASM, BLES, CONF, FCOM, CHRA)")

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

// TestLifeEvents_NotImplemented tests parsing of life status events.
// Gap: GRAD, RETI, NATU, ORDN, PROB, WILL, CREM not parsed.
// Priority: P2 (Important)
// Ref: FEATURE-GAPS.md Sections 1.2, 1.3, 1.4, 1.5
func TestLifeEvents_NotImplemented(t *testing.T) {
	t.Skip("Not yet implemented: Life events (GRAD, RETI, NATU, ORDN, PROB, WILL, CREM)")

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

// TestLDSOrdinances_NotImplemented tests parsing of LDS ordinance structures.
// Gap: BAPL, CONL, ENDL, SLGC, SLGS not parsed at all.
// Priority: P2 (Critical for LDS users)
// Ref: FEATURE-GAPS.md Section 5
func TestLDSOrdinances_NotImplemented(t *testing.T) {
	t.Skip("Not yet implemented: LDS ordinances (BAPL, CONL, ENDL, SLGC, SLGS)")

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
	// These fields don't exist yet - will fail compilation when uncommented
	// if len(indi.LDSOrdinances) != 4 {
	// 	t.Errorf("len(LDSOrdinances) = %d, want 4", len(indi.LDSOrdinances))
	// }
	//
	// expectedOrds := []string{"BAPL", "CONL", "ENDL", "SLGC"}
	// for i, expected := range expectedOrds {
	// 	if indi.LDSOrdinances[i].Type != expected {
	// 		t.Errorf("LDSOrdinance[%d].Type = %s, want %s", i, indi.LDSOrdinances[i].Type, expected)
	// 	}
	// 	if indi.LDSOrdinances[i].Temple == "" {
	// 		t.Errorf("LDSOrdinance[%d].Temple is empty", i)
	// 	}
	// }

	// Family ordinance
	fam := doc.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("Family not found")
	}
	// TODO: When implemented, assert fam.LDSOrdinances has 1 entry of type SLGS
	_, _ = indi, fam
}

// TestNameExtensions_NotImplemented tests parsing of extended name components.
// Gap: NICK, SPFX not parsed (TRAN also missing but lower priority).
// Priority: P2 (Important for international genealogy)
// Ref: FEATURE-GAPS.md Section 7.1
func TestNameExtensions_NotImplemented(t *testing.T) {
	t.Skip("Not yet implemented: Name extensions (NICK, SPFX, TRAN)")

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
	// TODO: When implemented, assert name.SurnamePrefix == "von" and name.Nickname == "Ludwig"
	_ = name
}

// TestIndividualAssociations_NotImplemented tests parsing of ASSO tag.
// Gap: ASSO tag with ROLE subordinate not parsed.
// Priority: P2 (Important for relationship context)
// Ref: FEATURE-GAPS.md Section 8.1
func TestIndividualAssociations_NotImplemented(t *testing.T) {
	t.Skip("Not yet implemented: Individual associations (ASSO with ROLE)")

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

	// TODO: When implemented, assert:
	// - indi.Associations has 2 entries
	// - Associations[0]: XRef=@I2@, Role=GODP (Godparent)
	// - Associations[1]: XRef=@I3@, Role=WITN (Witness)
	_ = indi
}
