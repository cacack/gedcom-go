package encoder

import "github.com/cacack/gedcom-go/gedcom"

// DefaultMaxLineLength is the recommended maximum line length for GEDCOM files.
// GEDCOM spec recommends lines not exceed 255 characters total.
// We use 248 to account for level number, space, tag, and delimiter overhead.
const DefaultMaxLineLength = 248

// EncodeOptions provides configuration for encoding GEDCOM files.
type EncodeOptions struct {
	// LineEnding specifies the line ending to use ("\r\n" or "\n")
	LineEnding string

	// MaxLineLength specifies the maximum length for line content before
	// splitting with CONC tags. Default is 248 characters.
	// Set to 0 to use the default value.
	MaxLineLength int

	// DisableLineWrap disables automatic CONC splitting for long lines.
	// When true, lines exceeding MaxLineLength will not be split.
	DisableLineWrap bool

	// TargetVersion specifies the GEDCOM version to target for output.
	// This can affect header generation and tag validity.
	// If empty, the version from the document header is preserved.
	TargetVersion gedcom.Version

	// PreserveUnknownTags controls whether custom/unknown tags are included
	// in the output. Custom tags are typically underscore-prefixed (e.g., _CUSTOM).
	// Default: true (preserve all tags)
	PreserveUnknownTags bool
}

// DefaultOptions returns the default encoding options.
func DefaultOptions() *EncodeOptions {
	return &EncodeOptions{
		LineEnding:          "\n",
		MaxLineLength:       DefaultMaxLineLength,
		DisableLineWrap:     false,
		PreserveUnknownTags: true,
	}
}

// effectiveMaxLineLength returns the max line length to use,
// defaulting to DefaultMaxLineLength if not set.
func (opts *EncodeOptions) effectiveMaxLineLength() int {
	if opts == nil || opts.MaxLineLength <= 0 {
		return DefaultMaxLineLength
	}
	return opts.MaxLineLength
}
