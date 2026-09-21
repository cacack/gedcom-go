package validator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
)

// The tests below decode real GEDCOM bytes instead of hand-assembling a
// gedcom.Header. That is the whole point: issue #503 was a decoder defect
// (buildHeader never read HEAD.SUBM), and every existing MISSING_SUBM test
// built the Header struct directly, so the suite stayed green while the
// validator warned on every 5.5/5.5.1 document in the wild.
//
// TestHeaderValidatorValidateHeader_SUBM still covers the validator's own
// logic; these cover the decode -> validate path it cannot see.

// strictnessLevels enumerates every declared Strictness level so the subtests
// below cover all of them. reportsWarnings records whether the level surfaces
// SeverityWarning issues at all — MISSING_SUBM is a warning, so Relaxed
// suppresses it regardless of the header's contents (see
// TestStrictnessRelaxedStillSuppressesWarnings).
// TestStrictnessLevelsTableIsComplete fails if a level is added and not listed.
var strictnessLevels = []struct {
	name            string
	level           Strictness
	reportsWarnings bool
}{
	{"Relaxed", StrictnessRelaxed, false},
	{"Normal", StrictnessNormal, true},
	{"Strict", StrictnessStrict, true},
}

// Documents with a header SUBM pointer and a matching submitter record.
const (
	doc55WithSubmitter = `0 HEAD
1 GEDC
2 VERS 5.5
2 FORM LINEAGE-LINKED
1 CHAR ANSEL
1 SUBM @U1@
0 @U1@ SUBM
1 NAME John Researcher
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	doc551WithSubmitter = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 SUBM @U1@
0 @U1@ SUBM
1 NAME John Researcher
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`
)

// The same documents with the header SUBM line removed.
const (
	doc55NoSubmitter = `0 HEAD
1 GEDC
2 VERS 5.5
2 FORM LINEAGE-LINKED
1 CHAR ANSEL
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`

	doc551NoSubmitter = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR`
)

// reportsMissingSUBM decodes input and reports whether ValidateAll emits
// CodeMissingSUBM at the given strictness level. Other codes are ignored: the
// fixtures trip unrelated validators and a count assertion would be brittle.
func reportsMissingSUBM(t *testing.T, input string, level Strictness) bool {
	t.Helper()

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	v := NewWithOptions(&ValidateOptions{Strictness: level})
	return codesReported(v.ValidateAll(doc))[CodeMissingSUBM]
}

// TestValidateAllDecodedHeaderWithSubmitter is the regression test for #503:
// a decoded document whose header carries SUBM must not draw MISSING_SUBM at
// any strictness level.
func TestValidateAllDecodedHeaderWithSubmitter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "5.5", input: doc55WithSubmitter},
		{name: "5.5.1", input: doc551WithSubmitter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, s := range strictnessLevels {
				t.Run(s.name, func(t *testing.T) {
					if reportsMissingSUBM(t, tt.input, s.level) {
						t.Errorf("%s reported for a decoded document with a header SUBM", CodeMissingSUBM)
					}
				})
			}
		})
	}
}

// TestValidateAllDecodedHeaderWithoutSubmitter keeps the warning honest: a
// decoded document genuinely lacking a header SUBM must still trip it wherever
// warnings are reported at all.
func TestValidateAllDecodedHeaderWithoutSubmitter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "5.5", input: doc55NoSubmitter},
		{name: "5.5.1", input: doc551NoSubmitter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, s := range strictnessLevels {
				t.Run(s.name, func(t *testing.T) {
					got := reportsMissingSUBM(t, tt.input, s.level)
					if got != s.reportsWarnings {
						t.Errorf("%s reported = %v, want %v for a decoded document with no header SUBM",
							CodeMissingSUBM, got, s.reportsWarnings)
					}
				})
			}
		})
	}
}

// TestStrictnessLevelsTableIsComplete guards the loops above: a Strictness
// level added to the package but not to strictnessLevels would otherwise be
// skipped in silence.
func TestStrictnessLevelsTableIsComplete(t *testing.T) {
	declared := declaredStrictnessConstants(t)
	if len(declared) != len(strictnessLevels) {
		t.Errorf("package declares %d Strictness constants %v, strictnessLevels covers %d",
			len(declared), declared, len(strictnessLevels))
	}
}

// declaredStrictnessConstants returns the names of the package's Strictness
// constants by parsing its non-test sources. Reflection cannot enumerate Go
// constants, so the source is the only authority.
func declaredStrictnessConstants(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading package directory: %v", err)
	}

	var names []string
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, ident := range value.Names {
					if strings.HasPrefix(ident.Name, "Strictness") {
						names = append(names, ident.Name)
					}
				}
			}
		}
	}

	if len(names) == 0 {
		t.Fatal("no Strictness constants found; the guard would prove nothing")
	}
	return names
}
