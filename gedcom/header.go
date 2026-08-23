package gedcom

import "time"

// Header contains metadata about the GEDCOM file.
type Header struct {
	// Version is the GEDCOM specification version
	Version Version

	// Encoding is the character encoding used in the file
	Encoding Encoding

	// SourceSystem identifies the software that created the file
	SourceSystem string

	// Date is when the file was created
	Date time.Time

	// Language is the primary language used in the file (optional)
	Language string

	// Copyright notice (optional)
	Copyright string

	// Submitter reference (optional)
	Submitter string

	// AncestryTreeID is the Ancestry.com tree identifier from HEAD.SOUR._TREE.
	// This is an Ancestry.com vendor extension that identifies the family tree
	// this GEDCOM was exported from.
	AncestryTreeID string

	// Tags contains all raw header sub-tags in document order, providing a
	// lossless record of the header (including custom/unmapped tags such as
	// _RTLSAVE or header NOTEs) alongside the typed fields above.
	//
	// Tags is authoritative when encoding. The encoder writes the header from
	// Tags whenever it is non-empty, and consults the typed fields above only
	// for a document that has none. Editing a typed field on a decoded document
	// therefore does not change the encoded header — edit the matching tag, or
	// use the converter, which keeps both in step. See the encoder package
	// documentation for the full rule.
	Tags []*Tag
}
