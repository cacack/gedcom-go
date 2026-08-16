package decoder

import "context"

// ProgressCallback reports parsing progress during GEDCOM decoding.
// bytesRead is the cumulative bytes read so far.
// totalBytes is the expected total size, or -1 if unknown.
type ProgressCallback func(bytesRead, totalBytes int64)

// DecodeOptions provides configuration options for decoding GEDCOM files.
type DecodeOptions struct {
	// Context allows cancellation and timeout control
	Context context.Context

	// MaxNestingDepth is deprecated and has no effect: the decoder never
	// reads it. The effective ceiling is parser.MaxNestingDepth-1, fixed by
	// the GEDCOM grammar's two-digit level field, so valid levels are 0-99 no
	// matter what this field is set to — setting it neither lowers nor raises
	// the ceiling. It is kept, and still defaulted to 100 by DefaultOptions,
	// for source compatibility through the v2 series.
	//
	// Deprecated: has no effect; the level ceiling is fixed at
	// parser.MaxNestingDepth-1 (99) by the GEDCOM grammar. Will be removed
	// in v3.
	MaxNestingDepth int

	// StrictMode controls how parsing errors are handled.
	//
	// When StrictMode is true:
	//   - Parsing fails immediately on the first syntax error
	//   - The error is returned from Decode/DecodeWithOptions
	//   - Use for files that must be fully valid or rejected
	//
	// When StrictMode is false (default):
	//   - Parsing continues after encountering errors
	//   - Malformed lines are skipped; valid lines are preserved
	//   - Level-jump lines (e.g., `1 BIRT` then `4 DATE`) are clamped to
	//     prevLevel+1 and preserved as recovery, not skipped; a
	//     CodeBadLevelJump diagnostic is emitted (SeverityWarning)
	//   - An XRef containing a space (e.g. `0 @NoTe ref@ NOTE text`) is
	//     recovered up to its closing `@` and the record is preserved; a
	//     CodeInvalidXRef diagnostic is emitted (SeverityError, since the
	//     identifier itself is not spec-conformant)
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
		Context:         context.Background(),
		MaxNestingDepth: 100,
		StrictMode:      false,
	}
}
