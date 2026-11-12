package converter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/charset"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// The gedcom7code test-files suite (issue #718) pairs each 5.5.1 file in 5/
// with its 7.0 conversion in 7/. Each pair is converted in both directions
// and the output compared with the other side under the equivalences the
// suite's README defines: a trailing space on a valueless line, the order of
// substructures with distinct tags, an explicit GREGORIAN calendar, the tag an
// extension URI is given in SCHMA, and the spelling of cross-reference
// identifiers. HEAD is compared less the declarations listed in headerDropped.
//
// The expected side is read from the upstream bytes, not decoded and
// re-encoded, so a decoder or encoder loss cannot hide by happening on both
// sides.
//
// conversionKnownGaps is self-cleaning in the style of byte_roundtrip_test.go:
// a pair that starts matching fails asking for its entry to be removed, and
// one that stops matching without an entry fails as a regression.

const gedcom7codeDir = "../testdata/gedcom7code"

// conversionKnownGaps lists conversions whose output is not equivalent to the
// suite's, keyed "name 5->7" or "name 7->5", with the cause.
//
// Most 7->5 entries are the inverse of a 5->7 rewrite the converter does not
// make yet. A few cannot close: where the 7/ file dropped something on the
// way up (SUBN, @#D...@ inside note text) or chose a record over an inline
// form, the 5/ file is one of several faithful downgrades, not the only one.
var conversionKnownGaps = map[string]string{
	"age-invalid.ged 5->7": "AGE payloads are not rewritten to 7.0 age syntax (0 -> 0y) (#704)",
	"age-invalid.ged 7->5": "7.0 age syntax (0y) is not rewritten to 5.5.1 (#704)",
	"age-valid.ged 5->7":   "AGE keywords are not moved into PHRASE (#704)",
	"age-valid.ged 7->5":   "7.0 AGE + PHRASE is not restored to the 5.5.1 keyword (#704)",

	"atsign.ged 5->7": "@@ inside a payload is not unescaped on decode (#708)",
	"atsign.ged 7->5": "not invertible: 7/ already rewrote the @#D...@ escapes in note text that 5/ keeps",

	"date-all.ged 5->7":          "dual-year dates are not rewritten to BET/AND + PHRASE (#722)",
	"date-all.ged 7->5":          "BET/AND + PHRASE is not restored to the dual-year form (#722)",
	"date-dual-invalid.ged 5->7": "dual-year dates are not rewritten to BET/AND + PHRASE (#722)",
	"date-dual-invalid.ged 7->5": "BET/AND + PHRASE is not restored to the dual-year form (#722)",

	"enum-ext.ged 5->7": "extension record tag _LOC is not documented in SCHMA",
	"enum-ext.ged 7->5": "NAME.TYPE OTHER + PHRASE is not restored to free text (#704)",

	"filename-1.ged 5->7": "FILE paths are not rewritten as file:// URIs (#705)",
	"filename-1.ged 7->5": "file:// URIs are not rewritten as 5.5.1 paths (#705)",

	"lang-all.ged 5->7": "LANG language names are not mapped to BCP 47 tags (#705)",
	"lang-all.ged 7->5": "LANG BCP 47 tags are not mapped to 5.5.1 language names (#705)",

	"obje-1.ged 5->7": "inline OBJE links are not promoted to OBJE records (#705)",
	"obje-1.ged 7->5": "not invertible: the OBJE record is kept as a record; 5/ uses the inline form",

	"obsolete-1.ged 7->5": "not invertible: 7/ has no SUBN record to restore",

	"pedi-1.ged 5->7": "PEDI enumeration case is not converted (#704)",
	"pedi-1.ged 7->5": "PEDI enumeration case is not converted (#704)",

	"rela_1.ged 5->7": "ASSO.RELA is not mapped to ROLE + PHRASE (#704)",
	"rela_1.ged 7->5": "ASSO.ROLE + PHRASE is not mapped to RELA (#704)",

	"sour-1.ged 5->7": "inline-text SOUR citations are not promoted to SOUR records (#793)",
	"sour-1.ged 7->5": "not invertible: the SOUR record is kept as a record; 5/ uses the inline form",

	"tiny-1.ged 7->5": "not invertible: the downgrade adds the HEAD.SUBM and SUBM record 5.5.1 requires (#705); 5/ omits both",

	"xref-case.ged 5->7": "HEAD.SUBM @test@ resolves case-insensitively to @TEST@; the suite treats it as dangling (@VOID@)",
	"xref-case.ged 7->5": "not invertible: HEAD.SUBM @VOID@ is dropped; 5/ holds a dangling @test@",
}

func TestGedcom7codeConversionPairs(t *testing.T) {
	pairs := gedcom7codePairs(t)
	for _, name := range pairs {
		for _, dir := range []struct {
			label, from, to string
			target          gedcom.Version
		}{
			{"5->7", "5", "7", gedcom.Version70},
			{"7->5", "7", "5", gedcom.Version551},
		} {
			key := name + " " + dir.label
			t.Run(key, func(t *testing.T) {
				diff := convertPairDiff(
					filepath.Join(gedcom7codeDir, dir.from, name),
					filepath.Join(gedcom7codeDir, dir.to, name),
					dir.target)
				reason, known := conversionKnownGaps[key]
				switch {
				case diff == "" && known:
					t.Errorf("conversion now matches; remove %q from conversionKnownGaps (was: %s)", key, reason)
				case diff != "" && !known:
					t.Errorf("conversion does not match the suite and is not in conversionKnownGaps:\n%s", diff)
				}
			})
		}
	}
}

// gedcom7codePairs returns the file names present in both 5/ and 7/.
func gedcom7codePairs(t *testing.T) []string {
	t.Helper()
	five, err := filepath.Glob(filepath.Join(gedcom7codeDir, "5", "*.ged"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	var names []string
	for _, p := range five {
		name := filepath.Base(p)
		if _, err := os.Stat(filepath.Join(gedcom7codeDir, "7", name)); err == nil {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		t.Fatal("no 5/ and 7/ pairs found; the test would prove nothing")
	}
	return names
}

// convertPairDiff converts the file at from to target and describes the
// first difference from the file at want, or returns "" when they match.
func convertPairDiff(from, want string, target gedcom.Version) string {
	src, err := os.ReadFile(from) // #nosec G304 -- path comes from globbing the repo's own testdata
	if err != nil {
		return fmt.Sprintf("read source: %v", err)
	}
	doc, err := decoder.Decode(bytes.NewReader(src))
	if err != nil {
		return fmt.Sprintf("decode source: %v", err)
	}
	converted, _, err := Convert(doc, target)
	if err != nil {
		return fmt.Sprintf("convert: %v", err)
	}
	var out bytes.Buffer
	if err := encoder.Encode(&out, converted); err != nil {
		return fmt.Sprintf("encode: %v", err)
	}

	expected, err := os.ReadFile(want) // #nosec G304 -- path comes from globbing the repo's own testdata
	if err != nil {
		return fmt.Sprintf("read expected: %v", err)
	}
	expectedUTF8, err := io.ReadAll(charset.NewReader(bytes.NewReader(expected)))
	if err != nil {
		return fmt.Sprintf("transcode expected: %v", err)
	}

	v7 := target == gedcom.Version70
	return firstLineDiff(canonicalGEDCOM(expectedUTF8, v7), canonicalGEDCOM(out.Bytes(), v7))
}

// suiteNode is one structure of a file read for comparison.
type suiteNode struct {
	xref, tag, value string
	children         []*suiteNode
}

// pointerPayload matches a payload that is a single pointer.
var pointerPayload = regexp.MustCompile(`^@[^@#]+@$`)

// gregorianEscape matches an explicit Gregorian calendar in either version.
var gregorianEscape = regexp.MustCompile(`(@#DGREGORIAN@|\bGREGORIAN) `)

// headerDropped lists the HEAD substructures canonicalGEDCOM does not compare:
// the version and encoding declarations (GEDC, CHAR, FORM), the SCHMA that
// extension tags are compared through, and SOUR and SUBN, which the suite's 7/
// headers add or drop by hand in ways its README does not call equivalent.
var headerDropped = map[string]bool{
	"GEDC": true, "CHAR": true, "FORM": true, "SCHMA": true, "SOUR": true, "SUBN": true,
}

// canonicalGEDCOM renders data in a form in which two files equivalent under
// the suite's rules are identical text. v7 selects the version's @ escaping.
func canonicalGEDCOM(data []byte, v7 bool) string {
	roots := readSuiteTree(data)

	// Extension tags documented in SCHMA are compared by URI, not by tag.
	// HEAD is compared without headerDropped, which holds the substructures
	// a conversion must change or that the suite's 7/ headers wrote by hand.
	uris := map[string]string{}
	for _, r := range roots {
		if r.tag != "HEAD" {
			continue
		}
		var kept []*suiteNode
		for _, c := range r.children {
			if c.tag == "SCHMA" {
				for _, tag := range c.children {
					if f := strings.Fields(tag.value); tag.tag == "TAG" && len(f) == 2 {
						uris[f[0]] = "<" + f[1] + ">"
					}
				}
			}
			if !headerDropped[c.tag] {
				kept = append(kept, c)
			}
		}
		r.children = kept
	}
	records := roots

	var normalize func(n *suiteNode)
	normalize = func(n *suiteNode) {
		if uri, ok := uris[n.tag]; ok {
			n.tag = uri
		}
		if !pointerPayload.MatchString(n.value) {
			n.value = unescapeAt(n.value, v7)
		}
		if n.tag == "DATE" {
			n.value = gregorianEscape.ReplaceAllString(n.value, "")
		}
		for _, c := range n.children {
			normalize(c)
		}
		sort.SliceStable(n.children, func(i, j int) bool { return n.children[i].tag < n.children[j].tag })
	}
	top := &suiteNode{children: records}
	normalize(top)

	// Identifiers are arbitrary: number them in canonical record order.
	ids := map[string]string{}
	for _, r := range top.children {
		if r.xref != "" {
			ids[r.xref] = fmt.Sprintf("@X%d@", len(ids)+1)
		}
	}

	var sb strings.Builder
	var render func(n *suiteNode, level int)
	render = func(n *suiteNode, level int) {
		fmt.Fprintf(&sb, "%d", level)
		if n.xref != "" {
			sb.WriteString(" " + ids[n.xref])
		}
		sb.WriteString(" " + n.tag)
		if id, ok := ids[n.value]; ok {
			sb.WriteString(" " + id)
		} else if n.value != "" {
			sb.WriteString(" " + strings.ReplaceAll(n.value, "\n", `\n`))
		}
		sb.WriteString("\n")
		for _, c := range n.children {
			render(c, level+1)
		}
	}
	for _, r := range top.children {
		render(r, 0)
	}
	return sb.String()
}

// unescapeAt removes @ doubling: everywhere in 5.5.1, only at the start of
// the payload in 7.0.
func unescapeAt(s string, v7 bool) string {
	if v7 {
		if strings.HasPrefix(s, "@@") {
			return s[1:]
		}
		return s
	}
	return strings.ReplaceAll(s, "@@", "@")
}

// readSuiteTree reads GEDCOM lines into a tree, folding CONT and CONC into
// their parent's payload. It assumes well-formed input, which the suite and
// the encoder both produce.
func readSuiteTree(data []byte) []*suiteNode {
	s := strings.TrimPrefix(string(data), "\uFEFF")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	var roots []*suiteNode
	var stack []*suiteNode
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		levelStr, rest, _ := strings.Cut(line, " ")
		var level int
		_, _ = fmt.Sscanf(levelStr, "%d", &level)

		n := &suiteNode{}
		if strings.HasPrefix(rest, "@") {
			if end := strings.Index(rest[1:], "@ "); end >= 0 {
				n.xref, rest = rest[:end+2], rest[end+3:]
			}
		}
		n.tag, n.value, _ = strings.Cut(rest, " ")

		if level > len(stack) {
			level = len(stack)
		}
		stack = stack[:level]
		if level > 0 {
			parent := stack[level-1]
			switch n.tag {
			case "CONT":
				parent.value += "\n" + n.value
				continue
			case "CONC":
				parent.value += n.value
				continue
			}
			parent.children = append(parent.children, n)
		} else {
			roots = append(roots, n)
		}
		stack = append(stack, n)
	}
	return roots
}

// firstLineDiff reports the first differing line of two canonical renderings.
func firstLineDiff(want, got string) string {
	if want == got {
		return ""
	}
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return fmt.Sprintf("  canonical line %d\n  want: %q\n  got:  %q", i+1, a, b)
		}
	}
	return "  (renderings differ only in length)"
}

// TestCanonicalGEDCOMEquivalences pins the comparison itself: each suite
// equivalence must compare equal, and reordering same-tag siblings must not.
func TestCanonicalGEDCOMEquivalences(t *testing.T) {
	const base = "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SCHMA\n2 TAG _LOC https://example.com/loc\n" +
		"0 @I1@ INDI\n1 NOTE first\n1 BIRT\n2 DATE GREGORIAN 1930\n1 SNOTE @N1@\n1 NOTE second\n" +
		"0 @N1@ SNOTE @@ a note\n1 CHAN \n0 @L1@ _LOC\n0 TRLR\n"
	const equivalent = "\uFEFF0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SCHMA\n2 TAG _PLACE https://example.com/loc\n" +
		"0 @A@ INDI\n1 BIRT\n2 DATE 1930\n1 NOTE first\n1 SNOTE @B@\n1 NOTE second\n" +
		"0 @B@ SNOTE @@ a\n1 CONC  note\n1 CHAN\n0 @C@ _PLACE\n0 TRLR\n"
	const different = "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SCHMA\n2 TAG _LOC https://example.com/loc\n" +
		"0 @I1@ INDI\n1 NOTE second\n1 BIRT\n2 DATE GREGORIAN 1930\n1 SNOTE @N1@\n1 NOTE first\n" +
		"0 @N1@ SNOTE @@ a note\n1 CHAN \n0 @L1@ _LOC\n0 TRLR\n"

	want := canonicalGEDCOM([]byte(base), true)
	if diff := firstLineDiff(want, canonicalGEDCOM([]byte(equivalent), true)); diff != "" {
		t.Errorf("equivalent file compares unequal:\n%s", diff)
	}
	if firstLineDiff(want, canonicalGEDCOM([]byte(different), true)) == "" {
		t.Error("reordering same-tag siblings compares equal")
	}
	if firstLineDiff(canonicalGEDCOM([]byte("0 HEAD\n1 NOTE \u00b6\n0 TRLR\n"), true),
		canonicalGEDCOM([]byte("0 HEAD\n1 NOTE ?\n0 TRLR\n"), true)) == "" {
		t.Error("a differing HEAD.NOTE compares equal")
	}
	if firstLineDiff(canonicalGEDCOM([]byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR ASCII\n0 TRLR\n"), false),
		canonicalGEDCOM([]byte("0 HEAD\n1 GEDC\n2 VERS 7.0\n0 TRLR\n"), false)) != "" {
		t.Error("the version and encoding declarations are compared")
	}
	if got := unescapeAt("a@@b", false); got != "a@b" {
		t.Errorf("5.5.1 unescape = %q, want a@b", got)
	}
}
