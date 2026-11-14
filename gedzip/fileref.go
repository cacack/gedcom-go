package gedzip

import (
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

// RefKind is what a GEDCOM 7 FILE or FILE.TRAN payload refers to, as decided by
// [ClassifyFileRef] (#127).
type RefKind int

const (
	// RefInvalid is a payload that is none of the other kinds: it has a scheme
	// other than ftp, http, https or file, or it has no scheme and breaks one
	// of the local file rules. It is the zero value.
	RefInvalid RefKind = iota
	// RefLocal is a local file: a payload with no scheme that names an entry
	// of the archive.
	RefLocal
	// RefWeb is a web file: a payload with an ftp, http or https scheme. It has
	// no archive entry.
	RefWeb
	// RefFileURL is a payload with a file scheme. It names a file on the
	// machine that wrote the dataset, has no archive entry, and must not be
	// used in a GEDZIP.
	RefFileURL
)

// String returns "invalid", "local", "web" or "file-url", and "RefKind(n)" for
// a value that is not one of the constants.
func (k RefKind) String() string {
	switch k {
	case RefInvalid:
		return "invalid"
	case RefLocal:
		return "local"
	case RefWeb:
		return "web"
	case RefFileURL:
		return "file-url"
	default:
		return "RefKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// ClassifyFileRef reports what ref, the payload of a GEDCOM 7 FILE or FILE.TRAN
// line, refers to, following the GEDCOM 7.0.18 "File Path" datatype (#127).
// entryName is the zip entry the payload names; it is non-empty only for
// [RefLocal].
//
// A payload with a scheme is classified by the scheme alone, compared without
// regard to case: ftp, http and https are [RefWeb], file is [RefFileURL], and
// every other scheme is [RefInvalid]. That includes a single letter, so a
// Windows path such as `C:\x` or `C:/x` is invalid rather than local.
//
// A payload with no scheme is [RefLocal] only if it is a clean relative path:
// it does not begin with "/", has no ".." segment, no backslash and no banned
// character, whether written literally or percent-escaped, and no query or
// fragment. The banned characters are U+0000 to U+0008, U+000B, U+000C, U+000E
// to U+001F, U+007F to U+009F, the surrogates, U+FFFE and U+FFFF.
//
// The entry name is the payload percent-decoded once, so `50%2520off.jpg` names
// the entry `50%20off.jpg`, and `a b.jpg` and `a%20b.jpg` name the same entry.
// It is valid UTF-8 and is meant to be matched byte-exact: nothing is case
// folded, Unicode normalised or trimmed.
//
// Where the datatype is silent the stricter reading is taken, and each of these
// is [RefInvalid]:
//   - the empty string, a trailing "/", and an empty or "." segment;
//   - a malformed percent escape, and a payload that is not valid UTF-8
//     either as written or once decoded;
//   - an escaped separator, %2F or %5C, so that decoding can neither create
//     nor hide a path separator;
//   - a tab, line feed or carriage return, literal or escaped, so that an
//     entry name never holds a control character;
//   - a ":" in the first segment that is not the end of a scheme, as in
//     `1:x`, which RFC 3986 does not allow in a relative reference.
//
// An entry name is therefore never absolute and never contains a "..", "." or
// empty segment, a backslash or a control character.
func ClassifyFileRef(ref string) (kind RefKind, entryName string) {
	if scheme, ok := splitScheme(ref); ok {
		return classifyScheme(scheme), ""
	}
	name, ok := localEntryName(ref)
	if !ok {
		return RefInvalid, ""
	}
	return RefLocal, name
}

// splitScheme returns the URI scheme ref starts with, without its colon. The
// grammar is RFC 3986's, `[A-Za-z][A-Za-z0-9+.-]*:`. It is split by hand
// because net/url.Parse is more lenient than the File Path datatype.
func splitScheme(ref string) (scheme string, ok bool) {
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z':
		case '0' <= c && c <= '9', c == '+', c == '.', c == '-':
			if i == 0 {
				return "", false
			}
		case c == ':':
			return ref[:i], i > 0
		default:
			return "", false
		}
	}
	return "", false
}

// classifyScheme maps a scheme to its kind, ignoring case.
func classifyScheme(scheme string) RefKind {
	switch strings.ToLower(scheme) {
	case "ftp", "http", "https":
		return RefWeb
	case "file":
		return RefFileURL
	default:
		return RefInvalid
	}
}

// localEntryName returns the entry name of a payload that has no scheme, and
// false if the payload is not a valid local file. It validates the raw payload,
// decodes it, then validates the decoded form, so neither a literal nor an
// escaped spelling of a forbidden construct gets through.
func localEntryName(ref string) (name string, ok bool) {
	first, _, _ := strings.Cut(ref, "/")
	if strings.Contains(first, ":") || strings.ContainsAny(ref, "?#") {
		return "", false
	}
	if hasEscapedSeparator(ref) || !isCleanRelative(ref) {
		return "", false
	}
	name, err := url.PathUnescape(ref)
	if err != nil || !isCleanRelative(name) {
		return "", false
	}
	return name, true
}

// hasEscapedSeparator reports whether ref percent-escapes "/" or "\". In a
// payload whose escapes are well formed, every "%" starts an escape made of the
// next two characters, so the substrings match exactly those escapes; a
// malformed payload is rejected later whatever this returns.
func hasEscapedSeparator(ref string) bool {
	upper := strings.ToUpper(ref)
	return strings.Contains(upper, "%2F") || strings.Contains(upper, "%5C")
}

// isCleanRelative reports whether s is valid UTF-8, has no backslash or
// forbidden character, and is a relative path whose segments are all non-empty
// and none of them "." or "..". An empty segment covers the empty string, a
// leading or trailing "/" and "//".
func isCleanRelative(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r == '\\' || isForbiddenRune(r) {
			return false
		}
	}
	for seg := range strings.SplitSeq(s, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

// isForbiddenRune reports whether r is a File Path banned character or one of
// the three C0 controls the datatype does not ban (tab, line feed, carriage
// return). Surrogates are banned too, but cannot occur in valid UTF-8, which
// isCleanRelative checks first.
func isForbiddenRune(r rune) bool {
	return r < 0x20 || (0x7F <= r && r <= 0x9F) || r == 0xFFFE || r == 0xFFFF
}
