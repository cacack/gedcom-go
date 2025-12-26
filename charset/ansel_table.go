// ANSEL to Unicode mapping tables for GEDCOM character encoding support.
//
// ANSEL (ANSI Z39.47) is a legacy character encoding used in GEDCOM 5.5 files.
// This file provides mapping tables for converting ANSEL-encoded text to Unicode.
//
// Source: Library of Congress MARC-8 specification
// https://www.loc.gov/marc/specifications/speccharmarc8.html
//
// IMPORTANT: ANSEL places combining diacritical marks BEFORE the base character,
// while Unicode places them AFTER. The decoder must handle this reordering.
// For example, ANSEL "´e" (acute + e) becomes Unicode "é" (e + combining acute).

package charset

// anselToUnicode maps ANSEL extended Latin characters (0xA1-0xC8) to Unicode code points.
// These are single-character mappings for special letters and symbols used in
// European languages and other contexts found in genealogical records.
var anselToUnicode = map[byte]rune{
	// Uppercase special letters
	0xA1: '\u0141', // Ł - Uppercase Polish L with stroke
	0xA2: '\u00D8', // Ø - Uppercase Scandinavian O with stroke
	0xA3: '\u0110', // Đ - Uppercase D with stroke (crossbar)
	0xA4: '\u00DE', // Þ - Uppercase Icelandic thorn
	0xA5: '\u00C6', // Æ - Uppercase AE ligature
	0xA6: '\u0152', // Œ - Uppercase OE ligature
	0xA7: '\u02B9', // ʹ - Modifier letter prime (soft sign)
	0xA8: '\u00B7', // · - Middle dot
	0xA9: '\u266D', // ♭ - Music flat sign
	0xAA: '\u00AE', // ® - Registered sign (patent mark)
	0xAB: '\u00B1', // ± - Plus-minus sign
	0xAC: '\u01A0', // Ơ - Uppercase O with horn (Vietnamese)
	0xAD: '\u01AF', // Ư - Uppercase U with horn (Vietnamese)
	0xAE: '\u02BC', // ʼ - Modifier letter apostrophe (alif)
	// 0xAF is undefined in ANSEL
	0xB0: '\u02BB', // ʻ - Modifier letter turned comma (ayn)

	// Lowercase special letters
	0xB1: '\u0142', // ł - Lowercase Polish L with stroke
	0xB2: '\u00F8', // ø - Lowercase Scandinavian O with stroke
	0xB3: '\u0111', // đ - Lowercase D with stroke (crossbar)
	0xB4: '\u00FE', // þ - Lowercase Icelandic thorn
	0xB5: '\u00E6', // æ - Lowercase AE ligature
	0xB6: '\u0153', // œ - Lowercase OE ligature
	0xB7: '\u02BA', // ʺ - Modifier letter double prime (hard sign)
	0xB8: '\u0131', // ı - Lowercase dotless i (Turkish)
	0xB9: '\u00A3', // £ - British pound sign
	0xBA: '\u00F0', // ð - Lowercase eth (Icelandic)
	// 0xBB is undefined in ANSEL
	0xBC: '\u01A1', // ơ - Lowercase O with horn (Vietnamese)
	0xBD: '\u01B0', // ư - Lowercase U with horn (Vietnamese)
	// 0xBE, 0xBF are LDS extensions (not standard ANSEL, but common in GEDCOM files)
	0xBE: '\u25A1', // White square (LDS extension: empty box placeholder)
	0xBF: '\u25A0', // Black square (LDS extension: black box placeholder)

	// Symbols and punctuation
	0xC0: '\u00B0', // ° - Degree sign
	0xC1: '\u2113', // ℓ - Script small L (liter)
	0xC2: '\u2117', // ℗ - Sound recording copyright
	0xC3: '\u00A9', // © - Copyright sign
	0xC4: '\u266F', // ♯ - Music sharp sign
	0xC5: '\u00BF', // ¿ - Inverted question mark (Spanish)
	0xC6: '\u00A1', // ¡ - Inverted exclamation mark (Spanish)
	0xC7: '\u00DF', // ß - Eszett (German sharp S)
	0xC8: '\u20AC', // € - Euro sign
	// 0xC9-0xCC are undefined in ANSEL
	// 0xCD, 0xCE, 0xCF are LDS extensions (not standard ANSEL, but common in GEDCOM files)
	0xCD: '\u0065', // Midline 'e' - LDS extension (rendered as lowercase e)
	0xCE: '\u006F', // Midline 'o' - LDS extension (rendered as lowercase o)
	0xCF: '\u00DF', // Alternate Eszett - LDS extension (same as 0xC7)
}

// anselCombining maps ANSEL combining diacritical marks (0xE0-0xFE) to Unicode
// combining characters. In ANSEL, these marks precede the base character they
// modify, but in Unicode, they follow the base character.
//
// Example: ANSEL byte sequence [0xE2, 0x65] (acute accent + 'e') should be
// converted to Unicode "e\u0301" (e + combining acute accent), which renders as "é".
var anselCombining = map[byte]rune{
	0xE0: '\u0309', // Combining hook above
	0xE1: '\u0300', // Combining grave accent
	0xE2: '\u0301', // Combining acute accent
	0xE3: '\u0302', // Combining circumflex accent
	0xE4: '\u0303', // Combining tilde
	0xE5: '\u0304', // Combining macron
	0xE6: '\u0306', // Combining breve
	0xE7: '\u0307', // Combining dot above
	0xE8: '\u0308', // Combining diaeresis (umlaut)
	0xE9: '\u030C', // Combining caron (hacek)
	0xEA: '\u030A', // Combining ring above
	0xEB: '\uFE20', // Combining ligature left half (deprecated, prefer U+0361)
	0xEC: '\uFE21', // Combining ligature right half (deprecated, prefer U+0361)
	0xED: '\u0315', // Combining comma above right
	0xEE: '\u030B', // Combining double acute accent
	0xEF: '\u0310', // Combining candrabindu
	0xF0: '\u0327', // Combining cedilla
	0xF1: '\u0328', // Combining ogonek
	0xF2: '\u0323', // Combining dot below
	0xF3: '\u0324', // Combining diaeresis below
	0xF4: '\u0325', // Combining ring below
	0xF5: '\u0333', // Combining double low line (double underscore)
	0xF6: '\u0332', // Combining low line (underscore)
	0xF7: '\u0326', // Combining comma below
	0xF8: '\u031C', // Combining left half ring below
	0xF9: '\u032E', // Combining breve below
	0xFA: '\u0360', // Combining double tilde (first half)
	0xFB: '\u0361', // Combining double inverted breve (ligature tie)
	// 0xFC, 0xFD are undefined in ANSEL
	0xFE: '\u0313', // Combining comma above (high comma off center)
}

// IsCombiningDiacritical returns true if the given byte is an ANSEL combining
// diacritical mark (0xE0-0xFE range). These marks modify the character that
// follows them in ANSEL encoding.
//
// Note: Not all bytes in the 0xE0-0xFE range are defined in ANSEL (0xFC and 0xFD
// are undefined), but this function returns true for the entire range to allow
// the decoder to handle undefined codes gracefully.
func IsCombiningDiacritical(b byte) bool {
	return b >= 0xE0 && b <= 0xFE
}
