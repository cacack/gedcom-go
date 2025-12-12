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
	if len(john.ChildInFamilies) != 1 || john.ChildInFamilies[0] != "@F1@" {
		t.Errorf("john.ChildInFamilies = %v, want [@F1@]", john.ChildInFamilies)
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
