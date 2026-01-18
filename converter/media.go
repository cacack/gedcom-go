package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// legacyToIANA maps legacy GEDCOM 5.5/5.5.1 media types to IANA media types.
var legacyToIANA = map[string]string{
	"JPG":  "image/jpeg",
	"JPEG": "image/jpeg",
	"PNG":  "image/png",
	"GIF":  "image/gif",
	"TIFF": "image/tiff",
	"TIF":  "image/tiff",
	"BMP":  "image/bmp",
	"MP3":  "audio/mpeg",
	"WAV":  "audio/wav",
	"MP4":  "video/mp4",
	"MPEG": "video/mpeg",
	"MPG":  "video/mpeg",
	"AVI":  "video/x-msvideo",
	"PDF":  "application/pdf",
	"TXT":  "text/plain",
	"TEXT": "text/plain",
}

// ianaToLegacy maps IANA media types to legacy GEDCOM 5.5/5.5.1 media types.
var ianaToLegacy = map[string]string{
	"image/jpeg":      "JPG",
	"image/png":       "PNG",
	"image/gif":       "GIF",
	"image/tiff":      "TIFF",
	"image/bmp":       "BMP",
	"audio/mpeg":      "MP3",
	"audio/wav":       "WAV",
	"video/mp4":       "MP4",
	"video/mpeg":      "MPEG",
	"video/x-msvideo": "AVI",
	"application/pdf": "PDF",
	"text/plain":      "TXT",
}

// transformMediaTypes updates media type formats in OBJE records based on target version.
// GEDCOM 5.5/5.5.1 uses short formats (JPG, PNG), while GEDCOM 7.0 uses IANA media types.
func transformMediaTypes(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	var transformCount int
	var details []string

	for _, record := range doc.Records {
		if record.Type != gedcom.RecordTypeMedia {
			continue
		}

		media, ok := record.GetMediaObject()
		if !ok || media == nil {
			continue
		}

		// Transform Form field in each MediaFile
		for _, file := range media.Files {
			if file == nil || file.Form == "" {
				continue
			}

			oldValue := file.Form
			var newValue string

			switch targetVersion {
			case gedcom.Version70:
				newValue = toIANAMediaType(oldValue)
			case gedcom.Version55, gedcom.Version551:
				newValue = toLegacyMediaType(oldValue)
			}

			if newValue != "" && newValue != oldValue {
				file.Form = newValue
				transformCount++
				details = append(details, oldValue+" -> "+newValue)
			}
		}

		// Also transform Form in MediaFile Translations
		for _, file := range media.Files {
			if file == nil {
				continue
			}
			for _, trans := range file.Translations {
				if trans == nil || trans.Form == "" {
					continue
				}

				oldValue := trans.Form
				var newValue string

				switch targetVersion {
				case gedcom.Version70:
					newValue = toIANAMediaType(oldValue)
				case gedcom.Version55, gedcom.Version551:
					newValue = toLegacyMediaType(oldValue)
				}

				if newValue != "" && newValue != oldValue {
					trans.Form = newValue
					transformCount++
					details = append(details, oldValue+" -> "+newValue)
				}
			}
		}
	}

	if transformCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "MEDIA_TYPE_MAPPED",
			Description: "Converted media type formats for target GEDCOM version",
			Count:       transformCount,
			Details:     details,
		})
	}
}

// toIANAMediaType converts a legacy media type to IANA format.
// Returns the IANA media type if a mapping exists, or empty string if not found.
// If the input already contains a slash (indicating IANA format), it is returned as-is.
func toIANAMediaType(legacy string) string {
	upper := strings.ToUpper(strings.TrimSpace(legacy))
	if iana, ok := legacyToIANA[upper]; ok {
		return iana
	}
	// Already in IANA format
	if strings.Contains(legacy, "/") {
		return legacy
	}
	return ""
}

// toLegacyMediaType converts an IANA media type to legacy format.
// Returns the legacy media type if a mapping exists, or empty string if not found.
// If the input does not contain a slash (indicating legacy format), it is returned as-is.
func toLegacyMediaType(iana string) string {
	lower := strings.ToLower(strings.TrimSpace(iana))
	if legacy, ok := ianaToLegacy[lower]; ok {
		return legacy
	}
	// Already in legacy format
	if !strings.Contains(iana, "/") {
		return iana
	}
	return ""
}
