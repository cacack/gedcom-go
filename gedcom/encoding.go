package gedcom

// Encoding represents character encoding used in a GEDCOM file.
type Encoding string

const (
	// EncodingUTF8 represents UTF-8 character encoding
	EncodingUTF8 Encoding = "UTF-8"

	// EncodingANSEL represents ANSEL (ANSI Z39.47) character encoding (GEDCOM 5.5 default)
	EncodingANSEL Encoding = "ANSEL"

	// EncodingASCII represents ASCII character encoding
	EncodingASCII Encoding = "ASCII"

	// EncodingLATIN1 represents a declared LATIN1 (ISO-8859-1) encoding. The
	// charset package decodes it as Windows-1252, as it does ANSI and ISO-8859-1
	EncodingLATIN1 Encoding = "LATIN1"

	// EncodingUNICODE is an alias for UTF-8 used in some GEDCOM files
	EncodingUNICODE Encoding = "UNICODE"

	// EncodingUTF16LE and EncodingUTF16BE name the UTF-16 byte orders the
	// decoder can apply. They appear only in Header.DetectedEncoding: they are
	// not HEAD.CHAR values, so IsValid reports false for them.
	EncodingUTF16LE Encoding = "UTF-16LE"
	EncodingUTF16BE Encoding = "UTF-16BE"

	// EncodingWindows1250, EncodingCP437, EncodingCP850 and EncodingMacRoman
	// name the other code pages the decoder can apply. Like the UTF-16
	// values they appear only in Header.DetectedEncoding, and IsValid
	// reports false for them.
	EncodingWindows1250 Encoding = "WINDOWS-1250"
	EncodingCP437       Encoding = "CP437"
	EncodingCP850       Encoding = "CP850"
	EncodingMacRoman    Encoding = "MACROMAN"
)

// String returns the string representation of the encoding.
func (e Encoding) String() string {
	return string(e)
}

// IsValid returns true if the encoding is one of the constants above. It does
// not report what the charset package can decode: Header.Encoding holds the
// CHAR value as written, so a value charset decodes (IBMPC, WINDOWS-1252,
// MACINTOSH and others) can still return false here.
func (e Encoding) IsValid() bool {
	switch e {
	case EncodingUTF8, EncodingANSEL, EncodingASCII, EncodingLATIN1, EncodingUNICODE:
		return true
	default:
		return false
	}
}
