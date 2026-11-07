package converter_test

import (
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// GEDCOM 7.0 has no SUBN structure, so an upgrade has to remove both the
// submission record and the HEAD.SUBN pointer to it, and report each one;
// carrying them into 7.0 output writes lines a strict 7.0 reader rejects.

func submissionDoc(version string) string {
	return `0 HEAD
1 SOUR CONVTEST
1 SUBM @U1@
1 SUBN @S1@
1 GEDC
2 VERS ` + version + `
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @U1@ SUBM
1 NAME Conversion Submitter
0 @S1@ SUBN
1 SUBM @U1@
1 FAMF family.ged
1 TEMP SLAKE
1 ANCE 3
0 @I1@ INDI
1 NAME Test /Person/
0 TRLR
`
}

// subnLines returns the encoded lines whose tag is SUBN.
func subnLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && (fields[1] == "SUBN" || (len(fields) >= 3 && fields[2] == "SUBN")) {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestConvertUpgradeDropsSubmission(t *testing.T) {
	for _, version := range []string{"5.5", "5.5.1"} {
		t.Run(version, func(t *testing.T) {
			doc := decodeString(t, submissionDoc(version))
			if doc.GetSubmission("@S1@") == nil {
				t.Fatal("fixture did not decode a submission record")
			}

			got, report, err := converter.Convert(doc, gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}

			for _, record := range got.Records {
				if record != nil && record.Type == gedcom.RecordTypeSubmission {
					t.Errorf("record %s of type SUBN survived the upgrade", record.XRef)
				}
			}
			if _, ok := got.XRefMap["@S1@"]; ok {
				t.Error("XRefMap still holds @S1@")
			}
			if len(got.Submissions()) != 0 {
				t.Errorf("Submissions() = %d, want 0", len(got.Submissions()))
			}
			if lines := subnLines(encodeString(t, got)); len(lines) != 0 {
				t.Errorf("7.0 output still has SUBN lines: %q", lines)
			}
			// The submitter the SUBN pointed at is a 7.0 record and stays.
			if got.GetSubmitter("@U1@") == nil {
				t.Error("submitter @U1@ was dropped with the submission")
			}

			var loss *gedcom.DataLossItem
			for i := range report.DataLoss {
				if report.DataLoss[i].Feature == "SUBN records" {
					loss = &report.DataLoss[i]
				}
			}
			if loss == nil || len(loss.AffectedRecords) != 1 || loss.AffectedRecords[0] != "@S1@" {
				t.Errorf("DataLoss = %+v, want one SUBN records item naming @S1@", report.DataLoss)
			}
			wantDropped := map[string]string{
				"Submission @S1@": "SUBN",
				"Header > SUBN":   "@S1@",
			}
			for _, note := range report.Dropped {
				if want, ok := wantDropped[note.Path]; ok && note.Original == want {
					delete(wantDropped, note.Path)
				}
			}
			if len(wantDropped) != 0 {
				t.Errorf("Dropped = %+v, missing %v", report.Dropped, wantDropped)
			}

			// The caller's document is an input and is left as it was.
			if doc.GetSubmission("@S1@") == nil {
				t.Error("Convert removed the submission from the source document")
			}
		})
	}
}

// TestConvertKeepsSubmissionBelow70 pins that only the 7.0 upgrade drops
// SUBN: 5.5 and 5.5.1 both define it.
func TestConvertKeepsSubmissionBelow70(t *testing.T) {
	tests := []struct {
		source, target gedcom.Version
		src            string
	}{
		{gedcom.Version55, gedcom.Version551, submissionDoc("5.5")},
		{gedcom.Version551, gedcom.Version55, submissionDoc("5.5.1")},
	}
	for _, tt := range tests {
		t.Run(string(tt.source)+"->"+string(tt.target), func(t *testing.T) {
			got, report, err := converter.Convert(decodeString(t, tt.src), tt.target)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if got.GetSubmission("@S1@") == nil {
				t.Error("submission @S1@ was dropped")
			}
			if lines := subnLines(encodeString(t, got)); len(lines) != 2 {
				t.Errorf("SUBN lines = %q, want the header pointer and the record", lines)
			}
			for _, note := range report.Dropped {
				if strings.Contains(note.Path, "SUBN") || strings.Contains(note.Path, "Submission") {
					t.Errorf("unexpected Dropped note %+v", note)
				}
			}
		})
	}
}

// TestConvertUpgradeNoSubmission pins that a document without SUBN gets no
// submission entries in its report.
func TestConvertUpgradeNoSubmission(t *testing.T) {
	const src = `0 HEAD
1 SOUR CONVTEST
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Test /Person/
0 TRLR
`
	_, report, err := converter.Convert(decodeString(t, src), gedcom.Version70)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, item := range report.DataLoss {
		if item.Feature == "SUBN records" {
			t.Errorf("unexpected DataLoss %+v", item)
		}
	}
	for _, note := range report.Dropped {
		if strings.Contains(note.Path, "SUBN") || strings.Contains(note.Path, "Submission") {
			t.Errorf("unexpected Dropped note %+v", note)
		}
	}
}

// TestConvertUpgradeDropsSubmissionCorpus runs the upgrade over the real
// exports that carry a submission record.
func TestConvertUpgradeDropsSubmissionCorpus(t *testing.T) {
	for _, path := range []string{
		"../testdata/edge-cases/vendor-ftm.ged",
		"../testdata/edge-cases/vendor-customtags-torture.ged",
	} {
		t.Run(path, func(t *testing.T) {
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer f.Close()
			doc, err := decoder.Decode(f)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(doc.Submissions()) == 0 {
				t.Fatal("fixture carries no submission record")
			}
			got, report, err := converter.Convert(doc, gedcom.Version70)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if lines := subnLines(encodeString(t, got)); len(lines) != 0 {
				t.Errorf("7.0 output still has SUBN lines: %q", lines)
			}
			reported := false
			for _, item := range report.DataLoss {
				reported = reported || item.Feature == "SUBN records"
			}
			if !reported {
				t.Error("report has no SUBN records data loss")
			}
		})
	}
}
