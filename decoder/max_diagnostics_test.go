package decoder

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/parser"
)

// garbageInput returns a 5.5.1 document with n unparseable lines after the
// header, each "garbage line <i>", which the parser rejects one by one.
func garbageInput(n int) string {
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "garbage line %d\n", i)
	}
	sb.WriteString("0 TRLR\n")
	return sb.String()
}

func decodeWithMax(t *testing.T, input string, maxDiags int) Diagnostics {
	t.Helper()
	opts := DefaultOptions()
	opts.MaxDiagnostics = maxDiags
	result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	return result.Diagnostics
}

// TestMaxDiagnosticsDefault is the #744 acceptance probe: a lenient decode of
// input that is mostly garbage keeps DefaultMaxDiagnostics diagnostics and
// appends one DIAGNOSTICS_TRUNCATED notice counting the rest, instead of one
// diagnostic per line.
func TestMaxDiagnosticsDefault(t *testing.T) {
	const extra = 7
	diags := decodeWithMax(t, garbageInput(DefaultMaxDiagnostics+extra), 0)

	if len(diags) != DefaultMaxDiagnostics+1 {
		t.Fatalf("got %d diagnostics, want %d plus one notice", len(diags), DefaultMaxDiagnostics)
	}
	// The garbage starts on line 4; the last kept one is the cap's last.
	if last := diags[DefaultMaxDiagnostics-1]; last.Line != DefaultMaxDiagnostics+3 ||
		last.Context != fmt.Sprintf("garbage line %d", DefaultMaxDiagnostics-1) {
		t.Errorf("last kept diagnostic = line %d %q, want line %d, the cap's last garbage line",
			last.Line, last.Context, DefaultMaxDiagnostics+3)
	}
	notice := diags[DefaultMaxDiagnostics]
	want := NewDiagnostic(0, SeverityError, CodeDiagnosticsTruncated,
		fmt.Sprintf("%d further diagnostics not reported: the limit of %d (DecodeOptions.MaxDiagnostics) was reached", extra, DefaultMaxDiagnostics), "")
	if notice != want {
		t.Errorf("notice = %+v, want %+v", notice, want)
	}
}

// TestMaxDiagnosticsExplicit checks an explicit bound keeps the first
// MaxDiagnostics diagnostics in reported order, across the parse and entity
// phases, and that the notice counts what each phase left out.
func TestMaxDiagnosticsExplicit(t *testing.T) {
	// Two parse errors (lines 4, 5), then three unknown tags (lines 7-9).
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\nbad one\nbad two\n0 @I1@ INDI\n1 ZZZA a\n1 ZZZB b\n1 ZZZC c\n0 TRLR\n"

	diags := decodeWithMax(t, input, 3)
	var got []string
	for _, d := range diags {
		got = append(got, fmt.Sprintf("%d %s %s", d.Line, d.Code, d.Severity))
	}
	want := []string{
		"4 INVALID_LEVEL ERROR",
		"5 INVALID_LEVEL ERROR",
		"7 UNKNOWN_TAG WARNING",
		"0 DIAGNOSTICS_TRUNCATED WARNING",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diagnostics = %q, want %q", got, want)
	}
	if msg := diags[3].Message; !strings.HasPrefix(msg, "2 further diagnostics not reported: the limit of 3 ") {
		t.Errorf("notice message = %q, want 2 left out of a limit of 3", msg)
	}

	// The bound of one stops at the first parse error; the notice takes the
	// severity of the parse error it left out, so HasErrors still holds.
	diags = decodeWithMax(t, input, 1)
	if len(diags) != 2 || diags[0].Line != 4 || diags[1].Code != CodeDiagnosticsTruncated ||
		diags[1].Severity != SeverityError || !strings.HasPrefix(diags[1].Message, "4 further ") {
		t.Errorf("diagnostics with a bound of 1 = %v, want line 4 then a notice counting 4 at ERROR", diags)
	}
}

// TestMaxDiagnosticsUnlimited checks UnlimitedDiagnostics reports every
// diagnostic and no notice.
func TestMaxDiagnosticsUnlimited(t *testing.T) {
	const n = DefaultMaxDiagnostics + 5
	diags := decodeWithMax(t, garbageInput(n), UnlimitedDiagnostics)
	if len(diags) != n {
		t.Fatalf("got %d diagnostics, want all %d", len(diags), n)
	}
	if last := diags[n-1]; last.Code != CodeInvalidLevel || last.Context != fmt.Sprintf("garbage line %d", n-1) {
		t.Errorf("last diagnostic = %v, want the last garbage line", last)
	}
}

// TestMaxDiagnosticsNoLines checks the bound applies when nothing parses, the
// path that returns an empty document and the first parse error.
func TestMaxDiagnosticsNoLines(t *testing.T) {
	opts := DefaultOptions()
	opts.MaxDiagnostics = 2
	result, err := DecodeWithDiagnostics(strings.NewReader("a\nb\nc\nd\n"), opts)
	if err == nil {
		t.Fatal("DecodeWithDiagnostics() error = nil, want no valid lines")
	}
	var got []string
	for _, d := range result.Diagnostics {
		got = append(got, fmt.Sprintf("%d %s", d.Line, d.Code))
	}
	if want := []string{"1 SYNTAX_ERROR", "2 SYNTAX_ERROR", "0 DIAGNOSTICS_TRUNCATED"}; !reflect.DeepEqual(got, want) {
		t.Errorf("diagnostics = %q, want %q", got, want)
	}
}

// TestMaxDiagnosticsDocumentUnchanged checks the bound limits only what is
// reported: a level-0 line dropped after the bound is reached still has its
// record skipped, rather than its subordinate lines attached to the record
// before it, and the decoded document matches the unbounded decode.
func TestMaxDiagnosticsDocumentUnchanged(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\nbad one\nbad two\n" +
		"0 @I1@ INDI\n1 NAME First /One/\n" +
		"x0 @I2@ INDI\n1 NAME Second /Two/\n" +
		"0 TRLR\n"

	decodeDoc := func(maxDiags int) []string {
		opts := DefaultOptions()
		opts.MaxDiagnostics = maxDiags
		result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)
		if err != nil {
			t.Fatalf("DecodeWithDiagnostics() error = %v", err)
		}
		var names []string
		for _, ind := range result.Document.Individuals() {
			for _, n := range ind.Names {
				names = append(names, ind.XRef+" "+n.Full)
			}
		}
		return names
	}

	want := []string{"@I1@ First /One/"}
	if got := decodeDoc(UnlimitedDiagnostics); !reflect.DeepEqual(got, want) {
		t.Fatalf("unbounded names = %q, want %q", got, want)
	}
	if got := decodeDoc(1); !reflect.DeepEqual(got, want) {
		t.Errorf("names with a bound of 1 = %q, want %q: the dropped @I2@ line must still skip its record", got, want)
	}
}

// TestMaxDiagnosticsPermittedXRefNotCounted checks that an XRef containing a
// space, which the parser reports but GEDCOM 5.5.1 allows, is not counted as
// a diagnostic left out once the bound is reached.
func TestMaxDiagnosticsPermittedXRefNotCounted(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\nbad one\nbad two\n" +
		"0 @I 1@ INDI\n1 NAME First /One/\n0 TRLR\n"
	diags := decodeWithMax(t, input, 1)
	if len(diags) != 2 || !strings.HasPrefix(diags[1].Message, "1 further ") {
		t.Errorf("diagnostics = %v, want line 4 then a notice counting only line 5", diags)
	}
}

// TestDiagnosticClipsEchoedInput checks a diagnostic does not echo a long
// value whole (#744): a 1 MiB coordinate used to give a 2 MB error string,
// quoted once in the message and again in the context.
func TestDiagnosticClipsEchoedInput(t *testing.T) {
	value := "N" + strings.Repeat("9", parser.MaxLineBytes-20)
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 BIRT\n2 PLAC Here\n3 MAP\n4 LATI " + value + "\n0 TRLR\n"

	result, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want the one INVALID_VALUE: %v", len(result.Diagnostics), result.Diagnostics)
	}
	d := result.Diagnostics[0]
	if want := value[:parser.MaxContextBytes] + "…"; d.Context != want {
		t.Errorf("Context has length %d, want the first %d bytes and an ellipsis", len(d.Context), parser.MaxContextBytes)
	}
	if len(d.Message) > parser.MaxContextBytes+len("…") || !strings.HasPrefix(d.Message, "invalid value for LATI: ") {
		t.Errorf("Message = %.80q... (length %d), want it clipped to %d bytes", d.Message, len(d.Message), parser.MaxContextBytes)
	}
	if n := len(d.Error()); n > 4*parser.MaxContextBytes {
		t.Errorf("Error() is %d bytes, want it bounded by the clipped fields", n)
	}
	// The typed value itself is not clipped (ADR 0003).
	birth := result.Document.GetIndividual("@I1@").Events[0]
	if got := birth.PlaceDetail.Coordinates.Latitude; got != value {
		t.Errorf("Latitude has length %d, want the full %d", len(got), len(value))
	}
}

// TestNewDiagnosticClips checks NewDiagnostic clips Message and Context and
// leaves short ones alone.
func TestNewDiagnosticClips(t *testing.T) {
	long := strings.Repeat("é", parser.MaxContextBytes) // two bytes each
	d := NewDiagnostic(1, SeverityWarning, CodeInvalidValue, long, long)
	want := strings.Repeat("é", parser.MaxContextBytes/2) + "…"
	if d.Message != want || d.Context != want {
		t.Errorf("NewDiagnostic() = %q / %q, want both clipped to %q", d.Message, d.Context, want)
	}
	short := NewDiagnostic(1, SeverityWarning, CodeInvalidValue, "msg", "ctx")
	if short.Message != "msg" || short.Context != "ctx" {
		t.Errorf("NewDiagnostic() changed short fields: %+v", short)
	}
}

// TestMaxDiagnosticsBoundsMemory is the #744 memory probe: 18 MB of garbage
// lines used to give 1,000,000 diagnostics and over 400 MB RSS, the result
// alone holding about 88 MB of heap. The process now peaks near 60 MB RSS;
// this checks the heap the result holds, which must stay under 32 MB.
// Skipped in short mode and under the race detector, which inflates memory.
func TestMaxDiagnosticsBoundsMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 18 MB memory probe in short mode")
	}
	if raceDetectorEnabled {
		t.Skip("Skipping memory probe under the race detector (memory amplification)")
	}
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n")
	for sb.Len() < 18<<20 {
		sb.WriteString("garbage line here\n")
	}
	input := sb.String()

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	result, err := DecodeWithDiagnostics(strings.NewReader(input), DefaultOptions())
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	held := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	runtime.KeepAlive(result)
	runtime.KeepAlive(input)

	if len(result.Diagnostics) != DefaultMaxDiagnostics+1 {
		t.Errorf("got %d diagnostics, want %d plus one notice", len(result.Diagnostics), DefaultMaxDiagnostics)
	}
	if held > 32<<20 {
		t.Errorf("result holds %d MB of heap, want under 32 MB", held>>20)
	}
	t.Logf("result holds %d MB of heap", held>>20)
}

// TestMaxDiagnosticsBoundsDroppedLevelZeroErrors checks that rejected lines
// shaped like level-0 lines count against the bound while parsing (#744):
// the parser keeps only the first limit of their errors, yet every one still
// skips its record and is counted in the notice.
func TestMaxDiagnosticsBoundsDroppedLevelZeroErrors(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&sb, "x0 @I%d@ INDI\n1 NAME N%d\n", i, i)
	}
	sb.WriteString("0 TRLR\n")
	input := sb.String()

	budget := newDiagnosticBudget(2)
	_, errs, fatal := parser.NewParser().ParseWithOptions(strings.NewReader(input),
		&parser.ParseOptions{Lenient: true, Filter: budget})
	if fatal != nil {
		t.Fatalf("ParseWithOptions() fatal = %v", fatal)
	}
	var kept []int
	for _, pe := range errs {
		kept = append(kept, pe.Line)
	}
	if want := []int{4, 6}; !reflect.DeepEqual(kept, want) {
		t.Errorf("kept parse errors on lines %v, want %v", kept, want)
	}
	if want := []int{4, 6, 8, 10, 12}; !reflect.DeepEqual(budget.levelZero, want) {
		t.Errorf("levelZero = %v, want %v", budget.levelZero, want)
	}

	opts := DefaultOptions()
	opts.MaxDiagnostics = 2
	result, err := DecodeWithDiagnostics(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	if n := len(result.Document.Individuals()); n != 0 {
		t.Errorf("got %d individuals, want every record skipped", n)
	}
	// 5 parse errors and 5 skipped records; 2 are reported.
	if got := result.Diagnostics; len(got) != 3 || got[0].Line != 4 || got[1].Line != 6 ||
		!strings.HasPrefix(got[2].Message, "8 further ") {
		t.Errorf("diagnostics = %v, want lines 4 and 6 then a notice counting 8", got)
	}
}
