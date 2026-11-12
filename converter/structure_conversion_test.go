package converter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Tests for issue #705: what one version defines and the other does not is
// mapped or dropped, with a report entry, and a downgrade carries the header
// 5.5.x requires.

const upgrade551 = `0 HEAD
1 SOUR My System
1 SUBM @U1@
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 LANG English
1 FILE family.ged
0 @U1@ SUBM
1 NAME Submitter
1 LANG Klingon
1 RFN 5431
0 @I1@ INDI
1 NAME John /Doe/
1 AFN 123456789
1 RFN Resource:5431
1 RIN 9876
1 _CUSTOM kept
2 RIN 1
1 OBJE
2 FILE photo.jpg
3 FORM jpg
2 TITL A photo
1 BIRT
2 OBJE
3 FILE birth.jpg
4 FORM jpg
0 @O1@ OBJE
1 FILE x.jpg
2 FORM jpg
0 TRLR
`

func TestUpgrade551MapsVersionForeignStructures(t *testing.T) {
	_, report, out := convertText(t, upgrade551, gedcom.Version70)

	for _, want := range []string{
		"1 LANG en\n",
		"0 @U1@ SUBM\n1 NAME Submitter\n1 LANG Klingon\n1 EXID 5431\n2 TYPE https://gedcom.io/terms/v7/RFN\n",
		"1 EXID 123456789\n2 TYPE https://gedcom.io/terms/v7/AFN\n" +
			"1 EXID 5431\n2 TYPE https://gedcom.io/terms/v7/RFN#Resource\n" +
			"1 EXID 9876\n2 TYPE https://gedcom.io/terms/v7/RIN#My%20System\n",
		"1 _CUSTOM kept\n2 RIN 1\n", // under an extension: not governed, kept
		"1 OBJE @O2@\n1 BIRT\n2 OBJE @O3@\n",
		"0 @O2@ OBJE\n1 FILE photo.jpg\n2 FORM jpg\n2 TITL A photo\n",
		"0 @O3@ OBJE\n1 FILE birth.jpg\n2 FORM jpg\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	for _, gone := range []string{"1 AFN", "1 RIN", "RFN Resource", "1 FILE family.ged"} {
		if strings.Contains(out, gone) {
			t.Errorf("output keeps %q:\n%s", gone, out)
		}
	}

	if !hasDropped(report, "Header > FILE") {
		t.Errorf("HEAD.FILE drop not reported: %+v %+v", report.Dropped, report.DataLoss)
	}
	notes := map[string]string{}
	for _, n := range report.Normalized {
		notes[n.Path+" "+n.Original] = n.Result
	}
	for key, result := range map[string]string{
		"Individual @I1@ > AFN AFN 123456789":                         "EXID 123456789 (TYPE https://gedcom.io/terms/v7/AFN)",
		"Individual @I1@ > RIN RIN 9876":                              "EXID 9876 (TYPE https://gedcom.io/terms/v7/RIN#My%20System)",
		"Individual @I1@ > OBJE OBJE (inline multimedia link)":        "OBJE @O2@",
		"Individual @I1@ > BIRT > OBJE OBJE (inline multimedia link)": "OBJE @O3@",
		"Header > LANG English":                                       "en",
	} {
		if notes[key] != result {
			t.Errorf("Normalized %q = %q, want %q (all: %v)", key, notes[key], result, notes)
		}
	}
	if !hasApproximated(report, "Submitter @U1@ > LANG", "Klingon", "Klingon") {
		t.Errorf("no Approximated note for the unmapped LANG: %+v", report.Approximated)
	}
}

// TestUpgrade55PromotesInlineMediaLink: the 5.5 layout puts FORM beside FILE
// and allows NOTE; FORM and TITL go under the FILE, NOTE stays on the record.
func TestUpgrade55PromotesInlineMediaLink(t *testing.T) {
	const doc55 = `0 HEAD
1 GEDC
2 VERS 5.5
2 FORM LINEAGE-LINKED
1 CHAR ASCII
0 @I1@ INDI
1 OBJE
2 FORM bmp
2 TITL Portrait
2 FILE portrait.bmp
2 NOTE A note
1 OBJE
2 FORM bmp
2 BLOB
3 CONT abc
0 TRLR
`
	converted, report, out := convertText(t, doc55, gedcom.Version70)
	if want := "0 @O1@ OBJE\n1 FILE portrait.bmp\n2 FORM bmp\n2 TITL Portrait\n1 NOTE A note\n"; !strings.Contains(out, want) {
		t.Errorf("output lacks %q:\n%s", want, out)
	}
	// A link with no FILE has nothing a 7.0 OBJE record could hold, so it is
	// not promoted; its 5.5 FORM has no 7.0 place and is dropped (#794).
	if !strings.Contains(out, "1 OBJE @O1@\n1 OBJE\n2 BLOB\n") {
		t.Errorf("FILE-less link not left in place:\n%s", out)
	}
	if !hasDropped(report, "Individual @I1@ > OBJE > FORM") {
		t.Errorf("FILE-less link's FORM drop not reported: %+v", report.Dropped)
	}
	if converted.XRefMap["@O1@"] == nil || converted.XRefMap["@O1@"].Type != gedcom.RecordTypeMedia {
		t.Error("promoted record not in XRefMap")
	}
	if media, ok := converted.XRefMap["@O1@"].GetMediaObject(); !ok || len(media.Files) != 1 || media.Files[0].FileRef != "portrait.bmp" {
		t.Errorf("promoted record's entity = %+v", converted.XRefMap["@O1@"].Entity)
	}
}

func TestFreeXRefSkipsUsed(t *testing.T) {
	used := map[string]bool{"@O1@": true, "@O2@": true}
	if xref, next := freeXRef(used, "O", 1); xref != "@O3@" || next != 4 {
		t.Errorf("freeXRef = %q, %d", xref, next)
	}
}

func TestDowngradeAddsRequiredHeader(t *testing.T) {
	t.Run("no submitter record", func(t *testing.T) {
		converted, report, out := convertText(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @SUBM1@ INDI\n0 TRLR\n", gedcom.Version551)
		if !strings.Contains(out, "1 SOUR gedcom-go\n") || !strings.Contains(out, "1 SUBM @SUBM2@\n") ||
			!strings.Contains(out, "0 @SUBM2@ SUBM\n1 NAME Unknown\n") {
			t.Errorf("output lacks the required header:\n%s", out)
		}
		if s := converted.GetSubmitter("@SUBM2@"); s == nil || s.Name != "Unknown" {
			t.Errorf("added submitter = %+v", s)
		}
		if converted.Header.Submitter != "@SUBM2@" || converted.Header.SourceSystem != "gedcom-go" {
			t.Errorf("typed header = %q %q", converted.Header.Submitter, converted.Header.SourceSystem)
		}
		for _, typ := range []string{"HEAD_SOUR_ADDED", "HEAD_SUBM_ADDED", "SUBM_RECORD_ADDED"} {
			if !hasTransformation(report, typ) {
				t.Errorf("no %s transformation: %+v", typ, report.Transformations)
			}
		}
	})
	t.Run("existing submitter and source", func(t *testing.T) {
		_, report, out := convertText(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SOUR APP\n0 @U9@ SUBM\n1 NAME Me\n0 TRLR\n", gedcom.Version55)
		if !strings.Contains(out, "1 SOUR APP\n") || !strings.Contains(out, "1 SUBM @U9@\n") || strings.Contains(out, "NAME Unknown") {
			t.Errorf("output:\n%s", out)
		}
		if hasTransformation(report, "HEAD_SOUR_ADDED") || hasTransformation(report, "SUBM_RECORD_ADDED") {
			t.Errorf("reported additions it did not make: %+v", report.Transformations)
		}
	})
	t.Run("header without tags", func(t *testing.T) {
		doc := &gedcom.Document{Header: &gedcom.Header{Version: gedcom.Version70}}
		converted, _, err := Convert(doc, gedcom.Version551)
		if err != nil {
			t.Fatal(err)
		}
		if converted.Header.SourceSystem != "gedcom-go" || converted.GetSubmitter(converted.Header.Submitter) == nil {
			t.Errorf("typed header = %+v", converted.Header)
		}
	})
}

func TestConvertLanguage(t *testing.T) {
	tests := []struct {
		value    string
		toV7     bool
		want     string
		noteKind string // "", "normalized", "approximated"
	}{
		{"English", true, "en", "normalized"},
		{"english", true, "en", "normalized"},
		{"en-GB", true, "en-GB", ""},
		{"Klingon", true, "Klingon", "approximated"},
		{"", true, "", ""},
		{"de", false, "German", "normalized"},
		{"en-US", false, "English", "approximated"},
		{"English", false, "English", ""},
		{"tlh", false, "tlh", "approximated"},
		{"x-bad!", false, "x-bad!", "approximated"},
	}
	for _, tt := range tests {
		report := &gedcom.ConversionReport{}
		if got := convertLanguage(tt.value, tt.toV7, "Header > LANG", report); got != tt.want {
			t.Errorf("convertLanguage(%q, %v) = %q, want %q", tt.value, tt.toV7, got, tt.want)
		}
		kind := ""
		switch {
		case len(report.Normalized) == 1 && len(report.Approximated) == 0:
			kind = "normalized"
		case len(report.Approximated) == 1 && len(report.Normalized) == 0:
			kind = "approximated"
		case len(report.Normalized)+len(report.Approximated) != 0:
			kind = "both"
		}
		if kind != tt.noteKind {
			t.Errorf("convertLanguage(%q, %v) reported %q, want %q", tt.value, tt.toV7, kind, tt.noteKind)
		}
	}
}

func TestLanguageNamesRoundTrip(t *testing.T) {
	for name, tag := range languageTags {
		if languageNames[tag] != name {
			t.Errorf("%s -> %s -> %s", name, tag, languageNames[tag])
		}
		if !looksLikeLanguageTag(tag) {
			t.Errorf("%s: %q is not tag-shaped", name, tag)
		}
	}
}

// TestDropForeignStructuresWalk covers the walk's edges: nil tags, an
// extension's subtree, CONT, a record type the source does not define, and
// a record with no Tags.
func TestDropForeignStructuresWalk(t *testing.T) {
	indi := &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Tags: []*gedcom.Tag{
		{Level: 1, Tag: "NAME", Value: "A /B/"},
		nil,
		{Level: 1, Tag: "INIL"},
		{Level: 2, Tag: "DATE", Value: "2000"},
		nil,
		{Level: 1, Tag: "_EXT"},
		{Level: 2, Tag: "SDATE"},
		{Level: 1, Tag: "NOTE", Value: "x"},
		{Level: 2, Tag: "CONT", Value: "y"},
		{Level: 2, Tag: "MIME", Value: "text/plain"},
	}}
	custom := &gedcom.Record{XRef: "@X1@", Type: "_LOC", Tags: []*gedcom.Tag{{Level: 1, Tag: "INIL"}}}
	bare := &gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{}}
	doc := &gedcom.Document{Records: []*gedcom.Record{indi, custom, bare, nil}}
	report := &gedcom.ConversionReport{}
	dropForeignStructures(doc, foreign70To551, report, gedcom.Version551)

	var got []string
	for _, tag := range indi.Tags {
		if tag == nil {
			got = append(got, "<nil>")
			continue
		}
		got = append(got, tag.Tag)
	}
	if want := "NAME <nil> <nil> _EXT SDATE NOTE CONT"; strings.Join(got, " ") != want {
		t.Errorf("tags = %v, want %s", got, want)
	}
	if len(custom.Tags) != 1 {
		t.Errorf("extension record walked: %v", custom.Tags)
	}
	if !hasDropped(report, "Individual @I1@ > INIL") || !hasDropped(report, "Individual @I1@ > NOTE > MIME") {
		t.Errorf("Dropped = %+v", report.Dropped)
	}
	if !hasDataLoss(report, "INDI.INIL structures", "@I1@") || !hasDataLoss(report, "NOTE.MIME structures", "@I1@") {
		t.Errorf("DataLoss = %+v", report.DataLoss)
	}
}

func hasApproximated(report *gedcom.ConversionReport, path, original, result string) bool {
	for _, n := range report.Approximated {
		if n.Path == path && n.Original == original && n.Result == result {
			return true
		}
	}
	return false
}

func hasTransformation(report *gedcom.ConversionReport, typ string) bool {
	for _, tr := range report.Transformations {
		if tr.Type == typ {
			return true
		}
	}
	return false
}

// TestUpgradeInlineMediaLinkSkipsDanglingXRef: a promoted OBJE record does
// not take the XRef a dangling pointer names, which would re-bind it.
func TestUpgradeInlineMediaLinkSkipsDanglingXRef(t *testing.T) {
	const doc = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 OBJE @O1@\n1 OBJE\n2 FILE photo.jpg\n3 FORM jpg\n0 TRLR\n"
	_, _, out := convertText(t, doc, gedcom.Version70)
	if want := "0 @I1@ INDI\n1 OBJE @O1@\n1 OBJE @O2@\n0 @O2@ OBJE\n1 FILE photo.jpg\n"; !strings.Contains(out, want) {
		t.Errorf("output lacks %q:\n%s", want, out)
	}
}

// TestDowngrade70To55DemotesMediaRecords is issue #794's OBJE criterion: a
// 5.5 OBJE record cannot hold a file, so each pointer becomes one inline link
// per FILE, carrying its FORM, TITL and the record's NOTE, and what no link
// holds is reported.
func TestDowngrade70To55DemotesMediaRecords(t *testing.T) {
	const doc70 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 OBJE @O1@
2 TITL Pointer title
2 _EXT kept
1 BIRT
2 OBJE @O1@
0 @O1@ OBJE
1 FILE a.jpg
2 FORM image/jpeg
3 MEDI PHOTO
2 TITL File title
2 _FX kept under FILE
1 FILE b.pdf
2 FORM application/pdf
1 REFN 7
1 NOTE A note
0 @O2@ OBJE
1 FILE orphan.jpg
2 FORM image/jpeg
0 @O3@ OBJE
1 _NOFILE x
0 @I2@ INDI
1 OBJE @O3@
0 TRLR
`
	converted, report, out := convertText(t, doc70, gedcom.Version55)

	for _, want := range []string{
		"0 @I1@ INDI\n" +
			"1 OBJE\n2 FORM image/jpeg\n2 TITL Pointer title\n2 FILE a.jpg\n3 _FX kept under FILE\n2 NOTE A note\n2 _EXT kept\n" +
			"1 OBJE\n2 FORM application/pdf\n2 TITL Pointer title\n2 FILE b.pdf\n2 NOTE A note\n" +
			"1 BIRT\n" +
			"2 OBJE\n3 FORM image/jpeg\n3 TITL File title\n3 FILE a.jpg\n4 _FX kept under FILE\n3 NOTE A note\n" +
			"2 OBJE\n3 FORM application/pdf\n3 FILE b.pdf\n3 NOTE A note\n",
		// A record with no FILE is not demoted.
		"1 OBJE @O3@\n", "0 @O3@ OBJE\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	for _, gone := range []string{"0 @O1@ OBJE", "0 @O2@ OBJE", "orphan.jpg", "MEDI", "REFN"} {
		if strings.Contains(out, gone) {
			t.Errorf("output keeps %q:\n%s", gone, out)
		}
	}
	if converted.XRefMap["@O1@"] != nil || converted.XRefMap["@O2@"] != nil {
		t.Error("demoted records left in XRefMap")
	}
	if media := converted.GetIndividual("@I1@").Media; len(media) != 2 || media[0].MediaXRef != "" || media[0].Title != "Pointer title" {
		t.Errorf("Individual.Media = %+v, want two inline links", media)
	}

	if !hasNormalized(report, "Individual @I1@ > BIRT > OBJE", "OBJE @O1@", "OBJE (inline multimedia link)") {
		t.Errorf("no Normalized note for the event's pointer: %+v", report.Normalized)
	}
	for _, path := range []string{"MediaObject @O1@ > REFN", "MediaObject @O1@ > FILE > FORM > MEDI", "MediaObject @O2@"} {
		if !hasDropped(report, path) {
			t.Errorf("no Dropped note for %q: %+v", path, report.Dropped)
		}
	}
	if !hasDataLoss(report, "OBJE record structures", "@O1@") || !hasDataLoss(report, "Unlinked OBJE records", "@O2@") {
		t.Errorf("DataLoss = %+v", report.DataLoss)
	}
	if !hasTransformation(report, "OBJE_RECORD_TO_INLINE") {
		t.Errorf("no OBJE_RECORD_TO_INLINE transformation: %+v", report.Transformations)
	}
	// The BIRT link holds the file's TITL, so it is not lost.
	if hasDropped(report, "MediaObject @O1@ > FILE > TITL") {
		t.Errorf("file TITL reported lost though a link holds it: %+v", report.Dropped)
	}
	// 5.5 has no FORM.TYPE, so the record's MEDI is lost, not renamed first.
	for _, n := range report.Normalized {
		if strings.HasPrefix(n.Path, "MediaObject @O1@ > FILE > FORM") {
			t.Errorf("record MEDI reported as translated before it is dropped: %+v", n)
		}
	}
}

// TestDowngrade70To55ReportsOverriddenFileTitle: when every pointer carries
// its own TITL, no inline link holds the file's TITL, so it is reported lost.
func TestDowngrade70To55ReportsOverriddenFileTitle(t *testing.T) {
	const doc70 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 OBJE @O1@
2 TITL Pointer title
0 @O1@ OBJE
1 FILE a.jpg
2 FORM image/jpeg
2 TITL File title
0 TRLR
`
	_, report, out := convertText(t, doc70, gedcom.Version55)
	if strings.Contains(out, "File title") {
		t.Fatalf("output keeps the file TITL:\n%s", out)
	}
	if !hasDropped(report, "MediaObject @O1@ > FILE > TITL") {
		t.Errorf("no Dropped note for the overridden file TITL: %+v", report.Dropped)
	}
	if !hasDataLoss(report, "OBJE record structures", "@O1@") {
		t.Errorf("DataLoss = %+v", report.DataLoss)
	}
}

// TestDowngrade70To55KeepsOversizedMediaRecord: a record whose inline links
// would copy more than maxInlineMediaTags tags (pointers * files) is kept as
// it is and reported, while a small record after it is still demoted.
func TestDowngrade70To55KeepsOversizedMediaRecord(t *testing.T) {
	const files, pointers = 1025, 400 // 400 * 1025 * 3 tags > maxInlineMediaTags
	var b strings.Builder
	b.WriteString("0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n")
	for range pointers {
		b.WriteString("1 OBJE @O1@\n")
	}
	b.WriteString("1 OBJE @O2@\n0 @O1@ OBJE\n")
	for range files {
		b.WriteString("1 FILE a.jpg\n2 FORM image/jpeg\n")
	}
	b.WriteString("0 @O2@ OBJE\n1 FILE b.jpg\n2 FORM image/jpeg\n0 TRLR\n")

	converted, report, out := convertText(t, b.String(), gedcom.Version55)
	if converted.XRefMap["@O1@"] == nil || !strings.Contains(out, "1 OBJE @O1@\n") {
		t.Fatal("oversized record @O1@ was demoted")
	}
	if converted.XRefMap["@O2@"] != nil || !strings.Contains(out, "1 OBJE\n2 FORM image/jpeg\n2 FILE b.jpg\n") {
		t.Errorf("small record @O2@ was not demoted")
	}
	if !hasDataLoss(report, "Oversized OBJE records", "@O1@") || hasDataLoss(report, "Oversized OBJE records", "@O2@") {
		t.Errorf("DataLoss = %+v", report.DataLoss)
	}
	found := false
	for _, n := range report.Preserved {
		found = found || n.Path == "MediaObject @O1@"
	}
	if !found {
		t.Errorf("no Preserved note for @O1@: %+v", report.Preserved)
	}
}

// TestDowngrade70To55KeepsMediaRecordWithTypedPointer: a record with no
// Tags is written from its entity, whose pointer the demotion cannot
// rewrite, so the OBJE record it points to is kept rather than left dangling.
func TestDowngrade70To55KeepsMediaRecordWithTypedPointer(t *testing.T) {
	doc, err := decoder.Decode(strings.NewReader(`0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 OBJE @O1@
0 @O1@ OBJE
1 FILE a.jpg
2 FORM image/jpeg
0 @O2@ OBJE
1 FILE b.jpg
2 FORM image/jpeg
0 TRLR
`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	built := &gedcom.Record{XRef: "@I2@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
		XRef: "@I2@", Media: []*gedcom.MediaLink{{MediaXRef: "@O2@"}},
	}}
	doc.Records = append(doc.Records, built)
	doc.XRefMap["@I2@"] = built

	converted, report, err := Convert(doc, gedcom.Version55)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if converted.XRefMap["@O2@"] == nil || !strings.Contains(out, "0 @O2@ OBJE\n") || !strings.Contains(out, "1 OBJE @O2@\n") {
		t.Errorf("record @O2@ that a built record points to was not kept:\n%s", out)
	}
	if converted.XRefMap["@O1@"] != nil || strings.Contains(out, "0 @O1@ OBJE") {
		t.Errorf("record @O1@ was not demoted:\n%s", out)
	}
	if !hasDataLoss(report, "OBJE records with typed pointers", "@O2@") || hasDataLoss(report, "OBJE records with typed pointers", "@O1@") {
		t.Errorf("DataLoss = %+v", report.DataLoss)
	}
}

// TestUpgrade55NestsMediaRecordForms: a 5.5 OBJE record written with a FILE
// beside its FORM and TITL (as Family Historian does) keeps both, under the
// FILE, rather than losing them as 5.5-only record structures (#794).
func TestUpgrade55NestsMediaRecordForms(t *testing.T) {
	const doc55 = `0 HEAD
1 GEDC
2 VERS 5.5
2 FORM LINEAGE-LINKED
1 CHAR ASCII
0 @I1@ INDI
1 OBJE @O1@
0 @O1@ OBJE
1 FORM jpeg
1 TITL Portrait
1 FILE photo.jpg
0 @O2@ OBJE
1 FORM bmp
1 BLOB
2 CONT abc
0 TRLR
`
	_, report, out := convertText(t, doc55, gedcom.Version70)
	if want := "0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM jpeg\n2 TITL Portrait\n"; !strings.Contains(out, want) {
		t.Errorf("output lacks %q:\n%s", want, out)
	}
	if !hasNormalized(report, "MediaObject @O1@ > FILE", "OBJE record FORM/TITL beside FILE", "FORM/TITL under FILE") {
		t.Errorf("no Normalized note for the move: %+v", report.Normalized)
	}
	// A true 5.5 record embeds a BLOB, which 7.0 has no form for.
	if strings.Contains(out, "BLOB") || !hasDropped(report, "MediaObject @O2@ > BLOB") {
		t.Errorf("BLOB not dropped and reported: %+v\n%s", report.Dropped, out)
	}
}
