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
	// Context allows cancellation and timeout control. It is checked before
	// the decode starts, on every read of the input and before each record's
	// entity is built, so a cancelled or expired context stops the decode
	// within one read buffer or one record. The decode then returns the
	// context's error (match it with errors.Is against context.Canceled or
	// context.DeadlineExceeded) and no document. Nil means no cancellation.
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
	//   - A byte that is not valid UTF-8, in input read as UTF-8 (declared
	//     UTF-8 or ASCII, or no recognized CHAR), fails the decode: the
	//     error wraps a *charset.InvalidUTF8Error giving its line, column and
	//     value
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
	//     not resolve to the recovered record. `0 @I1`, with no tag, is
	//     reported but not recovered: it keeps its pre-existing parse, with
	//     the identifier as the tag
	//   - An XRef on a level-0 HEAD or TRLR line (e.g. `0 @X1@ HEAD`, or a
	//     recovered `0 @I1 HEAD`) is not the document's header or trailer —
	//     the GEDCOM grammar gives neither tag an identifier — so the line
	//     becomes an ordinary record with Type "HEAD"/"TRLR", keeping its
	//     subordinate lines; a CodeInvalidXRef diagnostic is emitted
	//     (SeverityError), in addition to the one for a malformed identifier
	//   - A byte that is not valid UTF-8, in input read as UTF-8 (declared
	//     UTF-8 or ASCII, or no recognized CHAR), is replaced with U+FFFD and
	//     decoding continues, so a stray Windows-1252 byte costs one character
	//     rather than the rest of the file; a CodeInvalidEncoding diagnostic
	//     (SeverityError) is emitted per byte, at its line, with its column in
	//     the message, up to [MaxInvalidEncodingDiagnostics]; one further
	//     diagnostic counts the bytes past that
	//   - Diagnostics are collected for all issues encountered, up to
	//     MaxDiagnostics
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

	// MaxDiagnostics bounds the diagnostics a lenient decode reports, so that
	// input which is mostly malformed cannot make them, and the memory they
	// hold, grow with the file (#744). Zero means [DefaultMaxDiagnostics];
	// [UnlimitedDiagnostics] (any negative value) removes the bound.
	//
	// The first MaxDiagnostics diagnostics, in the order they are reported,
	// are kept. When any are left out, one CodeDiagnosticsTruncated
	// diagnostic is appended after them, counting those left out, so the
	// result holds at most MaxDiagnostics+1 entries. The document is the same
	// whatever the bound: it limits only what is reported. It has no effect
	// in strict mode, which reports no diagnostics.
	MaxDiagnostics int
}

// DefaultMaxDiagnostics is the number of diagnostics a lenient decode reports
// when [DecodeOptions.MaxDiagnostics] is zero.
//
// Real exports can report many: testdata/edge-cases/mhftb8-export.ged, a
// 2.4 MB MyHeritage file, reports 14,597, nearly all UNKNOWN_TAG warnings for
// vendor tags, so the bound is set well above that. It still caps the memory
// diagnostics hold at a few tens of megabytes whatever the input: 18 MB of
// garbage lines, which gave 1,000,000 diagnostics and about 400 MB of memory
// unbounded, decodes in about 60 MB.
const DefaultMaxDiagnostics = 100000

// UnlimitedDiagnostics removes the diagnostics bound when assigned to
// [DecodeOptions.MaxDiagnostics]. It exists because the field's zero value
// means "use DefaultMaxDiagnostics", not "no limit", so opting out has to be
// explicit. Only use it on input you control: on hostile input the
// diagnostics grow with the file.
const UnlimitedDiagnostics = -1

// diagnosticsLimit resolves MaxDiagnostics: the bound to apply, or a negative
// value for none.
func (o *DecodeOptions) diagnosticsLimit() int {
	if o.MaxDiagnostics == 0 {
		return DefaultMaxDiagnostics
	}
	return o.MaxDiagnostics
}

// DefaultOptions returns the default decoding options.
func DefaultOptions() *DecodeOptions {
	return &DecodeOptions{
		Context:    context.Background(),
		StrictMode: false,
	}
}
