package decoder

import (
	"errors"
	"fmt"
	"math"

	"github.com/cacack/gedcom-go/v3/parser"
)

// diagnosticBudget applies [DecodeOptions.MaxDiagnostics] to a lenient decode
// (#744). It bounds what is held while decoding, not only what is returned:
// the parser keeps at most limit of its errors (KeepError) and the
// collector at most limit diagnostics (keep), so a file of garbage lines
// costs memory in proportion to the bound, not to the file. apply then cuts
// the combined list to limit and appends the CodeDiagnosticsTruncated notice
// counting everything left out at any stage. Every omitted diagnostic passes
// through drop exactly once, so that count is exact.
//
// It is also where a lenient decode learns which rejected lines were level-0
// lines (levelZero): KeepError sees each line's full text, which a
// ParseError's clipped Context may not hold.
type diagnosticBudget struct {
	limit int

	// levelZero holds, in input order, the line number of every rejected line
	// that was meant as a level-0 line (isDroppedLevelZero), whether or not
	// its error was kept. skipOrphanedRecords skips the record each began.
	levelZero []int

	// parseKept is the number of bounded parse errors kept.
	parseKept int

	// suppressed counts the diagnostics left out; worst is the highest
	// severity among them, meaningful once suppressed > 0.
	suppressed int
	worst      Severity
}

// newDiagnosticBudget returns the budget for limit; a negative limit
// (unlimited) keeps everything.
func newDiagnosticBudget(limit int) *diagnosticBudget {
	if limit < 0 {
		limit = math.MaxInt
	}
	return &diagnosticBudget{limit: limit}
}

// keep reports whether a diagnostic of severity sev may join a list already
// holding n, and counts it as left out when it may not.
func (b *diagnosticBudget) keep(n int, sev Severity) bool {
	if n < b.limit {
		return true
	}
	b.drop(sev)
	return false
}

func (b *diagnosticBudget) drop(sev Severity) {
	if b.suppressed == 0 || sev < b.worst {
		b.worst = sev
	}
	b.suppressed++
}

// KeepError implements parser.ErrorFilter. It records each dropped level-0
// line in levelZero, from text, so the document does not depend on whether
// the error is kept or on how its Context is clipped. An XRef containing a
// space is kept whatever the budget: dropPermittedXRefErrors decides from it
// whether the identifier is valid for the version, and the parser keeps the
// line it rejected, so these errors grow with the document, not beyond it.
// apply still bounds the diagnostics they become. Every other parse error
// counts against the budget.
func (b *diagnosticBudget) KeepError(pe *parser.ParseError, text string) bool {
	if isDroppedLevelZero(text) {
		b.levelZero = append(b.levelZero, pe.Line)
	} else if errors.Is(pe, parser.ErrXRefContainsSpace) {
		return true
	}
	if !b.keep(b.parseKept, SeverityError) {
		return false
	}
	b.parseKept++
	return true
}

// apply returns the first limit of ds followed, when anything was left out
// here or earlier, by one CodeDiagnosticsTruncated diagnostic counting it. A
// nil budget (strict mode) returns ds unchanged. It is called once, on the complete list.
func (b *diagnosticBudget) apply(ds Diagnostics) Diagnostics {
	if b == nil {
		return ds
	}
	if len(ds) > b.limit {
		for _, d := range ds[b.limit:] {
			b.drop(d.Severity)
		}
		ds = ds[:b.limit:b.limit]
	}
	if b.suppressed == 0 {
		return ds
	}
	return append(ds, NewDiagnostic(0, b.worst, CodeDiagnosticsTruncated,
		fmt.Sprintf("%d further diagnostics not reported: the limit of %d (DecodeOptions.MaxDiagnostics) was reached",
			b.suppressed, b.limit), ""))
}
