package gedcom

// Header contains metadata about the GEDCOM file.
type Header struct {
	// Version is the GEDCOM specification version. The decoder guesses it
	// from the file's tags when the header declares none, so do not use it
	// for identifier-grammar decisions: use Document.IsPointerXRef, which
	// reads DeclaredVersion.
	Version Version

	// DeclaredVersion is the version the header declares (HEAD.GEDC.VERS),
	// set by the decoder, by converter.Convert to a different version or
	// when it adds a missing GEDC.VERS, and by merge.Combine when doc1 declares none. It is empty when the file
	// declares no version or one this package does not recognize, so unlike
	// Version it is never a guess. A GEDCOM 7 patch level is dropped (7.0.14
	// is Version70), but a later 7 minor keeps its major.minor: VERS 7.1 is
	// "7.1" here while Version is 7.0, whose rules it is read by. A vendor
	// spelling of a known version, such as 5.51, is that version. It selects
	// the identifier grammar Document.IsPointerXRef applies: GEDCOM 5.5 and
	// 5.5.1 allow spaces in pointers ("@N 1@"); 7.x and an empty value do not.
	//
	// Setting Version does not set it, nor the reverse: the two are
	// independent fields. NewDocument sets both from one version; code that
	// builds a Header literal must set DeclaredVersion itself to opt in to the
	// 5.5/5.5.1 grammar, and code that rewrites the header's GEDC.VERS must
	// update it to match. For a header with no Tags the encoder writes
	// GEDC.VERS from Version (or EncodeOptions.TargetVersion), never from
	// DeclaredVersion.
	DeclaredVersion Version

	// Encoding is the character encoding the file declares (HEAD.CHAR),
	// verbatim. It may differ from the encoding the decoder applied: see
	// DetectedEncoding.
	Encoding Encoding

	// DetectedEncoding is the encoding the decoder actually decoded the
	// file's bytes as: EncodingUTF8, EncodingANSEL, EncodingASCII,
	// EncodingLATIN1, EncodingWindows1250, EncodingCP437, EncodingCP850,
	// EncodingMacRoman, EncodingUTF16LE or EncodingUTF16BE. It uses a
	// normalized vocabulary while Encoding is the declared token verbatim,
	// so the two often differ as strings without any conflict (for example
	// "ANSI" vs LATIN1, "UNICODE" vs UTF-16LE, or an absent CHAR vs UTF-8):
	// do not compare them to detect a mismatch. The signal for a declaration
	// the decoder overrode (a UTF-8 BOM, or bytes declared as ANSEL,
	// ASCII or a Windows-1252 value recognized as UTF-8) is the CHARSET_MISMATCH
	// diagnostic. It is empty for a header built in memory.
	//
	// It is a read view filled by the decoder and describes the decoded
	// input only: the encoder never writes it, and the converter and merge
	// do not maintain it (a converted header keeps the source's value, and a
	// merged header keeps doc1's).
	DetectedEncoding Encoding

	// SourceSystem identifies the software that created the file
	SourceSystem string

	// SourceVersion is the version of the source software (HEAD.SOUR.VERS),
	// verbatim. Empty when the tag is absent. It is a read view filled by the
	// decoder: Tags governs encoding, so setting it on a decoded document does
	// not change the encoded header, and the encoder never writes it for a
	// header built in memory.
	SourceVersion string

	// SourceName is the source software's product name (HEAD.SOUR.NAME),
	// verbatim. Empty when the tag is absent. Like SourceVersion, it is a read
	// view: Tags governs encoding and the encoder never writes it.
	SourceName string

	// SourceCorporation is the business that owns the source software
	// (HEAD.SOUR.CORP), or nil when the tag is absent. Like SourceVersion, it
	// is a read view: Tags governs encoding and the encoder never writes it.
	SourceCorporation *Corporation

	// SourceData is the electronic data source the file was taken from
	// (HEAD.SOUR.DATA), or nil when the tag is absent; a DATA with no
	// subordinates gives a non-nil value with empty fields. Its Copyright is
	// the source data's copyright, with CONT/CONC lines folded in, not the
	// file's: that is Copyright. Like
	// SourceVersion, it is a read view: Tags governs encoding and the encoder
	// never writes it.
	SourceData *HeaderSourceData

	// Destination is the system the file is intended for (HEAD.DEST),
	// verbatim. Empty when the tag is absent. Like SourceVersion, it is a read
	// view: Tags governs encoding and the encoder never writes it.
	Destination string

	// Date is the transmission date from HEAD.DATE, verbatim as the file
	// wrote it (for example "7 AUG 2026"). It is a GEDCOM DATE_EXACT value,
	// kept as a string so it round-trips losslessly like every other date in
	// this package (see ADR 0001); call ParseDate for a structured value.
	//
	// HEAD.DATE.TIME is typed separately, as Time.
	Date string

	// Time is the transmission time from HEAD.DATE.TIME, verbatim (for
	// example "13:45:02"). Empty when the tag is absent. It is a read view
	// filled by the decoder: Tags governs encoding, so setting it on a decoded
	// document does not change the encoded header, and a header built in
	// memory, with no Tags, gets "1 DATE" and no TIME line.
	Time string

	// Language is the primary language used in the file, from HEAD.LANG
	// (optional). A LANG nested deeper, such as a 7.0 HEAD.NOTE.LANG, is the
	// language of that structure and does not set it.
	Language string

	// Copyright is the file's copyright notice, from HEAD.COPR (optional).
	// HEAD.SOUR.DATA.COPR is the source data's copyright and is
	// SourceData.Copyright instead.
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

	// PlaceForm is the default place hierarchy from HEAD.PLAC.FORM, verbatim
	// (for example "City, County, State, Country"): the jurisdiction list that
	// interprets every PLAC value in the file that has no FORM of its own.
	// Empty when the tag is absent. Like SourceVersion, it is a read view:
	// Tags governs encoding and the encoder never writes it.
	PlaceForm string

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
	// therefore does not change the encoded header — call encoder.SyncTags
	// after the edit, edit the matching tag, or, to change version, use the
	// converter. See the encoder package documentation for the full rule.
	Tags []*Tag
}

// HeaderSourceData is the DATA structure under HEAD.SOUR: the electronic data
// source the file was taken from, such as a published CD. Each field is
// verbatim and empty when its tag is absent. Like the Header fields that hold
// it, it is a read view: Header.Tags governs encoding.
type HeaderSourceData struct {
	// Name is the name of the source data (the DATA value).
	Name string

	// Date is the source data's publication date (DATA.DATE), verbatim.
	Date string

	// Time is the time of that date (DATA.DATE.TIME, GEDCOM 7.0), verbatim.
	Time string

	// Copyright is the source data's copyright notice (DATA.COPR).
	Copyright string
}

// Corporation is the CORP structure under HEAD.SOUR: the business that owns
// the software that produced the file, with its address and contact details.
// Like the Header fields that hold it, it is a read view: Header.Tags governs
// encoding.
type Corporation struct {
	// Name is the corporation's name (the CORP value).
	Name string

	// Address is the corporation's postal address (ADDR), or nil.
	Address *Address

	// Phone holds the corporation's phone numbers (PHON, {0:3}), in document
	// order. PHON is a sibling of ADDR in ADDRESS_STRUCTURE, not part of it.
	Phone []string

	// Email holds the corporation's email addresses (EMAIL, {0:3}, GEDCOM
	// 5.5.1+), in document order.
	Email []string

	// Fax holds the corporation's fax numbers (FAX, {0:3}, GEDCOM 5.5.1+), in
	// document order.
	Fax []string

	// Website holds the corporation's web addresses (WWW, {0:3}, GEDCOM
	// 5.5.1+), in document order.
	Website []string
}
