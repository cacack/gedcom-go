package validator

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

func grammarIssuesForFile(t *testing.T, path string) (issues []Issue, lines []string) {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- a test fixture path
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	doc, err := decoder.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode %s: %v", path, err)
	}
	lines = strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	return validateGrammar(doc), lines
}

// gedcom7code's 5/age-invalid.ged holds one malformed 5.5.1 AGE_AT_EVENT per
// AGE line; each must be reported, once, on its own line, and nothing else.
// Its 7/ conversion and both age-valid.ged files are valid.
func TestGrammar_GEDCOM7CodeAgeFiles(t *testing.T) {
	issues, lines := grammarIssuesForFile(t, "../testdata/gedcom7code/5/age-invalid.ged")
	var ageLines, issueLines []int
	for i, line := range lines {
		if fields := strings.Fields(line); len(fields) >= 2 && fields[1] == "AGE" {
			ageLines = append(ageLines, i+1)
		}
	}
	for _, i := range issues {
		if i.Code != CodeInvalidPayload || i.Details["datatype"] != "age" {
			t.Errorf("line %d: %s %v, want INVALID_PAYLOAD (age)", i.LineNumber, i.Code, i.Details)
		}
		issueLines = append(issueLines, i.LineNumber)
	}
	if len(ageLines) != 92 {
		t.Fatalf("age-invalid.ged has %d AGE lines, want 92; the fixture changed", len(ageLines))
	}
	if !slices.Equal(issueLines, ageLines) {
		t.Errorf("issue lines differ from the AGE lines\n got: %v\nwant: %v", issueLines, ageLines)
	}

	for _, f := range []string{"5/age-valid.ged", "7/age-valid.ged", "7/age-invalid.ged"} {
		if issues, _ := grammarIssuesForFile(t, "../testdata/gedcom7code/"+f); len(issues) > 0 {
			t.Errorf("%s: %q, want none", f, grammarFindings(issues))
		}
	}
}

// grammarFixtureFindings lists, per vendored fixture, what the grammar check
// reports. A fixture not listed must yield nothing; every finding below is a
// real departure from its version's grammar, not a false verdict.
var grammarFixtureFindings = map[string][]string{
	"gedcom-5.5.1/comprehensive.ged": {
		"UNEXPECTED_SUBSTRUCTURE L120 EMAIL", // 5.5.1 INDI has no address structure
		"UNEXPECTED_SUBSTRUCTURE L121 WWW",
	},
	"gedcom-5.5.1/minimal.ged": {
		"MISSING_REQUIRED_SUBSTRUCTURE L3 FORM", // GEDC.FORM is {1:1}
		"UNEXPECTED_SUBSTRUCTURE L14 EMAIL",
	},
	"gedcom7code/5/char_ascii_2.ged": {"INVALID_ENUM_VALUE L7 CHAR"}, // LATIN1 is not a 5.5.1 CHARACTER_SET
	"gedcom7code/5/date-all.ged":     {"MISSING_REQUIRED_SUBSTRUCTURE L0 SOUR"},
	"gedcom7code/5/obje-1.ged": {
		"MISSING_REQUIRED_SUBSTRUCTURE L0 SOUR",
		"INVALID_ENUM_VALUE L19 FORM", // "other" is not a 5.5.1 MULTIMEDIA_FORMAT
		"INVALID_ENUM_VALUE L22 FORM",
		"UNEXPECTED_SUBSTRUCTURE L25 NOTE", // 5.5.1 dropped NOTE from the embedded OBJE link
	},
	// An older revision of gedcom.io's maximal70, written before 7.0.x
	// deprecated these; the pinned current one (below) has none.
	"gedcom-7.0/maximal70.ged": {
		"DEPRECATED_STRUCTURE L23 ADR1", "DEPRECATED_STRUCTURE L24 ADR2", "DEPRECATED_STRUCTURE L25 ADR3",
		"DEPRECATED_STRUCTURE L38 DATA",
		"DEPRECATED_STRUCTURE L167 STAT", "DEPRECATED_STRUCTURE L182 STAT", "DEPRECATED_STRUCTURE L194 STAT",
		"DEPRECATED_STRUCTURE L198 STAT", "DEPRECATED_STRUCTURE L405 STAT", "DEPRECATED_STRUCTURE L410 STAT",
		"DEPRECATED_STRUCTURE L420 STAT",
		"DEPRECATED_STRUCTURE L634 ADR1", "DEPRECATED_STRUCTURE L635 ADR2", "DEPRECATED_STRUCTURE L636 ADR3",
		"DEPRECATED_STRUCTURE L809 ADR1", "DEPRECATED_STRUCTURE L810 ADR2", "DEPRECATED_STRUCTURE L811 ADR3",
	},
}

func TestGrammar_VendoredFixtures(t *testing.T) {
	var files []string
	for _, pattern := range []string{"gedcom-5.5.1/*.ged", "gedcom-7.0/*.ged", "gedcom7code/5/*.ged", "gedcom7code/7/*.ged"} {
		matches, err := filepath.Glob(filepath.Join("..", "testdata", pattern))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	if len(files) < 40 {
		t.Fatalf("found %d fixtures; the glob is wrong", len(files))
	}
	seen := map[string]bool{}
	for _, path := range files {
		name := filepath.ToSlash(strings.TrimPrefix(path, "../testdata/"))
		if name == "gedcom7code/5/age-invalid.ged" {
			continue // TestGrammar_GEDCOM7CodeAgeFiles
		}
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			issues, _ := grammarIssuesForFile(t, path)
			if got, want := grammarFindings(issues), grammarFixtureFindings[name]; !slices.Equal(got, want) {
				t.Errorf("findings changed; update grammarFixtureFindings\n got: %q\nwant: %q", got, want)
			}
		})
	}
	for name := range grammarFixtureFindings {
		if !seen[name] {
			t.Errorf("grammarFixtureFindings lists %s, which no longer exists", name)
		}
	}
}

// gedcom.io's 7.0 test files are valid 7.0, so the grammar check reports
// nothing on any of them. warnings70.ged is TestGEDCOMIOWarningsFiles's.
func TestGrammar_GEDCOMIOFiles(t *testing.T) {
	dir, lock := corpora.GEDCOMIO(t)
	checked := 0
	for _, e := range lock.Entries {
		if !strings.HasPrefix(e.Path, "gedcom70/") || !strings.HasSuffix(e.Path, ".ged") || e.Path == "gedcom70/warnings70.ged" {
			continue
		}
		checked++
		t.Run(e.Path, func(t *testing.T) {
			issues, _ := grammarIssuesForFile(t, filepath.Join(dir, filepath.FromSlash(e.Path)))
			if len(issues) > 0 {
				t.Errorf("%q, want none", grammarFindings(issues))
			}
		})
	}
	if checked < 20 {
		t.Fatalf("checked %d files; the lock changed", checked)
	}
}
