package decoder

import (
	"errors"
	"fmt"

	"github.com/cacack/gedcom-go/v3/parser"
)

// diagnosticBudget applies [DecodeOptions.MaxDiagnostics] to a lenient decode
// (#744). It bounds what is held while decoding, not only what is returned:
// the parser keeps at most limit of its errors (KeepError) and the
// collector at most limit diagnostics (keep), so a file of garbage lines
// costs memory in proportion to the bound, not to the file. apply then cuts
// the combined list to limit and appends the CodeDiagnosticsTruncated notice
// counting everything left out at any stage.
type diagnosticBudget struct {
	limit int

	// parseKept is the number of bounded parse errors kept.
	parseKept int

	// suppressed counts the diagnostics left out; worst is the highest
	// severity among them, meaningful once suppressed > 0.
	suppressed int
	worst      Severity
}

// newDiagnosticBudget returns the budget for limit, or nil, which keeps
// everything, when limit is negative (unlimited).
func newDiagnosticBudget(limit int) *diagnosticBudget {
	if limit < 0 {
		return nil
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

// KeepError implements parser.ErrorFilter. Two kinds of error are kept
// whatever the budget, because the document is built from them, so bounding
// them would make the document depend on MaxDiagnostics: an XRef containing a
// space (dropPermittedXRefErrors decides from it whether the identifier is
// valid for the version) and a dropped level-0 line (skipOrphanedRecords
// skips the record it began). apply still bounds the diagnostics they become.
// Every other parse error counts against the budget.
func (b *diagnosticBudget) KeepError(pe *parser.ParseError) bool {
	if errors.Is(pe, parser.ErrXRefContainsSpace) || isDroppedLevelZero(pe) {
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
// nil budget returns ds unchanged. It is called once, on the complete list.
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
