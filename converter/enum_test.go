package converter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #704: the converter translates enumerated payloads between the
// GEDCOM 5.5.1 and 7.0 vocabularies.

const enum551Input = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
2 TYPE aka
1 NAME Jack /Doe/
2 TYPE stage
1 SEX m
1 RESN privacy
1 FAMC @F1@
2 PEDI adopted
2 STAT challenged
1 BIRT
2 AGE <8
1 DEAT
2 AGE stillborn
1 ASSO @I2@
2 RELA Godparent
1 ASSO @I2@
2 RELA Best man
1 ASSO @I2@
2 RELA friend
1 BAPL
2 STAT PRE-1970
1 SOUR @S1@
2 EVEN BIRT
3 ROLE (Midwife)
0 @I2@ INDI
1 NAME Jane /Roe/
0 @F1@ FAM
1 CHIL @I1@
0 @S1@ SOUR
1 TITL Register
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM JPG
3 TYPE photo
0 TRLR
`

const enum70Input = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Doe/
2 TYPE AKA
1 NAME Jack /Doe/
2 TYPE OTHER
3 PHRASE stage
1 SEX X
1 RESN CONFIDENTIAL
1 FAMC @F1@
2 PEDI OTHER
3 PHRASE guardianship
1 FAMC @F2@
2 PEDI ADOPTED
2 STAT CHALLENGED
1 BIRT
2 AGE < 8y
3 PHRASE Child
1 DEAT
2 AGE > 2w 1d
1 ASSO @I2@
2 ROLE WITN
1 ASSO @I2@
2 ROLE OTHER
3 PHRASE Best man
1 BAPL
2 STAT DNS_CAN
3 DATE 1 JAN 2000
1 SOUR @S1@
2 EVEN BIRT
3 ROLE CLERGY
0 @I2@ INDI
1 NAME Jane /Roe/
0 @F1@ FAM
1 CHIL @I1@
0 @F2@ FAM
1 CHIL @I1@
0 @S1@ SOUR
1 TITL Register
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM image/jpeg
3 MEDI PHOTO
0 TRLR
`

func decodeEnumInput(t *testing.T, text string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(bytes.NewReader([]byte(text)))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

// assertBlocks checks that each of want (consecutive output lines) appears in
// out, and none of absent does.
func assertBlocks(t *testing.T, out string, want, absent []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w+"\n") {
			t.Errorf("output lacks %q", w)
		}
	}
	for _, a := range absent {
		if strings.Contains(out, a+"\n") {
			t.Errorf("output still has %q", a)
		}
	}
	if t.Failed() {
		t.Logf("output:\n%s", out)
	}
}

func TestConvertEnumerationsTo70(t *testing.T) {
	converted, report, out := convertAndEncode(t, decodeEnumInput(t, enum551Input), gedcom.Version70)

	assertBlocks(t, out, []string{
		"1 NAME John /Doe/\n2 TYPE AKA",
		"1 NAME Jack /Doe/\n2 TYPE OTHER\n3 PHRASE stage",
		"1 SEX M",
		"1 RESN PRIVACY",
		"2 PEDI ADOPTED",
		"2 STAT CHALLENGED",
		"1 BIRT\n2 AGE < 8y",
		"1 DEAT\n2 AGE 0y\n3 PHRASE stillborn",
		"1 ASSO @I2@\n2 ROLE GODP\n3 PHRASE Godparent",
		"1 ASSO @I2@\n2 ROLE OTHER\n3 PHRASE Best man",
		"1 ASSO @I2@\n2 ROLE FRIEND",
		"1 BAPL\n2 STAT PRE_1970",
		"2 EVEN BIRT\n3 ROLE OTHER\n4 PHRASE Midwife",
		"3 MEDI PHOTO",
	}, []string{
		"2 RELA Godparent", "3 PHRASE friend", "2 TYPE aka", "1 SEX m", "2 PEDI adopted", "3 TYPE photo",
	})

	// The typed model is re-derived from the converted tags.
	indi := converted.GetIndividual("@I1@")
	if indi == nil || len(indi.Associations) != 3 {
		t.Fatalf("converted associations = %+v", indi)
	}
	if a := indi.Associations[1]; a.Role != "OTHER" || a.RolePhrase != "Best man" {
		t.Errorf("association = %q + %q, want OTHER + Best man", a.Role, a.RolePhrase)
	}

	if hasDataLoss(report, "Enumerated values", "@I1@") {
		t.Errorf("every value maps to 7.0, but data loss was reported: %+v", report.DataLoss)
	}
	assertNormalized(t, report, "Individual @I1@ > ASSO > ROLE", "RELA Best man", "ROLE OTHER + PHRASE Best man")
	assertNormalized(t, report, "Individual @I1@ > FAMC > PEDI", "adopted", "ADOPTED")
}

func TestConvertEnumerationsTo551(t *testing.T) {
	_, report, out := convertAndEncode(t, decodeEnumInput(t, enum70Input), gedcom.Version551)

	assertBlocks(t, out, []string{
		"1 NAME John /Doe/\n2 TYPE aka",
		"1 NAME Jack /Doe/\n2 TYPE stage",
		"1 SEX X",
		"1 RESN confidential",
		"1 FAMC @F1@\n2 PEDI OTHER",
		"2 PEDI adopted",
		"2 STAT challenged",
		"1 BIRT\n2 AGE Child",
		"1 DEAT\n2 AGE >15d",
		"1 ASSO @I2@\n2 RELA witness",
		"1 ASSO @I2@\n2 RELA Best man",
		"1 BAPL\n2 STAT DNS/CAN",
		"2 EVEN BIRT\n3 ROLE (clergy)",
		"3 TYPE photo",
	}, []string{
		"2 ROLE WITN", "3 PHRASE Best man", "3 PHRASE stage", "3 MEDI PHOTO", "3 PHRASE guardianship", "3 PHRASE Child",
	})

	// SEX X and PEDI OTHER have no 5.5.1 value: kept, and reported.
	if !hasDataLoss(report, "Enumerated values", "@I1@") {
		t.Errorf("unmapped values not reported as data loss: %+v", report.DataLoss)
	}
	for _, value := range []string{"X", "OTHER"} {
		if !hasNote(report.Preserved, value, value) {
			t.Errorf("no Preserved note for %s: %+v", value, report.Preserved)
		}
	}
	// The PEDI OTHER phrase is not folded in, so it is reported as dropped.
	if !hasDataLoss(report, "PHRASE tags", "@I1@") {
		t.Errorf("unfolded PHRASE not reported: %+v", report.DataLoss)
	}
	assertNormalized(t, report, "Individual @I1@ > ASSO > RELA", "ROLE WITN", "RELA witness")
}

// TestConvertEnumerationsStrict pins that a value with no target equivalent
// fails a strict conversion, and that a fully mappable one does not.
func TestConvertEnumerationsStrict(t *testing.T) {
	opts := converter.DefaultOptions()
	opts.StrictDataLoss = true

	doc := decodeEnumInput(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 SEX X\n0 TRLR\n")
	if _, report, err := converter.ConvertWithOptions(doc, gedcom.Version551, opts); err == nil || report.Success {
		t.Errorf("strict SEX X downgrade: err = %v, Success = %v; want an error", err, report != nil && report.Success)
	}

	doc = decodeEnumInput(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 SEX F\n1 FAMC @F1@\n2 PEDI BIRTH\n0 @F1@ FAM\n0 TRLR\n")
	if _, report, err := converter.ConvertWithOptions(doc, gedcom.Version551, opts); err != nil || !report.Success {
		t.Errorf("strict mappable downgrade failed: %v", err)
	}
}

func TestConvertAgeSyntax(t *testing.T) {
	tests := []struct {
		name, from, to string
		target         gedcom.Version
		unmapped       bool
	}{
		{"bare count", "0", "2 AGE 0y", gedcom.Version70, false},
		{"spaced unit", "> 0 Y", "2 AGE > 0y", gedcom.Version70, false},
		{"reordered", "30d11m99y", "2 AGE 99y 11m 30d", gedcom.Version70, false},
		{"keyword", "INFANT", "2 AGE < 1y\n3 PHRASE INFANT", gedcom.Version70, false},
		{"duplicate unit", "1y 2y", "2 AGE 1y 2y", gedcom.Version70, true},
		{"not an age", "about 5", "2 AGE about 5", gedcom.Version70, true},
		{"bound only", "<", "2 AGE <", gedcom.Version70, true},
		{"bound to 5.5.1", "< 99y 11m", "2 AGE <99y 11m", gedcom.Version551, false},
		{"weeks to 5.5.1", "3w", "2 AGE 21d", gedcom.Version551, false},
		{"garbage to 5.5.1", "old", "2 AGE old", gedcom.Version551, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vers := "5.5.1"
			if tt.target == gedcom.Version551 {
				vers = "7.0"
			}
			doc := decodeEnumInput(t, "0 HEAD\n1 GEDC\n2 VERS "+vers+"\n0 @I1@ INDI\n1 BIRT\n2 AGE "+tt.from+"\n0 TRLR\n")
			_, report, out := convertAndEncode(t, doc, tt.target)
			if !strings.Contains(out, "1 BIRT\n"+tt.to+"\n") {
				t.Errorf("output lacks %q:\n%s", tt.to, out)
			}
			if got := hasDataLoss(report, "Enumerated values", "@I1@"); got != tt.unmapped {
				t.Errorf("data loss reported = %v, want %v", got, tt.unmapped)
			}
		})
	}
}

func assertNormalized(t *testing.T, report *gedcom.ConversionReport, path, original, result string) {
	t.Helper()
	for _, n := range report.Normalized {
		if n.Path == path && n.Original == original && n.Result == result {
			return
		}
	}
	t.Errorf("no Normalized note %s: %q -> %q in %+v", path, original, result, report.Normalized)
}
