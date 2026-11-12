package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// languageTags maps each GEDCOM 5.5/5.5.1 LANGUAGE_ID with a clear BCP 47
// equivalent to its primary language subtag. 5.x names languages in English
// words from a fixed list; 7.0 uses BCP 47 tags. Names with no single
// equivalent (Catalan_Spn, Serbo_Croa, Pahari) are left out and convert as
// unmapped.
var languageTags = map[string]string{
	"Afrikaans": "af", "Albanian": "sq", "Amharic": "am", "Anglo-Saxon": "ang",
	"Arabic": "ar", "Armenian": "hy", "Assamese": "as", "Belorusian": "be",
	"Bengali": "bn", "Braj": "bra", "Bulgarian": "bg", "Burmese": "my",
	"Cantonese": "yue", "Catalan": "ca", "Church-Slavic": "cu", "Czech": "cs",
	"Danish": "da", "Dogri": "doi", "Dutch": "nl", "English": "en",
	"Esperanto": "eo", "Estonian": "et", "Faroese": "fo", "Finnish": "fi",
	"French": "fr", "Georgian": "ka", "German": "de", "Greek": "el",
	"Gujarati": "gu", "Hawaiian": "haw", "Hebrew": "he", "Hindi": "hi",
	"Hungarian": "hu", "Icelandic": "is", "Indonesian": "id", "Italian": "it",
	"Japanese": "ja", "Kannada": "kn", "Khmer": "km", "Konkani": "kok",
	"Korean": "ko", "Lahnda": "lah", "Lao": "lo", "Latvian": "lv",
	"Lithuanian": "lt", "Macedonian": "mk", "Maithili": "mai", "Malayalam": "ml",
	"Mandrin": "cmn", "Manipuri": "mni", "Marathi": "mr", "Mewari": "mtr",
	"Navaho": "nv", "Nepali": "ne", "Norwegian": "no", "Oriya": "or",
	"Pali": "pi", "Panjabi": "pa", "Persian": "fa", "Polish": "pl",
	"Portuguese": "pt", "Prakrit": "pra", "Pusto": "ps", "Rajasthani": "raj",
	"Romanian": "ro", "Russian": "ru", "Sanskrit": "sa", "Serb": "sr",
	"Slovak": "sk", "Slovene": "sl", "Spanish": "es", "Swedish": "sv",
	"Tagalog": "tl", "Tamil": "ta", "Telugu": "te", "Thai": "th",
	"Tibetan": "bo", "Turkish": "tr", "Ukrainian": "uk", "Urdu": "ur",
	"Vietnamese": "vi", "Wendic": "wen", "Yiddish": "yi",
}

// languageNames is languageTags inverted, keyed by lowercase subtag.
var languageNames = func() map[string]string {
	names := make(map[string]string, len(languageTags))
	for name, tag := range languageTags {
		names[tag] = name
	}
	return names
}()

// transformLanguages rewrites the header's and each submitter's LANG between
// the 5.x language names and BCP 47 tags, the only LANG structures 5.x has
// (every other 7.0 LANG is dropped on a downgrade as a structure 5.5.1 does
// not define). The header's typed Language follows its tag.
//
// Upgrading, a known name ("English", any case) becomes its tag ("en"), a
// Normalized note. Downgrading, a tag whose primary subtag has a name becomes
// that name: Normalized when the tag is only that subtag, Approximated when
// it carried more (en-US -> English loses the region). A value already in the
// target's form is left as it is. A value with no mapping is kept verbatim,
// with an Approximated note saying so.
func transformLanguages(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	toV7 := targetVersion == gedcom.Version70
	if doc.Header != nil {
		if len(doc.Header.Tags) == 0 {
			doc.Header.Language = convertLanguage(doc.Header.Language, toV7, BuildNestedPath("HEAD", "", "LANG"), report)
		}
		for _, tag := range doc.Header.Tags {
			if tag != nil && tag.Level == 1 && tag.Tag == "LANG" {
				tag.Value = convertLanguage(tag.Value, toV7, BuildNestedPath("HEAD", "", "LANG"), report)
				doc.Header.Language = tag.Value
			}
		}
	}
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeSubmitter {
			continue
		}
		for _, tag := range record.Tags {
			if tag != nil && tag.Level == 1 && tag.Tag == "LANG" {
				tag.Value = convertLanguage(tag.Value, toV7, BuildNestedPath(string(record.Type), record.XRef, "LANG"), report)
			}
		}
	}
}

// convertLanguage returns value in the target's language form, reporting the
// change at path.
func convertLanguage(value string, toV7 bool, path string, report *gedcom.ConversionReport) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	if toV7 {
		for name, tag := range languageTags {
			if strings.EqualFold(name, trimmed) {
				report.AddNormalized(gedcom.ConversionNote{
					Path: path, Original: value, Result: tag,
					Reason: "GEDCOM 7.0 identifies languages by BCP 47 tag",
				})
				return tag
			}
		}
		if looksLikeLanguageTag(trimmed) {
			return value
		}
		report.AddApproximated(gedcom.ConversionNote{
			Path: path, Original: value, Result: value,
			Reason: "No BCP 47 tag is known for this GEDCOM 5.x language name; kept verbatim",
		})
		return value
	}

	if _, ok := languageTags[trimmed]; ok {
		return value
	}
	primary, rest, _ := strings.Cut(trimmed, "-")
	if name, ok := languageNames[strings.ToLower(primary)]; ok && looksLikeLanguageTag(trimmed) {
		note := gedcom.ConversionNote{Path: path, Original: value, Result: name}
		if rest == "" {
			note.Reason = "GEDCOM 5.x names languages instead of using BCP 47 tags"
			report.AddNormalized(note)
		} else {
			note.Reason = "GEDCOM 5.x names languages; the tag's script, region or variant has no 5.x form"
			report.AddApproximated(note)
		}
		return name
	}
	report.AddApproximated(gedcom.ConversionNote{
		Path: path, Original: value, Result: value,
		Reason: "No GEDCOM 5.x language name is known for this BCP 47 tag; kept verbatim",
	})
	return value
}

// looksLikeLanguageTag reports whether s has the shape of a BCP 47 tag: a
// 2-3 letter (or "und"-style) primary subtag, then hyphen-separated
// alphanumeric subtags of 1-8 characters. It checks shape, not registration.
func looksLikeLanguageTag(s string) bool {
	parts := strings.Split(s, "-")
	if n := len(parts[0]); n < 2 || n > 3 || !isAlpha(parts[0]) {
		return false
	}
	for _, p := range parts[1:] {
		if len(p) < 1 || len(p) > 8 {
			return false
		}
		for _, r := range p {
			if !isASCIILetter(r) && (r < '0' || r > '9') {
				return false
			}
		}
	}
	return true
}

func isAlpha(s string) bool {
	for _, r := range s {
		if !isASCIILetter(r) {
			return false
		}
	}
	return true
}

func isASCIILetter(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}
