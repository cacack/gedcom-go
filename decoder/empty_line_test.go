package decoder

import (
	"fmt"
	"strings"
	"testing"
)

// blankSeparated returns a 5.5 document of n individuals with a blank line
// after the header and after every record, as the Perl Gedcom.pm module
// writes. The blank lines are on lines 6, 9, 12, ...: the header takes
// lines 1-5 and each record two lines plus its blank one.
func blankSeparated(n int) string {
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "0 @I%d@ INDI\n1 NAME Person%d /Example/\n\n", i, i)
	}
	sb.WriteString("0 TRLR\n")
	return sb.String()
}

// TestEmptyLineIsWarning is the #885 reproduction: a blank line between
// records is skipped without loss, so it is reported as a warning and a
// file whose only problem is blank lines has no errors.
func TestEmptyLineIsWarning(t *testing.T) {
	result, err := DecodeWithDiagnostics(strings.NewReader(blankSeparated(1)), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	want := Diagnostics{
		NewDiagnostic(6, SeverityWarning, CodeEmptyLine, "empty line", "\n"),
		NewDiagnostic(9, SeverityWarning, CodeEmptyLine, "empty line", "\n"),
	}
	if len(result.Diagnostics) != len(want) {
		t.Fatalf("diagnostics = %v, want %v", result.Diagnostics, want)
	}
	for i := range want {
		if got := result.Diagnostics[i]; got.Line != want[i].Line || got.Severity != want[i].Severity || got.Code != want[i].Code {
			t.Errorf("diagnostic[%d] = %+v, want %+v", i, got, want[i])
		}
	}
	if result.Diagnostics.HasErrors() {
		t.Error("HasErrors() = true, want false for a file whose only problem is blank lines")
	}
	if ind := result.Document.GetIndividual("@I1@"); ind == nil || len(ind.Names) != 1 {
		t.Errorf("@I1@ = %+v, want one name", ind)
	}
}

// TestEmptyLineSummarized checks that blank lines past MaxEmptyLineDiagnostics
// are counted in one closing diagnostic, at the first of them, as unknown tag
// repeats are.
func TestEmptyLineSummarized(t *testing.T) {
	const records = 30 // 31 blank lines: one after the header, one per record
	result, err := DecodeWithDiagnostics(strings.NewReader(blankSeparated(records)), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	diags := result.Diagnostics
	if len(diags) != MaxEmptyLineDiagnostics+1 {
		t.Fatalf("got %d diagnostics, want %d plus one summary: %v", len(diags), MaxEmptyLineDiagnostics, diags)
	}
	for i, d := range diags[:MaxEmptyLineDiagnostics] {
		if d.Code != CodeEmptyLine || d.Severity != SeverityWarning || d.Line != 6+3*i {
			t.Errorf("diagnostic[%d] = %+v, want an EMPTY_LINE warning on line %d", i, d, 6+3*i)
		}
	}
	firstOver := 6 + 3*MaxEmptyLineDiagnostics
	want := NewDiagnostic(firstOver, SeverityWarning, CodeEmptyLine,
		fmt.Sprintf("empty line (%d more empty lines, from this line on, skipped and not reported individually)", records+1-MaxEmptyLineDiagnostics), "")
	if got := diags[MaxEmptyLineDiagnostics]; got != want {
		t.Errorf("summary = %+v, want %+v", got, want)
	}
	if got := len(result.Document.Individuals()); got != records {
		t.Errorf("decoded %d individuals, want %d", got, records)
	}
}

// TestEmptyLineSummaryOrder checks that the summary follows the other parse
// diagnostics and precedes the entity-level ones.
func TestEmptyLineSummaryOrder(t *testing.T) {
	input := strings.Replace(blankSeparated(MaxEmptyLineDiagnostics+1), "0 TRLR\n", "garbage\n0 @I99@ INDI\n1 ZZZZ x\n0 TRLR\n", 1)
	result, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	var codes []string
	for _, d := range result.Diagnostics[MaxEmptyLineDiagnostics:] {
		codes = append(codes, d.Code)
	}
	want := []string{CodeSyntaxError, CodeEmptyLine, CodeUnknownTag}
	if strings.Join(codes, ",") != strings.Join(want, ",") {
		t.Errorf("codes after the per-line blank lines = %v, want %v", codes, want)
	}
}

// TestEmptyLineBudgetSeverity checks that blank lines the MaxDiagnostics
// budget leaves out count as warnings, so the truncation notice does not
// make a file with only blank lines report an error.
func TestEmptyLineBudgetSeverity(t *testing.T) {
	diags := decodeWithMax(t, blankSeparated(5), 2)
	if len(diags) != 3 {
		t.Fatalf("got %d diagnostics, want 2 plus a truncation notice: %v", len(diags), diags)
	}
	if notice := diags[2]; notice.Code != CodeDiagnosticsTruncated || notice.Severity != SeverityWarning {
		t.Errorf("notice = %+v, want a DIAGNOSTICS_TRUNCATED warning", notice)
	}
	if diags.HasErrors() {
		t.Error("HasErrors() = true, want false")
	}
}

// TestEmptyLineStrict checks that strict mode still rejects a blank line: no
// GEDCOM version's grammar has one.
func TestEmptyLineStrict(t *testing.T) {
	opts := DefaultOptions()
	opts.StrictMode = true
	if _, err := DecodeWithOptions(strings.NewReader(blankSeparated(1)), opts); err == nil || !strings.Contains(err.Error(), "empty line") {
		t.Errorf("strict decode error = %v, want an empty line error", err)
	}
}
