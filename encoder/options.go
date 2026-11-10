package encoder

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/concsplit"
	"github.com/cacack/gedcom-go/v3/internal/gedcvers"
)

// DefaultMaxLineLength is the recommended maximum line length for GEDCOM files.
// GEDCOM spec recommends lines not exceed 255 characters total.
// We use 248 to account for level number, space, tag, and delimiter overhead.
const DefaultMaxLineLength = concsplit.MaxValueLength

// EncodeOptions provides configuration for encoding GEDCOM files.
type EncodeOptions struct {
	// LineEnding specifies the line ending to use ("\r\n" or "\n").
	// Set to "" (the zero value) to use "\n".
	LineEnding string

	// MaxLineLength specifies the maximum length, in bytes, for line content
	// before splitting with CONC tags. Default is 248.
	// Set to 0 to use the default value.
	//
	// It applies only to GEDCOM 5.5 and 5.5.1 output. GEDCOM 7.0 removed CONC
	// and has no line-length limit, so 7.0 output (TargetVersion, or the
	// header's version when no target is set) is never split.
	MaxLineLength int

	// DisableLineWrap disables automatic CONC splitting for long lines.
	// When true, lines exceeding MaxLineLength will not be split.
	// GEDCOM 7.0 output behaves as if this were true.
	DisableLineWrap bool

	// TargetVersion specifies the GEDCOM version to target for output.
	// This can affect header generation and tag validity.
	// If empty, the version from the document header is preserved.
	//
	// A record written from its typed entity (Record.Tags empty) gets only the
	// version-specific structures the output version defines; see the package
	// documentation for the full list (name variations, DATE PHRASE). With no
	// known output version (no TargetVersion and no header version) nothing is
	// dropped. The encoder does not report a drop; the converter package
	// reports data loss on a conversion. A record written from its
	// Record.Tags is written verbatim.
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

	// version is the resolved output version, set by forOutput. Writers must
	// be reached only with options forOutput returned; on any other options it
	// is empty, which reads as an unknown version (not 7.0, lacking nothing).
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

// outputVersion returns the GEDCOM version output is written as:
// opts.TargetVersion, or the header's version when no target is set -- the
// same version the header's GEDC.VERS is written with, exactly as given.
func outputVersion(header *gedcom.Header, opts *EncodeOptions) gedcom.Version {
	if opts != nil && opts.TargetVersion != "" {
		return opts.TargetVersion
	}
	if header != nil {
		return header.Version
	}
	return ""
}

// gateVersion returns the version the encoder's version gates compare against
// for output version v: the version gedcvers.Parse recognizes it as, the rule
// the decoder applies to GEDC.VERS, so a GEDCOM 7 patch version such as 7.0.14
// is 7.0 (issue #696). A value it does not recognize, such as 7.1, is kept as
// given: it is not 7.0, and not any other version either. So is a value with
// surrounding whitespace, which Parse would trim: the gates read it exactly as
// they did before #696, as an unrecognized version.
func gateVersion(v gedcom.Version) gedcom.Version {
	if strings.TrimSpace(string(v)) != string(v) {
		return v
	}
	if known := gedcvers.Parse(string(v)); known != "" {
		return known
	}
	return v
}

// forOutput returns the options to write a document with header: a copy of
// opts carrying the resolved output version (see outputVersion and
// gateVersion), with DisableLineWrap set for GEDCOM 7.0 output because 7.0
// removed CONC and has no line-length limit (issue #565). The caller's options
// are never modified.
func (opts *EncodeOptions) forOutput(header *gedcom.Header) *EncodeOptions {
	resolved := *opts
	resolved.version = gateVersion(outputVersion(header, opts))
	if resolved.version == gedcom.Version70 {
		resolved.DisableLineWrap = true
	}
	return &resolved
}

// is70 reports whether output is GEDCOM 7.0, per the version forOutput
// resolved. Nil options, and options forOutput has not resolved, are not 7.0.
func (opts *EncodeOptions) is70() bool {
	return opts != nil && opts.version == gedcom.Version70
}

// outputIs reports whether output is GEDCOM version v, per the version
// forOutput resolved. An unknown output version is no version.
func (opts *EncodeOptions) outputIs(v gedcom.Version) bool {
	return opts != nil && opts.version == v
}

// outputLacks reports whether a structure defined only by GEDCOM version
// defining must be dropped: the output version forOutput resolved is known
// and is not defining. An unknown output version (nil or unresolved options,
// or no TargetVersion and no header version) lacks nothing, so a writer
// gating on it drops nothing for such output.
func (opts *EncodeOptions) outputLacks(defining gedcom.Version) bool {
	return opts != nil && opts.version != "" && opts.version != defining
}

// effectiveMaxLineLength returns the max line length to use,
// defaulting to DefaultMaxLineLength if not set.
func (opts *EncodeOptions) effectiveMaxLineLength() int {
	if opts == nil || opts.MaxLineLength <= 0 {
		return DefaultMaxLineLength
	}
	return opts.MaxLineLength
}
