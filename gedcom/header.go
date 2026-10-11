package gedcom

// Header contains metadata about the GEDCOM file.
type Header struct {
	// Version is the GEDCOM specification version. The decoder guesses it
	// from the file's tags when the header declares none, so do not use it
	// for identifier-grammar decisions: use Document.IsPointerXRef, which
	// reads DeclaredVersion.
	Version Version

	// DeclaredVersion is the version the header declares (HEAD.GEDC.VERS),
	// set by the decoder, by converter.Convert to a different version, and
	// by merge.Combine when doc1 declares none. It is empty when the file
	// declares no version or one this package does not recognize, so unlike
	// Version it is never a guess. It selects the identifier grammar
	// Document.IsPointerXRef applies: GEDCOM 5.5 and 5.5.1 allow spaces in
	// pointers ("@N 1@"); 7.0 and an empty value do not.
	//
	// Setting Version does not set it: code that builds a Document must set
	// DeclaredVersion itself to opt in to the 5.5/5.5.1 grammar, and code
	// that rewrites the header's GEDC.VERS must update it to match.
	DeclaredVersion Version

	// Encoding is the character encoding used in the file
	Encoding Encoding

	// SourceSystem identifies the software that created the file
	SourceSystem string

	// Date is the transmission date from HEAD.DATE, verbatim as the file
	// wrote it (for example "7 AUG 2026"). It is a GEDCOM DATE_EXACT value,
	// kept as a string so it round-trips losslessly like every other date in
	// this package (see ADR 0001); call ParseDate for a structured value.
	//
	// HEAD.DATE.TIME has no typed field: it is kept in Tags with the rest of
	// the header, which is what a decoded document is encoded from. A header
	// built in memory, with no Tags, gets "1 DATE" and no TIME line.
	Date string

	// Language is the primary language used in the file (optional)
	Language string

	// Copyright notice (optional)
	Copyright string

	// Submitter is the cross-reference pointer to the SUBM record that
	// names whoever produced the file, in "@U1@" form (optional).
	//
	// The value is interpreted and written verbatim: apart from rejecting a
	// line break (encoder.ErrLineBreak), the encoder emits whatever string is
	// here without checking its shape, so a value that is not a well-formed
	// pointer produces invalid GEDCOM. Readers that
	// resolve it -- Document.GetSubmitter, Document.Subset -- treat a
	// value they cannot resolve to a SUBM record as absent rather than
	// reporting it.
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
