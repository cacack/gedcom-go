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
var windows1252Table = func() (t [256]rune) {
	for i := range t {
		r := charmap.Windows1252.DecodeByte(byte(i))
		if r == utf8.RuneError {
			r = rune(i)
		}
		t[i] = r
	}
	return t
}()

// windows1252Decoder is a transform.Transformer that converts Windows-1252
// bytes to UTF-8 using windows1252Table.
type windows1252Decoder struct{ transform.NopResetter }

// Transform decodes one byte per rune and keeps no state, so atEOF needs no
// handling. ASCII bytes are copied directly as a fast path; the table maps
// them to themselves too.
func (windows1252Decoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		b := src[nSrc]
		if b < utf8.RuneSelf {
			if nDst >= len(dst) {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[nDst] = b
			nDst++
		} else {
			r := windows1252Table[b]
			if nDst+utf8.RuneLen(r) > len(dst) {
				return nDst, nSrc, transform.ErrShortDst
			}
			nDst += utf8.EncodeRune(dst[nDst:], r)
		}
		nSrc++
	}
	return nDst, nSrc, nil
}
