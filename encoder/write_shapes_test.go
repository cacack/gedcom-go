package encoder

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #740: shapes the entity encoder wrote invalid for the output version
// -- ASSO ROLE in 5.5.x, STAT without its DATE, bare OBJE and ADOP -- and the
// shared-note substructures #711 stopped duplicating.

// encodeIndividual encodes one hand-built individual with header version
// version, writing it from its entity.
func encodeIndividual(t *testing.T, version gedcom.Version, indi *gedcom.Individual) string {
	t.Helper()
	indi.XRef = "@I1@"
	doc := &gedcom.Document{
		Header:  &gedcom.Header{Version: version},
		Records: []*gedcom.Record{{Type: gedcom.RecordTypeIndividual, XRef: "@I1@", Entity: indi}},
	}
	return encodeToString(t, doc)
}

func TestAssociationWrittenForOutputVersion(t *testing.T) {
	tests := []struct {
		version    gedcom.Version
		recordType string
		want       string
		absent     []string
	}{
		{gedcom.Version55, "", "1 ASSO @I2@\n2 TYPE INDI\n2 RELA Godfather\n", []string{"ROLE"}},
		{gedcom.Version55, "FAM", "1 ASSO @I2@\n2 TYPE FAM\n2 RELA Godfather\n", []string{"ROLE"}},
		{gedcom.Version551, "FAM", "1 ASSO @I2@\n2 RELA Godfather\n", []string{"ROLE", "TYPE"}},
		{gedcom.Version70, "FAM", "1 ASSO @I2@\n2 ROLE Godfather\n", []string{"RELA", "TYPE"}},
		{"", "", "1 ASSO @I2@\n2 ROLE Godfather\n", []string{"RELA", "TYPE"}},
		{"", "FAM", "1 ASSO @I2@\n2 TYPE FAM\n2 ROLE Godfather\n", []string{"RELA"}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q/%q", tt.version, tt.recordType), func(t *testing.T) {
			out := encodeIndividual(t, tt.version, &gedcom.Individual{
				Associations: []*gedcom.Association{{IndividualXRef: "@I2@", Role: "Godfather", RecordType: tt.recordType}},
			})
			if !strings.Contains(out, tt.want) {
				t.Errorf("output missing %q:\n%s", tt.want, out)
			}
			for _, tag := range tt.absent {
				if strings.Contains(out, " "+tag+" ") {
					t.Errorf("output has %s:\n%s", tag, out)
				}
			}
		})
	}
}

// TestAssociationRELARoundTrips551 is the reproduction from #740: a 5.5.1
// RELA came back as ROLE once Record.Tags was cleared.
func TestAssociationRELARoundTrips551(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME A /B/\n1 ASSO @I2@\n2 RELA Godfather\n" +
		"0 @I2@ INDI\n1 NAME C /D/\n0 TRLR\n"
	out := encodeToString(t, decodeEntityOnly(t, in))
	if !strings.Contains(out, "1 ASSO @I2@\n2 RELA Godfather\n") || strings.Contains(out, "ROLE") {
		t.Errorf("5.5.1 ASSO not written with RELA:\n%s", out)
	}
}

func TestLDSStatusWrittenForOutputVersion(t *testing.T) {
	dated := &gedcom.LDSOrdinance{Type: gedcom.LDSBaptism, Status: "COMPLETED", StatusDate: "27 MAR 2022", StatusTime: "15:48"}
	undated := &gedcom.LDSOrdinance{Type: gedcom.LDSBaptism, Status: "COMPLETED"}
	tests := []struct {
		name    string
		version gedcom.Version
		ord     *gedcom.LDSOrdinance
		want    string // the BAPL block, up to the next level-1 line
	}{
		{"7.0 dated", gedcom.Version70, dated, "1 BAPL\n2 STAT COMPLETED\n3 DATE 27 MAR 2022\n4 TIME 15:48\n"},
		{"5.5.1 dated drops TIME", gedcom.Version551, dated, "1 BAPL\n2 STAT COMPLETED\n3 DATE 27 MAR 2022\n"},
		{"5.5 has no STAT.DATE", gedcom.Version55, dated, "1 BAPL\n2 STAT COMPLETED\n"},
		{"unknown version drops nothing", "", dated, "1 BAPL\n2 STAT COMPLETED\n3 DATE 27 MAR 2022\n4 TIME 15:48\n"},
		{"7.0 undated omits STAT", gedcom.Version70, undated, "1 BAPL\n"},
		{"5.5.1 undated omits STAT", gedcom.Version551, undated, "1 BAPL\n"},
		{"5.5 undated", gedcom.Version55, undated, "1 BAPL\n2 STAT COMPLETED\n"},
		{"unknown version undated", "", undated, "1 BAPL\n2 STAT COMPLETED\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ord := *tt.ord
			out := encodeIndividual(t, tt.version, &gedcom.Individual{LDSOrdinances: []*gedcom.LDSOrdinance{&ord}})
			if !strings.Contains(out, tt.want+"0 TRLR\n") {
				t.Errorf("BAPL block: want\n%s\ngot:\n%s", tt.want, out)
			}
		})
	}
}

func TestLDSStatusDateRoundTrips(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME A /B/\n1 ENDL\n2 TEMP SLAKE\n2 STAT COMPLETED\n3 DATE 1 JAN 2000\n" +
		"0 TRLR\n"
	out := encodeToString(t, decodeEntityOnly(t, in))
	if !strings.Contains(out, "2 STAT COMPLETED\n3 DATE 1 JAN 2000\n") {
		t.Errorf("STAT written without its DATE:\n%s", out)
	}
}

func TestEventFamilyLinkWritten(t *testing.T) {
	out := encodeIndividual(t, gedcom.Version70, &gedcom.Individual{Events: []*gedcom.Event{
		{Type: gedcom.EventAdoption, FamilyXRef: "@F1@", AdoptedBy: "BOTH", AdoptedByPhrase: "Both parents"},
		{Type: gedcom.EventBirth, FamilyXRef: "@F2@"},
		// AdoptedBy has no parent without a FamilyXRef, so it is not written.
		{Type: gedcom.EventAdoption, Date: "1900", AdoptedBy: "HUSB"},
	}})
	want := "1 ADOP\n2 FAMC @F1@\n3 ADOP BOTH\n4 PHRASE Both parents\n" +
		"1 BIRT\n2 FAMC @F2@\n" +
		"1 ADOP\n2 DATE 1900\n0 TRLR\n"
	if !strings.Contains(out, want) {
		t.Errorf("want\n%s\ngot:\n%s", want, out)
	}
}

// TestEventFamilyLinkRoundTrips is the #740 reproduction on maximal70.ged:
// each ADOP event's FAMC was dropped, leaving a bare "1 ADOP".
func TestEventFamilyLinkRoundTrips(t *testing.T) {
	const in = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 ADOP\n2 FAMC @F1@\n3 ADOP WIFE\n4 PHRASE Adoption phrase\n" +
		"1 CHR\n2 FAMC @F1@\n0 @F1@ FAM\n0 TRLR\n"
	out := encodeToString(t, decodeEntityOnly(t, in))
	want := "1 ADOP\n2 FAMC @F1@\n3 ADOP WIFE\n4 PHRASE Adoption phrase\n1 CHR\n2 FAMC @F1@\n"
	if !strings.Contains(out, want) {
		t.Errorf("want\n%s\ngot:\n%s", want, out)
	}
}

func TestDegenerateMediaLinkSkipped(t *testing.T) {
	out := encodeIndividual(t, gedcom.Version551, &gedcom.Individual{Media: []*gedcom.MediaLink{
		{}, // nothing to write: skipped rather than written as a bare OBJE
		{MediaXRef: "@O1@"},
		{Title: "Inline, file not typed yet (#733)"},
	}})
	want := "0 @I1@ INDI\n1 OBJE @O1@\n1 OBJE\n2 TITL Inline, file not typed yet (#733)\n0 TRLR\n"
	if !strings.Contains(out, want) {
		t.Errorf("want\n%s\ngot:\n%s", want, out)
	}
}

// TestMaximal70ClearedTagsRoundTrip is the last #740 criterion: decode ->
// clear Tags -> encode -> decode of maximal70.ged has zero typed differences,
// and no record writes a level-1 structure more often than the file holds
// it, so none is written twice (the SNOTE case #711 fixed). Lines below level
// 1 are not counted: the encoder writes some the file leaves implicit, such
// as GIVN and SURN from a NAME's value.
func TestMaximal70ClearedTagsRoundTrip(t *testing.T) {
	const path = "../testdata/gedcom-7.0/maximal70.ged"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	orig := decodeGEDCOM(t, string(data))
	doc := decodeGEDCOM(t, string(data))
	clearRecordTags(doc)
	redoc := decodeGEDCOM(t, encodeToString(t, doc))

	if len(redoc.Records) != len(orig.Records) {
		t.Fatalf("record count %d, want %d", len(redoc.Records), len(orig.Records))
	}
	for i, rec := range orig.Records {
		got := redoc.Records[i]
		if diffs := typedDiffs(reflect.ValueOf(rec.Entity), reflect.ValueOf(got.Entity), rec.XRef); len(diffs) > 0 {
			t.Errorf("%s: typed fields differ after the round trip: %v", rec.XRef, diffs)
		}
		want := structureCounts(rec.Tags)
		for tag, n := range structureCounts(got.Tags) {
			if n > want[tag] {
				t.Errorf("%s: level-1 %s written %d times, the file has it %d", rec.XRef, tag, n, want[tag])
			}
		}
	}
}

func structureCounts(tags []*gedcom.Tag) map[string]int {
	counts := map[string]int{}
	for _, tag := range tags {
		if tag.Level == 1 {
			counts[tag.Tag]++
		}
	}
	return counts
}

// typedDiffs returns the paths at which two entities differ, ignoring every
// raw Tags field: those hold the file's lines, which the round trip rewrites.
func typedDiffs(a, b reflect.Value, path string) []string {
	if a.Kind() != b.Kind() {
		return []string{path}
	}
	switch a.Kind() {
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				return []string{path}
			}
			return nil
		}
		return typedDiffs(a.Elem(), b.Elem(), path)
	case reflect.Struct:
		var out []string
		for i := 0; i < a.NumField(); i++ {
			if name := a.Type().Field(i).Name; name != "Tags" {
				out = append(out, typedDiffs(a.Field(i), b.Field(i), path+"."+name)...)
			}
		}
		return out
	case reflect.Slice:
		if a.Len() != b.Len() {
			return []string{fmt.Sprintf("%s (len %d, want %d)", path, b.Len(), a.Len())}
		}
		var out []string
		for i := 0; i < a.Len(); i++ {
			out = append(out, typedDiffs(a.Index(i), b.Index(i), fmt.Sprintf("%s[%d]", path, i))...)
		}
		return out
	default:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			return []string{path}
		}
	}
	return nil
}
