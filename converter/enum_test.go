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
	// An age that does not parse is invalid in its source version too: kept,
	// a Preserved note, and not data loss.
	tests := []struct {
		name, from, to string
		target         gedcom.Version
		invalid        bool
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
			if hasDataLoss(report, "Enumerated values", "@I1@") {
				t.Errorf("age reported as data loss: %+v", report.DataLoss)
			}
			if got := hasNote(report.Preserved, tt.from, tt.from); got != tt.invalid {
				t.Errorf("Preserved note = %v, want %v: %+v", got, tt.invalid, report.Preserved)
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

// TestConvertEnumerationsSourceInvalid pins that a value its source version
// does not define either is kept and noted, but is not data loss, so a strict
// conversion of a file with free-text ages still succeeds.
func TestConvertEnumerationsSourceInvalid(t *testing.T) {
	opts := converter.DefaultOptions()
	opts.StrictDataLoss = true
	tests := []struct {
		name, vers, record, value string
		target                    gedcom.Version
	}{
		{"free-text age to 7.0", "5.5.1", "1 BIRT\n2 AGE Abt 45", "Abt 45", gedcom.Version70},
		{"unknown SEX to 5.5.1", "7.0", "1 SEX Q", "Q", gedcom.Version551},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decodeEnumInput(t, "0 HEAD\n1 GEDC\n2 VERS "+tt.vers+"\n0 @I1@ INDI\n"+tt.record+"\n0 TRLR\n")
			_, report, err := converter.ConvertWithOptions(doc, tt.target, opts)
			if err != nil {
				t.Fatalf("strict conversion failed: %v (data loss %+v)", err, report.DataLoss)
			}
			if !hasNote(report.Preserved, tt.value, tt.value) {
				t.Errorf("no Preserved note for %q: %+v", tt.value, report.Preserved)
			}
		})
	}
}

// TestConvertEnumerationsSkipExtensions pins that payloads beneath an
// extension tag, or in an extension record, are left alone: the
// specification does not define them.
func TestConvertEnumerationsSkipExtensions(t *testing.T) {
	opts := converter.DefaultOptions()
	opts.StrictDataLoss = true
	doc := decodeEnumInput(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 _FOO\n2 AGE 11m99y\n2 MEDI Unknown thing\n1 BIRT\n2 _X\n3 AGE Unknown\n0 @X1@ _LOC\n1 MEDI Unknown thing\n0 TRLR\n")
	converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version70, opts)
	if err != nil {
		t.Fatalf("strict conversion failed: %v (data loss %+v)", err, report.DataLoss)
	}
	out := encode(t, converted)
	assertBlocks(t, out, []string{
		"1 _FOO\n2 AGE 11m99y\n2 MEDI Unknown thing",
		"2 _X\n3 AGE Unknown",
		"0 @X1@ _LOC\n1 MEDI Unknown thing",
	}, nil)
	for _, n := range report.Normalized {
		if strings.Contains(n.Path, "_FOO") || strings.Contains(n.Path, "_X") || strings.Contains(n.Path, "@X1@") {
			t.Errorf("extension payload rewritten: %+v", n)
		}
	}
}

// TestConvertRecordMediaTypeTyped pins that the typed MediaType follows the
// OBJE.FILE.FORM media type through a conversion in both directions: 7.0
// MEDI becomes 5.5.1 TYPE, which the decoder reads back.
func TestConvertRecordMediaTypeTyped(t *testing.T) {
	down, _, out := convertAndEncode(t, decodeEnumInput(t, enum70Input), gedcom.Version551)
	if !strings.Contains(out, "\n3 TYPE photo\n") {
		t.Fatalf("output lacks TYPE photo:\n%s", out)
	}
	media := down.GetMediaObject("@O1@")
	if media == nil || len(media.Files) != 1 || media.Files[0].MediaType != "photo" {
		t.Fatalf("downgraded media = %+v, want MediaType photo", media)
	}

	up, _, _ := convertAndEncode(t, decodeEnumInput(t, enum551Input), gedcom.Version70)
	media = up.GetMediaObject("@O1@")
	if media == nil || len(media.Files) != 1 || media.Files[0].MediaType != "PHOTO" {
		t.Fatalf("upgraded media = %+v, want MediaType PHOTO", media)
	}
}

// TestEnumTablesRoundTrip walks the vocabulary tables: every 7.0 value with a
// 5.x spelling converts down to it and back up to itself.
func TestEnumTablesRoundTrip(t *testing.T) {
	cases := []struct{ record, v7 string }{
		{"1 SEX %s", "M"}, {"1 SEX %s", "F"}, {"1 SEX %s", "U"},
		{"1 RESN %s", "CONFIDENTIAL"}, {"1 RESN %s", "LOCKED"}, {"1 RESN %s", "PRIVACY"},
	}
	for _, v := range []string{"AKA", "BIRTH", "IMMIGRANT", "MAIDEN", "MARRIED"} {
		cases = append(cases, struct{ record, v7 string }{"1 NAME A /B/\n2 TYPE %s", v})
	}
	for _, v := range []string{"ADOPTED", "BIRTH", "FOSTER", "SEALING"} {
		cases = append(cases, struct{ record, v7 string }{"1 FAMC @F1@\n2 PEDI %s", v})
	}
	for _, v := range []string{"CHALLENGED", "DISPROVEN", "PROVEN"} {
		cases = append(cases, struct{ record, v7 string }{"1 FAMC @F1@\n2 STAT %s", v})
	}
	for _, v := range []string{"HUSB", "WIFE", "BOTH"} {
		cases = append(cases, struct{ record, v7 string }{"1 FAMC @F1@\n2 ADOP %s", v})
	}
	// Not INIL: 5.5.1 does not define it, so the downgrade drops it (#705).
	for _, ord := range []string{"BAPL", "CONL", "ENDL", "SLGC"} {
		for _, v := range []string{"BIC", "CANCELED", "CHILD", "COMPLETED", "DNS", "DNS_CAN", "EXCLUDED", "INFANT", "PRE_1970", "STILLBORN", "SUBMITTED", "UNCLEARED"} {
			cases = append(cases, struct{ record, v7 string }{"1 " + ord + "\n2 STAT %s", v})
		}
	}
	for _, v := range []string{"AUDIO", "BOOK", "CARD", "ELECTRONIC", "FICHE", "FILM", "MAGAZINE", "MANUSCRIPT", "MAP", "NEWSPAPER", "PHOTO", "TOMBSTONE", "VIDEO"} {
		cases = append(cases, struct{ record, v7 string }{"1 SEX M\n0 @O1@ OBJE\n1 FILE a.jpg\n2 FORM image/jpeg\n3 MEDI %s", v})
	}
	for _, c := range cases {
		line := strings.Replace(c.record, "%s", c.v7, 1)
		doc := decodeEnumInput(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n"+line+"\n0 @F1@ FAM\n0 @S1@ SOUR\n0 TRLR\n")
		down, report, _ := convertAndEncode(t, doc, gedcom.Version551)
		for _, d := range report.DataLoss {
			if d.Feature == "Enumerated values" {
				t.Errorf("%q has no 5.5.1 spelling: %+v", line, report.Preserved)
			}
		}
		_, _, out := convertAndEncode(t, down, gedcom.Version70)
		if !strings.Contains(out, line+"\n") {
			t.Errorf("%q did not round-trip through 5.5.1:\n%s", line, out)
		}
	}
}
