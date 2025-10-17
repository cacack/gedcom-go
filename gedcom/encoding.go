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

	// EncodingLATIN1 represents Latin-1 (ISO-8859-1) character encoding
	EncodingLATIN1 Encoding = "LATIN1"

	// EncodingUNICODE is an alias for UTF-8 used in some GEDCOM files
	EncodingUNICODE Encoding = "UNICODE"
)

// String returns the string representation of the encoding.
func (e Encoding) String() string {
	return string(e)
}

// IsValid returns true if the encoding is a recognized GEDCOM encoding.
func (e Encoding) IsValid() bool {
	switch e {
	case EncodingUTF8, EncodingANSEL, EncodingASCII, EncodingLATIN1, EncodingUNICODE:
		return true
	default:
		return false
	}
}
