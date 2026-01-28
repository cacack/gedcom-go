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

	// MaxNestingDepth sets the maximum allowed nesting depth (default: 100)
	// This prevents stack overflow with malformed files
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
