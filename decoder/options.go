package decoder

import "context"

// DecodeOptions provides configuration options for decoding GEDCOM files.
type DecodeOptions struct {
	// Context allows cancellation and timeout control
	Context context.Context

	// MaxNestingDepth sets the maximum allowed nesting depth (default: 100)
	// This prevents stack overflow with malformed files
	MaxNestingDepth int

	// StrictMode enables strict parsing (reject non-standard extensions)
	StrictMode bool
}

// DefaultOptions returns the default decoding options.
func DefaultOptions() *DecodeOptions {
	return &DecodeOptions{
		Context:         context.Background(),
		MaxNestingDepth: 100,
		StrictMode:      false,
	}
}
