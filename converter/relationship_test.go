package converter

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #732: an upgrade adds FAMC.PEDI from an unambiguous non-natural
// _FREL/_MREL, and an upgrade to 7.0 adds a TYPE MARRIED name from _MARNM.

const relationshipFixture = `0 HEAD
1 GEDC
2 VERS %s
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Ann Marie /Smith/ Jr
2 GIVN Ann Marie
2 _MARNM Jones
1 NAME Annie
2 _MARNM Ann /Brown/
1 FAMS @F1@
0 @I2@ INDI
1 FAMC @F1@
2 NOTE adopted by both
0 @I3@ INDI
1 FAMC @F1@
0 @I4@ INDI
1 FAMC @F1@
0 @I5@ INDI
1 FAMC @F1@
0 @I6@ INDI
1 FAMC @F1@
2 PEDI foster
0 @I7@ INDI
1 FAMC @F2@
0 @I8@ INDI
1 NAME Cy /Doe/
2 _MARNM Roe
1 NAME Cy /Roe/
2 TYPE married
0 @F1@ FAM
1 HUSB @H1@
1 WIFE @I1@
1 CHIL @I2@
2 _FREL Adopted
2 _MREL adopted
1 CHIL @I3@
2 _FREL Natural
2 _MREL Step
1 CHIL @I4@
2 _FREL Natural
2 _MREL Natural
1 CHIL @I5@
2 _FREL Adopted
1 CHIL @I6@
2 _FREL Adopted
2 _MREL Adopted
0 @F2@ FAM
1 HUSB @H2@
1 CHIL @I7@
2 _FREL Foster
0 TRLR
`

func decodeText(t *testing.T, text string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(text))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return doc
}

// tagLines renders a record's Tags one "level tag value" line each.
func tagLines(rec *gedcom.Record) []string {
	var out []string
	for _, tag := range rec.Tags {
		out = append(out, strings.TrimSpace(strconv.Itoa(tag.Level)+" "+tag.Tag+" "+tag.Value))
	}
	return out
}

func TestConvertSynthesizesPedigree(t *testing.T) {
	tests := []struct {
		from, target gedcom.Version
		adopted      string
		foster       string
	}{
		{gedcom.Version55, gedcom.Version551, "adopted", "foster"},
		{gedcom.Version55, gedcom.Version70, "ADOPTED", "FOSTER"},
		{gedcom.Version551, gedcom.Version70, "ADOPTED", "FOSTER"},
	}
	for _, tt := range tests {
		t.Run(tt.from.String()+" to "+tt.target.String(), func(t *testing.T) {
			src := strings.Replace(relationshipFixture, "%s", tt.from.String(), 1)
			converted, report, err := Convert(decodeText(t, src), tt.target)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}

			want := map[string][]string{
				// Both parents adopted: PEDI first under FAMC, before the NOTE.
				"@I2@": {"1 FAMC @F1@", "2 PEDI " + tt.adopted, "2 NOTE adopted by both"},
				// Step has no PEDI value; Natural is the default; a missing
				// _MREL for a family with a WIFE is unknown; an existing PEDI
				// is kept.
				"@I3@": {"1 FAMC @F1@"},
				"@I4@": {"1 FAMC @F1@"},
				"@I5@": {"1 FAMC @F1@"},
				"@I6@": {"1 FAMC @F1@", "2 PEDI foster"},
				// A family naming only a husband needs only _FREL.
				"@I7@": {"1 FAMC @F2@", "2 PEDI " + tt.foster},
			}
			for xref, lines := range want {
				if got := tagLines(converted.XRefMap[xref]); !reflect.DeepEqual(got, lines) {
					t.Errorf("%s Tags = %q, want %q", xref, got, lines)
				}
			}
			indi := converted.GetIndividual("@I2@")
			if got := indi.ChildInFamilies[0].Pedigree; got != tt.adopted {
				t.Errorf("@I2@ Pedigree = %q, want %q", got, tt.adopted)
			}

			var notes []gedcom.ConversionNote
			for _, n := range report.Normalized {
				if strings.HasSuffix(n.Path, "FAMC > PEDI") {
					notes = append(notes, n)
				}
			}
			wantNotes := []gedcom.ConversionNote{
				{
					Path: "Individual @I2@ > FAMC > PEDI", Original: "_FREL Adopted, _MREL adopted", Result: "PEDI " + tt.adopted,
					Reason: "Pedigree written from the vendor relationship tags under @F1@ CHIL",
				},
				{
					Path: "Individual @I7@ > FAMC > PEDI", Original: "_FREL Foster", Result: "PEDI " + tt.foster,
					Reason: "Pedigree written from the vendor relationship tags under @F2@ CHIL",
				},
			}
			if !reflect.DeepEqual(notes, wantNotes) {
				t.Errorf("PEDI notes = %+v, want %+v", notes, wantNotes)
			}
			if got := transformationCount(report, "PEDIGREE_FROM_VENDOR_TAG"); got != 2 {
				t.Errorf("PEDIGREE_FROM_VENDOR_TAG count = %d, want 2", got)
			}
		})
	}
}

func TestConvertMarriedNameToV7(t *testing.T) {
	for _, from := range []gedcom.Version{gedcom.Version55, gedcom.Version551} {
		t.Run(from.String(), func(t *testing.T) {
			src := strings.Replace(relationshipFixture, "%s", from.String(), 1)
			converted, report, err := Convert(decodeText(t, src), gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			want := []string{
				"1 NAME Ann Marie /Smith/ Jr", "2 GIVN Ann Marie", "2 _MARNM Jones",
				"1 NAME Ann Marie /Jones/ Jr", "2 TYPE MARRIED",
				"1 NAME Annie", "2 _MARNM Ann /Brown/",
				"1 NAME Ann /Brown/", "2 TYPE MARRIED",
				"1 FAMS @F1@",
			}
			if got := tagLines(converted.XRefMap["@I1@"]); !reflect.DeepEqual(got, want) {
				t.Errorf("@I1@ Tags =\n%q\nwant\n%q", got, want)
			}
			// @I8@ already has the married name, so none is added.
			if got := len(converted.GetIndividual("@I8@").Names); got != 2 {
				t.Errorf("@I8@ has %d names, want the 2 it had", got)
			}
			names := converted.GetIndividual("@I1@").Names
			if names[1].Type != "MARRIED" || names[1].Surname != "Jones" || names[0].MarriedName != "Jones" {
				t.Errorf("@I1@ names = %+v / %+v, want a MARRIED Jones name after the Smith one", *names[0], *names[1])
			}
			if got := transformationCount(report, "MARRIED_NAME_FROM_VENDOR_TAG"); got != 2 {
				t.Errorf("MARRIED_NAME_FROM_VENDOR_TAG count = %d, want 2", got)
			}
			found := false
			for _, n := range report.Normalized {
				if n.Path == "Individual @I1@ > NAME > _MARNM" && n.Original == "_MARNM Jones" &&
					n.Result == "NAME Ann Marie /Jones/ Jr (TYPE MARRIED)" {
					found = true
				}
			}
			if !found {
				t.Errorf("report lacks the _MARNM Jones note: %+v", report.Normalized)
			}
		})
	}

	// 5.5.1 is not 7.0: _MARNM is left as it is.
	src := strings.Replace(relationshipFixture, "%s", "5.5", 1)
	converted, report, err := Convert(decodeText(t, src), gedcom.Version551)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(converted.GetIndividual("@I1@").Names); got != 2 {
		t.Errorf("5.5 -> 5.5.1: @I1@ has %d names, want 2", got)
	}
	if got := transformationCount(report, "MARRIED_NAME_FROM_VENDOR_TAG"); got != 0 {
		t.Errorf("5.5 -> 5.5.1 reported %d married names", got)
	}
}

// TestConvertVendorRelationshipCorpus pins the counts on the two corpus
// exports the issue names: pres2020.ged records every child as Natural, so
// no PEDI is added, and each of mhftb8-export.ged's 724 _MARNM becomes a
// TYPE MARRIED name.
func TestConvertVendorRelationshipCorpus(t *testing.T) {
	pres := decodeCorpus(t, "../testdata/gedcom-5.5/pres2020.ged")
	converted, report, err := Convert(pres, gedcom.Version70)
	if err != nil {
		t.Fatal(err)
	}
	if got := transformationCount(report, "PEDIGREE_FROM_VENDOR_TAG"); got != 0 {
		t.Errorf("pres2020.ged: %d PEDI added from all-Natural relationships, want 0", got)
	}
	if fam := converted.GetFamily("@F1@"); fam.LinkForChild("@I66@") == nil {
		t.Error("pres2020.ged: @F1@ lost its ChildLink on conversion")
	}

	mh := decodeCorpus(t, "../testdata/edge-cases/mhftb8-export.ged")
	converted, report, err = Convert(mh, gedcom.Version70)
	if err != nil {
		t.Fatal(err)
	}
	if got := transformationCount(report, "MARRIED_NAME_FROM_VENDOR_TAG"); got != 724 {
		t.Errorf("mhftb8-export.ged: %d married names added, want 724", got)
	}
	names := converted.GetIndividual("@I589@").Names
	if len(names) != 2 || names[1].Full != "Margaret Agnes /Gascoigne/" || names[1].Type != "MARRIED" {
		t.Errorf("@I589@ names after conversion = %d, want Franke then MARRIED Margaret Agnes /Gascoigne/", len(names))
	}
}

func decodeCorpus(t *testing.T, path string) *gedcom.Document {
	t.Helper()
	f, err := os.Open(path) // #nosec G304 -- path is a constant in the repo's own testdata
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func transformationCount(report *gedcom.ConversionReport, typ string) int {
	for _, tr := range report.Transformations {
		if tr.Type == typ {
			return tr.Count
		}
	}
	return 0
}
