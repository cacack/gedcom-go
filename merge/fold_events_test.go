package merge_test

import (
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/merge"
)

// foldTree builds a document of two individuals, @I1@ (keep) and @I2@
// (drop), from their level-1 lines.
func foldTree(keep, drop string) string {
	return "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @S1@ SOUR\n1 TITL Census\n" +
		"0 @I1@ INDI\n" + keep + "0 @I2@ INDI\n" + drop + "0 TRLR\n"
}

func foldMerge(t *testing.T, keep, drop string) ([]string, *merge.RecordMergeReport) {
	t.Helper()
	out, report, err := merge.Individuals(decodeZip(t, foldTree(keep, drop)), "@I1@", "@I2@", merge.RecordMergeOptions{FoldEvents: true})
	if err != nil {
		t.Fatalf("Individuals: %v", err)
	}
	return recordLines(t, out, "@I1@"), report
}

func TestFoldEvents(t *testing.T) {
	tests := []struct {
		name       string
		keep, drop string
		want       []string
		folded     int
	}{
		{
			name:   "equal dates written differently",
			keep:   "1 BIRT\n2 DATE 07 Dec 1977\n2 NOTE Family bible.\n",
			drop:   "1 BIRT\n2 DATE 7 DEC 1977\n2 SOUR @S1@\n",
			want:   []string{"1 BIRT", "2 DATE 07 Dec 1977", "2 NOTE Family bible.", "2 SOUR @S1@"},
			folded: 1,
		},
		{
			name: "different dates kept",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n",
			drop: "1 BIRT\n2 DATE 8 DEC 1977\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "1 BIRT", "2 DATE 8 DEC 1977"},
		},
		{
			name: "modifier differs",
			keep: "1 BIRT\n2 DATE 1977\n2 NOTE a\n",
			drop: "1 BIRT\n2 DATE ABT 1977\n",
			want: []string{"1 BIRT", "2 DATE 1977", "2 NOTE a", "1 BIRT", "2 DATE ABT 1977"},
		},
		{
			name:   "range written differently",
			keep:   "1 RESI\n2 DATE BET 1900 AND 1910\n",
			drop:   "1 RESI\n2 DATE bet 1900 and 1910\n2 NOTE b\n",
			want:   []string{"1 RESI", "2 DATE BET 1900 AND 1910", "2 NOTE b"},
			folded: 1,
		},
		{
			name:   "place only in drop",
			keep:   "1 BIRT\n2 DATE 7 DEC 1977\n",
			drop:   "1 BIRT\n2 DATE 07 DEC 1977\n2 PLAC Springfield\n3 MAP\n4 LATI N39.8\n",
			want:   []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Springfield", "3 MAP", "4 LATI N39.8"},
			folded: 1,
		},
		{
			name:   "place only in keep",
			keep:   "1 BIRT\n2 DATE 7 DEC 1977\n2 PLAC Springfield\n",
			drop:   "1 BIRT\n2 DATE 07 DEC 1977\n2 NOTE c\n",
			want:   []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Springfield", "2 NOTE c"},
			folded: 1,
		},
		{
			name:   "same place, sub-lines unioned",
			keep:   "1 BIRT\n2 DATE 7 DEC 1977\n2 PLAC Springfield\n3 FORM City\n",
			drop:   "1 BIRT\n2 DATE 07 DEC 1977\n2 PLAC Springfield\n3 FORM City\n3 NOTE d\n",
			want:   []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Springfield", "3 FORM City", "3 NOTE d"},
			folded: 1,
		},
		{
			name: "different places kept",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n2 PLAC Springfield\n",
			drop: "1 BIRT\n2 DATE 07 DEC 1977\n2 PLAC Shelbyville\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 PLAC Springfield", "1 BIRT", "2 DATE 07 DEC 1977", "2 PLAC Shelbyville"},
		},
		{
			name:   "identical sub-lines once",
			keep:   "1 BIRT\n2 DATE 7 DEC 1977\n2 SOUR @S1@\n3 PAGE 12\n2 NOTE e\n",
			drop:   "1 BIRT\n2 DATE 07 DEC 1977\n2 SOUR @S1@\n3 PAGE 12\n2 SOUR @S1@\n3 PAGE 13\n",
			want:   []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 SOUR @S1@", "3 PAGE 12", "2 SOUR @S1@", "3 PAGE 13", "2 NOTE e"},
			folded: 1,
		},
		{
			name: "no DATE not folded",
			keep: "1 BIRT\n2 PLAC Springfield\n",
			drop: "1 BIRT\n2 PLAC Springfield\n2 NOTE f\n",
			want: []string{"1 BIRT", "2 PLAC Springfield", "1 BIRT", "2 PLAC Springfield", "2 NOTE f"},
		},
		{
			name: "date phrase not folded",
			keep: "1 BIRT\n2 DATE (unknown)\n",
			drop: "1 BIRT\n2 DATE (unknown)\n2 NOTE g\n",
			want: []string{"1 BIRT", "2 DATE (unknown)", "1 BIRT", "2 DATE (unknown)", "2 NOTE g"},
		},
		{
			name: "unparsable date not folded",
			keep: "1 BIRT\n2 DATE 7 XYZ 1977\n",
			drop: "1 BIRT\n2 DATE 7 XYZ 1977\n2 NOTE k\n",
			want: []string{"1 BIRT", "2 DATE 7 XYZ 1977", "1 BIRT", "2 DATE 7 XYZ 1977", "2 NOTE k"},
		},
		{
			name: "free-text date not folded",
			keep: "1 BIRT\n2 DATE sometime 1977\n",
			drop: "1 BIRT\n2 DATE sometime 1977\n2 NOTE k\n",
			want: []string{"1 BIRT", "2 DATE sometime 1977", "1 BIRT", "2 DATE sometime 1977", "2 NOTE k"},
		},
		{
			name: "empty date not folded",
			keep: "1 BIRT\n2 DATE\n",
			drop: "1 BIRT\n2 DATE\n2 NOTE l\n",
			want: []string{"1 BIRT", "2 DATE", "1 BIRT", "2 DATE", "2 NOTE l"},
		},
		{
			name: "one side unparsable not folded",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n",
			drop: "1 BIRT\n2 DATE 7 XYZ 1977\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "1 BIRT", "2 DATE 7 XYZ 1977"},
		},
		{
			name: "different calendars not folded",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n",
			drop: "1 BIRT\n2 DATE @#DJULIAN@ 7 DEC 1977\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "1 BIRT", "2 DATE @#DJULIAN@ 7 DEC 1977"},
		},
		{
			name: "invalid date not folded",
			keep: "1 BIRT\n2 DATE 32 DEC 1977\n",
			drop: "1 BIRT\n2 DATE 32 Dec 1977\n",
			want: []string{"1 BIRT", "2 DATE 32 DEC 1977", "1 BIRT", "2 DATE 32 Dec 1977"},
		},
		{
			name: "EVEN of different TYPE not folded",
			keep: "1 EVEN\n2 TYPE Graduation\n2 DATE 1990\n",
			drop: "1 EVEN\n2 TYPE Wedding\n2 DATE 1990\n",
			want: []string{"1 EVEN", "2 TYPE Graduation", "2 DATE 1990", "1 EVEN", "2 TYPE Wedding", "2 DATE 1990"},
		},
		{
			name: "FACT of different TYPE not folded",
			keep: "1 FACT Yes\n2 TYPE Veteran\n2 DATE 1990\n",
			drop: "1 FACT Yes\n2 TYPE Mason\n2 DATE 1990\n",
			want: []string{"1 FACT Yes", "2 TYPE Veteran", "2 DATE 1990", "1 FACT Yes", "2 TYPE Mason", "2 DATE 1990"},
		},
		{
			name: "TYPE on one side not folded",
			keep: "1 EVEN\n2 TYPE Graduation\n2 DATE 1990\n",
			drop: "1 EVEN\n2 DATE 1990\n",
			want: []string{"1 EVEN", "2 TYPE Graduation", "2 DATE 1990", "1 EVEN", "2 DATE 1990"},
		},
		{
			name:   "same TYPE folded",
			keep:   "1 EVEN\n2 TYPE Graduation\n2 DATE 1990\n",
			drop:   "1 EVEN\n2 TYPE Graduation\n2 DATE 1990\n2 NOTE m\n",
			want:   []string{"1 EVEN", "2 TYPE Graduation", "2 DATE 1990", "2 NOTE m"},
			folded: 1,
		},
		{
			name: "different TIME not folded",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n3 TIME 10:00\n",
			drop: "1 BIRT\n2 DATE 07 DEC 1977\n3 TIME 11:00\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "3 TIME 10:00", "1 BIRT", "2 DATE 07 DEC 1977", "3 TIME 11:00"},
		},
		{
			name:   "TIME only in drop",
			keep:   "1 BIRT\n2 DATE 7 DEC 1977\n",
			drop:   "1 BIRT\n2 DATE 07 DEC 1977\n3 TIME 11:00\n",
			want:   []string{"1 BIRT", "2 DATE 7 DEC 1977", "3 TIME 11:00"},
			folded: 1,
		},
		{
			name: "different PHRASE not folded",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n3 PHRASE a Wednesday\n",
			drop: "1 BIRT\n2 DATE 07 DEC 1977\n3 PHRASE St Nicholas week\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "3 PHRASE a Wednesday", "1 BIRT", "2 DATE 07 DEC 1977", "3 PHRASE St Nicholas week"},
		},
		{
			name: "different tag not folded",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n",
			drop: "1 CHR\n2 DATE 7 DEC 1977\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "1 CHR", "2 DATE 7 DEC 1977"},
		},
		{
			name: "different value not folded",
			keep: "1 OCCU Farmer\n2 DATE 1900\n",
			drop: "1 OCCU Teacher\n2 DATE 1900\n",
			want: []string{"1 OCCU Farmer", "2 DATE 1900", "1 OCCU Teacher", "2 DATE 1900"},
		},
		{
			name:   "DATE sub-lines unioned",
			keep:   "1 BIRT\n2 DATE 7 DEC 1977\n3 TIME 10:00\n",
			drop:   "1 BIRT\n2 DATE 07 DEC 1977\n3 TIME 10:00\n3 PHRASE Pearl Harbor day\n",
			want:   []string{"1 BIRT", "2 DATE 7 DEC 1977", "3 TIME 10:00", "3 PHRASE Pearl Harbor day"},
			folded: 1,
		},
		{
			name: "two DATEs not folded",
			keep: "1 BIRT\n2 DATE 7 DEC 1977\n",
			drop: "1 BIRT\n2 DATE 7 DEC 1977\n2 DATE 8 DEC 1977\n",
			want: []string{"1 BIRT", "2 DATE 7 DEC 1977", "1 BIRT", "2 DATE 7 DEC 1977", "2 DATE 8 DEC 1977"},
		},
		{
			name: "LDS ordinance not folded",
			keep: "1 BAPL\n2 DATE 7 DEC 1977\n",
			drop: "1 BAPL\n2 DATE 07 DEC 1977\n2 TEMP SLAKE\n",
			want: []string{"1 BAPL", "2 DATE 7 DEC 1977", "1 BAPL", "2 DATE 07 DEC 1977", "2 TEMP SLAKE"},
		},
		{
			name: "extension tag not folded",
			keep: "1 _MILT\n2 DATE 1917\n",
			drop: "1 _MILT\n2 DATE 1917\n2 NOTE n\n",
			want: []string{"1 _MILT", "2 DATE 1917", "1 _MILT", "2 DATE 1917", "2 NOTE n"},
		},
		{
			name: "family event on an individual not folded",
			keep: "1 MARR\n2 DATE 1 JUN 1925\n",
			drop: "1 MARR\n2 DATE 01 JUN 1925\n",
			want: []string{"1 MARR", "2 DATE 1 JUN 1925", "1 MARR", "2 DATE 01 JUN 1925"},
		},
		{
			name:   "NO assertion folded",
			keep:   "1 NO NATU\n2 DATE FROM 1900 TO 1910\n",
			drop:   "1 NO NATU\n2 DATE from 1900 to 1910\n2 NOTE o\n",
			want:   []string{"1 NO NATU", "2 DATE FROM 1900 TO 1910", "2 NOTE o"},
			folded: 1,
		},
		{
			name:   "first matching keep event",
			keep:   "1 BIRT\n2 DATE 1977\n1 BIRT\n2 DATE 7 DEC 1977\n1 BIRT\n2 DATE 7 DEC 1977\n2 NOTE h\n",
			drop:   "1 BIRT\n2 DATE 07 DEC 1977\n2 NOTE i\n1 BIRT\n2 DATE 7 Dec 1977\n2 NOTE j\n",
			want:   []string{"1 BIRT", "2 DATE 1977", "1 BIRT", "2 DATE 7 DEC 1977", "2 NOTE i", "2 NOTE j", "1 BIRT", "2 DATE 7 DEC 1977", "2 NOTE h"},
			folded: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, report := foldMerge(t, tt.keep, tt.drop)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("@I1@ lines =\n%q\nwant\n%q", got, tt.want)
			}
			if len(report.Folded) != tt.folded {
				t.Errorf("len(Folded) = %d, want %d", len(report.Folded), tt.folded)
			}
		})
	}
}

func TestFoldEvents_Report(t *testing.T) {
	keep := "1 NAME Ann /Example/\n1 BIRT\n2 DATE 07 Dec 1977\n2 NOTE Family bible.\n"
	drop := "1 NAME Ann /Example/\n1 BIRT\n2 DATE 7 DEC 1977\n2 SOUR @S1@\n1 DEAT\n2 DATE 2000\n"
	_, report := foldMerge(t, keep, drop)

	if len(report.Folded) != 1 {
		t.Fatalf("Folded = %+v, want one fold", report.Folded)
	}
	f := report.Folded[0]
	if want := []string{"1 BIRT", "2 DATE 7 DEC 1977", "2 SOUR @S1@"}; !reflect.DeepEqual(tagLines(f.Drop.Tags), want) {
		t.Errorf("Folded.Drop = %q, want %q", tagLines(f.Drop.Tags), want)
	}
	if want := []string{"1 BIRT", "2 DATE 07 Dec 1977", "2 NOTE Family bible.", "2 SOUR @S1@"}; !reflect.DeepEqual(tagLines(f.Into.Tags), want) {
		t.Errorf("Folded.Into = %q, want %q", tagLines(f.Into.Tags), want)
	}
	if len(report.Deduplicated) != 1 || len(report.Moved) != 1 || report.Moved[0].Tags[0].Tag != "DEAT" {
		t.Errorf("Deduplicated = %d, Moved = %+v; want the NAME deduplicated and the DEAT moved", len(report.Deduplicated), report.Moved)
	}
}

// TestFoldEvents_Off checks the default merge adds an equal-dated event
// beside keep's, as before FoldEvents.
func TestFoldEvents_Off(t *testing.T) {
	doc := decodeZip(t, foldTree("1 BIRT\n2 DATE 07 Dec 1977\n", "1 BIRT\n2 DATE 7 DEC 1977\n"))
	out, report := mustMerge(t, merge.Individuals, doc, "@I1@", "@I2@")
	if want := []string{"1 BIRT", "2 DATE 07 Dec 1977", "1 BIRT", "2 DATE 7 DEC 1977"}; !reflect.DeepEqual(recordLines(t, out, "@I1@"), want) {
		t.Errorf("@I1@ = %q, want %q", recordLines(t, out, "@I1@"), want)
	}
	if len(report.Folded) != 0 || len(report.Moved) != 1 {
		t.Errorf("report = %+v", report)
	}
}

// TestFoldEvents_RecordDates checks FoldEvents leaves CHAN and CREA, which
// date the record and not an event, as the default merge does (#940).
func TestFoldEvents_RecordDates(t *testing.T) {
	for _, tag := range []string{"CHAN", "CREA"} {
		t.Run(tag, func(t *testing.T) {
			keep := "1 " + tag + "\n2 DATE 7 DEC 1977\n3 TIME 10:00\n"
			drop := "1 " + tag + "\n2 DATE 07 DEC 1977\n2 NOTE b\n"
			off, offReport := mustMerge(t, merge.Individuals, decodeZip(t, foldTree(keep, drop)), "@I1@", "@I2@")
			got, report := foldMerge(t, keep, drop)
			want := []string{"1 " + tag, "2 DATE 7 DEC 1977", "3 TIME 10:00", "1 " + tag, "2 DATE 07 DEC 1977", "2 NOTE b"}
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(recordLines(t, off, "@I1@"), want) {
				t.Errorf("@I1@ = %q with FoldEvents, %q without; want both %q", got, recordLines(t, off, "@I1@"), want)
			}
			if len(report.Folded) != 0 || len(report.Moved) != 1 || len(offReport.Moved) != 1 {
				t.Errorf("Folded = %d, Moved = %d (%d without FoldEvents); want 0, 1 and 1", len(report.Folded), len(report.Moved), len(offReport.Moved))
			}
		})
	}
}

// TestFoldEvents_Families checks the option applies to Families too, and
// the typed view follows the folded lines.
func TestFoldEvents_Families(t *testing.T) {
	doc := decodeZip(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n"+
		"0 @F1@ FAM\n1 MARR\n2 DATE 1 JUN 1925\n"+
		"0 @F2@ FAM\n1 MARR\n2 DATE 01 JUN 1925\n2 PLAC Springfield\n0 TRLR\n")
	out, report, err := merge.Families(doc, "@F1@", "@F2@", merge.RecordMergeOptions{FoldEvents: true})
	if err != nil {
		t.Fatalf("Families: %v", err)
	}
	if want := []string{"1 MARR", "2 DATE 1 JUN 1925", "2 PLAC Springfield"}; !reflect.DeepEqual(recordLines(t, out, "@F1@"), want) {
		t.Errorf("@F1@ = %q, want %q", recordLines(t, out, "@F1@"), want)
	}
	if len(report.Folded) != 1 {
		t.Errorf("Folded = %+v, want one", report.Folded)
	}
	if ev := out.GetFamily("@F1@").Events; len(ev) != 1 || ev[0].PlaceName() != "Springfield" {
		t.Errorf("typed events = %+v, want one MARR at Springfield", ev)
	}
}
