package gedzip

import (
	"strings"
	"testing"
)

var classifyFileRefTests = []struct {
	name      string
	ref       string
	wantKind  RefKind
	wantEntry string
}{
	// Local files.
	{"spec example, escaped non-ASCII", "media/CharlotteBront%C3%AB.jpg", RefLocal, "media/CharlotteBrontë.jpg"},
	{"literal non-ASCII", "media/CharlotteBrontë.jpg", RefLocal, "media/CharlotteBrontë.jpg"},
	{"maximal70 local path", "path/to/file1", RefLocal, "path/to/file1"},
	{"single segment", "photo.jpg", RefLocal, "photo.jpg"},
	{"escaped percent decodes once", "50%2520off.jpg", RefLocal, "50%20off.jpg"},
	{"literal space", "a b.jpg", RefLocal, "a b.jpg"},
	{"escaped space", "a%20b.jpg", RefLocal, "a b.jpg"},
	{"lower-case hex escape", "caf%c3%a9.jpg", RefLocal, "café.jpg"},
	{"case is preserved", "Media/Photo.JPG", RefLocal, "Media/Photo.JPG"},
	{"surrounding spaces are kept", " a ", RefLocal, " a "},
	{"dots inside a segment", "a..b/.hidden/c.", RefLocal, "a..b/.hidden/c."},
	{"three dots is an ordinary segment", ".../x", RefLocal, ".../x"},
	{"colon after the first segment", "media/a:b.jpg", RefLocal, "media/a:b.jpg"},
	{"escaped colon in the first segment", "a%3Ab", RefLocal, "a:b"},
	{"escaped query and fragment characters", "a%3Fb%23c", RefLocal, "a?b#c"},
	{"last character before U+FFFE", "a\uFFFDb", RefLocal, "a\uFFFDb"},
	{"astral character", "a\U0001F600b", RefLocal, "a\U0001F600b"},
	{"the dataset name is not special here", "gedcom.ged", RefLocal, "gedcom.ged"},

	// Web files.
	{"maximal70 web path", "http://host.example.com/path/to/file2", RefWeb, ""},
	{"scheme is case-insensitive", "HTTP://x/y", RefWeb, ""},
	{"https with nothing after it", "https:", RefWeb, ""},
	{"ftp with nothing after it", "ftp:", RefWeb, ""},
	{"web path is not validated further", "https://x/../a\\b?q#f", RefWeb, ""},

	// file: URLs.
	{"file URL, empty host", "file:///x", RefFileURL, ""},
	{"file URL with host", "file://host/x", RefFileURL, ""},
	{"file scheme is case-insensitive", "FILE:x", RefFileURL, ""},

	// Other schemes.
	{"Windows drive with backslash", `C:\x`, RefInvalid, ""},
	{"Windows drive with slash", "C:/x", RefInvalid, ""},
	{"other scheme", "mailto:x", RefInvalid, ""},
	{"scheme with digit, plus, dot and hyphen", "a1+b.c-d:x", RefInvalid, ""},
	{"scheme alone", "data:", RefInvalid, ""},

	// A colon that does not end a scheme.
	{"leading colon", ":x", RefInvalid, ""},
	{"digit before colon", "1:x", RefInvalid, ""},
	{"plus before colon", "+a:x", RefInvalid, ""},
	{"non-scheme character before colon", "a b:x", RefInvalid, ""},
	{"non-ASCII before colon", "é:x", RefInvalid, ""},

	// Absolute paths and dot segments.
	{"absolute", "/abs", RefInvalid, ""},
	{"root", "/", RefInvalid, ""},
	{"parent segment in the middle", "a/../b", RefInvalid, ""},
	{"parent segment alone", "..", RefInvalid, ""},
	{"parent segment first", "../a", RefInvalid, ""},
	{"parent segment last", "a/..", RefInvalid, ""},
	{"escaped parent segment", "%2E%2E/x", RefInvalid, ""},
	{"half-escaped parent segment", ".%2e/x", RefInvalid, ""},
	{"escaped current segment", "%2E/x", RefInvalid, ""},
	{"empty", "", RefInvalid, ""},
	{"empty segment", "a//b", RefInvalid, ""},
	{"current segment first", "./a", RefInvalid, ""},
	{"current segment alone", ".", RefInvalid, ""},
	{"trailing slash", "a/", RefInvalid, ""},

	// Separators.
	{"backslash", `a\b`, RefInvalid, ""},
	{"escaped backslash", "a%5Cb", RefInvalid, ""},
	{"escaped backslash, lower-case hex", "a%5cb", RefInvalid, ""},
	{"escaped slash", "a%2Fb", RefInvalid, ""},
	{"escaped slash, lower-case hex", "a%2fb", RefInvalid, ""},

	// Query and fragment.
	{"query", "a?b", RefInvalid, ""},
	{"fragment", "a#b", RefInvalid, ""},

	// Malformed escapes.
	{"truncated escape", "a%", RefInvalid, ""},
	{"one-digit escape", "a%4", RefInvalid, ""},
	{"non-hex escape", "a%zz", RefInvalid, ""},
	{"malformed escape hiding a separator escape", "a%%2Fb", RefInvalid, ""},

	// Banned characters and invalid UTF-8.
	{"escaped NUL", "a%00b", RefInvalid, ""},
	{"literal NUL", "a\x00b", RefInvalid, ""},
	{"escaped U+0008", "a%08b", RefInvalid, ""},
	{"escaped U+000B", "a%0Bb", RefInvalid, ""},
	{"escaped U+001F", "a%1Fb", RefInvalid, ""},
	{"escaped DEL", "a%7Fb", RefInvalid, ""},
	{"literal DEL", "a\x7Fb", RefInvalid, ""},
	{"escaped U+0080", "a%C2%80b", RefInvalid, ""},
	{"literal U+009F", "a\u009Fb", RefInvalid, ""},
	{"escaped U+FFFE", "a%EF%BF%BEb", RefInvalid, ""},
	{"literal U+FFFF", "a\uFFFFb", RefInvalid, ""},
	{"escaped surrogate", "a%ED%A0%80b", RefInvalid, ""},
	{"literal surrogate bytes", "a\xED\xA0\x80b", RefInvalid, ""},
	{"invalid UTF-8 once decoded", "%FF", RefInvalid, ""},
	{"literal invalid UTF-8", "a\xFFb", RefInvalid, ""},

	// Controls the datatype does not ban, refused as the safe reading.
	{"escaped tab", "a%09b", RefInvalid, ""},
	{"literal tab", "a\tb", RefInvalid, ""},
	{"escaped line feed", "a%0Ab", RefInvalid, ""},
	{"escaped carriage return", "a%0Db", RefInvalid, ""},
}

func TestClassifyFileRef(t *testing.T) {
	for _, tt := range classifyFileRefTests {
		t.Run(tt.name, func(t *testing.T) {
			kind, entry := ClassifyFileRef(tt.ref)
			if kind != tt.wantKind || entry != tt.wantEntry {
				t.Errorf("ClassifyFileRef(%q) = (%v, %q), want (%v, %q)",
					tt.ref, kind, entry, tt.wantKind, tt.wantEntry)
			}
		})
	}
}

// TestClassifyFileRefEntryNameIsSafe checks the guarantee the reader and writer
// rely on, over every table input and every three-atom combination of the
// characters the rules turn on: an entry name is returned for RefLocal only,
// and is then a clean relative path.
func TestClassifyFileRefEntryNameIsSafe(t *testing.T) {
	refs := make([]string, 0, len(classifyFileRefTests))
	for _, tt := range classifyFileRefTests {
		refs = append(refs, tt.ref)
	}
	atoms := []string{
		"a", ".", "..", "/", `\`, ":", "?", "#", "%", " ", "\t", "\x00", "\x7F", "\xFF", "é",
		"%2E", "%2F", "%2f", "%5C", "%5c", "%25", "%00", "%09", "%0A", "%7F", "%C2%80", "%FF", "%3A",
	}
	for _, a := range atoms {
		for _, b := range atoms {
			for _, c := range atoms {
				refs = append(refs, a+b+c)
			}
		}
	}

	for _, ref := range refs {
		kind, entry := ClassifyFileRef(ref)
		if kind != RefLocal {
			if entry != "" {
				t.Errorf("ClassifyFileRef(%q) = (%v, %q), want an empty entry name", ref, kind, entry)
			}
			continue
		}
		if why := unsafeEntryName(entry); why != "" {
			t.Errorf("ClassifyFileRef(%q) entry name %q %s", ref, entry, why)
		}
	}
}

// unsafeEntryName says what is wrong with an entry name, independently of the
// implementation under test, or returns "" if nothing is.
func unsafeEntryName(name string) string {
	switch {
	case name == "":
		return "is empty"
	case strings.HasPrefix(name, "/"):
		return "is absolute"
	case strings.Contains(name, `\`):
		return "contains a backslash"
	case strings.ToValidUTF8(name, "") != name:
		return "is not valid UTF-8"
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "has an empty or dot segment"
		}
	}
	for _, r := range name {
		if r < 0x20 || (r >= 0x7F && r <= 0x9F) {
			return "contains a control character"
		}
	}
	return ""
}

func TestRefKindString(t *testing.T) {
	tests := []struct {
		kind RefKind
		want string
	}{
		{RefInvalid, "invalid"},
		{RefLocal, "local"},
		{RefWeb, "web"},
		{RefFileURL, "file-url"},
		{RefKind(0), "invalid"},
		{RefKind(4), "RefKind(4)"},
		{RefKind(-1), "RefKind(-1)"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("RefKind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
		}
	}
}
