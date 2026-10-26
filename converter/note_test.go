package converter_test

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// convertTo70 decodes input, converts it to 7.0 and encodes the result.
func convertTo70(t *testing.T, input string) (*gedcom.Document, *gedcom.ConversionReport, string) {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	converted, report, err := converter.Convert(doc, gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return converted, report, buf.String()
}

// transformationCount returns the Count of the report's transformation of
// type typ, or 0.
func transformationCount(report *gedcom.ConversionReport, typ string) int {
	for _, tr := range report.Transformations {
		if tr.Type == typ {
			return tr.Count
		}
	}
	return 0
}

// TestConvertNoteRecordsTo70 pins issue #567: a 5.x NOTE record becomes an
// SNOTE record with its CONC/CONT folded into its text, and each pointer to it
// becomes SNOTE, so the output is valid 7.0 and the shared note survives.
func TestConvertNoteRecordsTo70(t *testing.T) {
	for _, version := range []string{"5.5", "5.5.1"} {
		t.Run(version, func(t *testing.T) {
			input := "0 HEAD\n1 GEDC\n2 VERS " + version + "\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
				"0 @I1@ INDI\n1 NOTE @N1@\n1 NOTE inline text\n1 NOTE @MISSING@\n1 BIRT\n2 NOTE @N1@\n" +
				"0 @N1@ NOTE first\n1 CONC  part\n1 CONT second\n1 SOUR @S1@\n" +
				"0 @S1@ SOUR\n1 TITL Title\n0 TRLR\n"
			converted, report, out := convertTo70(t, input)

			for _, want := range []string{
				"0 @N1@ SNOTE first part\n1 CONT second\n1 SOUR @S1@\n",
				"1 SNOTE @N1@\n1 NOTE inline text\n1 NOTE @MISSING@\n1 BIRT\n2 SNOTE @N1@\n",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("output lacks %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, "CONC") || strings.Contains(out, " NOTE @N1@") {
				t.Errorf("output keeps CONC or a NOTE pointer:\n%s", out)
			}

			record := converted.GetRecord("@N1@")
			if record == nil || record.Type != gedcom.RecordTypeSharedNote {
				t.Fatalf("@N1@ record = %+v, want SNOTE", record)
			}
			snote, ok := record.GetSharedNote()
			if !ok || snote.XRef != "@N1@" || snote.Text != "first part\nsecond" {
				t.Errorf("@N1@ entity = %+v, want SharedNote with text", record.Entity)
			}

			for typ, want := range map[string]int{
				"NOTE_RECORD_TO_SNOTE":  1,
				"NOTE_POINTER_TO_SNOTE": 2,
				"CONC_REMOVED":          1,
				"CONT_CONVERTED":        1,
			} {
				if got := transformationCount(report, typ); got != want {
					t.Errorf("%s count = %d, want %d: %+v", typ, got, want, report.Transformations)
				}
			}
			var paths []string
			for _, n := range report.Normalized {
				paths = append(paths, n.Path)
			}
			if !slices.Contains(paths, "Note @N1@") {
				t.Errorf("Normalized paths %q lack Note @N1@", paths)
			}

			// The output reads back as a 7.0 shared note.
			got, err := decoder.Decode(strings.NewReader(out))
			if err != nil {
				t.Fatalf("re-decode: %v", err)
			}
			if sn := got.GetSharedNote("@N1@"); sn == nil || sn.Text != "first part\nsecond" {
				t.Errorf("re-decoded @N1@ = %+v, want shared note text", sn)
			}
		})
	}
}

// TestConvertNoteRecordWithoutContinuations covers a NOTE record whose text
// has no continuation lines and no pointer to it: it is still upgraded, and
// no pointer transformation is reported.
func TestConvertNoteRecordWithoutContinuations(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @N1@ NOTE alone\n0 TRLR\n"
	_, report, out := convertTo70(t, input)
	if !strings.Contains(out, "0 @N1@ SNOTE alone\n") {
		t.Errorf("output lacks SNOTE record:\n%s", out)
	}
	if got := transformationCount(report, "NOTE_POINTER_TO_SNOTE"); got != 0 {
		t.Errorf("NOTE_POINTER_TO_SNOTE count = %d, want 0", got)
	}
	if got := transformationCount(report, "CONC_REMOVED"); got != 0 {
		t.Errorf("CONC_REMOVED count = %d, want 0", got)
	}
}

// TestConvertDualYearDatesTo70 pins issue #567: GEDCOM 7.0 has no dual-year
// syntax, so a dual-year date is kept but reported as DataLoss naming each
// affected record once, with a Preserved note per date.
func TestConvertDualYearDatesTo70(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 BIRT\n2 DATE 11 FEB 1731/32\n1 DEAT\n2 DATE BET 1750/51 AND 1760\n" +
		"0 @I2@ INDI\n1 BIRT\n2 DATE INT 1800 (born 1799/00)\n" +
		"0 @F1@ FAM\n1 MARR\n2 DATE @#DJULIAN@ 1 JAN 1700/01\n0 TRLR\n"
	_, report, out := convertTo70(t, input)

	if !strings.Contains(out, "2 DATE 11 FEB 1731/32\n") || !strings.Contains(out, "2 DATE JULIAN 1 JAN 1700/01\n") {
		t.Errorf("dual-year dates not kept:\n%s", out)
	}
	var dual []gedcom.DataLossItem
	for _, d := range report.DataLoss {
		if d.Feature == "Dual-year dates" {
			dual = append(dual, d)
		}
	}
	if len(dual) != 1 {
		t.Fatalf("Dual-year DataLoss items = %+v, want 1", report.DataLoss)
	}
	if want := []string{"@I1@", "@F1@"}; !slices.Equal(dual[0].AffectedRecords, want) {
		t.Errorf("AffectedRecords = %q, want %q", dual[0].AffectedRecords, want)
	}
	var preserved []string
	for _, n := range report.Preserved {
		if strings.Contains(n.Reason, "dual-year") {
			preserved = append(preserved, n.Path)
		}
	}
	want := []string{
		"Individual @I1@ > BIRT > DATE",
		"Individual @I1@ > DEAT > DATE",
		"Family @F1@ > MARR > DATE",
	}
	if !slices.Equal(preserved, want) {
		t.Errorf("dual-year Preserved paths = %q, want %q", preserved, want)
	}

	// StrictDataLoss now refuses the conversion.
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	opts := converter.DefaultOptions()
	opts.StrictDataLoss = true
	if _, _, err := converter.ConvertWithOptions(doc, gedcom.Version70, opts); err == nil {
		t.Error("StrictDataLoss conversion with dual-year dates succeeded, want error")
	}
}

// TestConvertDualYearNotReportedFor5x checks the dual-year report is specific
// to a 7.0 target: 5.5.1 keeps the syntax.
func TestConvertDualYearNotReportedFor5x(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5\n1 CHAR UTF-8\n0 @I1@ INDI\n1 BIRT\n2 DATE 11 FEB 1731/32\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	_, report, err := converter.Convert(doc, gedcom.Version551)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if report.HasDataLoss() {
		t.Errorf("5.5 -> 5.5.1 DataLoss = %+v, want none", report.DataLoss)
	}
}

// TestConvertCorpusNotesTo70 checks the corpus files named in issue #567 no
// longer keep NOTE records, NOTE pointers to them, or CONC in 7.0 output, and
// that their dual-year dates are reported.
func TestConvertCorpusNotesTo70(t *testing.T) {
	tests := []struct {
		file     string
		wantDual bool
	}{
		{"../testdata/gedcom-5.5/pres2020.ged", true},
		{"../testdata/edge-cases/vendor-rootsmagic.ged", false},
		{"../testdata/gedcom-5.5/royal92.ged", true},
		{"../testdata/edge-cases/date-dual-years.ged", true},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			converted, report, out := convertTo70(t, string(data))
			for _, r := range converted.Records {
				if r.Type == gedcom.RecordTypeNote {
					t.Fatalf("NOTE record %s kept", r.XRef)
				}
			}
			for _, line := range strings.Split(out, "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 && fields[1] == "CONC" {
					t.Fatalf("output keeps %q", line)
				}
				if len(fields) == 3 && fields[1] == "NOTE" {
					if r := converted.GetRecord(fields[2]); r != nil {
						t.Fatalf("output keeps NOTE pointer %q to a record", line)
					}
				}
			}
			hasDual := false
			for _, d := range report.DataLoss {
				hasDual = hasDual || d.Feature == "Dual-year dates"
			}
			if hasDual != tt.wantDual {
				t.Errorf("dual-year DataLoss = %v, want %v", hasDual, tt.wantDual)
			}
		})
	}
}

// TestConvertDualYearAffectedRecordsUseOutputXRefs pins that the dual-year
// DataLoss names records by their XRefs in the converted document, not by the
// 5.x XRefs normalizeXRefsForV7 rewrites.
func TestConvertDualYearAffectedRecordsUseOutputXRefs(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @i1@ INDI\n1 BIRT\n2 DATE 11 FEB 1731/32\n0 TRLR\n"
	doc, report, _ := convertTo70(t, input)

	var affected []string
	for _, d := range report.DataLoss {
		if d.Feature == "Dual-year dates" {
			affected = append(affected, d.AffectedRecords...)
		}
	}
	if len(affected) != 1 || doc.GetIndividual(affected[0]) == nil {
		t.Errorf("AffectedRecords = %q, want one XRef resolving in the converted document", affected)
	}
	for _, n := range report.Preserved {
		if strings.Contains(n.Reason, "dual-year") && !strings.Contains(n.Path, affected[0]) {
			t.Errorf("Preserved path %q does not name %q", n.Path, affected[0])
		}
	}
}
