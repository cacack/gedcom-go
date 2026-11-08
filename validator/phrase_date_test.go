package validator

import (
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// decodePhraseDateDoc decodes a GEDCOM 7.0 document with two John Doe
// individuals whose birth and death DATE values gedcom.ParseDate rejects but
// carry a PHRASE, so each date decodes to a phrase-only Date (issue #695).
func decodePhraseDateDoc(t *testing.T) *gedcom.Document {
	t.Helper()
	var lines []string
	lines = append(lines, "0 HEAD", "1 GEDC", "2 VERS 7.0")
	for _, xref := range []string{"@I1@", "@I2@"} {
		lines = append(lines,
			"0 "+xref+" INDI",
			"1 NAME John /Doe/",
			"1 SEX M",
			"1 BIRT",
			"2 DATE CIRCA 1850",
			"3 PHRASE about 1850",
			"1 DEAT",
			"2 DATE CIRCA 1920",
			"3 PHRASE about 1920",
		)
	}
	lines = append(lines, "0 TRLR")
	doc, err := decoder.Decode(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return doc
}

// TestPhraseOnlyDateCountsAsRecorded pins that a phrase-only birth or death
// date counts as a recorded date: the quality report counts it towards
// birth/death coverage and emits no MISSING_BIRTH_DATE, and the duplicate
// detector's RequireBirthDate gate lets the pair through.
func TestPhraseOnlyDateCountsAsRecorded(t *testing.T) {
	doc := decodePhraseDateDoc(t)

	ind := doc.GetIndividual("@I1@")
	if d := ind.BirthDate(); d == nil || !d.IsPhrase {
		t.Fatalf("BirthDate() = %+v, want a phrase-only date", d)
	}

	report := NewQualityAnalyzer().Analyze(doc)
	if report.IndividualsWithBirthDate != 2 {
		t.Errorf("IndividualsWithBirthDate = %d, want 2", report.IndividualsWithBirthDate)
	}
	if report.IndividualsWithDeathDate != 2 {
		t.Errorf("IndividualsWithDeathDate = %d, want 2", report.IndividualsWithDeathDate)
	}
	if got := FilterByCode(report.CompletenessIssues, CodeMissingBirthDate); len(got) != 0 {
		t.Errorf("MISSING_BIRTH_DATE issues = %v, want none", got)
	}

	config := DefaultDuplicateConfig()
	config.RequireBirthDate = true
	if got := NewDuplicateDetector(config).FindDuplicates(doc); len(got) != 1 {
		t.Errorf("FindDuplicates() with RequireBirthDate = %d pairs, want 1", len(got))
	}
}
