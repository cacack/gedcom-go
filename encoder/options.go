package encoder

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// DefaultMaxLineLength is the recommended maximum line length for GEDCOM files.
// GEDCOM spec recommends lines not exceed 255 characters total.
// We use 248 to account for level number, space, tag, and delimiter overhead.
const DefaultMaxLineLength = 248

// EncodeOptions provides configuration for encoding GEDCOM files.
type EncodeOptions struct {
	// LineEnding specifies the line ending to use ("\r\n" or "\n").
	// Set to "" (the zero value) to use "\n".
	LineEnding string

	// MaxLineLength specifies the maximum length, in bytes, for line content
	// before splitting with CONC tags. Default is 248. Set to 0 to use the
	// default value. It has no effect on GEDCOM 7.0 output (TargetVersion, else
	// the header's version), which removed CONC and is never split.
	MaxLineLength int

	// DisableLineWrap disables automatic CONC splitting for long lines.
	// When true, lines exceeding MaxLineLength will not be split. GEDCOM 7.0
	// output is never split, so this only matters for 5.x.
	DisableLineWrap bool

	// TargetVersion specifies the GEDCOM version to target for output.
	// This can affect header generation and tag validity.
	// If empty, the version from the document header is preserved.
	TargetVersion gedcom.Version

	// DropUnknownTags controls whether custom/unknown tags are stripped from
	// the output. Custom tags are typically underscore-prefixed (e.g. _CUSTOM).
	//
	// The zero value keeps every tag. Together with LineEnding and
	// MaxLineLength defaulting on their own zero values, that makes a bare
	// &EncodeOptions{} lossless.
	//
	// When true, a custom tag is dropped along with everything subordinate to
	// it, and a record whose own type is a custom tag ("0 _ROOT", RootsMagic's
	// "0 _EVDEF") is dropped in full -- writing the level-0 line while stripping
	// its children would leave a stub that carries no data.
	DropUnknownTags bool

	// version is the GEDCOM version the records are being written as:
	// TargetVersion when set, else the header's own version. It is never set by
	// callers; EncodeWithOptions and StreamEncoder fill it on a private copy of
	// the options once the header is known (see withVersion), so the record
	// writers can tell 7.0 output, which has no CONC, from 5.x output.
	version gedcom.Version
}

// DefaultOptions returns the default encoding options.
func DefaultOptions() *EncodeOptions {
	return &EncodeOptions{
		LineEnding:      "\n",
		MaxLineLength:   DefaultMaxLineLength,
		DisableLineWrap: false,
	}
}

// effectiveLineEnding returns the line ending to use, defaulting to "\n" when
// unset. Without this, a hand-built &EncodeOptions{} wrote every line with no
// separator at all, producing a single unparseable line.
func (opts *EncodeOptions) effectiveLineEnding() string {
	if opts == nil || opts.LineEnding == "" {
		return "\n"
	}
	return opts.LineEnding
}

// effectiveMaxLineLength returns the max line length to use,
// defaulting to DefaultMaxLineLength if not set.
func (opts *EncodeOptions) effectiveMaxLineLength() int {
	if opts == nil || opts.MaxLineLength <= 0 {
		return DefaultMaxLineLength
	}
	return opts.MaxLineLength
}

// withVersion returns a copy of opts recording the version the records will
// be written as: TargetVersion when set, else header's version. The caller's
// options are never modified, so one EncodeOptions value can be shared across
// documents of different versions.
func (opts *EncodeOptions) withVersion(header *gedcom.Header) *EncodeOptions {
	c := *opts
	c.version = opts.TargetVersion
	if c.version == "" && header != nil {
		c.version = header.Version
	}
	return &c
}

// targetsV7 reports whether the output is GEDCOM 7.0, which removed CONC. A
// hand-built header may spell the version with a patch level ("7.0.14"), so
// any 7.x version counts.
func (opts *EncodeOptions) targetsV7() bool {
	return opts != nil && strings.HasPrefix(string(opts.version), "7.")
}
