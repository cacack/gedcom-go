package decoder

import "context"

// ProgressCallback reports parsing progress during GEDCOM decoding.
// bytesRead is the cumulative bytes read so far.
// totalBytes is the expected total size, or -1 if unknown.
type ProgressCallback func(bytesRead, totalBytes int64)

// DecodeOptions provides configuration options for decoding GEDCOM files.
//
// The nesting ceiling is not configurable: the GEDCOM grammar's two-digit
// level field fixes it at parser.MaxNestingDepth-1 (99).
type DecodeOptions struct {
	// Context allows cancellation and timeout control
	Context context.Context

	// StrictMode controls how parsing errors are handled. It has the same
	// meaning for [DecodeWithOptions] and [DecodeWithDiagnostics]; [Decode]
	// uses the default (false). The entry points differ only in what they
	// return: DecodeWithDiagnostics also returns the diagnostics.
	//
	// When StrictMode is true:
	//   - Parsing fails on the first syntax error that stands. It stops
	//     reading there, except that an XRef containing a space is decided
	//     only once the header is known: GEDCOM 5.5 and 5.5.1 allow one, so
	//     it is not an error in a document whose header declares either
	//     version, and the first such XRef is the error otherwise
	//   - The error is returned with a nil document or result
	//   - Use for files that must be fully valid or rejected
	//
	// When StrictMode is false (default):
	//   - Parsing continues after encountering errors
	//   - Malformed lines are skipped; valid lines are preserved
	//   - Level-jump lines (e.g., `1 BIRT` then `4 DATE`) are clamped to
	//     prevLevel+1 and preserved as recovery, not skipped; a
	//     CodeBadLevelJump diagnostic is emitted (SeverityWarning)
	//   - An XRef containing a space (e.g. `0 @NoTe ref@ NOTE text`) in a
	//     document whose header does not declare GEDCOM 5.5 or 5.5.1, or a
	//     level-0 XRef with no closing `@` (e.g. `0 @I1 INDI`), is recovered
	//     verbatim and the record is preserved; a CodeInvalidXRef diagnostic
	//     is emitted (SeverityError, since the identifier itself is not
	//     spec-conformant). Verbatim means `@I1` is stored as `@I1`, which is
	//     not pointer-shaped, so `@I1@` references elsewhere in the file do
	//     not resolve to the recovered record. `0 @I1`, `0 @I1 HEAD` and
	//     `0 @I1 TRLR` are reported but not recovered: they keep their
	//     pre-existing parse, with the identifier as the tag
	//   - A well-formed XRef on a level-0 HEAD or TRLR line (e.g.
	//     `0 @X1@ HEAD`) is not the document's header or trailer — the GEDCOM
	//     grammar gives neither tag an identifier — so the line becomes an
	//     ordinary record with Type "HEAD"/"TRLR", keeping its subordinate
	//     lines; a CodeInvalidXRef diagnostic is emitted (SeverityError)
	//   - Diagnostics are collected for all issues encountered
	//   - Use [DecodeWithDiagnostics] to access diagnostics
	//   - A partial document is returned if any valid records exist
	//
	// Lenient mode (StrictMode=false) is recommended for importing vendor
	// GEDCOMs, which often contain non-standard extensions or formatting quirks.
	StrictMode bool

	// OnProgress is called periodically during parsing to report progress.
	// If nil, no progress reporting occurs (zero overhead).
	OnProgress ProgressCallback

	// TotalSize is the expected total size of the input in bytes.
	// Set to 0 (default) if unknown; will be reported as -1 to the callback.
	TotalSize int64
}

// DefaultOptions returns the default decoding options.
func DefaultOptions() *DecodeOptions {
	return &DecodeOptions{
		Context:    context.Background(),
		StrictMode: false,
	}
}
