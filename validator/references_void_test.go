package validator

import (
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #703: the GEDCOM 7.0 null pointer @VOID@ is deliberate, not a
// reference to a missing record, so the typed ORPHANED_* checks must skip it
// under the same rule BROKEN_XREF already applies.

func decodeString(t *testing.T, lines ...string) *gedcom.Document {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return doc
}

func orphanedIssues(issues []Issue) []Issue {
	var out []Issue
	for _, issue := range issues {
		if strings.HasPrefix(issue.Code, "ORPHANED_") {
			out = append(out, issue)
		}
	}
	return out
}

func TestReferenceValidator_VoidIsNotOrphanedInMaximal70(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	for _, issue := range orphanedIssues(New().ValidateAll(doc)) {
		t.Errorf("unexpected %s at line %d: %s", issue.Code, issue.LineNumber, issue.Message)
	}

	report := NewReferenceValidator().Report(doc)
	if report.OrphanedReferences != 0 {
		t.Errorf("Report().OrphanedReferences = %d, want 0 (%v)", report.OrphanedReferences, report.OrphanedByType)
	}
}

func TestReferenceValidator_VoidInFamilyAndIndividual70(t *testing.T) {
	doc := decodeString(t,
		"0 HEAD",
		"1 GEDC",
		"2 VERS 7.0",
		"0 @I1@ INDI",
		"1 FAMS @VOID@",
		"1 FAMC @VOID@",
		"0 @F1@ FAM",
		"1 HUSB @VOID@",
		"1 WIFE @VOID@",
		"1 CHIL @VOID@",
		"1 CHIL @I1@",
		"0 TRLR",
	)

	if got := orphanedIssues(NewReferenceValidator().Validate(doc)); len(got) != 0 {
		t.Errorf("Validate() reported %d ORPHANED_* issues for @VOID@, want 0: %+v", len(got), got)
	}
	report := NewReferenceValidator().Report(doc)
	if report.TotalReferences != 1 || report.OrphanedReferences != 0 {
		t.Errorf("Report() = total %d orphaned %d, want 1 and 0 (null pointers are not references)",
			report.TotalReferences, report.OrphanedReferences)
	}
}

func TestReferenceValidator_VoidIsAnIdentifierIn551(t *testing.T) {
	doc := decodeString(t,
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"2 FORM LINEAGE-LINKED",
		"1 CHAR UTF-8",
		"0 @VOID@ INDI",
		"1 NAME Void /Person/",
		"1 FAMS @F1@",
		"0 @F1@ FAM",
		"1 HUSB @VOID@",
		"1 WIFE @MISSING@",
		"0 TRLR",
	)

	got := orphanedIssues(NewReferenceValidator().Validate(doc))
	if len(got) != 1 || got[0].Code != CodeOrphanedWIFE {
		t.Fatalf("Validate() = %+v, want exactly one ORPHANED_WIFE for @MISSING@", got)
	}
	report := NewReferenceValidator().Report(doc)
	if report.TotalReferences != 3 || report.OrphanedReferences != 1 {
		t.Errorf("Report() = total %d orphaned %d, want 3 and 1 (@VOID@ resolves to a record)",
			report.TotalReferences, report.OrphanedReferences)
	}
}

func TestReferenceValidator_VoidUndeclaredVersionIsNull(t *testing.T) {
	// A document built in code declares no version and has no @VOID@
	// record, so @VOID@ is the null pointer there too.
	doc := newTestDocument()
	addFamily(doc, &gedcom.Family{XRef: "@F1@", Husband: "@VOID@", Children: []string{"@VOID@", "@I9@"}})

	got := orphanedIssues(NewReferenceValidator().Validate(doc))
	if len(got) != 1 || got[0].Code != CodeOrphanedCHIL || got[0].RelatedXRef != "@I9@" {
		t.Fatalf("Validate() = %+v, want exactly one ORPHANED_CHIL for @I9@", got)
	}
}

func TestStreamingValidator_VoidFollowsVersionOption(t *testing.T) {
	feed := func(v gedcom.Version) []Issue {
		sv := NewStreamingValidator(StreamingOptions{Version: v})
		var issues []Issue
		issues = append(issues, sv.ValidateRecord(&gedcom.Record{
			XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@", SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@VOID@"}}},
		})...)
		issues = append(issues, sv.ValidateRecord(&gedcom.Record{
			XRef: "@F1@", Type: gedcom.RecordTypeFamily,
			Entity: &gedcom.Family{XRef: "@F1@", Husband: "@VOID@", Wife: "@MISSING@", Children: []string{"@VOID@"}},
		})...)
		return orphanedIssues(append(issues, sv.Finalize()...))
	}

	for _, v := range []gedcom.Version{gedcom.Version70, ""} {
		got := feed(v)
		if len(got) != 1 || got[0].Code != CodeOrphanedWIFE {
			t.Errorf("Version %q: Finalize() = %+v, want only ORPHANED_WIFE for @MISSING@", v, got)
		}
	}
	for _, v := range []gedcom.Version{gedcom.Version55, gedcom.Version551} {
		got := feed(v)
		codes := map[string]int{}
		for _, issue := range got {
			codes[issue.Code]++
		}
		want := map[string]int{CodeOrphanedFAMS: 1, CodeOrphanedHUSB: 1, CodeOrphanedWIFE: 1, CodeOrphanedCHIL: 1}
		if len(codes) != len(want) {
			t.Errorf("Version %q: Finalize() codes = %v, want %v (@VOID@ is an identifier)", v, codes, want)
			continue
		}
		for code, n := range want {
			if codes[code] != n {
				t.Errorf("Version %q: %s reported %d times, want %d", v, code, codes[code], n)
			}
		}
	}
}
