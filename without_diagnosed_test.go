package gedcomgo

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/validator"
)

// duplicateXRefGEDCOM is the reproduction file of #880: @I1@ is defined on
// line 6 and again on line 8.
const duplicateXRefGEDCOM = `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME First /Example/
0 @I1@ INDI
1 NAME Second /Example/
0 TRLR
`

// duplicateXRefIssues returns the DUPLICATE_XREF issues.
func duplicateXRefIssues(issues []Issue) []Issue {
	var matched []Issue
	for _, issue := range issues {
		if issue.Code == validator.CodeDuplicateXRef {
			matched = append(matched, issue)
		}
	}
	return matched
}

// TestWithoutDiagnosed_DecodedFile pins #880: for a decoded file the
// duplicated XRef is left to the decoder's diagnostic, and every other
// validation issue is kept in order.
func TestWithoutDiagnosed_DecodedFile(t *testing.T) {
	result, err := DecodeWithDiagnostics(strings.NewReader(duplicateXRefGEDCOM))
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics: %v", err)
	}
	var diagnosed decoder.Diagnostics
	for _, d := range result.Diagnostics {
		if d.Code == decoder.CodeDuplicateXRef {
			diagnosed = append(diagnosed, d)
		}
	}
	if len(diagnosed) != 1 || diagnosed[0].Line != 8 || diagnosed[0].XRef != "@I1@" {
		t.Fatalf("DUPLICATE_XREF diagnostics = %v, want one on @I1@ at line 8", diagnosed)
	}

	issues := ValidateAll(result.Document)
	before := duplicateXRefIssues(issues)
	if len(before) != 1 || before[0].LineNumber != 8 || before[0].RecordXRef != "@I1@" {
		t.Fatalf("DUPLICATE_XREF issues = %v, want one on @I1@ at line 8", before)
	}

	kept := WithoutDiagnosed(issues, result.Diagnostics)
	if dups := duplicateXRefIssues(kept); len(dups) != 0 {
		t.Errorf("DUPLICATE_XREF issues after WithoutDiagnosed = %v, want none", dups)
	}
	var want, got []string
	for _, issue := range issues {
		if issue.Code != validator.CodeDuplicateXRef {
			want = append(want, issue.String())
		}
	}
	for _, issue := range kept {
		got = append(got, issue.String())
	}
	if !slices.Equal(got, want) {
		t.Errorf("kept issues = %q, want every other issue in order: %q", got, want)
	}
	if after := duplicateXRefIssues(issues); len(after) != 1 {
		t.Errorf("input slice was modified: DUPLICATE_XREF issues = %v, want one", after)
	}
}

// TestWithoutDiagnosed_KeepsUndiagnosed pins that the validator's issue
// survives whenever the decoder did not report that line: a document built in
// memory, a strict decode, a record appended after decoding, and a diagnostic
// for another line, another XRef or with another code.
func TestWithoutDiagnosed_KeepsUndiagnosed(t *testing.T) {
	duplicate := func() *gedcom.Record {
		return &gedcom.Record{XRef: "@I1@", Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{XRef: "@I1@"}}
	}
	built := &gedcom.Document{
		Header:  &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{duplicate(), duplicate()},
	}
	builtIssues := duplicateXRefIssues(ValidateAll(built))
	if len(builtIssues) != 1 || builtIssues[0].LineNumber != 0 {
		t.Fatalf("built document DUPLICATE_XREF issues = %v, want one with no line", builtIssues)
	}

	strict, err := decoder.DecodeWithDiagnostics(strings.NewReader(duplicateXRefGEDCOM),
		&decoder.DecodeOptions{StrictMode: true})
	if err != nil {
		t.Fatalf("strict DecodeWithDiagnostics: %v", err)
	}
	strictIssues := duplicateXRefIssues(ValidateAll(strict.Document))
	if len(strict.Diagnostics) != 0 || len(strictIssues) != 1 {
		t.Fatalf("strict decode: diagnostics = %v, DUPLICATE_XREF issues = %v, want none and one",
			strict.Diagnostics, strictIssues)
	}

	lenient, err := DecodeWithDiagnostics(strings.NewReader(duplicateXRefGEDCOM))
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics: %v", err)
	}
	lenient.Document.Records = append(lenient.Document.Records, duplicate())
	edited := duplicateXRefIssues(ValidateAll(lenient.Document))
	if len(edited) != 2 {
		t.Fatalf("edited document DUPLICATE_XREF issues = %v, want two", edited)
	}

	at8 := validator.NewIssue(validator.SeverityError, validator.CodeDuplicateXRef, "dup", "@I1@").WithLineNumber(8)
	tests := []struct {
		name        string
		issues      []Issue
		diagnostics decoder.Diagnostics
		wantLines   []int
	}{
		{"built in memory, no diagnostics", builtIssues, nil, []int{0}},
		{"built in memory, diagnostic with no line", builtIssues,
			decoder.Diagnostics{{Line: 0, Code: decoder.CodeDuplicateXRef}}, []int{0}},
		{"strict decode", strictIssues, strict.Diagnostics, []int{8}},
		{"record appended after decoding", edited, lenient.Diagnostics, []int{0}},
		{"diagnostic for another line", []Issue{at8},
			decoder.Diagnostics{{Line: 9, Code: decoder.CodeDuplicateXRef}}, []int{8}},
		{"diagnostic for another XRef on the same line", []Issue{at8},
			decoder.Diagnostics{{Line: 8, Code: decoder.CodeDuplicateXRef, XRef: "@I2@"}}, []int{8}},
		{"another code on the same line", []Issue{at8},
			decoder.Diagnostics{{Line: 8, Code: decoder.CodeUnknownTag}}, []int{8}},
		{"a code the packages do not share", []Issue{{Code: decoder.CodeUnknownTag, LineNumber: 8}},
			decoder.Diagnostics{{Line: 8, Code: decoder.CodeUnknownTag}}, []int{8}},
		{"diagnostic with no XRef on the same line", []Issue{at8},
			decoder.Diagnostics{{Line: 8, Code: decoder.CodeDuplicateXRef}}, nil},
		{"no issues", nil, lenient.Diagnostics, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lines []int
			for _, issue := range WithoutDiagnosed(tt.issues, tt.diagnostics) {
				lines = append(lines, issue.LineNumber)
			}
			if !slices.Equal(lines, tt.wantLines) {
				t.Errorf("kept issue lines = %v, want %v", lines, tt.wantLines)
			}
		})
	}
}

// TestSharedFindingCodes pins the set of code strings the decoder and the
// validator both declare (#880). A new overlap fails here until it is either
// renamed or added to sharedFindingCodes and to the documentation that names
// the shared codes: the decoder and validator package docs, FEATURES.md and
// docs/guides/decoding.md.
func TestSharedFindingCodes(t *testing.T) {
	decoderCodes := codeConstants(t, "decoder")
	validatorCodes := codeConstants(t, "validator")
	if len(decoderCodes) < 10 || len(validatorCodes) < 10 {
		t.Fatalf("found %d decoder and %d validator Code constants; the scan is broken",
			len(decoderCodes), len(validatorCodes))
	}
	var shared []string
	for code := range decoderCodes {
		if validatorCodes[code] {
			shared = append(shared, code)
		}
	}
	slices.Sort(shared)
	want := slices.Clone(sharedFindingCodes)
	slices.Sort(want)
	if !slices.Equal(shared, want) {
		t.Errorf("codes declared by both decoder and validator = %q, sharedFindingCodes = %q", shared, want)
	}
}

// codeConstants returns the values of the string constants named Code* that
// the package in dir declares outside its tests. It relies on both packages
// declaring every code as a Code* constant with a string literal value, and
// fails on one that is not, since its value would escape the comparison.
func codeConstants(t *testing.T, dir string) map[string]bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("list %s: %v", dir, err)
	}
	codes := make(map[string]bool)
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file) //nolint:gosec // G304: a source file of this module
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		f, err := goparser.ParseFile(fset, file, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for i, ident := range spec.Names {
				if !strings.HasPrefix(ident.Name, "Code") || i >= len(spec.Values) {
					continue
				}
				lit, ok := spec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("%s: %s is not a string literal; declare it as one so its value is compared",
						file, ident.Name)
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: %s: %v", file, ident.Name, err)
				}
				codes[value] = true
			}
			return true
		})
	}
	return codes
}
