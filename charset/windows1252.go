package charset

import (
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// windows1252Table maps each byte to its Windows-1252 rune. The five bytes
// Windows-1252 leaves undefined (0x81, 0x8D, 0x8F, 0x90, 0x9D) map to their
// ISO-8859-1 code points, as WHATWG specifies, rather than to U+FFFD as
// charmap.Windows1252 decodes them, so no input byte is lost.
var windows1252Table = codePageTable(charmap.Windows1252)

// windows1250Table maps each byte to its Windows-1250 rune, keeping the bytes
// Windows-1250 leaves undefined as windows1252Table does.
var windows1250Table = codePageTable(charmap.Windows1250)

// codePageTable returns cm's byte-to-rune table, with each byte cm leaves
// undefined mapped to its ISO-8859-1 code point.
func codePageTable(cm *charmap.Charmap) (t [256]rune) {
	for i := range t {
		r := cm.DecodeByte(byte(i))
		if r == utf8.RuneError {
			r = rune(i)
		}
		t[i] = r
	}
	return t
}

// tableDecoder is a transform.Transformer that converts single-byte code page
// bytes to UTF-8 using a code page table.
type tableDecoder struct {
	transform.NopResetter
	table *[256]rune
}

// windows1252Decoder returns a decoder for Windows-1252 using windows1252Table.
func windows1252Decoder() tableDecoder { return tableDecoder{table: &windows1252Table} }

// windows1250Decoder returns a decoder for Windows-1250 using windows1250Table.
func windows1250Decoder() tableDecoder { return tableDecoder{table: &windows1250Table} }

// Transform decodes one byte per rune and keeps no state, so atEOF needs no
// handling. ASCII bytes are copied directly as a fast path; the table maps
// them to themselves too.
func (d tableDecoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		b := src[nSrc]
		if b < utf8.RuneSelf {
			if nDst >= len(dst) {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[nDst] = b
			nDst++
		} else {
			r := d.table[b]
			if nDst+utf8.RuneLen(r) > len(dst) {
				return nDst, nSrc, transform.ErrShortDst
			}
			nDst += utf8.EncodeRune(dst[nDst:], r)
		}
		nSrc++
	}
	return nDst, nSrc, nil
}
